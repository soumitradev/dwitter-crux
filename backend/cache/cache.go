// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/util"
)

/*
# Cache Design Philosophy

Cached objects are cached with an hour-long expiry time. Why? idk.

Dweets and user objects will be cached as seperate keys with key names such as:
- user:full:myUsername:username
- user:full:myUsername:email

- dweet:full:aa6A9k71Ab:dweetBody
- dweet:full:aa6A9k71Ab:likeUsers
- dweet:full:aa6A9k71Ab:author

and basic objects will be cached as:
- user:basic:myUsername:username
- user:basic:myUsername:email

- dweet:basic:aa6A9k71Ab:dweetBody
- dweet:basic:aa6A9k71Ab:author

Note how every object is squashed until we can use an existing Redis data structure for it

Subtypes will only be referenced with their IDs, so a dweet will be referenced with it's ID, and a user with their username

For redweets, we will cache the ID as "Redweet(authorID, originalRedweetID)" since the redweet object itself doesn't have any uniquely identifying information

Redweets are always considered "full" objects (since they have child "basic" objects)

Unrequested subtypes will not be cached, and will be represented with ["<?>"]

Since this will not collide with any reasonable interpretation of any subtype, this is safe

Basic subtypes are treated as strict subsets of their full counterparts.

So, if a basic user is requested when a full user is cached, the information is extracted from the existing full user.

However, if a basic user is cached but a full one is requested, it is considered a cache miss.

Since the data the API responds with changes based on who views it, all information in the cache is stored with "sudo" privileges.

So, if an unathenticated user requests a cached user, the API will have to make sure the email is hidden in the response.

This also extends to including only the "known" users in likeUsers, redweetUsers, following, and followers

Paginated results are cached with a <skip> tag before and a <?> tag after.

So, if 5 dweets are loaded after the first 10 dweets of a user, and we don't know if there are more after it, the dweets will be formatted as:
[<10>, abcdefghi1, abcdefghi2, abcdefghi3, abcdefghi4, abcdefghi5, <?>]

But, if the user only made 13 dweets in total,
[<10>, abcdefghi1, abcdefghi2, abcdefghi3]

Now if we load the 5 dweets after 5 dweets, the new updated cached value will be:
[<5>, abcdefghi-4, abcdefghi-3, abcdefghi-2, abcdefghi-1, abcdefghi0, abcdefghi1, abcdefghi2, abcdefghi3]

If we load 5 dweets after the first 3 dweets, the last 3 dweets will cache hit, and the first 2 will cache miss, and the new cached value at the end will be:
[<3>, abcdefghi-6, abcdefghi-5, abcdefghi-4, abcdefghi-3, abcdefghi-2, abcdefghi-1, abcdefghi0, abcdefghi1, abcdefghi2, abcdefghi3]

Cached results are mutated on mutation.

So, if someone follows a cached user, the follower's username is appended to the follower list of that user, and the followerCount is incremented, and similar actions are done on the follower's "following" fields

Note that the user that follows the user is not cached, since the API doesn't have a reasonable use-case scenario for that data.

If a cached user redweets a dweet, the redweet object is prepended to both the redweet, feed fields on that user, so it behaves like a paginated but not fetched value since the same <?> tag is used

The redweeted dweet is also cached, and added to the redweetedDweets field.

One big pitfall this can lead to is an object like this:

feedObjects: [abcdefghi1, <3>, abcdefghi5, <5>, abcdefgh11, <?>]

This means that <skip> tags can occur in the middle of lists, and there might be multiple such tags in a list.

Luckily, since dweets are sorted by creation time and not edit time, stranger things won't happen with this where things move around once added.

So, for example, if we cache a User object with 5 feedObjects (4 dweets and 1 redweet),

We will first save the keys:
- user:full:myUsername:username = "myUsername"
- user:full:myUsername:name = "My Name"
- user:full:myUsername:email = "myemail@mail.com"
- user:full:myUsername:bio = "Example bio"
- user:full:myUsername:pfpURL = "https://..."
- user:full:myUsername:dweets = ["<?>"]
- user:full:myUsername:redweets = ["<?>"]
- user:full:myUsername:feedObjects = [abcdefghi1, abcdefghi2, abcdefghi3, abcdefghi4, abcdefghi5]
- user:full:myUsername:redweetedDweets = ["<?>"]
- user:full:myUsername:likedDweets = ["<?>"]
- user:full:myUsername:followerCount = "1236"
- user:full:myUsername:followers = [follower1, follower2, ..., commonfollower1236]
- user:full:myUsername:followingCount = "2317"
- user:full:myUsername:following = [followed1, followed2, ..., followed2317]
- user:full:myUsername:createdAt = "2022-03-08T04:20:49.962Z"

and then, we will cache
- dweet:basic:abcdefghi1:dweetBody = "bruhh 💀💀💀"
- dweet:basic:abcdefghi1:id = "abcdefghi1"
- dweet:basic:abcdefghi1:author = "myUsername"
- dweet:basic:abcdefghi1:authorID = "myUsername"
- dweet:basic:abcdefghi1:postedAt = "2022-03-10T04:20:49.962Z"
- dweet:basic:abcdefghi1:lastUpdatedAt = "2022-03-10T04:20:49.962Z"
- dweet:basic:abcdefghi1:likeCount = "10"
- dweet:basic:abcdefghi1:isReply = "false"
- dweet:basic:abcdefghi1:originalReplyID = ""
- dweet:basic:abcdefghi1:replyCount = "3"
- dweet:basic:abcdefghi1:redweetCount = "4"
- dweet:basic:abcdefghi1:media = [https://..., https://...]

and so on for the rest of the dweets

and for the redweet,
- redweet:full:Redweet(myUsername, abcdefghi8):author = "myUsername"
- redweet:full:Redweet(myUsername, abcdefghi8):authorID = "myUsername"
- redweet:full:Redweet(myUsername, abcdefghi8):redweetOf = "abcdefghi8"
- redweet:full:Redweet(myUsername, abcdefghi8):originalRedweetID = "abcdefghi8"
- redweet:full:Redweet(myUsername, abcdefghi8):redweetTime = "2022-03-9T04:20:49.962Z"

and then since a full user with username "myUsername" already exists (we check the key "user:full:myUsername:username"), we grab the information from there

we will also have to cache a basic version of the original dweet abcdefghi8

and then, we will cache
- dweet:basic:abcdefghi8:dweetBody = "can someone please redweet this one"
- dweet:basic:abcdefghi8:id = "abcdefghi8"
- dweet:basic:abcdefghi8:author = "otherUsername"
- dweet:basic:abcdefghi8:authorID = "otherUsername"
- dweet:basic:abcdefghi8:postedAt = "2022-03-08T04:20:49.962Z"
- dweet:basic:abcdefghi8:lastUpdatedAt = "2022-03-08T04:20:49.962Z"
- dweet:basic:abcdefghi8:likeCount = "23"
- dweet:basic:abcdefghi8:isReply = "false"
- dweet:basic:abcdefghi8:originalReplyID = ""
- dweet:basic:abcdefghi8:replyCount = "6"
- dweet:basic:abcdefghi8:redweetCount = "7"
- dweet:basic:abcdefghi8:media = []

this will also lead us to cache the other user with username otherUsername

- user:basic:otherUsername:username = "otherUsername"
- user:basic:otherUsername:name = "Other Name"
- user:basic:otherUsername:email = "myotheremail@mail.com"
- user:basic:otherUsername:bio = "Different bio"
- user:basic:otherUsername:pfpURL = "https://..."
- user:basic:otherUsername:followerCount = "1829"
- user:basic:otherUsername:followingCount = "2729"
- user:basic:otherUsername:createdAt = "2022-03-05T04:20:49.962Z"

Some terms

- Cache miss: A situation where the requested information is not sufficiently cached such that we must perform a full object fetch from the db

- Partial cache miss: A situation where the requested information is partially cached, requiring us to only fetch a part of the object.

Consider a case where one dweet is cached, the one before and after it isn't, but the one 2 spaces ahead of it is.
[<1>, X, <1>, X+2]
Consider such a case where the dweet being talked about is X, and the dweets requested are: the one before X, X, one after X and second one after X
Such a case will still be considered a partial cache miss, but X will not be considered cached, since it is highly inefficient to perform multiple DB fetches (DB fetches need to be contiguous)
Note that this isn't a cache miss however, so the contiguous edge cached values are considered cached, and will be used from the cache.

This is to maintain a degree of speed in such an operation, since even one non-contiguous occurence of an uncached value can ruin the cached-ness of these values

Let us look at some edge cases though

Consider
[X-3, X-2, <1>, X, <1>, X+2]
Here, we have two edge cached values that we can consider cached. Since we can make a contiguous fetch for the part <1> X <1>, both the edges are considered cached

Consider
[<1>, X, <1>]
Here, there are no edge cached values. This is considered a cache miss

Consider
[<1>, X, <1>, X+2, X+3, X+4, ..., X+10000000, <1>]
Here, there are no edge cached values. This is unfortunately also considered a cache miss.
Of course, the resulting operation might be much slower than fetching 3 of these uncached values, but this is an extreme case, and I can't imagine a better solution that isnt painful to implement.


- Cache hit: A situation where the requested information is fully cached, and no DB operations need to be performed on the requested entity

- Cache update: A situation where cached information needs to be updated

To clear some of the confusion around partial cache miss, and cache miss in particular, we will look at examples:

- When fetching a full user, if the user is not cached, or if the user is cached as a basic type, it is a cache miss

- When fetching a full user, if some of the dweets, redweets, feedObjects... are not cached, it is a partial cache miss.
In this case, only the requested information will be fetched and updated on the cache. We will not fetch the followers, following etc.

===

- All partial cache misses will result in a cache update

- Mutations also result in a cache update

- This implies all partial cache misses are cache updates but the reverse need not be true

---

I might consider pulling full objects from the DB when basic ones are needed just for the cache, but meh
sounds like more work, and its also pretty inefficient. Plus, where do you end?
Do you have a caching depth where you only go to some depth of pulling full objects?
Why pull data when not needed? Doesn't that destroy the whole point of a "cache"?

I might also consider omitting the ID fields for relations since they're useless and take up extra memory

Lots of memory optimization can be done by reducing key name sizes too

But hey this is already pretty painful, and I'm almost next to sure that something will break along the way.

Whew that was a lot of work
*/

