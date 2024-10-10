package post

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func (c *Content) handleDismiss(ctx app.Context, a app.Action) {
	ctx.Dispatch(func(ctx app.Context) {
		c.toast.TText = ""
		c.toast.TType = ""
		c.postButtonsDisabled = false
	})
}