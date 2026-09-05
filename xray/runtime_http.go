package xray

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func validRuntimeToken(value string) bool {
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
		strconv.Itoa(port) != portText || !validRuntimeToken(config.Token) {
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
	if request.URL.Path != "/runtime" {
		http.NotFound(w, request)
		return
	}
	if request.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snapshot, err := readRuntimeState(r.config.StatePath)
	if err != nil {
		http.Error(w, "runtime snapshot unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}
