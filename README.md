# litter-go (littr)

[![Go Reference](https://pkg.go.dev/badge/go.savla.dev/littr.svg)](https://pkg.go.dev/go.savla.dev/littr)
[![Go Report Card](https://goreportcard.com/badge/go.savla.dev/littr)](https://goreportcard.com/report/go.savla.dev/littr)

litter again, now in Go as a PWA --- a microblogging service without notifications (by default off) and pesky messaging, just a raw mind _flow_

[read more](https://krusty.space/projects/litter/) (a blog post)

## repo vademecum

`backend/`
+ REST API backend service
+ service is used by WASM client for fetching of app data

`config/`
+ configuration procedures for litter module packages

`data/`
+ sample data used to flush existing container data by `make flush`

`frontend/`
+ frontend pages

`models/`
+ model declarations

`web/`
+ static web files, logos, web manifest

`.env` + `.env.example`
+ environmental contants and vars for the app to run smoothly (in Docker)

`http_server.go`
+ init app file for the app's backend side with REST API service

`wasm_client.go`
+ lightened version of HTTP server, includes basic app router, lacks REST API service

## how it should work
+ users must register (`/register`) or existed users login (`/login`)
+ users can navigate to the flow (`/flow`) and read other's mind _flows_
+ users can change their passphrase or the _about_ description in settings (`/settings`)
+ posts can be written and sent (`/post`) by any logged-in user
+ users can logout (`/logout`)

## features

+ ~~switchable end-to-end encryption (JSON/octet-stream REST API and LocalStorage for service worker)~~
+ in-memory runtime cache(s)
+ data persistence on container restart (on `SIGINT`)
+ flow posts filtering using the FlowList --- simply choose who to follow
+ can run offline
+ shade function to block other accounts from following and reading their posts

## REST API service (outdated)
+ the service is reachable via (`/api`) endpoint (~~API auth-wall to be implemented~~)
+ to-be-swagger-ized
+ there are six main endpoints: 

```http
POST   /api/auth

GET    /api/dump

POST   /api/flow
GET    /api/flow
PUT    /api/flow
DELETE /api/flow/:key

POST   /api/polls
GET    /api/polls
PUT    /api/polls
DELETE /api/polls/:key

POST   /api/push
PUT    /api/push

# users management
POST   /api/users
GET    /api/users
PUT    /api/users
DELETE /api/users/:key
```

## how to run

```bash
# create env file copy and modify it
cp .env.example .env
vi .env

# build docker image (Docker engine mandatory)
make build

# run docker-compose stack, start up the stack
make run

# Makefile helper print
make info

# flush app data --- copy empty files from 'data/' to the container
make flush kill run
```

## development

`litter-backend` container can be run locally on any dev machine (using Docker engine or using the required tag-locked Go runtime)

```
make dev

http://localhost:8093/flow
```

### investigating the runtime data

```
make sh

ls -laR data/
```

```
data/:
total 44
drwxr-xr-x    1 littr    littr           96 Nov 14 22:21 .
drwxr-sr-x    1 littr    littr           24 Dec 17 20:58 ..
-rw-r--r--    1 littr    littr         2562 Dec 17 22:02 polls.json
-rw-r--r--    1 littr    littr        24309 Dec 17 22:02 posts.json
-rw-r-----    1 littr    littr         1154 Dec 17 22:02 subscriptions.json
-rw-r--r--    1 littr    littr        11923 Dec 17 22:02 users.json
```

In the listing above we can see freshly dumped data from littr runtime. We can fetch it outside the container using `make fetch_run_data`.

```
make fetch_running_dump

 Copying dumped data from the container...

Successfully copied 13.8kB to /home/user/litter-go/run_data/
Successfully copied 4.61kB to /home/user/litter-go/run_data/
Successfully copied 26.1kB to /home/user/litter-go/run_data/
Successfully copied 3.07kB to /home/user/litter-go/run_data/
```

### nice-to-have(s)

+ ~~account deletion (`settings` page)~~
+ ~~add timestamps on server, render them on client side~~
+ ~~automatic links for known TLDs~~
+ ~~autosubmit on password manager autofill (when username and password get filled quite quickly)~~
+ ~~break lines on \n in posts~~
+ check for double posting same content
+ custom colour theme per user
+ deep code refactoring
+ dismiss any modal by clicking anywhere on the screen
+ ~~fix update indicator checker (runs only on once after reload)~~ (autoupdate)
+ implement customizable navbar items order
+ show offline mode notification
+ ~~show user's details on the top of /flow/<username> page~~
+ swagger docs
+ ~~test if dump dir writable (on init)~~ (chown included in Dockerfile)

### known bugs
+ [...] the actual usermap could one obtain using sniffing on the register page via nickname field brute-force changing; the obtained map would be a partial (can't elaborate) usermap copy of the genuine database (to be fixed)

### backlog
+ add periodic subscription checker (older than a week will be deleted on no activity)
+ add timestamp to a subscription
+ implement Ctrl+Enter to submit posts like YouTube
+ implement searching for flow using hashtags
+ improve the UI (review issues related to UI on Github)

### roadmap to v0.27
+ implement mailing (verification mails) on backend
+ ~~implement forgotten password recovery~~ (wip)

### roadmap to v0.26
+ ~~implement combined picture-with-text (PwT) posting~~ (wip)

### roadmap to v0.25
+ ~~implement simple loading/broadcasting of new posts~~ (wip)
+ ~~fix single-post and user flow subpages on flow~~ (wip)
### roadmap to v0.24
+ ~~add modal for post deletion confirmation~~
+ ~~implement JWT for auth (wip)~~
+ ~~preprocess and paginate posts on backend (wip)~~

### roadmap to v0.23
+ ~~reintroduce `/api/stats` route to simplify the stats page~~
+ ~~use a router (Gorilla Mux, Go-Chi)~~

### roadmap to v0.22
+ fix flow reorganize glitch (single page, after post deletion etc) (wip)
+ ~~send frontend's tagged version to backend (improve user debugging)~~

### roadmap to v0.21
+ ~~implement simple notification service (wip)~~
+ ~~implement subscriptions (SubscriptionCache)~~
+ ~~show user's profile on top of the flow single page view~~

### roadmap to v0.20
+ ~~fix thumbnail multiple-loading on scroll~~
+ ~~implement image uploading directly from browser~~

### roadmap to v0.19
+ ~~add avatar preview to settings page~~
+ ~~fix local JS and CSS deps (hotfix via remote CDN)~~
+ ~~fix NaN% when loading~~
+ ~~implement custom JSON log wrapper~~ (~~goroutine?~~)
+ ~~implement `<details>` tag for posts longer than `MaxPostLength)~~
+ ~~implement shadow function (acc blocking)~~
+ ~~implement some user's stats on users page~~
+ ~~show selected user's posts only --- button/link on users page~~

### roadmap to v0.18
+ ~~add links to a single flow post~~
+ ~~show replies (+history) in the single post view~~

### roadmap to v0.17
+ ~~implement user deletion (settings page)~~

### roadmap to v0.16
+ ~~add switch for light/dark mode toggle~~
+ ~~profile icons on the flow page~~ (with link to user info modal???)
+ ~~fix register date (migration)~~

### roadmap to v0.15
+ implement inf. scroll to stats (complex)???
+ ~~replies to any post in flow~~
+ ~~add simple ToS (terms of service)~~

### roadmap to v0.14
+ ~~ensure infinite scroll works properly with flow filtering~~
+ ~~disable update button, autoreload on update instead~~
+ ~~include infinite scroll to `polls` and `users`~~

### roadmap to v0.13
+ ~~implement infinite scroll (for `flow` only this time)~~
+ ~~implement pagination for flow (on backend prolly)~~

### roadmap to v0.12
+ ~~implement user's profile (real name, home page, followers) as modal~~
+ ~~mark selected fields with data labels (`data-author` etc)~~
+ ~~notice user's activity (`User.LastActiveTime`)~~

### roadmap to v0.11
+ ~~upgrade toasts to snackbars (beercss v3.3.5)~~
+ ~~disable buttons on action to prevent multiple action fires~~
+ ~~implement system stats, add flowers (follower) count~~
+ ~~add LitterJS external lib~~

### roadmap to v0.10
+ ~~implement search bar (`stats` and `users` pages only)~~

### roadmap to v0.9
+ ~~implement polls (create poll, voting)~~
+ ~~healthcheck as a periodic cache dumper~~

### roadmap to v0.8
+ ~~implement stats~~
+ ~~ensure system user unregisterable by the app logic (not hardcoded)~~
  - ~~define data/users.json system users (regular users, but simply taken already by the init time)~~
+ ~~implement picture posting~~ (via URL)

### readmap to v0.7
+ ~~implement functional flowList editting via UI~~
+ ~~implement functional filtering using flowList~~
+ ~~fix dynamic stargazing and post deletion~~

### roadmap to v0.6
+ ~~fix settings (broken pwd and about text change)~~
+ ~~fix register when given user exists already~~

### roadmap to v0.4/v0.5
+ ~~stabilize database~~ (implement in-memory caches)
+ ~~fix layout (unwanted less-data page scrolling~~
+ ~~dump data on container restart/SIGINT~~

### roadmap to v0.3
+ ~~use local JSON storage~~
+ ~~implement `backend.authUser`~~
+ ~~functional user login/logout~~
+ ~~functional settings page~~
+ ~~functional add/remove flow user~~

### roadmap to v0.2
+ ~~Go backend (BE) --- server-side~~
+ ~~connect frontend (FE) to BE~~
+ ~~application logic --- functional pages' inputs, buttons, lists (flow, users)~~
