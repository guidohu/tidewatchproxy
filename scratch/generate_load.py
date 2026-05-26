import requests
import random
import time

BASE_URL = "http://localhost:8080"
APP_ID = "myappid"
NUM_REQUESTS = 100

ENDPOINTS = [
    "/tides/extremes",
    "/tides/timeline",
    "/data/reverse-geocode",
    "/ping"
]

def generate_random_coords():
    # Random global coordinates
    lat = random.uniform(-90, 90)
    lng = random.uniform(-180, 180)
    return lat, lng

def run_load():
    headers = {
        "X-App-Id": APP_ID
    }
    
    print(f"Starting load generation: {NUM_REQUESTS} requests...")
    
    success_count = 0
    error_count = 0
    
    for i in range(NUM_REQUESTS):
        endpoint = random.choice(ENDPOINTS)
        url = f"{BASE_URL}{endpoint}"
        
        if endpoint == "/ping":
            uuid = f"uuid-{random.randint(1, 30)}"
            version = random.choice(["1.0.0", "1.0.1", "1.1.0", "2.0.0"])
            params = {
                "uuid": uuid,
                "version": version
            }
        else:
            lat, lng = generate_random_coords()
            params = {
                "latitude": lat,
                "longitude": lng
            }
        
        try:
            response = requests.get(url, headers=headers, params=params, timeout=5)
            if response.status_code == 200:
                success_count += 1
            else:
                error_count += 1
                print(f"Request {i+1} failed: {endpoint} -> {response.status_code}")
        except Exception as e:
            error_count += 1
            print(f"Request {i+1} error: {endpoint} -> {str(e)}")
            
        if (i + 1) % 10 == 0:
            print(f"Progress: {i+1}/{NUM_REQUESTS} requests completed...")
            
    print("\nLoad Generation Complete!")
    print(f"Success: {success_count}")
    print(f"Errors:  {error_count}")

if __name__ == "__main__":
    run_load()
