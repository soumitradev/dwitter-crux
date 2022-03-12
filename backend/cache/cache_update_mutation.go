// Package cache provides useful functions to use the Redis LRU cache
package cache

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

createDweet UPSERTS the dweet created.
We also use these dweets in paginated results for full-detail-level cached users, so we will UPDATE such users.

Counterintuitively, createReply will UPSERT the dweet replied to in basic detail and the full version of the reply dweet
The original dweet would ideally be cached in full detail because replies indicate a level of usefulness of the dweet replied to.
But, this would result in more DB fetches, which would make uncached requests slower.
Additionally, since this is also a dweet, it will also have the same behaviour as createDweet, so the author is UPDATEd if cached in full detail

Redweets will UPSERT the redweet object in full detail, and a basic version of the user that redweets it and the dweet redweeted (since redweetUsers is a field on the original dweet)
If the user redweeting is cached in full detail, the redweet object is added to the user's feedObjects and redweets fields
If the dweet redweeted is cached in full detail, the redweetUsers object is updated

Follows will UPSERT the followed user in full detail, and the following user in basic detail (due to the followers field)
If the follower is cached in full detail additionally, their following field is also UPDATEd

Likes will UPSERT the dweet liked in full detail, and the user that likes it (since likeUsers is a field on the original dweet)
If the user that likes the dweet is cached in full detail, their likedDweets field is UPDATEd

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
