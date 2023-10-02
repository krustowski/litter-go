package frontend

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"go.savla.dev/littr/config"
	"go.savla.dev/littr/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type StatsPage struct {
	app.Compo
}

type statsContent struct {
	app.Compo

	postCount int
	userCount int

	posts map[string]models.Post
	stats map[string]int
	users map[string]models.User

	flowStats map[string]int
	userStats map[string]userStat

	nicknames []string

	loaderShow bool

	searchString string

	toastShow bool
	toastText string
}

type userStat struct {
	// PostCount is a number of posts of such user.
	PostCount int `default:0`

	// ReactionCount tells the number of interactions (stars given).
	ReactionCount int `default:0`

	// FlowerCount is basically a number of followers.
	FlowerCount int `default:0`

	// Searched is a special boolean used by the search engine to mark who is to be shown in search results.
	Searched bool `default:true`
}

func (p *StatsPage) OnNav(ctx app.Context) {
	ctx.Page().SetTitle("stats / littr")
}

func (p *StatsPage) Render() app.UI {
	return app.Div().Body(
		app.Body().Class("dark"),
		&header{},
		&footer{},
		&statsContent{},
	)
}

func (c *statsContent) onSearch(ctx app.Context, e app.Event) {
	val := ctx.JSSrc().Get("value").String()

	//if c.searchString == "" {
	//if val == "" {
	//	return
	//}

	if len(val) > 20 {
		return
	}

	ctx.NewActionWithValue("search", val)
}

func (c *statsContent) handleSearch(ctx app.Context, a app.Action) {
	matchedList := []string{}

	val, ok := a.Value.(string)
	if !ok {
		return
	}

	ctx.Async(func() {
		users := c.userStats

		// iterate over calculated stats' "rows" and find matchings
		for key, user := range users {
			//user := users[key]
			user.Searched = false

			if strings.Contains(key, val) {
				log.Println(key)
				user.Searched = true

				//matchedList = append(matchedList, key)
			}

			users[key] = user
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.userStats = users
			c.nicknames = matchedList

			c.loaderShow = false
			log.Println("search dispatch")
		})
		return
	})

}

func (c *statsContent) dismissToast(ctx app.Context, e app.Event) {
	c.toastShow = false
	c.toastText = ""
}

func (c *statsContent) OnMount(ctx app.Context) {
	c.loaderShow = true
	ctx.Handle("search", c.handleSearch)
}

func (c *statsContent) OnNav(ctx app.Context) {
	ctx.Async(func() {
		postsRaw := struct {
			Posts map[string]models.Post `json:"posts"`
			Count int                    `json:"count"`
		}{}

		usersRaw := struct {
			Users map[string]models.User `json:"users"`
			Count int                    `json:"count"`
		}{}

		// fetch posts
		if byteData, _ := litterAPI("GET", "/api/flow", nil); byteData != nil {
			err := json.Unmarshal(*byteData, &postsRaw)
			if err != nil {
				log.Println(err.Error())

				ctx.Dispatch(func(ctx app.Context) {
					c.toastText = "backend error: " + err.Error()
					c.toastShow = (c.toastText != "")
				})
				return
			}
		} else {
			log.Println("cannot fetch post flow list")

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = "cannot fetch post flow list"
				c.toastShow = (c.toastText != "")
			})
			return
		}

		// fetch users
		if byteData, _ := litterAPI("GET", "/api/users", nil); byteData != nil {
			err := json.Unmarshal(*byteData, &usersRaw)
			if err != nil {
				log.Println(err.Error())

				ctx.Dispatch(func(ctx app.Context) {
					c.toastText = "backend error: " + err.Error()
					c.toastShow = (c.toastText != "")
				})
				return
			}
		} else {
			log.Println("cannot fetch user list")

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = "cannot fetch user list"
				c.toastShow = (c.toastText != "")
			})
			return
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.posts = postsRaw.Posts
			c.postCount = postsRaw.Count

			c.users = usersRaw.Users
			c.userCount = usersRaw.Count

			c.flowStats, c.userStats = c.calculateStats()

			c.loaderShow = false
			log.Println("dispatch ends")
		})
		return
	})
}

