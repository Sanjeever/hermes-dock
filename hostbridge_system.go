package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type hostProcessInfo struct {
	PID         int32    `json:"pid"`
	ParentPID   int32    `json:"parent_pid"`
	Name        string   `json:"name"`
	Executable  string   `json:"executable"`
	CommandLine []string `json:"command_line"`
	Username    string   `json:"username"`
	StartedAt   string   `json:"started_at,omitempty"`
}

type hostPortInfo struct {
	Protocol      string `json:"protocol"`
	LocalAddress  string `json:"local_address"`
	LocalPort     uint32 `json:"local_port"`
	RemoteAddress string `json:"remote_address"`
	RemotePort    uint32 `json:"remote_port"`
	State         string `json:"state"`
	PID           int32  `json:"pid"`
	ProcessName   string `json:"process_name"`
}

func (a *App) handleHostProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var pidFilter int64
	var err error
	if value := strings.TrimSpace(r.URL.Query().Get("pid")); value != "" {
		pidFilter, err = strconv.ParseInt(value, 10, 32)
		if err != nil || pidFilter <= 0 {
			writeHostError(w, http.StatusBadRequest, fmt.Errorf("pid 无效"))
			return
		}
	}
	nameFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("name")))
	processes, err := process.ProcessesWithContext(r.Context())
	if err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	result := make([]hostProcessInfo, 0, len(processes))
	partial := false
	for _, item := range processes {
		if pidFilter > 0 && item.Pid != int32(pidFilter) {
			continue
		}
		info, incomplete := readHostProcess(r, item)
		partial = partial || incomplete
		if nameFilter != "" && !strings.Contains(strings.ToLower(info.Name), nameFilter) {
			continue
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PID < result[j].PID })
	writeHostJSON(w, http.StatusOK, map[string]interface{}{
		"processes": result,
		"partial":   partial,
	})
}

func readHostProcess(r *http.Request, item *process.Process) (hostProcessInfo, bool) {
	ctx := r.Context()
	result := hostProcessInfo{PID: item.Pid, CommandLine: []string{}}
	partial := false
	var err error
	if result.ParentPID, err = item.PpidWithContext(ctx); err != nil {
		partial = true
	}
	if result.Name, err = item.NameWithContext(ctx); err != nil {
		partial = true
	}
	if result.Executable, err = item.ExeWithContext(ctx); err != nil {
		partial = true
	}
	if result.CommandLine, err = item.CmdlineSliceWithContext(ctx); err != nil {
		result.CommandLine = []string{}
		partial = true
	}
	if result.Username, err = item.UsernameWithContext(ctx); err != nil {
		partial = true
	}
	started, err := item.CreateTimeWithContext(ctx)
	if err != nil {
		partial = true
	} else if started > 0 {
		result.StartedAt = time.UnixMilli(started).UTC().Format(time.RFC3339)
	}
	return result, partial
}

func (a *App) handleHostPorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var portFilter uint64
	var err error
	if value := strings.TrimSpace(r.URL.Query().Get("port")); value != "" {
		portFilter, err = strconv.ParseUint(value, 10, 16)
		if err != nil || portFilter == 0 {
			writeHostError(w, http.StatusBadRequest, fmt.Errorf("端口无效"))
			return
		}
	}
	listeningOnly := strings.EqualFold(r.URL.Query().Get("listening"), "true")
	connections, err := gopsnet.ConnectionsWithContext(r.Context(), "inet")
	if err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	processNames := map[int32]string{}
	partial := false
	result := make([]hostPortInfo, 0, len(connections))
	for _, connection := range connections {
		if portFilter > 0 && connection.Laddr.Port != uint32(portFilter) {
			continue
		}
		if listeningOnly && !strings.EqualFold(connection.Status, "LISTEN") {
			continue
		}
		protocol := "tcp"
		if connection.Type == 2 {
			protocol = "udp"
		}
		name := processNames[connection.Pid]
		if connection.Pid > 0 && name == "" {
			item, err := process.NewProcessWithContext(r.Context(), connection.Pid)
			if err == nil {
				name, err = item.NameWithContext(r.Context())
			}
			if err != nil {
				partial = true
			}
			processNames[connection.Pid] = name
		}
		result = append(result, hostPortInfo{
			Protocol:      protocol,
			LocalAddress:  connection.Laddr.IP,
			LocalPort:     connection.Laddr.Port,
			RemoteAddress: connection.Raddr.IP,
			RemotePort:    connection.Raddr.Port,
			State:         connection.Status,
			PID:           connection.Pid,
			ProcessName:   name,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LocalPort == result[j].LocalPort {
			return result[i].PID < result[j].PID
		}
		return result[i].LocalPort < result[j].LocalPort
	})
	writeHostJSON(w, http.StatusOK, map[string]interface{}{
		"connections": result,
		"partial":     partial,
	})
}
