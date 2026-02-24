#!/usr/bin/env bash
set -euo pipefail

# Seed one request per widget type plus a multi-step JS script request,
# so operators can click through the full confirm-runtime UI surface.
#
# Defaults target the integrated inventory host mount.
#
# Usage:
#   BASE_URL="http://127.0.0.1:8091/confirm" \
#   SESSION_ID="global" \
#   bash ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/scripts/seed_clickthrough_all_widgets_with_js_script.sh

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"
TITLE_PREFIX="${TITLE_PREFIX:-C4}"
SCRIPT_SEED="${SCRIPT_SEED:-$(date +%s)}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required binary: $1" >&2
    exit 1
  }
}

require_bin curl
require_bin jq

probe_code="$(
  curl -sS -o /dev/null -w '%{http_code}' \
    -X POST "${BASE_URL}/api/requests" \
    -H 'content-type: application/json' \
    -d '{}'
)"
if [[ "$probe_code" == "000" ]]; then
  echo "ERROR: backend not reachable at ${BASE_URL}" >&2
  echo "Start go-inventory-chat/plz-confirm backend and retry." >&2
  exit 1
fi

post_request() {
  local payload="$1"
  curl -sS -X POST "${BASE_URL}/api/requests" \
    -H 'content-type: application/json' \
    -d "$payload"
}

enqueue_request() {
  local type="$1"
  local input_object_json="$2"
  local payload
  payload="$(jq -nc --arg type "$type" --arg session "$SESSION_ID" --argjson input "$input_object_json" '{type:$type, sessionId:$session} + $input')"

  local response
  response="$(post_request "$payload")"

  local id
  id="$(echo "$response" | jq -r '.id // empty')"
  if [[ -z "$id" ]]; then
    echo "ERROR: failed to create ${type} request" >&2
    echo "$response" | jq . >&2 || echo "$response" >&2
    exit 1
  fi

  local title
  title="$(echo "$response" | jq -r '.confirmInput.title // .selectInput.title // .formInput.title // .tableInput.title // .uploadInput.title // .imageInput.title // .scriptInput.title // "(no title)"')"

  printf '%-8s  %s  %s\n' "$type" "$id" "$title"
}

echo "Seeding click-through suite"
echo "BASE_URL=${BASE_URL}"
echo "SESSION_ID=${SESSION_ID}"
echo
printf '%-8s  %-36s  %s\n' "TYPE" "REQUEST_ID" "TITLE"
printf '%-8s  %-36s  %s\n' "--------" "------------------------------------" "-----------------------------"

confirm_input="$(jq -nc --arg title "${TITLE_PREFIX} confirm" --arg message "Deploy now?" '{confirmInput:{title:$title,message:$message}}')"
enqueue_request "confirm" "$confirm_input"

select_input="$(jq -nc --arg title "${TITLE_PREFIX} select" '{selectInput:{title:$title,options:["alpha","beta","gamma"],multi:true,searchable:true}}')"
enqueue_request "select" "$select_input"

form_input="$(jq -nc --arg title "${TITLE_PREFIX} form" '{formInput:{title:$title,schema:{type:"object",properties:{name:{type:"string"},count:{type:"number"},urgent:{type:"boolean"}},required:["name"]}}}')"
enqueue_request "form" "$form_input"

table_input="$(jq -nc --arg title "${TITLE_PREFIX} table" '{tableInput:{title:$title,columns:["id","env","status"],data:[{id:"srv-1",env:"staging",status:"ok"},{id:"srv-2",env:"prod",status:"degraded"}],multiSelect:true,searchable:true}}')"
enqueue_request "table" "$table_input"

upload_input="$(jq -nc --arg title "${TITLE_PREFIX} upload" '{uploadInput:{title:$title,accept:[".log","text/plain"],multiple:true,maxSize:"10485760"}}')"
enqueue_request "upload" "$upload_input"

image_input="$(jq -nc --arg title "${TITLE_PREFIX} image" '{imageInput:{title:$title,message:"Pick one",mode:"select",multi:false,options:[],images:[{label:"One",src:"https://placehold.co/320x180/png?text=One"},{label:"Two",src:"https://placehold.co/320x180/png?text=Two"}]}}')"
enqueue_request "image" "$image_input"

read -r -d '' SCRIPT_SOURCE <<'JS' || true
module.exports = {
  describe: function () {
    return { name: "c4-script", version: "1.0.0" };
  },
  init: function () {
    return { step: "confirm" };
  },
  view: function (state) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        stepId: "confirm",
        title: "C4 script",
        description: "Step 1: confirm before rating",
        input: { title: "Proceed?", message: "Script test" },
        progress: { current: 1, total: 2, label: "Step 1 of 2" }
      };
    }

    return {
      widgetType: "rating",
      stepId: "rate",
      title: "C4 script",
      description: "Step 2: rate confidence",
      allowBack: true,
      backLabel: "Back",
      sections: [
        {
          id: "context",
          kind: "display",
          widgetType: "display",
          input: {
            title: "Context",
            content: "You approved step 1. Now provide a confidence rating."
          }
        },
        {
          id: "rating",
          kind: "interactive",
          widgetType: "rating",
          input: { title: "Rate confidence", scale: 5, style: "stars" }
        }
      ],
      progress: { current: 2, total: 2, label: "Step 2 of 2" }
    };
  },
  update: function (state, event) {
    if (event && event.type === "back") {
      state.step = "confirm";
      return state;
    }

    if (state.step === "confirm") {
      if (event && event.type === "submit" && event.data && event.data.approved) {
        state.step = "rate";
        return state;
      }
      return { done: true, result: { approved: false, stage: "confirm" } };
    }

    if (state.step === "rate" && event && event.type === "submit") {
      return { done: true, result: { approved: true, rating: event.data ? event.data.value : null } };
    }

    return state;
  }
};
JS

script_input="$(jq -nc --arg title "${TITLE_PREFIX} script" --arg script "$SCRIPT_SOURCE" --argjson seed "$SCRIPT_SEED" '{scriptInput:{title:$title,script:$script,props:{__pc_seed:$seed}}}')"
enqueue_request "script" "$script_input"

echo
echo "Done. Open the Confirm Queue window in inventory and click through each request."
echo "Optional WS watcher (from plz-confirm repo root):"
echo "  go run ./cmd/plz-confirm ws --base-url \"${BASE_URL}\" --session-id \"${SESSION_ID}\" --pretty"