func (c *statsContent) calculateStats() (map[string]int, map[string]userStat) {
	flowStats := make(map[string]int)
	userStats := make(map[string]userStat)
	flowers := make(map[string]int)

	flowStats["posts"] = c.postCount
	//flowStats["users"] = c.userCount
	flowStats["users"] = 0
	flowStats["stars"] = 0

	// iterate over all posts, compose stats results
	for _, val := range c.posts {
		// increment user's stats
		stat, ok := userStats[val.Nickname]
		if !ok {
			// create a blank stat
			stat = userStat{}
			stat.Searched = true
		}

		stat.PostCount++
		stat.ReactionCount += val.ReactionCount
		userStats[val.Nickname] = stat
		flowStats["stars"] += val.ReactionCount
	}

	// iterate over all users, compose global flower count
	for _, user := range c.users {
		for key, enabled := range user.FlowList {
			if enabled {
				flowers[key]++
			}
		}
		flowStats["users"]++
	}

	// iterate over composed flowers, assign the count to a user
	for key, count := range flowers {
		stat := userStats[key]

		// FlowList also contains the user itself
		stat.FlowerCount = (count - 1)
		userStats[key] = stat
	}

	return flowStats, userStats
}

func (c *statsContent) Render() app.UI {
	users := c.userStats
	flowStats := c.flowStats

	return app.Main().Class("responsive").Body(
		app.H5().Text("littr stats").Style("padding-top", config.HeaderTopPadding),
		app.P().Text("wanna know your flow stats? how many you got in the flow and vice versa? yo"),
		app.Div().Class("space"),

		// snackbar
		app.A().OnClick(c.dismissToast).Body(
			app.If(c.toastText != "",
				app.Div().Class("snackbar red10 white-text top active").Body(
					app.I().Text("error"),
					app.Span().Text(c.toastText),
				),
			),
		),

		app.Div().Class("field prefix round fill").Body(
			app.I().Class("front").Text("search"),
			//app.Input().Type("search").OnChange(c.ValueTo(&c.searchString)).OnSearch(c.onSearch),
			//app.Input().ID("search").Type("text").OnChange(c.ValueTo(&c.searchString)).OnSearch(c.onSearch),
			app.Input().ID("search").Type("text").OnChange(c.onSearch).OnSearch(c.onSearch),
		),

		app.Table().Class("border left-align").Body(
			// table header
			app.THead().Body(
				app.Tr().Body(
					app.Th().Class("align-left").Text("nickname"),
					app.Th().Class("align-left").Text("posts"),
					app.Th().Class("align-left").Text("stars"),
					app.Th().Class("align-left").Text("flowers"),
					app.Th().Class("align-left").Text("ratio"),
				),
			),

			// table body
			app.TBody().Body(
				app.Range(users).Map(func(key string) app.UI {
					// calculate the ratio
					ratio := func() float64 {
						if users[key].PostCount <= 0 {
							return 0
						}

						return float64(users[key].ReactionCount) / float64(users[key].PostCount)
					}()

					// filter out unmatched keys
					//log.Printf("%s: %t\n", key, users[key].Searched)

					if !users[key].Searched || c.users[key].Nickname == "system" {
						return app.P().Text("")
					}

					return app.Tr().Body(
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.P().Body(
									app.B().Text(key).Class("deep-orange-text"),
								),
							),
						),
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text(strconv.FormatInt(int64(users[key].PostCount), 10)),
							),
						),
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text(strconv.FormatInt(int64(users[key].ReactionCount), 10)),
							),
						),
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text(strconv.FormatInt(int64(users[key].FlowerCount), 10)),
							),
						),
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text(strconv.FormatFloat(ratio, 'f', 2, 64)),
							),
						),
					)
				}),
			),
		),

		app.Div().Class("large-space"),

		app.H5().Text("system stats"),
		app.P().Text("pop in to see how much this instance lit nocap"),
		app.Div().Class("space"),

		app.Table().Class("border left-align").Body(
			// table header
			app.THead().Body(
				app.Tr().Body(
					app.Th().Class("align-left").Text("property"),
					app.Th().Class("align-left").Text("value"),
				),
			),
			// table body
			app.TBody().Body(
				app.Range(flowStats).Map(func(key string) app.UI {
					return app.Tr().Body(
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.P().Body(
									app.B().Text(key).Class("deep-orange-text"),
								),
							),
						),
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text(strconv.FormatInt(int64(flowStats[key]), 10)),
							),
						),
					)
				}),
			),
		),

		app.If(c.loaderShow,
			app.Div().Class("small-space"),
			app.Div().Class("loader center large deep-orange active"),
		),
	)
}
