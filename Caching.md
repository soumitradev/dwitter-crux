# Cache Design Philosophy

Cached objects are cached with an hour-long expiry time. Why? idk.

Dweets and user objects will be cached as seperate keys with key names such as:

- `user:full:myUsername:username`
- `user:full:myUsername:email`

- `dweet:full:aa6A9k71Ab:dweetBody`
- `dweet:full:aa6A9k71Ab:likeUsers`
- `dweet:full:aa6A9k71Ab:author`

and basic objects will be cached as:

- `user:basic:myUsername:username`
- `user:basic:myUsername:email`

- `dweet:basic:aa6A9k71Ab:dweetBody`
- `dweet:basic:aa6A9k71Ab:author`

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

Paginated results are cached with a <`skip`> tag before and a <?> tag after.

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

This means that <`skip`> tags can occur in the middle of lists, and there might be multiple such tags in a list.

Luckily, since dweets are sorted by creation time and not edit time, stranger things won't happen with this where things move around once added.

So, for example, if we cache a User object with 5 feedObjects (4 dweets and 1 redweet),

We will first save the keys:

- `user:full:myUsername:username = "myUsername"`
- `user:full:myUsername:name = "My Name"`
- `user:full:myUsername:email = "myemail@mail.com"`
- `user:full:myUsername:bio = "Example bio"`
- `user:full:myUsername:pfpURL = "https://..."`
- `user:full:myUsername:dweets = ["<?>"]`
- `user:full:myUsername:redweets = ["<?>"]`
- `user:full:myUsername:feedObjects = [abcdefghi1, abcdefghi2, abcdefghi3, abcdefghi4, abcdefghi5]`
- `user:full:myUsername:redweetedDweets = ["<?>"]`
- `user:full:myUsername:likedDweets = ["<?>"]`
- `user:full:myUsername:followerCount = "1236"`
- `user:full:myUsername:followers = [follower1, follower2, ..., commonfollower1236]`
- `user:full:myUsername:followingCount = "2317"`
- `user:full:myUsername:following = [followed1, followed2, ..., followed2317]`
- `user:full:myUsername:createdAt = "2022-03-08T04:20:49.962Z"`

and then, we will cache

- `dweet:basic:abcdefghi1:dweetBody = "bruhh 💀💀💀"`
- `dweet:basic:abcdefghi1:id = "abcdefghi1"`
- `dweet:basic:abcdefghi1:author = "myUsername"`
- `dweet:basic:abcdefghi1:authorID = "myUsername"`
- `dweet:basic:abcdefghi1:postedAt = "2022-03-10T04:20:49.962Z"`
- `dweet:basic:abcdefghi1:lastUpdatedAt = "2022-03-10T04:20:49.962Z"`
- `dweet:basic:abcdefghi1:likeCount = "10"`
- `dweet:basic:abcdefghi1:isReply = "false"`
- `dweet:basic:abcdefghi1:originalReplyID = ""`
- `dweet:basic:abcdefghi1:replyCount = "3"`
- `dweet:basic:abcdefghi1:redweetCount = "4"`
- `dweet:basic:abcdefghi1:media = [https://..., https://...]`

and so on for the rest of the dweets

and for the redweet,

- `redweet:full:Redweet(myUsername, abcdefghi8):author = "myUsername"`
- `redweet:full:Redweet(myUsername, abcdefghi8):authorID = "myUsername"`
- `redweet:full:Redweet(myUsername, abcdefghi8):redweetOf = "abcdefghi8"`
- `redweet:full:Redweet(myUsername, abcdefghi8):originalRedweetID = "abcdefghi8"`
- `redweet:full:Redweet(myUsername, abcdefghi8):redweetTime = "2022-03-9T04:20:49.962Z"`

and then since a full user with username "myUsername" already exists (we check the key "user:full:myUsername:username"), we grab the information from there

we will also have to cache a basic version of the original dweet abcdefghi8

and then, we will cache

- `dweet:basic:abcdefghi8:dweetBody = "can someone please redweet this one"`
- `dweet:basic:abcdefghi8:id = "abcdefghi8"`
- `dweet:basic:abcdefghi8:author = "otherUsername"`
- `dweet:basic:abcdefghi8:authorID = "otherUsername"`
- `dweet:basic:abcdefghi8:postedAt = "2022-03-08T04:20:49.962Z"`
- `dweet:basic:abcdefghi8:lastUpdatedAt = "2022-03-08T04:20:49.962Z"`
- `dweet:basic:abcdefghi8:likeCount = "23"`
- `dweet:basic:abcdefghi8:isReply = "false"`
- `dweet:basic:abcdefghi8:originalReplyID = ""`
- `dweet:basic:abcdefghi8:replyCount = "6"`
- `dweet:basic:abcdefghi8:redweetCount = "7"`
- `dweet:basic:abcdefghi8:media = []`

this will also lead us to cache the other user with username otherUsername

- `user:basic:otherUsername:username = "otherUsername"`
- `user:basic:otherUsername:name = "Other Name"`
- `user:basic:otherUsername:email = "myotheremail@mail.com"`
- `user:basic:otherUsername:bio = "Different bio"`
- `user:basic:otherUsername:pfpURL = "https://..."`
- `user:basic:otherUsername:followerCount = "1829"`
- `user:basic:otherUsername:followingCount = "2729"`
- `user:basic:otherUsername:createdAt = "2022-03-05T04:20:49.962Z"`

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

# Mutation caching philosophy

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

1. Eviction on mutation: De-cache something when its mutated (how does that make sense though, why does liking a post make it uncache? 🤔)
2. Update cache on mutation: Requires much more complex logic, and might even make uncached mutations slower (since we might have to pull additional info from DB)

===

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

> "I liked this dweet so I have like 5 replies to this dweet, see if this info helps"
>
> "Sure thanks for the first 5 replies I only had like 1"
>
> _grabs the current reply list, throws those 5 replies when needed (when encountering a stub) and caches them_
>
> Similarly for users
