package packaging

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var privateKeyPEM = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)

func validWebComponent(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") || strings.TrimSpace(name) != name ||
		strings.HasSuffix(name, ".") || strings.ContainsAny(name, `<>:"/\|?*`) ||
		strings.ContainsFunc(name, unicode.IsControl) || forbiddenNames.MatchString(name) {
		return false
	}
	device, _, _ := strings.Cut(strings.ToUpper(name), ".")
	device = strings.TrimRight(device, " ")
	switch device {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return false
	}
	for _, prefix := range []string{"COM", "LPT"} {
		if suffix, ok := strings.CutPrefix(device, prefix); ok &&
			len([]rune(suffix)) == 1 && strings.Contains("123456789¹²³", suffix) {
			return false
		}
	}
	return true
}

func webAssetType(name string) bool {
	if webImageType(name) {
		return true
	}
	if name == "LICENSE" {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".css", ".js", ".mjs", ".html", ".webmanifest", ".woff", ".woff2", ".ttf", ".otf":
		return true
	}
	return false
}

func webImageType(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".svg", ".gif", ".ico", ".webp", ".avif":
		return true
	}
	return false
}

// openWebDir pins an approved directory. Both a direct link and a directory
// substituted between Lstat and OpenRoot are rejected.
func openWebDir(parent *os.Root, path string) (*os.Root, error) {
	info, err := parent.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s must be a real directory without links or junctions", path)
	}
	tree, err := parent.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	opened, err := tree.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		tree.Close()
		return nil, fmt.Errorf("%s changed while opening it", path)
	}
	return tree, nil
}

func copyWebAssets(entry, bundle, programDir string, pwa bool) ([]string, error) {
	project, err := os.OpenRoot(filepath.Dir(entry))
	if err != nil {
		return nil, err
	}
	defer project.Close()
	public, err := openWebDir(project, "public")
	if err != nil {
		return nil, err
	}
	defer public.Close()
	copied, err := copyWebTree(public, bundle, filepath.Join(programDir, "public"))
	if err != nil || !pwa {
		return copied, err
	}
	pwaRoot, err := openWebDir(project, "PWA")
	if err != nil {
		return nil, err
	}
	defer pwaRoot.Close()
	pwaPublic, err := openWebDir(pwaRoot, "public")
	if err != nil {
		return nil, err
	}
	defer pwaPublic.Close()
	pwaDir := filepath.Join(programDir, "PWA")
	assets, err := copyWebTree(pwaPublic, bundle, filepath.Join(pwaDir, "public"))
	if err != nil {
		return nil, err
	}
	copied = append(copied, assets...)
	for _, name := range []string{"worker.js", "app.webmanifest"} {
		relative := filepath.Join(pwaDir, name)
		if err := copyWebAsset(pwaRoot, name, filepath.Join(bundle, relative)); err != nil {
			return nil, err
		}
		copied = append(copied, relative)
	}
	if err := validateWebPWA(filepath.Join(bundle, pwaDir)); err != nil {
		return nil, err
	}
	return copied, nil
}

func copyWebTree(tree *os.Root, bundle, destination string) ([]string, error) {
	var copied []string
	paths := make(map[string]string)
	err := fs.WalkDir(tree.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(destination, filepath.FromSlash(path))
		if path == "." {
			return os.MkdirAll(filepath.Join(bundle, target), 0o755)
		}
		if !validWebComponent(entry.Name()) {
			return fmt.Errorf("%s/%s is not an approved public path", tree.Name(), path)
		}
		key := strings.ToUpper(path)
		if other, exists := paths[key]; exists {
			return fmt.Errorf("%s: public paths %q and %q collide on case-insensitive systems", tree.Name(), other, path)
		}
		paths[key] = path
		if entry.IsDir() {
			switch strings.ToLower(entry.Name()) {
			case "private", "secrets", "credentials", "config", "certs", "database":
				return fmt.Errorf("%s/%s is a private directory", tree.Name(), path)
			}
			return os.Mkdir(filepath.Join(bundle, target), 0o755)
		}
		if !webAssetType(entry.Name()) {
			return fmt.Errorf("%s/%s is not a supported public asset type", tree.Name(), path)
		}
		if err := copyWebAsset(tree, filepath.FromSlash(path), filepath.Join(bundle, target)); err != nil {
			return err
		}
		copied = append(copied, target)
		return nil
	})
	return copied, err
}

