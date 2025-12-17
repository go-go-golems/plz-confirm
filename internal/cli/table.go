package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/client"
	agenttypes "github.com/go-go-golems/plz-confirm/internal/types"
)

type TableCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &TableCommand{}

type TableSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title       string   `glazed.parameter:"title"`
	Data        string   `glazed.parameter:"data"`
	Columns     []string `glazed.parameter:"columns"`
	MultiSelect bool     `glazed.parameter:"multi-select"`
	Searchable  bool     `glazed.parameter:"searchable"`
}

func NewTableCommand(layersList ...layers.ParameterLayer) (*TableCommand, error) {
	desc := cmds.NewCommandDescription(
		"table",
		cmds.WithShort("Request table selection via the agent-ui web frontend"),
		cmds.WithLong("Creates a table widget request with rows, waits for the user selection, and outputs the result."),
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
				"data",
				parameters.ParameterTypeString,
				parameters.WithHelp("Path to JSON file with array of row objects (use @file.json or - for stdin)"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"columns",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Optional column names (auto-derived if omitted)"),
			),
			parameters.NewParameterDefinition(
				"multi-select",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Allow selecting multiple rows"),
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

	return &TableCommand{CommandDescription: desc}, nil
}

func (c *TableCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &TableSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return err
	}

	// Read data file
	var dataReader io.Reader
	if settings.Data == "-" {
		dataReader = os.Stdin
	} else {
		// Handle @file.json pattern
		dataPath := settings.Data
		if len(dataPath) > 0 && dataPath[0] == '@' {
			dataPath = dataPath[1:]
		}
		f, err := os.Open(dataPath)
		if err != nil {
			return errors.Wrapf(err, "open data file %s", dataPath)
		}
		defer f.Close()
		dataReader = f
	}

	var rows []any
	if err := json.NewDecoder(dataReader).Decode(&rows); err != nil {
		return errors.Wrap(err, "decode data JSON")
	}

	cl := client.New(settings.BaseURL)
	input := agenttypes.TableInput{
		Title:       settings.Title,
		Data:        rows,
		Columns:     settings.Columns,
		MultiSelect: &settings.MultiSelect,
		Searchable:  &settings.Searchable,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      agenttypes.WidgetTable,
		SessionID: "global", // ignored by server; kept for compatibility
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create table request")
	}

	completed, err := cl.WaitRequest(ctx, created.ID, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for table response")
	}

	var out agenttypes.TableOutput
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
