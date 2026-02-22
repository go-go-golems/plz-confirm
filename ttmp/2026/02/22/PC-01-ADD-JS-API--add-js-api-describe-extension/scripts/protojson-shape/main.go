package main

import (
	"fmt"

	v1 "github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	create := &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "global",
		Input: &v1.UIRequest_ConfirmInput{
			ConfirmInput: &v1.ConfirmInput{
				Title: "Deploy?",
			},
		},
	}

	createJSON, err := protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true}.Marshal(create)
	if err != nil {
		panic(err)
	}
	fmt.Println("create payload:")
	fmt.Println(string(createJSON))

	response := &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "global",
		Output: &v1.UIRequest_ConfirmOutput{
			ConfirmOutput: &v1.ConfirmOutput{Approved: true, Timestamp: "2026-02-22T20:00:00Z"},
		},
	}
	responseJSON, err := protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true}.Marshal(response)
	if err != nil {
		panic(err)
	}
	fmt.Println("response payload:")
	fmt.Println(string(responseJSON))
}
