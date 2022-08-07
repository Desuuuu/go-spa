// Package spa implements a http.Handler to simplify the process of serving
// static files for SPAs.
package spa

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Option is used to modify the behavior of StaticHandler.
type Option func(*handler)

type handler struct {
	fs            http.FileSystem
	fallback      string
	indexRedirect bool
}

// StaticHandler returns a http.Handler that serves HTTP requests with the
// with the contents of the specified file system.
//
// Requests for non-existing files will fallback to serving the content of
// "/index.html" by default. The fallback path can be changed using Fallback.
//
// Requests for anything other than regular files (directories, symlinks...)
// also use the fallback logic.
//
// Any request ending in "/index.html" is redirected to the same path, without
// the final "index.html". This can be disabled using NoIndexRedirect.
func StaticHandler(fs http.FileSystem, opts ...Option) http.Handler {
	h := &handler{
		fs:            fs,
		fallback:      "/index.html",
		indexRedirect: true,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Fallback is used to modify the fallback path of StaticHandler.
//
//	StaticHandler(fs, Fallback("/index.txt"))
func Fallback(f string) Option {
	return func(h *handler) {
		h.fallback = f
	}
}

// NoIndexRedirect is used to disable the index redirection of StaticHandler.
//
//	StaticHandler(fs, NoIndexRedirect())
func NoIndexRedirect() Option {
	return func(h *handler) {
		h.indexRedirect = false
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fpath := r.URL.Path
	if !strings.HasPrefix(fpath, "/") {
		fpath = "/" + fpath
	}

	if h.indexRedirect && strings.HasSuffix(fpath, "/index.html") {
		redirect(w, r, "./")
		return
	}

	if fpath != "/" && strings.HasSuffix(fpath, "/") {
		h.handleError(w, r, fs.ErrNotExist)
		return
	}

	f, err := h.fs.Open(path.Clean(fpath))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	if !fi.Mode().IsRegular() {
		h.handleError(w, r, fs.ErrNotExist)
		return
	}

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

func (h *handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if h.fallback == "" || (!errors.Is(err, fs.ErrNotExist) && !errors.Is(err, fs.ErrPermission)) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	f, err := h.fs.Open(h.fallback)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if !fi.Mode().IsRegular() {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

func redirect(w http.ResponseWriter, r *http.Request, dst string) {
	if q := r.URL.RawQuery; q != "" {
		dst += "?" + q
	}

	if f := r.URL.EscapedFragment(); f != "" {
		dst += "#" + f
	}

	w.Header().Set("Location", dst)
	w.WriteHeader(http.StatusMovedPermanently)
}
