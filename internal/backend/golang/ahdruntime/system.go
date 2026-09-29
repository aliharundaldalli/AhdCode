package ahdruntime

// The v2.7.0 read-only system-inspection primitives: Disk.inspect (capacity
// of the filesystem that contains a path) and Service.status (state of a
// Linux systemd unit). Built only on the Go standard library, emitted
// verbatim into native programs, and called directly by the evaluator.
//
// Neither runs a shell. Disk uses the operating system's filesystem API
// (statfs, GetDiskFreeSpaceExW; see disk_*.go). Service runs the absolute
// systemctl executable through ProcessRun -- executable plus argument list,
// bounded in time and output -- and reads the machine-readable
// `systemctl show` properties, never the human-facing `systemctl status`.
// Both are observation only: nothing here starts, stops, or changes anything.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// --- Disk ----------------------------------------------------------------

// ahdDiskUsage is one filesystem reading in bytes, as the platform reports it.
type ahdDiskUsage struct {
	Total, Free, Available uint64
}

type ahdDiskInfoData struct {
	Path      string `json:"path"`
	Total     int64  `json:"total"`
	Used      int64  `json:"used"`
	Free      int64  `json:"free"`
	Available int64  `json:"available"`
}

func ahdDiskFail(path, reason string) error {
	return errors.New("inspect disk of " + strconv.Quote(path) + " failed: " + reason)
}

// ahdDiskBytes multiplies a block count by a block size, refusing a product
// that an AhdCode Int (int64) cannot hold instead of wrapping around.
func ahdDiskBytes(blocks, size uint64) (uint64, bool) {
	if size != 0 && blocks > math.MaxInt64/size {
		return 0, false
	}
	return blocks * size, true
}

// DiskInspectAs reports the capacity of the filesystem that contains
// resolved; display is the path exactly as the program wrote it, used in the
// result and in messages. The path need not be a mount point, but it must
// exist.
func DiskInspectAs(resolved, display string) (string, error) {
	switch {
	case display == "":
		return "", ahdDiskFail(display, "the path is empty")
	case strings.ContainsRune(display, 0):
		return "", ahdDiskFail(display, "the path contains a NUL byte")
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", ahdDiskFail(display, ahdFSReason(err))
	}
	usage, err := ahdDiskUsageOf(resolved)
	if err != nil {
		return "", ahdDiskFail(display, err.Error())
	}
	if usage.Total > math.MaxInt64 || usage.Free > usage.Total || usage.Available > usage.Free {
		return "", ahdDiskFail(display, "the filesystem reported inconsistent sizes")
	}
	encoded, err := json.Marshal(ahdDiskInfoData{
		Path: display, Total: int64(usage.Total), Used: int64(usage.Total - usage.Free),
		Free: int64(usage.Free), Available: int64(usage.Available),
	})
	if err != nil {
		return "", ahdDiskFail(display, err.Error())
	}
	return string(encoded), nil
}

// DiskInspect is DiskInspectAs for a path that needs no resolution.
func DiskInspect(path string) (string, error) { return DiskInspectAs(path, path) }

func ahdDiskDecode(data string) (ahdDiskInfoData, error) {
	var info ahdDiskInfoData
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return info, errors.New("DiskInfo storage is corrupted")
	}
	return info, nil
}

func DiskInfoPath(data string) (string, error) {
	info, err := ahdDiskDecode(data)
	return info.Path, err
}

func DiskInfoTotalBytes(data string) (int64, error) {
	info, err := ahdDiskDecode(data)
	return info.Total, err
}

func DiskInfoUsedBytes(data string) (int64, error) {
	info, err := ahdDiskDecode(data)
	return info.Used, err
}

func DiskInfoFreeBytes(data string) (int64, error) {
	info, err := ahdDiskDecode(data)
	return info.Free, err
}

func DiskInfoAvailableBytes(data string) (int64, error) {
	info, err := ahdDiskDecode(data)
	return info.Available, err
}

