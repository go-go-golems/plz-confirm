package cli

import (
	"context"
	"encoding/json"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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

func NewSelectCommand(layersList ...layers.ParameterLayer) (*SelectCommand, error) {
	desc := cmds.NewCommandDescription(
		"select",
		cmds.WithShort("Request a selection via the agent-ui web frontend"),
		cmds.WithLong("Creates a select widget request, waits for the user selection, and outputs the result."),
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
				"option",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Option value (repeatable)"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"multi",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Allow selecting multiple options"),
			),
			parameters.NewParameterDefinition(
				"searchable",
				parameters.ParameterTypeBool,
				parameters.WithDefault(true),
				parameters.WithHelp("Enable search/filter box in the UI"),
			),
		),
		cmds.WithLayersList(layersList...),
	)

	return &SelectCommand{CommandDescription: desc}, nil
}

func (c *SelectCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &SelectSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
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
