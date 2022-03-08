package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"

	"github.com/golang/gddo/httputil/header"
	"golang.org/x/crypto/bcrypt"
)

type DiscordTokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type DiscordUserData struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	AvatarHash string `json:"avatar"`
	Email      string `json:"email"`
}

// Handles login requests (only works with Discord for now)
func OAuth2callbackHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Println(r)

	// Check if content type is "application/json"
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: "Content-Type header is not application/json",
			})
			return
		}
	}

	data := url.Values{}
	data.Set("client_id", os.Getenv("DISCORD_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("DISCORD_CLIENT_SECRET"))
	data.Set("grant_type", "authorization_code")
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("redirect_uri", "http://localhost:5000/callback")

	fmt.Println(data)

	req, err := http.NewRequest("POST", "https://discord.com/api/v8/oauth2/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: err.Error(),
		})
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: err.Error(),
		})
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		tokenData := DiscordTokenData{}
		err = json.NewDecoder(resp.Body).Decode(&tokenData)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}

		req, err := http.NewRequest("GET", "https://discord.com/api/v8/users/@me", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}

		if resp.StatusCode == 200 {
			userData := DiscordUserData{}
			err = json.NewDecoder(resp.Body).Decode(&userData)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			fmt.Println(userData)

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(tokenData.RefreshToken), bcrypt.DefaultCost)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			// Check if user with username or email already exists

			// TODO: Check if username is taken, and allow setting a new username
			_, err1 := common.Client.User.FindUnique(
				db.User.Username.Equals(userData.Username),
			).Exec(common.BaseCtx)
			_, err2 := common.Client.User.FindUnique(
				db.User.Email.Equals(userData.Email),
			).Exec(common.BaseCtx)
			if (err1 == db.ErrNotFound) && (err2 == db.ErrNotFound) {
				// Create user if no such user exists
				_, err := common.Client.User.CreateOne(
					db.User.Username.Set(userData.Username),
					db.User.PasswordHash.Set(string(passwordHash)),
					db.User.Name.Set(userData.Username),
					db.User.Email.Set(userData.Email),
					db.User.Bio.Set(""),
					db.User.ProfilePicURL.Set("https://cdn.discordapp.com/avatars/"+userData.ID+"/"+userData.AvatarHash+".png"),
					db.User.TokenVersion.Set(rand.Intn(10000)),
					db.User.CreatedAt.Set(time.Now()),
					db.User.OAuthProvider.Set("Discord"),
				).With(
					db.User.Dweets.Fetch().With(
						db.Dweet.Author.Fetch(),
					).OrderBy(
						db.Dweet.PostedAt.Order(db.DESC),
					),
				).Exec(common.BaseCtx)

				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: err.Error(),
					})
					return
				}

				// After checking for any errors, log the user in, and generate tokens
				sessionData, err := generateSession(userData.Username, tokenData.RefreshToken)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: err.Error(),
					})
					return
				}

				// Base 64 encode the session data because cookies are fuck
				jsonSession, err := json.Marshal(&sessionData)
				base64SessionString := base64.StdEncoding.EncodeToString(jsonSession)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: fmt.Sprintf("internal server error: %v", err),
					})
					return
				}

				// Send the refresh token in a HTTPOnly cookie
				c := http.Cookie{
					Name:     "session",
					Value:    base64SessionString,
					HttpOnly: true,
					Secure:   true,
					Path:     "/",
					Expires:  sessionData.Expires,
				}
				http.SetCookie(w, &c)

				// Set the response headers
				w.Header().Set("Content-Type", "application/json")
				// Send the access token in JSON
				json.NewEncoder(w).Encode(sessionData)
			} else {
				// TODO: Check if user already exists, and OAuth login them in.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: "Username/Email already taken",
				})
				return
			}
		} else {
			// Send back any errors
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			json.NewEncoder(w).Encode(string(body))
		}
	} else {
		// Send back any errors
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(string(body))
	}
}