// TODO:
// - Finish cache integration of list-objects

func CacheUser(detailLevel string, id string, obj *db.UserModel, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) error {
	keyStem := GenerateKey("user", detailLevel, id, "")
	userMap := map[string]interface{}{
		keyStem + "username":       obj.Username,
		keyStem + "name":           obj.Name,
		keyStem + "email":          obj.Email,
		keyStem + "bio":            obj.Bio,
		keyStem + "pfpURL":         obj.ProfilePicURL,
		keyStem + "followerCount":  strconv.Itoa(obj.FollowerCount),
		keyStem + "followingCount": strconv.Itoa(obj.FollowingCount),
		keyStem + "createdAt":      obj.CreatedAt.UTC().Format(util.TimeUTCFormat),
	}
	err := cacheDB.MSet(common.BaseCtx, userMap).Err()
	if err != nil {
		return err
	}

	expireTime := time.Now().UTC().Add(time.Hour)
	for key := range userMap {
		err = cacheDB.PExpireAt(common.BaseCtx, key, expireTime).Err()
		if err != nil {
			return err
		}
	}

	if detailLevel == "basic" {
		return nil
	} else if detailLevel == "full" {
		followersUserList := obj.Followers()
		followers := make([]interface{}, len(followersUserList))
		for i, user := range followersUserList {
			followers[i] = user.Username
			err := CacheUser("basic", user.Username, &followersUserList[i], "", 0, 0)
			if err != nil {
				return err
			}
		}

		followingUserList := obj.Following()
		following := make([]interface{}, len(followingUserList))
		for i, user := range followingUserList {
			following[i] = user.Username
			err := CacheUser("basic", user.Username, &followingUserList[i], "", 0, 0)
			if err != nil {
				return err
			}
		}

		err = cacheDB.LPush(common.BaseCtx, keyStem+"followers", followers...).Err()
		if err != nil {
			return err
		}

		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"followers", expireTime).Err()
		if err != nil {
			return err
		}

		err = cacheDB.LPush(common.BaseCtx, keyStem+"following", following...).Err()
		if err != nil {
			return err
		}

		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"following", expireTime).Err()
		if err != nil {
			return err
		}

		switch objectsToFetch {
		case "feed":
			err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}

			if feedObjectsToFetch >= 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", uncachedStub).Err()
				if err != nil {
					return err
				}
			} else {
				feedObjectsOffset = 0
			}

			merged := util.MergeDweetRedweetList(obj.Dweets(), obj.Redweets())
			iterLen := util.Min(feedObjectsToFetch, len(merged))
			feedObjectList := make([]interface{}, iterLen)
			feedObjectIDList := make([]interface{}, iterLen)
			for i := 0; i < iterLen; i++ {
				feedObjectList[i] = merged[i+feedObjectsOffset]
			}

			for i, obj := range feedObjectList {
				if dweet, ok := obj.(db.DweetModel); ok {
					feedObjectIDList[i] = dweet.ID
					err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
					if err != nil {
						return err
					}
				} else if redweet, ok := obj.(db.RedweetModel); ok {
					feedObjectIDList[i] = ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
					err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
					if err != nil {
						return err
					}
				} else {
					return errors.New("internal server error")
				}
			}

			err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", feedObjectIDList...).Err()
			if err != nil {
				return err
			}

			if feedObjectsOffset > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", "<"+fmt.Sprintf("%d", feedObjectsOffset)+">").Err()
				if err != nil {
					return err
				}
			}

		case "dweet":
			err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}

			if feedObjectsToFetch >= 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", uncachedStub).Err()
				if err != nil {
					return err
				}
			}

			dweets := obj.Dweets()
			dweetIDs := make([]interface{}, len(dweets))

			for i, dweet := range dweets {
				dweetIDs[i] = dweet.ID
				err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", dweetIDs...).Err()
			if err != nil {
				return err
			}

			if feedObjectsOffset > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", "<"+fmt.Sprintf("%d", feedObjectsOffset)+">").Err()
				if err != nil {
					return err
				}
			}

		case "redweet":
			err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}

			if feedObjectsToFetch >= 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
				if err != nil {
					return err
				}
			}

			redweets := obj.Redweets()
			redweetIDs := make([]interface{}, len(redweets))

			for i, redweet := range redweets {
				redweetIDs[i] = ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
				err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
				if err != nil {
					return err
				}
			}

			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", redweetIDs...).Err()
			if err != nil {
				return err
			}

			if feedObjectsOffset > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", "<"+fmt.Sprintf("%d", feedObjectsOffset)+">").Err()
				if err != nil {
					return err
				}
			}

		case "redweetedDweet":
			err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}

			if feedObjectsToFetch >= 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", uncachedStub).Err()
				if err != nil {
					return err
				}
			}

			dweets := obj.RedweetedDweets()
			dweetIDs := make([]interface{}, len(dweets))

			for i, dweet := range dweets {
				dweetIDs[i] = dweet.ID
				err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", dweetIDs...).Err()
			if err != nil {
				return err
			}

			if feedObjectsOffset > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", "<"+fmt.Sprintf("%d", feedObjectsOffset)+">").Err()
				if err != nil {
					return err
				}
			}

		case "liked":
			err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", uncachedStub).Err()
			if err != nil {
				return err
			}
			err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", uncachedStub).Err()
			if err != nil {
				return err
			}

			if feedObjectsToFetch >= 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", uncachedStub).Err()
				if err != nil {
					return err
				}
			}

			dweets := obj.LikedDweets()
			dweetIDs := make([]interface{}, len(dweets))

			for i, dweet := range dweets {
				dweetIDs[i] = dweet.ID
				err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", dweetIDs...).Err()
			if err != nil {
				return err
			}

			if feedObjectsOffset > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", "<"+fmt.Sprintf("%d", feedObjectsOffset)+">").Err()
				if err != nil {
					return err
				}
			}

		default:
			return errors.New("unknown objectsToFetch")
		}

		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"feedObjects", expireTime).Err()
		if err != nil {
			return err
		}
		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"dweets", expireTime).Err()
		if err != nil {
			return err
		}
		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"redweets", expireTime).Err()
		if err != nil {
			return err
		}
		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"redweetedDweets", expireTime).Err()
		if err != nil {
			return err
		}
		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"likedDweets", expireTime).Err()
		if err != nil {
			return err
		}

		return nil
	} else {
		return errors.New("unknown detailLevel")
	}
}

