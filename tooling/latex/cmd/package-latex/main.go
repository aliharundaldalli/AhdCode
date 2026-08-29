// Command package-latex reproducibly stages AhdCode's shared offline Latex
// runtime. It downloads only pinned, checksummed release/build inputs; ordinary
// AhdCode execution never calls this tool and never downloads resources.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const hardLatexPayloadLimit = int64(250 * 1024 * 1024)

type assetsManifest struct {
	Schema             int           `json:"schema"`
	TectonicVersion    string        `json:"tectonic_version"`
	TectonicReleaseURL string        `json:"tectonic_release_url"`
	Engines            []engineAsset `json:"engines"`
	Bundle             bundleAsset   `json:"bundle"`
}

type engineAsset struct {
	GOOS, GOARCH string
	Archive      string `json:"archive"`
	Filename     string `json:"filename"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	SHA256       string `json:"sha256"`
}

type bundleAsset struct {
	Profile               string `json:"profile"`
	SourceURL             string `json:"source_url"`
	SourceSize            int64  `json:"source_size"`
	SourceContentIdentity string `json:"source_content_identity"`
	ContentSHA256         string `json:"content_sha256"`
	TTBSHA256             string `json:"ttb_sha256"`
	TTBSize               int64  `json:"ttb_size"`
	ResourceManifest      string `json:"resource_manifest"`
}

type resource struct {
	Name   string `json:"name"`
	Offset int64  `json:"offset"`
	Length int64  `json:"length"`
	SHA256 string `json:"sha256"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "capture" {
		if err := capture(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "package-latex capture:", err)
			os.Exit(1)
		}
		return
	}
	if err := packageRuntime(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "package-latex:", err)
		os.Exit(1)
	}
}

func packageRuntime(arguments []string) error {
	flags := flag.NewFlagSet("package-latex", flag.ContinueOnError)
	assetsPath := flags.String("manifest", "tooling/latex/assets.json", "asset manifest")
	output := flags.String("output", "", "distribution staging directory")
	compiler := flags.String("compiler", "", "optional ahdcode compiler binary to stage")
	engineArchive := flags.String("engine-archive", "", "optional already-downloaded pinned engine archive")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *output == "" {
		return errors.New("--output is required")
	}
	manifest, err := readAssets(*assetsPath)
	if err != nil {
		return err
	}
	asset, err := selectEngine(manifest.Engines, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	resourcesPath := filepath.Join(filepath.Dir(*assetsPath), manifest.Bundle.ResourceManifest)
	resources, err := readResources(resourcesPath)
	if err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "ahdcode-latex-package-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	archivePath := *engineArchive
	if archivePath == "" {
		archivePath = filepath.Join(temporary, asset.Filename)
		if err := downloadFile(asset.URL, archivePath); err != nil {
			return fmt.Errorf("download Tectonic: %w", err)
		}
	}
	if err := verifyFile(archivePath, asset.Size, asset.SHA256); err != nil {
		return fmt.Errorf("verify Tectonic archive: %w", err)
	}
	enginePath := filepath.Join(temporary, "tectonic")
	if runtime.GOOS == "windows" {
		enginePath += ".exe"
	}
	if err := extractEngine(archivePath, asset.Archive, enginePath); err != nil {
		return err
	}
	if err := os.Chmod(enginePath, 0o755); err != nil {
		return err
	}

	include := filepath.Join(temporary, "bundle", "include")
	if err := os.MkdirAll(include, 0o755); err != nil {
		return err
	}
	if err := fetchResources(manifest.Bundle.SourceURL, resources, include); err != nil {
		return err
	}
	spec := filepath.Join(temporary, "bundle", "bundle.toml")
	text := "[bundle]\nname = \"ahdcode-latex\"\nexpected_hash = \"" + manifest.Bundle.ContentSHA256 + "\"\nsearch_order = [{ input = \"include\" }]\n\n[inputs.\"include\"]\nsource.dir.path = \"include\"\n"
	if err := os.WriteFile(spec, []byte(text), 0o600); err != nil {
		return err
	}
	buildDirectory := filepath.Join(temporary, "bundle-build")
	command := exec.Command(enginePath, "-X", "bundle", "create", "--build-dir", buildDirectory, spec, "v1")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build local Tectonic bundle: %w", err)
	}
	bundlePath := filepath.Join(buildDirectory, "ahdcode-latex", "ahdcode-latex.ttb")
	if err := verifyFile(bundlePath, manifest.Bundle.TTBSize, manifest.Bundle.TTBSHA256); err != nil {
		return fmt.Errorf("verify generated bundle: %w", err)
	}

	runtimeDirectory := filepath.Join(*output, "libexec", "ahdcode", "latex")
	if err := os.MkdirAll(runtimeDirectory, 0o755); err != nil {
		return err
	}
	engineName := "tectonic"
	if runtime.GOOS == "windows" {
		engineName += ".exe"
	}
	if err := copyFile(enginePath, filepath.Join(runtimeDirectory, engineName), 0o755); err != nil {
		return err
	}
	if err := copyFile(bundlePath, filepath.Join(runtimeDirectory, "ahdcode-latex.ttb"), 0o644); err != nil {
		return err
	}
	notices := filepath.Join(filepath.Dir(*assetsPath), "THIRD_PARTY_NOTICES.txt")
	if err := copyFile(notices, filepath.Join(runtimeDirectory, "THIRD_PARTY_NOTICES.txt"), 0o644); err != nil {
		return err
	}
	licenses := filepath.Join(filepath.Dir(*assetsPath), "licenses")
	if err := copyTree(licenses, filepath.Join(runtimeDirectory, "licenses")); err != nil {
		return err
	}
	if *compiler != "" {
		name := "ahdcode"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := copyFile(*compiler, filepath.Join(*output, "bin", name), 0o755); err != nil {
			return err
		}
	}
	payload, err := treeSize(runtimeDirectory)
	if err != nil {
		return err
	}
	if payload > hardLatexPayloadLimit {
		return fmt.Errorf("Latex payload is %.2f MiB; hard limit is 250 MiB", mib(payload))
	}
	complete, err := treeSize(*output)
	if err != nil {
		return err
	}
	fmt.Printf("Tectonic archive: %s (%s)\n", asset.Filename, asset.SHA256)
	fmt.Printf("Tectonic executable: %.2f MiB\n", mib(fileSize(filepath.Join(runtimeDirectory, engineName))))
	fmt.Printf("local resource bundle: %.2f MiB (%s)\n", mib(manifest.Bundle.TTBSize), manifest.Bundle.TTBSHA256)
	fmt.Printf("total Latex payload: %.2f MiB\n", mib(payload))
	fmt.Printf("complete staged distribution: %.2f MiB\n", mib(complete))
	return nil
}

