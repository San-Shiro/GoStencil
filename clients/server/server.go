// Package server provides the GoStencil web UI editor and HTTP API.
package server

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/xob0t/GoStencil/pkg/generator"
	"github.com/xob0t/GoStencil/pkg/template"
)

//go:embed web/*
var webContent embed.FS

// ── Asset Manager ──

type asset struct {
	Name string
	Data []byte
	Mime string
}

type assetManager struct {
	mu     sync.RWMutex
	assets map[string]*asset
}

func newAssetManager() *assetManager {
	return &assetManager{assets: make(map[string]*asset)}
}

func (am *assetManager) add(name string, data []byte, mimeType string) string {
	id := randomID()
	am.mu.Lock()
	am.assets[id] = &asset{Name: name, Data: data, Mime: mimeType}
	am.mu.Unlock()
	return id
}

func (am *assetManager) get(id string) (*asset, bool) {
	am.mu.RLock()
	a, ok := am.assets[id]
	am.mu.RUnlock()
	return a, ok
}

func (am *assetManager) listAll() []map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()
	result := make([]map[string]interface{}, 0, len(am.assets))
	for id, a := range am.assets {
		result = append(result, map[string]interface{}{
			"id":   id,
			"name": a.Name,
			"mime": a.Mime,
			"size": len(a.Data),
		})
	}
	return result
}

