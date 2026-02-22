package server

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-go-golems/plz-confirm/internal/scriptengine"
	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) handleScriptEvent(w http.ResponseWriter, r *http.Request, id string) {
	existingReq, err := s.store.Get(r.Context(), id)
	if err != nil {
		if stderrors.Is(err, store.ErrNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existingReq.Type != v1.WidgetType_script {
		http.Error(w, "request is not script widget", http.StatusBadRequest)
		return
	}
	if existingReq.Status != v1.RequestStatus_pending {
		http.Error(w, "request already completed", http.StatusConflict)
		return
	}
	if existingReq.GetScriptInput() == nil {
		http.Error(w, "missing script input", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	event := &v1.ScriptEvent{}
	if err := protojson.Unmarshal(bodyBytes, event); err != nil {
		http.Error(w, "invalid protojson ScriptEvent: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(event.GetType()) == "" {
		http.Error(w, "script event type is required", http.StatusBadRequest)
		return
	}

	state := map[string]any{}
	if existingReq.GetScriptState() != nil {
		state = existingReq.GetScriptState().AsMap()
	}
	eventMap := eventToMap(event)

	updateResult, err := s.scripts.UpdateAndView(r.Context(), existingReq.GetScriptInput(), state, eventMap)
	if err != nil {
		http.Error(w, "script update failed: "+err.Error(), statusForScriptError(err))
		return
	}

	if updateResult.Done {
		resultStruct, err := mapToStruct(updateResult.Result)
		if err != nil {
			http.Error(w, "invalid script result: "+err.Error(), http.StatusBadRequest)
			return
		}
		outputReq := &v1.UIRequest{
			Type: v1.WidgetType_script,
			Output: &v1.UIRequest_ScriptOutput{
				ScriptOutput: &v1.ScriptOutput{
					Result: resultStruct,
					Logs:   updateResult.Logs,
				},
			},
		}
		req, err := s.store.Complete(r.Context(), id, outputReq)
		if err != nil {
			if stderrors.Is(err, store.ErrNotFound) {
				http.Error(w, "request not found", http.StatusNotFound)
				return
			}
			if stderrors.Is(err, store.ErrAlreadyCompleted) {
				http.Error(w, "request already completed", http.StatusConflict)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if msg, err := marshalWSEvent("request_completed", req); err == nil {
			s.ws.BroadcastRawJSON(req.SessionId, msg)
		}

		writeProtoJSON(w, http.StatusOK, req)
		return
	}

	stateStruct, viewProto, err := scriptUpdateResultToProto(updateResult)
	if err != nil {
		http.Error(w, "invalid script update result: "+err.Error(), http.StatusBadRequest)
		return
	}

	req, err := s.store.PatchScript(r.Context(), id, stateStruct, viewProto)
	if err != nil {
		if stderrors.Is(err, store.ErrNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		if stderrors.Is(err, store.ErrAlreadyCompleted) {
			http.Error(w, "request already completed", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if msg, err := marshalWSEvent("request_updated", req); err == nil {
		s.ws.BroadcastRawJSON(req.SessionId, msg)
	}

	writeProtoJSON(w, http.StatusOK, req)
}

func scriptInitResultToProto(res *scriptengine.InitAndViewResult) (*structpb.Struct, *v1.ScriptView, *v1.ScriptDescribe, error) {
	if res == nil {
		return nil, nil, nil, fmt.Errorf("script init result is nil")
	}
	stateStruct, err := mapToStruct(res.State)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("state: %w", err)
	}
	viewProto, err := mapToScriptView(res.View)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("view: %w", err)
	}
	describeProto, err := mapToScriptDescribe(res.Describe)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("describe: %w", err)
	}
	return stateStruct, viewProto, describeProto, nil
}

func scriptUpdateResultToProto(res *scriptengine.UpdateAndViewResult) (*structpb.Struct, *v1.ScriptView, error) {
	if res == nil {
		return nil, nil, fmt.Errorf("script update result is nil")
	}
	stateStruct, err := mapToStruct(res.State)
	if err != nil {
		return nil, nil, fmt.Errorf("state: %w", err)
	}
	viewProto, err := mapToScriptView(res.View)
	if err != nil {
		return nil, nil, fmt.Errorf("view: %w", err)
	}
	return stateStruct, viewProto, nil
}

func eventToMap(ev *v1.ScriptEvent) map[string]any {
	m := map[string]any{"type": ev.GetType()}
	if ev.GetStepId() != "" {
		m["stepId"] = ev.GetStepId()
	}
	if ev.GetActionId() != "" {
		m["actionId"] = ev.GetActionId()
	}
	if ev.GetData() != nil {
		m["data"] = ev.GetData().AsMap()
	}
	return m
}

func mapToStruct(m map[string]any) (*structpb.Struct, error) {
	if m == nil {
		m = map[string]any{}
	}
	return structpb.NewStruct(m)
}

func mapToScriptView(m map[string]any) (*v1.ScriptView, error) {
	if m == nil {
		return nil, fmt.Errorf("view must be object")
	}
	widgetType, _ := m["widgetType"].(string)
	if strings.TrimSpace(widgetType) == "" {
		return nil, fmt.Errorf("view.widgetType is required")
	}
	inputMap := map[string]any{}
	if raw, ok := m["input"]; ok {
		typed, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("view.input must be object")
		}
		inputMap = typed
	}
	inputStruct, err := mapToStruct(inputMap)
	if err != nil {
		return nil, err
	}
	view := &v1.ScriptView{
		WidgetType: widgetType,
		Input:      inputStruct,
	}
	if stepID, ok := m["stepId"].(string); ok && strings.TrimSpace(stepID) != "" {
		view.StepId = &stepID
	}
	if title, ok := m["title"].(string); ok && strings.TrimSpace(title) != "" {
		view.Title = &title
	}
	if description, ok := m["description"].(string); ok && strings.TrimSpace(description) != "" {
		view.Description = &description
	}
	return view, nil
}

func mapToScriptDescribe(m map[string]any) (*v1.ScriptDescribe, error) {
	if m == nil {
		return nil, fmt.Errorf("describe result must be object")
	}
	name, _ := m["name"].(string)
	version, _ := m["version"].(string)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("describe.name is required")
	}
	if strings.TrimSpace(version) == "" {
		return nil, fmt.Errorf("describe.version is required")
	}
	desc := &v1.ScriptDescribe{Name: name, Version: version}
	if apiVersion, ok := m["apiVersion"].(string); ok && strings.TrimSpace(apiVersion) != "" {
		desc.ApiVersion = &apiVersion
	}
	if caps, ok := m["capabilities"].([]any); ok {
		for _, capV := range caps {
			capStr, ok := capV.(string)
			if !ok || strings.TrimSpace(capStr) == "" {
				continue
			}
			desc.Capabilities = append(desc.Capabilities, capStr)
		}
	}
	return desc, nil
}

func statusForScriptError(err error) int {
	if err == nil {
		return http.StatusBadRequest
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}
	if stderrors.Is(err, context.Canceled) {
		return http.StatusRequestTimeout
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"):
		return http.StatusGatewayTimeout
	case strings.Contains(msg, "cancel"):
		return http.StatusRequestTimeout
	case strings.Contains(msg, "must export"),
		strings.Contains(msg, "is required"),
		strings.Contains(msg, "must be object"),
		strings.Contains(msg, "invalid protojson"),
		strings.Contains(msg, "script source"):
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}
