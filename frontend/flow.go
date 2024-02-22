package frontend

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.savla.dev/littr/config"
	"go.savla.dev/littr/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type FlowPage struct {
	app.Compo
}

type flowContent struct {
	app.Compo

	eventListener func()

	loaderShow      bool
	loaderShowImage bool

	loggedUser string
	user       models.User

	toastShow bool
	toastText string

	buttonDisabled      bool
	postButtonsDisabled bool
	modalReplyActive    bool
	replyPostContent    string

	interactedPostKey string
	singlePostID      string
	isPost            bool

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
}

func (p *FlowPage) OnNav(ctx app.Context) {
	ctx.Page().SetTitle("flow / littr")
}

func (p *FlowPage) Render() app.UI {
	return app.Div().Body(
		&header{},
		&footer{},
		&flowContent{},
	)
}

func (c *flowContent) onClickLink(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()

	url := ctx.Page().URL()
	scheme := url.Scheme
	host := url.Host

	// write the link to browsers's clipboard
	app.Window().Get("navigator").Get("clipboard").Call("writeText", scheme+"://"+host+"/flow/post/"+key)
	ctx.Navigate("/flow/post/" + key)
}

func (c *flowContent) onClickDismiss(ctx app.Context, e app.Event) {
	c.toastShow = false
	c.toastText = ""
	c.modalReplyActive = false
	c.buttonDisabled = false
	c.postButtonsDisabled = false
}

func (c *flowContent) onClickImage(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	src := ctx.JSSrc().Get("src").String()

	split := strings.Split(src, ".")
	ext := split[len(split)-1]

	// image preview (thumbnail) to the actual image logic
	if strings.Contains(src, "thumb") {
		ctx.JSSrc().Set("src", "/web/pix/"+key+"."+ext)
		//ctx.JSSrc().Set("style", "max-height: 90vh; max-height: 100%; transition: max-height 0.1s; z-index: 1; max-width: 100%; background-position: center")
		ctx.JSSrc().Set("style", "max-height: 90vh; transition: max-height 0.1s; z-index: 1; max-width: 100%; background-position")
	} else {
		ctx.JSSrc().Set("src", "/web/pix/thumb_"+key+"."+ext)
		ctx.JSSrc().Set("style", "z-index: 0; max-height: 100%; max-width: 100%")
	}
}

func (c *flowContent) handleImage(ctx app.Context, a app.Action) {
	ctx.JSSrc().Set("src", "")
}

func (c *flowContent) onClickUserFlow(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	//c.buttonDisabled = true

	ctx.Navigate("/flow/user/" + key)
}

func (c *flowContent) onClickReply(ctx app.Context, e app.Event) {
	c.interactedPostKey = ctx.JSSrc().Get("id").String()

	c.modalReplyActive = true
	c.buttonDisabled = true
}

func (c *flowContent) onClickPostReply(ctx app.Context, e app.Event) {
	//c.interactedPostKey = ctx.JSSrc().Get("id").String()

	c.modalReplyActive = true
	c.postButtonsDisabled = true
	c.buttonDisabled = true

	ctx.NewAction("reply")
}

