#!/bin/bash

# FIPS Compliance Checker Script

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Compliance check functions
check_go_crypto() {
    echo -e "${YELLOW}Checking Go Cryptographic Libraries...${NC}"
    
    # Check for FIPS-approved cryptographic usage
    go_crypto_files=$(find . -type f -name "*.go" | xargs grep -l "crypto\.")
    
    if [ -z "$go_crypto_files" ]; then
        echo -e "${GREEN}✓ No direct crypto usage found${NC}"
        return 0
    fi
    
    non_fips_patterns=(
        "md5"
        "sha1"
        "rc4"
        "des"
        "blowfish"
    )
    
    non_fips_findings=()
    
    for file in $go_crypto_files; do
        for pattern in "${non_fips_patterns[@]}"; do
            if grep -q "$pattern" "$file"; then
                non_fips_findings+=("$file")
                break
            fi
        done
    done
    
    if [ ${#non_fips_findings[@]} -eq 0 ]; then
        echo -e "${GREEN}✓ All cryptographic libraries appear FIPS-compliant${NC}"
    else
        echo -e "${RED}✗ Non-FIPS compliant crypto found in:${NC}"
        printf '%s\n' "${non_fips_findings[@]}"
        return 1
    fi
}

check_tls_config() {
    echo -e "${YELLOW}Checking TLS/SSL Configurations...${NC}"
    
    tls_files=$(find . -type f \( -name "*.go" -o -name "*.yaml" -o -name "*.yml" \) | xargs grep -l "tls\.")
    
    weak_tls_patterns=(
        "TLSv1.0"
        "TLSv1.1"
        "InsecureSkipVerify"
    )
    
    tls_vulnerabilities=()
    
    for file in $tls_files; do
        for pattern in "${weak_tls_patterns[@]}"; do
            if grep -q "$pattern" "$file"; then
                tls_vulnerabilities+=("$file")
                break
            fi
        done
    done
    
    if [ ${#tls_vulnerabilities[@]} -eq 0 ]; then
        echo -e "${GREEN}✓ TLS configurations appear secure${NC}"
    else
        echo -e "${RED}✗ Weak TLS configurations found in:${NC}"
        printf '%s\n' "${tls_vulnerabilities[@]}"
        return 1
    fi
}

check_secret_management() {
    echo -e "${YELLOW}Checking Secret Management...${NC}"
    
    secret_files=$(find . -type f \( -name "*.go" -o -name "*.yaml" -o -name "*.yml" \))
    
    hardcoded_secret_patterns=(
        "password\s*="
        "secret\s*="
        "token\s*="
        "apiKey\s*="
    )
    
    secret_vulnerabilities=()
    
    for file in $secret_files; do
        for pattern in "${hardcoded_secret_patterns[@]}"; do
            if grep -Pq "$pattern" "$file"; then
                secret_vulnerabilities+=("$file")
                break
            fi
        done
    done
    
    if [ ${#secret_vulnerabilities[@]} -eq 0 ]; then
        echo -e "${GREEN}✓ No hardcoded secrets found${NC}"
    else
        echo -e "${RED}✗ Potential secret exposure in:${NC}"
        printf '%s\n' "${secret_vulnerabilities[@]}"
        return 1
    fi
}

main() {
    echo -e "${YELLOW}Starting FIPS Compliance Check${NC}"
    
    checks=(
        check_go_crypto
        check_tls_config
        check_secret_management
    )
    
    overall_status=0
    
    for check in "${checks[@]}"; do
        $check
        status=$?
        if [ $status -ne 0 ]; then
            overall_status=1
        fi
    done
    
    if [ $overall_status -eq 0 ]; then
        echo -e "${GREEN}✓ FIPS Compliance Check Passed${NC}"
    else
        echo -e "${RED}✗ FIPS Compliance Check Failed${NC}"
    fi
    
    exit $overall_status
}

main