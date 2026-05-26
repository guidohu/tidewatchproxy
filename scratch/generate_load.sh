#!/bin/bash

BASE_URL="http://localhost:8080"
APP_ID="myappid"
NUM_REQUESTS=1
CONCURRENCY=10

ENDPOINTS=(
    "/tides/extremes"
    "/tides/timeline"
    "/data/reverse-geocode"
    "/ping"
)

echo "Starting optimized load generation: $NUM_REQUESTS requests with concurrency $CONCURRENCY..."

# Function to generate a random coordinate or ping
gen_url() {
    local endpoint=${ENDPOINTS[$RANDOM % ${#ENDPOINTS[@]}]}
    if [ "$endpoint" = "/ping" ]; then
        local uuid_num=$(( RANDOM % 30 + 1 ))
        local versions=("1.0.0" "1.0.1" "1.1.0" "2.0.0")
        local ver=${versions[$RANDOM % 4]}
        echo "${BASE_URL}${endpoint}?uuid=uuid-${uuid_num}&version=${ver}"
    else
        local lat_raw=$(( RANDOM % 18001 - 9000 ))
        local lng_raw=$(( RANDOM % 36001 - 18000 ))
        # Use awk for reliable float formatting to avoid issues like "0.-50"
        local lat=$(awk "BEGIN {printf \"%.2f\", $lat_raw / 100}")
        local lng=$(awk "BEGIN {printf \"%.2f\", $lng_raw / 100}")
        echo "${BASE_URL}${endpoint}?latitude=${lat}&longitude=${lng}"
    fi
}

# Pre-generate URLs to avoid overhead during execution
echo "Generating URL list..."
URLS_FILE=$(mktemp)
for i in $(seq 1 $NUM_REQUESTS); do
    gen_url >> "$URLS_FILE"
done

echo "Executing requests..."
start_time=$(date +%s)

# Use xargs to run curl in parallel
cat "$URLS_FILE" | xargs -I {} -P $CONCURRENCY curl -s -o /dev/null -w "%{http_code}\n" -H "X-App-Id: ${APP_ID}" "{}" | \
    awk -v total_req="$NUM_REQUESTS" '
    BEGIN {
        width = 50
    }
    {
        count[$1]++
        total++
        
        # Calculate progress bar
        percent = total / total_req
        filled = int(percent * width)
        bar = ""
        for (i = 0; i < filled; i++) bar = bar "#"
        for (i = filled; i < width; i++) bar = bar "-"
        
        printf "\r[%s] %d%% (%d/%d)", bar, int(percent * 100), total, total_req
        fflush()
    }
    END {
        printf "\n\nLoad Generation Complete!\n"
        printf "%-10s %-10s\n", "Status", "Count"
        print "---------------------"
        for (code in count) {
            printf "%-10s %-10d\n", code, count[code]
        }
    }'

end_time=$(date +%s)
duration=$((end_time - start_time))

echo "Total Time: ${duration}s"
if [ $duration -gt 0 ]; then
    echo "Requests per second: $((NUM_REQUESTS / duration))"
fi

rm "$URLS_FILE"
