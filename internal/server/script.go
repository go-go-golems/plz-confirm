package server

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-go-golems/plz-confirm/internal/scriptengine"
	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const scriptSeedStateKey = "__pc_seed"

func newScriptSeed() (int64, error) {
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & 0x7fffffffffffffff), nil
}

func stateSeedValue(state map[string]any) (int64, bool) {
	if state == nil {
		return 0, false
	}
	raw, ok := state[scriptSeedStateKey]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	default:
		return 0, false
	}
}

func ensureSeedInState(state map[string]any, seed int64) map[string]any {
	if state == nil {
		state = map[string]any{}
	}
	state[scriptSeedStateKey] = float64(seed)
	return state
}

func scriptInputWithSeed(in *v1.ScriptInput, seed int64) (*v1.ScriptInput, error) {
	if in == nil {
		return nil, fmt.Errorf("script input is required")
	}
	propsMap := map[string]any{}
	if in.GetProps() != nil {
		for k, v := range in.GetProps().AsMap() {
			propsMap[k] = v
		}
	}
	propsMap[scriptSeedStateKey] = float64(seed)
	propsStruct, err := structpb.NewStruct(propsMap)
	if err != nil {
		return nil, fmt.Errorf("script props: %w", err)
	}

	clone, ok := proto.Clone(in).(*v1.ScriptInput)
	if !ok {
		return nil, fmt.Errorf("failed to clone script input")
	}
	clone.Props = propsStruct
	return clone, nil
}

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
	seed, ok := stateSeedValue(state)
	if !ok {
		var err error
		seed, err = newScriptSeed()
		if err != nil {
			http.Error(w, "failed to allocate script seed", http.StatusInternalServerError)
			return
		}
	}
	state = ensureSeedInState(state, seed)
	eventMap := eventToMap(event)
	seededInput, err := scriptInputWithSeed(existingReq.GetScriptInput(), seed)
	if err != nil {
		http.Error(w, "invalid script input: "+err.Error(), http.StatusBadRequest)
		return
	}

	updateResult, err := s.scripts.UpdateAndView(r.Context(), seededInput, state, eventMap)
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
	updateResult.State = ensureSeedInState(updateResult.State, seed)

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
	inputMap := map[string]any{}
	hasTopLevelInput := false
	if raw, ok := m["input"]; ok {
		hasTopLevelInput = true
		typed, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("view.input must be object")
		}
		inputMap = typed
	}

	sections, parsedSections, err := mapToScriptViewSections(m["sections"])
	if err != nil {
		return nil, err
	}
	if len(parsedSections) > 0 {
		interactive, err := selectInteractiveSection(parsedSections)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(widgetType) == "" {
			widgetType = interactive.widgetType
		} else if !strings.EqualFold(strings.TrimSpace(widgetType), interactive.widgetType) {
			return nil, fmt.Errorf("view.widgetType must match the interactive section widgetType")
		}
		if !hasTopLevelInput {
			inputMap = interactive.input
		}
	}
	if strings.TrimSpace(widgetType) == "" {
		return nil, fmt.Errorf("view.widgetType is required")
	}
	if err := validateScriptViewInput(widgetType, inputMap); err != nil {
		return nil, err
	}
	inputStruct, err := mapToStruct(inputMap)
	if err != nil {
		return nil, err
	}
	view := &v1.ScriptView{
		WidgetType: widgetType,
		Input:      inputStruct,
		Sections:   sections,
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
	progress, err := mapToScriptProgress(m["progress"])
	if err != nil {
		return nil, err
	}
	if progress != nil {
		view.Progress = progress
	}
	if allowBack, ok := m["allowBack"].(bool); ok {
		view.AllowBack = &allowBack
	} else if showBack, ok := m["showBack"].(bool); ok {
		view.AllowBack = &showBack
	}
	if backLabel, ok := m["backLabel"].(string); ok && strings.TrimSpace(backLabel) != "" {
		view.BackLabel = &backLabel
	}
	toastCfg, err := mapToScriptToast(m["toast"])
	if err != nil {
		return nil, err
	}
	if toastCfg != nil {
		view.Toast = toastCfg
	}
	return view, nil
}

type parsedScriptViewSection struct {
	widgetType string
	input      map[string]any
}

