#!/bin/bash

# Load language file
LANGUAGE_FILE="languages/${LANG}.lang"
if [[ -f "$LANGUAGE_FILE" ]]; then
    source "$LANGUAGE_FILE"
else
    echo "Language file not found: $LANGUAGE_FILE"
    exit 1
fi

# ANSI Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Print messages with color
print_message() {
    echo -e "${BLUE}-----------------------------------${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${BLUE}-----------------------------------${NC}"
}

print_warning() {
    echo -e "${YELLOW}${LANG_WARNING}$1${NC}"
}

print_error() {
    echo -e "${RED}${LANG_ERROR}$1${NC}"
}