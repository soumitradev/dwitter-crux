package database

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/soumitradev/Dwitter/backend/auth"
	"github.com/soumitradev/Dwitter/backend/cache"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/schema"
	"github.com/soumitradev/Dwitter/backend/subscriptions"
	"github.com/soumitradev/Dwitter/backend/util"
	"golang.org/x/crypto/bcrypt"
)

// Create a User
func SignUpUser(username string, password string, name string, bio string, email string) (schema.UserType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(password, "required,lte=128,gte=8,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=abcdefghijklmnopqrstuvwxyz,containsany=1234567890,containsany=!@#$%^&*`~-_=+/?.")
	if err != nil {
		return schema.UserType{}, fmt.Errorf("password must be minimum eight characters, maximum 128 characters, have at least one uppercase letter, one lowercase letter, one number and one special character : %v", err)
	}

	err = common.Validate.Var(name, "required,lte=80")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(bio, "lte=160")
	if err != nil {
		return schema.UserType{}, err
	}

	err = common.Validate.Var(email, "required,email,lte=100")
	if err != nil {
		return schema.UserType{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Check if user with username or email already exists
	_, err1 := common.Client.User.FindUnique(
		db.User.Username.Equals(username),
	).Exec(common.BaseCtx)
	_, err2 := common.Client.User.FindUnique(
		db.User.Email.Equals(email),
	).Exec(common.BaseCtx)
	if (err1 == db.ErrNotFound) && (err2 == db.ErrNotFound) {
		// Send verification email
		token := util.GenToken(5)

		common.AccountCreatedButNotVerified[token] = username
		link := "http://localhost:5000/api/verify/" + token
		_, err := auth.SendVerificationEmail(email, link)

		go auth.DeleteUserAfterExpire(60, token)

		if err != nil {
			return schema.UserType{}, errors.New("error verifying email, please try again later")
		}

		// Create user if no such user exists
		createdUser, err := common.Client.User.CreateOne(
			db.User.Username.Set(username),
			db.User.PasswordHash.Set(string(passwordHash)),
			db.User.Name.Set(name),
			db.User.Email.Set(email),
			db.User.Bio.Set(bio),
			db.User.ProfilePicURL.Set(common.DefaultPFPURL),
			db.User.TokenVersion.Set(rand.Intn(10000)),
			db.User.CreatedAt.Set(time.Now().UTC()),
			db.User.OAuthProvider.Set("None"),
			db.User.Verified.Set(false),
		).With(
			db.User.Dweets.Fetch().With(
				db.Dweet.Author.Fetch(),
			).OrderBy(
				db.Dweet.PostedAt.Order(db.DESC),
			),
		).Exec(common.BaseCtx)

		if err != nil {
			return schema.UserType{}, fmt.Errorf("internal server error: %v", err)
		}

		nuser, err := schema.FormatAsUserType(createdUser, []db.UserModel{}, []db.UserModel{}, "", []interface{}{}, true)
		return nuser, err
	} else {
		return schema.UserType{}, errors.New("username/email already taken")
	}
}

// Create a Post
func NewDweet(body, username string, mediaLinks []string) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(mediaLinks, "lte=8,dive,required,url")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(body, "required,lte=240,gt=0")
	if err != nil {
		if body == "" {
			err = common.Validate.Var(mediaLinks, "required,gte=1,lte=8,dive,required,url,gt=1")
			if err != nil {
				return schema.DweetType{}, err
			}
		} else {
			return schema.DweetType{}, err
		}
	}

	// Generate a unique ID
	randID := util.GenID(10)
	_, err = common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(randID),
	).Exec(common.BaseCtx)

	for err != db.ErrNotFound {
		randID := util.GenID(10)

		_, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(randID),
		).Exec(common.BaseCtx)
	}

	now := time.Now().UTC()
	createdPost, err := common.Client.Dweet.CreateOne(
		db.Dweet.DweetBody.Set(body),
		db.Dweet.ID.Set(randID),
		db.Dweet.Author.Link(db.User.Username.Equals(username)),
		db.Dweet.Media.Set(mediaLinks),
		db.Dweet.PostedAt.Set(now),
		db.Dweet.LastUpdatedAt.Set(now),
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
	).Exec(common.BaseCtx)
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	err = cache.CreateDweetCacheUpdate(*createdPost)
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Mark media as used to prevent deletion on expiry
	for _, link := range mediaLinks {
		common.MediaCreatedButNotUsed[link] = false
		delete(common.MediaCreatedButNotUsed, link)
	}

	subscriptions.NotifyUserSubscribersDweet("newDweet", *createdPost, createdPost.Author().Subscribers)

	// Format and return
	post := schema.FormatAsDweetType(createdPost, []db.UserModel{}, []db.UserModel{})
	return post, err
}

