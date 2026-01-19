#!/bin/bash
# Generates LDAP configmap.yaml from dev-users.yaml

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
USERS_FILE="${SCRIPT_DIR}/../dev-users.yaml"
OUTPUT_FILE="${SCRIPT_DIR}/configmap.yaml"

if [[ ! -f "$USERS_FILE" ]]; then
    echo "Error: dev-users.yaml not found"
    exit 1
fi

# Temp files for collecting data
TMP_USERS=$(mktemp)
TMP_GROUPS=$(mktemp)
trap "rm -f $TMP_USERS $TMP_GROUPS" EXIT

# Parse users.yaml
in_ldap=false
while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^ldap_users: ]] && { in_ldap=true; continue; }
    [[ "$line" =~ ^dex_users: ]] && { in_ldap=false; continue; }

    if $in_ldap; then
        [[ "$line" =~ ^[[:space:]]*-[[:space:]]*cn:[[:space:]]*(.+) ]] && cn="${BASH_REMATCH[1]}"
        [[ "$line" =~ ^[[:space:]]*email:[[:space:]]*(.+) ]] && email="${BASH_REMATCH[1]}"
        [[ "$line" =~ ^[[:space:]]*password:[[:space:]]*(.+) ]] && password="${BASH_REMATCH[1]}"
        [[ "$line" =~ ^[[:space:]]*groups:[[:space:]]*(.+) ]] && {
            groups="${BASH_REMATCH[1]}"
            echo "${cn}|${email}|${password}|${groups}" >> "$TMP_USERS"
        }
    fi
done < "$USERS_FILE"

# Generate configmap
cat > "$OUTPUT_FILE" << 'HEADER'
# Auto-generated from dev-users.yaml - do not edit manually
# Run: make dev-auth-generate
apiVersion: v1
kind: ConfigMap
metadata:
  name: ldap-configmap
  labels:
    app: ldap
data:
  config-ldap.ldif: |-
    dn: ou=People,dc=example,dc=org
    objectClass: organizationalUnit
    ou: People

HEADER

# Generate user entries
while IFS='|' read -r cn email password groups; do
    cat >> "$OUTPUT_FILE" << EOF
    dn: cn=${cn},ou=People,dc=example,dc=org
    objectClass: person
    objectClass: inetOrgPerson
    sn: ${cn}
    cn: ${cn}
    mail: ${email}
    userpassword: ${password}

EOF
    # Add each group for this user (supports comma-separated groups)
    IFS=',' read -ra group_list <<< "$groups"
    for group in "${group_list[@]}"; do
        group=$(echo "$group" | xargs)  # trim whitespace
        echo "${group}|${cn}" >> "$TMP_GROUPS"
    done
done < "$TMP_USERS"

# Generate groups section
cat >> "$OUTPUT_FILE" << 'GROUPS_HEADER'
    # Group definitions
    dn: ou=Groups,dc=example,dc=org
    objectClass: organizationalUnit
    ou: Groups

GROUPS_HEADER

# Get unique groups and generate entries
cut -d'|' -f1 "$TMP_GROUPS" | sort -u | while read -r group; do
    cat >> "$OUTPUT_FILE" << EOF
    dn: cn=${group},ou=Groups,dc=example,dc=org
    objectClass: groupOfNames
    cn: ${group}
EOF
    # Add members for this group
    grep "^${group}|" "$TMP_GROUPS" | cut -d'|' -f2 | while read -r cn; do
        echo "    member: cn=${cn},ou=People,dc=example,dc=org" >> "$OUTPUT_FILE"
    done
    echo "" >> "$OUTPUT_FILE"
done

echo "Generated: $OUTPUT_FILE"