// DiskInfoUsedPercent is usedBytes / totalBytes * 100, and 0 for a filesystem
// that reports no capacity at all.
func DiskInfoUsedPercent(data string) (float64, error) {
	info, err := ahdDiskDecode(data)
	if err != nil || info.Total == 0 {
		return 0, err
	}
	return float64(info.Used) / float64(info.Total) * 100, nil
}

// --- Service -------------------------------------------------------------

const (
	ahdServiceTimeoutSeconds = int64(10)
	ahdServiceMaxOutputBytes = int64(64 * 1024)
	ahdServiceMaxNameLength  = 255
)

// ahdServicePlatform and ahdServiceShow are the test seams: production uses
// the real operating system and runs systemctl; tests substitute both. They
// are unexported, so an AhdCode program can never reach them.
var ahdServicePlatform = runtime.GOOS

// ahdServiceShow runs `systemctl show` for one unit and returns its exit code
// and standard output/error. It reports a launch problem (including a missing
// systemctl) as an error.
var ahdServiceShow = func(unit string) (int64, string, string, error) {
	command := ""
	for _, candidate := range []string{"/usr/bin/systemctl", "/bin/systemctl"} {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			command = candidate
			break
		}
	}
	if command == "" {
		return 0, "", "", errors.New("systemd is not available on this system (systemctl was not found)")
	}
	data, err := ProcessRun(command, ahdServiceArguments(unit), ahdServiceTimeoutSeconds, ahdServiceMaxOutputBytes, "")
	if err != nil {
		if strings.Contains(err.Error(), "timed out") {
			return 0, "", "", fmt.Errorf("the inspection timed out after %d seconds", ahdServiceTimeoutSeconds)
		}
		return 0, "", "", errors.New("systemctl could not be run")
	}
	result, err := ahdProcessDecode(data)
	if err != nil {
		return 0, "", "", err
	}
	return result.ExitCode, result.Stdout, result.Stderr, nil
}

// ahdServiceArguments is the exact argument list: machine-readable properties
// only, and `--` so a unit name can never be read as an option.
func ahdServiceArguments(unit string) []string {
	return []string{"show", "--no-pager", "--property=Id,LoadState,ActiveState,SubState,UnitFileState", "--", unit}
}

type ahdServiceInfoData struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	SubState string `json:"subState"`
	Enabled  bool   `json:"enabled"`
}

func ahdServiceFail(name, reason string) error {
	return errors.New("service " + strconv.Quote(name) + " status failed: " + reason)
}

// ahdServiceValidName accepts systemd unit-name characters only: ASCII
// letters, digits, and ":_.@-", at most 255 characters, not starting with "-"
// or ".". Anything else -- spaces, slashes, quotes, ";", "$", control
// characters -- is refused before any process runs.
func ahdServiceValidName(name string) string {
	switch {
	case name == "":
		return "the service name is empty"
	case len(name) > ahdServiceMaxNameLength:
		return fmt.Sprintf("the service name is longer than %d characters", ahdServiceMaxNameLength)
	case name[0] == '-' || name[0] == '.':
		return "the service name must not start with '-' or '.'"
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(":_.@-", r)) {
			if r < 0x20 || r == 0x7f {
				return "the service name contains a control character"
			}
			return "the service name contains " + strconv.QuoteRune(r) + "; only letters, digits, and ':_.@-' are allowed"
		}
	}
	return ""
}

// ahdServiceState normalizes systemd's ActiveState to the documented
// vocabulary: active, inactive, failed, activating, deactivating, unknown.
func ahdServiceState(active string) string {
	switch active {
	case "active", "reloading", "refreshing":
		return "active"
	case "inactive", "failed", "activating", "deactivating":
		return active
	}
	return "unknown"
}

