// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
)

/*
Mutation caching philosophy

I am heavily sleep deprived to a few reaons:
- I just came back from vacation, and I didn't get enough time when I was out of town to work on this shit
- Travelling is painful
- I have to buy shit last minute since im coming on campus in like 7 days
- My sem just started and I am NOT prepared (I also need to focus on cgpa since dual)

I have two options in front of me:
1. Complete the task, with a simple mutation policy
2. Risk task completion for a much more sophisticated caching policy

I have less than 24 hours to submit at this point, and 13th March is looking gruesome in terms of getting stuff done for coming on campus.

I have to link my SIM thing to my aadhar card, update some bank info, link bank account, finish getting UPI to work, so much more stuff that needs me to be away from my laptop

Plus this is the last weekend I have before coming on campus, so I need to finish this stuff tomorrow

I think I'll implment the simple mutation policy first, finish up my task, and if I have the time and sanity left to implement the sophisticated policy, I'll do that too.

I really hope the judges cut me a little slack on this (since literally only the implementation is left, which is mostly mechanical work, the idea of how the cache will update is fully fleshed out)

anyways no more crying on github, time to tell you my options:
====
1. Eviction on mutation: De-cache something when its mutated (how does that make sense though, why does liking a post make it uncache? 🤔)
2. Update cache on mutation: Requires much more complex logic, and might even make uncached mutations slower (since we might have to pull additional info from DB)

====
This was my idea for the 2nd option:

If a mutation "caches" some data, it can mean two things:
- UPSERT: Inserts if not present, but updates if present
- UPDATE: Updates if present

When we casually say "insert" or "cache" or "add to cache" we actually mean UPSERT.

Both of these cache update conditions will reset the expiry time of the underlying object

We will use these terms to describe cache update conditions.

Mutations cannot partially cache hit.
This is because we want to use single, bulk operations
(which is why we fetch contiguous chunks of data even when parts of that contigous array are cached)

Mutations involve both updating, and getting the value of an object, so there are 2 sub-actions involved.
So, if we allow mutations to partially cache hit, we will be seperating those two actions, since the cache will fetch once, and the API will update once more.
Best we can really do is return some kind of type that contains info about the uncached contiguous array,
and the API is asked to fetch and update that, but it gets too complicated
and messes with the current design of the backend for very little benefit.

Since mutations fetch full objects that they mutate, the mutated objects are cached in full.

For deletion mutations, we will only delete the objects that are deleted, and update related objects that may need updating
We DO NOT DELETE any secondary objects since they might be required by other cached results.

Deletion mutations that only destroy objects and not update them (deleteDweet, deleteRedweet) do not UPSERT the underlying objects

Additionally, we will perform some additional caching:

Though createUser is a mutation, it will not have any effect on the cache since:
- We don't know if the user data is useful yet (email unverified, user can't login yet, accesses to user are meaningless)
- It can be abused since no verification is needed to create a User

createDweet does not UPSERT the dweet created since:
- We don't know if the user data is useful yet (just the act of creating a dweet does not justify a cache-level of importance)
BUT,
We also use these dweets in paginated results for full-detail-level cached users, so we will UPDATE such users.

With createReply things get a bit tricky
The original dweet would ideally be cached in full detail because replies indicate a level of usefulness of the dweet replied to.
But, this would result in more disconnected DB fetches, which would make uncached requests slower.
But first off, since this is also a dweet, it will also have the same basic behaviour as createDweet, so the author is UPDATEd if cached in full detail
But since this is also a reply, the dweet replied to might have a use for this data. So, if the dweet replied to is cached in full, it is UPDATEd

Redweets behave a little similar to replies:
If the user redweeting is cached in full detail, the redweet object is UPDATEd to the user's feedObjects and redweets fields
If the dweet redweeted is cached in full detail, the redweetUsers object is UPDATEd
The redweet object itself is not cached

Likes will UPSERT the dweet liked in full detail, and the user that likes it (since likeUsers is a field on the original dweet)
If the user that likes the dweet is cached in full detail, their likedDweets field is UPDATEd

Follows will UPSERT the followed user in full detail, and the following user in basic detail (due to the followers field)
If the follower is cached in full detail additionally, their following field is also UPDATEd

Removal of likes will UPSERT the dweet liked such that the user that likes it is removed from the list
If the user that unliked the dweet is cached in full detail, their likedDweets field is UPDATEd

Removal of redweets will destroy the redweet object only
If the user that redweeted the dweet is cached in full detail, their feedObjects and redweets fields are UPDATEd

Removal of follows will UPSERT the user unfollowed such that the user that unfollowed it is removed from the list
If the user that unfollowed the user is cached in full detail, their following field is UPDATEd

editDweet UPSERTs the edited dweet

editUser UPSERTs the edited user

Removal of dweets will destroy the dweet object, all reply dweets and all its redweets
If the user that posted the dweet is cached in full detail, their feedObjects and dweets fields are UPDATEd
If the dweet is a reply to a dweet then the dweet replied to is UPDATEd
*/