func (c *flowContent) handleReply(ctx app.Context, a app.Action) {
	ctx.Async(func() {
		toastText := ""

		// TODO: allow figs in replies
		// check if the contents is a valid URL, then change the type to "fig"
		postType := "post"

		// trim the spaces on the extremites
		replyPost := strings.TrimSpace(c.replyPostContent)

		if replyPost == "" {
			toastText = "no valid reply entered"

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = toastText
				c.toastShow = (toastText != "")
			})
			return
		}

		//newPostID := time.Now()
		//stringID := strconv.FormatInt(newPostID.UnixNano(), 10)

		path := "/api/flow"

		// TODO: the Post data model has to be changed
		// migrate Post.ReplyID (int) to Post.ReplyID (string)
		// ReplyID is to be string key to easily refer to other post
		payload := models.Post{
			//ID:        stringID,
			Nickname: c.user.Nickname,
			Type:     postType,
			Content:  replyPost,
			//Timestamp: newPostID,
			//ReplyTo: replyID, <--- is type int
			ReplyToID: c.interactedPostKey,
		}

		postsRaw := struct {
			Posts map[string]models.Post `posts`
		}{}

		// add new post/poll to backend struct
		if resp, _ := litterAPI("POST", path, payload, c.user.Nickname, c.pageNo); resp != nil {
			err := json.Unmarshal(*resp, &postsRaw)
			if err != nil {
				log.Println(err.Error())
				toastText = "JSON parsing error: " + err.Error()

				ctx.Dispatch(func(ctx app.Context) {
					c.toastText = toastText
					c.toastShow = (toastText != "")
				})
				return
			}
		} else {
			log.Println("cannot fetch post flow list")
			toastText = "API error: cannot fetch the post list"

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = toastText
				c.toastShow = (toastText != "")
			})
			return
		}

		payloadNotif := struct {
			OriginalPost string `json:"original_post"`
		}{
			OriginalPost: c.interactedPostKey,
		}

		// create a notification
		if _, ok := litterAPI("PUT", "/api/push", payloadNotif, c.user.Nickname, c.pageNo); !ok {
			toastText = "cannot PUT new notification"

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = toastText
				c.toastShow = (toastText != "")
			})
			return
		}

		posts := c.posts

		// we do not know the ID, as it is assigned in the BE logic,
		// so we need to loop over the list of posts (1)...
		for k, p := range postsRaw.Posts {
			posts[k] = p
		}

		ctx.Dispatch(func(ctx app.Context) {
			// add new post to post list on frontend side to render
			//c.posts[stringID] = payload
			c.posts = posts

			c.modalReplyActive = false
			c.postButtonsDisabled = false
			c.buttonDisabled = false
		})
	})
}

func (c *flowContent) onScroll(ctx app.Context, e app.Event) {
	ctx.NewAction("scroll")
}

func (c *flowContent) handleScroll(ctx app.Context, a app.Action) {
	ctx.Async(func() {
		elem := app.Window().GetElementByID("page-end-anchor")
		boundary := elem.JSValue().Call("getBoundingClientRect")
		bottom := boundary.Get("bottom").Int()

		_, height := app.Window().Size()

		// limit the fire rate to 1/5 Hz
		now := time.Now().Unix()
		if now-c.lastFire < 2 {
			return
		}

		if bottom-height < 0 && !c.paginationEnd && !c.processingFire {
			ctx.Dispatch(func(ctx app.Context) {
				c.loaderShow = true
				c.processingFire = true
			})

			var newPosts map[string]models.Post
			var newUsers map[string]models.User

			posts := c.posts
			users := c.users

			updated := false
			lastPageFetched := c.lastPageFetched

			// fetch more posts
			//if (c.pageNoToFetch+1)*(c.pagination*2) >= len(posts) && !lastPageFetched {
			if !lastPageFetched {
				opts := pageOptions{
					PageNo:   c.pageNoToFetch,
					Context:  ctx,
					CallerID: c.user.Nickname,
				}

				newPosts, newUsers = c.fetchFlowPage(opts)
				postControlCount := len(posts)

				// patch single-post and user flow atypical scenarios
				if posts == nil {
					posts = make(map[string]models.Post)
				}
				if users == nil {
					users = make(map[string]models.User)
				}

				// append/insert more posts/users
				for key, post := range newPosts {
					posts[key] = post
				}
				for key, user := range newUsers {
					users[key] = user
				}

				updated = true

				// no more posts, fetching another page does not make sense
				if len(posts) == postControlCount {
					updated = false
					lastPageFetched = true

				}
			}

			ctx.Dispatch(func(ctx app.Context) {
				c.lastFire = now
				c.pageNoToFetch++
				c.pageNo++

				if updated {
					c.posts = posts
					c.users = users

					log.Println("updated")
				}

				c.processingFire = false
				c.loaderShow = false
				c.lastPageFetched = lastPageFetched

				log.Println("new content page request fired")
			})
			return
		}
	})
}

