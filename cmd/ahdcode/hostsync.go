package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"ahdcode/internal/localdev"
)

type hostSyncResult struct {
	mapped   bool
	mutated  bool
	message  string
	refused  bool
	nonTTY   bool
}

type hostSyncRequest struct {
	hostname    string
	path        string
	input       io.Reader
	output      io.Writer
	errorOutput io.Writer
	interactive bool
}

// syncManagedHostname makes hostname resolve to loopback through AhdCode's
// managed hosts block when the user has authorized that, and never as a
// silent system change on first use.
func syncManagedHostname(request hostSyncRequest) hostSyncResult {
	if request.path == "" {
		request.path = localdev.SystemHostsPath()
	}
	if request.output == nil {
		request.output = io.Discard
	}
	if request.errorOutput == nil {
		request.errorOutput = io.Discard
	}
	if !localdev.ValidHostname(request.hostname) {
		return hostSyncResult{message: "the local hostname is not valid"}
	}

	current, err := readHostsFile(request.path)
	if err != nil {
		return hostSyncResult{message: fmt.Sprintf("the hosts file could not be read (%v)", err)}
	}
	if localdev.HostsMapsLoopback(current, request.hostname) {
		return hostSyncResult{mapped: true}
	}

	existing := localdev.HostsBlockNames(current)
	updated := localdev.ApplyHostsBlock(current, localdev.RenderHostsBlock(mergedHostnames(existing, []string{request.hostname, localdev.StudioHost})))
	if updated == current {
		if localdev.HostsMapsLoopback(updated, request.hostname) {
			return hostSyncResult{mapped: true}
		}
		return hostSyncResult{message: request.hostname + " could not be added to the managed hosts block"}
	}

	authorized := localdev.HostsAuthorized() || len(existing) > 0
	if !authorized {
		if !request.interactive {
			return hostSyncResult{
				nonTTY:  true,
				message: request.hostname + " is not mapped to 127.0.0.1; the direct bind address still works",
			}
		}
		fmt.Fprintln(request.output)
		fmt.Fprintln(request.output, "AhdCode can enable local .test names on this computer.")
		fmt.Fprintln(request.output, "This only manages AhdCode-owned loopback host entries.")
		fmt.Fprintln(request.output)
		if !confirmYesDefault(request.input, request.output, "Enable local .test names? [Y/n] ", true) {
			return hostSyncResult{
				refused: true,
				message: request.hostname + " was not added; the direct bind address still works",
			}
		}
		if err := localdev.MarkHostsAuthorized(); err != nil {
			return hostSyncResult{message: fmt.Sprintf("local host authorization could not be recorded (%v)", err)}
		}
		authorized = true
	}

	if err := replaceFileContents(request.path, updated); err == nil {
		_ = localdev.MarkHostsAuthorized()
		return hostSyncResult{mapped: true, mutated: true}
	}
	if !request.interactive {
		return hostSyncResult{
			nonTTY:  true,
			message: request.hostname + " could not be mapped without administrator access; the direct bind address still works",
		}
	}
	if request.path == localdev.SystemHostsPath() {
		staged, stageErr := stageHostsFile(updated)
		if stageErr == nil {
			defer func() { _ = os.Remove(staged) }()
			if err := elevatedReplace(staged, request.path); err == nil {
				_ = localdev.MarkHostsAuthorized()
				return hostSyncResult{mapped: true, mutated: true}
			}
		}
	}
	return hostSyncResult{
		message: request.hostname + " needs administrator access to update " + request.path + "; the direct bind address still works",
	}
}

func readHostsFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(content), nil
}

func confirmYesDefault(input io.Reader, output io.Writer, prompt string, defaultYes bool) bool {
	fmt.Fprint(output, prompt)
	if input == nil {
		return defaultYes
	}
	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return defaultYes
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "" {
		return defaultYes
	}
	if answer == "y" || answer == "yes" {
		return true
	}
	if answer == "n" || answer == "no" {
		return false
	}
	return defaultYes
}

func hostIntegrationState(content string, authorized bool) string {
	managed := localdev.HostsBlockNames(content)
	if len(managed) > 0 {
		return "enabled"
	}
	if authorized {
		return "unavailable"
	}
	return "not authorized"
}
