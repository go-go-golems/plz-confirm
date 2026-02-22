package cli

import (
	"context"
	"encoding/json"

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

type SelectCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &SelectCommand{}

type SelectSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	SessionID   string `glazed.parameter:"session-id"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title      string   `glazed.parameter:"title"`
	Options    []string `glazed.parameter:"option"`
	Multi      bool     `glazed.parameter:"multi"`
	Searchable bool     `glazed.parameter:"searchable"`
}

func NewSelectCommand() (*SelectCommand, error) {
	desc := cmds.NewCommandDescription(
		"select",
		cmds.WithShort("Request a selection via the agent-ui web frontend"),
		cmds.WithLong("Creates a select widget request, waits for the user selection, and outputs the result."),
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
				"option",
				fields.TypeStringList,
				fields.WithHelp("Option value (repeatable)"),
				fields.WithRequired(true),
			),
			fields.New(
				"multi",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Allow selecting multiple options"),
			),
			fields.New(
				"searchable",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Enable search/filter box in the UI"),
			),
		),
	)

	return &SelectCommand{CommandDescription: desc}, nil
}

func (c *SelectCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	settings := &SelectSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	cl := client.New(settings.BaseURL)

	input := &v1.SelectInput{
		Title:      settings.Title,
		Options:    settings.Options,
		Multi:      &settings.Multi,
		Searchable: &settings.Searchable,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      v1.WidgetType_select,
		SessionID: settings.SessionID,
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create select request")
	}

	completed, err := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for select response")
	}

	if completed.Status != v1.RequestStatus_completed {
		return errors.Errorf("request %s ended with status=%s", created.Id, completed.Status.String())
	}

	out := completed.GetSelectOutput()

	var selectedAny any
	if out != nil {
		switch sel := out.Selected.(type) {
		case *v1.SelectOutput_SelectedSingle:
			selectedAny = sel.SelectedSingle
		case *v1.SelectOutput_SelectedMulti:
			if sel.SelectedMulti != nil {
				selectedAny = sel.SelectedMulti.Values
			}
		default:
			selectedAny = nil
		}
	}

	selectedJSON, _ := json.Marshal(selectedAny)
	comment := ""
	if out != nil && out.Comment != nil {
		comment = *out.Comment
	}

	row := types.NewRow(
		types.MRP("request_id", created.Id),
		types.MRP("selected_json", string(selectedJSON)),
		types.MRP("comment", comment),
	)
	return gp.AddRow(ctx, row)
}