func (c *flowContent) onClickDelete(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	ctx.NewActionWithValue("delete", key)
}

func (c *flowContent) handleDelete(ctx app.Context, a app.Action) {
	key, ok := a.Value.(string)
	if !ok {
		return
	}

	c.postKey = key

	ctx.Async(func() {
		var toastText string = ""

		key := c.postKey
		interactedPost := c.posts[key]

		if _, ok := litterAPI("DELETE", "/api/flow", interactedPost, c.user.Nickname, c.pageNo); !ok {
			toastText = "backend error: cannot delete a post"
		}

		ctx.Dispatch(func(ctx app.Context) {
			delete(c.posts, key)

			c.toastText = toastText
			c.toastShow = (toastText != "")
		})
	})
}

func (c *flowContent) onClickStar(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	ctx.NewActionWithValue("star", key)
}

func (c *flowContent) handleStar(ctx app.Context, a app.Action) {
	key, ok := a.Value.(string)
	if !ok {
		return
	}

	// runs on the main UI goroutine via a component ActionHandler
	post := c.posts[key]
	post.ReactionCount++
	c.posts[key] = post
	c.postKey = key

	ctx.Async(func() {
		//var author string
		var toastText string = ""

		//key := ctx.JSSrc().Get("id").String()
		key := c.postKey
		//author = c.user.Nickname

		interactedPost := c.posts[key]
		//interactedPost.ReactionCount++

		postsRaw := struct {
			Posts map[string]models.Post `json:"posts"`
		}{}

		// add new post to backend struct
		if resp, ok := litterAPI("PUT", "/api/flow/star", interactedPost, c.user.Nickname, c.pageNo); ok {
			err := json.Unmarshal(*resp, &postsRaw)
			if err != nil {
				log.Println(err.Error())
				toastText = "JSON parsing error: " + err.Error()

				ctx.Dispatch(func(ctx app.Context) {
					c.toastText = toastText
					c.toastShow = (toastText != "")
				})
				return
			}
		} else {
			toastText = "backend error: cannot rate a post"
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.posts[key] = postsRaw.Posts[key]
			c.toastText = toastText
			c.toastShow = (toastText != "")
		})
	})
}

func (c *flowContent) OnMount(ctx app.Context) {
	ctx.Handle("delete", c.handleDelete)
	ctx.Handle("image", c.handleImage)
	ctx.Handle("reply", c.handleReply)
	ctx.Handle("scroll", c.handleScroll)
	ctx.Handle("star", c.handleStar)

	c.paginationEnd = false
	c.pagination = 0
	c.pageNo = 1
	c.pageNoToFetch = 0
	c.lastPageFetched = false

	var user string
	ctx.LocalStorage().Get("user", &user)
	juser, _ := base64.StdEncoding.DecodeString(user)
	err := json.Unmarshal(juser, &c.user)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println(c.user.Nickname)

	c.eventListener = app.Window().AddEventListener("scroll", c.onScroll)
}

func (c *flowContent) OnDismount() {
	// https://go-app.dev/reference#BrowserWindow
	//c.eventListener()
}

func (c *flowContent) onClickRefresh(ctx app.Context, e app.Event) {
	ctx.Async(func() {
		ctx.Dispatch(func(ctx app.Context) {
			c.loaderShow = true
			c.loaderShowImage = true
			c.refreshClicked = true
			c.postButtonsDisabled = true
			//c.pageNoToFetch = 0

			c.toastText = ""

			c.posts = nil
			c.users = nil
		})

		// nasty hotfix, TODO
		c.pageNoToFetch = 0
		opts := pageOptions{
			PageNo:  c.pageNoToFetch,
			Context: ctx,
			//CallerID: c.user.Nickname,
		}
		posts, users := c.fetchFlowPage(opts)

		ctx.Dispatch(func(ctx app.Context) {
			c.posts = posts
			c.users = users

			c.loaderShow = false
			c.loaderShowImage = false
			c.refreshClicked = false
			c.postButtonsDisabled = false
		})

	})
}