// "I liked this dweet so I have like 5 replies to this dweet, see if this info helps"
// "Sure thanks for the first 5 replies I only had like 1"
// *grabs the current reply list, throws those 5 replies when needed (when encountering a stub) and caches them*
// Similarly for users

func CreateDweetCacheUpdate(dweet db.DweetModel) error {
	// Check if author is cached in full, if yes, cache basic version of dweet and add it to user object
	keyStem := GenerateKey("user", "full", dweet.AuthorID, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full, return
		if err == redis.Nil {
			return nil
		}
		return err
	}

	fmt.Println("test")

	err = CacheDweet("basic", dweet.ID, &dweet, 0, 0)
	if err != nil {
		return err
	}
	err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", dweet.ID).Err()
	if err != nil {
		return err
	}
	err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", dweet.ID).Err()
	if err != nil {
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)
	err = ExpireUserAt("full", dweet.AuthorID, expireTime)
	if err != nil {
		return err
	}

	return nil
}

func CreateReplyCacheUpdate(dweet db.DweetModel) error {
	// Check if author is cached in full, if yes, cache basic version of dweet and add it to user object
	// Check if dweet replied to is cached in full, if yes, cache basic version of dweet and add it to dweet object
	userCached := true
	dweetCached := true

	keyStem := GenerateKey("user", "full", dweet.AuthorID, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userCached = false
		}
		return err
	}

	originalReplyID, ok := dweet.OriginalReplyID()
	if ok {
		keyStem := GenerateKey("dweet", "full", originalReplyID, "")
		err := cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
		if err != nil {
			// If dweet isnt cached in full,
			if err == redis.Nil {
				dweetCached = false
			}
			return err
		}

	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userCached {
		keyStem := GenerateKey("user", "full", dweet.AuthorID, "")
		err = CacheDweet("basic", dweet.ID, &dweet, 0, 0)
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", dweet.ID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", dweet.ID).Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("full", dweet.AuthorID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetCached {
		keyStem := GenerateKey("dweet", "full", originalReplyID, "")
		// Check if user already cached the dweet and we dont need to
		if !userCached {
			err = CacheDweet("basic", dweet.ID, &dweet, 0, 0)
			if err != nil {
				return err
			}
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", dweet.ID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.Incr(common.BaseCtx, keyStem+"replyCount").Err()
		if err != nil {
			return err
		}

		err = ExpireDweetAt("full", originalReplyID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WASNT REDWEETED ALREADY
func RedweetCacheUpdate(redweet db.RedweetModel) error {
	// Check if author is cached in full, if yes, cache basic version of dweet and add it to user object
	// Check if dweet redweeted is cached in full, if yes, cache basic version of redweet author and add it to redweetUsers field object
	userCached := true
	dweetCachedFull := true
	dweetCachedBasic := true

	keyStem := GenerateKey("user", "full", redweet.AuthorID, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userCached = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "full", redweet.OriginalRedweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetCachedFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "basic", redweet.OriginalRedweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetCachedBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)
	redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)

	if userCached {
		keyStem := GenerateKey("user", "full", redweet.AuthorID, "")
		err = CacheRedweet("full", redweetID, &redweet)
		if err != nil {
			return err
		}
		err = cacheDB.LRem(common.BaseCtx, keyStem+"redweets", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", redweetID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LRem(common.BaseCtx, keyStem+"feedObjects", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", redweetID).Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("full", redweet.AuthorID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetCachedFull {
		keyStem := GenerateKey("dweet", "full", redweet.OriginalRedweetID, "")
		// If user is already in cache, we dont need to cache it
		if !userCached {
			err = CacheUser("basic", redweet.AuthorID, redweet.Author(), "feed", 0, 0)
			if err != nil {
				return err
			}
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetUsers", redweet.AuthorID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.Incr(common.BaseCtx, keyStem+"redweetCount").Err()
		if err != nil {
			return err
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetCachedBasic {
		keyStem := GenerateKey("dweet", "basic", redweet.OriginalRedweetID, "")
		// If user is already in cache, we dont need to cache it
		err = cacheDB.Incr(common.BaseCtx, keyStem+"redweetCount").Err()
		if err != nil {
			return err
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WASNT LIKED ALREADY
func LikeCacheUpdate(dweet db.DweetModel, userThatLiked db.UserModel, repliesToFetch int, repliesOffset int) error {
	// Check if user that liked is cached in full
	// If yes, add dweet ID to likedDweets
	// If not, cache a basic version of the user since we'll use it later
	// Check if dweet liked is cached in full
	// If yes, add user ID to likeUsers
	// If not, cache a full version of the dweet

	userInFull := true
	dweetInFull := true
	dweetInBasic := true

	keyStem := GenerateKey("user", "full", userThatLiked.Username, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "full", dweet.ID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "basic", dweet.ID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userInFull {
		keyStem := GenerateKey("user", "full", userThatLiked.Username, "")
		err = cacheDB.LRem(common.BaseCtx, keyStem+"likedDweets", 1, dweet.ID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", dweet.ID).Err()
		if err != nil {
			return err
		}
		err = ExpireUserAt("full", userThatLiked.Username, expireTime)
		if err != nil {
			return err
		}
	} else {
		CacheUser("basic", userThatLiked.Username, &userThatLiked, "feed", 0, 0)
	}

	if dweetInFull {
		keyStem := GenerateKey("dweet", "full", dweet.ID, "")
		// Some version of the user is already cached
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"likedDweets", 1, userThatLiked.Username).Result()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"likeUser", userThatLiked.Username).Err()
		if err != nil {
			return err
		}
		if removed == 0 {
			err = cacheDB.Incr(common.BaseCtx, keyStem+"likeCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireDweetAt("full", dweet.ID, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = UpsertDweet(dweet.ID, &dweet, repliesToFetch, repliesOffset)
		if err != nil {
			return err
		}
	}

	if dweetInBasic {
		keyStem := GenerateKey("dweet", "basic", dweet.ID, "")
		// Some version of the user is already cached
		err = cacheDB.Incr(common.BaseCtx, keyStem+"likeCount").Err()
		if err != nil {
			return err
		}

		err = ExpireDweetAt("basic", dweet.ID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE USER WASNT FOLLOWING ALREADY
func FollowCacheUpdate(userThatWasFollowed db.UserModel, userThatFollowed db.UserModel, objectsToFetch string, feedObjectsToFetch int, feedObjectOffset int) error {
	// Check if user that followed is cached in full
	// If yes, add followed username to following
	// If not, cache a basic version of the user since we'll use it later
	// Check if user followed is cached in full
	// If yes, add user ID to followers
	// If not, cache a full version of the followed user

	userFollowingInFull := true
	userFollowingInBasic := true
	userFollowedInFull := true
	userFollowedInBasic := true

	keyStem := GenerateKey("user", "full", userThatFollowed.Username, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowingInFull = false
		}
		return err
	}

	keyStem = GenerateKey("user", "basic", userThatFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowingInBasic = false
		}
		return err
	}

	keyStem = GenerateKey("user", "full", userThatWasFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowedInFull = false
		}
		return err
	}

	keyStem = GenerateKey("user", "basic", userThatWasFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowedInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userFollowingInFull {
		keyStem := GenerateKey("user", "full", userThatFollowed.Username, "")
		err = cacheDB.LPush(common.BaseCtx, keyStem+"following", userThatWasFollowed.Username).Err()
		if err != nil {
			return err
		}
		err = cacheDB.Incr(common.BaseCtx, keyStem+"followingCount").Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("full", userThatFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = CacheUser("basic", userThatFollowed.Username, &userThatFollowed, "feed", 0, 0)
		if err != nil {
			return err
		}
	}

	if userFollowingInBasic {
		keyStem := GenerateKey("user", "basic", userThatFollowed.Username, "")
		err = cacheDB.Incr(common.BaseCtx, keyStem+"followingCount").Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("basic", userThatFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	}

	if userFollowedInFull {
		keyStem := GenerateKey("user", "full", userThatWasFollowed.Username, "")
		// Some version of the user is already cached
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"followers", 1, userThatFollowed.Username).Result()
		if err != nil {
			return err
		}
		err = cacheDB.LPush(common.BaseCtx, keyStem+"followers", userThatFollowed.Username).Err()
		if err != nil {
			return err
		}
		if removed == 0 {
			err = cacheDB.Incr(common.BaseCtx, keyStem+"followerCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireUserAt("full", userThatWasFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = UpsertUser(userThatWasFollowed.Username, &userThatWasFollowed, objectsToFetch, feedObjectsToFetch, feedObjectOffset)
		if err != nil {
			return err
		}
	}

	if userFollowedInBasic {
		keyStem := GenerateKey("user", "basic", userThatWasFollowed.Username, "")
		// Some version of the user is already cached
		err = cacheDB.Incr(common.BaseCtx, keyStem+"followerCount").Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("basic", userThatWasFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WAS LIKED ALREADY
func UnlikeCacheUpdate(dweet db.DweetModel, userThatLiked db.UserModel, repliesToFetch int, repliesOffset int) error {
	// Check if user that unliked is cached in full
	// If yes, remove dweet ID from likedDweets
	// Check if dweet liked is cached in full
	// If yes, remove user ID from likeUsers
	// If not, cache a full version of the dweet

	userInFull := true
	dweetInFull := true
	dweetInBasic := true

	keyStem := GenerateKey("user", "full", userThatLiked.Username, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "full", dweet.ID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "basic", dweet.ID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userInFull {
		keyStem := GenerateKey("user", "full", userThatLiked.Username, "")
		err = cacheDB.LRem(common.BaseCtx, keyStem+"likedDweets", 1, dweet.ID).Err()
		if err != nil {
			return err
		}
		err = ExpireUserAt("full", userThatLiked.Username, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInFull {
		keyStem := GenerateKey("dweet", "full", dweet.ID, "")
		// Some version of the user is already cached
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"likeUser", 1, userThatLiked.Username).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"likeCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireDweetAt("full", dweet.ID, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = UpsertDweet(dweet.ID, &dweet, repliesToFetch, repliesOffset)
		if err != nil {
			return err
		}
	}

	if dweetInBasic {
		keyStem := GenerateKey("dweet", "basic", dweet.ID, "")
		// Some version of the user is already cached
		err = cacheDB.Decr(common.BaseCtx, keyStem+"likeCount").Err()
		if err != nil {
			return err
		}

		err = ExpireDweetAt("full", dweet.ID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WAS REDWEETED ALREADY
func UnredweetCacheUpdate(redweet db.RedweetModel, userThatUnredweeted db.UserModel) error {
	// Check if user that unredweeted is cached in full
	// If yes, remove redweet ID from feedObjects and redweets fields
	// Destroy the redweet object if in cache
	// UPSERT the redweeted dweet

	userInFull := true
	dweetInFull := true
	dweetInBasic := true

	keyStem := GenerateKey("user", "full", userThatUnredweeted.Username, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "full", redweet.OriginalRedweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "basic", redweet.OriginalRedweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)
	redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)

	if userInFull {
		keyStem := GenerateKey("user", "full", userThatUnredweeted.Username, "")
		err = cacheDB.LRem(common.BaseCtx, keyStem+"redweets", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LRem(common.BaseCtx, keyStem+"feedObjects", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = ExpireUserAt("full", userThatUnredweeted.Username, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInFull {
		keyStem := GenerateKey("dweet", "full", redweet.OriginalRedweetID, "")
		// If user is already in cache, we dont need to cache it

		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"redweetUsers", 1, redweet.AuthorID).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"redweetCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInBasic {
		keyStem := GenerateKey("dweet", "basic", redweet.OriginalRedweetID, "")
		// If user is already in cache, we dont need to cache it
		err = cacheDB.Decr(common.BaseCtx, keyStem+"redweetCount").Err()
		if err != nil {
			return err
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	keyStem = GenerateKey("redweet", "full", redweetID, "")
	dweetMap := []string{
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "redweetOf",
		keyStem + "originalRedweetID",
		keyStem + "redweetTime",
	}

	err = cacheDB.Del(common.BaseCtx, dweetMap...).Err()
	if err != nil {
		return err
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE USER WAS FOLLOWED ALREADY
func UnfollowCacheUpdate(userThatWasFollowed db.UserModel, userThatFollowed db.UserModel, objectsToFetch string, feedObjectsToFetch int, feedObjectOffset int) error {
	// Check if user that unfollowed is cached in full
	// If yes, remove user that was unfollowed ID from following
	// Check if user followed is cached in full
	// If yes, remove user ID from followers
	// If not, cache a full version of the user unfollowed

	userFollowingInFull := true
	userFollowingInBasic := true
	userFollowedInFull := true
	userFollowedInBasic := true

	keyStem := GenerateKey("user", "full", userThatFollowed.Username, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowingInFull = false
		}
		return err
	}

	keyStem = GenerateKey("user", "basic", userThatFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userFollowingInBasic = false
		}
		return err
	}

	keyStem = GenerateKey("user", "full", userThatWasFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			userFollowedInFull = false
		}
		return err
	}

	keyStem = GenerateKey("user", "basic", userThatWasFollowed.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			userFollowedInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userFollowingInFull {
		keyStem := GenerateKey("user", "full", userThatFollowed.Username, "")
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"following", 1, userThatWasFollowed.Username).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"followingCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireUserAt("full", userThatFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = CacheUser("basic", userThatFollowed.Username, &userThatFollowed, "feed", 0, 0)
		if err != nil {
			return err
		}
	}

	if userFollowingInBasic {
		keyStem := GenerateKey("user", "basic", userThatFollowed.Username, "")
		err = cacheDB.Decr(common.BaseCtx, keyStem+"followingCount").Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("basic", userThatFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	}

	if userFollowedInFull {
		keyStem := GenerateKey("dweet", "full", userThatWasFollowed.Username, "")
		// Some version of the user is already cached
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"followers", 1, userThatFollowed.Username).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"followerCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireUserAt("full", userThatWasFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	} else {
		err = UpsertUser(userThatWasFollowed.Username, &userThatWasFollowed, objectsToFetch, feedObjectsToFetch, feedObjectOffset)
		if err != nil {
			return err
		}
	}

	if userFollowedInBasic {
		keyStem := GenerateKey("user", "basic", userThatWasFollowed.Username, "")
		err = cacheDB.Decr(common.BaseCtx, keyStem+"followingCount").Err()
		if err != nil {
			return err
		}

		err = ExpireUserAt("basic", userThatWasFollowed.Username, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func EditDweetCacheUpdate(dweet db.DweetModel, repliesToFetch int, repliesOffset int) error {
	// Check if author is cached in full, if yes, cache basic version of dweet and add it to user object
	err := UpsertDweet(dweet.ID, &dweet, repliesToFetch, repliesOffset)
	if err != nil {
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)
	err = ExpireDweetAt("full", dweet.ID, expireTime)
	if err != nil {
		return err
	}

	keyStem := GenerateKey("dweet", "basic", dweet.ID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	// If dweet is cached in basic, update
	err = CacheDweet("basic", dweet.ID, &dweet, 0, 0)
	if err != nil {
		return err
	}

	return nil
}

func EditUserCacheUpdate(user db.UserModel, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) error {
	// Check if author is cached in full, if yes, cache basic version of dweet and add it to user object
	err := UpsertUser(user.Username, &user, objectsToFetch, feedObjectsToFetch, feedObjectsOffset)
	if err != nil {
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)
	err = ExpireUserAt("full", user.Username, expireTime)
	if err != nil {
		return err
	}

	keyStem := GenerateKey("user", "basic", user.Username, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	// If user is cached in basic, update
	err = CacheUser("basic", user.Username, &user, "feed", 0, 0)
	if err != nil {
		return err
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WAS LIKED ALREADY
func unlikeCacheUpdateInternal(dweetID string, usernameThatLiked string) error {

	userInFull := true
	dweetInFull := true
	dweetInBasic := true

	keyStem := GenerateKey("user", "full", usernameThatLiked, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "full", dweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInFull = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userInFull {
		keyStem := GenerateKey("user", "full", usernameThatLiked, "")
		err = cacheDB.LRem(common.BaseCtx, keyStem+"likedDweets", 1, dweetID).Err()
		if err != nil {
			return err
		}
		err = ExpireUserAt("full", usernameThatLiked, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInFull {
		keyStem := GenerateKey("dweet", "full", dweetID, "")
		// Some version of the user is already cached
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"likeUser", 1, usernameThatLiked).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"likeCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireDweetAt("full", dweetID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInBasic {
		keyStem := GenerateKey("dweet", "basic", dweetID, "")
		// Some version of the user is already cached
		err = cacheDB.Decr(common.BaseCtx, keyStem+"likeCount").Err()
		if err != nil {
			return err
		}

		err = ExpireDweetAt("basic", dweetID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// NOTE: THIS FUNCTION IS ONLY CALLED IF THE DWEET WAS REDWEETED ALREADY
func unredweetCacheUpdateInternal(redweetID string, usernameThatUnredweeted string) error {
	// Check if user that unredweeted is cached in full
	// If yes, remove redweet ID from feedObjects and redweets fields
	// Destroy the redweet object if in cache
	// UPSERT the redweeted dweet

	userInFull := true
	dweetInFull := true
	dweetInBasic := true

	keyStem := GenerateKey("user", "full", usernameThatUnredweeted, "")
	err := cacheDB.Get(common.BaseCtx, keyStem+"username").Err()
	if err != nil {
		// If user isnt cached in full,
		if err == redis.Nil {
			userInFull = false
		}
		return err
	}

	dweetID, _ := ParseRedweetID(redweetID)

	keyStem = GenerateKey("dweet", "full", dweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInFull = false
		}
		return err
	}

	keyStem = GenerateKey("dweet", "basic", dweetID, "")
	err = cacheDB.Get(common.BaseCtx, keyStem+"id").Err()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			dweetInBasic = false
		}
		return err
	}

	expireTime := time.Now().UTC().Add(cacheObjTTL)

	if userInFull {
		keyStem := GenerateKey("user", "full", usernameThatUnredweeted, "")
		err = cacheDB.LRem(common.BaseCtx, keyStem+"redweets", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = cacheDB.LRem(common.BaseCtx, keyStem+"feedObjects", 1, redweetID).Err()
		if err != nil {
			return err
		}
		err = ExpireUserAt("full", usernameThatUnredweeted, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInFull {
		keyStem := GenerateKey("dweet", "full", dweetID, "")
		// If user is already in cache, we dont need to cache it

		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"redweetUsers", 1, usernameThatUnredweeted).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"redweetCount").Err()
			if err != nil {
				return err
			}
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	if dweetInBasic {
		keyStem := GenerateKey("dweet", "basic", dweetID, "")
		// If user is already in cache, we dont need to cache it
		err = cacheDB.Decr(common.BaseCtx, keyStem+"redweetCount").Err()
		if err != nil {
			return err
		}

		err = ExpireRedweetAt(redweetID, expireTime)
		if err != nil {
			return err
		}
	}

	keyStem = GenerateKey("redweet", "full", redweetID, "")
	dweetMap := []string{
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "redweetOf",
		keyStem + "originalRedweetID",
		keyStem + "redweetTime",
	}

	err = cacheDB.Del(common.BaseCtx, dweetMap...).Err()
	if err != nil {
		return err
	}

	return nil
}

// Removal of dweets will destroy the dweet object, all reply dweets and all its redweets
// If the user that posted the dweet is cached in full detail, their feedObjects and dweets fields are UPDATEd
// If the dweet is a reply to a dweet then the dweet replied to is UPDATEd
func DeleteDweetCacheUpdate(dweetID string) error {
	expireTime := time.Now().UTC().Add(cacheObjTTL)

	keyStem := GenerateKey("dweet", "full", dweetID, "")
	replyIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"replyDweets", 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			replyIDs = []string{}
		} else {
			return err
		}
	}
	for _, replyID := range replyIDs {
		DeleteDweetCacheUpdate(replyID)
	}

	likeUserIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"likeUsers", 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			likeUserIDs = []string{}
		} else {
			return err
		}
	}
	for _, likeUserID := range likeUserIDs {
		unlikeCacheUpdateInternal(dweetID, likeUserID)
	}

	redweetUserIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweetUsers", 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			redweetUserIDs = []string{}
		} else {
			return err
		}
	}
	for _, redweetUserID := range redweetUserIDs {
		unredweetCacheUpdateInternal(dweetID, redweetUserID)
	}

	keyStem = GenerateKey("dweet", "basic", dweetID, "")
	dweetMap := []string{
		keyStem + "dweetBody",
		keyStem + "id",
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "postedAt",
		keyStem + "lastUpdatedAt",
		keyStem + "likeCount",
		keyStem + "isReply",
		keyStem + "originalReplyID",
		keyStem + "replyCount",
		keyStem + "redweetCount",
		keyStem + "media",
	}
	err = cacheDB.Del(common.BaseCtx, dweetMap...).Err()
	if err != nil {
		return err
	}

	isReply := true
	keyStem = GenerateKey("dweet", "full", dweetID, "")
	replyID, err := cacheDB.Get(common.BaseCtx, keyStem+"replyTo").Result()
	if err != nil {
		// If dweet isnt cached in full,
		if err == redis.Nil {
			isReply = false
		}
		return err
	}
	if isReply {
		keyStem = GenerateKey("dweet", "full", replyID, "")
		// If user is already in cache, we dont need to cache it
		removed, err := cacheDB.LRem(common.BaseCtx, keyStem+"replyDweets", 1, dweetID).Result()
		if err != nil {
			return err
		}
		if removed != 0 {
			err = cacheDB.Decr(common.BaseCtx, keyStem+"replyCount").Err()
			if err != nil {
				return err
			}
		}
		err = ExpireDweetAt("full", replyID, expireTime)
		if err != nil {
			return err
		}

		keyStem = GenerateKey("dweet", "basic", replyID, "")
		// If user is already in cache, we dont need to cache it
		err = cacheDB.Decr(common.BaseCtx, keyStem+"replyCount").Err()
		if err != nil {
			return err
		}
		err = ExpireDweetAt("basic", replyID, expireTime)
		if err != nil {
			return err
		}
	}

	keyStem = GenerateKey("dweet", "full", dweetID, "")
	dweetMap = []string{
		keyStem + "dweetBody",
		keyStem + "id",
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "postedAt",
		keyStem + "lastUpdatedAt",
		keyStem + "likeCount",
		keyStem + "isReply",
		keyStem + "originalReplyID",
		keyStem + "replyCount",
		keyStem + "redweetCount",
		keyStem + "media",
		keyStem + "replyTo",
	}
	err = cacheDB.Del(common.BaseCtx, dweetMap...).Err()
	if err != nil {
		return err
	}

	return nil
}
