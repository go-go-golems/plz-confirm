package server

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type wsEvent struct {
	Type    string          `json:"type"`
	Request json.RawMessage `json:"request"`
}

func marshalWSEvent(eventType string, req proto.Message) ([]byte, error) {
	reqJSON, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   false, // use json_name (camelCase)
	}.Marshal(req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(wsEvent{
		Type:    eventType,
		Request: reqJSON,
	})
}