type pageOptions struct {
	PageNo   int `default:0`
	Context  app.Context
	CallerID string

	SinglePost bool `default:false`
	UserFlow   bool `default:false`

	SinglePostID string `default:""`
	UserFlowNick string `default:""`
}

func (c *flowContent) fetchFlowPage(opts pageOptions) (map[string]models.Post, map[string]models.User) {
	var toastText string

	resp := struct {
		Posts map[string]models.Post `json:"posts"`
		Users map[string]models.User `json:"users"`
	}{}

	ctx := opts.Context
	pageNo := opts.PageNo

	if opts.Context == nil {
		toastText = "app context pointer cannot be nil"
		log.Println(toastText)

		return nil, nil
	}

	//pageNo := c.pageNoToFetch
	if c.refreshClicked {
		pageNo = 0
	}
	//pageNoString := strconv.FormatInt(int64(pageNo), 10)

	url := "/api/flow"
	if opts.UserFlow || opts.SinglePost {
		if opts.SinglePostID != "" {
			url += "/post/" + opts.SinglePostID
		}

		if opts.UserFlowNick != "" {
			url += "/user/" + opts.UserFlowNick
		}

		if opts.SinglePostID == "" && opts.UserFlowNick == "" {
			toastText = "single post/user flow parameters cannot be blank"

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = toastText
				c.toastShow = (toastText != "")
				c.refreshClicked = false
			})
			return nil, nil
		}

	}

	if byteData, _ := litterAPI("GET", url, nil, c.user.Nickname, pageNo); byteData != nil {
		err := json.Unmarshal(*byteData, &resp)
		if err != nil {
			log.Println(err.Error())
			toastText = "JSON parsing error: " + err.Error()

			ctx.Dispatch(func(ctx app.Context) {
				c.toastText = toastText
				c.toastShow = (toastText != "")
				c.refreshClicked = false
			})
			return nil, nil
		}
	} else {
		log.Println("cannot fetch the flow page")
		toastText = "API error: cannot fetch the flow page"

		ctx.Dispatch(func(ctx app.Context) {
			c.toastText = toastText
			c.toastShow = (toastText != "")
			c.refreshClicked = false
		})
		return nil, nil
	}

	ctx.Dispatch(func(ctx app.Context) {
		c.refreshClicked = false
	})

	return resp.Posts, resp.Users
}

func (c *flowContent) OnNav(ctx app.Context) {
	c.loaderShow = true
	c.loaderShowImage = true

	toastText := ""

	singlePost := false
	singlePostID := ""
	userFlow := false
	userFlowNick := ""

	isPost := true

	ctx.Async(func() {
		url := strings.Split(ctx.Page().URL().Path, "/")

		if len(url) > 3 && url[3] != "" {
			switch url[2] {
			case "post":
				singlePost = true
				singlePostID = url[3]
				break
			case "user":
				userFlow = true
				userFlowNick = url[3]
				break
			}
		}

		if _, err := strconv.Atoi(singlePostID); singlePostID != "" && err != nil {
			// prolly not a post ID, but an user's nickname
			isPost = false
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.isPost = isPost
		})

		opts := pageOptions{
			PageNo:   0,
			Context:  ctx,
			CallerID: c.user.Nickname,

			SinglePost:   singlePost,
			SinglePostID: singlePostID,
			UserFlow:     userFlow,
			UserFlowNick: userFlowNick,
		}

		posts, users := c.fetchFlowPage(opts)

		// try the singlePostID/userFlowNick var if present
		if singlePostID != "" && singlePost {
			if _, found := posts[singlePostID]; !found {
				toastText = "post not found"
			}
		}
		if userFlowNick != "" && userFlow {
			if _, found := users[userFlowNick]; !found {
				toastText = "user not found"
			}
		}

		// Storing HTTP response in component field:
		ctx.Dispatch(func(ctx app.Context) {
			c.pagination = 25
			c.pageNo = 1
			c.pageNoToFetch = 1

			c.users = users
			c.posts = posts
			c.singlePostID = singlePostID

			c.toastText = toastText
			c.toastShow = (toastText != "")

			c.loaderShow = false
			c.loaderShowImage = false
		})
		return
	})
}

