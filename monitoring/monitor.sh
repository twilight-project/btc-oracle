#!/bin/bash
# BTC Oracle Log Monitor - Posts errors to Discord
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${1:-$SCRIPT_DIR/config.env}"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Config file not found: $CONFIG_FILE"
    exit 1
fi

source "$CONFIG_FILE"

if [ -z "$DISCORD_WEBHOOK_URL" ]; then
    echo "Error: DISCORD_WEBHOOK_URL is not set in $CONFIG_FILE"
    exit 1
fi

send_discord() {
    local message="$1"
    if [ ${#message} -gt "$MAX_MSG_LENGTH" ]; then
        message="${message:0:$MAX_MSG_LENGTH}..."
    fi

    message=$(echo "$message" | python3 -c 'import sys,json; print(json.dumps(sys.stdin.read())[1:-1])')
    local payload="{\"content\":\"$message\"}"

    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$DISCORD_WEBHOOK_URL")

    if [ "$response" = "204" ] || [ "$response" = "200" ]; then
        return 0
    elif [ "$response" = "429" ]; then
        echo "$(date): Rate limited by Discord, sleeping 5s..."
        sleep 5
        return 1
    else
        echo "$(date): Discord webhook returned HTTP $response"
        return 1
    fi
}

get_file_lines() {
    wc -l < "$1" 2>/dev/null | tr -d ' '
}

get_file_inode() {
    stat -c '%i' "$1" 2>/dev/null || stat -f '%i' "$1" 2>/dev/null || echo "0"
}

# Post matching lines to Discord
# Args: lines pattern label
# Returns 0 if no errors or posted successfully, 1 if post failed
post_errors() {
    local lines="$1"
    local pattern="$2"
    local label="$3"

    local matched
    matched=$(echo "$lines" | grep -iE "$pattern" || true)

    if [ -n "$matched" ]; then
        local error_count
        error_count=$(echo "$matched" | wc -l | tr -d ' ')

        local timestamp
        timestamp=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
        local msg="🚨 **[$SERVER_NAME / $NODE_NAME - $label]** $error_count error(s) at $timestamp\n\`\`\`\n$matched\n\`\`\`"

        echo "$(date): [$label] Found $error_count error(s), posting to Discord..."

        if send_discord "$msg"; then
            echo "$(date): [$label] Posted to Discord successfully"
            return 0
        else
            echo "$(date): [$label] Failed to post to Discord, will retry next cycle"
            return 1
        fi
    fi
    return 0
}

# Monitor a single log file
# Args: file_path pattern offset_file label
# Offset file format: "inode:line_number"
monitor_file() {
    local file="$1"
    local pattern="$2"
    local offset_file="$3"
    local label="$4"

    if [ ! -f "$file" ]; then
        return
    fi

    local current_lines
    current_lines=$(get_file_lines "$file")
    local current_inode
    current_inode=$(get_file_inode "$file")

    local last_line=0
    local last_inode="0"
    if [ -f "$offset_file" ]; then
        local stored
        stored=$(cat "$offset_file")
        if [[ "$stored" == *:* ]]; then
            last_inode="${stored%%:*}"
            last_line="${stored##*:}"
        else
            # Legacy format (just line number)
            last_line="$stored"
        fi
    fi

    # Detect log rotation: inode changed or file shrunk
    if [ "$current_inode" != "$last_inode" ] && [ "$last_inode" != "0" ]; then
        echo "$(date): Log rotation detected for $file (inode changed: $last_inode -> $current_inode)"

        # Check the rotated file (.1 suffix) for missed lines since last offset
        local rotated_file="${file}.1"
        if [ -f "$rotated_file" ]; then
            local rotated_lines
            rotated_lines=$(get_file_lines "$rotated_file")
            if [ "$rotated_lines" -gt "$last_line" ]; then
                echo "$(date): [$label] Checking rotated file for ${rotated_lines} - ${last_line} missed lines..."
                local missed
                missed=$(tail -n +"$((last_line + 1))" "$rotated_file" || true)
                if [ -n "$missed" ]; then
                    post_errors "$missed" "$pattern" "$label (rotated)"
                    # Even if post fails, we can't retry rotated file reliably
                    # (it may get overwritten by next rotation), so we move on
                fi
            fi
        fi

        # Reset to read new file from beginning
        last_line=0
    elif [ "$current_lines" -lt "$last_line" ]; then
        # Same inode but file shrunk (truncation)
        echo "$(date): Log truncation detected for $file, resetting offset"
        last_line=0
    fi

    if [ "$current_lines" -gt "$last_line" ]; then
        local new_lines
        new_lines=$(tail -n +"$((last_line + 1))" "$file" | head -n "$((current_lines - last_line))")

        if post_errors "$new_lines" "$pattern" "$label"; then
            echo "$current_inode:$current_lines" > "$offset_file"
        fi
        # If post_errors failed, offset stays — retry next cycle
    else
        # No new lines, just update inode in case it changed
        echo "$current_inode:$current_lines" > "$offset_file"
    fi
}

# Build offset file path from label (sanitize label to safe filename)
get_offset_file() {
    local label="$1"
    local safe_label
    safe_label=$(echo "$label" | tr -cs 'a-zA-Z0-9_-' '_')
    echo "$SCRIPT_DIR/.monitor_offset_${safe_label}"
}

if [ ${#MONITOR_FILES[@]} -eq 0 ]; then
    echo "Error: No MONITOR_FILES defined in $CONFIG_FILE"
    exit 1
fi

# Initialize offsets to current end of file (skip old errors on first run)
for entry in "${MONITOR_FILES[@]}"; do
    IFS='|' read -r file pattern label <<< "$entry"
    offset_file=$(get_offset_file "$label")
    if [ -f "$file" ] && [ ! -f "$offset_file" ]; then
        inode=$(get_file_inode "$file")
        lines=$(get_file_lines "$file")
        echo "$inode:$lines" > "$offset_file"
    fi
done

echo "$(date): BTC Oracle Log Monitor started"
echo "  Config: $CONFIG_FILE"
echo "  Server: $SERVER_NAME / $NODE_NAME"
echo "  Poll interval: ${POLL_INTERVAL}s"
echo "  Monitoring ${#MONITOR_FILES[@]} file(s):"
for entry in "${MONITOR_FILES[@]}"; do
    IFS='|' read -r file pattern label <<< "$entry"
    echo "    - [$label] $file (pattern: $pattern)"
done

while true; do
    for entry in "${MONITOR_FILES[@]}"; do
        IFS='|' read -r file pattern label <<< "$entry"
        offset_file=$(get_offset_file "$label")
        monitor_file "$file" "$pattern" "$offset_file" "$label"
    done

    sleep "$POLL_INTERVAL"
done