func CacheDweet(detailLevel string, id string, obj *db.DweetModel, repliesToFetch int, replyOffset int) error {
	keyStem := GenerateKey("dweet", detailLevel, id, "")
	isReply := "false"
	if obj.IsReply {
		isReply = "true"
	}
	dweetMap := map[string]interface{}{
		keyStem + "dweetBody":       obj.DweetBody,
		keyStem + "id":              obj.ID,
		keyStem + "author":          obj.AuthorID,
		keyStem + "authorID":        obj.AuthorID,
		keyStem + "postedAt":        obj.PostedAt.UTC().Format(util.TimeUTCFormat),
		keyStem + "lastUpdatedAt":   obj.LastUpdatedAt.UTC().Format(util.TimeUTCFormat),
		keyStem + "likeCount":       strconv.Itoa(obj.LikeCount),
		keyStem + "isReply":         isReply,
		keyStem + "originalReplyID": obj.OriginalReplyID,
		keyStem + "replyCount":      strconv.Itoa(obj.ReplyCount),
		keyStem + "redweetCount":    strconv.Itoa(obj.RedweetCount),
	}

	interfaceList := make([]interface{}, len(obj.Media))
	for i, mediaLink := range obj.Media {
		interfaceList[i] = mediaLink
	}
	cacheDB.LPush(common.BaseCtx, keyStem+"media", interfaceList...)

	err := cacheDB.MSet(common.BaseCtx, dweetMap).Err()
	if err != nil {
		return err
	}

	err = CacheUser("basic", obj.AuthorID, obj.Author(), "feed", 0, 0)
	if err != nil {
		return err
	}

	expireTime := time.Now().UTC().Add(time.Hour * 1)
	for key := range dweetMap {
		err = cacheDB.PExpireAt(common.BaseCtx, key, expireTime).Err()
		if err != nil {
			return err
		}
	}
	if detailLevel == "basic" {
		return nil
	} else if detailLevel == "full" {
		likeUsers := obj.LikeUsers()
		likeUserIDs := make([]interface{}, len(likeUsers))

		for i, user := range likeUsers {
			likeUserIDs[i] = user.Username
			err := CacheUser("basic", user.Username, &user, "feed", 0, 0)
			if err != nil {
				return err
			}
		}

		redweetUsers := obj.RedweetUsers()
		redweetUserIDs := make([]interface{}, len(redweetUsers))

		for i, user := range redweetUsers {
			redweetUserIDs[i] = user.Username
			err := CacheUser("basic", user.Username, &user, "feed", 0, 0)
			if err != nil {
				return err
			}
		}

		err = cacheDB.LPush(common.BaseCtx, keyStem+"likeUsers", likeUserIDs...).Err()
		if err != nil {
			return err
		}

		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"likeUsers", expireTime).Err()
		if err != nil {
			return err
		}

		err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetUsers", redweetUserIDs...).Err()
		if err != nil {
			return err
		}

		err = cacheDB.PExpireAt(common.BaseCtx, keyStem+"redweetUsers", expireTime).Err()
		if err != nil {
			return err
		}

		if obj.IsReply {
			if replyTo, ok := obj.ReplyTo(); ok {
				err := CacheDweet("basic", replyTo.ID, replyTo, 0, 0)
				if err != nil {
					return err
				}

				err = cacheDB.Do(common.BaseCtx, "set", keyStem+"replyTo", replyTo.ID, "PXAT", expireTime.UnixNano()/int64(time.Millisecond)).Err()
				if err != nil {
					return err
				}
			} else {
				err = cacheDB.Do(common.BaseCtx, "set", keyStem+"replyTo", "", "PXAT", expireTime.UnixNano()/int64(time.Millisecond)).Err()
				if err != nil {
					return err
				}
			}
		}

		if repliesToFetch >= 0 {
			err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", uncachedStub).Err()
			if err != nil {
				return err
			}
		}

		replyDweets := obj.ReplyDweets()
		replyDweetIDs := make([]interface{}, len(replyDweets))

		for i, dweet := range replyDweets {
			replyDweetIDs[i] = dweet.ID
			err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
			if err != nil {
				return err
			}
		}

		err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", replyDweetIDs...).Err()
		if err != nil {
			return err
		}

		if replyOffset > 0 {
			err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", "<"+fmt.Sprintf("%d", replyOffset)+">").Err()
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		return errors.New("unknown detailLevel")
	}
}

