package utils

import "strings"

func GetWildcardDomain(domain string) string {
	var wildcardDomain string
	domainParts := strings.Split(domain, ".")
	for i, dp := range domainParts {
		if i == 0 {
			wildcardDomain = "*"
			continue
		}
		wildcardDomain = wildcardDomain + "." + dp
	}

	return wildcardDomain
}
