#!/bin/bash

# Devtron Security Migration Assistant

echo "Devtron Security Migration Assistant"
echo "===================================="

# Check for required tools
REQUIRED_TOOLS=("go" "vault" "grep" "sed")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if ! command -v "$tool" &> /dev/null; then
        echo "Error: $tool is not installed. Please install before proceeding."
        exit 1
    fi
done

# Configuration
MIGRATION_LOG="security_migration.log"
PROJECT_ROOT=$(pwd)

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$MIGRATION_LOG"
}

# Find files with potential security risks
find_security_risks() {
    log "Scanning for potential security risks..."
    
    # Search for hardcoded credentials
    grep -rn "password\s*=" "$PROJECT_ROOT" --include=\*.go | tee -a "$MIGRATION_LOG"
    grep -rn "token\s*=" "$PROJECT_ROOT" --include=\*.go | tee -a "$MIGRATION_LOG"
    grep -rn "secret\s*=" "$PROJECT_ROOT" --include=\*.go | tee -a "$MIGRATION_LOG"
}

# Replace hardcoded secrets with environment variable references
replace_hardcoded_secrets() {
    log "Replacing hardcoded secrets with environment variable references..."
    
    # Example replacements (customize based on specific patterns)
    find "$PROJECT_ROOT" -type f -name "*.go" -print0 | xargs -0 sed -i \
        -e 's/password\s*=\s*"[^"]*"/password = os.Getenv("APP_PASSWORD")/g' \
        -e 's/token\s*=\s*"[^"]*"/token = os.Getenv("APP_TOKEN")/g' \
        -e 's/secret\s*=\s*"[^"]*"/secret = os.Getenv("APP_SECRET")/g'
}

# Add secret validation function
add_secret_validation() {
    log "Adding secret validation function..."
    
    cat << 'EOF' >> "$PROJECT_ROOT/pkg/util/secret_validator.go"
package util

import (
    "errors"
    "os"
    "strings"
)

func ValidateSecret(secretKey string) error {
    secret := os.Getenv(secretKey)
    
    if secret == "" {
        return errors.New("secret cannot be empty")
    }
    
    if len(secret) < 12 {
        return errors.New("secret is too short")
    }
    
    // Check for common weak patterns
    weakPatterns := []string{
        "password", "123456", "qwerty", 
        "admin", "default", "letmein"
    }
    
    for _, pattern := range weakPatterns {
        if strings.Contains(strings.ToLower(secret), pattern) {
            return errors.New("secret contains a weak pattern")
        }
    }
    
    return nil
}
EOF
}

# Update JWT token generation
update_jwt_token_generation() {
    log "Updating JWT token generation with improved security..."
    
    # Find files related to token generation
    TOKEN_FILES=$(find "$PROJECT_ROOT" -type f -name "*token*.go")
    
    for file in $TOKEN_FILES; do
        sed -i \
            -e 's/jwt\.SigningMethodHS256/jwt.SigningMethodHS512/g' \
            -e 's/ExpireAt: time\.Now()\./ExpireAt: time.Now().Add(time.Hour * 24)/g' \
            "$file"
    done
}

# Main migration process
main() {
    log "Starting Devtron Security Migration..."
    
    find_security_risks
    replace_hardcoded_secrets
    add_secret_validation
    update_jwt_token_generation
    
    log "Security migration completed. Please review changes in $MIGRATION_LOG"
}

# Run the migration
main