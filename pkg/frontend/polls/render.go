package polls

import (
	"sort"
	"strconv"
	"time"

	"go.vxn.dev/littr/pkg/models"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func (c *Content) Render() app.UI {
	toastColor := ""

	switch c.toastType {
	case "success":
		toastColor = "green10"
		break

	case "info":
		toastColor = "blue10"
		break

	default:
		toastColor = "red10"
	}

	var sortedPolls []models.Poll

	for _, sortedPoll := range c.polls {
		sortedPolls = append(sortedPolls, sortedPoll)
	}

	// order polls by timestamp DESC
	sort.SliceStable(sortedPolls, func(i, j int) bool {
		return sortedPolls[i].Timestamp.After(sortedPolls[j].Timestamp)
	})

	// prepare polls according to the actual pagination and pageNo
	pagedPolls := []models.Poll{}

	end := len(sortedPolls)
	start := 0

	stop := func(c *Content) int {
		var pos int

		if c.pagination > 0 {
			// (c.pageNo - 1) * c.pagination + c.pagination
			pos = c.pageNo * c.pagination
		}

		if pos > end {
			// kill the eventListener (observers scrolling)
			c.scrollEventListener()
			c.paginationEnd = true

			return (end)
		}

		if pos < 0 {
			return 0
		}

		return pos
	}(c)

	if end > 0 && stop > 0 {
		pagedPolls = sortedPolls[start:stop]
	}

	return app.Main().Class("responsive").Body(
		app.Div().Class("row").Body(
			app.Div().Class("max padding").Body(
				app.H5().Text("polls"),
				//app.P().Text("brace yourself"),
			),
		),
		app.Div().Class("space"),

		// snackbar
		app.A().OnClick(c.onClickDismiss).Body(
			app.If(c.toastText != "",
				app.Div().ID("snackbar").Class("snackbar "+toastColor+" white-text top active").Body(
					app.I().Text("error"),
					app.Span().Text(c.toastText),
				),
			),
		),

		// poll deletion modal
		app.If(c.deletePollModalShow,
			app.Dialog().ID("delete-modal").Class("grey9 white-text active").Style("border-radius", "8px").Body(
				app.Nav().Class("center-align").Body(
					app.H5().Text("poll deletion"),
				),

				app.Div().Class("space"),
				app.Article().Class("row").Body(
					app.I().Text("warning").Class("amber-text"),
					app.P().Class("max").Body(
						app.Span().Text("are you sure you want to delete your poll?"),
					),
				),
				app.Div().Class("space"),

				app.Div().Class("row").Body(
					app.Button().Class("max border red10 white-text").Style("border-radius", "8px").OnClick(c.onClickDelete).Disabled(c.deleteModalButtonsDisabled).Body(
						app.If(c.deleteModalButtonsDisabled,
							app.Progress().Class("circle white-border small"),
						),
						app.Text("yeah"),
					),
					app.Button().Class("max border black white-text").Style("border-radius", "8px").Text("nope").OnClick(c.onClickDismiss).Disabled(c.deleteModalButtonsDisabled),
				),
			),
		),

		app.Table().Class("left-align border").ID("table-poll").Style("padding", "0 0 2em 0").Style("border-spacing", "0.1em").Body(
			app.TBody().Body(
				app.Range(pagedPolls).Slice(func(idx int) app.UI {
					poll := pagedPolls[idx]
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

					var pollTimestamp string

					// use JS toLocaleString() function to reformat the timestamp
					// use negated LocalTimeMode boolean as true! (HELP)
					if !c.user.LocalTimeMode {
						pollLocale := app.Window().
							Get("Date").
							New(poll.Timestamp.Format(time.RFC3339))

						pollTimestamp = pollLocale.Call("toLocaleString", "en-GB").String()
					} else {
						pollTimestamp = poll.Timestamp.Format("Jan 02, 2006 / 15:04:05")
					}

					return app.Tr().Body(
						app.Td().Attr("data-timestamp", poll.Timestamp.UnixNano()).Class("align-left").Body(
							app.Div().Class("row top-padding").Body(
								app.P().Body(
									app.Span().Title("question").Text("Q: "),
									app.Span().Text(poll.Question).Class("deep-orange-text space bold"),
								),
							),
							app.Div().Class("space"),

							// show buttons to vote
							app.If(!userVoted && poll.Author != c.user.Nickname,
								app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionOne.Content).DataSet("option", poll.OptionOne.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionOne.Content).Disabled(c.pollsButtonDisabled).Style("border-radius", "8px"),
								app.Div().Class("space"),
								app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionTwo.Content).DataSet("option", poll.OptionTwo.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionTwo.Content).Disabled(c.pollsButtonDisabled).Style("border-radius", "8px"),
								app.Div().Class("space"),
								app.If(poll.OptionThree.Content != "",
									app.Button().Class("deep-orange7 bold white-text responsive").Text(poll.OptionThree.Content).DataSet("option", poll.OptionThree.Content).OnClick(c.onClickPollOption).ID(poll.ID).Name(poll.OptionThree.Content).Disabled(c.pollsButtonDisabled).Style("border-radius", "8px"),
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

								app.Div().Class("space"),

								// voted option III
								app.If(poll.OptionThree.Content != "",
									app.Div().Class("space"),
									app.Div().Class("medium-space border").Body(
										app.Div().Class("bold progress left deep-orange9 medium padding").Style("clip-path", "polygon(0% 0%, 0% 100%, "+strconv.Itoa(optionThreeShare)+"% 100%, "+strconv.Itoa(optionThreeShare)+"% 0%);"),
										//app.Progress().Value(strconv.Itoa(optionThreeShare)).Max(100).Class("deep-orange-text deep-orange padding medium bold left"),
										app.Div().Class("middle bold right-align").Body(
											app.Span().Text(poll.OptionThree.Content+" ("+strconv.Itoa(optionThreeShare)+"%)"),
										),
									),

									app.Div().Class("space"),
								),
							),

							// bottom row of the poll
							app.Div().Class("row").Body(
								app.Div().Class("max").Body(
									//app.Text(poll.Timestamp.Format("Jan 02, 2006; 15:04:05")),
									app.Text(pollTimestamp),
								),
								app.If(poll.Author == c.user.Nickname,
									app.B().Title("vote count").Text(len(poll.Voted)),
									app.Button().Title("delete this poll").ID(key).Class("transparent circle").OnClick(c.onClickDeleteButton).Body(
										app.I().Text("delete"),
									),
								).Else(
									app.B().Title("vote count").Text(len(poll.Voted)),
									app.Button().Title("just voting allowed").ID(key).Class("transparent circle").Disabled(true).Body(
										app.I().Text("how_to_vote"),
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