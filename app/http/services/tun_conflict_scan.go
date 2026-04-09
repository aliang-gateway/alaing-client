package services

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"aliang.one/nursorgate/app/http/models"
	"aliang.one/nursorgate/app/http/storage"
)

type TunConflictInterface struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	MatchReason string `json:"match_reason,omitempty"`
}

type TunConflictScanResult struct {
	Supported       bool                   `json:"supported"`
	Platform        string                 `json:"platform"`
	HasConflict     bool                   `json:"has_conflict"`
	ShouldPrompt    bool                   `json:"should_prompt"`
	FirstTimePrompt bool                   `json:"first_time_prompt"`
	PromptReason    string                 `json:"prompt_reason,omitempty"`
	Interfaces      []TunConflictInterface `json:"interfaces"`
	Recommendation  string                 `json:"recommendation,omitempty"`
	Warning         string                 `json:"warning,omitempty"`
	ScannedAtUnix   int64                  `json:"scanned_at_unix"`
}

type tunInterfaceSnapshot struct {
	Name        string
	Description string
	Status      string
}

type tunConflictPromptStore interface {
	GetByKey(promptKey string) (*models.UIPromptState, error)
	Upsert(state models.UIPromptState) error
}

const tunConflictPromptKey = "deep_mode_tun_conflict_notice_v1"

var tunConflictPromptStoreFactory = func() tunConflictPromptStore {
	return storage.NewUIPromptStateStore()
}

var execTunConflictCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

var tunInterfaceSnapshotLoader = loadTunInterfaceSnapshots

func ScanTunConflictInterfaces() TunConflictScanResult {
	result := TunConflictScanResult{
		Supported:     true,
		Platform:      runtime.GOOS,
		Interfaces:    make([]TunConflictInterface, 0),
		ScannedAtUnix: time.Now().Unix(),
	}

	snapshots, warning := tunInterfaceSnapshotLoader()
	if warning != "" {
		result.Warning = warning
	}

	result.Interfaces = detectTunConflictInterfaces(snapshots)
	result.HasConflict = len(result.Interfaces) > 0
	if result.HasConflict {
		result.Recommendation = "Switch your VPN away from TUN mode before enabling Deep Mode. Global mode is recommended."
	}

	return result
}

func GetTunConflictPromptStatus() TunConflictScanResult {
	result := ScanTunConflictInterfaces()

	store := tunConflictPromptStoreFactory()
	if store == nil {
		result.ShouldPrompt = true
		result.FirstTimePrompt = true
		result.PromptReason = "first_time_store_unavailable"
		if result.Recommendation == "" {
			result.Recommendation = "Deep Mode uses a TUN device and may conflict with VPN products that also rely on TUN. Switching the VPN to a different mode is recommended."
		}
		return result
	}

	state, err := store.GetByKey(tunConflictPromptKey)
	if err != nil {
		result.ShouldPrompt = true
		result.FirstTimePrompt = true
		result.PromptReason = "first_time_store_error"
		result.Warning = appendTunConflictWarning(result.Warning, fmt.Sprintf("Prompt state lookup failed: %v", err))
		if result.Recommendation == "" {
			result.Recommendation = "Deep Mode uses a TUN device and may conflict with VPN products that also rely on TUN. Switching the VPN to a different mode is recommended."
		}
		return result
	}

	if state == nil {
		result.ShouldPrompt = true
		result.FirstTimePrompt = true
		result.PromptReason = "first_time"
		if result.Recommendation == "" {
			result.Recommendation = "Deep Mode uses a TUN device and may conflict with VPN products that also rely on TUN. Switching the VPN to a different mode is recommended."
		}
		if err := store.Upsert(models.UIPromptState{
			PromptKey: tunConflictPromptKey,
			SeenAt:    time.Now(),
		}); err != nil {
			result.Warning = appendTunConflictWarning(result.Warning, fmt.Sprintf("Failed to persist first-time prompt state: %v", err))
		}
		return result
	}

	if result.HasConflict {
		result.ShouldPrompt = true
		result.PromptReason = "virtual_adapter_detected"
		if result.Recommendation == "" {
			result.Recommendation = "Switch your VPN away from TUN mode before enabling Deep Mode. Global mode is recommended."
		}
	}

	return result
}

func loadTunInterfaceSnapshots() ([]tunInterfaceSnapshot, string) {
	snapshots := snapshotsFromNetInterfaces()

	if runtime.GOOS == "windows" {
		windowsSnapshots, err := loadWindowsTunInterfaceSnapshots()
		if err != nil {
			return dedupeTunInterfaceSnapshots(snapshots), fmt.Sprintf("Windows adapter scan fallback: %v", err)
		}
		snapshots = append(snapshots, windowsSnapshots...)
	}

	return dedupeTunInterfaceSnapshots(snapshots), ""
}

