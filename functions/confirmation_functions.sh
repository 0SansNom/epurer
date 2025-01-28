#!/bin/bash

source functions/common_functions.sh

# Function to ask for user confirmation
confirm_action() {
    read -p "${LANG_CONFIRM}$1${LANG_CONFIRMATION}" response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY]|[oO][uU][iI]|[oO])$ ]]; then
        return 0
    else
        print_warning "${LANG_ABORTED}"
        return 1
    fi
}