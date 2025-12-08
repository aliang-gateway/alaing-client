package rules

import (
	"fmt"
	"net/netip"
	"strings"

	"nursor.org/nursorgate/processor/config"
)

// BypassRules holds bypass rule configurations for direct routing
type BypassRules struct {
	domainExact    map[string]bool // Exact domain matches
	domainWildcard []string        // Wildcard patterns (e.g., *.example.com)
	domainSuffix   []string        // Domain suffix matches (e.g., .cn)
	ipRanges       []netip.Prefix  // CIDR IP ranges
	enabled        bool            // Whether bypass rules are enabled
}

// NewBypassRules creates a new bypass rules matcher from configuration
func NewBypassRules(config *config.BypassRulesConfig) (*BypassRules, error) {
	if config == nil || !config.Enabled {
		return &BypassRules{enabled: false}, nil
	}

	// Parse CIDR ranges
	prefixes := make([]netip.Prefix, 0, len(config.IPRanges))
	for _, cidr := range config.IPRanges {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
		}
		prefixes = append(prefixes, prefix)
	}

	// Categorize domain rules
	exactMap := make(map[string]bool)
	wildcards := make([]string, 0)

	for _, domain := range config.Domains {
		if strings.Contains(domain, "*") {
			// Wildcard pattern
			wildcards = append(wildcards, strings.ToLower(domain))
		} else {
			// Exact match
			exactMap[strings.ToLower(domain)] = true
		}
	}

	// Normalize domain suffixes
	suffixes := make([]string, len(config.DomainSuffixes))
	for i, suffix := range config.DomainSuffixes {
		suffixes[i] = strings.ToLower(suffix)
	}

	return &BypassRules{
		domainExact:    exactMap,
		domainWildcard: wildcards,
		domainSuffix:   suffixes,
		ipRanges:       prefixes,
		enabled:        true,
	}, nil
}

// MatchDomain checks if a domain matches any bypass rule
func (b *BypassRules) MatchDomain(domain string) bool {
	if !b.enabled || domain == "" {
		return false
	}

	domain = strings.ToLower(domain)

	// Check exact match
	if b.domainExact[domain] {
		return true
	}

	// Check suffix match
	for _, suffix := range b.domainSuffix {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}

	// Check wildcard match
	for _, pattern := range b.domainWildcard {
		if matchWildcard(pattern, domain) {
			return true
		}
	}

	return false
}

// MatchIP checks if an IP address matches any bypass IP range
func (b *BypassRules) MatchIP(ip netip.Addr) bool {
	if !b.enabled {
		return false
	}

	for _, prefix := range b.ipRanges {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// IsEnabled returns whether bypass rules are active
func (b *BypassRules) IsEnabled() bool {
	return b.enabled
}

// matchWildcard performs simple wildcard pattern matching
// Supports patterns like *.example.com
func matchWildcard(pattern, domain string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == domain
	}

	// Handle *.example.com pattern
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:] // Remove "*."

		// Match both "example.com" and "sub.example.com"
		if domain == suffix {
			return true
		}
		if strings.HasSuffix(domain, "."+suffix) {
			return true
		}
		return false
	}

	// Handle example.* pattern
	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2] // Remove ".*"

		if domain == prefix {
			return true
		}
		if strings.HasPrefix(domain, prefix+".") {
			return true
		}
		return false
	}

	// Handle *example* pattern (contains)
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		middle := pattern[1 : len(pattern)-1]
		return strings.Contains(domain, middle)
	}

	// Handle *example pattern (suffix)
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(domain, suffix)
	}

	// Handle example* pattern (prefix)
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(domain, prefix)
	}

	return false
}
