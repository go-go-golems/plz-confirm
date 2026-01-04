package cli

import (
	"context"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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
	BaseURL     string `glazed.parameter:"base-url"`
	SessionID   string `glazed.parameter:"session-id"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title       string  `glazed.parameter:"title"`
	Message     *string `glazed.parameter:"message"`
	ApproveText *string `glazed.parameter:"approve-text"`
	RejectText  *string `glazed.parameter:"reject-text"`
}

func NewConfirmCommand(layersList ...layers.ParameterLayer) (*ConfirmCommand, error) {
	desc := cmds.NewCommandDescription(
		"confirm",
		cmds.WithShort("Request a confirmation via the agent-ui web frontend"),
		cmds.WithLong("Creates a confirm widget request, waits for the user response, and outputs the result."),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"base-url",
				parameters.ParameterTypeString,
				parameters.WithDefault("http://localhost:3000"),
				parameters.WithHelp("Base URL (default: http://localhost:3000)"),
			),
			parameters.NewParameterDefinition(
				"session-id",
				parameters.ParameterTypeString,
				parameters.WithDefault("global"),
				parameters.WithHelp("Session ID (used for WebSocket scoping)"),
			),
			parameters.NewParameterDefinition(
				"timeout",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(300),
				parameters.WithHelp("Request expiration in seconds (server-side)"),
			),
			parameters.NewParameterDefinition(
				"wait-timeout",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(300),
				parameters.WithHelp("How long to wait for a response in seconds (0 = wait forever)"),
			),
			parameters.NewParameterDefinition(
				"title",
				parameters.ParameterTypeString,
				parameters.WithHelp("Dialog title"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"message",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional dialog message"),
			),
			parameters.NewParameterDefinition(
				"approve-text",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional approve button text"),
			),
			parameters.NewParameterDefinition(
				"reject-text",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional reject button text"),
			),
		),
		cmds.WithLayersList(layersList...),
	)

	return &ConfirmCommand{CommandDescription: desc}, nil
}

func (c *ConfirmCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &ConfirmSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
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
