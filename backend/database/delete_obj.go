package database

import (
	"errors"
	"fmt"

	"github.com/soumitradev/Dwitter/backend/cache"
	"github.com/soumitradev/Dwitter/backend/cdn"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Delete a dweet
func DeleteDweet(postID string, username string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(postID, "required,alphanum,len=10")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(replyOffset, "gte=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	var deleted *db.DweetModel

	// Check params and return data accordingly
	if repliesToFetch < 0 {
		deleted, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(postID),
		).With(
			db.Dweet.Author.Fetch().With(
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			),
		).Exec(common.BaseCtx)
	} else {
		deleted, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(postID),
		).With(
			db.Dweet.Author.Fetch().With(
				db.User.Following.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			).Take(repliesToFetch).Skip(replyOffset),
		).Exec(common.BaseCtx)
	}
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	err = cache.DeleteDweetCacheUpdate(deleted.ID)
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Check if authorized to delete dweet
	if deleted.Author().Username == username {
		_, err := common.InternalDeleteDweet(postID)

		// Delete the media that isn't used anymore
		oldMedia := deleted.Media
		for _, mediaLink := range oldMedia {
			loc, err := cdn.LinkToLocation(mediaLink)
			if err != nil {
				return schema.DweetType{}, err
			}
			err = cdn.DeleteLocation(loc, true)
			if err != nil {
				return schema.DweetType{}, err
			}
		}

		if err != nil {
			return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
		}

		// Format and return with common likes
		knownUsers := deleted.Author().Following()
		knownUsers = append(knownUsers, *deleted.Author())

		mutualLikes := util.HashIntersectUsers(deleted.LikeUsers(), knownUsers)
		mutualRedweets := util.HashIntersectUsers(deleted.RedweetUsers(), knownUsers)

		formatted := schema.FormatAsDweetType(deleted, mutualLikes, mutualRedweets)
		return formatted, err
	}

	return schema.DweetType{}, fmt.Errorf("internal server error: %v", errors.New("Unauthorized"))
}

// Delete User
func DeleteUser(username string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (schema.UserType, error) {
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

	var user *db.UserModel
	var alsoFollowedBy []db.UserModel
	var alsoFollowing []db.UserModel
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
		case "liked":
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
		default:
			break
		}
	}
	var showEmail bool

	alsoFollowedBy = user.Followers()
	alsoFollowing = user.Following()
	showEmail = true

	// Send back the user requested, along with mutuals in the followers field
	nuser, err := schema.FormatAsUserType(user, alsoFollowedBy, alsoFollowing, objectsToFetch, feedObjectList, showEmail)
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Delete the user
	_, err = common.InternalDeleteUser(username)
	if err != nil {
		return schema.UserType{}, fmt.Errorf("error deleting user: %v", err)
	}
	return nuser, err
}

// Delete a redweet
func DeleteRedweet(postID string, username string) (schema.RedweetType, error) {
	// Validate params
	err := common.Validate.Var(postID, "required,alphanum,len=10")
	if err != nil {
		return schema.RedweetType{}, err
	}

	err = common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.RedweetType{}, err
	}

	redweet, err := common.InternalDeleteRedweet(postID, username)
	if err == db.ErrNotFound {
		return schema.RedweetType{}, fmt.Errorf("redweet not found: %v", err)
	}
	if err != nil {
		return schema.RedweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	user, err := common.Client.User.FindUnique(
		db.User.Username.Equals(username),
	).Exec(common.BaseCtx)
	if err != nil {
		return schema.RedweetType{}, err
	}

	err = cache.UnredweetCacheUpdate(*redweet, *user)
	if err != nil {
		return schema.RedweetType{}, err
	}

	formatted := schema.FormatAsRedweetType(redweet)
	return formatted, err
}
