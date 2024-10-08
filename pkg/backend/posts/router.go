package posts

import (
	//"log"
	//"os"
	"time"

	//sse "github.com/tmaxmax/go-sse"
	sse "github.com/alexandrevicenzi/go-sse"
	//sse "github.com/r3labs/sse/v2"
	chi "github.com/go-chi/chi/v5"
	cfg "go.vxn.dev/littr/configs"
)

var Streamer *sse.Server

// Get live flow event stream
//
// @Summary      Get real-time posts event stream (SSE stream)
// @Description  get real-time posts event stream
// @Tags         posts
// @Accept       text/event-stream
// @Produce      text/event-stream
// @Success      200  {object}  string
// @Failure      500  {object}  nil
// @Router       /posts/live [get]
func beat() {
	// ID, data, event
	// https://github.com/alexandrevicenzi/go-sse/blob/master/message.go#L23
	for {
		//Streamer.SendMessage("/api/v1/posts/live", sse.SimpleMessage("heartbeat"))
		Streamer.SendMessage("/api/v1/posts/live", sse.NewMessage("", "heartbeat", "keepalive"))
		time.Sleep(time.Second * cfg.HEARTBEAT_SLEEP_TIME)
	}
}

func Router() chi.Router {
	r := chi.NewRouter()

	Streamer = sse.NewServer(&sse.Options{
		Logger: nil,
	})

	go beat()

	r.Route("/", func(r chi.Router) {
		r.Get("/", getPosts)
		r.Post("/", addNewPost)
		// ->backend/streamer.go
		r.Mount("/live", Streamer)
		// single-post view request
		r.Get("/{postID}", getSinglePost)
		// user flow page request
		/*r.Route("/user", func(r chi.Router) {
			r.Get("/{nick}", getUserPosts)
		})*/
		//r.Put("/", updatePost)
		r.Patch("/{postID}/star", updatePostStarCount)
		r.Delete("/{postID}", deletePost)

		r.Get("/hashtag/{hashtag}", fetchHashtaggedPosts)
	})

	return r
}
