#!/bin/bash

# Interactive Git Branch Deletion Script
# This script allows you to interactively delete local Git branches
# It will preserve main, develop, and the current branch

# Get all branches except main, develop, and current branch
branches=$(git branch | grep -v "^\*\|main\|develop" | sed 's/^[[:space:]]*//')

if [ -z "$branches" ]; then
    echo "No branches to delete (keeping main, develop, and current branch)"
    exit 0
fi

echo "Available branches to delete:"
echo "$branches"
echo
echo "You can delete branches one by one. Press Ctrl+C to exit anytime."
echo

for branch in $branches; do
    echo -n "Delete branch '$branch'? (y/n/q): "
    read -r answer
    case $answer in
        [Yy]* )
            echo "Deleting $branch..."
            git branch -d "$branch" 2>/dev/null
            if [ $? -ne 0 ]; then
                echo -n "Branch '$branch' has unmerged commits. Force delete? (y/n): "
                read -r force_answer
                case $force_answer in
                    [Yy]* )
                        git branch -D "$branch"
                        echo "Force deleted $branch"
                        ;;
                    * )
                        echo "Skipped $branch"
                        ;;
                esac
            else
                echo "Deleted $branch"
            fi
            ;;
        [Qq]* )
            echo "Exiting..."
            exit 0
            ;;
        * )
            echo "Skipped $branch"
            ;;
    esac
    echo
done

echo "Branch cleanup completed!"