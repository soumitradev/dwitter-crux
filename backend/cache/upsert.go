// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"errors"
	"fmt"

	"container/list"

	"github.com/go-redis/redis/v8"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/util"
)

func UpsertUser(userID string, obj *db.UserModel, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) error {
	isCached, err := CheckIfCached("user", "full", userID)
	if err != nil {
		return err
	}
	if !isCached {
		err := CacheUser("full", userID, obj, objectsToFetch, feedObjectsToFetch, feedObjectsOffset)
		if err != nil {
			return err
		}
	}

	keyStem := GenerateKey("user", "full", userID, "")
	switch objectsToFetch {
	case "feed":
		if feedObjectsToFetch < 0 {
			merged := util.MergeDweetRedweetList(obj.Dweets(), obj.Redweets())
			iterLen := util.Min(feedObjectsToFetch, len(merged))
			feedObjectList := make([]interface{}, iterLen)
			feedObjectIDList := make([]interface{}, iterLen)
			for i := 0; i < iterLen; i++ {
				feedObjectList[i] = merged[i]
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

			err = cacheDB.Del(common.BaseCtx, keyStem+"feedObjects").Err()
			if err != nil {
				return err
			}
			if len(feedObjectIDList) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", feedObjectIDList...).Err()
				if err != nil {
					return err
				}
			}
		} else {
			cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"feedObjects", 0, -1).Result()
			if err != nil {
				if err == redis.Nil {
					cachedArr = []string{}
				} else {
					return err
				}
			}

			cachedList := list.New()
			for _, id := range cachedArr {
				cachedList.PushBack(id)
			}

			var cursor *list.Element = cachedList.Front()

			originalObjectsToFetch := feedObjectsToFetch
			for feedObjectsOffset != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert your data after a stub of <feedObjectOffset>
						merged := util.MergeDweetRedweetList(obj.Dweets(), obj.Redweets())
						iterLen := len(merged)
						feedObjectList := make([]interface{}, iterLen)
						feedObjectIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							feedObjectList[i] = merged[i]
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

						cursor = cursor.Prev()

						if feedObjectsOffset > 0 {
							cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", feedObjectsOffset)+">", cursor)
						}

						for _, v := range feedObjectIDList {
							cursor = cachedList.InsertAfter(v, cursor)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"feedObjects").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						feedObjectsOffset -= jump
					}
				} else {
					feedObjectsOffset--
				}
				cursor = cursor.Next()
			}

			for feedObjectsToFetch != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert rest of data
						merged := util.MergeDweetRedweetList(obj.Dweets(), obj.Redweets())
						iterLen := len(merged)
						feedObjectList := make([]interface{}, iterLen)
						feedObjectIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							feedObjectList[i] = merged[i]
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

						for revIter := originalObjectsToFetch; revIter > (originalObjectsToFetch - feedObjectsToFetch); revIter-- {
							cursor = cachedList.InsertBefore(feedObjectIDList[revIter], cursor)
						}

						return nil
					} else {
						sliceStart := originalObjectsToFetch - feedObjectsToFetch

						merged := util.MergeDweetRedweetList(obj.Dweets(), obj.Redweets())
						feedObjectList := make([]interface{}, jump)
						feedObjectIDList := make([]interface{}, jump)
						for i := 0; i < jump; i++ {
							feedObjectList[i] = merged[i+feedObjectsOffset+sliceStart]
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

						stubPtr := cursor

						for _, id := range feedObjectIDList {
							cursor = cachedList.InsertAfter(id, cursor)
						}

						cachedList.Remove(stubPtr)
						feedObjectsToFetch -= jump
					}
				} else {
					feedObjectsToFetch--
				}
				cursor = cursor.Next()
			}

			processedArr := make([]interface{}, cachedList.Len())
			cursor = cachedList.Front()
			processedIter := 0
			for cursor != nil {
				processedArr[processedIter] = *cursor
				cursor = cursor.Next()
				processedIter++
			}
			err = cacheDB.Del(common.BaseCtx, keyStem+"feedObjects").Err()
			if err != nil {
				return err
			}
			if len(processedArr) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"feedObjects", processedArr...).Err()
				if err != nil {
					return err
				}
			}
		}
	case "dweet":
		if feedObjectsToFetch < 0 {
			dweets := obj.Dweets()
			iterLen := util.Min(feedObjectsToFetch, len(dweets))
			dweetIDList := make([]interface{}, iterLen)

			for i, dweet := range dweets {
				dweetIDList[i] = dweet.ID
				err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
			if err != nil {
				return err
			}
			if len(dweetIDList) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", dweetIDList...).Err()
				if err != nil {
					return err
				}
			}
		} else {
			cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"dweets", 0, -1).Result()
			if err != nil {
				if err == redis.Nil {
					cachedArr = []string{}
				} else {
					return err
				}
			}

			cachedList := list.New()
			for _, id := range cachedArr {
				cachedList.PushBack(id)
			}

			var cursor *list.Element = cachedList.Front()

			originalObjectsToFetch := feedObjectsToFetch
			for feedObjectsOffset != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert your data after a stub of <feedObjectOffset>
						dweets := obj.Dweets()
						iterLen := len(dweets)
						dweetIDList := make([]interface{}, iterLen)

						for i, dweet := range dweets {
							dweetIDList[i] = dweet.ID
							err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
							if err != nil {
								return err
							}
						}

						stubPtr := cursor
						if iterLen < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						cursor = cursor.Prev()

						if feedObjectsOffset > 0 {
							cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", feedObjectsOffset)+">", cursor)
						}

						for _, v := range dweetIDList {
							cursor = cachedList.InsertAfter(v, cursor)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						feedObjectsOffset -= jump
					}
				} else {
					feedObjectsOffset--
				}
				cursor = cursor.Next()
			}

			for feedObjectsToFetch != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert rest of data
						dweets := obj.Dweets()
						iterLen := originalObjectsToFetch - feedObjectsToFetch
						dweetList := make([]interface{}, iterLen)
						dweetIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							dweetList[i] = dweets[i+iterLen]
						}

						for i, obj := range dweetList {
							if dweet, ok := obj.(db.DweetModel); ok {
								dweetIDList[i] = dweet.ID
								err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
								if err != nil {
									return err
								}
							}
						}

						stubPtr := cursor

						for revIter := originalObjectsToFetch; revIter > (originalObjectsToFetch - feedObjectsToFetch); revIter-- {
							cursor = cachedList.InsertBefore(dweetIDList[revIter], cursor)
						}

						if len(dweetIDList) < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						sliceStart := originalObjectsToFetch - feedObjectsToFetch

						dweets := obj.Dweets()
						dweetList := make([]interface{}, jump)
						dweetIDList := make([]interface{}, jump)
						for i := 0; i < jump; i++ {
							dweetList[i] = dweets[i+sliceStart]
						}

						for i, obj := range dweetList {
							if dweet, ok := obj.(db.DweetModel); ok {
								dweetIDList[i] = dweet.ID
								err := CacheDweet("basic", dweet.ID, &dweet, 0, 0)
								if err != nil {
									return err
								}
							} else {
								return errors.New("internal server error")
							}
						}

						stubPtr := cursor

						for _, id := range dweetIDList {
							cursor = cachedList.InsertAfter(id, cursor)
						}

						cachedList.Remove(stubPtr)
						feedObjectsToFetch -= jump
					}
				} else {
					feedObjectsToFetch--
				}
				cursor = cursor.Next()
			}

			processedArr := make([]interface{}, cachedList.Len())
			cursor = cachedList.Front()
			processedIter := 0
			for cursor != nil {
				processedArr[processedIter] = *cursor
				cursor = cursor.Next()
				processedIter++
			}
			err = cacheDB.Del(common.BaseCtx, keyStem+"dweets").Err()
			if err != nil {
				return err
			}
			if len(processedArr) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"dweets", processedArr...).Err()
				if err != nil {
					return err
				}
			}
		}
	case "redweet":
		if feedObjectsToFetch < 0 {
			redweets := obj.Redweets()
			iterLen := util.Min(feedObjectsToFetch, len(redweets))
			redweetIDList := make([]interface{}, iterLen)

			for i, redweet := range redweets {
				redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
				redweetIDList[i] = redweetID
				err := CacheRedweet("full", redweetID, &redweet)
				if err != nil {
					return err
				}
			}

			err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
			if err != nil {
				return err
			}
			if len(redweetIDList) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", redweetIDList...).Err()
				if err != nil {
					return err
				}
			}
		} else {
			cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweets", 0, -1).Result()
			if err != nil {
				if err == redis.Nil {
					cachedArr = []string{}
				} else {
					return err
				}
			}

			cachedList := list.New()
			for _, id := range cachedArr {
				cachedList.PushBack(id)
			}

			var cursor *list.Element = cachedList.Front()

			originalObjectsToFetch := feedObjectsToFetch
			for feedObjectsOffset != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert your data after a stub of <feedObjectOffset>
						redweets := obj.Redweets()
						iterLen := len(redweets)
						redweetIDList := make([]interface{}, iterLen)

						for i, redweet := range redweets {
							redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
							redweetIDList[i] = redweetID
							err := CacheRedweet("full", redweetID, &redweet)
							if err != nil {
								return err
							}
						}

						stubPtr := cursor
						if iterLen < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						cursor = cursor.Prev()

						if feedObjectsOffset > 0 {
							cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", feedObjectsOffset)+">", cursor)
						}

						for _, v := range redweetIDList {
							cursor = cachedList.InsertAfter(v, cursor)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						feedObjectsOffset -= jump
					}
				} else {
					feedObjectsOffset--
				}
				cursor = cursor.Next()
			}

			for feedObjectsToFetch != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert rest of data
						redweets := obj.Redweets()
						iterLen := originalObjectsToFetch - feedObjectsToFetch
						redweetList := make([]interface{}, iterLen)
						redweetIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							redweetList[i] = redweets[i+iterLen]
						}

						for i, obj := range redweetList {
							if redweet, ok := obj.(db.RedweetModel); ok {
								redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
								redweetIDList[i] = redweetID
								err := CacheRedweet("full", redweetID, &redweet)
								if err != nil {
									return err
								}
							}
						}

						stubPtr := cursor

						for revIter := originalObjectsToFetch; revIter > (originalObjectsToFetch - feedObjectsToFetch); revIter-- {
							cursor = cachedList.InsertBefore(redweetIDList[revIter], cursor)
						}

						if len(redweetIDList) < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						sliceStart := originalObjectsToFetch - feedObjectsToFetch

						redweets := obj.Redweets()
						redweetList := make([]interface{}, jump)
						redweetIDList := make([]interface{}, jump)
						for i := 0; i < jump; i++ {
							redweetList[i] = redweets[i+sliceStart]
						}

						for i, obj := range redweetList {
							if redweet, ok := obj.(db.RedweetModel); ok {
								redweetID := ConstructRedweetID(redweet.AuthorID, redweet.OriginalRedweetID)
								redweetIDList[i] = redweetID
								err := CacheRedweet("basic", redweetID, &redweet)
								if err != nil {
									return err
								}
							} else {
								return errors.New("internal server error")
							}
						}

						stubPtr := cursor

						for _, id := range redweetIDList {
							cursor = cachedList.InsertAfter(id, cursor)
						}

						cachedList.Remove(stubPtr)
						feedObjectsToFetch -= jump
					}
				} else {
					feedObjectsToFetch--
				}
				cursor = cursor.Next()
			}

			processedArr := make([]interface{}, cachedList.Len())
			cursor = cachedList.Front()
			processedIter := 0
			for cursor != nil {
				processedArr[processedIter] = *cursor
				cursor = cursor.Next()
				processedIter++
			}
			err = cacheDB.Del(common.BaseCtx, keyStem+"redweets").Err()
			if err != nil {
				return err
			}
			if len(processedArr) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweets", processedArr...).Err()
				if err != nil {
					return err
				}
			}
		}
	case "redweetedDweet":
		if feedObjectsToFetch < 0 {
			redweetedDweets := obj.RedweetedDweets()
			iterLen := util.Min(feedObjectsToFetch, len(redweetedDweets))
			redweetedDweetIDList := make([]interface{}, iterLen)

			for i, redweetedDweet := range redweetedDweets {
				redweetedDweetIDList[i] = redweetedDweet.ID
				err := CacheDweet("basic", redweetedDweet.ID, &redweetedDweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
			if err != nil {
				return err
			}
			if len(redweetedDweetIDList) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", redweetedDweetIDList...).Err()
				if err != nil {
					return err
				}
			}
		} else {
			cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"redweetedDweets", 0, -1).Result()
			if err != nil {
				if err == redis.Nil {
					cachedArr = []string{}
				} else {
					return err
				}
			}

			cachedList := list.New()
			for _, id := range cachedArr {
				cachedList.PushBack(id)
			}

			var cursor *list.Element = cachedList.Front()

			originalObjectsToFetch := feedObjectsToFetch
			for feedObjectsOffset != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert your data after a stub of <feedObjectOffset>
						redweetedDweets := obj.RedweetedDweets()
						iterLen := len(redweetedDweets)
						redweetedDweetIDList := make([]interface{}, iterLen)

						for i, redweetedDweet := range redweetedDweets {
							redweetedDweetIDList[i] = redweetedDweet.ID
							err := CacheDweet("basic", redweetedDweet.ID, &redweetedDweet, 0, 0)
							if err != nil {
								return err
							}
						}

						stubPtr := cursor
						if iterLen < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						cursor = cursor.Prev()

						if feedObjectsOffset > 0 {
							cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", feedObjectsOffset)+">", cursor)
						}

						for _, v := range redweetedDweetIDList {
							cursor = cachedList.InsertAfter(v, cursor)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						feedObjectsOffset -= jump
					}
				} else {
					feedObjectsOffset--
				}
				cursor = cursor.Next()
			}

			for feedObjectsToFetch != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert rest of data
						redweetedDweets := obj.RedweetedDweets()
						iterLen := originalObjectsToFetch - feedObjectsToFetch
						redweetedDweetList := make([]interface{}, iterLen)
						redweetedDweetIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							redweetedDweetList[i] = redweetedDweets[i+iterLen]
						}

						for i, obj := range redweetedDweetList {
							if redweetedDweet, ok := obj.(db.DweetModel); ok {
								redweetedDweetIDList[i] = redweetedDweet.ID
								err := CacheDweet("basic", redweetedDweet.ID, &redweetedDweet, 0, 0)
								if err != nil {
									return err
								}
							}
						}

						stubPtr := cursor

						for revIter := originalObjectsToFetch; revIter > (originalObjectsToFetch - feedObjectsToFetch); revIter-- {
							cursor = cachedList.InsertBefore(redweetedDweetIDList[revIter], cursor)
						}

						if len(redweetedDweetIDList) < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						sliceStart := originalObjectsToFetch - feedObjectsToFetch

						redweetedDweets := obj.RedweetedDweets()
						redweetedDweetList := make([]interface{}, jump)
						redweetedDweetIDList := make([]interface{}, jump)
						for i := 0; i < jump; i++ {
							redweetedDweetList[i] = redweetedDweets[i+sliceStart]
						}

						for i, obj := range redweetedDweetList {
							if redweetedDweet, ok := obj.(db.DweetModel); ok {
								redweetedDweetIDList[i] = redweetedDweet.ID
								err := CacheDweet("basic", redweetedDweet.ID, &redweetedDweet, 0, 0)
								if err != nil {
									return err
								}
							} else {
								return errors.New("internal server error")
							}
						}

						stubPtr := cursor

						for _, id := range redweetedDweetIDList {
							cursor = cachedList.InsertAfter(id, cursor)
						}

						cachedList.Remove(stubPtr)
						feedObjectsToFetch -= jump
					}
				} else {
					feedObjectsToFetch--
				}
				cursor = cursor.Next()
			}

			processedArr := make([]interface{}, cachedList.Len())
			cursor = cachedList.Front()
			processedIter := 0
			for cursor != nil {
				processedArr[processedIter] = *cursor
				cursor = cursor.Next()
				processedIter++
			}
			err = cacheDB.Del(common.BaseCtx, keyStem+"redweetedDweets").Err()
			if err != nil {
				return err
			}
			if len(processedArr) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"redweetedDweets", processedArr...).Err()
				if err != nil {
					return err
				}
			}
		}
	case "liked":
		if feedObjectsToFetch < 0 {
			likedDweets := obj.LikedDweets()
			iterLen := util.Min(feedObjectsToFetch, len(likedDweets))
			likedDweetIDList := make([]interface{}, iterLen)

			for i, likedDweet := range likedDweets {
				likedDweetIDList[i] = likedDweet.ID
				err := CacheDweet("basic", likedDweet.ID, &likedDweet, 0, 0)
				if err != nil {
					return err
				}
			}

			err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
			if err != nil {
				return err
			}
			if len(likedDweetIDList) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", likedDweetIDList...).Err()
				if err != nil {
					return err
				}
			}
		} else {
			cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"likedDweets", 0, -1).Result()
			if err != nil {
				if err == redis.Nil {
					cachedArr = []string{}
				} else {
					return err
				}
			}

			cachedList := list.New()
			for _, id := range cachedArr {
				cachedList.PushBack(id)
			}

			var cursor *list.Element = cachedList.Front()

			originalObjectsToFetch := feedObjectsToFetch
			for feedObjectsOffset != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert your data after a stub of <feedObjectOffset>
						likedDweets := obj.LikedDweets()
						iterLen := len(likedDweets)
						likedDweetIDList := make([]interface{}, iterLen)

						for i, likedDweet := range likedDweets {
							likedDweetIDList[i] = likedDweet.ID
							err := CacheDweet("basic", likedDweet.ID, &likedDweet, 0, 0)
							if err != nil {
								return err
							}
						}

						stubPtr := cursor
						if iterLen < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						cursor = cursor.Prev()

						if feedObjectsOffset > 0 {
							cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", feedObjectsOffset)+">", cursor)
						}

						for _, v := range likedDweetIDList {
							cursor = cachedList.InsertAfter(v, cursor)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						feedObjectsOffset -= jump
					}
				} else {
					feedObjectsOffset--
				}
				cursor = cursor.Next()
			}

			for feedObjectsToFetch != 0 && cursor.Next() != nil {
				if IsStub(cursor.Value.(string)) {
					jump, err := ParseStub(cursor.Value.(string))
					if err != nil {
						return err
					}
					if jump == -1 {
						// Insert rest of data
						likedDweets := obj.LikedDweets()
						iterLen := originalObjectsToFetch - feedObjectsToFetch
						likedDweetList := make([]interface{}, iterLen)
						likedDweetIDList := make([]interface{}, iterLen)
						for i := 0; i < iterLen; i++ {
							likedDweetList[i] = likedDweets[i+iterLen]
						}

						for i, obj := range likedDweetList {
							if likedDweet, ok := obj.(db.DweetModel); ok {
								likedDweetIDList[i] = likedDweet.ID
								err := CacheDweet("basic", likedDweet.ID, &likedDweet, 0, 0)
								if err != nil {
									return err
								}
							}
						}

						stubPtr := cursor

						for revIter := originalObjectsToFetch; revIter > (originalObjectsToFetch - feedObjectsToFetch); revIter-- {
							cursor = cachedList.InsertBefore(likedDweetIDList[revIter], cursor)
						}

						if len(likedDweetIDList) < originalObjectsToFetch {
							cachedList.Remove(stubPtr)
						}

						processedArr := make([]interface{}, cachedList.Len())
						cursor = cachedList.Front()
						processedIter := 0
						for cursor != nil {
							processedArr[processedIter] = *cursor
							cursor = cursor.Next()
							processedIter++
						}
						err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
						if err != nil {
							return err
						}
						if len(processedArr) > 0 {
							err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", processedArr...).Err()
							if err != nil {
								return err
							}
						}

						return nil
					} else {
						sliceStart := originalObjectsToFetch - feedObjectsToFetch

						likedDweets := obj.LikedDweets()
						likedDweetList := make([]interface{}, jump)
						likedDweetIDList := make([]interface{}, jump)
						for i := 0; i < jump; i++ {
							likedDweetList[i] = likedDweets[i+sliceStart]
						}

						for i, obj := range likedDweetList {
							if likedDweet, ok := obj.(db.DweetModel); ok {
								likedDweetIDList[i] = likedDweet.ID
								err := CacheDweet("basic", likedDweet.ID, &likedDweet, 0, 0)
								if err != nil {
									return err
								}
							} else {
								return errors.New("internal server error")
							}
						}

						stubPtr := cursor

						for _, id := range likedDweetIDList {
							cursor = cachedList.InsertAfter(id, cursor)
						}

						cachedList.Remove(stubPtr)
						feedObjectsToFetch -= jump
					}
				} else {
					feedObjectsToFetch--
				}
				cursor = cursor.Next()
			}

			processedArr := make([]interface{}, cachedList.Len())
			cursor = cachedList.Front()
			processedIter := 0
			for cursor != nil {
				processedArr[processedIter] = *cursor
				cursor = cursor.Next()
				processedIter++
			}
			err = cacheDB.Del(common.BaseCtx, keyStem+"likedDweets").Err()
			if err != nil {
				return err
			}
			if len(processedArr) > 0 {
				err = cacheDB.LPush(common.BaseCtx, keyStem+"likedDweets", processedArr...).Err()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func UpsertDweet(dweetID string, obj *db.DweetModel, repliesToFetch int, replyOffset int) error {
	isCached, err := CheckIfCached("dweet", "full", dweetID)
	if err != nil {
		return err
	}
	if !isCached {
		err := CacheDweet("full", dweetID, obj, repliesToFetch, replyOffset)
		if err != nil {
			return err
		}
	}

	keyStem := GenerateKey("dweet", "full", dweetID, "")
	if repliesToFetch < 0 {
		replies := obj.ReplyDweets()
		iterLen := util.Min(repliesToFetch, len(replies))
		replyIDList := make([]interface{}, iterLen)

		for i, reply := range replies {
			replyIDList[i] = reply.ID
			err := CacheDweet("basic", reply.ID, &reply, 0, 0)
			if err != nil {
				return err
			}
		}

		err = cacheDB.Del(common.BaseCtx, keyStem+"replyDweets").Err()
		if err != nil {
			return err
		}
		if len(replyIDList) > 0 {
			err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", replyIDList...).Err()
			if err != nil {
				return err
			}
		}
	} else {
		cachedArr, err := cacheDB.LRange(common.BaseCtx, keyStem+"replyDweets", 0, -1).Result()
		if err != nil {
			if err == redis.Nil {
				cachedArr = []string{}
			} else {
				return err
			}
		}

		cachedList := list.New()
		for _, id := range cachedArr {
			cachedList.PushBack(id)
		}

		var cursor *list.Element = cachedList.Front()

		originalRepliesToFetch := repliesToFetch
		for repliesToFetch != 0 && cursor.Next() != nil {
			if IsStub(cursor.Value.(string)) {
				jump, err := ParseStub(cursor.Value.(string))
				if err != nil {
					return err
				}
				if jump == -1 {
					// Insert your data after a stub of <feedObjectOffset>
					replies := obj.ReplyDweets()
					iterLen := len(replies)
					replyIDList := make([]interface{}, iterLen)

					for i, reply := range replies {
						replyIDList[i] = reply.ID
						err := CacheDweet("basic", reply.ID, &reply, 0, 0)
						if err != nil {
							return err
						}
					}

					stubPtr := cursor
					if iterLen < originalRepliesToFetch {
						cachedList.Remove(stubPtr)
					}

					cursor = cursor.Prev()

					if replyOffset > 0 {
						cursor = cachedList.InsertAfter("<"+fmt.Sprintf("%d", replyOffset)+">", cursor)
					}

					for _, v := range replyIDList {
						cursor = cachedList.InsertAfter(v, cursor)
					}

					processedArr := make([]interface{}, cachedList.Len())
					cursor = cachedList.Front()
					processedIter := 0
					for cursor != nil {
						processedArr[processedIter] = *cursor
						cursor = cursor.Next()
						processedIter++
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"replyDweets").Err()
					if err != nil {
						return err
					}
					if len(processedArr) > 0 {
						err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", processedArr...).Err()
						if err != nil {
							return err
						}
					}

					return nil
				} else {
					replyOffset -= jump
				}
			} else {
				replyOffset--
			}
			cursor = cursor.Next()
		}

		for repliesToFetch != 0 && cursor.Next() != nil {
			if IsStub(cursor.Value.(string)) {
				jump, err := ParseStub(cursor.Value.(string))
				if err != nil {
					return err
				}
				if jump == -1 {
					// Insert rest of data
					replies := obj.ReplyDweets()
					iterLen := originalRepliesToFetch - repliesToFetch
					replyList := make([]interface{}, iterLen)
					replyIDList := make([]interface{}, iterLen)
					for i := 0; i < iterLen; i++ {
						replyList[i] = replies[i+iterLen]
					}

					for i, obj := range replyList {
						if reply, ok := obj.(db.DweetModel); ok {
							replyIDList[i] = reply.ID
							err := CacheDweet("basic", reply.ID, &reply, 0, 0)
							if err != nil {
								return err
							}
						}
					}

					stubPtr := cursor

					for revIter := originalRepliesToFetch; revIter > (originalRepliesToFetch - repliesToFetch); revIter-- {
						cursor = cachedList.InsertBefore(replyIDList[revIter], cursor)
					}

					if len(replyIDList) < originalRepliesToFetch {
						cachedList.Remove(stubPtr)
					}

					processedArr := make([]interface{}, cachedList.Len())
					cursor = cachedList.Front()
					processedIter := 0
					for cursor != nil {
						processedArr[processedIter] = *cursor
						cursor = cursor.Next()
						processedIter++
					}
					err = cacheDB.Del(common.BaseCtx, keyStem+"replyDweets").Err()
					if err != nil {
						return err
					}
					if len(processedArr) > 0 {
						err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", processedArr...).Err()
						if err != nil {
							return err
						}
					}

					return nil
				} else {
					sliceStart := originalRepliesToFetch - replyOffset

					replies := obj.ReplyDweets()
					replyList := make([]interface{}, jump)
					replyIDList := make([]interface{}, jump)
					for i := 0; i < jump; i++ {
						replyList[i] = replies[i+sliceStart]
					}

					for i, obj := range replyList {
						if reply, ok := obj.(db.DweetModel); ok {
							replyIDList[i] = reply.ID
							err := CacheDweet("basic", reply.ID, &reply, 0, 0)
							if err != nil {
								return err
							}
						} else {
							return errors.New("internal server error")
						}
					}

					stubPtr := cursor

					for _, id := range replyIDList {
						cursor = cachedList.InsertAfter(id, cursor)
					}

					cachedList.Remove(stubPtr)
					repliesToFetch -= jump
				}
			} else {
				repliesToFetch--
			}
			cursor = cursor.Next()
		}

		processedArr := make([]interface{}, cachedList.Len())
		cursor = cachedList.Front()
		processedIter := 0
		for cursor != nil {
			processedArr[processedIter] = *cursor
			cursor = cursor.Next()
			processedIter++
		}
		err = cacheDB.Del(common.BaseCtx, keyStem+"replyDweets").Err()
		if err != nil {
			return err
		}
		if len(processedArr) > 0 {
			err = cacheDB.LPush(common.BaseCtx, keyStem+"replyDweets", processedArr...).Err()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
