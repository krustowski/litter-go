package frontend

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"

	"go.savla.dev/littr/config"
	"go.savla.dev/littr/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type PollsPage struct {
	app.Compo
}

func (p *PollsPage) OnNav(ctx app.Context) {
	ctx.Page().SetTitle("polls / littr")
}

func (p *PollsPage) Render() app.UI {
	return app.Div().Body(
		app.Body().Class("dark"),
		&header{},
		&footer{},
		&pollsContent{},
	)
}

type pollsContent struct {
	app.Compo

	loggedUser string
	user       models.User

	loaderShow bool

	toastShow bool
	toastText string

	polls map[string]models.Poll

	pollsButtonDisabled bool
}

func (c *pollsContent) dismissToast(ctx app.Context, e app.Event) {
	c.toastShow = false
	c.pollsButtonDisabled = false
}

func (c *pollsContent) OnNav(ctx app.Context) {
	// show loader
	c.loaderShow = true

	ctx.Async(func() {
		pollsRaw := struct {
			Polls map[string]models.Poll `json:"polls"`
		}{}

		if byteData, _ := litterAPI("GET", "/api/polls", nil); byteData != nil {
			err := json.Unmarshal(*byteData, &pollsRaw)
			if err != nil {
				log.Println(err.Error())
				return
			}
		} else {
			log.Println("cannot fetch polls list")
			return
		}

		// Storing HTTP response in component field:
		ctx.Dispatch(func(ctx app.Context) {
			c.polls = pollsRaw.Polls
			//c.sortedPosts = posts

			c.pollsButtonDisabled = false
			c.loaderShow = false
			log.Println("dispatch ends")
		})
	})
	return
}

func (c *pollsContent) onClickPollOption(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	option := ctx.JSSrc().Get("name").String()
	ctx.NewActionWithValue("vote", []string{key, option})
	c.pollsButtonDisabled = true
}

func (c *pollsContent) handleVote(ctx app.Context, a app.Action) {
	keys, ok := a.Value.([]string)
	if !ok {
		return
	}

	key := keys[0]
	option := keys[1]

	poll := c.polls[key]
	toastText := ""

	poll.Voted = append(poll.Voted, c.user.Nickname)

	// check where to vote
	options := []string{
		poll.OptionOne.Content,
		poll.OptionTwo.Content,
		poll.OptionThree.Content,
	}

	// use the vote
	if found := contains(options, option); found {
		switch option {
		case poll.OptionOne.Content:
			poll.OptionOne.Counter++
			break

		case poll.OptionTwo.Content:
			poll.OptionTwo.Counter++
			break

		case poll.OptionThree.Content:
			poll.OptionThree.Counter++
			break
		}
	} else {
		toastText = "option not associated to the poll well"
	}

	ctx.Async(func() {
		//var toastText string

		if _, ok := litterAPI("PUT", "/api/polls", poll); !ok {
			toastText = "backend error: cannot update a poll"
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.polls[key] = poll

			c.pollsButtonDisabled = false
			c.toastText = toastText
			c.toastShow = (toastText != "")
		})
	})
}

func (c *pollsContent) onClickDelete(ctx app.Context, e app.Event) {
	key := ctx.JSSrc().Get("id").String()
	ctx.NewActionWithValue("delete", key)
	c.pollsButtonDisabled = true
}

func (c *pollsContent) handleDelete(ctx app.Context, a app.Action) {
	key, ok := a.Value.(string)
	if !ok {
		return
	}

	ctx.Async(func() {
		var toastText string = ""

		interactedPoll := c.polls[key]

		if _, ok := litterAPI("DELETE", "/api/polls", interactedPoll); !ok {
			toastText = "backend error: cannot delete a poll"
		}

		ctx.Dispatch(func(ctx app.Context) {
			delete(c.polls, key)

			c.pollsButtonDisabled = false
			c.toastText = toastText
			c.toastShow = (toastText != "")
		})
	})
}

func (c *pollsContent) OnMount(ctx app.Context) {
	var enUser string
	var user models.User
	var toastText string = ""

	ctx.Handle("vote", c.handleVote)
	ctx.Handle("delete", c.handleDelete)

	ctx.LocalStorage().Get("user", &enUser)
	// decode, decrypt and unmarshal the local storage string
	if err := prepare(enUser, &user); err != nil {
		toastText = "frontend decoding/decryption failed: " + err.Error()
	}

	ctx.Dispatch(func(ctx app.Context) {
		c.user = user
		c.loggedUser = user.Nickname
		c.toastText = toastText
		c.toastShow = (toastText != "")
	})
	return
}

