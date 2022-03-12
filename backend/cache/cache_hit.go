// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"errors"
	"fmt"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/util"
)

func GetCacheHit(cachedList []string, objectsToFetch int, objectsOffset int) (hitIDs []string, partialHit PartialHitType, err error) {
	// Returns IDs of requested objects from cached objects, includes stubs on partial hits
	originalObjectsToFetch := objectsToFetch
	i := 0
	for objectsOffset != 0 && i < len(cachedList) {
		if IsStub(cachedList[i]) {
			jump, err := ParseStub(cachedList[i])
			if err != nil {
				return []string{}, PartialHitType{}, err
			}
			if jump == -1 {
				// Full miss
				hit := PartialHitType{
					isPartial:  false,
					FirstIndex: 0,
					LastIndex:  0,
				}
				return []string{uncachedStub}, hit, nil
			}
			objectsOffset -= jump
		} else {
			objectsOffset--
		}
		i++
	}

	firstStub := len(cachedList)
	lastStub := i - 1
	j := i
	for objectsToFetch != 0 && j < len(cachedList) {
		if IsStub(cachedList[j]) {
			firstStub = util.Min(firstStub, j)
			lastStub = j
			jump, err := ParseStub(cachedList[j])
			if err != nil {
				return []string{}, PartialHitType{}, err
			}
			if jump == -1 {
				// Partial hit with <?> at end
				partialCacheHit := make([]string, 0, firstStub-i+1)
				partialCacheHit = append(partialCacheHit, cachedList[i:firstStub]...)
				partialCacheHit = append(partialCacheHit, uncachedStub)
				hit := PartialHitType{
					isPartial:  true,
					FirstIndex: j,
					LastIndex:  j,
				}
				return partialCacheHit, hit, nil
			}
			objectsToFetch -= jump
		} else {
			objectsToFetch--
		}
		j++
	}

	if firstStub == len(cachedList) || lastStub == i-1 {
		// If firstStub == len(cachedList) or if lastStub == i - 1, then there are no stubs, meaning it is a full hit
		hit := PartialHitType{
			isPartial:  false,
			FirstIndex: 0,
			LastIndex:  0,
		}
		return cachedList[i:j], hit, nil
	} else {
		// Otherwise, there is atleast one stub, and we can find the number uncached values as lastStub - firstStub + 1 starting at firstStub
		partialCacheHit := make([]string, 0, originalObjectsToFetch-lastStub+firstStub)
		partialCacheHit = append(partialCacheHit, cachedList[i:firstStub]...)
		partialCacheHit = append(partialCacheHit, "<"+fmt.Sprintf("%d", lastStub-firstStub+1)+">")
		partialCacheHit = append(partialCacheHit, cachedList[lastStub+1:j]...)

		hit := PartialHitType{
			isPartial:  true,
			FirstIndex: firstStub,
			LastIndex:  lastStub,
		}
		return partialCacheHit, hit, nil
	}
}

func ResolvePartialHitUser(id string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) ([]interface{}, error) {
	var user *db.UserModel
	var err error
	switch objectsToFetch {
	case "feed":
		if feedObjectsToFetch < 0 {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
			return merged, nil
		} else {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
			).Exec(common.BaseCtx)

			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}
			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			return merged[:util.Min(feedObjectsToFetch, len(merged))], nil
		}
	case "dweet":
		if feedObjectsToFetch < 0 {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userDweets := user.Dweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userDweets[i]
			}
			return feedObjectList, nil
		} else {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userDweets := user.Dweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userDweets[i]
			}
			return feedObjectList, nil
		}
	case "redweet":
		if feedObjectsToFetch < 0 {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userRedweets := user.Redweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userRedweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userRedweets[i]
			}
			return feedObjectList, nil
		} else {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userRedweets := user.Redweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userRedweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userRedweets[i]
			}
			return feedObjectList, nil

		}
	case "redweetedDweet":
		if feedObjectsToFetch < 0 {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.RedweetedDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userRedweetedDweets := user.RedweetedDweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userRedweetedDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userRedweetedDweets[i]
			}
			return feedObjectList, nil
		} else {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.RedweetedDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userRedweetedDweets := user.RedweetedDweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userRedweetedDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userRedweetedDweets[i]
			}
			return feedObjectList, nil
		}
	case "liked":
		if feedObjectsToFetch < 0 {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.LikedDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userLikedDweets := user.LikedDweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userLikedDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userLikedDweets[i]
			}
			return feedObjectList, nil
		} else {
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(id),
			).With(
				db.User.LikedDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return []interface{}{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return []interface{}{}, fmt.Errorf("internal server error: %v", err)
			}

			userLikedDweets := user.LikedDweets()
			feedObjectList := make([]interface{}, util.Min(feedObjectsToFetch, len(userLikedDweets)))
			for i := range feedObjectList {
				feedObjectList[i] = userLikedDweets[i]
			}
			return feedObjectList, nil
		}
	default:
		return []interface{}{}, errors.New("unknown objectsToFetch")
	}
}

func ResolvePartialHitDweet(id string, repliesToFetch int, replyOffset int) ([]interface{}, error) {
	var dweet *db.DweetModel
	var err error

	if repliesToFetch < 0 {
		dweet, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(id),
		).With(
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			).Take(repliesToFetch).Skip(replyOffset),
		).Exec(common.BaseCtx)
	} else {
		dweet, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(id),
		).With(
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			).Take(repliesToFetch).Skip(replyOffset),
		).Exec(common.BaseCtx)
	}
	if err == db.ErrNotFound {
		return []interface{}{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return []interface{}{}, fmt.Errorf("internal server error: %v", err)
	}

	replyDweets := dweet.ReplyDweets()
	replyObjectList := make([]interface{}, util.Min(repliesToFetch, len(replyDweets)))
	for i := range replyObjectList {
		replyObjectList[i] = replyDweets[i]
	}
	return replyObjectList, nil
}
