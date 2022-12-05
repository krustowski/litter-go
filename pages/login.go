package pages

import (
	"litter-go/backend"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type LoginPage struct {
	app.Compo
	userLogged bool
}

type loginContent struct {
	app.Compo

	nickname   string
	passphrase string

	toastShow bool
	toastText string
}

func (p *LoginPage) OnNav(ctx app.Context) {
	ctx.Page().SetTitle("login / littr")
}

func (p *LoginPage) Render() app.UI {
	return app.Div().Body(
		app.Body().Class("dark"),
		&header{},
		&footer{},
		&loginContent{},
	)
}

func (c *loginContent) onClick(ctx app.Context, e app.Event) {
	ctx.Async(func() {
		c.toastShow = true
		if c.nickname == "" || c.passphrase == "" {
			c.toastText = "all fields need to be filled"
			return
		}

		if ok := backend.AuthUser(c.nickname, c.passphrase); !ok {
			c.toastText = "generic backend error"
			return
		}

		c.toastShow = false
		ctx.Navigate("/flow")
	})
}

func (c *loginContent) dismissToast(ctx app.Context, e app.Event) {
	c.toastShow = false
}

func (c *loginContent) Render() app.UI {
	toastActiveClass := ""
	if c.toastShow {
		toastActiveClass = " active"
	}

	return app.Main().Class("responsive").Body(
		app.H5().Text("littr login"),
		app.P().Body(
			app.A().Href("/register").Text("littr, bc even litter can be lit ---> register here"),
		),
		app.Div().Class("space"),

		app.A().OnClick(c.dismissToast).Body(
			app.Div().Class("toast red10 white-text top"+toastActiveClass).Body(
				app.I().Text("error"),
				app.Span().Text(c.toastText),
			),
		),

		app.Div().Class("field label border deep-orange-text").Body(
			app.Input().Type("text").Required(true).OnChange(c.ValueTo(&c.nickname)),
			app.Label().Text("nickname"),
		),
		app.Div().Class("field label border deep-orange-text").Body(
			app.Input().Type("password").Required(true).OnChange(c.ValueTo(&c.passphrase)),
			app.Label().Text("passphrase"),
		),
		app.Button().Class("responsive deep-orange7 white-text bold").Text("login").OnClick(c.onClick),
	)
}