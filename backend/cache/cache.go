// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"os"

	"github.com/go-redis/redis/v8"
)

/*
# Cache Design Philosophy

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

Unrequested subtypes will not be cached, and will be represented with "<?>"

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
- user:full:myUsername:dweets = "<?>"
- user:full:myUsername:redweets = "<?>"
- user:full:myUsername:feedObjects = [abcdefghi1, abcdefghi2, abcdefghi3, abcdefghi4, abcdefghi5]
- user:full:myUsername:redweetedDweets = "<?>"
- user:full:myUsername:likedDweets = "<?>"
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

- user:basic:myUsername:username = "otherUsername"
- user:basic:myUsername:name = "Other Name"
- user:basic:myUsername:email = "myotheremail@mail.com"
- user:basic:myUsername:bio = "Different bio"
- user:basic:myUsername:pfpURL = "https://..."
- user:basic:myUsername:followerCount = "1829"
- user:basic:myUsername:followingCount = "2729"
- user:basic:myUsername:createdAt = "2022-03-05T04:20:49.962Z"

I might consider pulling full objects from the DB when basic ones are needed just for the cache, but meh
sounds like more work, and its also pretty inefficient. Plus, where do you end?
Do you have a caching depth where you only go to some depth of pulling full objects?
Why pull data when not needed? Doesn't that destroy the whole point of a "cache"?

I might also consider omitting the ID fields for relations since they're useless and take up extra memory

Lots of memory optimization can be done by reducing key name sizes too

But hey this is already pretty painful, and I'm almost next to sure that something will break along the way.

Whew that was a lot of work
*/

var cacheDB *redis.Client

func InitCache() {
	cacheDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6421",
		Password: os.Getenv("REDIS_6421_PASS"),
		DB:       0,
	})
}
