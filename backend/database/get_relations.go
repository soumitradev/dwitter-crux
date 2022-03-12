package database

import (
	"fmt"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Get users that follow user
func GetFollowers(username string, numberToFetch int, numOffset int, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) ([]schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(numOffset, "gte=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(objectsToFetch, "required,alpha,gt=0,oneof=feed dweet redweet redweetedDweet")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(feedObjectsOffset, "gte=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	var user *db.UserModel
	var feedObjectList [][]interface{}

	// Check params and return data accordingly
	if numberToFetch < 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	} else {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Followers.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					).Take(numberToFetch).Skip(numOffset),
					db.User.Following.Fetch().OrderBy(
						db.User.FollowerCount.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				users := user.Followers()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	}
	if err == db.ErrNotFound {
		return []schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return []schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Add common followers and format
	var followers []schema.UserType

	knownUsers := user.Following()
	knownUsers = append(knownUsers, *user)

	for followerIndex, follower := range user.Followers() {
		followerFollowers := follower.Followers()
		followerFollowing := follower.Followers()

		mutualFollowers := util.HashIntersectUsers(followerFollowers, knownUsers)
		mutualFollowing := util.HashIntersectUsers(followerFollowing, knownUsers)

		formatted, err := schema.FormatAsUserType(&follower, mutualFollowing, mutualFollowers, objectsToFetch, feedObjectList[followerIndex], false)
		if err != nil {
			return []schema.UserType{}, err
		}
		followers = append(followers, formatted)
	}
	return followers, err
}

// Get users that user follows
func GetFollowing(username string, numberToFetch int, numOffset int, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) ([]schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(numOffset, "gte=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(objectsToFetch, "required,alpha,gt=0,oneof=feed dweet redweet redweetedDweet")
	if err != nil {
		return []schema.UserType{}, err
	}

	err = common.Validate.Var(feedObjectsOffset, "gte=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	var user *db.UserModel
	var feedObjectList [][]interface{}

	// Check params and return data accordingly
	if numberToFetch < 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	} else {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(follower.Dweets(), follower.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Dweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range follower.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.Following.Fetch().With(
						db.User.RedweetedDweets.Fetch().With(
							db.Dweet.Author.Fetch(),
						).OrderBy(
							db.Dweet.PostedAt.Order(db.DESC),
						).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
						db.User.Followers.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
						db.User.Following.Fetch().OrderBy(
							db.User.FollowerCount.Order(db.DESC),
						),
					).Take(numberToFetch).Skip(numOffset),
				).Exec(common.BaseCtx)

				users := user.Following()
				for index, follower := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range follower.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	}
	if err == db.ErrNotFound {
		return []schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return []schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Add common followers and return
	userFullFollowing, err := common.Client.User.FindUnique(
		db.User.Username.Equals(username),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return []schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return []schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	knownUsers := userFullFollowing.Following()
	knownUsers = append(knownUsers, *userFullFollowing)

	var result []schema.UserType

	for followedIndex, followed := range user.Following() {
		followerFollowers := followed.Followers()
		followerFollowing := followed.Following()

		mutualFollowers := util.HashIntersectUsers(followerFollowers, knownUsers)
		mutualFollowing := util.HashIntersectUsers(followerFollowing, knownUsers)

		formatted, err := schema.FormatAsUserType(&followed, mutualFollowers, mutualFollowing, objectsToFetch, feedObjectList[followedIndex], false)
		if err != nil {
			return []schema.UserType{}, err
		}
		result = append(result, formatted)
	}

	return result, err
}
