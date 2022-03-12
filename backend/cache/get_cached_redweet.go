// Package cache provides useful functions to use the Redis LRU cache
package cache

import "github.com/soumitradev/Dwitter/backend/schema"

func GetCachedRedweet(id string) (schema.RedweetType, error) {
	return schema.RedweetType{}, nil
}
