#!/bin/bash
# Shows test credentials from dev-users.yaml

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
USERS_FILE="${SCRIPT_DIR}/dev-users.yaml"

# Colors
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m'

if [[ ! -f "$USERS_FILE" ]]; then
    echo "Error: $USERS_FILE not found"
    exit 1
fi

echo ""
echo -e "${CYAN}LDAP Users${NC}"
echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"
printf "  %-26s %-14s %s\n" "EMAIL" "PASSWORD" "GROUPS"
echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"

in_ldap=false
in_dex=false

while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^ldap_users: ]] && { in_ldap=true; in_dex=false; continue; }
    [[ "$line" =~ ^dex_users: ]] && {
        in_ldap=false
        in_dex=true
        echo ""
        echo -e "${CYAN}Dex Static Users${NC}"
        echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"
        printf "  %-26s %s\n" "EMAIL" "PASSWORD"
        echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"
        continue
    }

    if $in_ldap; then
        [[ "$line" =~ email:[[:space:]]*(.+) ]] && email="${BASH_REMATCH[1]}"
        [[ "$line" =~ password:[[:space:]]*(.+) ]] && password="${BASH_REMATCH[1]}"
        [[ "$line" =~ groups:[[:space:]]*(.+) ]] && {
            groups="${BASH_REMATCH[1]}"
            printf "  %-26s %-14s %s\n" "$email" "$password" "$groups"
        }
    fi

    if $in_dex; then
        [[ "$line" =~ email:[[:space:]]*(.+) ]] && email="${BASH_REMATCH[1]}"
        [[ "$line" =~ password:[[:space:]]*(.+) ]] && {
            password="${BASH_REMATCH[1]}"
            printf "  %-26s %s\n" "$email" "$password"
        }
    fi
done < "$USERS_FILE"

echo ""
