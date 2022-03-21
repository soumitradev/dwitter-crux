// Package util provides useful utility functions when dealing with the API, like intersections, differences of slices,
// cryptographically secure random integers, merging two slices in a specified order, etc.
package util

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	math_rand "math/rand"
	"reflect"

	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
)

const AlphanumBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const LoweralphaBytes = "abcdefghijklmnopqrstuvwxyz"
const TimeUTCFormat = "Mon, 2 Jan 2006 15:04:05 MST"

// More secure random seeding than usual: https://stackoverflow.com/a/54491783
func init() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		panic("cannot seed math/rand package with cryptographically secure random number generator")
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}

// Make a random string of length n
func GenID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = AlphanumBytes[math_rand.Intn(len(AlphanumBytes))]
	}
	return string(b)
}

// Make a random string of length 5n - 1 with hyphens every 4 letters
// e.g. GenToken(5) will return a string of format aaaa-bbbb-cccc-dddd-eeee
func GenToken(n int) string {
	b := make([]byte, 5*n-1)
	for i := range b {
		if (i+1)%5 == 0 {
			b[i] = '-'
		} else {
			b[i] = LoweralphaBytes[math_rand.Intn(len(LoweralphaBytes))]
		}
	}
	return string(b)
}

// Hash function to find intersection of two slices
// with modifications from: https://github.com/juliangruber/go-intersect/blob/2e99d8c0a75f6975a52f7efeb81926a19b221214/intersect.go#L42-L62
// Hash has complexity: O(n * x) where x is a factor of hash function efficiency (between 1 and 2)
func HashIntersectUsers(a []db.UserModel, b []db.UserModel) []db.UserModel {
	set := make([]db.UserModel, 0)
	hash := make(map[string]bool)
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		elt := el.(db.UserModel).Username
		hash[elt] = true
	}

	for i := 0; i < bv.Len(); i++ {
		el := bv.Index(i).Interface()
		elt := el.(db.UserModel).Username
		if _, found := hash[elt]; found {
			set = append(set, el.(db.UserModel))
		}
	}

	return set
}

// Hash function to find intersection of two slices
// with modifications from: https://github.com/juliangruber/go-intersect/blob/2e99d8c0a75f6975a52f7efeb81926a19b221214/intersect.go#L42-L62
// Hash has complexity: O(n * x) where x is a factor of hash function efficiency (between 1 and 2)
func HashIntersectUserSchema(a []schema.BasicUserType, b []schema.BasicUserType) []schema.BasicUserType {
	set := make([]schema.BasicUserType, 0)
	hash := make(map[string]bool)
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		elt := el.(schema.BasicUserType).Username
		hash[elt] = true
	}

	for i := 0; i < bv.Len(); i++ {
		el := bv.Index(i).Interface()
		elt := el.(schema.BasicUserType).Username
		if _, found := hash[elt]; found {
			set = append(set, el.(schema.BasicUserType))
		}
	}

	return set
}

// Hash difference function from: https://stackoverflow.com/a/45428032
func HashDifference(a []string, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// Modified Merge sort for merging dweets and redweets
// adapted from: https://www.golangprograms.com/golang-program-for-implementation-of-mergesort.html
func MergeDweetRedweetList(dweets []db.DweetModel, redweets []db.RedweetModel) []interface{} {
	result := make([]interface{}, len(dweets)+len(redweets))

	i := 0
	for len(dweets) > 0 && len(redweets) > 0 {
		if dweets[0].PostedAt.Unix() > redweets[0].RedweetTime.Unix() {
			result[i] = dweets[0]
			dweets = dweets[1:]
		} else {
			result[i] = redweets[0]
			redweets = redweets[1:]
		}
		i++
	}

	for j := 0; j < len(dweets); j++ {
		result[i] = dweets[j]
		i++
	}
	for j := 0; j < len(redweets); j++ {
		result[i] = redweets[j]
		i++
	}

	return result
}

// Modified Merge sort for merging dweet slices
// adapted from: https://www.golangprograms.com/golang-program-for-implementation-of-mergesort.html
func MergeDweetLists(dweetsA []db.DweetModel, dweetsB []db.DweetModel) []db.DweetModel {
	result := make([]db.DweetModel, len(dweetsA)+len(dweetsB))

	i := 0
	for len(dweetsA) > 0 && len(dweetsB) > 0 {
		if dweetsA[0].PostedAt.Unix() > dweetsB[0].PostedAt.Unix() {
			result[i] = dweetsA[0]
			dweetsA = dweetsA[1:]
		} else {
			result[i] = dweetsB[0]
			dweetsB = dweetsB[1:]
		}
		i++
	}

	for j := 0; j < len(dweetsA); j++ {
		result[i] = dweetsA[j]
		i++
	}
	for j := 0; j < len(dweetsB); j++ {
		result[i] = dweetsB[j]
		i++
	}

	return result
}

// Modified Merge sort for merging redweet slices
// adapted from: https://www.golangprograms.com/golang-program-for-implementation-of-mergesort.html
func MergeRedweetLists(redweetsA []db.RedweetModel, redweetsB []db.RedweetModel) []db.RedweetModel {
	result := make([]db.RedweetModel, len(redweetsA)+len(redweetsB))

	i := 0
	for len(redweetsA) > 0 && len(redweetsB) > 0 {
		if redweetsA[0].RedweetTime.Unix() > redweetsB[0].RedweetTime.Unix() {
			result[i] = redweetsA[0]
			redweetsA = redweetsA[1:]
		} else {
			result[i] = redweetsB[0]
			redweetsB = redweetsB[1:]
		}
		i++
	}

	for j := 0; j < len(redweetsA); j++ {
		result[i] = redweetsA[j]
		i++
	}
	for j := 0; j < len(redweetsB); j++ {
		result[i] = redweetsB[j]
		i++
	}

	return result
}

// Minimum function for integers, because generics aren't a thing, and casting to float is slow
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
