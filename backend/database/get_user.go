package database

import (
	"errors"
	"fmt"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Get user when not authenticated
func GetUserUnauth(username string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(objectsToFetch, "required,alpha,gt=0,oneof=feed dweet redweet redweetedDweet")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(feedObjectsOffset, "gte=0")
	if err != nil {
		return schema.UserType{}, err
	}

	var user *db.UserModel
	var feedObjectList []interface{}

	if feedObjectsToFetch < 0 {
		switch objectsToFetch {
		case "feed":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			feedObjectList = append(feedObjectList, merged...)
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
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
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
				feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
			}
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
			}
		default:
			break
		}
	}

	// TODO: Some kind of weird bug where you get nil objects in list when requesting dweets and redweets

	// Send back the user requested, along with mutuals in the followers field
	nuser, err := schema.FormatAsUserType(user, []db.UserModel{}, []db.UserModel{}, objectsToFetch, feedObjectList, false)
	return nuser, err
}

// Get user when authenticated
func GetUser(username string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int, viewerUsername string) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(objectsToFetch, "required,alpha,gt=0,oneof=feed dweet redweet redweetedDweet liked")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(feedObjectsOffset, "gte=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(viewerUsername, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	var user *db.UserModel
	var alsoFollowedBy []db.UserModel
	var alsoFollowing []db.UserModel
	var feedObjectList []interface{}

	// Get your own following-list
	viewUser, err := common.Client.User.FindUnique(
		db.User.Username.Equals(viewerUsername),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	if feedObjectsToFetch < 0 {
		switch objectsToFetch {
		case "feed":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			feedObjectList = append(feedObjectList, merged...)
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
			}
		case "liked":
			if viewerUsername == username {
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.LikedDweets.Fetch().With(
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
				).Exec(common.BaseCtx)
				if err == db.ErrNotFound {
					return schema.UserType{}, fmt.Errorf("user not found: %v", err)
				}
				if err != nil {
					return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
				}

				likes := user.LikedDweets()
				for i := 0; i < len(likes); i++ {
					feedObjectList = append(feedObjectList, likes[i])
				}
			} else {
				return schema.UserType{}, errors.New("unauthorized")
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
				db.User.Dweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.PostedAt.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Take(feedObjectsToFetch+feedObjectsOffset),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
				feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
			}
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
			).With(
				db.User.Redweets.Fetch().With(
					db.Redweet.Author.Fetch(),
					db.Redweet.RedweetOf.Fetch().With(
						db.Dweet.Author.Fetch(),
					),
				).OrderBy(
					db.Redweet.RedweetTime.Order(db.DESC),
				).Skip(feedObjectsOffset).Take(feedObjectsToFetch),
				db.User.Followers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(username),
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
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.UserType{}, fmt.Errorf("user not found: %v", err)
			}
			if err != nil {
				return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
			}

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
			}
		case "liked":
			if viewerUsername == username {
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(username),
				).With(
					db.User.LikedDweets.Fetch().With(
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
				).Exec(common.BaseCtx)
				if err == db.ErrNotFound {
					return schema.UserType{}, fmt.Errorf("user not found: %v", err)
				}
				if err != nil {
					return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
				}

				likes := user.LikedDweets()
				for i := 0; i < len(likes); i++ {
					feedObjectList = append(feedObjectList, likes[i])
				}
			} else {
				return schema.UserType{}, errors.New("unauthorized")
			}
		default:
			break
		}
	}
	var showEmail bool

	if viewerUsername == username {
		alsoFollowedBy = user.Followers()
		alsoFollowing = user.Following()
		showEmail = true
	} else {
		usersFollowed := append(viewUser.Following(), *viewUser)

		// Get mutuals
		followers := user.Followers()
		following := user.Following()
		showEmail = false

		alsoFollowedBy = util.HashIntersectUsers(followers, usersFollowed)
		alsoFollowing = util.HashIntersectUsers(following, usersFollowed)
	}

	// Send back the user requested, along with mutuals in the followers field
	nuser, err := schema.FormatAsUserType(user, alsoFollowedBy, alsoFollowing, objectsToFetch, feedObjectList, showEmail)
	return nuser, err
}
