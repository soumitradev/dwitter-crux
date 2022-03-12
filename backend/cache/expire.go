// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"errors"
	"time"

	"github.com/soumitradev/Dwitter/backend/common"
)

func ExpireRedweetAt(id string, expireTime time.Time) error {
	keyStem := GenerateKey("redweet", "full", id, "")
	redweetProps := []string{
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "redweetOf",
		keyStem + "originalRedweetID",
		keyStem + "redweetTime",
	}

	for _, hash := range redweetProps {
		err := cacheDB.PExpireAt(common.BaseCtx, hash, expireTime).Err()
		if err != nil {
			return err
		}
	}

	authorID, err := cacheDB.Get(common.BaseCtx, keyStem+"author").Result()
	if err != nil {
		return err
	}
	err = ExpireUserAt("basic", authorID, expireTime)
	if err != nil {
		return err
	}

	originalRedweetID, err := cacheDB.Get(common.BaseCtx, keyStem+"originalRedweetID").Result()
	if err != nil {
		return err
	}
	if originalRedweetID != "" {
		err = ExpireDweetAt("basic", originalRedweetID, expireTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func ExpireDweetAt(detailLevel string, id string, expireTime time.Time) error {
	keyStem := GenerateKey("dweet", detailLevel, id, "")
	dweetProps := []string{
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

	for _, hash := range dweetProps {
		err := cacheDB.PExpireAt(common.BaseCtx, hash, expireTime).Err()
		if err != nil {
			return err
		}
	}

	authorID, err := cacheDB.Get(common.BaseCtx, keyStem+"author").Result()
	if err != nil {
		return err
	}
	err = ExpireUserAt("basic", authorID, expireTime)
	if err != nil {
		return err
	}

	if detailLevel == "basic" {
		return nil
	} else if detailLevel == "full" {
		userListsToExpire := []string{"likeUsers", "redweetUsers"}
		feedObjectListsToExpire := []string{"replyDweets"}
		for _, userList := range userListsToExpire {
			err := cacheDB.PExpireAt(common.BaseCtx, keyStem+userList, expireTime).Err()
			if err != nil {
				return err
			}

			userList, err := cacheDB.LRange(common.BaseCtx, keyStem+userList, 0, -1).Result()
			if err != nil {
				return err
			}

			for _, v := range userList {
				if !IsStub(v) {
					ExpireUserAt("basic", v, expireTime)
				}
			}
		}

		for _, objectList := range feedObjectListsToExpire {
			err := cacheDB.PExpireAt(common.BaseCtx, keyStem+objectList, expireTime).Err()
			if err != nil {
				return err
			}

			objList, err := cacheDB.LRange(common.BaseCtx, keyStem+objectList, 0, -1).Result()
			if err != nil {
				return err
			}

			for _, v := range objList {
				if !IsStub(v) {
					if isRedweet(v) {
						err = ExpireRedweetAt(v, expireTime)
					} else {
						err = ExpireDweetAt("basic", v, expireTime)
					}
					if err != nil {
						return err
					}
				}
			}
		}

		originalReplyID, err := cacheDB.Get(common.BaseCtx, keyStem+"originalReplyID").Result()
		if err != nil {
			return err
		}
		if originalReplyID != "" {
			err = ExpireDweetAt("basic", originalReplyID, expireTime)
			if err != nil {
				return err
			}
		}

		return nil
	} else {
		return errors.New("unknown detailLevel")
	}
}

func ExpireUserAt(detailLevel string, id string, expireTime time.Time) error {
	keyStem := GenerateKey("user", detailLevel, id, "")
	userProps := []string{
		keyStem + "username",
		keyStem + "name",
		keyStem + "email",
		keyStem + "bio",
		keyStem + "pfpURL",
		keyStem + "followerCount",
		keyStem + "followingCount",
		keyStem + "createdAt",
	}

	for _, hash := range userProps {
		err := cacheDB.PExpireAt(common.BaseCtx, hash, expireTime).Err()
		if err != nil {
			return err
		}
	}

	if detailLevel == "basic" {
		return nil
	} else if detailLevel == "full" {
		userListsToExpire := []string{"followers", "following"}
		feedObjectListsToExpire := []string{"feedObjects", "dweets", "redweets", "redweetedDweets", "likedDweets"}
		for _, userList := range userListsToExpire {
			err := cacheDB.PExpireAt(common.BaseCtx, keyStem+userList, expireTime).Err()
			if err != nil {
				return err
			}

			userList, err := cacheDB.LRange(common.BaseCtx, keyStem+userList, 0, -1).Result()
			if err != nil {
				return err
			}

			for _, v := range userList {
				if !IsStub(v) {
					ExpireUserAt("basic", v, expireTime)
				}
			}
		}

		for _, objectList := range feedObjectListsToExpire {
			err := cacheDB.PExpireAt(common.BaseCtx, keyStem+objectList, expireTime).Err()
			if err != nil {
				return err
			}

			objList, err := cacheDB.LRange(common.BaseCtx, keyStem+objectList, 0, -1).Result()
			if err != nil {
				return err
			}

			for _, v := range objList {
				if !IsStub(v) {
					if isRedweet(v) {
						err = ExpireRedweetAt(v, expireTime)
					} else {
						err = ExpireDweetAt("basic", v, expireTime)
					}
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	} else {
		return errors.New("unknown detailLevel")
	}
}
