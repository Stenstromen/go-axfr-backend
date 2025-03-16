#!/bin/bash
set -e

APP_CONTAINER="test-axfr"
APP_IP="localhost"
DB_CONTAINER="test-mariadb"
DB_PASSWORD="testpass123"

# Wait for the application to be ready
echo "ℹ️ Waiting for application to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "http://$APP_IP:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        fail "Timeout waiting for application to start"
    fi
    echo "ℹ️ Waiting... ($(($RETRY_COUNT + 1))/$MAX_RETRIES)"
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "✅ Application is ready"

echo "ℹ️ Running tests..."

# Function to test an endpoint and compare with expected output
test_endpoint() {
    local endpoint=$1
    local expected_file=$2
    local description=$3
    
    echo "🧪 Testing $description..."
    
    # Create a temporary file for the actual response
    local actual_file=$(mktemp)
    
    # Make the request and save the response
    if ! curl -s "http://$APP_IP:8080$endpoint" | jq > "$actual_file"; then
        echo "❌ Failed to call endpoint: $endpoint"
        rm "$actual_file"
        return 1
    fi
    
    # Compare the actual response with the expected response
    if diff -w "$actual_file" "$expected_file" > /dev/null; then
        echo "✅ Test passed: $description"
        rm "$actual_file"
        return 0
    else
        echo "❌ Test failed: $description"
        echo "Expected:"
        cat "$expected_file"
        echo "Actual:"
        cat "$actual_file"
        rm "$actual_file"
        return 1
    fi
}

# Create expected output files
mkdir -p test_data

# Test 1: /nu/0 endpoint
cat > test_data/nu_0.json << 'EOF'
[
  {
    "date": 20250314,
    "amount": 44
  }
]
EOF

# Test 2: /nudomains/20250314/1 endpoint
cat > test_data/nudomains.json << 'EOF'
[
  {
    "domain": "mdsab.nu"
  },
  {
    "domain": "movemind.nu"
  },
  {
    "domain": "oemaayah.nu"
  },
  {
    "domain": "profixsverige.nu"
  },
  {
    "domain": "projektera.nu"
  },
  {
    "domain": "promeet.nu"
  },
  {
    "domain": "protreptik.nu"
  },
  {
    "domain": "rum13.nu"
  },
  {
    "domain": "scouting.nu"
  },
  {
    "domain": "sinme.nu"
  },
  {
    "domain": "slackline.nu"
  },
  {
    "domain": "snall.nu"
  },
  {
    "domain": "stroomvoorbedrijven.nu"
  },
  {
    "domain": "swecanab.nu"
  },
  {
    "domain": "swedshop.nu"
  },
  {
    "domain": "tagrensning.nu"
  },
  {
    "domain": "texas-holdem.nu"
  },
  {
    "domain": "vetterberg.nu"
  },
  {
    "domain": "vkproffsen.nu"
  },
  {
    "domain": "werkenmettrauma.nu"
  }
]
EOF

# Test 3: /search/nu/010 endpoint
cat > test_data/search.json << 'EOF'
[
  {
    "domain": "010.nu"
  },
  {
    "domain": "010housing.nu"
  },
  {
    "domain": "010jongeren.nu"
  },
  {
    "domain": "010acupunctuur.nu"
  },
  {
    "domain": "010jongerenwerk.nu"
  }
]
EOF

# Test 4: /stats/nu endpoint
cat > test_data/stats.json << 'EOF'
[
  {
    "date": "2025-03-15",
    "amount": 207820
  }
]
EOF

# Test 5: /nuappearance/digitalisering.nu endpoint
cat > test_data/appearance.json << 'EOF'
{
  "earliest_date": "2025-03-14"
}
EOF

# Run the tests
failed_tests=0

test_endpoint "/nu/0" "test_data/nu_0.json" "NU domains count" || failed_tests=$((failed_tests + 1))
test_endpoint "/nudomains/20250314/1" "test_data/nudomains.json" "NU domains list for date 20250314 page 1" || failed_tests=$((failed_tests + 1))
test_endpoint "/search/nu/010" "test_data/search.json" "Search NU domains with '010'" || failed_tests=$((failed_tests + 1))
test_endpoint "/stats/nu" "test_data/stats.json" "NU domains statistics" || failed_tests=$((failed_tests + 1))
test_endpoint "/nuappearance/digitalisering.nu" "test_data/appearance.json" "First appearance of digitalisering.nu" || failed_tests=$((failed_tests + 1))

# Report test results
if [ $failed_tests -eq 0 ]; then
    echo "✅ All tests passed!"
    rm -rf test_data
    exit 0
else
    echo "❌ $failed_tests test(s) failed!"
    rm -rf test_data
    exit 1
fi