func (am *assetManager) remove(id string) {
	am.mu.Lock()
	delete(am.assets, id)
	am.mu.Unlock()
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── Server ──

type srv struct {
	assets *assetManager
	tmpDir string
}

// RunServe starts the web UI server on the given port.
func RunServe(args []string) error {
	port := "8080"
	for i, a := range args {
		if (a == "--port" || a == "-p") && i+1 < len(args) {
			port = args[i+1]
		}
	}

	tmpDir, err := os.MkdirTemp("", "gostencil-serve-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	s := &srv{
		assets: newAssetManager(),
		tmpDir: tmpDir,
	}

	webFS, err := fs.Sub(webContent, "web")
	if err != nil {
		return fmt.Errorf("embed web: %w", err)
	}

	mux := http.NewServeMux()

	// API routes.
	mux.HandleFunc("POST /api/render", s.handleRender)
	mux.HandleFunc("POST /api/export/png", s.handleExportPNG)
	mux.HandleFunc("POST /api/export/avi", s.handleExportAVI)
	mux.HandleFunc("POST /api/export/gspresets", s.handleExportGSPresets)
	mux.HandleFunc("POST /api/export/json", s.handleExportJSON)
	mux.HandleFunc("POST /api/upload/font", s.handleUploadFont)
	mux.HandleFunc("POST /api/upload/image", s.handleUploadImage)
	mux.HandleFunc("POST /api/import/gspresets", s.handleImportGSPresets)
	mux.HandleFunc("GET /api/assets/{id}", s.handleGetAsset)
	mux.HandleFunc("DELETE /api/assets/{id}", s.handleDeleteAsset)
	mux.HandleFunc("GET /api/assets", s.handleListAssets)

	// Static files.
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	addr := ":" + port
	log.Printf("GoStencil UI → http://localhost%s", addr)

	// Open browser.
	go openBrowser("http://localhost" + addr)

	return http.ListenAndServe(addr, mux)
}

// ── Render (core) ──

type renderRequest struct {
	Preset json.RawMessage `json:"preset"`
	Data   json.RawMessage `json:"data"`
}

func (s *srv) renderImage(body []byte) ([]byte, error) {
	var req renderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	var preset template.Preset
	if err := json.Unmarshal(req.Preset, &preset); err != nil {
		return nil, fmt.Errorf("parse preset: %w", err)
	}

	// Apply canvas preset.
	if dims, ok := template.Presets[preset.Canvas.Preset]; ok {
		preset.Canvas.Width = dims[0]
		preset.Canvas.Height = dims[1]
	}
	preset.Canvas.Width = max(preset.Canvas.Width, 320)
	preset.Canvas.Height = max(preset.Canvas.Height, 240)

	if preset.Background.Color == "" {
		preset.Background.Color = "#1a1a2e"
	}

	// Resolve asset references to temp files.
	fontPath := s.resolveAssetPath(preset.Font.Path)
	preset.Background.Source = s.resolveAssetPath(preset.Background.Source)
	for i := range preset.Components {
		preset.Components[i].Style.BackgroundImage = s.resolveAssetPath(preset.Components[i].Style.BackgroundImage)
		preset.Components[i].Style.FontPath = s.resolveAssetPath(preset.Components[i].Style.FontPath)
		applyCompDefaults(&preset.Components[i])
	}

	// Parse data.
	var data *template.DataSpec
	if len(req.Data) > 0 && string(req.Data) != "null" && string(req.Data) != "{}" {
		var d template.DataSpec
		if err := json.Unmarshal(req.Data, &d); err == nil {
			data = &d
		}
	}

	// Merge + render.
	components := template.MergeData(&preset, data)
	renderer, err := template.NewRenderer(fontPath)
	if err != nil {
		return nil, fmt.Errorf("renderer: %w", err)
	}

	img, err := renderer.RenderPreset(&preset, components)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *srv) handleRender(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	data, err := s.renderImage(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

// ── Export ──

func (s *srv) handleExportPNG(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	data, err := s.renderImage(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `attachment; filename="output.png"`)
	w.Write(data)
}

func (s *srv) handleExportAVI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Preset   json.RawMessage `json:"preset"`
		Data     json.RawMessage `json:"data"`
		Duration int             `json:"duration"`
	}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pngData, err := s.renderImage(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		http.Error(w, "decode rendered image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	dur := max(req.Duration, 1)
	tmpPath := filepath.Join(s.tmpDir, "export_"+randomID()+".avi")
	cfg := generator.Config{Image: img, Duration: dur}
	if err := generator.Generate(tmpPath, cfg); err != nil {
		http.Error(w, "generate AVI: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpPath)

	aviData, err := os.ReadFile(tmpPath)
	if err != nil {
		http.Error(w, "read AVI: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "video/avi")
	w.Header().Set("Content-Disposition", `attachment; filename="output.avi"`)
	w.Write(aviData)
}

func (s *srv) handleExportGSPresets(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Preset json.RawMessage `json:"preset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Write preset.json (pretty-printed).
	pw, _ := zw.Create("preset.json")
	var prettyPreset bytes.Buffer
	json.Indent(&prettyPreset, req.Preset, "", "  ")
	pw.Write(prettyPreset.Bytes())

	// Write all uploaded assets.
	s.assets.mu.RLock()
	for id, a := range s.assets.assets {
		ext := extensionForMime(a.Mime)
		aw, _ := zw.Create("assets/" + id + ext)
		aw.Write(a.Data)
	}
	s.assets.mu.RUnlock()

	zw.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="preset.gspresets"`)
	w.Write(buf.Bytes())
}

func (s *srv) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filename := req.Type + ".json"
	var pretty bytes.Buffer
	json.Indent(&pretty, req.Content, "", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(pretty.Bytes())
}

// ── Import ──

func (s *srv) handleImportGSPresets(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(50 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, _ := io.ReadAll(file)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		http.Error(w, "invalid ZIP: "+err.Error(), http.StatusBadRequest)
		return
	}

	var presetJSON json.RawMessage
	importedAssets := make([]map[string]string, 0)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, _ := f.Open()
		fdata, _ := io.ReadAll(rc)
		rc.Close()

		if f.Name == "preset.json" {
			presetJSON = fdata
		} else {
			mimeType := mime.TypeByExtension(filepath.Ext(f.Name))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			id := s.assets.add(filepath.Base(f.Name), fdata, mimeType)
			importedAssets = append(importedAssets, map[string]string{
				"id":           id,
				"name":         filepath.Base(f.Name),
				"originalPath": f.Name,
				"url":          "/api/assets/" + id,
			})
		}
	}

	if presetJSON == nil {
		http.Error(w, "no preset.json found in archive", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"preset": presetJSON,
		"assets": importedAssets,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ── Upload ──

func (s *srv) handleUploadFont(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, _ := io.ReadAll(file)
	id := s.assets.add(header.Filename, data, "font/ttf")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   id,
		"name": header.Filename,
		"url":  "/api/assets/" + id,
	})
}

func (s *srv) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, _ := io.ReadAll(file)
	mimeType := mime.TypeByExtension(filepath.Ext(header.Filename))
	if mimeType == "" {
		mimeType = "image/png"
	}
	id := s.assets.add(header.Filename, data, mimeType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   id,
		"name": header.Filename,
		"url":  "/api/assets/" + id,
	})
}

// ── Asset serving ──

func (s *srv) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, ok := s.assets.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", a.Mime)
	w.Write(a.Data)
}

func (s *srv) handleListAssets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.assets.listAll())
}

func (s *srv) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, ok := s.assets.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.assets.remove(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": id})
}

// ── Helpers ──

func applyCompDefaults(c *template.Component) {
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
	if c.Defaults.Visible == nil {
		t := true
		c.Defaults.Visible = &t
	}
}

func (s *srv) resolveAssetPath(path string) string {
	if path == "" {
		return ""
	}
	a, ok := s.assets.get(path)
	if !ok {
		return path
	}
	tmpPath := filepath.Join(s.tmpDir, path+"_"+sanitizeFilename(a.Name))
	os.WriteFile(tmpPath, a.Data, 0644)
	return tmpPath
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func extensionForMime(m string) string {
	switch {
	case strings.Contains(m, "ttf"), strings.Contains(m, "font"):
		return ".ttf"
	case strings.Contains(m, "png"):
		return ".png"
	case strings.Contains(m, "jpeg"), strings.Contains(m, "jpg"):
		return ".jpg"
	default:
		return ""
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
