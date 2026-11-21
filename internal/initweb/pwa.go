package initweb

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/draw"
)

//go:embed pwa
var pwaSources embed.FS

type pwaIcon struct {
	Source string `json:"src"`
	Sizes  string `json:"sizes"`
	Type   string `json:"type"`
}

type pwaManifest struct {
	Name            string    `json:"name"`
	ShortName       string    `json:"short_name"`
	ID              string    `json:"id"`
	StartURL        string    `json:"start_url"`
	Scope           string    `json:"scope"`
	Display         string    `json:"display"`
	ThemeColor      string    `json:"theme_color"`
	BackgroundColor string    `json:"background_color"`
	Icons           []pwaIcon `json:"icons"`
}

type pwaAsset struct {
	Path      string `json:"path"`
	MIME      string `json:"mime"`
	Integrity string `json:"integrity"`
}

// PWA writes an optional, standalone group of files without editing app source.
func PWA(project, name, shortName string) error {
	name = strings.TrimSpace(name)
	shortName = strings.TrimSpace(shortName)
	if name == "" {
		name = filepath.Base(project)
	}
	if shortName == "" {
		shortName = name
	}
	for _, value := range []string{name, shortName} {
		if value == "" || value == "." || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 128 ||
			strings.ContainsFunc(value, unicode.IsControl) {
			return fmt.Errorf("manifest names must be nonempty UTF-8 text of at most 128 characters without control characters")
		}
	}
	files, err := pwaFiles(name, shortName)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Mkdir("PWA", 0o755); err != nil {
		return fmt.Errorf("create PWA directory (existing files are preserved): %w", err)
	}
	created := []string{"PWA"}
	success := false
	defer func() {
		if !success {
			for i := len(created) - 1; i >= 0; i-- {
				_ = root.Remove(created[i])
			}
		}
	}()
	public := filepath.Join("PWA", "public")
	if err := root.Mkdir(public, 0o755); err != nil {
		return err
	}
	created = append(created, public)
	for _, file := range files {
		path := filepath.Join("PWA", file.name)
		output, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		created = append(created, path)
		_, writeErr := output.Write(file.data)
		closeErr := output.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	success = true
	return nil
}

type pwaFile struct {
	name string
	data []byte
}

func pwaFiles(name, shortName string) ([]pwaFile, error) {
	logo, err := templates.ReadFile("templates/shared/public/ahdcode-logo.png")
	if err != nil {
		return nil, err
	}
	src, err := png.Decode(bytes.NewReader(logo))
	if err != nil {
		return nil, err
	}
	manifest := pwaManifest{
		Name: name, ShortName: shortName, ID: "/", StartURL: "/", Scope: "/",
		Display: "standalone", ThemeColor: "#212529", BackgroundColor: "#ffffff",
	}
	var files []pwaFile
	var assets []pwaAsset
	for _, size := range []int{192, 512} {
		icon := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(icon, icon.Bounds(), src, src.Bounds(), draw.Src, nil)
		var data bytes.Buffer
		if err := png.Encode(&data, icon); err != nil {
			return nil, err
		}
		name := "icon-" + strconv.Itoa(size) + ".png"
		path := "/ahd-pwa/" + name
		dimensions := fmt.Sprintf("%dx%d", size, size)
		manifest.Icons = append(manifest.Icons, pwaIcon{path, dimensions, "image/png"})
		hash := sha256.Sum256(data.Bytes())
		assets = append(assets, pwaAsset{path, "image/png", "sha256-" + base64.StdEncoding.EncodeToString(hash[:])})
		files = append(files, pwaFile{filepath.Join("public", name), data.Bytes()})
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	assetList, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return nil, err
	}
	files = append(files, pwaFile{"app.webmanifest", append(body, '\n')})
	for _, file := range []string{"Integration.ahd", "worker.js", "public/register.js"} {
		data, err := pwaSources.ReadFile("pwa/" + file)
		if err != nil {
			return nil, err
		}
		if file == "worker.js" {
			data = bytes.Replace(data, []byte("/* PUBLIC_ASSET_LIST */[]"), assetList, 1)
		}
		files = append(files, pwaFile{filepath.FromSlash(file), data})
	}
	return files, nil
}
