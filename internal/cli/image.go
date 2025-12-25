package cli

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"

	"github.com/go-go-golems/plz-confirm/internal/client"
	agenttypes "github.com/go-go-golems/plz-confirm/internal/types"
)

type ImageCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = &ImageCommand{}

type ImageSettings struct {
	BaseURL     string `glazed.parameter:"base-url"`
	TimeoutS    int    `glazed.parameter:"timeout"`
	WaitTimeout int    `glazed.parameter:"wait-timeout"`

	Title   string  `glazed.parameter:"title"`
	Message *string `glazed.parameter:"message"`

	Mode string `glazed.parameter:"mode"` // select|confirm

	Images        []string `glazed.parameter:"image"`
	ImageLabels   []string `glazed.parameter:"image-label"`
	ImageAlts     []string `glazed.parameter:"image-alt"`
	ImageCaptions []string `glazed.parameter:"image-caption"`

	// Used for select mode:
	Multi   bool     `glazed.parameter:"multi"`
	Options []string `glazed.parameter:"option"`
}

func isLikelyBase64(s string) bool {
	if len(s) < 8 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '+' || r == '/' || r == '=':
		default:
			return false
		}
	}
	return true
}

// normalizeDataURIImages repairs a common CLI parsing gotcha:
//
// Glazed maps ParameterTypeStringList to Cobra/pflag's "stringSlice" flags, which parse
// comma-separated values. Unfortunately, data URIs are of the form:
//
//	data:image/png;base64,<payload>
//
// and will be split at the comma into two list entries. That breaks per-image metadata
// count validation (`--image-label`, `--image-alt`, `--image-caption`) and results in
// confusing errors for users.
//
// We detect and re-join the split pair when it looks like a base64 data URI.
func normalizeDataURIImages(images []string) []string {
	out := make([]string, 0, len(images))
	for i := 0; i < len(images); i++ {
		cur := images[i]

		// If the user passed a data URI, pflag may have split it into:
		//   ["data:image/png;base64", "<payload>"]
		//
		// Only re-join when:
		// - current starts with "data:"
		// - current does NOT already contain a comma
		// - it includes ";base64" (we only try to fix the image/base64 case)
		// - next token looks like base64 payload
		if strings.HasPrefix(cur, "data:") &&
			!strings.Contains(cur, ",") &&
			strings.Contains(cur, ";base64") &&
			i+1 < len(images) &&
			isLikelyBase64(images[i+1]) {
			out = append(out, cur+","+images[i+1])
			i++
			continue
		}

		out = append(out, cur)
	}
	return out
}

func NewImageCommand(layersList ...layers.ParameterLayer) (*ImageCommand, error) {
	desc := cmds.NewCommandDescription(
		"image",
		cmds.WithShort("Request an image-based selection/confirmation via the agent-ui web frontend"),
		cmds.WithLong("Creates an image widget request (with one or more images), waits for the user response, and outputs the result."),
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
				"message",
				parameters.ParameterTypeString,
				parameters.WithHelp("Optional dialog message / question"),
			),
			parameters.NewParameterDefinition(
				"mode",
				parameters.ParameterTypeString,
				parameters.WithDefault("select"),
				parameters.WithHelp("Widget mode: select|confirm"),
			),
			parameters.NewParameterDefinition(
				"image",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Image source (repeatable): local file path, URL, or data:image/... URI"),
				parameters.WithRequired(true),
			),
			parameters.NewParameterDefinition(
				"image-label",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Optional per-image label (repeatable; must match number of --image entries if provided)"),
			),
			parameters.NewParameterDefinition(
				"image-alt",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Optional per-image alt text (repeatable; must match number of --image entries if provided)"),
			),
			parameters.NewParameterDefinition(
				"image-caption",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Optional per-image caption (repeatable; must match number of --image entries if provided)"),
			),
			parameters.NewParameterDefinition(
				"option",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Option value (repeatable). Used for the \"images-as-context + multi-select question\" variant."),
			),
			parameters.NewParameterDefinition(
				"multi",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Allow selecting multiple options / multiple images (select mode)"),
			),
		),
		cmds.WithLayersList(layersList...),
	)
	return &ImageCommand{CommandDescription: desc}, nil
}

func (c *ImageCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	settings := &ImageSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings); err != nil {
		return err
	}

	// Repair comma-splitting for base64 data URIs in --image.
	settings.Images = normalizeDataURIImages(settings.Images)

	if len(settings.ImageLabels) > 0 && len(settings.ImageLabels) != len(settings.Images) {
		return errors.Errorf("--image-label count (%d) must match --image count (%d)", len(settings.ImageLabels), len(settings.Images))
	}
	if len(settings.ImageAlts) > 0 && len(settings.ImageAlts) != len(settings.Images) {
		return errors.Errorf("--image-alt count (%d) must match --image count (%d)", len(settings.ImageAlts), len(settings.Images))
	}
	if len(settings.ImageCaptions) > 0 && len(settings.ImageCaptions) != len(settings.Images) {
		return errors.Errorf("--image-caption count (%d) must match --image count (%d)", len(settings.ImageCaptions), len(settings.Images))
	}

	cl := client.New(settings.BaseURL)

	ttl := settings.TimeoutS
	if ttl <= 0 {
		ttl = 300
	}

	images := make([]agenttypes.ImageItem, 0, len(settings.Images))
	for i, raw := range settings.Images {
		src := raw
		// Treat non-URL / non-data URIs as local paths to upload.
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") && !strings.HasPrefix(raw, "data:") {
			up, err := cl.UploadImage(ctx, raw, ttl)
			if err != nil {
				return errors.Wrapf(err, "upload image %q", raw)
			}
			src = up.URL
		}

		var label *string
		if len(settings.ImageLabels) == len(settings.Images) {
			label = &settings.ImageLabels[i]
		}
		var alt *string
		if len(settings.ImageAlts) == len(settings.Images) {
			alt = &settings.ImageAlts[i]
		}
		var caption *string
		if len(settings.ImageCaptions) == len(settings.Images) {
			caption = &settings.ImageCaptions[i]
		}

		images = append(images, agenttypes.ImageItem{
			Src:     src,
			Alt:     alt,
			Label:   label,
			Caption: caption,
		})
	}

	input := agenttypes.ImageInput{
		Title:   settings.Title,
		Message: settings.Message,
		Images:  images,
		Mode:    settings.Mode,
		Options: settings.Options,
		Multi:   &settings.Multi,
	}

	created, err := cl.CreateRequest(ctx, client.CreateRequestParams{
		Type:      agenttypes.WidgetImage,
		SessionID: "global", // ignored by server; kept for compatibility
		Input:     input,
		TimeoutS:  settings.TimeoutS,
	})
	if err != nil {
		return errors.Wrap(err, "create image request")
	}

	completed, err := cl.WaitRequest(ctx, created.ID, settings.WaitTimeout)
	if err != nil {
		return errors.Wrap(err, "wait for image response")
	}

	var out agenttypes.ImageOutput
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
		types.MRP("timestamp", out.Timestamp),
	)
	return gp.AddRow(ctx, row)
}
