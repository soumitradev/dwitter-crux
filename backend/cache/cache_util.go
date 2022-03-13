// Package cache provides useful functions to use the Redis LRU cache
package cache

import (
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

type PartialHitType struct {
	isPartial  bool
	FirstIndex int
	LastIndex  int
}

var cacheDB *redis.Client

var uncachedStub string = "<?>"

func InitCache() {
	cacheDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6421",
		Password: os.Getenv("REDIS_6421_PASS"),
		DB:       0,
	})
}

func GenerateKey(objType, detailLevel, id, field string) string {
	return objType + ":" + detailLevel + ":" + id + ":" + field
}

func ConstructRedweetID(authorID string, originalRedweetID string) string {
	return "Redweet(" + authorID + ", " + originalRedweetID + ")"
}
func ParseRedweetID(redweetID string) (authorID string, originalRedweetID string) {
	noShell := redweetID[8 : len(redweetID)-1]
	strings := strings.Split(noShell, ", ")
	return strings[0], strings[1]
}

func isRedweet(id string) bool {
	if len(id) < 8 {
		return false
	}
	return id[:8] == "Redweet("
}

func ParseStub(stub string) (int, error) {
	if stub == uncachedStub {
		return -1, nil
	} else {
		return strconv.Atoi(stub[1 : len(stub)-1])
	}
}

func IsStub(stub string) bool {
	if stub[1] == '<' && stub[len(stub)-1] == '>' {
		return true
	} else {
		return false
	}
}