func mapToScriptViewSections(raw any) ([]*v1.ScriptViewSection, []parsedScriptViewSection, error) {
	if raw == nil {
		return nil, nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("view.sections must be an array")
	}
	if len(items) == 0 {
		return nil, nil, fmt.Errorf("view.sections must include at least one section")
	}

	sections := make([]*v1.ScriptViewSection, 0, len(items))
	parsed := make([]parsedScriptViewSection, 0, len(items))
	for i, item := range items {
		sectionMap, ok := item.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("view.sections[%d] must be an object", i)
		}
		widgetType, _ := sectionMap["widgetType"].(string)
		if strings.TrimSpace(widgetType) == "" {
			return nil, nil, fmt.Errorf("view.sections[%d].widgetType is required", i)
		}

		inputMap := map[string]any{}
		if rawInput, ok := sectionMap["input"]; ok {
			typed, ok := rawInput.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("view.sections[%d].input must be object", i)
			}
			inputMap = typed
		}
		if err := validateScriptViewInput(widgetType, inputMap); err != nil {
			return nil, nil, err
		}
		inputStruct, err := mapToStruct(inputMap)
		if err != nil {
			return nil, nil, err
		}
		sections = append(sections, &v1.ScriptViewSection{
			WidgetType: widgetType,
			Input:      inputStruct,
		})
		parsed = append(parsed, parsedScriptViewSection{
			widgetType: strings.ToLower(strings.TrimSpace(widgetType)),
			input:      inputMap,
		})
	}
	return sections, parsed, nil
}

func selectInteractiveSection(sections []parsedScriptViewSection) (*parsedScriptViewSection, error) {
	interactiveCount := 0
	var interactive *parsedScriptViewSection
	for i := range sections {
		if sections[i].widgetType == "display" {
			continue
		}
		interactiveCount++
		interactive = &sections[i]
	}
	if interactiveCount != 1 {
		return nil, fmt.Errorf("view.sections must include exactly one interactive section")
	}
	return interactive, nil
}

func validateScriptViewInput(widgetType string, input map[string]any) error {
	switch strings.ToLower(strings.TrimSpace(widgetType)) {
	case "grid":
		return validateGridInput(input)
	case "display":
		return validateDisplayInput(input)
	case "rating":
		return validateRatingInput(input)
	default:
		return nil
	}
}

func validateDisplayInput(input map[string]any) error {
	content, _ := input["content"].(string)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("view.input.content is required for display widget")
	}
	if rawFormat, ok := input["format"]; ok {
		format, ok := rawFormat.(string)
		if !ok {
			return fmt.Errorf("view.input.format must be string for display widget")
		}
		switch strings.ToLower(strings.TrimSpace(format)) {
		case "", "markdown", "text", "html":
		default:
			return fmt.Errorf("view.input.format must be markdown, text, or html for display widget")
		}
	}
	return nil
}

func validateGridInput(input map[string]any) error {
	rows, ok := numberAsPositiveInt(input["rows"])
	if !ok {
		return fmt.Errorf("view.input.rows must be a positive integer for grid widget")
	}
	cols, ok := numberAsPositiveInt(input["cols"])
	if !ok {
		return fmt.Errorf("view.input.cols must be a positive integer for grid widget")
	}
	if rows*cols > 400 {
		return fmt.Errorf("view.input grid size exceeds max cells (400)")
	}

	cellsV, ok := input["cells"]
	if !ok {
		return fmt.Errorf("view.input.cells is required for grid widget")
	}
	cells, ok := cellsV.([]any)
	if !ok {
		return fmt.Errorf("view.input.cells must be an array for grid widget")
	}
	if len(cells) != rows*cols {
		return fmt.Errorf("view.input.cells length must equal rows*cols for grid widget")
	}
	for i, cellV := range cells {
		cell, ok := cellV.(map[string]any)
		if !ok {
			return fmt.Errorf("view.input.cells[%d] must be an object for grid widget", i)
		}
		if v, ok := cell["value"]; ok {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("view.input.cells[%d].value must be string", i)
			}
		}
		if v, ok := cell["style"]; ok {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("view.input.cells[%d].style must be string", i)
			}
		}
		if v, ok := cell["disabled"]; ok {
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("view.input.cells[%d].disabled must be boolean", i)
			}
		}
	}

	return nil
}

