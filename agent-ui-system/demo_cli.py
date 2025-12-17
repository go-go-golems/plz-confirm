import requests
import time
import uuid
import json
import sys

# Configuration
API_URL = "http://localhost:3000/api"
SESSION_ID = "550e8400-e29b-41d4-a716-446655440000" # Matches the frontend default

def print_header(text):
    print(f"\n\033[1;32m=== {text} ===\033[0m")

def create_request(req_type, input_data):
    print(f"\n[CLI] Sending {req_type} request...")
    try:
        response = requests.post(f"{API_URL}/requests", json={
            "type": req_type,
            "sessionId": SESSION_ID,
            "input": input_data,
            "timeout": 300
        })
        response.raise_for_status()
        req_data = response.json()
        req_id = req_data["id"]
        print(f"[CLI] Request created: {req_id}")
        print(f"[CLI] Waiting for user interaction on the web UI...")
        return req_id
    except requests.exceptions.RequestException as e:
        print(f"\033[1;31m[Error] Failed to create request: {e}\033[0m")
        sys.exit(1)

def wait_for_response(req_id):
    try:
        # Long poll for 60 seconds
        response = requests.get(f"{API_URL}/requests/{req_id}/wait?timeout=60")
        if response.status_code == 408:
            print("\033[1;33m[CLI] Timeout waiting for response.\033[0m")
            return None
        
        response.raise_for_status()
        data = response.json()
        print(f"\033[1;36m[CLI] Response received!\033[0m")
        print(json.dumps(data.get("output"), indent=2))
        return data.get("output")
    except requests.exceptions.RequestException as e:
        print(f"\033[1;31m[Error] Failed to get response: {e}\033[0m")
        return None

def main():
    print_header("Agent UI System - CLI Demo")
    print(f"Session ID: {SESSION_ID}")
    print("Make sure the web UI is open at http://localhost:3000")
    
    # 1. Confirm Dialog
    print_header("Step 1: Confirmation")
    req_id = create_request("confirm", {
        "title": "System Update Required",
        "message": "A critical security patch (v2.4.0) is available. Do you want to install it now? This will require a restart.",
        "approveText": "Install & Restart",
        "rejectText": "Remind Me Later"
    })
    result = wait_for_response(req_id)
    
    if not result or not result.get("approved"):
        print("\n[CLI] Update cancelled by user. Exiting demo.")
        return

    print("\n[CLI] User approved update. Proceeding...")
    time.sleep(1)

    # 2. Select Dialog
    print_header("Step 2: Configuration")
    req_id = create_request("select", {
        "title": "Select Region",
        "options": ["us-east-1", "us-west-2", "eu-central-1", "ap-northeast-1"],
        "multi": False,
        "searchable": True
    })
    result = wait_for_response(req_id)
    
    if not result: return
    region = result.get("selected")
    print(f"\n[CLI] Selected region: {region}")
    time.sleep(1)

    # 3. Form Dialog
    print_header("Step 3: User Details")
    req_id = create_request("form", {
        "title": "Administrator Details",
        "schema": {
            "properties": {
                "username": {"type": "string", "minLength": 3},
                "email": {"type": "string", "format": "email"},
                "accessLevel": {"type": "number", "minimum": 1, "maximum": 5}
            },
            "required": ["username", "email"]
        }
    })
    result = wait_for_response(req_id)
    
    if not result: return
    print(f"\n[CLI] Admin configured: {result.get('data')}")
    
    print_header("Demo Completed Successfully")

if __name__ == "__main__":
    main()
