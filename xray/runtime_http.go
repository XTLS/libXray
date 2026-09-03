package xray

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const runtimeResponseLimit = 16 * 1024 * 1024

type runtimeFiles struct {
	Current  *RuntimeSnapshot  `json:"current"`
	Archived []RuntimeSnapshot `json:"archived"`
}

func validRuntimeID(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 16 && value == strings.ToLower(value)
}

func validateRuntimeHTTP(config *RuntimeConfig) error {
	if config.Listen == "" && config.Token == "" {
		return nil
	}
	host, portText, err := net.SplitHostPort(config.Listen)
	port, portErr := strconv.Atoi(portText)
	if err != nil || host != "127.0.0.1" || portErr != nil || port < 1 || port > 65535 ||
		strconv.Itoa(port) != portText || !validRuntimeID(config.Token) {
		return errors.New("runtime HTTP requires listen 127.0.0.1:1..65535 and a 32-character lowercase hex token")
	}
	return nil
}

func (r *managedRuntime) listenHTTP() (net.Listener, error) {
	if r.config.Listen == "" {
		return nil, nil
	}
	listener, err := net.Listen("tcp4", r.config.Listen)
	if err != nil {
		return nil, errors.New("runtime HTTP listener is unavailable")
	}
	return listener, nil
}

func (r *managedRuntime) serveHTTP(listener net.Listener) {
	r.httpListener = listener
	r.httpServer = &http.Server{
		Handler:           http.HandlerFunc(r.handleHTTP),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 * 1024,
	}
	server := r.httpServer
	go func() { _ = server.Serve(listener) }()
}

func (r *managedRuntime) handleHTTP(w http.ResponseWriter, request *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if subtle.ConstantTimeCompare([]byte(request.Header.Get("Authorization")), []byte("Bearer "+r.config.Token)) != 1 {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var removeSessionIDs []string
	switch request.URL.Path {
	case "/runtime":
		if request.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case "/runtime/ack":
		if request.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			RemoveSessionIDs []string `json:"removeSessionIds"`
		}
		request.Body = http.MaxBytesReader(w, request.Body, 64*1024)
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		var extra any
		if decoder.Decode(&payload) != nil || decoder.Decode(&extra) != io.EOF || payload.RemoveSessionIDs == nil {
			http.Error(w, "invalid runtime acknowledgment", http.StatusBadRequest)
			return
		}
		for _, id := range payload.RemoveSessionIDs {
			if !validRuntimeID(id) {
				http.Error(w, "invalid runtime session ID", http.StatusBadRequest)
				return
			}
		}
		removeSessionIDs = payload.RemoveSessionIDs
	default:
		http.NotFound(w, request)
		return
	}
	// Files are atomically replaced by the ticker; HTTP never reads or mutates
	// its in-memory sample. Serialize only archive readers/acknowledgments.
	r.httpMu.Lock()
	defer r.httpMu.Unlock()
	if r.httpClosed || request.Context().Err() != nil {
		http.Error(w, "runtime stopped", http.StatusServiceUnavailable)
		return
	}
	files, err := r.readHTTPFiles(removeSessionIDs)
	if err != nil {
		http.Error(w, "runtime snapshots unavailable", http.StatusServiceUnavailable)
		return
	}
	data, err := json.Marshal(files)
	if err != nil || len(data) > runtimeResponseLimit {
		http.Error(w, "runtime snapshots exceed response limit", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (r *managedRuntime) readHTTPFiles(removeSessionIDs []string) (runtimeFiles, error) {
	files := runtimeFiles{Archived: []RuntimeSnapshot{}}
	current, err := readRuntimeState(r.config.StatePath)
	if err != nil {
		return files, err
	}
	if current.Version != 0 {
		files.Current = &current
	} else if len(removeSessionIDs) != 0 {
		return files, errors.New("runtime current session is unavailable")
	}
	directory := filepath.Join(filepath.Dir(r.config.StatePath), "runtime-sessions")
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil || !info.IsDir() {
		return files, errors.New("runtime archive directory is unavailable")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return files, err
	}
	remove := make(map[string]bool, len(removeSessionIDs))
	for _, id := range removeSessionIDs {
		if id != current.Session.ID {
			remove[id] = true
		}
	}
	// Bound the accumulated response before marshaling the complete envelope.
	encoded, _ := json.Marshal(files)
	size := len(encoded)
	for _, entry := range entries {
		id, isJSON := strings.CutSuffix(entry.Name(), ".json")
		if !isJSON || !validRuntimeID(id) {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		state, err := readRuntimeState(path)
		if err != nil || state.Version == 0 || state.Session.ID != id {
			return files, errors.New("runtime archive is invalid")
		}
		if remove[id] && os.Remove(path) == nil {
			continue
		}
		encoded, _ := json.Marshal(state)
		size += len(encoded) + 1
		if size > runtimeResponseLimit {
			return files, errors.New("runtime snapshots exceed response limit")
		}
		files.Archived = append(files.Archived, state)
	}
	return files, nil
}
