package store

import (
	"encoding/json"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// widgetTypeToProto converts internal/types.WidgetType to protobuf WidgetType
func widgetTypeToProto(wt string) v1.WidgetType {
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

// widgetTypeFromProto converts protobuf WidgetType to string
func widgetTypeFromProto(wt v1.WidgetType) string {
	switch wt {
	case v1.WidgetType_WIDGET_TYPE_CONFIRM:
		return "confirm"
	case v1.WidgetType_WIDGET_TYPE_SELECT:
		return "select"
	case v1.WidgetType_WIDGET_TYPE_FORM:
		return "form"
	case v1.WidgetType_WIDGET_TYPE_UPLOAD:
		return "upload"
	case v1.WidgetType_WIDGET_TYPE_TABLE:
		return "table"
	case v1.WidgetType_WIDGET_TYPE_IMAGE:
		return "image"
	default:
		return ""
	}
}

// requestStatusToProto converts string status to protobuf RequestStatus
func requestStatusToProto(status string) v1.RequestStatus {
	switch status {
	case "pending":
		return v1.RequestStatus_REQUEST_STATUS_PENDING
	case "completed":
		return v1.RequestStatus_REQUEST_STATUS_COMPLETED
	case "timeout":
		return v1.RequestStatus_REQUEST_STATUS_TIMEOUT
	case "error":
		return v1.RequestStatus_REQUEST_STATUS_ERROR
	default:
		return v1.RequestStatus_REQUEST_STATUS_UNSPECIFIED
	}
}

// requestStatusFromProto converts protobuf RequestStatus to string
func requestStatusFromProto(status v1.RequestStatus) string {
	switch status {
	case v1.RequestStatus_REQUEST_STATUS_PENDING:
		return "pending"
	case v1.RequestStatus_REQUEST_STATUS_COMPLETED:
		return "completed"
	case v1.RequestStatus_REQUEST_STATUS_TIMEOUT:
		return "timeout"
	case v1.RequestStatus_REQUEST_STATUS_ERROR:
		return "error"
	default:
		return ""
	}
}

// convertInputToProto converts JSON input (any) to protobuf UIRequest input oneof
// based on widget type
func convertInputToProto(widgetType string, inputJSON any) (proto.Message, error) {
	// Marshal inputJSON to JSON bytes
	jsonBytes, err := json.Marshal(inputJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal based on widget type
	switch widgetType {
	case "confirm":
		var input v1.ConfirmInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case "select":
		var input v1.SelectInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case "form":
		var input v1.FormInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case "upload":
		var input v1.UploadInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case "table":
		var input v1.TableInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	case "image":
		var input v1.ImageInput
		if err := protojson.Unmarshal(jsonBytes, &input); err != nil {
			return nil, err
		}
		return &input, nil
	default:
		return nil, json.Unmarshal(jsonBytes, &structpb.Struct{})
	}
}

// convertOutputToProto converts JSON output (any) to protobuf UIRequest output oneof
// based on widget type
func convertOutputToProto(widgetType string, outputJSON any) (proto.Message, error) {
	jsonBytes, err := json.Marshal(outputJSON)
	if err != nil {
		return nil, err
	}

	switch widgetType {
	case "confirm":
		var output v1.ConfirmOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case "select":
		var output v1.SelectOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case "form":
		var output v1.FormOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case "upload":
		var output v1.UploadOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case "table":
		var output v1.TableOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	case "image":
		var output v1.ImageOutput
		if err := protojson.Unmarshal(jsonBytes, &output); err != nil {
			return nil, err
		}
		return &output, nil
	default:
		return nil, json.Unmarshal(jsonBytes, &structpb.Struct{})
	}
}

// protoToJSON converts a protobuf message to JSON (any)
func protoToJSON(msg proto.Message) (any, error) {
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var result any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

