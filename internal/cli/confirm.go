package cli

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/client"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

type ConfirmCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &ConfirmCommand{}

type ConfirmSettings struct {
	BaseURL     string `glazed:"base-url"`
	SessionID   string `glazed:"session-id"`
	TimeoutS    int    `glazed:"timeout"`
	WaitTimeout int    `glazed:"wait-timeout"`

	Title       string  `glazed:"title"`
	Message     *string `glazed:"message"`
	ApproveText *string `glazed:"approve-text"`
	RejectText  *string `glazed:"reject-text"`
}

func NewConfirmCommand() (*ConfirmCommand, error) {
	desc := cmds.NewCommandDescription(
		"confirm",
		cmds.WithShort("Request a confirmation via the agent-ui web frontend"),
		cmds.WithLong("Creates a confirm widget request, waits for the user response, and outputs the result."),
		cmds.WithFlags(
			fields.New(
				"base-url",
				fields.TypeString,
				fields.WithDefault("http://localhost:3000"),
				fields.WithHelp("Base URL (default: http://localhost:3000)"),
			),
			fields.New(
				"session-id",
				fields.TypeString,
				fields.WithDefault("global"),
				fields.WithHelp("Session ID (used for WebSocket scoping)"),
			),
			fields.New(
				"timeout",
				fields.TypeInteger,
				fields.WithDefault(300),
				fields.WithHelp("Request expiration in seconds (server-side)"),
			),
			fields.New(
				"wait-timeout",
				fields.TypeInteger,
				fields.WithDefault(300),
				fields.WithHelp("How long to wait for a response in seconds (0 = wait forever)"),
			),
			fields.New(
				"title",
				fields.TypeString,
				fields.WithHelp("Dialog title"),
				fields.WithRequired(true),
			),
			fields.New(
				"message",
				fields.TypeString,
				fields.WithHelp("Optional dialog message"),
			),
			fields.New(
				"approve-text",
				fields.TypeString,
				fields.WithHelp("Optional approve button text"),
			),
			fields.New(
				"reject-text",
				fields.TypeString,
				fields.WithHelp("Optional reject button text"),
			),
		),
	)

	return &ConfirmCommand{CommandDescription: desc}, nil
}

func (c *ConfirmCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	settings := &ConfirmSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	cl := client.New(settings.BaseURL)
	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      v1.WidgetType_confirm,
		SessionID: settings.SessionID,
		Input: &v1.ConfirmInput{
			Title:       settings.Title,
			Message:     settings.Message,
			ApproveText: settings.ApproveText,
			RejectText:  settings.RejectText,
		},
		TimeoutS: settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create confirm request")
	}

	completed, err := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for confirm response")
	}

	if completed.Status != v1.RequestStatus_completed {
		return errors.Errorf("request %s ended with status=%s", created.Id, completed.Status.String())
	}

	out := completed.GetConfirmOutput()

	comment := ""
	if out != nil && out.Comment != nil {
		comment = *out.Comment
	}

	row := types.NewRow(
		types.MRP("request_id", created.Id),
		types.MRP("approved", out.GetApproved()),
		types.MRP("timestamp", out.GetTimestamp()),
		types.MRP("comment", comment),
	)
	return gp.AddRow(ctx, row)
}