func capture(arguments []string) error {
	flags := flag.NewFlagSet("capture", flag.ContinueOnError)
	indexPath := flags.String("index", "", "Tectonic source index")
	cachePath := flags.String("cache", "", "flat fetched-resource directory")
	output := flags.String("output", "tooling/latex/resources.json", "output resource manifest")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *indexPath == "" || *cachePath == "" {
		return errors.New("--index and --cache are required")
	}
	indexBytes, err := os.ReadFile(*indexPath)
	if err != nil {
		return err
	}
	locations := make(map[string]resource)
	for _, line := range strings.Split(string(indexBytes), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		offset, offsetError := strconv.ParseInt(fields[1], 10, 64)
		length, lengthError := strconv.ParseInt(fields[2], 10, 64)
		if offsetError == nil && lengthError == nil {
			locations[fields[0]] = resource{Name: fields[0], Offset: offset, Length: length}
		}
	}
	entries, err := os.ReadDir(*cachePath)
	if err != nil {
		return err
	}
	resources := make([]resource, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected non-file resource %s", entry.Name())
		}
		item, ok := locations[entry.Name()]
		if !ok {
			return fmt.Errorf("resource %s is absent from source index", entry.Name())
		}
		path := filepath.Join(*cachePath, entry.Name())
		item.SHA256, err = fileSHA256(path)
		if err != nil {
			return err
		}
		if fileSize(path) != item.Length {
			return fmt.Errorf("resource %s length differs from source index", entry.Name())
		}
		resources = append(resources, item)
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Name < resources[j].Name })
	encoded, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(*output, encoded, 0o644)
}

func readAssets(path string) (assetsManifest, error) {
	var result assetsManifest
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &result)
	}
	if err != nil {
		return result, fmt.Errorf("read asset manifest: %w", err)
	}
	if result.Schema != 1 || result.Bundle.ResourceManifest == "" {
		return result, errors.New("unsupported or incomplete asset manifest")
	}
	return result, nil
}