func (c *flowContent) sortPosts() []models.Post {
	var sortedPosts []models.Post

	posts := make(map[string]models.Post)
	posts = c.posts

	// fetch posts and put them in an array
	for _, sortedPost := range posts {
		// do not append a post that is not meant to be shown
		if !c.user.FlowList[sortedPost.Nickname] && sortedPost.Nickname != "system" {
			continue
		}

		sortedPosts = append(sortedPosts, sortedPost)
	}

	return sortedPosts
}

func (c *flowContent) Render() app.UI {
	counter := 0

	sortedPosts := c.sortPosts()

	// order posts by timestamp DESC
	sort.SliceStable(sortedPosts, func(i, j int) bool {
		return sortedPosts[i].Timestamp.After(sortedPosts[j].Timestamp)
	})

	// compose a summary of a long post to be replied to
	replySummary := ""
	if c.modalReplyActive && len(c.posts[c.interactedPostKey].Content) > config.MaxPostLength {
		replySummary = c.posts[c.interactedPostKey].Content[:config.MaxPostLength/10] + "- [...]"
	}

	return app.Main().Class("responsive").Body(
		// page heading
		app.Div().Class("row").Body(
			app.Div().Class("max").Body(
				app.If(c.singlePostID != "" && !c.isPost,
					app.H5().Text(c.singlePostID+"'s flow").Style("padding-top", config.HeaderTopPadding),
					//app.P().Text("exclusive content incoming frfr"),

					// post header (author avatar + name + link button)
					app.Div().Class("row top-padding").Body(
						app.Img().Class("responsive max left").Src(c.users[c.singlePostID].AvatarURL).Style("max-width", "80px").Style("border-radius", "50%"),
						/*;app.P().Class("max").Body(
							app.A().Class("vold deep-orange-text").Text(c.singlePostID).ID(c.singlePostID),
							//app.B().Text(post.Nickname).Class("deep-orange-text"),
						),*/

						app.If(c.users[c.singlePostID].About != "",
							app.Article().Class("max").Style("word-break", "break-word").Style("hyphens", "auto").Text(c.users[c.singlePostID].About),
						),
					),
				).ElseIf(c.singlePostID != "" && c.isPost,
					app.H5().Text("single post and its interactions").Style("padding-top", config.HeaderTopPadding),
				).Else(
					app.H5().Text("littr flow").Style("padding-top", config.HeaderTopPadding),
					app.P().Text("exclusive content incoming frfr"),
				),
			),
			app.Button().Class("border black white-text bold").OnClick(c.onClickRefresh).Disabled(c.postButtonsDisabled).Body(
				app.If(c.refreshClicked,
					app.Progress().Class("circle deep-orange-border small"),
				),
				app.Text("refresh"),
			),
		),
		app.Div().Class("space"),

		// snackbar
		app.A().OnClick(c.onClickDismiss).Body(
			app.If(c.toastText != "",
				app.Div().Class("snackbar red10 white-text top active").Body(
					app.I().Text("error"),
					app.Span().Text(c.toastText),
				),
			),
		),

		// sketchy reply modal
		app.If(c.modalReplyActive,
			app.Dialog().Class("grey9 white-text center-align active").Style("max-width", "90%").Body(
				app.Div().Class("space"),

				app.Article().Class("post").Style("max-width", "100%").Body(
					app.If(replySummary != "",
						app.Details().Body(
							app.Summary().Text(replySummary).Style("word-break", "break-word").Style("hyphens", "auto").Class("italic"),
							app.Div().Class("space"),
							app.Span().Text(c.posts[c.interactedPostKey].Content).Style("word-break", "break-word").Style("hyphens", "auto").Style("font-type", "italic"),
						),
					).Else(
						app.Span().Text(c.posts[c.interactedPostKey].Content).Style("word-break", "break-word").Style("hyphens", "auto").Style("font-type", "italic"),
					),
				),

				app.Div().Class("field textarea label border invalid extra deep-orange-text").Body(
					app.Textarea().Class("active").Name("replyPost").OnChange(c.ValueTo(&c.replyPostContent)).AutoFocus(true),
					app.Label().Text("reply to: "+c.posts[c.interactedPostKey].Nickname).Class("active"),
				),

				app.Nav().Class("center-align").Body(
					app.Button().Class("border deep-orange7 white-text bold").Text("cancel").OnClick(c.onClickDismiss).Disabled(c.postButtonsDisabled),
					app.Button().ID("").Class("border deep-orange7 white-text bold").OnClick(c.onClickPostReply).Disabled(c.postButtonsDisabled).Body(
						app.If(c.postButtonsDisabled,
							app.Progress().Class("circle white-border small"),
						),
						app.Text("reply"),
					),
				),
				app.Div().Class("space"),
			),
		),

		// flow posts/articles
		app.Table().Class("border left-align").ID("table-flow").Body(
			// table body
			app.TBody().Body(
				//app.Range(c.posts).Map(func(key string) app.UI {
				//app.Range(pagedPosts).Slice(func(idx int) app.UI {
				app.Range(sortedPosts).Slice(func(idx int) app.UI {
					counter++
					if counter > c.pagination*c.pageNo {
						return nil
					}

					//post := c.sortedPosts[idx]
					post := sortedPosts[idx]
					key := post.ID

					previousContent := ""

					// prepare reply parameters to render
					if post.ReplyToID != "" {
						//c.posts[post.ReplyToID].ReplyCount++
						if previous, found := c.posts[post.ReplyToID]; found {
							previousContent = previous.Nickname + " posted: " + previous.Content
						} else {
							previousContent = "the post was deleted bye"
						}
					}

					// filter out not-single-post items
					if c.singlePostID != "" {
						if c.isPost && post.ID != c.singlePostID && c.singlePostID != post.ReplyToID {
							return nil
						}

						if _, found := c.users[c.singlePostID]; (!c.isPost && !found) || (found && post.Nickname != c.singlePostID) {
							return nil
						}
					}

					// only show posts of users in one's flowList
					if !c.user.FlowList[post.Nickname] && post.Nickname != "system" {
						return nil
					}

					// check the post's length, on threshold use <details> tag
					postDetailsSummary := ""
					if len(post.Content) > config.MaxPostLength {
						postDetailsSummary = post.Content[:config.MaxPostLength/10] + "- [...]"
					}

					// the same as above with the previous post's length for reply render
					previousDetailsSummary := ""
					if len(previousContent) > config.MaxPostLength {
						previousDetailsSummary = previousContent[:config.MaxPostLength/10] + "- [...]"
					}

					// fetch the image
					var imgSrc string

					// check the URL/URI format
					if _, err := url.ParseRequestURI(post.Content); err == nil {
						imgSrc = post.Content
					} else {
						imgSrc = "/web/pix/thumb_" + post.Content
					}

					// fetch binary image data
					/*if post.Type == "fig" && imgSrc == "" {
						payload := struct {
							PostID  string `json:"post_id"`
							Content string `json:"content"`
						}{
							PostID:  post.ID,
							Content: post.Content,
						}

						var resp *[]byte
						var ok bool

						if resp, ok = litterAPI("POST", "/api/pix", payload, c.user.Nickname); !ok {
							log.Println("api failed")
							imgSrc = "/web/android-chrome-512x512.png"
						} else {
							imgSrc = "data:image/*;base64," + b64.StdEncoding.EncodeToString(*resp)
						}
					}*/

					return app.Tr().Class().Body(
						//app.Td().Class("post align-left").Attr("data-author", post.Nickname).Attr("data-timestamp", post.Timestamp.UnixNano()).On("scroll", c.onScroll).Body(
						app.Td().Class("post align-left").Attr("data-author", post.Nickname).Attr("data-timestamp", post.Timestamp.UnixNano()).Body(

							// post header (author avatar + name + link button)
							app.Div().Class("row top-padding").Body(
								app.Img().Class("responsive max left").Src(c.users[post.Nickname].AvatarURL).Style("max-width", "60px").Style("border-radius", "50%"),
								app.P().Class("max").Body(
									app.A().Class("vold deep-orange-text").OnClick(c.onClickUserFlow).Text(post.Nickname).ID(post.Nickname),
									//app.B().Text(post.Nickname).Class("deep-orange-text"),
								),
							),

							// pic post
							app.If(post.Type == "fig",
								app.Article().Class("medium no-padding transparent").Body(
									app.If(c.loaderShowImage,
										app.Div().Class("small-space"),
										app.Div().Class("loader center large deep-orange active"),
									),
									//app.Img().Class("no-padding absolute center middle lazy").Src(pixDestination).Style("max-width", "100%").Style("max-height", "100%").Attr("loading", "lazy"),
									app.Img().Class("no-padding absolute center middle lazy").Src(imgSrc).Style("max-width", "100%").Style("max-height", "100%").Attr("loading", "lazy").OnClick(c.onClickImage).ID(post.ID),
								),

							// reply + post
							).Else(
								app.If(post.ReplyToID != "",
									app.Article().Class("post black-text yellow10").Style("max-width", "100%").Body(
										app.Div().Class("row max").Body(
											app.If(previousDetailsSummary != "",
												app.Details().Class("max").Body(
													app.Summary().Text(previousDetailsSummary).Style("word-break", "break-word").Style("hyphens", "auto").Class("italic"),
													app.Div().Class("space"),
													app.Span().Class("italic").Text(previousContent).Style("word-break", "break-word").Style("hyphens", "auto").Style("white-space", "pre-line"),
												),
											).Else(
												app.Span().Class("max italic").Text(previousContent).Style("word-break", "break-word").Style("hyphens", "auto").Style("white-space", "pre-line"),
											),

											app.Button().ID(post.ReplyToID).Class("transparent circle").OnClick(c.onClickLink).Disabled(c.buttonDisabled).Body(
												app.I().Text("history"),
											),
										),
									),
								),
								app.Article().Class("post").Style("max-width", "100%").Body(
									app.If(postDetailsSummary != "",
										app.Details().Body(
											app.Summary().Text(postDetailsSummary).Style("hyphens", "auto").Style("word-break", "break-word"),
											app.Div().Class("space"),
											app.Span().Text(post.Content).Style("word-break", "break-word").Style("hyphens", "auto").Style("white-space", "pre-line"),
										),
									).Else(
										app.Span().Text(post.Content).Style("word-break", "break-word").Style("hyphens", "auto").Style("white-space", "pre-line"),
									),
								),
							),

							// post footer (timestamp + reply buttom + star/delete button)
							app.Div().Class("row").Body(
								app.Div().Class("max").Body(
									app.Text(post.Timestamp.Format("Jan 02, 2006 / 15:04:05")),
								),
								app.If(post.Nickname != "system",
									//app.B().Text(post.ReplyCount).Class("left-padding"),
									app.Button().ID(key).Class("transparent circle").OnClick(c.onClickReply).Disabled(c.buttonDisabled).Body(
										app.I().Text("reply"),
									),
									app.Button().ID(key).Class("transparent circle").OnClick(c.onClickLink).Disabled(c.buttonDisabled).Body(
										app.I().Text("link"),
									),
								),
								app.If(c.user.Nickname == post.Nickname,
									app.B().Text(post.ReactionCount).Class("left-padding"),
									app.Button().ID(key).Class("transparent circle").OnClick(c.onClickDelete).Disabled(c.buttonDisabled).Body(
										app.I().Text("delete"),
									),
								).Else(
									app.B().Text(post.ReactionCount).Class("left-padding"),
									app.Button().ID(key).Class("transparent circle").OnClick(c.onClickStar).Disabled(c.buttonDisabled).Body(
										//app.I().Text("ac_unit"),
										app.I().Text("pill"),
									),
								),
							),
						),
					)
				}),
			),
		),
		app.Div().ID("page-end-anchor"),
		app.If(c.loaderShow,
			app.Div().Class("small-space"),
			app.Progress().Class("circle center large deep-orange-border active"),
		),
	)
}
