#!/bin/bash

source functions/common_functions.sh
source functions/confirmation_functions.sh

# Function to clean the trash
cleanup_trash() {
    print_message "Vidage de la Corbeille..."
    if confirm_action "vider la corbeille ?"; then
        rm -rf ~/.Trash/* 2>/dev/null || print_warning "Impossible de vider la corbeille utilisateur."
        rm -rf /Volumes/*/.Trashes/* 2>/dev/null || print_warning "Impossible de vider les corbeilles externes."
        print_message "${LANG_SUCCESS}Corbeille vidée."
    fi
}

# Function to clear system caches
cleanup_caches() {
    print_message "Effacement des caches système..."
    if confirm_action "effacer les caches ?"; then
        sudo rm -rf ~/Library/Caches/* 2>/dev/null || print_warning "Impossible d'effacer les caches utilisateur."
        sudo rm -rf /Library/Caches/* 2>/dev/null || print_warning "Impossible d'effacer les caches système."
        sudo rm -rf /System/Library/Caches/* 2>/dev/null || print_warning "Impossible d'effacer les caches principaux."
        print_message "${LANG_SUCCESS}Caches effacés."
    fi
}

# Function to remove old log files
cleanup_logs() {
    print_message "Suppression des anciens fichiers journaux..."
    if confirm_action "supprimer les anciens fichiers journaux ?"; then
        sudo rm -rf /private/var/log/asl/*.asl 2>/dev/null || print_warning "Impossible de supprimer les fichiers ASL."
        sudo rm -rf /private/var/log/*.log 2>/dev/null || print_warning "Impossible de supprimer les fichiers journaux."
        print_message "${LANG_SUCCESS}Fichiers journaux supprimés."
    fi
}

# Function to remove temporary files
cleanup_temp_files() {
    print_message "Suppression des fichiers temporaires..."
    if confirm_action "supprimer les fichiers temporaires ?"; then
        sudo rm -rf /private/var/tmp/* 2>/dev/null || print_warning "Impossible de supprimer les fichiers temporaires du système."
        sudo rm -rf /private/tmp/* 2>/dev/null || print_warning "Impossible de supprimer les fichiers temporaires globaux."
        print_message "${LANG_SUCCESS}Fichiers temporaires supprimés."
    fi
}

# Function to clear DNS cache (requires sudo)
cleanup_dns_cache() {
    print_message "Effacement du cache DNS..."
    if confirm_action "effacer le cache DNS ?"; then
        sudo dscacheutil -flushcache && sudo killall -HUP mDNSResponder 2>/dev/null && print_message "${LANG_SUCCESS}Cache DNS effacé." || print_error "Échec de l'effacement du cache DNS."
    fi
}

# Function to clean up Xcode derived data (if installed)
cleanup_xcode_data() {
    if [ -d ~/Library/Developer/Xcode/DerivedData ]; then
        print_message "Nettoyage des données dérivées de Xcode..."
        if confirm_action "nettoyer les données dérivées de Xcode ?"; then
            rm -rf ~/Library/Developer/Xcode/DerivedData/* 2>/dev/null || print_warning "Impossible de nettoyer les données dérivées de Xcode."
            print_message "${LANG_SUCCESS}Données dérivées de Xcode nettoyées."
        fi
    else
        print_warning "Xcode n'est pas installé ou les données dérivées sont introuvables."
    fi
}

# Function to clean up Homebrew cache (if installed)
cleanup_homebrew_cache() {
    if command -v brew &> /dev/null; then
        print_message "Nettoyage du cache Homebrew..."
        if confirm_action "nettoyer le cache Homebrew ?"; then
            brew cleanup 2>/dev/null && brew cask cleanup 2>/dev/null && brew prune 2>/dev/null && rm -rf $(brew --cache) 2>/dev/null && print_message "${LANG_SUCCESS}Cache Homebrew nettoyé." || print_warning "Impossible de nettoyer le cache Homebrew."
        fi
    else
        print_warning "Homebrew n'est pas installé."
    fi
}

# Function to clean up unused localizations (language files)
cleanup_localizations() {
    print_message "Suppression des localisations non utilisées..."
    if confirm_action "supprimer les localisations non utilisées ?"; then
        sudo find / -iname "*.lproj" -type d | grep -v en | grep -v fr | grep -v es | xargs rm -rf 2>/dev/null || print_warning "Impossible de supprimer certaines localisations."
        print_message "${LANG_SUCCESS}Localisations non utilisées supprimées."
    fi
}

# Function to remove outdated iOS device backups (if any)
cleanup_ios_backups() {
    if [ -d ~/Library/Application\ Support/MobileSync/Backup ]; then
        print_message "Suppression des sauvegardes obsolètes des appareils iOS..."
        if confirm_action "supprimer les sauvegardes obsolètes des appareils iOS ?"; then
            # Uncomment the line below to actually remove the backups
            # WARNING: This will delete all iOS backups!
            # rm -rf ~/Library/Application\ Support/MobileSync/Backup/* 2>/dev/null || print_warning "Impossible de supprimer les sauvegardes iOS."
            print_warning "Cette fonctionnalité est désactivée dans ce script par défaut."
        fi
    else
        print_warning "Aucune sauvegarde iOS trouvée."
    fi
}

# Function to clean up Launchpad database (if needed)
cleanup_launchpad_db() {
    print_message "Nettoyage de la base de données Launchpad..."
    if confirm_action "nettoyer la base de données Launchpad ?"; then
        sqlite3 ~/Library/Application\ Support/Dock/*.db "DELETE FROM apps; DELETE FROM groups; DELETE FROM items; VACUUM;" 2>/dev/null && killall Dock 2>/dev/null && print_message "${LANG_SUCCESS}Base de données Launchpad nettoyée." || print_warning "Impossible de nettoyer la base de données Launchpad."
    fi
}