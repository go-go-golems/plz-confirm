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

type UploadCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &UploadCommand{}

type UploadSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title       string   `glazed.parameter:"title"`
	Accept      []string `glazed.parameter:"accept"`
	Multiple    bool     `glazed.parameter:"multiple"`
	MaxSize     *int64   `glazed.parameter:"max-size"`
	CallbackURL *string  `glazed.parameter:"callback-url"`
}

func NewUploadCommand(layersList ...layers.ParameterLayer) (*UploadCommand, error) {
	desc := cmds.NewCommandDescription(
		"upload",
		cmds.WithShort("Request file upload via the agent-ui web frontend"),
		cmds.WithLong("Creates an upload widget request, waits for the user to upload files, and outputs the result."),
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
				"accept",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("File extensions or MIME types (e.g., .log, .txt, image/png)"),
			),
			parameters.NewParameterDefinition(
				"multiple",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Allow uploading multiple files"),
			),
			parameters.NewParameterDefinition(
				"max-size",
				parameters.ParameterTypeInteger,
				parameters.WithHelp("Maximum file size in bytes"),
			),
			parameters.NewParameterDefinition(
				"callback-url",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional callback URL (not implemented)"),
			),
		),
		cmds.WithLayersList(layersList...),
	)

	return &UploadCommand{CommandDescription: desc}, nil
}

func (c *UploadCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &UploadSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return err
	}

	cl := client.New(settings.BaseURL)
	input := agenttypes.UploadInput{
		Title:       settings.Title,
		Accept:      settings.Accept,
		Multiple:    &settings.Multiple,
		MaxSize:     settings.MaxSize,
		CallbackURL: settings.CallbackURL,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      agenttypes.WidgetUpload,
		SessionID: "global", // ignored by server; kept for compatibility
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create upload request")
	}

	completed, err := cl.WaitRequest(ctx, created.ID, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for upload response")
	}

	var out agenttypes.UploadOutput
	if completed.Output != nil {
		b, err := json.Marshal(completed.Output)
		if err != nil {
			return errors.Wrap(err, "marshal output")
		}
		if err := json.Unmarshal(b, &out); err != nil {
			return errors.Wrap(err, "unmarshal output")
		}
	}

	comment := ""
	if out.Comment != nil {
		comment = *out.Comment
	}

	// Output files as rows (one per file)
	for _, file := range out.Files {
		row := types.NewRow(
			types.MRP("request_id", created.ID),
			types.MRP("file_name", file.Name),
			types.MRP("file_size", file.Size),
			types.MRP("file_path", file.Path),
			types.MRP("mime_type", file.MimeType),
			types.MRP("comment", comment),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	// If no files, still output a row with request_id
	if len(out.Files) == 0 {
		row := types.NewRow(
			types.MRP("request_id", created.ID),
			types.MRP("file_name", ""),
			types.MRP("file_size", int64(0)),
			types.MRP("file_path", ""),
			types.MRP("mime_type", ""),
			types.MRP("comment", comment),
		)
		return gp.AddRow(ctx, row)
	}

	return nil
}
