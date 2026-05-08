#!/bin/bash

BASE_URL="http://localhost:8080"
APP_ID="myappid"
NUM_REQUESTS=100

ENDPOINTS=(
    "/tides/extremes"
    "/tides/timeline"
    "/data/reverse-geocode-client"
)

echo "Starting load generation: $NUM_REQUESTS requests..."

SUCCESS=0
FAILED=0

for i in $(seq 1 $NUM_REQUESTS); do
    # Generate random latitude (-90 to 90) and longitude (-180 to 180)
    LAT=$(python3 -c "import random; print(round(random.uniform(-90, 90), 4))")
    LNG=$(python3 -c "import random; print(round(random.uniform(-180, 180), 4))")
    
    # Pick a random endpoint
    ENDPOINT=${ENDPOINTS[$RANDOM % ${#ENDPOINTS[@]}]}
    
    URL="${BASE_URL}${ENDPOINT}?latitude=${LAT}&longitude=${LNG}"
    
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-App-Id: ${APP_ID}" "${URL}")
    
    if [ "$RESPONSE" == "200" ]; then
        ((SUCCESS++))
    else
        ((FAILED++))
        echo "Request $i failed: $ENDPOINT -> $RESPONSE"
    fi
    
    if (( i % 10 == 0 )); then
        echo "Progress: $i/$NUM_REQUESTS completed..."
    fi
done

echo ""
echo "Load Generation Complete!"
echo "Success: $SUCCESS"
echo "Failed:  $FAILED"
