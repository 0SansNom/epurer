#!/bin/bash

# Set default language
LANG=${LANG:-fr_FR}

# Function to determine the appropriate language file
get_language_file() {
    local base_lang="languages/${LANG}.lang"
    if [[ -f "$base_lang" ]]; then
        echo "$base_lang"
    else
        # Try with UTF-8 suffix
        local utf8_lang="languages/${LANG}.UTF-8.lang"
        if [[ -f "$utf8_lang" ]]; then
            echo "$utf8_lang"
        else
            # Default to English if no language file is found
            echo "languages/en_US.lang"
        fi
    fi
}

# Source necessary files
LANGUAGE_FILE=$(get_language_file)
if [[ -f "$LANGUAGE_FILE" ]]; then
    source "$LANGUAGE_FILE"
else
    echo "Language file not found: $LANGUAGE_FILE"
    exit 1
fi

source functions/common_functions.sh
source functions/confirmation_functions.sh
source functions/cleanup_functions.sh

# Main script logic
print_message "Début du processus de nettoyage avec EPURER..."

# Call cleanup functions with user confirmation
cleanup_trash
cleanup_caches
cleanup_logs
cleanup_temp_files
cleanup_dns_cache
cleanup_xcode_data
cleanup_homebrew_cache
cleanup_localizations
cleanup_ios_backups
cleanup_launchpad_db

print_message "${LANG_CLEANUP_COMPLETED}"

exit 0