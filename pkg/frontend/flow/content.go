package flow

import (
	"go.vxn.dev/littr/pkg/frontend/common"
	"go.vxn.dev/littr/pkg/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type Content struct {
	app.Compo

	loaderShow      bool
	loaderShowImage bool

	hideReplies bool

	contentLoadFinished bool

	loggedUser string
	user       models.User
	key        string

	toast common.Toast

	buttonDisabled      bool
	postButtonsDisabled bool
	modalReplyActive    bool
	replyPostContent    string
	newFigLink          string
	newFigFile          string
	newFigData          []byte

	escapePressed bool

	deletePostModalShow        bool
	deleteModalButtonsDisabled bool

	interactedPostKey string
	singlePostID      string
	isPost            bool
	userFlowNick      string

	paginationEnd  bool
	pagination     int
	pageNo         int
	pageNoToFetch  int
	lastFire       int64
	processingFire bool

	lastPageFetched bool

	postKey     string
	posts       map[string]models.Post
	users       map[string]models.User
	sortedPosts []models.Post

	refreshClicked bool

	hashtag string

	eventListener        func()
	eventListenerMsg     func()
	keyDownEventListener func()
	dismissEventListener func()
}

func (c *Content) OnMount(ctx app.Context) {
	ctx.Handle("delete", c.handleDelete)
	ctx.Handle("image", c.handleImage)
	ctx.Handle("reply", c.handleReply)
	ctx.Handle("scroll", c.handleScroll)
	ctx.Handle("star", c.handleStar)
	ctx.Handle("clear", c.handleClear)
	ctx.Handle("dismiss", c.handleDismiss)
	ctx.Handle("refresh", c.handleRefresh)
	//ctx.Handle("message", c.handleNewPost)

	c.paginationEnd = false
	c.pagination = 0
	c.pageNo = 1
	c.pageNoToFetch = 0
	c.lastPageFetched = false

	c.deletePostModalShow = false
	c.deleteModalButtonsDisabled = false

	c.eventListener = app.Window().AddEventListener("scroll", c.onScroll)
	//c.eventListenerMsg = app.Window().AddEventListener("message", c.onMessage)
	c.keyDownEventListener = app.Window().AddEventListener("keydown", c.onKeyDown)
	//c.dismissEventListener = app.Window().AddEventListener("click", c.onClickGeneric)
}

func (c *Content) OnDismount() {
	// https://go-app.dev/reference#BrowserWindow
	//c.eventListener()
}

func (c *Content) OnNav(ctx app.Context) {
	ctx.Dispatch(func(ctx app.Context) {
		c.loaderShow = true
		c.loaderShowImage = true
		c.contentLoadFinished = false

		c.toast.TText = ""

		c.posts = nil
		c.users = nil
	})

	toast := common.Toast{AppContext: &ctx}

	isPost := true

	ctx.Async(func() {
		parts := c.parseFlowURI(ctx)

		opts := pageOptions{
			PageNo:   0,
			Context:  ctx,
			CallerID: c.user.Nickname,

			SinglePost:   parts.SinglePost,
			SinglePostID: parts.SinglePostID,
			UserFlow:     parts.UserFlow,
			UserFlowNick: parts.UserFlowNick,
			Hashtag:      parts.Hashtag,
			HideReplies:  c.hideReplies,
		}

		posts, users := c.fetchFlowPage(opts)

		// try the singlePostID/userFlowNick var if present
		if parts.SinglePostID != "" && parts.SinglePost {
			if _, found := (*posts)[parts.SinglePostID]; !found {
				toast.Text("post not found").Type("error").Dispatch(c, dispatch)
			}
		}
		if parts.UserFlowNick != "" && parts.UserFlow {
			if _, found := (*users)[parts.UserFlowNick]; !found {
				toast.Text("user not found").Type("error").Dispatch(c, dispatch)
			}

			if value, found := c.user.FlowList[parts.UserFlowNick]; !value || !found {
				toast.Text("follow the user to see their posts").Type("info").Dispatch(c, dispatch)
			}

			isPost = false
		}

		// Storing HTTP response in component field:
		ctx.Dispatch(func(ctx app.Context) {
			c.pagination = 25
			c.pageNo = 1
			c.pageNoToFetch = 1

			c.user = (*users)[c.key]

			c.users = *users
			c.posts = *posts
			c.singlePostID = parts.SinglePostID
			c.userFlowNick = parts.UserFlowNick
			c.isPost = isPost
			c.hashtag = parts.Hashtag

			c.loaderShow = false
			c.loaderShowImage = false
			c.contentLoadFinished = true
		})
		return
	})
}