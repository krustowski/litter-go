package backend

import (
	"math/rand"
	"time"

	"go.savla.dev/littr/models"
)

// https://stackoverflow.com/a/34816623
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// https://stackoverflow.com/a/31832326
// https://stackoverflow.com/a/22892986
func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func noteUsersActivity(caller string) bool {
	// check if caller exists
	callerUser, found := getOne(UserCache, caller, models.User{})
	if !found {
		return false
	}

	// update user's activity timestamp
	callerUser.LastActiveTime = time.Now()

	return setOne(UserCache, caller, callerUser)
}