func CacheRedweet(detailLevel string, id string, obj *db.RedweetModel) error {
	if detailLevel == "full" {
		keyStem := GenerateKey("redweet", detailLevel, ConstructRedweetID(obj.AuthorID, obj.OriginalRedweetID), "")
		dweetMap := map[string]interface{}{
			keyStem + "author":            obj.AuthorID,
			keyStem + "authorID":          obj.AuthorID,
			keyStem + "redweetOf":         obj.OriginalRedweetID,
			keyStem + "originalRedweetID": obj.OriginalRedweetID,
			keyStem + "redweetTime":       obj.RedweetTime,
		}

		err := cacheDB.MSet(common.BaseCtx, dweetMap).Err()
		if err != nil {
			return err
		}

		expireTime := time.Now().UTC().Add(time.Hour * 1)
		for key := range dweetMap {
			err = cacheDB.PExpireAt(common.BaseCtx, key, expireTime).Err()
			if err != nil {
				return err
			}
		}

		err = CacheUser("basic", obj.AuthorID, obj.Author(), "feed", 0, 0)
		if err != nil {
			return err
		}
		err = CacheDweet("basic", obj.OriginalRedweetID, obj.RedweetOf(), 0, 0)
		if err != nil {
			return err
		}

		return nil
	} else {
		return errors.New("unknown detailLevel")
	}
}

