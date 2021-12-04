package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/soumitradev/Dwitter/backend/cdn"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/util"
)

// Update a dweet
func UpdateDweet(postID string, username string, body string, mediaLinks []string, repliesToFetch int, replyOffset int) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(postID, "required,alphanum,len=10")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(body, "lte=240")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(mediaLinks, "lte=8,dive,required,url")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(replyOffset, "gte=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	post, err := common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(postID),
	).With(
		db.Dweet.Author.Fetch(),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Check if user owns dweet
	if post.Author().Username != username {
		return schema.DweetType{}, fmt.Errorf("authorization error: %v", errors.New("not authorized to edit dweet"))
	}

	// Delete the media that isn't used anymore
	oldMedia := post.Media
	toDelete := util.HashDifference(oldMedia, mediaLinks)
	for _, mediaLink := range toDelete {
		loc, err := cdn.LinkToLocation(mediaLink)
		if err != nil {
			return schema.DweetType{}, err
		}
		err = cdn.DeleteLocation(loc, true)
		if err != nil {
			return schema.DweetType{}, err
		}
	}

	// Check params and return data accordingly
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
				db.Dweet.PostedAt.Order(db.DESC),
			),
			db.Dweet.LikeUsers.Fetch(),
			db.Dweet.RedweetUsers.Fetch(),
		).Update(
			db.Dweet.DweetBody.Set(body),
			db.Dweet.Media.Set(mediaLinks),
			db.Dweet.LastUpdatedAt.Set(time.Now()),
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
				db.Dweet.PostedAt.Order(db.DESC),
			).Take(repliesToFetch).Skip(replyOffset),
			db.Dweet.LikeUsers.Fetch(),
			db.Dweet.RedweetUsers.Fetch(),
		).Update(
			db.Dweet.DweetBody.Set(body),
			db.Dweet.Media.Set(mediaLinks),
			db.Dweet.LastUpdatedAt.Set(time.Now()),
		).Exec(common.BaseCtx)
	}
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Mark media as used to prevent auto-deletion on expiry
	for _, link := range mediaLinks {
		delete(common.MediaCreatedButNotUsed, link)
	}

	// Add common likes and format
	user, err := common.Client.User.FindUnique(
		db.User.Username.Equals(username),
	).With(
		db.User.Following.Fetch(),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	mutualLikes := util.HashIntersectUsers(user.Following(), post.LikeUsers())
	mutualRedweets := util.HashIntersectUsers(user.Following(), post.RedweetUsers())

	npost := schema.FormatAsDweetType(post, mutualLikes, mutualRedweets)
	return npost, err
}

// Update a user
func UpdateUser(username string, name string, email string, bio string, PfpUrl string, followersToFetch int, followersOffset int, followingToFetch int, followingOffset int, objectsToFetch string, feedObjectsToFetch int, feedObjectsOffset int) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(name, "lte=80")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(email, "email,lte=100")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(bio, "lte=160")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(PfpUrl, "url")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(objectsToFetch, "required,alpha,gt=0,oneof=feed dweet redweet redweetedDweet liked")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(followersOffset, "gte=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(followingOffset, "gte=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(feedObjectsOffset, "gte=0")
	if err != nil {
		return schema.UserType{}, err
	}

	var user *db.UserModel
	var feedObjectList []interface{}

	// Check params and return data accordingly
	if followingToFetch < 0 {
		if followersToFetch < 0 {
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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						).Take(feedObjectsOffset+feedObjectsToFetch),
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsOffset+feedObjectsToFetch),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					dweets := user.Dweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, dweets[i])
					}
				case "redweet":
					user, err = common.Client.User.FindUnique(
						db.User.Username.Equals(username),
					).With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweets := user.Redweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweetedDweets := user.RedweetedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					likes := user.LikedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, likes[i])
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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					dweets := user.Dweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, dweets[i])
					}
				case "redweet":
					user, err = common.Client.User.FindUnique(
						db.User.Username.Equals(username),
					).With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweets := user.Redweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweetedDweets := user.RedweetedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch(),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					likes := user.LikedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, likes[i])
					}
				default:
					break
				}
			}

		}
	} else {
		if followersToFetch < 0 {
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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					dweets := user.Dweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, dweets[i])
					}
				case "redweet":
					user, err = common.Client.User.FindUnique(
						db.User.Username.Equals(username),
					).With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweets := user.Redweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweetedDweets := user.RedweetedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch(),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					likes := user.LikedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, likes[i])
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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

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
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch+feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					merged := util.MergeDweetRedweetList(user.Dweets(), user.Redweets())

					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					dweets := user.Dweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, dweets[i])
					}
				case "redweet":
					user, err = common.Client.User.FindUnique(
						db.User.Username.Equals(username),
					).With(
						db.User.Redweets.Fetch().With(
							db.Redweet.Author.Fetch(),
							db.Redweet.RedweetOf.Fetch(),
						).OrderBy(
							db.Redweet.RedweetTime.Order(db.DESC),
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweets := user.Redweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					redweetedDweets := user.RedweetedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
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
						).Take(feedObjectsToFetch).Skip(feedObjectsOffset),
						db.User.Followers.Fetch().Take(followersToFetch).Skip(followersOffset),
						db.User.Following.Fetch().Take(followingToFetch).Skip(followingOffset),
					).Update(
						db.User.Name.Set(name),
						db.User.Email.Set(email),
						db.User.Bio.Set(bio),
						db.User.ProfilePicURL.Set(PfpUrl),
					).Exec(common.BaseCtx)

					likes := user.LikedDweets()
					for i := 0; i < feedObjectsToFetch; i++ {
						feedObjectList = append(feedObjectList, likes[i])
					}
				default:
					break
				}
			}
		}
	}

	if err == db.ErrNotFound {
		return schema.UserType{}, fmt.Errorf("user not found: %v", err)
	}
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	nuser, err := schema.FormatAsUserType(user, user.Followers(), user.Following(), objectsToFetch, feedObjectList)
	return nuser, err
}
