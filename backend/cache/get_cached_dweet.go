// Package cache provides useful functions to use the Redis LRU cache
package cache

import "github.com/soumitradev/Dwitter/backend/schema"

func GetCachedDweetFull(id string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	return schema.DweetType{}, nil
}

func GetCachedDweetBasic(id string) (schema.BasicDweetType, error) {
	return schema.BasicDweetType{}, nil
}
