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
	agenttypes "github.com/go-go-golems/plz-confirm/internal/types"
)

type SelectCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &SelectCommand{}

type SelectSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
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
				"timeout",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(300),
				parameters.WithHelp("Request expiration in seconds (server-side)"),
			),
			parameters.NewParameterDefinition(
				"wait-timeout",
				parameters.ParameterTypeInteger,
				parameters.WithDefault(60),
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

	input := agenttypes.SelectInput{
		Title:      settings.Title,
		Options:    settings.Options,
		Multi:      &settings.Multi,
		Searchable: &settings.Searchable,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      agenttypes.WidgetSelect,
		SessionID: "global", // ignored by server; kept for compatibility
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create select request")
	}

	completed, err := cl.WaitRequest(ctx, created.ID, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for select response")
	}

	// selected can be string or []string (or other, depending on frontend).
	var out agenttypes.SelectOutput
	if completed.Output != nil {
		b, err := json.Marshal(completed.Output)
		if err != nil {
			return errors.Wrap(err, "marshal output")
		}
		if err := json.Unmarshal(b, &out); err != nil {
			return errors.Wrap(err, "unmarshal output")
		}
	}

	selectedJSON, _ := json.Marshal(out.Selected)

	row := types.NewRow(
		types.MRP("request_id", created.ID),
		types.MRP("selected_json", string(selectedJSON)),
	)
	return gp.AddRow(ctx, row)
}
