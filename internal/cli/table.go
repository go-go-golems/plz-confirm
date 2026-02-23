package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/client"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type TableCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &TableCommand{}

type TableSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	SessionID   string `glazed.parameter:"session-id"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title       string   `glazed.parameter:"title"`
	Data        string   `glazed.parameter:"data"`
	Columns     []string `glazed.parameter:"columns"`
	MultiSelect bool     `glazed.parameter:"multi-select"`
	Searchable  bool     `glazed.parameter:"searchable"`
}

func NewTableCommand() (*TableCommand, error) {
	desc := cmds.NewCommandDescription(
		"table",
		cmds.WithShort("Request table selection via the agent-ui web frontend"),
		cmds.WithLong("Creates a table widget request with rows, waits for the user selection, and outputs the result."),
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
				"data",
				fields.TypeString,
				fields.WithHelp("Path to JSON file with array of row objects (use @file.json or - for stdin)"),
				fields.WithRequired(true),
			),
			fields.New(
				"columns",
				fields.TypeStringList,
				fields.WithHelp("Optional column names (auto-derived if omitted)"),
			),
			fields.New(
				"multi-select",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Allow selecting multiple rows"),
			),
			fields.New(
				"searchable",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Enable search/filter box in the UI"),
			),
		),
	)

	return &TableCommand{CommandDescription: desc}, nil
}

func (c *TableCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	settings := &TableSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
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
		defer func() {
			_ = f.Close()
		}()
		dataReader = f
	}

	var rows []any
	if err := json.NewDecoder(dataReader).Decode(&rows); err != nil {
		return errors.Wrap(err, "decode data JSON")
	}

	pbRows := make([]*structpb.Struct, 0, len(rows))
	for i, row := range rows {
		rowBytes, err := json.Marshal(row)
		if err != nil {
			return errors.Wrapf(err, "marshal row %d", i)
		}
		st := &structpb.Struct{}
		if err := protojson.Unmarshal(rowBytes, st); err != nil {
			return errors.Wrapf(err, "protojson unmarshal row %d", i)
		}
		pbRows = append(pbRows, st)
	}

	cl := client.New(settings.BaseURL)
	input := &v1.TableInput{
		Title:       settings.Title,
		Data:        pbRows,
		Columns:     settings.Columns,
		MultiSelect: &settings.MultiSelect,
		Searchable:  &settings.Searchable,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      v1.WidgetType_table,
		SessionID: settings.SessionID,
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create table request")
	}

	completed, err := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for table response")
	}

	if completed.Status != v1.RequestStatus_completed {
		return errors.Errorf("request %s ended with status=%s", created.Id, completed.Status.String())
	}

	out := completed.GetTableOutput()

	var selectedAny any
	comment := ""
	if out != nil {
		switch sel := out.Selected.(type) {
		case *v1.TableOutput_SelectedSingle:
			if sel.SelectedSingle != nil {
				selectedAny = sel.SelectedSingle.AsMap()
			}
		case *v1.TableOutput_SelectedMulti:
			if sel.SelectedMulti != nil {
				arr := make([]any, 0, len(sel.SelectedMulti.Values))
				for _, s := range sel.SelectedMulti.Values {
					if s != nil {
						arr = append(arr, s.AsMap())
					}
				}
				selectedAny = arr
			}
		default:
			selectedAny = nil
		}
		if out.Comment != nil {
			comment = *out.Comment
		}
	}

	selectedJSON, _ := json.Marshal(selectedAny)

	row := types.NewRow(
		types.MRP("request_id", created.Id),
		types.MRP("selected_json", string(selectedJSON)),
		types.MRP("comment", comment),
	)
	return gp.AddRow(ctx, row)
}
