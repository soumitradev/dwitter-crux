// Package auth provides functions useful for using authentication in this API.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
	"github.com/soumitradev/Dwitter/backend/util"

	"github.com/golang/gddo/httputil/header"
)

var authDB *redis.Client

// A SessionType stores info for a session
type SessionType struct {
	Username string    `json:"username"`
	Sid      string    `json:"sid"`
	Expires  time.Time `json:"expires"`
}

// A loginType stores login info
type loginType struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func InitAuth() {
	authDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6420",
		Password: os.Getenv("REDIS_6420_PASS"),
		DB:       0,
	})
}

// Extract session from cookie
func ParseCookie(cookieString string) SessionType {
	session := SessionType{}
	decodedCookieString, err := base64.StdEncoding.DecodeString(cookieString)
	if err != nil {
		return SessionType{
			Username: "",
			Sid:      "",
			Expires:  time.Unix(0, 0),
		}
	}
	err = json.Unmarshal(decodedCookieString, &session)
	if err != nil {
		return SessionType{
			Username: "",
			Sid:      "",
			Expires:  time.Unix(0, 0),
		}
	}
	return session
}

func generateSession(username string, password string) (SessionType, error) {
	authenticated, err := common.CheckCreds(username, password)
	if authenticated {
		sid := util.GenID(20)
		_, err := authDB.Get(common.BaseCtx, sid).Result()
		for err == nil {
			sid = util.GenID(20)
			_, err = authDB.Get(common.BaseCtx, sid).Result()
		}
		if err != redis.Nil {
			return SessionType{}, err
		}
		session := SessionType{
			Username: username,
			Sid:      sid,
			Expires:  time.Now().UTC().Add(time.Hour * 24),
		}

		// Convert struct into a hashmap
		sessionMap := make(map[string]string)
		sessionMap["username"] = session.Username
		sessionMap["sid"] = session.Sid
		sessionMap["expires"] = session.Expires.UTC().Format(util.TimeUTCFormat)

		err = authDB.HSet(common.BaseCtx, sid, sessionMap).Err()
		if err != nil {
			return SessionType{}, err
		}
		err = authDB.PExpireAt(common.BaseCtx, sid, session.Expires).Err()
		if err != nil {
			return SessionType{}, err
		}

		return session, nil
	} else {
		return SessionType{}, err
	}
}

func VerifySessionID(sessionID string) (SessionType, bool, error) {
	// Handle unauth
	if sessionID == "" {
		return SessionType{}, false, nil
	}

	// Validate session
	res, err := authDB.HGetAll(common.BaseCtx, sessionID).Result()
	if err == nil {
		var session SessionType
		session.Username = res["username"]
		session.Sid = res["sid"]
		session.Expires, err = time.Parse(util.TimeUTCFormat, res["expires"])
		if err != nil {
			return SessionType{}, false, err
		}
		_, err := common.Client.User.FindUnique(
			db.User.Username.Equals(session.Username),
		).Exec(common.BaseCtx)
		if err == db.ErrNotFound {
			return SessionType{}, false, errors.New("user doesn't exist")
		}
		return session, true, nil
	} else {
		return SessionType{}, false, errors.New("unauthorized")
	}
}

// Handles login requests
func LoginHandler(w http.ResponseWriter, r *http.Request) {
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

	// Read a maximum of 1MB from body
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	// Create a JSON decoder and decode the request JSON
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var loginData loginType
	err := decoder.Decode(&loginData)

	// If any error occurred during the decoding, send an appropriate response
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		// Return errors based on what error JSON parser returned
		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})

		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
		}
		return
	}

	// Decode it and check for an external JSON error
	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: msg,
		})
		return
	}

	// After checking for any errors, log the user in, and generate tokens
	sessionData, err := generateSession(loginData.Username, loginData.Password)
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
}

// Check cookie of request and authenticate
func Authenticate(authCookie string) (string, error) {
	session := ParseCookie(authCookie)
	data, isAuth, err := VerifySessionID(session.Sid)
	if (err != nil) || !isAuth {
		return "", errors.New("Unauthorized")
	}

	username := data.Username
	return username, nil
}
