#!/bin/bash

BASE_URL="http://localhost:8080"
APP_ID="myappid"
NUM_REQUESTS=${1:-1}
CONCURRENCY=${2:-10}

echo "Starting optimized load generation: $NUM_REQUESTS iterations with concurrency $CONCURRENCY..."

# Pre-generate URLs to avoid overhead during execution
echo "Generating URL list..."
URLS_FILE=$(mktemp)

versions=("1.0.0" "1.0.1" "1.1.0" "2.0.0")

for i in $(seq 1 $NUM_REQUESTS); do
    uuid_num=$(( RANDOM % 30 + 1 ))
    ver=${versions[$RANDOM % 4]}
    lat_raw=$(( RANDOM % 18001 - 9000 ))
    lng_raw=$(( RANDOM % 36001 - 18000 ))
    # Use awk for reliable float formatting to avoid issues like "0.-50"
    lat=$(awk "BEGIN {printf \"%.2f\", $lat_raw / 100}")
    lng=$(awk "BEGIN {printf \"%.2f\", $lng_raw / 100}")

    echo "${BASE_URL}/ping?uuid=uuid-${uuid_num}&version=${ver}" >> "$URLS_FILE"
    echo "${BASE_URL}/tides/extremes?latitude=${lat}&longitude=${lng}" >> "$URLS_FILE"
    echo "${BASE_URL}/tides/timeline?latitude=${lat}&longitude=${lng}" >> "$URLS_FILE"
    echo "${BASE_URL}/data/reverse-geocode?latitude=${lat}&longitude=${lng}" >> "$URLS_FILE"
done

TOTAL_REQUESTS=$(( NUM_REQUESTS * 4 ))
echo "Executing $TOTAL_REQUESTS requests..."
start_time=$(date +%s)

# Use xargs to run curl in parallel
cat "$URLS_FILE" | xargs -I {} -P $CONCURRENCY curl -s -o /dev/null -w "%{http_code}\n" -H "X-App-Id: ${APP_ID}" "{}" | \
    awk -v total_req="$TOTAL_REQUESTS" '
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
    echo "Requests per second: $((TOTAL_REQUESTS / duration))"
fi

rm "$URLS_FILE"
