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
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type FormCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &FormCommand{}

type FormSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title  string `glazed.parameter:"title"`
	Schema string `glazed.parameter:"schema"`
}

func NewFormCommand(layersList ...layers.ParameterLayer) (*FormCommand, error) {
	desc := cmds.NewCommandDescription(
		"form",
		cmds.WithShort("Request form input via the agent-ui web frontend"),
		cmds.WithLong("Creates a form widget request based on a JSON Schema, waits for the user input, and outputs the result."),
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
				"schema",
				parameters.ParameterTypeString,
				parameters.WithHelp("Path to JSON Schema file (use @file.json or - for stdin)"),
				parameters.WithRequired(true),
			),
		),
		cmds.WithLayersList(layersList...),
	)

	return &FormCommand{CommandDescription: desc}, nil
}

func (c *FormCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &FormSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return err
	}

	// Read schema file
	var schemaReader io.Reader
	if settings.Schema == "-" {
		schemaReader = os.Stdin
	} else {
		// Handle @file.json pattern
		schemaPath := settings.Schema
		if len(schemaPath) > 0 && schemaPath[0] == '@' {
			schemaPath = schemaPath[1:]
		}
		f, err := os.Open(schemaPath)
		if err != nil {
			return errors.Wrapf(err, "open schema file %s", schemaPath)
		}
		defer func() {
			_ = f.Close()
		}()
		schemaReader = f
	}

	var schema any
	if err := json.NewDecoder(schemaReader).Decode(&schema); err != nil {
		return errors.Wrap(err, "decode schema JSON")
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return errors.Wrap(err, "marshal schema")
	}
	schemaPB := &structpb.Struct{}
	if err := protojson.Unmarshal(schemaBytes, schemaPB); err != nil {
		return errors.Wrap(err, "protojson unmarshal schema into structpb.Struct")
	}

	cl := client.New(settings.BaseURL)
	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      v1.WidgetType_form,
		SessionID: "global", // ignored by server; kept for compatibility
		Input: &v1.FormInput{
			Title:  settings.Title,
			Schema: schemaPB,
		},
		TimeoutS: settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create form request")
	}

	completed, err := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for form response")
	}

	out := completed.GetFormOutput()

	dataJSON := "null"
	if out != nil && out.Data != nil {
		if b, err := protojson.Marshal(out.Data); err == nil {
			dataJSON = string(b)
		}
	}
	comment := ""
	if out != nil && out.Comment != nil {
		comment = *out.Comment
	}

	row := types.NewRow(
		types.MRP("request_id", created.Id),
		types.MRP("data_json", string(dataJSON)),
		types.MRP("comment", comment),
	)
	return gp.AddRow(ctx, row)
}
