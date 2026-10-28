package procmon

import (
"fmt"
"os"
"path/filepath"
"strconv"
"strings"
"time"
)

type AlertCallback func(msg string)

// StartMonitor watches open file descriptors (/proc/<pid>/fd) of the target process in real time.
func StartMonitor(pid int, blockedPaths []string, stopCh <-chan struct{}, onAlert AlertCallback) {
procFdDir := filepath.Join("/proc", strconv.Itoa(pid), "fd")
if _, err := os.Stat(procFdDir); err != nil {
return
}

ticker := time.NewTicker(120 * time.Millisecond)
defer ticker.Stop()

seen := make(map[string]bool)

for {
select {
case <-stopCh:
return
case <-ticker.C:
entries, err := os.ReadDir(procFdDir)
if err != nil {
return
}
for _, entry := range entries {
fdPath := filepath.Join(procFdDir, entry.Name())
target, err := os.Readlink(fdPath)
if err != nil || seen[target] {
continue
}
for _, restricted := range blockedPaths {
if strings.Contains(strings.ToLower(target), strings.ToLower(restricted)) {
seen[target] = true
onAlert(fmt.Sprintf("OS-Level FD Access Violation: PID %d opened restricted path '%s'", pid, target))
}
}
}
}
}
}