// Create a Reply
func NewReply(originalPostID string, body string, authorUsername string, mediaLinks []string) (schema.DweetType, error) {
	// Validate params
	err := common.Validate.Var(originalPostID, "required,alphanum,len=10")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(authorUsername, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(mediaLinks, "lte=8,dive,required,url")
	if err != nil {
		return schema.DweetType{}, err
	}

	err = common.Validate.Var(body, "required,lte=240,gt=0")
	if err != nil {
		if body == "" {
			err = common.Validate.Var(mediaLinks, "required,gte=1,lte=8,dive,required,url,gt=1")
			if err != nil {
				return schema.DweetType{}, err
			}
		} else {
			return schema.DweetType{}, err
		}
	}

	// Generate unique ID
	randID := util.GenID(10)
	_, err = common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(randID),
	).Exec(common.BaseCtx)

	for err != db.ErrNotFound {
		randID := util.GenID(10)

		_, err = common.Client.Dweet.FindUnique(
			db.Dweet.ID.Equals(randID),
		).Exec(common.BaseCtx)
	}

	now := time.Now().UTC()
	// Create a Reply
	createdReply, err := common.Client.Dweet.CreateOne(
		db.Dweet.DweetBody.Set(body),
		db.Dweet.ID.Set(randID),
		db.Dweet.Author.Link(db.User.Username.Equals(authorUsername)),
		db.Dweet.Media.Set(mediaLinks),
		db.Dweet.IsReply.Set(true),
		db.Dweet.ReplyTo.Link(
			db.Dweet.ID.Equals(originalPostID),
		),
		db.Dweet.PostedAt.Set(now),
		db.Dweet.LastUpdatedAt.Set(now),
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
	).Exec(common.BaseCtx)
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}
	for _, link := range mediaLinks {
		delete(common.MediaCreatedButNotUsed, link)
	}

	err = cache.CreateReplyCacheUpdate(*createdReply)
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Update original Dweet to show reply
	replied, err := common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(originalPostID),
	).Update(
		db.Dweet.ReplyDweets.Link(
			db.Dweet.ID.Equals(createdReply.ID),
		),
		db.Dweet.ReplyCount.Increment(1),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.DweetType{}, fmt.Errorf("original dweet not found: %v", err)
	}
	if err != nil {
		return schema.DweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	subscriptions.NotifyDweetSubscribers("editDweet", *replied, replied.Subscribers)

	subscriptions.NotifyUserSubscribersDweet("newDweet", *createdReply, createdReply.Author().Subscribers)

	post := schema.FormatAsDweetType(createdReply, []db.UserModel{}, []db.UserModel{})
	return post, err
}

// Create a new Redweet of a Dweet
func Redweet(originalPostID, username string) (schema.RedweetType, error) {
	// Validate params
	err := common.Validate.Var(originalPostID, "required,alphanum,len=10")
	if err != nil {
		return schema.RedweetType{}, err
	}

	err = common.Validate.Var(username, "required,alphanum,lte=20,gt=0")
	if err != nil {
		return schema.RedweetType{}, err
	}

	// Create a Redweet
	user, err := common.Client.User.FindUnique(
		db.User.Username.Equals(username),
	).With(
		db.User.Redweets.Fetch(
			db.Redweet.OriginalRedweetID.Equals(originalPostID),
		),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.RedweetType{}, fmt.Errorf("original dweet not found: %v", err)
	}
	if err != nil {
		return schema.RedweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// If already redweeted, return redweet
	if len(user.Redweets()) > 0 {
		redweet, err := common.Client.Redweet.FindUnique(
			db.Redweet.DbID.Equals(user.Redweets()[0].DbID),
		).With(
			db.Redweet.Author.Fetch(),
			db.Redweet.RedweetOf.Fetch().With(
				db.Dweet.Author.Fetch(),
			),
		).Exec(common.BaseCtx)
		return schema.FormatAsRedweetType(redweet), err
	}

	// Create a Redweet
	createdRedweet, err := common.Client.Redweet.CreateOne(
		db.Redweet.Author.Link(
			db.User.Username.Equals(username),
		),
		db.Redweet.RedweetOf.Link(
			db.Dweet.ID.Equals(originalPostID),
		),
		db.Redweet.RedweetTime.Set(time.Now().UTC()),
	).With(
		db.Redweet.Author.Fetch(),
		db.Redweet.RedweetOf.Fetch().With(
			db.Dweet.Author.Fetch(),
		),
	).Exec(common.BaseCtx)
	if err != nil {
		return schema.RedweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	err = cache.RedweetCacheUpdate(*createdRedweet)
	if err != nil {
		return schema.RedweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	// Update original Dweet to show redweet
	_, err = common.Client.Dweet.FindUnique(
		db.Dweet.ID.Equals(originalPostID),
	).Update(
		db.Dweet.RedweetDweets.Link(
			db.Redweet.DbID.Equals(createdRedweet.DbID),
		),
		db.Dweet.RedweetCount.Increment(1),
	).Exec(common.BaseCtx)
	if err == db.ErrNotFound {
		return schema.RedweetType{}, fmt.Errorf("original dweet not found: %v", err)
	}
	if err != nil {
		return schema.RedweetType{}, fmt.Errorf("internal server error: %v", err)
	}

	subscriptions.NotifyUserSubscribersRedweet("newDweet", *createdRedweet, createdRedweet.Author().Subscribers)

	return schema.FormatAsRedweetType(createdRedweet), err
}