// ahdServiceParse turns `systemctl show` KEY=VALUE output into ServiceInfo
// data. It is separate from invocation so it can be tested with fixtures.
func ahdServiceParse(name, output string) (ahdServiceInfoData, error) {
	properties := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		key, value, found := strings.Cut(strings.TrimRight(line, "\r"), "=")
		if found {
			properties[key] = value
		}
	}
	load, known := properties["LoadState"]
	if !known {
		return ahdServiceInfoData{}, ahdServiceFail(name, "systemd returned no unit information")
	}
	if load == "not-found" {
		return ahdServiceInfoData{}, ahdServiceFail(name, "service not found")
	}
	subState := properties["SubState"]
	for _, r := range subState {
		if !(r >= 'a' && r <= 'z' || r == '-') {
			subState = "unknown"
			break
		}
	}
	if subState == "" {
		subState = "unknown"
	}
	unitFile := properties["UnitFileState"]
	id := properties["Id"]
	if id == "" || ahdServiceValidName(id) != "" {
		id = name
	}
	return ahdServiceInfoData{
		Name: id, State: ahdServiceState(properties["ActiveState"]), SubState: subState,
		Enabled: unitFile == "enabled" || unitFile == "enabled-runtime",
	}, nil
}

// ServiceStatus reports the state of one systemd unit. It is supported on
// Linux with systemd; elsewhere it fails clearly instead of guessing.
func ServiceStatus(name string) (string, error) {
	if reason := ahdServiceValidName(name); reason != "" {
		return "", ahdServiceFail(name, reason)
	}
	if ahdServicePlatform != "linux" {
		return "", ahdServiceFail(name, "service inspection is supported only on Linux with systemd; this system is "+ahdServicePlatform)
	}
	exitCode, stdout, stderr, err := ahdServiceShow(name)
	if err != nil {
		return "", ahdServiceFail(name, err.Error())
	}
	if exitCode != 0 {
		lower := strings.ToLower(stderr)
		switch {
		case strings.Contains(lower, "not been booted with systemd") || strings.Contains(lower, "failed to connect to bus"):
			return "", ahdServiceFail(name, "systemd is not available on this system")
		case strings.Contains(lower, "access denied") || strings.Contains(lower, "permission denied"):
			return "", ahdServiceFail(name, "permission denied")
		}
		return "", ahdServiceFail(name, fmt.Sprintf("systemctl reported an error (exit code %d)", exitCode))
	}
	info, err := ahdServiceParse(name, stdout)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		return "", ahdServiceFail(name, err.Error())
	}
	return string(encoded), nil
}

func ahdServiceDecode(data string) (ahdServiceInfoData, error) {
	var info ahdServiceInfoData
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return info, errors.New("ServiceInfo storage is corrupted")
	}
	return info, nil
}

func ServiceInfoName(data string) (string, error) {
	info, err := ahdServiceDecode(data)
	return info.Name, err
}

func ServiceInfoActiveState(data string) (string, error) {
	info, err := ahdServiceDecode(data)
	return info.State, err
}

func ServiceInfoSubState(data string) (string, error) {
	info, err := ahdServiceDecode(data)
	return info.SubState, err
}

// ServiceInfoRunning is true when the unit's processes are running
// (SubState "running"); an active one-shot unit that has exited is not.
func ServiceInfoRunning(data string) (bool, error) {
	info, err := ahdServiceDecode(data)
	return info.SubState == "running", err
}

func ServiceInfoEnabled(data string) (bool, error) {
	info, err := ahdServiceDecode(data)
	return info.Enabled, err
}

// --- native wrappers -------------------------------------------------------

func ahdSystemRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdDiskInspect(class *AhdClass, path string) string {
	data, err := DiskInspect(path)
	ahdSystemRaise(class, err)
	return data
}

func AhdServiceStatus(class *AhdClass, name string) string {
	data, err := ServiceStatus(name)
	ahdSystemRaise(class, err)
	return data
}

func AhdSystemString(class *AhdClass, data string, read func(string) (string, error)) string {
	value, err := read(data)
	ahdSystemRaise(class, err)
	return value
}

func AhdSystemInt(class *AhdClass, data string, read func(string) (int64, error)) int64 {
	value, err := read(data)
	ahdSystemRaise(class, err)
	return value
}

func AhdSystemBool(class *AhdClass, data string, read func(string) (bool, error)) bool {
	value, err := read(data)
	ahdSystemRaise(class, err)
	return value
}

func AhdSystemReal(class *AhdClass, data string, read func(string) (float64, error)) float64 {
	value, err := read(data)
	ahdSystemRaise(class, err)
	return value
}