func copyWebAsset(tree *os.Root, name, destination string) error {
	info, err := tree.Lstat(name)
	if err != nil {
		return fmt.Errorf("%s/%s: %w", tree.Name(), name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s/%s must be a regular file without links or junctions", tree.Name(), name)
	}
	input, err := tree.Open(name)
	if err != nil {
		return err
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return fmt.Errorf("%s/%s changed while opening it", tree.Name(), name)
	}
	if err := copyFrom(input, destination, 0o644); err != nil {
		return err
	}
	staged, err := os.Open(destination)
	if err != nil {
		return err
	}
	defer staged.Close()
	if privateKeyPEM.MatchReader(bufio.NewReader(staged)) {
		return fmt.Errorf("%s/%s appears to contain a PEM private key", tree.Name(), name)
	}
	return nil
}

func validateWebPWA(directory string) error {
	for _, name := range []string{"worker.js", filepath.Join("public", "register.js")} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || info.Size() == 0 {
			return fmt.Errorf("PWA/%s is missing or empty", name)
		}
	}
	file, err := os.Open(filepath.Join(directory, "app.webmanifest"))
	if err != nil {
		return err
	}
	defer file.Close()
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(data) > limit {
		return fmt.Errorf("PWA/app.webmanifest cannot be read or exceeds 1 MiB")
	}
	var manifest struct {
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
		Icons     []struct {
			Source string `json:"src"`
		} `json:"icons"`
	}
	if json.Unmarshal(data, &manifest) != nil ||
		(strings.TrimSpace(manifest.Name) == "" && strings.TrimSpace(manifest.ShortName) == "") || len(manifest.Icons) == 0 {
		return fmt.Errorf("PWA/app.webmanifest needs valid JSON with a name or short_name and at least one icon")
	}
	for _, icon := range manifest.Icons {
		address, err := url.Parse(icon.Source)
		if err != nil || address.Host != "" || address.Scheme != "" || address.User != nil ||
			address.RawQuery != "" || address.Fragment != "" {
			return fmt.Errorf("PWA manifest icon %q must name a local /ahd-pwa/ asset", icon.Source)
		}
		prefix := strings.TrimPrefix(address.Path, "/")
		asset, ok := strings.CutPrefix(prefix, "ahd-pwa/")
		if !ok || !fs.ValidPath(asset) || !webImageType(asset) {
			return fmt.Errorf("PWA manifest icon %q must name a local /ahd-pwa/ image", icon.Source)
		}
		for _, part := range strings.Split(asset, "/") {
			if !validWebComponent(part) {
				return fmt.Errorf("invalid PWA manifest icon path %q", icon.Source)
			}
		}
		info, err := os.Lstat(filepath.Join(directory, "public", filepath.FromSlash(asset)))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("PWA manifest icon %q is missing or empty in PWA/public", icon.Source)
		}
	}
	return nil
}

// publishWeb reserves a fresh directory and uses an exclusive archive open.
// If any step fails, only entries moved from this staging directory are undone.
func publishWeb(staged, final, archive string) error {
	children, err := os.ReadDir(staged)
	if err != nil {
		return err
	}
	if err := os.Mkdir(final, 0o755); err != nil {
		return err
	}
	var moved []string
	success := false
	defer func() {
		if !success {
			for _, name := range moved {
				_ = os.Rename(filepath.Join(final, name), filepath.Join(staged, name))
			}
			_ = os.Remove(final)
		}
	}()
	for _, child := range children {
		name := child.Name()
		if err := os.Rename(filepath.Join(staged, name), filepath.Join(final, name)); err != nil {
			return err
		}
		moved = append(moved, name)
	}
	if archive != "" {
		if err := copyFile(filepath.Join(filepath.Dir(staged), archive), filepath.Join(filepath.Dir(final), archive), 0o644); err != nil {
			return err
		}
	}
	success = true
	return nil
}
