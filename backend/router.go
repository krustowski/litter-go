package backend

import (
	"github.com/go-chi/chi/v5"
)

// the very main API router
func LoadAPIRouter() chi.Router {
	r := chi.NewRouter()

	r.Post("/auth", authHandler)
	r.Get("/dump", dumpHandler)

	r.Route("/flow", func(r chi.Router) {
		r.Get("/{pageNo}", getPosts)
		r.Post("/", postNewPost)
		r.Delete("/", deletePost)
	})
	r.Route("/polls", func(r chi.Router) {
		r.Get("/{pageNo}", getPolls)
		r.Post("/", postNewPoll)
		r.Delete("/", deletePoll)
	})
	r.Route("/push", func(r chi.Router) {
		r.Post("/", subscribeToNotifs)
		r.Put("/", sendNotif)
	})
	r.Route("/users", func(r chi.Router) {
		r.Get("/{pageNo}", getUsers)
		r.Post("/", addNewUser)
		r.Put("/", updateUser)
		r.Delete("/", deleteUser)
	})

	return r
}