// contains checks if a string is present in a slice
// https://freshman.tech/snippets/go/check-if-slice-contains-element/
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func (c *pollsContent) Render() app.UI {
	var sortedPolls []models.Poll

	for _, sortedPoll := range c.polls {
		sortedPolls = append(sortedPolls, sortedPoll)
	}

	// order polls by timestamp DESC
	sort.SliceStable(sortedPolls, func(i, j int) bool {
		return sortedPolls[i].Timestamp.After(sortedPolls[j].Timestamp)
	})

	return app.Main().Class("responsive").Body(
		app.H5().Text("littr polls").Style("padding-top", config.HeaderTopPadding),
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

		app.Table().Class("border left-align").Body(
			app.THead().Body(
				app.Tr().Body(
					app.Th().Class("align-left").Text("poll question, options, timestamp"),
				),
			),
			app.TBody().Body(
				app.Range(sortedPolls).Slice(func(idx int) app.UI {
					poll := sortedPolls[idx]
					key := poll.ID

					userVoted := contains(poll.Voted, c.user.Nickname)

					optionOneShare := 0
					optionTwoShare := 0
					optionThreeShare := 0

					pollCounterSum := 0
					pollCounterSum = poll.OptionOne.Counter + poll.OptionTwo.Counter
					if poll.OptionThree.Content != "" {
						pollCounterSum += poll.OptionThree.Counter
					}

					// at least one vote has to be already recorded to show the progresses
					if pollCounterSum > 0 {
						optionOneShare = poll.OptionOne.Counter * 100 / pollCounterSum
						optionTwoShare = poll.OptionTwo.Counter * 100 / pollCounterSum
						optionThreeShare = poll.OptionThree.Counter * 100 / pollCounterSum
					}

					return app.Tr().Body(
						app.Td().Class("align-left").Body(
							app.P().Body(
								app.Text("Q: "),
								app.B().Text(poll.Question).Class("deep-orange-text space"),
							),
							app.Div().Class("space"),

							// show buttons to vote
							app.If(!userVoted && poll.Author != c.user.Nickname,
								app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionOne.Content).DataSet("option", poll.OptionOne.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionOne.Content).Disabled(c.pollsButtonDisabled),
								app.Div().Class("space"),
								app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionTwo.Content).DataSet("option", poll.OptionTwo.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionTwo.Content).Disabled(c.pollsButtonDisabled),
								app.Div().Class("space"),
								app.If(poll.OptionThree.Content != "",
									app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionThree.Content).DataSet("option", poll.OptionThree.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionThree.Content).Disabled(c.pollsButtonDisabled),
									app.Div().Class("space"),
								),

							// show results instead
							).ElseIf(userVoted || poll.Author == c.user.Nickname,

								// voted option I
								app.Div().Class("medium-space border").Body(
									app.Div().Class("bold progress left deep-orange3 medium padding").Style("clip-path", "polygon(0% 0%, 0% 100%, "+strconv.Itoa(optionOneShare)+"% 100%, "+strconv.Itoa(optionOneShare)+"% 0%);"),
									//app.Progress().Value(strconv.Itoa(optionOneShare)).Max(100).Class("deep-orange-text padding medium bold left"),
									//app.Div().Class("progress left light-green"),
									app.Div().Class("middle right-align bold").Body(
										app.Span().Text(poll.OptionOne.Content+" ("+strconv.Itoa(optionOneShare)+"%)"),
									),
								),

								app.Div().Class("medium-space"),

								// voted option II
								app.Div().Class("medium-space border").Body(
									app.Div().Class("bold progress left deep-orange5 medium padding").Style("clip-path", "polygon(0% 0%, 0% 100%, "+strconv.Itoa(optionTwoShare)+"% 100%, "+strconv.Itoa(optionTwoShare)+"% 0%);").Body(),
									//app.Progress().Value(strconv.Itoa(optionTwoShare)).Max(100).Class("deep-orange-text padding medium bold left"),
									app.Div().Class("middle right-align bold").Body(
										app.Span().Text(poll.OptionTwo.Content+" ("+strconv.Itoa(optionTwoShare)+"%)"),
									),
								),

								app.Div().Class("medium-space"),

								// voted option III
								app.If(poll.OptionThree.Content != "",
									app.Div().Class("medium-space border").Body(
										app.Div().Class("bold progress left deep-orange9 medium padding").Style("clip-path", "polygon(0% 0%, 0% 100%, "+strconv.Itoa(optionThreeShare)+"% 100%, "+strconv.Itoa(optionThreeShare)+"% 0%);"),
										//app.Progress().Value(strconv.Itoa(optionThreeShare)).Max(100).Class("deep-orange-text deep-orange padding medium bold left"),
										app.Div().Class("middle bold right-align").Body(
											app.Span().Text(poll.OptionThree.Content+" ("+strconv.Itoa(optionThreeShare)+"%)"),
										),
									),

									app.Div().Class("medium-space"),
								),
							),

							// bottom row of the poll
							app.Div().Class("row").Body(
								app.Div().Class("max").Body(
									app.Text(poll.Timestamp.Format("Jan 02, 2006; 15:04:05")),
								),
								app.If(poll.Author == c.user.Nickname,
									app.B().Text(len(poll.Voted)),
									app.Button().ID(key).Class("transparent circle").OnClick(c.onClickDelete).Body(
										app.I().Text("delete"),
									),
								).Else(
									app.B().Text(len(poll.Voted)),
									app.Button().ID(key).Class("transparent circle").Disabled(true).Body(
										app.I().Text("how_to_vote"),
									),
								),
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
