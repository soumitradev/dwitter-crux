package database

import (
	"fmt"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Search users when not authenticated
func SearchUsersUnauth(query string, numberToFetch int, numOffset int, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) ([]schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(query, "required,gt=0")
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

	var users []db.UserModel
	var feedObjectList [][]interface{}

	if numberToFetch < 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Dweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Redweets.Fetch().With(
						db.Redweet.Author.Fetch(),
						db.Redweet.RedweetOf.Fetch(),
					).OrderBy(
						db.Redweet.RedweetTime.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.RedweetedDweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Dweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Redweets.Fetch().With(
						db.Redweet.Author.Fetch(),
						db.Redweet.RedweetOf.Fetch(),
					).OrderBy(
						db.Redweet.RedweetTime.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.RedweetedDweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
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
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Dweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Redweets.Fetch().With(
						db.Redweet.Author.Fetch(),
						db.Redweet.RedweetOf.Fetch(),
					).OrderBy(
						db.Redweet.RedweetTime.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.RedweetedDweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Dweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.Redweets.Fetch().With(
						db.Redweet.Author.Fetch(),
						db.Redweet.RedweetOf.Fetch(),
					).OrderBy(
						db.Redweet.RedweetTime.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
					db.User.RedweetedDweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				).OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	}

	var formatted []schema.UserType

	for userIndex, user := range users {
		nuser, err := schema.FormatAsUserType(&user, []db.UserModel{}, []db.UserModel{}, objectsToFetch, feedObjectList[userIndex], false)
		if err != nil {
			return []schema.UserType{}, nil
		}
		formatted = append(formatted, nuser)
	}

	return formatted, err
}

// Search users when authenticated
func SearchUsers(query string, numberToFetch int, numOffset int, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int, viewerUsername string) ([]schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(query, "required,gt=0")
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

	err = common.Validate.Var(viewerUsername, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return []schema.UserType{}, err
	}

	var users []db.UserModel
	var alsoFollowedBy [][]db.UserModel
	var alsoFollowing [][]db.UserModel
	var feedObjectList [][]interface{}

	// Get your own following-list
	viewUser, err := common.Client.User.FindUnique(
		db.User.Username.Equals(viewerUsername),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
		db.User.Followers.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return []schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return []schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	if numberToFetch < 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
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
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					feedObjectList[index] = append(feedObjectList[index], merged...)
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.RedweetedDweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		} else {
			switch objectsToFetch {
			case "feed":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())
					for _, obj := range merged {
						feedObjectList[index] = append(feedObjectList[index+feedObjectsOffset], obj)
					}
				}
			case "dweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			case "redweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, redweet := range user.Redweets() {
						feedObjectList[index] = append(feedObjectList[index], redweet)
					}
				}
			case "redweetedDweet":
				users, err = common.Client.User.FindMany(
					db.User.Username.Contains(query),
				).With(
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
				).Take(numberToFetch).Skip(numOffset).Exec(common.BaseCtx)

				for index, user := range users {
					feedObjectList = append(feedObjectList, []interface{}{})
					for _, dweet := range user.Dweets() {
						feedObjectList[index] = append(feedObjectList[index], dweet)
					}
				}
			default:
				break
			}
		}
	}

	var formatted []schema.UserType
	var showEmail bool

	for userIndex, user := range users {
		if viewerUsername == user.Username {
			alsoFollowedBy[userIndex] = append(alsoFollowedBy[userIndex], user.Followers()...)
			alsoFollowing[userIndex] = append(alsoFollowing[userIndex], user.Following()...)
			showEmail = true
		} else {
			usersFollowed := append(viewUser.Following(), *viewUser)

			// Get mutuals
			followers := user.Followers()
			following := user.Following()
			showEmail = false

			alsoFollowedBy[userIndex] = append(alsoFollowedBy[userIndex], util.HashIntersectUsers(followers, usersFollowed)...)
			alsoFollowing[userIndex] = append(alsoFollowedBy[userIndex], util.HashIntersectUsers(following, usersFollowed)...)
		}
		nuser, err := schema.FormatAsUserType(&user, alsoFollowedBy[userIndex], alsoFollowing[userIndex], objectsToFetch, feedObjectList[userIndex], showEmail)
		if err != nil {
			return []schema.UserType{}, nil
		}
		formatted = append(formatted, nuser)
	}

	return formatted, err
}
