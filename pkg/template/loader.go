// loader.go — Load .gspresets (ZIP) bundles and parse preset.json.
package template

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadPreset opens a .gspresets ZIP, extracts it to a temp directory,
// parses preset.json, resolves all asset paths, and returns the preset.
// The returned cleanup function removes the temp directory.
func LoadPreset(path string) (*Preset, func(), error) {
	noop := func() {}

	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, noop, fmt.Errorf("open %s: %w", path, err)
	}
	defer r.Close()

	// Extract to temp dir.
	tmpDir, err := os.MkdirTemp("", "gspresets-*")
	if err != nil {
		return nil, noop, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	if err := extractZip(r, tmpDir); err != nil {
		cleanup()
		return nil, noop, fmt.Errorf("extract %s: %w", path, err)
	}

	// Parse preset.json.
	presetPath := filepath.Join(tmpDir, "preset.json")
	data, err := os.ReadFile(presetPath)
	if err != nil {
		cleanup()
		return nil, noop, fmt.Errorf("read preset.json: %w", err)
	}

	var preset Preset
	if err := json.Unmarshal(data, &preset); err != nil {
		cleanup()
		return nil, noop, fmt.Errorf("parse preset.json: %w", err)
	}

	// Apply canvas preset.
	if dims, ok := Presets[preset.Canvas.Preset]; ok {
		preset.Canvas.Width = dims[0]
		preset.Canvas.Height = dims[1]
	}
	preset.Canvas.Width = max(preset.Canvas.Width, 1280)
	preset.Canvas.Height = max(preset.Canvas.Height, 720)

	if preset.Background.Color == "" {
		preset.Background.Color = "#1a1a2e"
	}

	// Resolve asset paths relative to tmpDir.
	resolveAssetPaths(&preset, tmpDir)

	// Apply component style defaults.
	for i := range preset.Components {
		applyComponentDefaults(&preset.Components[i])
	}

	return &preset, cleanup, nil
}

// LoadData reads and parses a data.json file. Returns warnings for issues.
func LoadData(path string) (*DataSpec, []string, error) {
	var warnings []string

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read data.json: %w", err)
	}

	var spec DataSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		warnings = append(warnings, fmt.Sprintf("malformed data.json: %v — using all defaults", err))
		return &DataSpec{Components: make(map[string]ComponentData)}, warnings, nil
	}

	if spec.Components == nil {
		spec.Components = make(map[string]ComponentData)
	}

	return &spec, warnings, nil
}

// resolveAssetPaths makes all relative asset paths absolute using baseDir.
func resolveAssetPaths(preset *Preset, baseDir string) {
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(baseDir, p)
	}

	preset.Font.Path = resolve(preset.Font.Path)
	preset.Background.Source = resolve(preset.Background.Source)

	for i := range preset.Components {
		preset.Components[i].Style.BackgroundImage = resolve(preset.Components[i].Style.BackgroundImage)
	}
}

// applyComponentDefaults sets sane fallbacks for component style fields.
func applyComponentDefaults(c *Component) {
	s := &c.Style
	if s.FontSize <= 0 {
		s.FontSize = 24
	}
	if s.Color == "" {
		s.Color = "#ffffff"
	}
	if s.LineHeight <= 0 {
		s.LineHeight = 1.5
	}
	if s.TextAlign == "" {
		s.TextAlign = "left"
	}

	// Default visibility = true.
	if c.Defaults.Visible == nil {
		t := true
		c.Defaults.Visible = &t
	}
}

// extractZip extracts all files from a zip reader into destDir.
func extractZip(r *zip.ReadCloser, destDir string) error {
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		// Guard against zip slip.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		if err := extractFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

// extractFile writes a single zip entry to disk.
func extractFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
