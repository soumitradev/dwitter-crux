// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"time"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

func GetCachedRedweet(id string) (schema.RedweetType, error) {
	keyStem := GenerateKey("dweet", "full", id, "")
	keyList := []string{
		keyStem + "author",
		keyStem + "authorID",
		keyStem + "redweetOf",
		keyStem + "originalRedweetID",
		keyStem + "redweetTime",
	}
	valList, err := cacheDB.MGet(common.BaseCtx, keyList...).Result()
	if err != nil {
		return schema.RedweetType{}, err
	}

	author, err := GetCachedUserBasic(valList[0].(string))
	if err != nil {
		return schema.RedweetType{}, err
	}

	redweetOf, err := GetCachedDweetBasic(valList[2].(string))
	if err != nil {
		return schema.RedweetType{}, err
	}

	redweetTime, err := time.Parse(util.TimeUTCFormat, valList[4].(string))
	if err != nil {
		return schema.RedweetType{}, err
	}
	cachedRedweet := schema.RedweetType{
		Author:            author,
		AuthorID:          valList[1].(string),
		RedweetOf:         redweetOf,
		OriginalRedweetID: valList[3].(string),
		RedweetTime:       redweetTime,
	}
	return cachedRedweet, nil
}
