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

type UploadCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &UploadCommand{}

type UploadSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	SessionID   string `glazed.parameter:"session-id"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title       string   `glazed.parameter:"title"`
	Accept      []string `glazed.parameter:"accept"`
	Multiple    bool     `glazed.parameter:"multiple"`
	MaxSize     *int64   `glazed.parameter:"max-size"`
	CallbackURL *string  `glazed.parameter:"callback-url"`
}

func NewUploadCommand() (*UploadCommand, error) {
	desc := cmds.NewCommandDescription(
		"upload",
		cmds.WithShort("Request file upload via the agent-ui web frontend"),
		cmds.WithLong("Creates an upload widget request, waits for the user to upload files, and outputs the result."),
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
				"accept",
				fields.TypeStringList,
				fields.WithHelp("File extensions or MIME types (e.g., .log, .txt, image/png)"),
			),
			fields.New(
				"multiple",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Allow uploading multiple files"),
			),
			fields.New(
				"max-size",
				fields.TypeInteger,
				fields.WithHelp("Maximum file size in bytes"),
			),
			fields.New(
				"callback-url",
				fields.TypeString,
				fields.WithHelp("Optional callback URL (not implemented)"),
			),
		),
	)

	return &UploadCommand{CommandDescription: desc}, nil
}

func (c *UploadCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	settings := &UploadSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}

	cl := client.New(settings.BaseURL)
	input := &v1.UploadInput{
		Title:       settings.Title,
		Accept:      settings.Accept,
		Multiple:    &settings.Multiple,
		MaxSize:     settings.MaxSize,
		CallbackUrl: settings.CallbackURL,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      v1.WidgetType_upload,
		SessionID: settings.SessionID,
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create upload request")
	}

	completed, err := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for upload response")
	}

	if completed.Status != v1.RequestStatus_completed {
		return errors.Errorf("request %s ended with status=%s", created.Id, completed.Status.String())
	}

	out := completed.GetUploadOutput()

	comment := ""
	if out != nil && out.Comment != nil {
		comment = *out.Comment
	}

	// Output files as rows (one per file)
	files := out.GetFiles()
	for _, file := range files {
		row := types.NewRow(
			types.MRP("request_id", created.Id),
			types.MRP("file_name", file.GetName()),
			types.MRP("file_size", file.GetSize()),
			types.MRP("file_path", file.GetPath()),
			types.MRP("mime_type", file.GetMimeType()),
			types.MRP("comment", comment),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}

	// If no files, still output a row with request_id
	if len(files) == 0 {
		row := types.NewRow(
			types.MRP("request_id", created.Id),
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