// The return of the "useless" function
func CheckIfCached(objType, detailLevel, id string) (bool, error) {
	var err error
	if detailLevel == "full" {
		if objType == "dweet" {
			_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, detailLevel, id, "id")).Result()
		} else if objType == "user" {
			_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, detailLevel, id, "username")).Result()
		} else if objType == "redweet" {
			_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, detailLevel, id, "author")).Result()
		} else {
			return false, errors.New("unknown objType")
		}

		if err == redis.Nil {
			return false, nil
		} else if err != nil {
			return false, err
		}
	} else if detailLevel == "basic" {
		if objType == "dweet" {
			_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, detailLevel, id, "id")).Result()
		} else if objType == "user" {
			_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, detailLevel, id, "username")).Result()
		} else {
			return false, errors.New("unknown objType for detailLevel basic")
		}

		// If basic objects are not found, check their full counterparts
		if err == redis.Nil {
			if objType == "dweet" {
				_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, "full", id, "id")).Result()
			} else {
				_, err = cacheDB.Get(common.BaseCtx, GenerateKey(objType, "full", id, "username")).Result()
			}

			if err == redis.Nil {
				return false, nil
			} else if err != nil {
				return false, err
			}
		} else if err != nil {
			return false, err
		}
	} else {
		return false, errors.New("unknown objType")
	}

	// No errors means we found the object
	return true, nil
}
