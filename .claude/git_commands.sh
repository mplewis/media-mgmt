#!/bin/bash

# Git Commands Script for Claude
# This script provides common git operations for Claude to use

set -e  # Exit on any error

case "$1" in
    "add")
        shift
        git add "$@"
        ;;
    "rm")
        shift
        git rm "$@"
        ;;
    "status")
        git status
        ;;
    "commit")
        shift
        git commit "$@"
        ;;
    "add-all")
        git add -A
        ;;
    "commit-message")
        # Usage: git_commands.sh commit-message "Your commit message"
        shift
        git commit -m "$1"
        ;;
    "add-commit")
        # Usage: git_commands.sh add-commit file1 file2 "commit message"
        # Add all files except the last argument, then commit with the last argument as message
        files=("$@")
        message="${files[-1]}"
        unset 'files[-1]'
        
        if [ ${#files[@]} -gt 0 ]; then
            git add "${files[@]}"
        else
            git add -A
        fi
        git commit -m "$message"
        ;;
    *)
        echo "Usage: $0 {add|rm|status|commit|add-all|commit-message|add-commit}"
        echo ""
        echo "Commands:"
        echo "  add <files...>                 - Add files to staging"
        echo "  rm <files...>                  - Remove files from git"
        echo "  status                         - Show git status"
        echo "  commit <args...>               - Commit with arguments"
        echo "  add-all                        - Add all changes"
        echo "  commit-message <message>       - Commit with message"
        echo "  add-commit <files...> <msg>    - Add files and commit"
        exit 1
        ;;
esac