//go:build !wasm
// +build !wasm

package main

import (
	"compress/flate"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.savla.dev/littr/configs"
	be "go.savla.dev/littr/pkg/backend"
	fe "go.savla.dev/littr/pkg/frontend"
	"go.savla.dev/swis/v5/pkg/core"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func initClient() {
	app.Route("/", &fe.LoginPage{})
	app.Route("/flow", &fe.FlowPage{})
	app.RouteWithRegexp("/flow/post/\\d+", &fe.FlowPage{})
	app.RouteWithRegexp("/flow/user/\\w+", &fe.FlowPage{})
	app.Route("/login", &fe.LoginPage{})
	app.Route("/logout", &fe.LoginPage{})
	app.Route("/polls", &fe.PollsPage{})
	app.Route("/post", &fe.PostPage{})
	app.Route("/register", &fe.RegisterPage{})
	app.Route("/reset", &fe.ResetPage{})
	app.Route("/settings", &fe.SettingsPage{})
	app.Route("/stats", &fe.StatsPage{})
	app.Route("/tos", &fe.ToSPage{})
	app.Route("/users", &fe.UsersPage{})

	app.RunWhenOnBrowser()
}

func initServer() {
	r := chi.NewRouter()

	r.Use(middleware.CleanPath)
	//r.Use(middleware.Logger)
	compressor := middleware.NewCompressor(flate.DefaultCompression,
		"application/wasm", "text/css", "image/svg+xml", "application/json")
	r.Use(compressor.Handler)
	r.Use(middleware.Recoverer)

	// custom listener
	// https://github.com/oderwat/go-nats-app/blob/master/back/back.go
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	// custom server
	// https://github.com/oderwat/go-nats-app/blob/master/back/back.go
	server := &http.Server{
		Addr: listener.Addr().String(),
	}

	// prepare the Logger instance
	l := backend.Logger{
		CallerID:   "system",
		WorkerName: "initServer",
		Version:    "system",
	}

	// parse ENV contants from .env file (should be loaded using Makefile and docker-compose.yml file)
	configs.ParseEnv()

	// initialize caches
	be.FlowCache = &core.Cache{}
	be.PollCache = &core.Cache{}
	be.SubscriptionCache = &core.Cache{}
	be.TokenCache = &core.Cache{}
	be.UserCache = &core.Cache{}

	l.Println("caches initialized", http.StatusOK)

	// load up data from local dumps (/opt/data/)
	// TODO: catch an error there!
	be.LoadAll()

	l.Println("dumped data loaded", http.StatusOK)

	// run migrations
	be.RunMigrations()

	// handle system calls, signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// signals goroutine
	go func() {
		sig := <-sigs
		l.Println("caught signal '"+sig.String()+"', dumping data...", http.StatusCreated)

		be.DumpAll()
	}()

	// API router
	r.Mount("/api/v1", be.APIRouter())

	appHandler := &app.Handler{
		Name:         "litter-go",
		ShortName:    "littr",
		Title:        "littr",
		Description:  "nanoblogging platform as PWA built on go-app framework with beercss and in-memory database",
		Author:       "krusty",
		LoadingLabel: "loading...",
		Lang:         "en",
		Keywords: []string{
			"go-app",
			"nanoblogging",
			"microblogging",
			"social network",
		},
		AutoUpdateInterval: time.Minute * 1,
		Icon: app.Icon{
			Default:    "/web/android-chrome-192x192.png",
			SVG:        "/web/android-chrome-512x512.svg",
			Large:      "/web/android-chrome-512x512.png",
			AppleTouch: "/web/apple-touch-icon.png",
		},
		Image: "/web/android-chrome-512x512.svg",
		Body: func() app.HTMLBody {
			return app.Body().Class("dark")
		},
		BackgroundColor: "#000000",
		ThemeColor:      "#000000",
		Version:         os.Getenv("APP_VERSION") + time.Now().String(),
		Env: map[string]string{
			"APP_VERSION":   os.Getenv("APP_VERSION"),
			"APP_PEPPER":    os.Getenv("APP_PEPPER"),
			"VAPID_PUB_KEY": os.Getenv("VAPID_PUB_KEY"),
		},
		Preconnect: []string{
			"https://cdn.savla.dev/",
		},
		Fonts: []string{
			"https://cdn.savla.dev/webfonts/material-symbols-outlined.woff2",
		},
		Styles: []string{
			"https://cdn.savla.dev/css/beercss.min.css",
			"https://cdn.savla.dev/css/sortable.min.css",
			"/web/litter.css",
		},
		Scripts: []string{
			"https://cdn.savla.dev/js/jquery.min.js",
			"https://cdn.savla.dev/js/beer.nomodule.min.js",
			"https://cdn.savla.dev/js/material-dynamic-colors.nomodule.min.js",
			"https://cdn.savla.dev/js/sortable.min.js",
			"/web/litter.js",
			//"https://cdn.savla.dev/js/litter.js",
			"https://cdn.savla.dev/js/eventsource.min.js",
		},
	}

	r.Handle("/*", appHandler)
	server.Handler = r

	l.Println("starting the server...", http.StatusOK)

	// TODO use http.ErrServerClosed for graceful shutdown
	if err := server.Serve(listener); err != nil {
		panic(err)
	}
}
