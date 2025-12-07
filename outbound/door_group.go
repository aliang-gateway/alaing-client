package outbound

import (
	"fmt"
	"sync"
	"time"

	"nursor.org/nursorgate/outbound/proxy"
)

// DoorProxyMemberInfo represents runtime information about a door proxy member
type DoorProxyMemberInfo struct {
	ShowName   string
	Proxy      proxy.Proxy
	Latency    int64 // 延迟（毫秒）
	LastUpdate int64 // 最后更新时间戳
}

// DoorProxyGroup manages a collection of proxies for the door
type DoorProxyGroup struct {
	mu            sync.RWMutex
	members       map[string]*DoorProxyMemberInfo
	currentMember string // 手动指定的成员名称
	autoSelect    bool   // 是否自动选择最佳节点
}

// NewDoorProxyGroup creates a new door proxy group
func NewDoorProxyGroup() *DoorProxyGroup {
	return &DoorProxyGroup{
		members:    make(map[string]*DoorProxyMemberInfo),
		autoSelect: true, // 默认启用自动选择
	}
}

// AddMember adds a member to the door proxy group
func (dpg *DoorProxyGroup) AddMember(showName string, proxy proxy.Proxy, latency int64) error {
	if showName == "" {
		return fmt.Errorf("showName cannot be empty")
	}
	if proxy == nil {
		return fmt.Errorf("proxy cannot be nil")
	}

	dpg.mu.Lock()
	defer dpg.mu.Unlock()

	dpg.members[showName] = &DoorProxyMemberInfo{
		ShowName:   showName,
		Proxy:      proxy,
		Latency:    latency,
		LastUpdate: time.Now().Unix(),
	}

	return nil
}

// GetMember returns a specific member by name
func (dpg *DoorProxyGroup) GetMember(showName string) (proxy.Proxy, error) {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	member, exists := dpg.members[showName]
	if !exists {
		return nil, fmt.Errorf("door member '%s' not found", showName)
	}

	return member.Proxy, nil
}

// GetBestMember returns the member with the lowest latency
func (dpg *DoorProxyGroup) GetBestMember() (proxy.Proxy, error) {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	if len(dpg.members) == 0 {
		return nil, fmt.Errorf("no members in door proxy group")
	}

	var bestMember *DoorProxyMemberInfo
	var lowestLatency int64 = -1

	for _, member := range dpg.members {
		if lowestLatency == -1 || member.Latency < lowestLatency {
			lowestLatency = member.Latency
			bestMember = member
		}
	}

	if bestMember == nil {
		return nil, fmt.Errorf("failed to find best member")
	}

	return bestMember.Proxy, nil
}

// GetCurrentOrBest returns the current manually selected member, or the best member if auto-select is enabled
func (dpg *DoorProxyGroup) GetCurrentOrBest() (proxy.Proxy, error) {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	// If a specific member is manually selected and exists, return it
	if dpg.currentMember != "" {
		if member, exists := dpg.members[dpg.currentMember]; exists {
			return member.Proxy, nil
		}
	}

	// Otherwise, find the best member (lowest latency)
	if len(dpg.members) == 0 {
		return nil, fmt.Errorf("no members in door proxy group")
	}

	var bestMember *DoorProxyMemberInfo
	var lowestLatency int64 = -1

	for _, member := range dpg.members {
		if lowestLatency == -1 || member.Latency < lowestLatency {
			lowestLatency = member.Latency
			bestMember = member
		}
	}

	if bestMember == nil {
		return nil, fmt.Errorf("failed to find best member")
	}

	return bestMember.Proxy, nil
}

// SetCurrentMember manually selects a specific member
func (dpg *DoorProxyGroup) SetCurrentMember(showName string) error {
	dpg.mu.Lock()
	defer dpg.mu.Unlock()

	if showName == "" {
		// Empty string means enable auto-select
		dpg.currentMember = ""
		dpg.autoSelect = true
		return nil
	}

	if _, exists := dpg.members[showName]; !exists {
		return fmt.Errorf("door member '%s' not found", showName)
	}

	dpg.currentMember = showName
	dpg.autoSelect = false
	return nil
}

// EnableAutoSelect enables automatic selection of the best member
func (dpg *DoorProxyGroup) EnableAutoSelect() {
	dpg.mu.Lock()
	defer dpg.mu.Unlock()

	dpg.currentMember = ""
	dpg.autoSelect = true
}

// UpdateLatency updates the latency for a specific member
func (dpg *DoorProxyGroup) UpdateLatency(showName string, latency int64) error {
	dpg.mu.Lock()
	defer dpg.mu.Unlock()

	member, exists := dpg.members[showName]
	if !exists {
		return fmt.Errorf("door member '%s' not found", showName)
	}

	member.Latency = latency
	member.LastUpdate = time.Now().Unix()
	return nil
}

// ListMembers returns information about all members
func (dpg *DoorProxyGroup) ListMembers() []DoorProxyMemberInfo {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	members := make([]DoorProxyMemberInfo, 0, len(dpg.members))
	for _, member := range dpg.members {
		members = append(members, *member)
	}

	return members
}

// GetCurrentMemberName returns the name of the currently selected member, or empty string if auto-select
func (dpg *DoorProxyGroup) GetCurrentMemberName() string {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	return dpg.currentMember
}

// IsAutoSelect returns whether auto-select is enabled
func (dpg *DoorProxyGroup) IsAutoSelect() bool {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	return dpg.autoSelect
}

// Count returns the number of members in the group
func (dpg *DoorProxyGroup) Count() int {
	dpg.mu.RLock()
	defer dpg.mu.RUnlock()

	return len(dpg.members)
}