func snapshotsFromNetInterfaces() []tunInterfaceSnapshot {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	snapshots := make([]tunInterfaceSnapshot, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		status := "down"
		switch {
		case iface.Flags&net.FlagRunning != 0:
			status = "running"
		case iface.Flags&net.FlagUp != 0:
			status = "up"
		}

		snapshots = append(snapshots, tunInterfaceSnapshot{
			Name:   strings.TrimSpace(iface.Name),
			Status: status,
		})
	}
	return snapshots
}

func loadWindowsTunInterfaceSnapshots() ([]tunInterfaceSnapshot, error) {
	output, err := execTunConflictCommand(
		"powershell.exe",
		"-NoProfile",
		"-Command",
		"Get-NetAdapter -IncludeHidden | Select-Object Name, InterfaceDescription, Status | ConvertTo-Json -Compress",
	)
	if err != nil {
		return nil, err
	}

	return parseWindowsTunInterfaceSnapshots(output)
}

func parseWindowsTunInterfaceSnapshots(raw []byte) ([]tunInterfaceSnapshot, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	type adapter struct {
		Name                 string `json:"Name"`
		InterfaceDescription string `json:"InterfaceDescription"`
		Status               string `json:"Status"`
	}

	var many []adapter
	if err := json.Unmarshal([]byte(trimmed), &many); err == nil {
		snapshots := make([]tunInterfaceSnapshot, 0, len(many))
		for _, item := range many {
			snapshots = append(snapshots, tunInterfaceSnapshot{
				Name:        strings.TrimSpace(item.Name),
				Description: strings.TrimSpace(item.InterfaceDescription),
				Status:      strings.TrimSpace(strings.ToLower(item.Status)),
			})
		}
		return snapshots, nil
	}

	var one adapter
	if err := json.Unmarshal([]byte(trimmed), &one); err != nil {
		return nil, err
	}

	return []tunInterfaceSnapshot{{
		Name:        strings.TrimSpace(one.Name),
		Description: strings.TrimSpace(one.InterfaceDescription),
		Status:      strings.TrimSpace(strings.ToLower(one.Status)),
	}}, nil
}

func dedupeTunInterfaceSnapshots(items []tunInterfaceSnapshot) []tunInterfaceSnapshot {
	if len(items) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(items))
	deduped := make([]tunInterfaceSnapshot, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		description := strings.TrimSpace(item.Description)
		if name == "" && description == "" {
			continue
		}

		key := strings.ToLower(name) + "\x00" + strings.ToLower(description)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, tunInterfaceSnapshot{
			Name:        name,
			Description: description,
			Status:      strings.TrimSpace(item.Status),
		})
	}
	return deduped
}

func detectTunConflictInterfaces(items []tunInterfaceSnapshot) []TunConflictInterface {
	conflicts := make([]TunConflictInterface, 0)
	for _, item := range items {
		reason := matchTunConflictReason(item.Name, item.Description)
		if reason == "" {
			continue
		}
		conflicts = append(conflicts, TunConflictInterface{
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Status:      strings.TrimSpace(item.Status),
			MatchReason: reason,
		})
	}

	sort.Slice(conflicts, func(i, j int) bool {
		return strings.ToLower(conflicts[i].Name) < strings.ToLower(conflicts[j].Name)
	})
	return conflicts
}

func matchTunConflictReason(name, description string) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerDesc := strings.ToLower(strings.TrimSpace(description))

	switch {
	case strings.HasPrefix(lowerName, "utun"):
		return "utun interface"
	case strings.HasPrefix(lowerName, "wintun") || strings.Contains(lowerDesc, "wintun"):
		return "wintun adapter"
	case strings.HasPrefix(lowerName, "tailscale") || strings.Contains(lowerDesc, "tailscale"):
		return "tailscale tunnel"
	case strings.HasPrefix(lowerName, "wg") || strings.Contains(lowerDesc, "wireguard"):
		return "wireguard adapter"
	case strings.HasPrefix(lowerName, "tun"):
		return "tun adapter"
	case strings.HasPrefix(lowerName, "tap"):
		return "tap adapter"
	case strings.Contains(lowerDesc, "tap-windows"):
		return "tap adapter"
	case strings.Contains(lowerDesc, "openvpn"):
		return "openvpn adapter"
	case strings.Contains(lowerName, "vpn") || strings.Contains(lowerDesc, " vpn"):
		return "vpn virtual adapter"
	case strings.Contains(lowerDesc, "virtual") && (strings.Contains(lowerDesc, "tun") || strings.Contains(lowerDesc, "tunnel")):
		return "virtual tunnel adapter"
	case strings.Contains(lowerDesc, "zerotier") || strings.HasPrefix(lowerName, "zt"):
		return "zerotier virtual adapter"
	default:
		return ""
	}
}

func appendTunConflictWarning(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + " " + next
}
