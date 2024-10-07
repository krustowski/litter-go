package settings

import (
	"go.vxn.dev/littr/pkg/frontend/common"
	"go.vxn.dev/littr/pkg/helpers"
	"go.vxn.dev/littr/pkg/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type Content struct {
	app.Compo

	// TODO: review this
	loggedUser string

	// used with forms
	passphrase        string
	passphraseAgain   string
	passphraseCurrent string
	aboutText         string
	website           string

	// loaded logged user's struct
	user models.User

	// message toast fields
	toast     common.Toast
	toastShow bool
	toastText string
	toastType string

	darkModeOn   bool
	replyNotifOn bool

	notificationPermission app.NotificationPermission
	subscribed             bool
	subscription           struct {
		Replies  bool
		Mentions bool
	}
	mentionNotificationEnabled bool

	settingsButtonDisabled bool

	deleteAccountModalShow      bool
	deleteSubscriptionModalShow bool

	VAPIDpublic string
	devices     []models.Device
	thisDevice  models.Device

	thisDeviceUUID string
	interactedUUID string

	newFigLink string
	newFigData []byte
	newFigFile string

	keyDownEventListener func()
}

func (c *Content) OnMount(ctx app.Context) {
	c.notificationPermission = ctx.Notifications().Permission()

	ctx.Handle("dismiss", c.handleDismiss)

	c.keyDownEventListener = app.Window().AddEventListener("keydown", c.onKeyDown)
}

func (c *Content) OnNav(ctx app.Context) {
	toast := common.Toast{AppContext: &ctx}

	ctx.Dispatch(func(ctx app.Context) {
		c.settingsButtonDisabled = true
	})

	ctx.Async(func() {
		output := struct {
			PublicKey string          `json:"public_key"`
			User      models.User     `json:"user"`
			Devices   []models.Device `json:"devices"`
			Code      int             `json:"code"`
			Message   string          `json:"message"`
		}{}

		input := common.CallInput{
			Method: "GET",
			Url:    "/api/v1/users/caller",
			PageNo: 0,
		}

		if ok := common.CallAPI(input, &output); !ok {
			toast.Text("cannot fetch data").Type("error").Dispatch(c, dispatch)
			return
		}

		if output.Code == 401 {
			ctx.LocalStorage().Set("user", "")
			ctx.LocalStorage().Set("authGranted", false)

			toast.Text("please log-in again").Type("info").Dispatch(c, dispatch)
			return
		}

		var thisDevice models.Device
		for _, dev := range output.Devices {
			if dev.UUID == ctx.DeviceID() {
				thisDevice = dev
				break
			}
		}

		subscription := c.subscription
		if helpers.Contains(thisDevice.Tags, "reply") {
			subscription.Replies = true
		}
		if helpers.Contains(thisDevice.Tags, "mention") {
			subscription.Mentions = true
		}

		// get the mode
		var mode string
		ctx.LocalStorage().Get("mode", &mode)
		//ctx.LocalStorage().Set("mode", user.AppBgMode)

		ctx.Dispatch(func(ctx app.Context) {
			c.user = output.User
			c.loggedUser = output.User.Nickname
			c.devices = output.Devices

			//c.subscribed = output.Subscribed
			c.subscription = subscription

			c.darkModeOn = mode == "dark"
			//c.darkModeOn = user.AppBgMode == "dark"

			c.VAPIDpublic = output.PublicKey
			c.thisDeviceUUID = ctx.DeviceID()
			c.thisDevice = thisDevice

			c.replyNotifOn = c.notificationPermission == app.NotificationGranted
			//c.replyNotifOn = user.ReplyNotificationOn

			c.settingsButtonDisabled = false
		})
		return
	})
	return
}
