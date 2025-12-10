#!/bin/bash
set -e

# Cleanup function
cleanup() {
    echo ""
    echo "ℹ️ Cleaning up containers..."
    podman stop "$APP_CONTAINER" "$DB_CONTAINER" "$REDIS_CONTAINER" 2>/dev/null || true
    podman rm -v "$APP_CONTAINER" "$DB_CONTAINER" "$REDIS_CONTAINER" 2>/dev/null || true
    podman network rm "$NETWORK_NAME" 2>/dev/null || true
    rm -f Dockerfile.trace
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Script to trace syscalls during integration test and generate seccomp profile

NETWORK_NAME="${NETWORK_NAME:-testnetwork}"
DB_CONTAINER="${DB_CONTAINER:-test-mariadb}"
APP_CONTAINER="${APP_CONTAINER:-test-axfr-trace}"
REDIS_CONTAINER="${REDIS_CONTAINER:-test-redis}"
DB_PASSWORD="${DB_PASSWORD:-testpass123}"
STRACE_OUTPUT="${STRACE_OUTPUT:-/tmp/strace_output.log}"
SECCOMP_PROFILE="${SECCOMP_PROFILE:-seccomp-profile.json}"

echo "ℹ️ Creating podman network..."
podman network create "$NETWORK_NAME" || true

echo "ℹ️ Starting MariaDB container..."
podman run -d --name "$DB_CONTAINER" \
    --network "$NETWORK_NAME" \
    -e MYSQL_ROOT_PASSWORD="$DB_PASSWORD" \
    docker.io/library/mariadb:latest

echo "ℹ️ Starting Redis container..."
podman run -d --name "$REDIS_CONTAINER" \
    --network "$NETWORK_NAME" \
    docker.io/library/redis:latest

echo "ℹ️ Building application container..."
podman build -t "$APP_CONTAINER" .

echo "ℹ️ Building tracing container with strace..."
# Create a temporary Dockerfile for tracing
cat > Dockerfile.trace << 'EOF'
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN GOEXPERIMENT=greenteagc CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o /go-axfr-backend ./cmd/server

FROM alpine:latest
RUN apk add --no-cache strace bash
COPY --from=build /go-axfr-backend /go-axfr-backend
RUN mkdir -p /strace-output && chmod 777 /strace-output
EXPOSE 8080
# Use strace with -f to follow forks and -e trace=all to capture all syscalls
# Output to /strace-output/strace.log to avoid /tmp symlink issues
# Use shell form to ensure proper execution
CMD ["/bin/sh", "-c", "strace -f -e trace=all -o /strace-output/strace.log -s 200 /go-axfr-backend 2>&1"]
EOF

podman build -f Dockerfile.trace -t "${APP_CONTAINER}-trace" .

echo "ℹ️ Waiting for MariaDB to be ready..."
sleep 5

echo "ℹ️ Importing database dumps..."
podman exec -i "$DB_CONTAINER" bash -c "until mariadb -u root -p$DB_PASSWORD -e 'SELECT 1'; do sleep 1; done" || true

podman exec -i "$DB_CONTAINER" mariadb -u root -p"$DB_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS nudiff;" || true
podman exec -i "$DB_CONTAINER" mariadb -u root -p"$DB_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS nudump;" || true

podman cp migrations/nudiff.sql "$DB_CONTAINER":/tmp/nudiff.sql
podman cp migrations/nudump.sql "$DB_CONTAINER":/tmp/nudump.sql
podman exec "$DB_CONTAINER" bash -c "mariadb -u root -p'$DB_PASSWORD' nudiff < /tmp/nudiff.sql" || true
podman exec "$DB_CONTAINER" bash -c "mariadb -u root -p'$DB_PASSWORD' nudump < /tmp/nudump.sql" || true

echo "✅ Database dumps imported successfully"

echo "ℹ️ Starting application container with strace..."
# Run with SYS_PTRACE capability so strace can trace processes
podman run -d --name "$APP_CONTAINER" \
    --cap-add=SYS_PTRACE \
    -p 8080:8080 \
    --network "$NETWORK_NAME" \
    -e MYSQL_HOSTNAME="$DB_CONTAINER" \
    -e MYSQL_NUDUMP_DATABASE=nudump \
    -e MYSQL_NUDUMP_USERNAME=root \
    -e MYSQL_NUDUMP_PASSWORD="$DB_PASSWORD" \
    -e MYSQL_NU_DATABASE=nudiff \
    -e MYSQL_NU_USERNAME=root \
    -e MYSQL_NU_PASSWORD="$DB_PASSWORD" \
    -e REDIS_URL="$REDIS_CONTAINER:6379" \
    "${APP_CONTAINER}-trace"

echo "✅ Test environment is ready!"

echo "ℹ️ Waiting for application to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "http://localhost:8080/ready" > /dev/null 2>&1; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "❌ Timeout waiting for application to start"
        exit 1
    fi
    sleep 2
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "✅ Application is ready"

# Check if strace is writing output
echo "ℹ️ Verifying strace is capturing syscalls..."
sleep 2
if podman exec "$APP_CONTAINER" test -f /strace-output/strace.log 2>/dev/null; then
    INITIAL_LINES=$(podman exec "$APP_CONTAINER" wc -l < /strace-output/strace.log 2>/dev/null || echo "0")
    echo "ℹ️ Strace log exists with $INITIAL_LINES lines before tests"
else
    echo "⚠️  Strace log not found yet, strace may not be writing output"
    podman exec "$APP_CONTAINER" ps aux 2>/dev/null || true
fi

echo "ℹ️ Running integration tests to capture syscalls..."
./integration_test.sh

# Check strace output after tests
if podman exec "$APP_CONTAINER" test -f /strace-output/strace.log 2>/dev/null; then
    FINAL_LINES=$(podman exec "$APP_CONTAINER" wc -l < /strace-output/strace.log 2>/dev/null || echo "0")
    echo "ℹ️ Strace log now has $FINAL_LINES lines after tests"
fi

echo "ℹ️ Extracting strace log from running container..."
# Extract the log while container is still running to ensure strace has flushed
# Check if strace log exists
if podman exec "$APP_CONTAINER" test -f /strace-output/strace.log 2>/dev/null; then
    echo "ℹ️ Found strace log, extracting..."
    # Use podman exec to cat the file instead of cp to avoid symlink issues
    podman exec "$APP_CONTAINER" cat /strace-output/strace.log > "$STRACE_OUTPUT" 2>/dev/null || {
        echo "⚠️  Could not extract via exec, trying cp..."
        podman cp "$APP_CONTAINER":/strace-output/strace.log "$STRACE_OUTPUT" 2>/dev/null || true
    }
    # Verify we got content
    if [ -f "$STRACE_OUTPUT" ] && [ -s "$STRACE_OUTPUT" ]; then
        echo "✅ Successfully extracted strace log ($(wc -l < "$STRACE_OUTPUT" | tr -d ' ') lines)"
    else
        echo "⚠️  Strace log appears empty or missing"
        podman exec "$APP_CONTAINER" ls -la /strace-output/ 2>/dev/null || true
    fi
else
    echo "⚠️  Strace log file not found at /strace-output/strace.log"
    echo "ℹ️ Checking container filesystem..."
    podman exec "$APP_CONTAINER" ls -la /strace-output/ 2>/dev/null || true
    podman exec "$APP_CONTAINER" find / -name "strace.log" 2>/dev/null | head -5 || true
fi

echo "ℹ️ Stopping container..."
podman stop "$APP_CONTAINER" || true

echo "ℹ️ Generating seccomp profile from strace output..."
# Extract unique syscall names from strace output
# Format: strace outputs syscalls like "openat(..." or "[pid 123] openat(..."
# We need to extract just the syscall names
# 
# This script validates syscalls to filter out:
# - Non-syscall C library functions (htonl, htons, inet_addr, etc.)
# - Invalid entries from parsing errors (single chars, fragments)
# - Entries with invalid characters or formats

if [ ! -f "$STRACE_OUTPUT" ]; then
    echo "❌ Strace output file not found. Trying alternative location..."
    # Try to get it from container logs or different path
    podman logs "$APP_CONTAINER" > "$STRACE_OUTPUT" 2>&1 || true
fi

# Extract syscall names from strace output
# Strace formats:
# - "[pid 123] syscall_name("
# - "syscall_name("
# - "syscall_name(...) = result"
# - Lines starting with syscall names

# Known non-syscall functions that appear in strace output (C library functions, etc.)
NON_SYSCALLS="htonl htons inet_addr inet_ntoa inet_aton ntohl ntohs"

# Extract syscall names from strace output
SYSCALLS=$(cat "$STRACE_OUTPUT" 2>/dev/null | \
    # Remove process ID prefixes like "[pid 123]"
    sed 's/\[pid [0-9]*\] //' | \
    # Match syscall patterns: word followed by opening parenthesis
    # More precise pattern: syscall name must be followed by ( and be at start of line or after whitespace
    grep -oE '(^|[[:space:]]+)[a-zA-Z][a-zA-Z0-9_]+\([[:space:]]*[^)]*\)' | \
    # Extract just the syscall name (remove leading whitespace and everything after opening paren)
    sed 's/^[[:space:]]*//' | \
    sed 's/(.*$//' | \
    # Filter out strace control messages and invalid patterns
    grep -vE '^$|^---|^\+\+\+|^strace|^exit|^resumed|^unfinished|^detached|^\[|^\]|^\{|^\}|^\(|^\)|^,|^;|^:|^"|^'"'"'|^`|^&|^\*|^#|^%|^@|^!|^\?|^~|^\||^\\|^/|^\-|^\+|^=|^[0-9]|^[0-9a-fA-F]+$' | \
    # Must be at least 3 characters (shortest real syscall is "brk")
    awk 'length($0) >= 3' | \
    # Remove known non-syscall functions
    grep -vE "^($(echo $NON_SYSCALLS | tr ' ' '|'))$" | \
    sort -u || true)

# If extraction failed or got too few syscalls, try alternative methods
if [ -z "$SYSCALLS" ] || [ "$(echo "$SYSCALLS" | wc -l | tr -d ' ')" -lt 10 ]; then
    echo "ℹ️ Trying alternative syscall extraction method..."
    # Try extracting from first word of each line that looks like a syscall
    SYSCALLS=$(cat "$STRACE_OUTPUT" 2>/dev/null | \
        # Get first word from lines that contain syscall patterns
        awk '/^[a-zA-Z][a-zA-Z0-9_]*\(/ {print $1}' | \
        sed 's/(.*$//' | \
        grep -E '^[a-zA-Z][a-zA-Z0-9_]{2,}$' | \
        grep -vE '^strace|^exit|^resumed|^unfinished|^detached|^---' | \
        grep -vE "^($(echo $NON_SYSCALLS | tr ' ' '|'))$" | \
        sort -u || true)
fi

# Additional validation: filter out entries that are clearly not syscalls
# Common Linux syscalls are typically lowercase with underscores, 3-30 chars
SYSCALLS=$(echo "$SYSCALLS" | \
    # Must be valid identifier format
    grep -E '^[a-z][a-z0-9_]{2,29}$' | \
    # Remove single character or very short entries that are likely fragments
    awk 'length($0) >= 3' | \
    # Remove entries with uppercase (except for specific known syscalls, but most are lowercase)
    grep -vE '[A-Z]' | \
    sort -u || true)

if [ -z "$SYSCALLS" ]; then
    echo "❌ Failed to extract syscalls from strace output"
    echo "ℹ️ Strace output preview:"
    head -50 "$STRACE_OUTPUT" || true
    echo ""
    echo "ℹ️ Full strace output saved to: $STRACE_OUTPUT"
    echo "ℹ️ You may need to manually inspect the strace output"
    exit 1
fi

SYSCALL_COUNT=$(echo "$SYSCALLS" | grep -v '^$' | wc -l | tr -d ' ')
if [ "$SYSCALL_COUNT" -lt 10 ]; then
    echo "⚠️  Warning: Only found $SYSCALL_COUNT syscalls, which seems low."
    echo "ℹ️ This might indicate an issue with strace output parsing."
    echo "ℹ️ Strace output preview:"
    head -50 "$STRACE_OUTPUT" || true
fi

echo "ℹ️ Found $SYSCALL_COUNT unique syscalls (before validation)"

# Generate seccomp profile JSON
# Note: This profile is generated by tracing syscalls during integration tests
# To use in Kubernetes, add to your pod spec:
#   securityContext:
#     seccompProfile:
#       type: Localhost
#       localhostProfile: seccomp-profile.json
cat > "$SECCOMP_PROFILE" << 'PROFILE_EOF'
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": [
    "SCMP_ARCH_X86_64",
    "SCMP_ARCH_X86",
    "SCMP_ARCH_X32"
  ],
  "syscalls": [
PROFILE_EOF

# Add essential syscalls that might not be captured but are needed
# These are split properly (one per line)
ESSENTIAL_SYSCALLS=$(cat << 'EOF'
rt_sigreturn
epoll_pwait
epoll_wait
EOF
)

# Combine extracted syscalls with essential ones, removing duplicates
COMBINED_SYSCALLS=$(echo -e "$SYSCALLS\n$ESSENTIAL_SYSCALLS" | grep -v '^$' | sort -u)

# Final validation: ensure each syscall is valid before adding to profile
VALIDATED_SYSCALLS=""
while IFS= read -r syscall; do
    if [ -n "$syscall" ] && [ "$syscall" != "" ]; then
        # Validate: must be lowercase, alphanumeric/underscore, 3-30 chars
        if echo "$syscall" | grep -qE '^[a-z][a-z0-9_]{2,29}$'; then
            # Skip if it's a known non-syscall
            if ! echo "$NON_SYSCALLS" | grep -qE "^$syscall | $syscall | $syscall$|^$syscall$"; then
                VALIDATED_SYSCALLS="${VALIDATED_SYSCALLS}${syscall}\n"
            fi
        fi
    fi
done <<< "$COMBINED_SYSCALLS"

# Remove trailing newline and get unique list of validated syscalls
ALL_SYSCALLS=$(echo -e "$VALIDATED_SYSCALLS" | grep -v '^$' | sort -u)

VALIDATED_COUNT=$(echo "$ALL_SYSCALLS" | grep -v '^$' | wc -l | tr -d ' ')
FILTERED_COUNT=$((SYSCALL_COUNT + $(echo "$ESSENTIAL_SYSCALLS" | wc -l | tr -d ' ') - VALIDATED_COUNT))
if [ "$FILTERED_COUNT" -gt 0 ]; then
    echo "ℹ️ Filtered out $FILTERED_COUNT invalid syscall entries"
fi

# Add each validated syscall to the profile
FIRST=true
while IFS= read -r syscall; do
    if [ -n "$syscall" ] && [ "$syscall" != "" ]; then
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo "," >> "$SECCOMP_PROFILE"
        fi
        cat >> "$SECCOMP_PROFILE" << EOF
    {
      "names": ["$syscall"],
      "action": "SCMP_ACT_ALLOW"
    }
EOF
    fi
done <<< "$ALL_SYSCALLS"

cat >> "$SECCOMP_PROFILE" << 'PROFILE_EOF'
  ]
}
PROFILE_EOF

FINAL_COUNT=$(echo "$ALL_SYSCALLS" | grep -v '^$' | wc -l | tr -d ' ')
echo "✅ Seccomp profile generated: $SECCOMP_PROFILE"
echo "ℹ️ Profile contains $FINAL_COUNT validated syscalls"

echo "✅ Done! Seccomp profile saved to $SECCOMP_PROFILE"
echo ""
echo "ℹ️ To use this profile in Kubernetes, copy it to your nodes and reference it in your pod spec:"
echo "   securityContext:"
echo "     seccompProfile:"
echo "       type: Localhost"
echo "       localhostProfile: seccomp-profile.json"

