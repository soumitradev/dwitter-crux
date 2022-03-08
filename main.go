package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/graphql-go/handler"
	"github.com/joho/godotenv"
	"github.com/soumitradev/Dwitter/backend/auth"
	"github.com/soumitradev/Dwitter/backend/cdn"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/database"
	"github.com/soumitradev/Dwitter/backend/gql"
	"github.com/soumitradev/Dwitter/backend/middleware"
	"github.com/soumitradev/Dwitter/frontend"
	"github.com/unrolled/secure"
)

func main() {
	// When returning from main(), make sure to disconnect from database
	defer database.DisconnectDB()

	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	// Initialize sendgrid
	common.InitSendgrid()

	// Check for an error in schema at runtime
	if gql.SchemaError != nil {
		panic(gql.SchemaError)
	}

	// Set flag for timeout to close all connections before quitting
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	// Create a new router
	router := mux.NewRouter().StrictSlash(true)

	// Create a validator for data validation
	common.Validate = validator.New()

	// Create a graphql query handler
	h := handler.New(&handler.Config{
		Schema:     &gql.Schema,
		Pretty:     true,
		GraphiQL:   false,
		Playground: true,
		// This is a way to pass context about the request into the resolver function of graphql
		RootObjectFn: func(myCtx context.Context, r *http.Request) map[string]interface{} {
			// Pass down the authorization token to the graphql query
			cookie, _ := r.Cookie("session")
			var sid string
			if cookie != nil {
				cookieString := cookie.Value
				session := auth.ParseCookie(cookieString)
				sid = session.Sid
			} else {
				sid = ""
			}

			return map[string]interface{}{
				"sid": sid,
			}
		},
	})

	// Map /graphql to the graphql handler, and attach a middleware to it
	router.Handle("/api/graphql", h)

	// Handle some API endpoints using a non-GraphQL solution
	router.HandleFunc("/api/login", auth.LoginHandler).Methods("POST")
	router.HandleFunc("/api/verify/{token}", auth.VerifyHandler).Methods("GET")
	router.HandleFunc("/api/media_upload", cdn.UploadMediaHandler).Methods("POST")
	router.HandleFunc("/api/pfp_upload", cdn.UploadPFPHandler).Methods("POST")
	router.HandleFunc("/api/callback", auth.OAuth2callbackHandler)
	router.Handle("/api/subscriptions", common.GraphqlwsHandler)

	// Handle frontend
	frontend := frontend.FrontendHandler{StaticPath: "frontend/dist", IndexPath: "index.html"}
	router.PathPrefix("/").Handler(frontend)

	// Initialize middleware and use it
	secureMiddleware := secure.New(secure.Options{
		FrameDeny: true,
	})

	router.Use(handlers.CompressHandler)
	router.Use(middleware.LoggingHandler)
	router.Use(middleware.ContentTypeHandler)
	router.Use(middleware.RecoveryHandler)
	router.Use(middleware.SizeHandler)
	router.Use(secureMiddleware.Handler)
	// CORS Handler. Make sure to turn on/off in production!
	router.Use(middleware.CORSTestingHandler)

	// Create an HTTP server
	srv := &http.Server{
		Handler: router,
		Addr:    "127.0.0.1:5000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	fmt.Println("Server now running on port 5000, access /graphql")

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println()
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	BaseCtx, cancel := context.WithTimeout(common.BaseCtx, wait)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(BaseCtx)

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-main.BaseCtx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("Shutting down")
	os.Exit(0)
}
