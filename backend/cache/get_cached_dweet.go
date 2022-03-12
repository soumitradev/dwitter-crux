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
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

func GetCachedDweetFull(id string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	keyStem := GenerateKey("dweet", "full", id, "")
	keyList := []string{
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
	}
	valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
	if err != nil {
		return schema.DweetType{}, err
	}

	likeCount, err := strconv.Atoi(valList[6].(string))
	if err != nil {
		return schema.DweetType{}, err
	}
	replyCount, err := strconv.Atoi(valList[9].(string))
	if err != nil {
		return schema.DweetType{}, err
	}
	redweetCount, err := strconv.Atoi(valList[10].(string))
	if err != nil {
		return schema.DweetType{}, err
	}

	isReply := false
	if valList[7].(string) == "true" {
		isReply = true
	}

	author, err := GetCachedUserBasic(valList[3].(string))
	if err != nil {
		return schema.DweetType{}, err
	}

	mediaLinks, err := cacheDB.LRange(common.BaseCtx, keyStem+"media", 0, -1).Result()
	if err != nil {
		return schema.DweetType{}, err
	}

	postedAt, err := time.Parse(util.TimeUTCFormat, valList[4].(string))
	if err != nil {
		return schema.DweetType{}, err
	}
	lastUpdatedAt, err := time.Parse(util.TimeUTCFormat, valList[5].(string))
	if err != nil {
		return schema.DweetType{}, err
	}
	replyTo := schema.BasicDweetType{}
	if isReply {
		replyTo, err = GetCachedDweetBasic(valList[8].(string))
		if err != nil {
			return schema.DweetType{}, err
		}
	}

	cachedDweet := schema.DweetType{
		DweetBody:       valList[0].(string),
		ID:              valList[1].(string),
		Author:          author,
		AuthorID:        valList[3].(string),
		PostedAt:        postedAt,
		LastUpdatedAt:   lastUpdatedAt,
		LikeCount:       likeCount,
		IsReply:         isReply,
		OriginalReplyID: valList[8].(string),
		ReplyTo:         replyTo,
		ReplyCount:      replyCount,
		RedweetCount:    redweetCount,
		Media:           mediaLinks,
	}

	replyIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"replyDweets", 0, -1).Result()
	if err != nil {
		return schema.DweetType{}, err
	}
	hitIDs, hitInfo, err := GetCacheHit(replyIDs, repliesToFetch, replyOffset)
	if err != nil {
		return schema.DweetType{}, err
	}
	expireTime := time.Now().UTC().Add(time.Hour)
	var replyList []schema.BasicDweetType
	objCount := 0
	if repliesToFetch < 0 {
		replyList = make([]schema.BasicDweetType, 0)
		for i, replyID := range hitIDs {
			if IsStub(replyID) {
				num, err := ParseStub(replyID)
				if err != nil {
					return schema.DweetType{}, err
				}
				if num < 0 {
					num = repliesToFetch - i
				}
				objects, err := ResolvePartialHitDweet(id, num, replyOffset+i)
				if err != nil {
					return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
				}
				for _, obj := range objects {
					if dweet, ok := obj.(db.DweetModel); ok {
						replyList = append(replyList, schema.FormatAsBasicDweetType(&dweet))
						objCount++
						err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
						if err != nil {
							return schema.DweetType{}, err
						}
					} else {
						return schema.DweetType{}, errors.New("internal server error")
					}
				}
				newIDList := make([]string, 0, len(replyIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
				newIDList = append(newIDList, replyIDs[:hitInfo.FirstIndex]...)
				newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
				newIDList = append(newIDList, replyIDs[hitInfo.LastIndex+1:]...)
				interfaceIDList := make([]interface{}, len(replyIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
				for listIndex, v := range newIDList {
					interfaceIDList[listIndex] = v
				}
				err = cacheDB.Del(common.BaseCtx, keyStem+"replyDweets").Err()
				if err != nil {
					return schema.DweetType{}, err
				}
				cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", interfaceIDList...)
				ExpireDweetAt("full", id, expireTime)
			} else {
				feedObject, err := GetCachedDweetBasic(replyID)
				if err != nil {
					return schema.DweetType{}, err
				}
				replyList = append(replyList, feedObject)
				objCount++
			}
		}
	} else {
		replyList = make([]schema.BasicDweetType, repliesToFetch)
		for i, feedObjectID := range hitIDs {
			if IsStub(feedObjectID) {
				num, err := ParseStub(feedObjectID)
				if err != nil {
					return schema.DweetType{}, err
				}
				if num < 0 {
					num = repliesToFetch - i
				}
				objects, err := ResolvePartialHitDweet(id, num, replyOffset+i)
				if err != nil {
					return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
				}
				for _, obj := range objects {
					if dweet, ok := obj.(db.DweetModel); ok {
						replyList[objCount] = schema.FormatAsBasicDweetType(&dweet)
						err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
						if err != nil {
							return schema.DweetType{}, err
						}
					} else {
						return schema.DweetType{}, errors.New("internal server error")
					}
					objCount++
				}
				newIDList := make([]string, 0, len(replyIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
				newIDList = append(newIDList, replyIDs[:hitInfo.FirstIndex]...)
				newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
				newIDList = append(newIDList, replyIDs[hitInfo.LastIndex+1:]...)
				interfaceIDList := make([]interface{}, len(replyIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
				for listIndex, v := range newIDList {
					interfaceIDList[listIndex] = v
				}
				err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
				if err != nil {
					return schema.DweetType{}, err
				}
				cacheDB.LPush(common.BaseCtx, keyStem+"dweets", interfaceIDList...)
				ExpireDweetAt("full", id, expireTime)
			} else {
				reply, err := GetCachedDweetBasic(feedObjectID)
				if err != nil {
					return schema.DweetType{}, err
				}
				replyList[objCount] = reply
				objCount++
			}
		}
	}
	cachedDweet.ReplyDweets = replyList[:objCount]

	likeUsers, err := cacheDB.LRange(common.BaseCtx, keyStem+"likeUsers", 0, -1).Result()
	if err != nil {
		return schema.DweetType{}, err
	}

	likeUserList := make([]schema.BasicUserType, len(likeUsers))
	for i, likeUserUsername := range likeUsers {
		likeUser, err := GetCachedUserBasic(likeUserUsername)
		if err != nil {
			return schema.DweetType{}, err
		}
		likeUserList[i] = likeUser
	}

	cachedDweet.LikeUsers = likeUserList

	redweetUsers, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweetUsers", 0, -1).Result()
	if err != nil {
		return schema.DweetType{}, err
	}

	redweetUserList := make([]schema.BasicUserType, len(redweetUsers))
	for i, redweetUserUsername := range redweetUsers {
		redweetUser, err := GetCachedUserBasic(redweetUserUsername)
		if err != nil {
			return schema.DweetType{}, err
		}
		likeUserList[i] = redweetUser
	}

	cachedDweet.RedweetUsers = redweetUserList

	// ReplyDweets     []BasicDweetType `json:"replyDweets"`

	return cachedDweet, nil
}

func GetCachedDweetBasic(id string) (schema.BasicDweetType, error) {
	keyStem := GenerateKey("dweet", "full", id, "")
	keyList := []string{
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
	}
	valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
	if err != nil {
		// If full user is not found, check for basic user
		if err == redis.Nil {
			keyStem := GenerateKey("dweet", "basic", id, "")
			keyList := []string{
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
			}
			valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
			if err != nil {
				return schema.BasicDweetType{}, err
			}

			likeCount, err := strconv.Atoi(valList[6].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}
			replyCount, err := strconv.Atoi(valList[9].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}
			redweetCount, err := strconv.Atoi(valList[10].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}

			isReply := false
			if valList[7].(string) == "true" {
				isReply = true
			}

			author, err := GetCachedUserBasic(valList[3].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}

			mediaLinks, err := cacheDB.LRange(common.BaseCtx, keyStem+"media", 0, -1).Result()
			if err != nil {
				return schema.BasicDweetType{}, err
			}

			postedAt, err := time.Parse(util.TimeUTCFormat, valList[4].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}
			lastUpdatedAt, err := time.Parse(util.TimeUTCFormat, valList[5].(string))
			if err != nil {
				return schema.BasicDweetType{}, err
			}
			cachedDweet := schema.BasicDweetType{
				DweetBody:       valList[0].(string),
				ID:              valList[1].(string),
				Author:          author,
				AuthorID:        valList[3].(string),
				PostedAt:        postedAt,
				LastUpdatedAt:   lastUpdatedAt,
				LikeCount:       likeCount,
				IsReply:         isReply,
				OriginalReplyID: valList[8].(string),
				ReplyCount:      replyCount,
				RedweetCount:    redweetCount,
				Media:           mediaLinks,
			}
			return cachedDweet, nil
		}
		return schema.BasicDweetType{}, err
	}

	likeCount, err := strconv.Atoi(valList[6].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}
	replyCount, err := strconv.Atoi(valList[9].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}
	redweetCount, err := strconv.Atoi(valList[10].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}

	isReply := false
	if valList[7].(string) == "true" {
		isReply = true
	}

	author, err := GetCachedUserBasic(valList[3].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}

	mediaLinks, err := cacheDB.LRange(common.BaseCtx, keyStem+"media", 0, -1).Result()
	if err != nil {
		return schema.BasicDweetType{}, err
	}

	postedAt, err := time.Parse(util.TimeUTCFormat, valList[4].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}
	lastUpdatedAt, err := time.Parse(util.TimeUTCFormat, valList[5].(string))
	if err != nil {
		return schema.BasicDweetType{}, err
	}
	cachedDweet := schema.BasicDweetType{
		DweetBody:       valList[0].(string),
		ID:              valList[1].(string),
		Author:          author,
		AuthorID:        valList[3].(string),
		PostedAt:        postedAt,
		LastUpdatedAt:   lastUpdatedAt,
		LikeCount:       likeCount,
		IsReply:         isReply,
		OriginalReplyID: valList[8].(string),
		ReplyCount:      replyCount,
		RedweetCount:    redweetCount,
		Media:           mediaLinks,
	}
	return cachedDweet, nil
}
