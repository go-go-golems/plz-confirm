package server

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var errInvalidType = errors.New("invalid widget type")

// widgetTypeStringToProto converts string widget type to protobuf WidgetType
func widgetTypeStringToProto(wt string) v1.WidgetType {
	switch wt {
	case "confirm":
		return v1.WidgetType_WIDGET_TYPE_CONFIRM
	case "select":
		return v1.WidgetType_WIDGET_TYPE_SELECT
	case "form":
		return v1.WidgetType_WIDGET_TYPE_FORM
	case "upload":
		return v1.WidgetType_WIDGET_TYPE_UPLOAD
	case "table":
		return v1.WidgetType_WIDGET_TYPE_TABLE
	case "image":
		return v1.WidgetType_WIDGET_TYPE_IMAGE
	default:
		return v1.WidgetType_WIDGET_TYPE_UNSPECIFIED
	}
}

// convertJSONInputToProto converts JSON input (any) to protobuf UIRequest input oneof
func convertJSONInputToProto(widgetType v1.WidgetType, inputJSON any) (proto.Message, error) {
	jsonBytes, err := json.Marshal(inputJSON)
	if err != nil {
		return nil, err
	}

	switch widgetType {
	case v1.WidgetType_WIDGET_TYPE_CONFIRM:
		var input v1.ConfirmInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case v1.WidgetType_WIDGET_TYPE_SELECT:
		var input v1.SelectInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case v1.WidgetType_WIDGET_TYPE_FORM:
		var input v1.FormInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case v1.WidgetType_WIDGET_TYPE_UPLOAD:
		var input v1.UploadInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case v1.WidgetType_WIDGET_TYPE_TABLE:
		var input v1.TableInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case v1.WidgetType_WIDGET_TYPE_IMAGE:
		var input v1.ImageInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	default:
		return nil, errInvalidType
	}
}

// convertJSONOutputToProto converts JSON output (any) to protobuf UIRequest output oneof
func convertJSONOutputToProto(widgetType v1.WidgetType, outputJSON any) (proto.Message, error) {
	jsonBytes, err := json.Marshal(outputJSON)
	if err != nil {
		return nil, err
	}

	switch widgetType {
	case v1.WidgetType_WIDGET_TYPE_CONFIRM:
		var output v1.ConfirmOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case v1.WidgetType_WIDGET_TYPE_SELECT:
		var output v1.SelectOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case v1.WidgetType_WIDGET_TYPE_FORM:
		var output v1.FormOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case v1.WidgetType_WIDGET_TYPE_UPLOAD:
		var output v1.UploadOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case v1.WidgetType_WIDGET_TYPE_TABLE:
		var output v1.TableOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case v1.WidgetType_WIDGET_TYPE_IMAGE:
		var output v1.ImageOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	default:
		return nil, errInvalidType
	}
}

// createUIRequestFromJSON creates a *v1.UIRequest from JSON createRequestBody
func createUIRequestFromJSON(typeStr string, sessionID string, inputJSON any, timeoutS int) (*v1.UIRequest, error) {
	widgetType := widgetTypeStringToProto(typeStr)
	if widgetType == v1.WidgetType_WIDGET_TYPE_UNSPECIFIED {
		return nil, errInvalidType
	}

	inputProto, err := convertJSONInputToProto(widgetType, inputJSON)
	if err != nil {
		return nil, err
	}

	req := &v1.UIRequest{
		Type:      widgetType,
		SessionId: sessionID,
	}

	// We keep JSON wire format for REST, so server accepts "timeout" seconds like before.
	// Store.Create uses ExpiresAt as a hint; if empty it defaults to 300s.
	if timeoutS > 0 {
		now := time.Now().UTC()
		req.ExpiresAt = now.Add(time.Duration(timeoutS) * time.Second).Format(time.RFC3339Nano)
	}

	// Set the appropriate oneof field
	switch input := inputProto.(type) {
	case *v1.ConfirmInput:
		req.Input = &v1.UIRequest_ConfirmInput{ConfirmInput: input}
	case *v1.SelectInput:
		req.Input = &v1.UIRequest_SelectInput{SelectInput: input}
	case *v1.FormInput:
		req.Input = &v1.UIRequest_FormInput{FormInput: input}
	case *v1.UploadInput:
		req.Input = &v1.UIRequest_UploadInput{UploadInput: input}
	case *v1.TableInput:
		req.Input = &v1.UIRequest_TableInput{TableInput: input}
	case *v1.ImageInput:
		req.Input = &v1.UIRequest_ImageInput{ImageInput: input}
	default:
		return nil, errInvalidType
	}

	return req, nil
}

// createUIRequestWithOutput creates a *v1.UIRequest with output oneof populated
func createUIRequestWithOutput(widgetType v1.WidgetType, outputJSON any) (*v1.UIRequest, error) {
	outputProto, err := convertJSONOutputToProto(widgetType, outputJSON)
	if err != nil {
		return nil, err
	}

	req := &v1.UIRequest{
		Type: widgetType,
	}

	// Set the appropriate oneof field
	switch output := outputProto.(type) {
	case *v1.ConfirmOutput:
		req.Output = &v1.UIRequest_ConfirmOutput{ConfirmOutput: output}
	case *v1.SelectOutput:
		req.Output = &v1.UIRequest_SelectOutput{SelectOutput: output}
	case *v1.FormOutput:
		req.Output = &v1.UIRequest_FormOutput{FormOutput: output}
	case *v1.UploadOutput:
		req.Output = &v1.UIRequest_UploadOutput{UploadOutput: output}
	case *v1.TableOutput:
		req.Output = &v1.UIRequest_TableOutput{TableOutput: output}
	case *v1.ImageOutput:
		req.Output = &v1.UIRequest_ImageOutput{ImageOutput: output}
	default:
		return nil, errInvalidType
	}

	return req, nil
}
