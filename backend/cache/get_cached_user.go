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

func GetCachedUserBasic(id string) (user schema.BasicUserType, err error) {
	keyStem := GenerateKey("user", "full", id, "")
	keyList := []string{
		keyStem + "username",
		keyStem + "name",
		keyStem + "email",
		keyStem + "bio",
		keyStem + "pfpURL",
		keyStem + "followerCount",
		keyStem + "followingCount",
		keyStem + "createdAt",
	}
	valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
	if err != nil {
		// If full user is not found, check for basic user
		if err == redis.Nil {
			keyStem := GenerateKey("user", "basic", id, "")
			keyList := []string{
				keyStem + "username",
				keyStem + "name",
				keyStem + "email",
				keyStem + "bio",
				keyStem + "pfpURL",
				keyStem + "followerCount",
				keyStem + "followingCount",
				keyStem + "createdAt",
			}
			valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
			if err != nil {
				return schema.BasicUserType{}, err
			}

			followerCount := 0
			if valList[5] == nil {
				return schema.BasicUserType{}, redis.Nil
			} else if followerCountString, ok := valList[5].(string); ok {
				followerCount, err = strconv.Atoi(followerCountString)
				if err != nil {
					return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
				}
			} else {
				return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
			}

			followingCount := 0
			if valList[6] == nil {
				return schema.BasicUserType{}, redis.Nil
			} else if followingCountString, ok := valList[6].(string); ok {
				followingCount, err = strconv.Atoi(followingCountString)
				if err != nil {
					return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
				}
			} else {
				return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
			}

			createdAt := time.Now().UTC()
			if valList[7] == nil {
				return schema.BasicUserType{}, redis.Nil
			} else if createdAtString, ok := valList[7].(string); ok {
				createdAt, err = time.Parse(util.TimeUTCFormat, createdAtString)
				if err != nil {
					return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
				}
			} else {
				return schema.BasicUserType{}, fmt.Errorf("internal server error: %v", err)
			}
			cachedUser := schema.BasicUserType{
				Username:       valList[0].(string),
				Name:           valList[1].(string),
				Email:          valList[2].(string),
				Bio:            valList[3].(string),
				PfpURL:         valList[4].(string),
				FollowerCount:  followerCount,
				FollowingCount: followingCount,
				CreatedAt:      createdAt,
			}
			return cachedUser, err
		}
		return schema.BasicUserType{}, err
	}
	followerCount, err := strconv.Atoi(valList[5].(string))
	if err != nil {
		return schema.BasicUserType{}, err
	}
	followingCount, err := strconv.Atoi(valList[6].(string))
	if err != nil {
		return schema.BasicUserType{}, err
	}
	createdAt, err := time.Parse(util.TimeUTCFormat, valList[7].(string))
	if err != nil {
		return schema.BasicUserType{}, err
	}
	cachedUser := schema.BasicUserType{
		Username:       valList[0].(string),
		Name:           valList[1].(string),
		Email:          valList[2].(string),
		Bio:            valList[3].(string),
		PfpURL:         valList[4].(string),
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		CreatedAt:      createdAt,
	}
	return cachedUser, nil
}

