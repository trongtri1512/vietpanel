// Package filemanager provides a basic file browser, upload, editor, and
// directory operations. In production the base path is scoped to the user's
// home container; in dev it defaults to a configurable root.
package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

type bcItem struct {
	Path  string
	Label string
}

type fmData struct {
	Dir         string
	Parent      string
	Entries     []fileEntry
	Breadcrumbs []bcItem
}

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /files", auth.Require(list(a)))
	mux.HandleFunc("GET /files/view", auth.Require(view(a)))
	mux.HandleFunc("GET /files/edit", auth.Require(editPage(a)))
	mux.HandleFunc("POST /files/edit", auth.Require(editAction(a)))
	mux.HandleFunc("POST /files/upload", auth.Require(upload(a)))
	mux.HandleFunc("POST /files/mkdir", auth.Require(mkdir(a)))
	mux.HandleFunc("POST /files/delete", auth.Require(deleteFile(a)))
	mux.HandleFunc("GET /files/download", auth.Require(download(a)))
}

type fileEntry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime string
	Ext     string
}

func basePath() string {
	if d := os.Getenv("FILEMANAGER_BASE"); d != "" {
		return d
	}
	h, _ := os.UserHomeDir()
	return h
}

func resolvePath(userPath string) (string, error) {
	base := basePath()
	clean := filepath.Clean("/" + strings.TrimPrefix(userPath, "/"))
	full := filepath.Join(base, clean)
	if !strings.HasPrefix(full, base) {
		return "", fmt.Errorf("access denied")
	}
	return full, nil
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			dir = "/"
		}
		full, err := resolvePath(dir)
		if err != nil {
			web.Render(w, r, a, "user/filemanager.html", web.PageData{
				Title: "Files", ActiveNav: "files", Error: err.Error(),
			})
			return
		}
		entries, readErr := readDir(full)
		parent := filepath.Dir(dir)
		if parent == "." {
			parent = "/"
		}
		bc := buildBreadcrumbs(dir)
		web.Render(w, r, a, "user/filemanager.html", web.PageData{
			Title: "Files: " + dir, ActiveNav: "files", User: auth.FromContext(r),
			Error: errStr(readErr),
			Extra: map[string]any{"FM": fmData{
				Dir: dir, Parent: parent, Entries: entries, Breadcrumbs: bc,
			}},
		})
	}
}

func buildBreadcrumbs(path string) []bcItem {
	if path == "/" {
		return []bcItem{{Path: "/", Label: "Home"}}
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var bc []bcItem
	bc = append(bc, bcItem{Path: "/", Label: "Home"})
	accum := ""
	for _, p := range parts {
		accum += "/" + p
		bc = append(bc, bcItem{Path: accum, Label: p})
	}
	return bc
}

func errStr(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func readDir(path string) ([]fileEntry, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var entries []fileEntry
	for _, d := range dirs {
		info, _ := d.Info()
		size := int64(0)
		modTime := "-"
		if info != nil {
			size = info.Size()
			modTime = info.ModTime().Format("2006-01-02 15:04")
		}
		entries = append(entries, fileEntry{
			Name: d.Name(), IsDir: d.IsDir(),
			Size: size, ModTime: modTime,
			Ext: strings.ToLower(filepath.Ext(d.Name())),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func view(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if file == "" {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		full, err := resolvePath(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, full)
	}
}

func editPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if file == "" {
			http.Redirect(w, r, "/files", http.StatusFound)
			return
		}
		full, err := resolvePath(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		b, err := os.ReadFile(full)
		content := ""
		if err == nil && len(b) < 1024*1024 {
			content = string(b)
		}
		type editData struct{ File, Dir, Content string }
		web.Render(w, r, a, "user/filemanager_edit.html", web.PageData{
			Title: "Edit: " + filepath.Base(file), ActiveNav: "files", User: auth.FromContext(r),
			Extra: map[string]any{"Edit": editData{
				File: file, Dir: filepath.Dir(file), Content: content,
			}},
		})
	}
}

func editAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		file := r.FormValue("file")
		content := r.FormValue("content")
		dir := r.FormValue("dir")
		if file == "" {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		full, err := resolvePath(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			http.Error(w, "write failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/files?dir="+dir, http.StatusFound)
	}
}

func upload(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			http.Error(w, "upload too large", http.StatusBadRequest)
			return
		}
		dir := r.FormValue("dir")
		if dir == "" {
			dir = "/"
		}
		f, hdr, err := r.FormFile("uploaded_file")
		if err != nil {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		defer f.Close()
		fullDir, err := resolvePath(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		dst, err := os.Create(filepath.Join(fullDir, hdr.Filename))
		if err != nil {
			http.Error(w, "create failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		io.Copy(dst, f)
		http.Redirect(w, r, "/files?dir="+dir, http.StatusFound)
	}
}

func mkdir(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		dir := r.FormValue("dir")
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}
		fullDir, err := resolvePath(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err := os.MkdirAll(filepath.Join(fullDir, name), 0o755); err != nil {
			http.Error(w, "mkdir failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/files?dir="+dir, http.StatusFound)
	}
}

func deleteFile(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		path := r.FormValue("path")
		dir := r.FormValue("dir")
		if path == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		full, err := resolvePath(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		info, err := os.Stat(full)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if info.IsDir() {
			os.RemoveAll(full)
		} else {
			os.Remove(full)
		}
		http.Redirect(w, r, "/files?dir="+dir, http.StatusFound)
	}
}

func download(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if file == "" {
			http.Error(w, "missing file", http.StatusBadRequest)
			return
		}
		full, err := resolvePath(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(file)))
		http.ServeFile(w, r, full)
	}
}