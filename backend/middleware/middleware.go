// Package middleware provides useful custom middleware
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/soumitradev/Dwitter/backend/common"

	"github.com/gorilla/handlers"
)

// Limit size of request
func SizeAndTimeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > (65 << 20) {
			msg := "Request too large."
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})
			return
		}
		startTime := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("Request duration: %s\n", time.Since(startTime).String())
	})
}

// Log requests
func LoggingHandler(next http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, next)
}

// Limit content types
func ContentTypeHandler(next http.Handler) http.Handler {
	return handlers.ContentTypeHandler(next, "application/json", "application/graphql", "multipart/form-data")
}

// Handle recoveries
func RecoveryHandler(next http.Handler) http.Handler {
	return handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(next)
}

func CORSHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		next.ServeHTTP(w, r)
	})
}

func CORSTestingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		next.ServeHTTP(w, r)
	})
}