func GetCachedUserFull(id string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (user schema.UserType, err error) {
	// Check a bunch of keys at a time and return if cache hit
	keyStem := GenerateKey("user", "full", id, "")
	keyList := []string{
		keyStem + "username",
		keyStem + "name",
		keyStem + "email",
		keyStem + "bio",
		keyStem + "pfpURL",
		keyStem + "followerCount",
		keyStem + "followingCount",
		keyStem + "createdAt",
	}
	valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
	if err != nil {
		return schema.UserType{}, err
	}

	followerCount := 0
	if valList[5] == nil {
		return schema.UserType{}, redis.Nil
	} else if followerCountString, ok := valList[5].(string); ok {
		followerCount, err = strconv.Atoi(followerCountString)
		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}
	} else {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	followingCount := 0
	if valList[6] == nil {
		return schema.UserType{}, redis.Nil
	} else if followingCountString, ok := valList[6].(string); ok {
		followingCount, err = strconv.Atoi(followingCountString)
		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}
	} else {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	createdAt := time.Now().UTC()
	if valList[7] == nil {
		return schema.UserType{}, redis.Nil
	} else if createdAtString, ok := valList[7].(string); ok {
		createdAt, err = time.Parse(util.TimeUTCFormat, createdAtString)
		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}
	} else {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	cachedUser := schema.UserType{
		Username:       valList[0].(string),
		Name:           valList[1].(string),
		Email:          valList[2].(string),
		Bio:            valList[3].(string),
		PfpURL:         valList[4].(string),
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		CreatedAt:      createdAt,
	}

	// Check feed, dweets, redweets etc. caching, and handle partial hit/miss
	switch objectsToFetch {
	case "feed":
		feedObjectIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"feedObjects", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				feedObjectIDs = []string{}
			} else {
				return schema.UserType{}, err
			}
		}
		hitIDs, hitInfo, err := GetCacheHit(feedObjectIDs, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return schema.UserType{}, err
		}
		expireTime := time.Now().UTC().Add(cacheObjTTL)
		var feedObjectList []interface{}
		objCount := 0
		if feedObjectsToFetch < 0 {
			feedObjectList = make([]interface{}, 0)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					// This is great and all but we also need to add the thing to the user object
					// Also reset the expiry of user
					// We have fetched num results after feedObjectsOffset+i results, so update the user feed object
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsBasicDweetType(&dweet))
							objCount++
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else if redweet, ok := obj.(db.RedweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsRedweetType(&redweet))
							objCount++
							err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"feedObjects").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					var feedObject interface{}
					if isRedweet(feedObjectID) {
						feedObject, err = GetCachedRedweet(feedObjectID)
					} else {
						feedObject, err = GetCachedDweetBasic(feedObjectID)
					}
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList = append(feedObjectList, feedObject)
					objCount++
				}
			}
		} else {
			feedObjectList = make([]interface{}, feedObjectsToFetch)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsBasicDweetType(&dweet)
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else if redweet, ok := obj.(db.RedweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsRedweetType(&redweet)
							err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
						objCount++
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"feedObjects").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					var feedObject interface{}
					if isRedweet(feedObjectID) {
						feedObject, err = GetCachedRedweet(feedObjectID)
					} else {
						feedObject, err = GetCachedDweetBasic(feedObjectID)
					}
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList[objCount] = feedObject
					objCount++
				}
			}
		}
		cachedUser.FeedObjects = feedObjectList[:objCount]
	case "dweet":
		feedObjectIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"dweets", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				feedObjectIDs = []string{}
			} else {
				return schema.UserType{}, err
			}
		}
		hitIDs, hitInfo, err := GetCacheHit(feedObjectIDs, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return schema.UserType{}, err
		}
		expireTime := time.Now().UTC().Add(cacheObjTTL)
		var feedObjectList []schema.BasicDweetType
		objCount := 0
		if feedObjectsToFetch < 0 {
			feedObjectList = make([]schema.BasicDweetType, 0)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					// This is great and all but we also need to add the thing to the user object
					// Also reset the expiry of user
					// We have fetched num results after feedObjectsOffset+i results, so update the user feed object
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsBasicDweetType(&dweet))
							objCount++
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"dweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList = append(feedObjectList, feedObject)
					objCount++
				}
			}
		} else {
			feedObjectList = make([]schema.BasicDweetType, feedObjectsToFetch)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsBasicDweetType(&dweet)
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
						objCount++
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"dweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList[objCount] = feedObject
					objCount++
				}
			}
		}
		cachedUser.Dweets = feedObjectList[:objCount]
	case "redweet":
		feedObjectIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweets", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				feedObjectIDs = []string{}
			} else {
				return schema.UserType{}, err
			}
		}
		hitIDs, hitInfo, err := GetCacheHit(feedObjectIDs, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return schema.UserType{}, err
		}
		expireTime := time.Now().UTC().Add(cacheObjTTL)
		var feedObjectList []schema.RedweetType
		objCount := 0
		if feedObjectsToFetch < 0 {
			feedObjectList = make([]schema.RedweetType, 0)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					// This is great and all but we also need to add the thing to the user object
					// Also reset the expiry of user
					// We have fetched num results after feedObjectsOffset+i results, so update the user feed object
					for _, obj := range objects {
						if redweet, ok := obj.(db.RedweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsRedweetType(&redweet))
							objCount++
							err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"redweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedRedweet(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList = append(feedObjectList, feedObject)
					objCount++
				}
			}
		} else {
			feedObjectList = make([]schema.RedweetType, feedObjectsToFetch)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					for _, obj := range objects {
						if redweet, ok := obj.(db.RedweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsRedweetType(&redweet)
							err := CacheRedweet("full", ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID), &redweet)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
						objCount++
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"redweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedRedweet(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList[objCount] = feedObject
					objCount++
				}
			}
		}
		cachedUser.Redweets = feedObjectList[:objCount]
	case "redweetedDweet":
		feedObjectIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweetedDweets", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				feedObjectIDs = []string{}
			} else {
				return schema.UserType{}, err
			}
		}
		hitIDs, hitInfo, err := GetCacheHit(feedObjectIDs, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return schema.UserType{}, err
		}
		expireTime := time.Now().UTC().Add(cacheObjTTL)
		var feedObjectList []schema.BasicDweetType
		objCount := 0
		if feedObjectsToFetch < 0 {
			feedObjectList = make([]schema.BasicDweetType, 0)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					// This is great and all but we also need to add the thing to the user object
					// Also reset the expiry of user
					// We have fetched num results after feedObjectsOffset+i results, so update the user feed object
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsBasicDweetType(&dweet))
							objCount++
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList = append(feedObjectList, feedObject)
					objCount++
				}
			}
		} else {
			feedObjectList = make([]schema.BasicDweetType, feedObjectsToFetch)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsBasicDweetType(&dweet)
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
						objCount++
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList[objCount] = feedObject
					objCount++
				}
			}
		}
		cachedUser.RedweetedDweets = feedObjectList[:objCount]
	case "liked":
		feedObjectIDs, err := cacheDB.LRange(common.BaseCtx, keyStem+"likedDweets", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				feedObjectIDs = []string{}
			} else {
				return schema.UserType{}, err
			}
		}
		hitIDs, hitInfo, err := GetCacheHit(feedObjectIDs, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return schema.UserType{}, err
		}
		expireTime := time.Now().UTC().Add(cacheObjTTL)
		var feedObjectList []schema.BasicDweetType
		objCount := 0
		if feedObjectsToFetch < 0 {
			feedObjectList = make([]schema.BasicDweetType, 0)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					// This is great and all but we also need to add the thing to the user object
					// Also reset the expiry of user
					// We have fetched num results after feedObjectsOffset+i results, so update the user feed object
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList = append(feedObjectList, schema.FormatAsBasicDweetType(&dweet))
							objCount++
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList = append(feedObjectList, feedObject)
					objCount++
				}
			}
		} else {
			feedObjectList = make([]schema.BasicDweetType, feedObjectsToFetch)
			for i, feedObjectID := range hitIDs {
				if IsStub(feedObjectID) {
					num, err := ParseStub(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					if num < 0 {
						num = feedObjectsToFetch - i
					}
					objects, err := ResolvePartialHitUser(id, objectsToFetch, num, feedObjectsOffset+i)
					if err != nil {
						return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
					}
					for _, obj := range objects {
						if dweet, ok := obj.(db.DweetModel); ok {
							feedObjectList[objCount] = schema.FormatAsBasicDweetType(&dweet)
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return schema.UserType{}, err
							}
						} else {
							return schema.UserType{}, errors.New("internal server error")
						}
						objCount++
					}
					newIDList := make([]string, 0, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					newIDList = append(newIDList, feedObjectIDs[:hitInfo.FirstIndex]...)
					newIDList = append(newIDList, "<"+fmt.Sprintf("%d", hitInfo.LastIndex-hitInfo.FirstIndex+1)+">")
					newIDList = append(newIDList, feedObjectIDs[hitInfo.LastIndex+1:]...)
					interfaceIDList := make([]interface{}, len(feedObjectIDs)-(hitInfo.LastIndex-hitInfo.FirstIndex+1)+num)
					for listIndex, v := range newIDList {
						interfaceIDList[listIndex] = v
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
					if err != nil {
						return schema.UserType{}, err
					}
					if len(interfaceIDList) > 0 {
						err := cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", interfaceIDList...).Err()
						if err != nil {
							return schema.UserType{}, err
						}
					}
					ExpireUserAt("full", id, expireTime)
				} else {
					feedObject, err := GetCachedDweetBasic(feedObjectID)
					if err != nil {
						return schema.UserType{}, err
					}
					feedObjectList[objCount] = feedObject
					objCount++
				}
			}
		}
		cachedUser.LikedDweets = feedObjectList[:objCount]
	default:
		return schema.UserType{}, errors.New("unknown objectsToFetch")
	}

	followers, err := cacheDB.LRange(common.BaseCtx, keyStem+"followers", 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			followers = []string{}
		} else {
			return schema.UserType{}, err
		}
	}

	followerList := make([]schema.BasicUserType, len(followers))
	for i, followerUsername := range followers {
		follower, err := GetCachedUserBasic(followerUsername)
		if err != nil {
			return schema.UserType{}, err
		}
		followerList[i] = follower
	}

	cachedUser.Followers = followerList

	following, err := cacheDB.LRange(common.BaseCtx, keyStem+"following", 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			following = []string{}
		} else {
			return schema.UserType{}, err
		}
	}

	followingList := make([]schema.BasicUserType, len(following))
	for i, followingUsername := range following {
		followed, err := GetCachedUserBasic(followingUsername)
		if err != nil {
			return schema.UserType{}, err
		}
		followingList[i] = followed
	}

	cachedUser.Following = followingList

	return cachedUser, nil
}
