package database

import (
	"fmt"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Create a follower relation
func Follow(followedID string, followerID string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(followedID, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(followerID, "required,alphanum,lte=20,gt=0")
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

	// Check if user already followed this user
	personBeingFollowed, err := common.Client.User.FindUnique(
		db.User.Username.Equals(followedID),
	).With(
		db.User.Followers.Fetch(
			db.User.Username.Equals(followerID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	var user *db.UserModel
	var feedObjectList []interface{}

	// If yes, then skip following the user
	if len(personBeingFollowed.Followers()) > 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

				feedObjectList = append(feedObjectList, merged...)

			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				dweets := user.Dweets()
				for i := 0; i < len(dweets); i++ {
					feedObjectList = append(feedObjectList, dweets[i])
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				redweets := user.Redweets()
				for i := 0; i < len(redweets); i++ {
					feedObjectList = append(feedObjectList, redweets[i])
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

				for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
					feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				dweets := user.Dweets()
				for i := 0; i < len(dweets); i++ {
					feedObjectList = append(feedObjectList, dweets[i])
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				redweets := user.Redweets()
				for i := 0; i < len(redweets); i++ {
					feedObjectList = append(feedObjectList, redweets[i])
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				redweetedDweets := user.RedweetedDweets()
				for i := 0; i < len(redweetedDweets); i++ {
					feedObjectList = append(feedObjectList, redweetedDweets[i])
				}
			default:
				break
			}
		}

		if err == db.ErrNotFound {
			return schema.UserType{}, fmt.Errorf("user not found: %v", err)
		}
		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}

		authenticatedUser, err := common.Client.User.FindUnique(
			db.User.Username.Equals(followerID),
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

		knownUsers := authenticatedUser.Following()
		knownUsers = append(knownUsers, *authenticatedUser)

		followers := user.Followers()
		following := user.Following()

		knownFollowers := util.HashIntersectUsers(followers, knownUsers)
		knownFollowing := util.HashIntersectUsers(following, knownUsers)

		formatted, err := schema.FormatAsUserType(user, knownFollowers, knownFollowing, objectsToFetch, feedObjectList, false)
		return formatted, err
	}

	// Else, create new follow relation
	// Add follower to followed's follower list

	if feedObjectsToFetch < 0 {
		switch objectsToFetch {
		case "feed":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			feedObjectList = append(feedObjectList, merged...)
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

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
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
				feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
			}
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Increment(1),
				db.User.Followers.Link(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
			}
		default:
			break
		}
	}
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Add followed to follower's following list
	authenticatedUser, err := common.Client.User.FindUnique(
		db.User.Username.Equals(followerID),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Update(
		db.User.FollowingCount.Increment(1),
		db.User.Following.Link(
			db.User.Username.Equals(followedID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	knownUsers := authenticatedUser.Following()
	knownUsers = append(knownUsers, *authenticatedUser)

	followers := user.Followers()
	following := user.Following()

	// Check if user iss in followers list
	found := false
	for _, follower := range followers {
		if follower.Username == followerID {
			found = true
			break
		}
	}

	// If not, add them
	if !found {
		followers = append(followers, *authenticatedUser)
	}

	knownFollowers := util.HashIntersectUsers(followers, knownUsers)
	knownFollowing := util.HashIntersectUsers(following, knownUsers)

	formatted, err := schema.FormatAsUserType(user, knownFollowers, knownFollowing, objectsToFetch, feedObjectList, false)
	return formatted, err
}

// Add a like to a dweet
func Like(likedPostID string, userID string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(likedPostID, "required,alphanum,len=10")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(userID, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(replyOffset, "gte=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	// Check if user already liked this dweet
	likedPost, err := common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(likedPostID),
	).With(
		db.Dweet.LikeUsers.Fetch(
			db.User.Username.Equals(userID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// If yes, then skip liking the dweet
	if len(likedPost.LikeUsers()) > 0 {
		if repliesToFetch < 0 {
			likedPost, err = common.Client.Dweet.FindUnique(
				db.Dweet.ID.Equals(likedPostID),
			).With(
				db.Dweet.Author.Fetch(),
				db.Dweet.ReplyTo.Fetch().With(
					db.Dweet.Author.Fetch(),
				),
				db.Dweet.ReplyDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.LikeCount.Order(db.DESC),
				),
				db.Dweet.LikeUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.Dweet.RedweetUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
			}
			if err != nil {
				return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
			}
		} else {
			likedPost, err = common.Client.Dweet.FindUnique(
				db.Dweet.ID.Equals(likedPostID),
			).With(
				db.Dweet.Author.Fetch(),
				db.Dweet.ReplyTo.Fetch().With(
					db.Dweet.Author.Fetch(),
				),
				db.Dweet.ReplyDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.LikeCount.Order(db.DESC),
				).Take(repliesToFetch).Skip(replyOffset),
				db.Dweet.LikeUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.Dweet.RedweetUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
			}
			if err != nil {
				return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
			}
		}

		user, err := common.Client.User.FindUnique(
			db.User.Username.Equals(userID),
		).With(
			db.User.Following.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Exec(common.BaseCtx)
		if err == db.ErrNotFound {
			return schema.DweetType{}, fmt.Errorf("user not found: %v", err)
		}
		if err != nil {
			return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
		}

		// Find known people that liked the dweet
		knownUsers := user.Following()
		knownUsers = append(knownUsers, *user)

		mutualLikes := util.HashIntersectUsers(likedPost.LikeUsers(), knownUsers)
		mutualRedweets := util.HashIntersectUsers(likedPost.RedweetUsers(), knownUsers)

		formatted := schema.FormatAsDweetType(likedPost, mutualLikes, mutualRedweets)
		return formatted, err
	}

	// Else, if not already liked,
	// Create a Like on the post if not created already
	var like *db.DweetModel
	if repliesToFetch < 0 {
		like, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(likedPostID),
		).With(
			db.Dweet.Author.Fetch(),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Update(
			db.Dweet.LikeCount.Increment(1),
			db.Dweet.LikeUsers.Link(
				db.User.Username.Equals(userID),
			),
		).Exec(common.BaseCtx)
		if err == db.ErrNotFound {
			return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
		}
		if err != nil {
			return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
		}
	} else {
		like, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(likedPostID),
		).With(
			db.Dweet.Author.Fetch(),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			).Take(repliesToFetch),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Update(
			db.Dweet.LikeCount.Increment(1),
			db.Dweet.LikeUsers.Link(
				db.User.Username.Equals(userID),
			),
		).Exec(common.BaseCtx)
		if err == db.ErrNotFound {
			return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
		}
		if err != nil {
			return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
		}
	}

	// Add post to user's liked dweets
	user, err := common.Client.User.FindUnique(
		db.User.Username.Equals(userID),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Update(
		db.User.LikedDweets.Link(
			db.Dweet.ID.Equals(like.ID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Find known people that liked thw dweet

	knownUsers := user.Following()
	knownUsers = append(knownUsers, *user)

	likes := like.LikeUsers()
	redweets := like.RedweetUsers()

	likes = append(likes, *user)

	mutualLikes := util.HashIntersectUsers(likes, knownUsers)
	mutualRedweets := util.HashIntersectUsers(redweets, knownUsers)

	formatted := schema.FormatAsDweetType(like, mutualLikes, mutualRedweets)

	return formatted, err
}

// Remove a like from a dweet
func Unlike(postID string, userID string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(postID, "required,alphanum,len=10")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(userID, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(replyOffset, "gte=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	likedPost, err := common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(postID),
	).With(
		db.Dweet.LikeUsers.Fetch(
			db.User.Username.Equals(userID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// If yes, then skip unliking the dweet
	if len(likedPost.LikeUsers()) == 0 {
		var post *db.DweetModel
		if repliesToFetch < 0 {
			post, err = common.Client.Dweet.FindUnique(
				db.Dweet.ID.Equals(postID),
			).With(
				db.Dweet.Author.Fetch(),
				db.Dweet.ReplyTo.Fetch().With(
					db.Dweet.Author.Fetch(),
				),
				db.Dweet.ReplyDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.LikeCount.Order(db.DESC),
				),
				db.Dweet.LikeUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.Dweet.RedweetUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
			}
			if err != nil {
				return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
			}
		} else {
			post, err = common.Client.Dweet.FindUnique(
				db.Dweet.ID.Equals(postID),
			).With(
				db.Dweet.Author.Fetch(),
				db.Dweet.ReplyTo.Fetch().With(
					db.Dweet.Author.Fetch(),
				),
				db.Dweet.ReplyDweets.Fetch().With(
					db.Dweet.Author.Fetch(),
				).OrderBy(
					db.Dweet.LikeCount.Order(db.DESC),
				).Take(repliesToFetch).Skip(replyOffset),
				db.Dweet.LikeUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
				db.Dweet.RedweetUsers.Fetch().OrderBy(
					db.User.FollowerCount.Order(db.DESC),
				),
			).Exec(common.BaseCtx)
			if err == db.ErrNotFound {
				return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
			}
			if err != nil {
				return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
			}
		}

		user, err := common.Client.User.FindUnique(
			db.User.Username.Equals(userID),
		).With(
			db.User.Following.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Exec(common.BaseCtx)
		if err == db.ErrNotFound {
			return schema.DweetType{}, fmt.Errorf("user not found: %v", err)
		}
		if err != nil {
			return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
		}

		knownUsers := user.Following()
		knownUsers = append(knownUsers, *user)

		// Find known people that liked the dweet
		mutualLikes := util.HashIntersectUsers(post.LikeUsers(), knownUsers)
		mutualRedweets := util.HashIntersectUsers(post.RedweetUsers(), knownUsers)

		formatted := schema.FormatAsDweetType(post, mutualLikes, mutualRedweets)

		return formatted, err
	}

	// Find the post and decrease its likes by 1
	var post *db.DweetModel
	if repliesToFetch < 0 {
		post, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(postID),
		).With(
			db.Dweet.Author.Fetch(),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Update(
			db.Dweet.LikeCount.Decrement(1),
			db.Dweet.LikeUsers.Unlink(
				db.User.Username.Equals(userID),
			),
		).Exec(common.BaseCtx)
	} else {
		post, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(postID),
		).With(
			db.Dweet.Author.Fetch(),
			db.Dweet.ReplyTo.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
			db.Dweet.ReplyDweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.LikeCount.Order(db.DESC),
			).Take(repliesToFetch),
			db.Dweet.LikeUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
			db.Dweet.RedweetUsers.Fetch().OrderBy(
				db.User.FollowerCount.Order(db.DESC),
			),
		).Update(
			db.Dweet.LikeCount.Decrement(1),
			db.Dweet.LikeUsers.Unlink(
				db.User.Username.Equals(userID),
			),
		).Exec(common.BaseCtx)
	}
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	user, err := common.Client.User.FindUnique(
		db.User.Username.Equals(userID),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	knownUsers := user.Following()
	knownUsers = append(knownUsers, *user)

	// Find known people that liked the dweet
	mutualLikes := util.HashIntersectUsers(post.LikeUsers(), knownUsers)
	mutualRedweets := util.HashIntersectUsers(post.RedweetUsers(), knownUsers)

	var mutualLikesRemoved []db.UserModel

	for _, likeUser := range mutualLikes {
		if user.Username != likeUser.Username {
			mutualLikesRemoved = append(mutualLikesRemoved, likeUser)
		}
	}

	formatted := schema.FormatAsDweetType(post, mutualLikesRemoved, mutualRedweets)

	return formatted, err
}

// Delete a follower relation
func Unfollow(followedID string, followerID string, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(followedID, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(followerID, "required,alphanum,lte=20,gt=0")
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

	// Check if user doesn't follow this user in the first place
	personBeingUnfollowed, err := common.Client.User.FindUnique(
		db.User.Username.Equals(followedID),
	).With(
		db.User.Followers.Fetch(
			db.User.Username.Equals(followerID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	var user *db.UserModel
	var feedObjectList []interface{}

	// If yes, then skip unfollowing the user
	if len(personBeingUnfollowed.Followers()) == 0 {
		if feedObjectsToFetch < 0 {
			switch objectsToFetch {
			case "feed":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

				feedObjectList = append(feedObjectList, merged...)
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				dweets := user.Dweets()
				for i := 0; i < len(dweets); i++ {
					feedObjectList = append(feedObjectList, dweets[i])
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				redweets := user.Redweets()
				for i := 0; i < len(redweets); i++ {
					feedObjectList = append(feedObjectList, redweets[i])
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

				for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
					feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
				}
			case "dweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				dweets := user.Dweets()
				for i := 0; i < len(dweets); i++ {
					feedObjectList = append(feedObjectList, dweets[i])
				}
			case "redweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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
				).Exec(common.BaseCtx)

				redweets := user.Redweets()
				for i := 0; i < len(redweets); i++ {
					feedObjectList = append(feedObjectList, redweets[i])
				}
			case "redweetedDweet":
				user, err = common.Client.User.FindUnique(
					db.User.Username.Equals(followedID),
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

				redweetedDweets := user.RedweetedDweets()
				for i := 0; i < len(redweetedDweets); i++ {
					feedObjectList = append(feedObjectList, redweetedDweets[i])
				}
			default:
				break
			}
		}
		if err == db.ErrNotFound {
			return schema.UserType{}, fmt.Errorf("user not found: %v", err)
		}
		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}

		authenticatedUser, err := common.Client.User.FindUnique(
			db.User.Username.Equals(followerID),
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

		knownUsers := authenticatedUser.Following()
		knownUsers = append(knownUsers, *authenticatedUser)

		// Find known people that liked the dweet
		knownFollowers := util.HashIntersectUsers(user.Followers(), knownUsers)
		knownFollowing := util.HashIntersectUsers(user.Following(), knownUsers)

		formatted, err := schema.FormatAsUserType(user, knownFollowers, knownFollowing, objectsToFetch, feedObjectList, false)
		return formatted, err
	}

	// Add follower to followed's follower list

	if feedObjectsToFetch < 0 {
		switch objectsToFetch {
		case "feed":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			feedObjectList = append(feedObjectList, merged...)
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

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
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

			for i := 0; i < util.Min(feedObjectsToFetch, len(merged)); i++ {
				feedObjectList = append(feedObjectList, merged[i+feedObjectsOffset])
			}
		case "dweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			dweets := user.Dweets()
			for i := 0; i < len(dweets); i++ {
				feedObjectList = append(feedObjectList, dweets[i])
			}
		case "redweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweets := user.Redweets()
			for i := 0; i < len(redweets); i++ {
				feedObjectList = append(feedObjectList, redweets[i])
			}
		case "redweetedDweet":
			user, err = common.Client.User.FindUnique(
				db.User.Username.Equals(followedID),
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
			).Update(
				db.User.FollowerCount.Decrement(1),
				db.User.Followers.Unlink(
					db.User.Username.Equals(followerID),
				),
			).Exec(common.BaseCtx)

			redweetedDweets := user.RedweetedDweets()
			for i := 0; i < len(redweetedDweets); i++ {
				feedObjectList = append(feedObjectList, redweetedDweets[i])
			}
		default:
			break
		}
	}
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Add followed to follower's following list
	authenticatedUser, err := common.Client.User.FindUnique(
		db.User.Username.Equals(followerID),
	).With(
		db.User.Following.Fetch().OrderBy(
			db.User.FollowerCount.Order(db.DESC),
		),
	).Update(
		db.User.FollowingCount.Decrement(1),
		db.User.Following.Unlink(
			db.User.Username.Equals(followedID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	knownUsers := authenticatedUser.Following()
	knownUsers = append(knownUsers, *authenticatedUser)

	// Find known people that liked the dweet
	knownFollowers := util.HashIntersectUsers(user.Followers(), knownUsers)
	knownFollowing := util.HashIntersectUsers(user.Following(), knownUsers)

	var knownFollowersRemoved []db.UserModel

	for _, user := range knownFollowers {
		if authenticatedUser.Username != user.Username {
			knownFollowersRemoved = append(knownFollowersRemoved, user)
		}
	}

	formatted, err := schema.FormatAsUserType(user, knownFollowersRemoved, knownFollowing, objectsToFetch, feedObjectList, false)
	return formatted, err
}
