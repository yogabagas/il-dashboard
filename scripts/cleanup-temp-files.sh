#!/bin/bash

# Cleanup script for temporary files in dir/ folder
# This script removes old temporary files created by SendFileBytes

DIR_PATH="./dir"
PATTERN="sendFile*"

echo "Cleaning up temporary files in $DIR_PATH..."

# Count files before cleanup
BEFORE=$(find "$DIR_PATH" -name "$PATTERN" -type f 2>/dev/null | wc -l | tr -d ' ')
echo "Found $BEFORE temporary file(s)"

if [ "$BEFORE" -gt 0 ]; then
    # Remove files
    find "$DIR_PATH" -name "$PATTERN" -type f -delete 2>/dev/null

    # Count files after cleanup
    AFTER=$(find "$DIR_PATH" -name "$PATTERN" -type f 2>/dev/null | wc -l | tr -d ' ')
    REMOVED=$((BEFORE - AFTER))

    echo "Removed $REMOVED file(s)"
    echo "Cleanup completed!"
else
    echo "No temporary files to clean"
fi

