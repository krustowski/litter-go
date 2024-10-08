package db

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	configs "go.vxn.dev/littr/configs"
	"go.vxn.dev/littr/pkg/backend/common"
	"go.vxn.dev/littr/pkg/helpers"
	"go.vxn.dev/littr/pkg/models"
)

const defaultAvatarImage = "/web/android-chrome-192x192.png"

var urlsChan chan string

// RunMigrations is a "wrapper" function for the migration registration and execution
func RunMigrations() bool {
	l := common.Logger{
		CallerID:   "system",
		WorkerName: "migrations",
		Version:    "system",
	}

	code := http.StatusOK

	// fetch the data
	users, _ := GetAll(UserCache, models.User{})
	//polls, _ := GetAll(PollCache, models.Poll{})
	posts, _ := GetAll(FlowCache, models.Post{})
	subs, _ := GetAll(SubscriptionCache, []models.Device{})

	// migrateEmptyDeviceTags function takes care of filling empty device tags arrays.
	for key, devs := range subs {
		changed := false
		for idx, dev := range devs {
			if len(dev.Tags) == 0 {
				dev.Tags = []string{
					"reply",
					"mention",
				}
				changed = true
				devs[idx] = dev
			}
		}

		if changed {
			SetOne(SubscriptionCache, key, devs)
		}
	}

	// migrateAvatarURL function takes care of (re)assigning custom, or default avatars to all users having blank or default strings saved in their data chunk. Function returns bool based on the process result.
	urlsChan := make(chan string)

	for key, user := range users {
		if user.AvatarURL != "" && user.AvatarURL != defaultAvatarImage {
			continue
		}

		go GetGravatarURL(user.Email, urlsChan)
		url := <-urlsChan

		if url != user.AvatarURL {
			user.AvatarURL = url

			if ok := SetOne(UserCache, key, user); !ok {
				l.Println("migrateAvatarURL: cannot save an avatar: "+key, http.StatusInternalServerError)
				return false
			}
			users[key] = user
		}
	}

	close(urlsChan)

	// migrateFlowPurge function deletes all pseudoaccounts and their posts, those psaudeaccounts are not registered accounts, thus not real users.
	for key, post := range posts {
		if _, found := users[post.Nickname]; !found {
			if deleted := DeleteOne(FlowCache, key); !deleted {
				l.Println("migrateFlowPurge: cannot delete user: "+key, http.StatusInternalServerError)
				return false
			}
			delete(posts, key)
		}
	}

	// migrateUserDeletion function takes care of default users deletion from the database. Function returns bool based on the process result.
	bank := configs.UserDeletionList

	for key, user := range users {
		if helpers.Contains(bank, user.Nickname) {
			l.Println("deleting "+user.Nickname, http.StatusProcessing)

			if deleted := DeleteOne(UserCache, key); !deleted {
				l.Println("migrateUserDeletion: cannot delete an user: "+key, http.StatusInternalServerError)
				return false
			}
			delete(users, key)
		}
	}

	for key, post := range posts {
		if helpers.Contains(bank, post.Nickname) {
			if deleted := DeleteOne(FlowCache, key); !deleted {
				l.Println("migrateUserDeletion: cannot delete a post: "+key, http.StatusInternalServerError)
				return false
			}
			delete(posts, key)
		}
	}

	// migrateUserRegisteredTime function fixes the initial registration date if it defaults to the "null" time.Time string. Function returns bool based on the process result.
	for key, user := range users {
		if user.RegisteredTime == time.Date(0001, 1, 1, 0, 0, 0, 0, time.UTC) {
			user.RegisteredTime = time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
			if ok := SetOne(UserCache, key, user); !ok {
				l.Println("migrateUserRegisteredTime: cannot save an user: "+key, http.StatusInternalServerError)
				return false
			}
			users[key] = user
		}
	}

	// migrateUserShadeList function lists ShadeList items and ensures user shaded (no mutual following, no replying).
	for key, user := range users {
		shadeList := user.ShadeList
		flowList := user.FlowList

		if flowList == nil {
			flowList = make(map[string]bool)
		}

		// ShadeList map[string]bool `json:"shade_list"`
		for name, state := range shadeList {
			if state && name != user.Nickname {
				flowList[name] = false
				user.FlowList = flowList
				//setOne(UserCache, key, user)
			}

		}

		// ensure that users can see themselves
		flowList[key] = true
		user.FlowList = flowList
		if ok := SetOne(UserCache, key, user); !ok {
			//return false
		}
		users[key] = user
	}

	// migrateUserUnshade function lists all users and unshades manually some explicitly list users
	usersToUnshade := configs.UsersToUnshade

	for key, user := range users {
		if !helpers.Contains(usersToUnshade, key) {
			continue
		}

		shadeList := user.ShadeList

		for name := range shadeList {
			if helpers.Contains(usersToUnshade, name) {
				shadeList[name] = false
			}
		}

		user.ShadeList = shadeList
		if ok := SetOne(UserCache, key, user); !ok {
			l.Println("migrateUserUnshade: cannot save an user: "+key, http.StatusInternalServerError)
			return false
		}
		users[key] = user
	}

	// migrateBlankAboutText function loops over user accounts and adds "newbie" where the about-text field is blank
	for key, user := range users {
		if len(user.About) == 0 {
			user.About = "newbie"
		}

		if ok := SetOne(UserCache, key, user); !ok {
			l.Println("migrateBlankAboutText: cannot save an user: "+key, http.StatusInternalServerError)
			return false
		}
		users[key] = user
	}

	// migrateSystemFlowOn function ensures everyone has system account in the flow
	for key, user := range users {
		if user.FlowList == nil {
			user.FlowList = make(map[string]bool)
			user.FlowList[user.Nickname] = true
		}

		user.FlowList["system"] = true

		if ok := SetOne(UserCache, key, user); !ok {
			l.Println("migrateSystemFlowOn: cannot save an user: "+key, http.StatusInternalServerError)
			return false
		}
		users[key] = user
	}

	l.Println("migrations done", code)
	return true
}

/*
 *  helpers
 */

// GetGravatarURL function returns the avatar image location/URL, or it defaults to a app logo.
func GetGravatarURL(emailInput string, channel chan string) string {
	email := strings.ToLower(emailInput)
	size := 150

	sha := sha256.New()
	sha.Write([]byte(email))

	hashedStringEmail := fmt.Sprintf("%x", sha.Sum(nil))

	// hash the emailInput
	//byteEmail := []byte(email)
	//hashEmail := md5.Sum(byteEmail)
	//hashedStringEmail := hex.EncodeToString(hashEmail[:])

	url := "https://www.gravatar.com/avatar/" + hashedStringEmail + "?s=" + strconv.Itoa(size)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		//log.Println(resp.StatusCode)
		//log.Println(err.Error())
		url = defaultAvatarImage
	} else {
		resp.Body.Close()
	}

	// maybe we are running in a goroutine...
	if channel != nil {
		channel <- url
	}
	return url
}