func validateRatingInput(input map[string]any) error {
	title, _ := input["title"].(string)
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("view.input.title is required for rating widget")
	}

	scale := 5
	if rawScale, ok := input["scale"]; ok {
		n, ok := numberAsInt(rawScale)
		if !ok {
			return fmt.Errorf("view.input.scale must be integer for rating widget")
		}
		if n < 2 || n > 10 {
			return fmt.Errorf("view.input.scale must be between 2 and 10 for rating widget")
		}
		scale = n
	}

	if rawStyle, ok := input["style"]; ok {
		style, ok := rawStyle.(string)
		if !ok {
			return fmt.Errorf("view.input.style must be string for rating widget")
		}
		switch strings.ToLower(strings.TrimSpace(style)) {
		case "", "stars", "numbers", "emoji", "slider":
		default:
			return fmt.Errorf("view.input.style must be stars, numbers, emoji, or slider for rating widget")
		}
	}

	if rawLabels, ok := input["labels"]; ok {
		labels, ok := rawLabels.(map[string]any)
		if !ok {
			return fmt.Errorf("view.input.labels must be object for rating widget")
		}
		if low, ok := labels["low"]; ok {
			if _, ok := low.(string); !ok {
				return fmt.Errorf("view.input.labels.low must be string for rating widget")
			}
		}
		if high, ok := labels["high"]; ok {
			if _, ok := high.(string); !ok {
				return fmt.Errorf("view.input.labels.high must be string for rating widget")
			}
		}
	}

	if rawDefault, ok := input["defaultValue"]; ok {
		n, ok := numberAsInt(rawDefault)
		if !ok {
			return fmt.Errorf("view.input.defaultValue must be integer for rating widget")
		}
		if n < 1 || n > scale {
			return fmt.Errorf("view.input.defaultValue must be between 1 and scale for rating widget")
		}
	}
	return nil
}

func numberAsPositiveInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, n > 0
	case int32:
		return int(n), n > 0
	case int64:
		return int(n), n > 0
	case float32:
		i := int(n)
		return i, float32(i) == n && i > 0
	case float64:
		i := int(n)
		return i, float64(i) == n && i > 0
	default:
		return 0, false
	}
}

func numberAsInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float32:
		i := int(n)
		return i, float32(i) == n
	case float64:
		i := int(n)
		return i, float64(i) == n
	default:
		return 0, false
	}
}

func mapToScriptProgress(raw any) (*v1.ScriptProgress, error) {
	if raw == nil {
		return nil, nil
	}
	progressMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("view.progress must be object")
	}
	current, ok := numberAsInt(progressMap["current"])
	if !ok {
		return nil, fmt.Errorf("view.progress.current is required and must be integer")
	}
	total, ok := numberAsInt(progressMap["total"])
	if !ok {
		return nil, fmt.Errorf("view.progress.total is required and must be integer")
	}
	if total <= 0 {
		return nil, fmt.Errorf("view.progress.total must be > 0")
	}
	if current < 0 {
		return nil, fmt.Errorf("view.progress.current must be >= 0")
	}
	if current > total {
		return nil, fmt.Errorf("view.progress.current must be <= total")
	}

	progress := &v1.ScriptProgress{
		Current: int32(current),
		Total:   int32(total),
	}
	if label, ok := progressMap["label"].(string); ok && strings.TrimSpace(label) != "" {
		progress.Label = &label
	}
	return progress, nil
}

func mapToScriptToast(raw any) (*v1.ScriptToast, error) {
	if raw == nil {
		return nil, nil
	}
	toastMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("view.toast must be object")
	}
	message, _ := toastMap["message"].(string)
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("view.toast.message is required")
	}

	toast := &v1.ScriptToast{
		Message: message,
	}
	if rawDuration, ok := toastMap["duration"]; ok {
		d, ok := numberAsInt(rawDuration)
		if !ok {
			return nil, fmt.Errorf("view.toast.duration must be integer")
		}
		if d <= 0 || d > 30000 {
			return nil, fmt.Errorf("view.toast.duration must be between 1 and 30000")
		}
		duration := int32(d)
		toast.DurationMs = &duration
	}
	if rawDuration, ok := toastMap["durationMs"]; ok {
		d, ok := numberAsInt(rawDuration)
		if !ok {
			return nil, fmt.Errorf("view.toast.durationMs must be integer")
		}
		if d <= 0 || d > 30000 {
			return nil, fmt.Errorf("view.toast.durationMs must be between 1 and 30000")
		}
		duration := int32(d)
		toast.DurationMs = &duration
	}
	if rawStyle, ok := toastMap["style"]; ok {
		style, ok := rawStyle.(string)
		if !ok {
			return nil, fmt.Errorf("view.toast.style must be string")
		}
		switch strings.ToLower(strings.TrimSpace(style)) {
		case "", "info", "success", "warning", "error":
		default:
			return nil, fmt.Errorf("view.toast.style must be info, success, warning, or error")
		}
		if strings.TrimSpace(style) != "" {
			toast.Style = &style
		}
	}
	return toast, nil
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
		strings.Contains(msg, "must include"),
		strings.Contains(msg, "must be object"),
		strings.Contains(msg, "invalid protojson"),
		strings.Contains(msg, "script source"):
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}