func readResources(path string) ([]resource, error) {
	var result []resource
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &result)
	}
	if err != nil {
		return nil, fmt.Errorf("read resource manifest: %w", err)
	}
	for _, item := range result {
		if item.Name == "" || filepath.Base(item.Name) != item.Name || item.Offset < 0 || item.Length <= 0 || len(item.SHA256) != 64 {
			return nil, fmt.Errorf("unsafe or incomplete resource entry %q", item.Name)
		}
	}
	return result, nil
}

func selectEngine(assets []engineAsset, goos, goarch string) (engineAsset, error) {
	for _, asset := range assets {
		if asset.GOOS == goos && asset.GOARCH == goarch {
			return asset, nil
		}
	}
	return engineAsset{}, fmt.Errorf("no pinned Tectonic asset for %s/%s", goos, goarch)
}

func downloadFile(url, path string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", url, response.Status)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyError := io.Copy(file, response.Body)
	closeError := file.Close()
	if copyError != nil {
		return copyError
	}
	return closeError
}

func fetchResources(url string, resources []resource, directory string) error {
	jobs := make(chan resource)
	errorsFound := make(chan error, 1)
	var wait sync.WaitGroup
	client := &http.Client{}
	worker := func() {
		defer wait.Done()
		for item := range jobs {
			request, err := http.NewRequest(http.MethodGet, url, nil)
			if err == nil {
				request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", item.Offset, item.Offset+item.Length-1))
				var response *http.Response
				response, err = client.Do(request)
				if err == nil {
					if response.StatusCode != http.StatusPartialContent {
						err = fmt.Errorf("resource %s range returned %s", item.Name, response.Status)
					} else {
						data, readError := io.ReadAll(io.LimitReader(response.Body, item.Length+1))
						if readError != nil {
							err = readError
						} else if int64(len(data)) != item.Length || bytesSHA256(data) != item.SHA256 {
							err = fmt.Errorf("resource %s checksum or length mismatch", item.Name)
						} else {
							err = os.WriteFile(filepath.Join(directory, item.Name), data, 0o600)
						}
					}
					response.Body.Close()
				}
			}
			if err != nil {
				select {
				case errorsFound <- err:
				default:
				}
			}
		}
	}
	for range 8 {
		wait.Add(1)
		go worker()
	}
	for _, item := range resources {
		select {
		case jobs <- item:
		case err := <-errorsFound:
			close(jobs)
			wait.Wait()
			return err
		}
	}
	close(jobs)
	wait.Wait()
	select {
	case err := <-errorsFound:
		return err
	default:
		return nil
	}
}

func extractEngine(archivePath, kind, output string) error {
	switch kind {
	case "tar.gz":
		file, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer file.Close()
		compressed, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer compressed.Close()
		reader := tar.NewReader(compressed)
		for {
			header, nextError := reader.Next()
			if nextError == io.EOF {
				break
			}
			if nextError != nil {
				return nextError
			}
			if filepath.Base(header.Name) == "tectonic" && header.Typeflag == tar.TypeReg {
				return writeLimited(output, reader, header.Size)
			}
		}
	case "zip":
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer reader.Close()
		for _, file := range reader.File {
			if filepath.Base(file.Name) != "tectonic.exe" {
				continue
			}
			input, openError := file.Open()
			if openError != nil {
				return openError
			}
			err = writeLimited(output, input, int64(file.UncompressedSize64))
			input.Close()
			return err
		}
	default:
		return fmt.Errorf("unsupported engine archive %q", kind)
	}
	return errors.New("Tectonic executable is missing from verified archive")
}

func writeLimited(path string, input io.Reader, size int64) error {
	if size <= 0 || size > hardLatexPayloadLimit {
		return fmt.Errorf("unsafe extracted engine size %d", size)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	written, copyError := io.Copy(file, io.LimitReader(input, size+1))
	closeError := file.Close()
	if copyError != nil {
		return copyError
	}
	if closeError != nil {
		return closeError
	}
	if written != size {
		return fmt.Errorf("extracted engine size is %d, want %d", written, size)
	}
	return nil
}

func verifyFile(path string, size int64, expected string) error {
	if fileSize(path) != size {
		return fmt.Errorf("size is %d, want %d", fileSize(path), size)
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("SHA-256 is %s, want %s", actual, expected)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func bytesSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func copyFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyError := io.Copy(output, input)
	closeError := output.Close()
	if copyError != nil {
		return copyError
	}
	return closeError
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("license tree contains non-file %s", path)
		}
		return copyFile(path, target, 0o644)
	})
}

func treeSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return info.Size()
}

func mib(size int64) float64 { return float64(size) / (1024 * 1024) }
