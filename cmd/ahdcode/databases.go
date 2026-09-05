package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ahdcode/internal/initweb"
)

func runDatabases(arguments []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(arguments) != 0 {
		fmt.Fprintln(errorOutput, "ahdcode databases: no arguments are accepted")
		return 2
	}
	dir, err := initweb.LocateAhdDataStudio()
	if err != nil {
		fmt.Fprintln(errorOutput, err.Error())
		return 1
	}
	if err := ensureStudioEnv(dir); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(errorOutput, "ahdcode databases: %v\n", err)
		return 1
	}
	defer func() { _ = os.Chdir(cwd) }()

	fmt.Fprintf(output, "AhdDataStudio: %s\n", initweb.AhdDataStudioPublicURL())
	openURL := initweb.AhdDataStudioPublicURL()
	if !studioTestNameInHosts() {
		fmt.Fprintf(output, "Add this line to /etc/hosts (once):\n  127.0.0.1 %s\n", initweb.AhdDataStudioHost)
		openURL = initweb.AhdDataStudioLoopbackURL()
		fmt.Fprintf(output, "Using: %s\n", openURL)
	}
	go openStudioLater(openURL)
	return runRun([]string{"app.ahd"}, input, output, errorOutput)
}

// Read local configuration only: DNS for an unconfigured .test can stall startup.
func studioTestNameInHosts() bool {
	path := "/etc/hosts"
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			return false
		}
		path = filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	return studioHostsMapsLoopback(file)
}

func studioHostsMapsLoopback(input io.Reader) bool {
	found := false
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line, _, _ := strings.Cut(scanner.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, host := range fields[1:] {
			if strings.EqualFold(strings.TrimSuffix(host, "."), initweb.AhdDataStudioHost) {
				// Studio listens on IPv4 only. Conflicting or IPv6 mappings use fallback.
				if fields[0] != "127.0.0.1" {
					return false
				}
				found = true
			}
		}
	}
	return found && scanner.Err() == nil
}

func openStudioLater(rawURL string) {
	time.Sleep(400 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	_ = cmd.Start()
}

func ensureStudioEnv(dir string) error {
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	example := filepath.Join(dir, ".env.example")
	data, err := os.ReadFile(example)
	if err != nil {
		return nil
	}
	return os.WriteFile(envPath, data, 0o600)
}
