package users

import (
	"go.vxn.dev/littr/pkg/frontend/common"
	"go.vxn.dev/littr/pkg/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type Content struct {
	app.Compo

	scrollEventListener  func()
	keyDownEventListener func()

	//polls map[string]models.Poll `json:"polls"`
	//posts map[string]models.Post `json:"posts"`
	users map[string]models.User `json:"users"`

	user        models.User
	userInModal models.User

	flowStats map[string]int
	userStats map[string]models.UserStat

	postCount int

	userButtonDisabled bool

	loaderShow bool

	paginationEnd bool
	pagination    int
	pageNo        int

	toast     common.Toast
	toastShow bool
	toastText string
	toastType string

	usersButtonDisabled  bool
	showUserPreviewModal bool
}

func (c *Content) OnNav(ctx app.Context) {
	// show loader
	c.loaderShow = true
	toast := common.Toast{AppContext: &ctx}

	ctx.Async(func() {
		input := common.CallInput{
			Method:      "GET",
			Url:         "/api/v1/users",
			Data:        nil,
			CallerID:    "",
			PageNo:      0,
			HideReplies: false,
		}

		response := struct {
			Users     map[string]models.User     `json:"users"`
			UserStats map[string]models.UserStat `json:"user_stats"`
			Key       string                     `json:"key"`
			Code      int                        `json:"code"`
		}{}

		if ok := common.CallAPI(input, &response); !ok {
			toast.Text("cannot fetch data").Type("error").Dispatch(c, dispatch)
			return
		}

		if response.Code == 401 {
			ctx.LocalStorage().Set("user", "")
			ctx.LocalStorage().Set("authGranted", false)

			toast.Text("please log-in again").Type("info").Dispatch(c, dispatch)
			return
		}

		// manually toggle all users to be "searched for" on init
		for k, u := range response.Users {
			u.Searched = true
			response.Users[k] = u
		}

		// Storing HTTP response in component field:
		ctx.Dispatch(func(ctx app.Context) {
			c.user = response.Users[response.Key]
			c.users = response.Users
			c.userStats = response.UserStats

			//c.posts = postsPre.Posts

			c.pagination = 20
			c.pageNo = 1

			//c.flowStats, c.userStats = c.calculateStats()

			c.loaderShow = false
		})
	})
}

func (c *Content) OnMount(ctx app.Context) {
	ctx.Handle("toggle", c.handleToggle)
	ctx.Handle("search", c.handleSearch)
	ctx.Handle("preview", c.handleUserPreview)
	ctx.Handle("scroll", c.handleScroll)
	ctx.Handle("dismiss", c.handleDismiss)

	c.paginationEnd = false
	c.pagination = 0
	c.pageNo = 1

	c.scrollEventListener = app.Window().AddEventListener("scroll", c.onScroll)
	c.keyDownEventListener = app.Window().AddEventListener("keydown", c.onKeyDown)

	// hotfix to catch panic
	//c.polls = make(map[string]models.Poll)
}
