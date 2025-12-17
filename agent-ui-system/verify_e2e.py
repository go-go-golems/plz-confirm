import subprocess
import time
import requests
import re
import sys
import threading

API_URL = "http://localhost:3000/api"

def simulate_user_interaction(process):
    print("[Test] User simulator started")
    
    # Keep track of handled requests to avoid double-responding
    handled_requests = set()

    while True:
        # Read line from process stdout
        line = process.stdout.readline()
        if not line:
            break
        
        line = line.decode('utf-8').strip()
        print(f"[CLI Output] {line}")
        
        # Detect request creation
        match = re.search(r"Request created: ([a-zA-Z0-9_-]+)", line)
        if match:
            req_id = match.group(1)
            if req_id in handled_requests:
                continue
                
            handled_requests.add(req_id)
            print(f"[Test] Detected request {req_id}. Simulating user interaction in 2 seconds...")
            
            # Wait a bit to simulate user reading the UI
            time.sleep(2)
            
            # Determine response based on request type (we can fetch the request to check type)
            try:
                req_details = requests.get(f"{API_URL}/requests/{req_id}").json()
                req_type = req_details.get("type")
                
                response_data = {}
                if req_type == "confirm":
                    response_data = {"approved": True, "timestamp": "2023-10-27T10:00:00Z"}
                    print(f"[Test] Clicking 'Approve' for request {req_id}")
                elif req_type == "select":
                    response_data = {"selected": "us-west-2"}
                    print(f"[Test] Selecting 'us-west-2' for request {req_id}")
                elif req_type == "form":
                    response_data = {"data": {"username": "admin_user", "email": "admin@example.com", "accessLevel": 5}}
                    print(f"[Test] Filling form for request {req_id}")
                
                # Submit response
                requests.post(f"{API_URL}/requests/{req_id}/response", json={"output": response_data})
                print(f"[Test] Response submitted for {req_id}")
                
            except Exception as e:
                print(f"[Test] Error simulating interaction: {e}")

def main():
    print("=== Starting E2E Verification ===")
    
    # Start the CLI demo script
    process = subprocess.Popen(
        ["python3", "demo_cli.py"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        bufsize=1
    )
    
    # Start interaction thread
    interaction_thread = threading.Thread(target=simulate_user_interaction, args=(process,))
    interaction_thread.start()
    
    # Wait for process to finish
    process.wait()
    interaction_thread.join()
    
    print("=== E2E Verification Completed ===")

if __name__ == "__main__":
    main()
