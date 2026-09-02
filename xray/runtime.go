package xray

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/features/stats"
)

// RuntimeConfig is host metadata, never part of the Xray configuration.
type RuntimeConfig struct {
	StatePath  string `json:"statePath"`
	PlanID     string `json:"planId"`
	InboundTag string `json:"inboundTag"`
}

type RuntimeSession struct {
	ID          string `json:"id"`
	PlanID      string `json:"planId"`
	StartedAtMs int64  `json:"startedAtMs"`
	EndedAtMs   int64  `json:"endedAtMs"`
	Uplink      int64  `json:"uplink"`
	Downlink    int64  `json:"downlink"`
}

// RuntimeSnapshot contains only this session's raw inbound counter values.
// It contains no application totals, configuration, credentials, or control API.
type RuntimeSnapshot struct {
	Version     int            `json:"version"`
	Session     RuntimeSession `json:"session"`
	Available   bool           `json:"available"`
	SampledAtMs int64          `json:"sampledAtMs"`
	SavedAtMs   int64          `json:"savedAtMs"`
	Error       string         `json:"error"`
}

type managedRuntime struct {
	config               RuntimeConfig
	snapshot             RuntimeSnapshot
	manager              stats.Manager
	stateLock            *os.File
	stopTicker, tickDone chan struct{}
}

func prepareRuntime(config *RuntimeConfig) (*managedRuntime, error) {
	if config == nil {
		return nil, nil
	}
	if !filepath.IsAbs(config.StatePath) || strings.TrimSpace(config.PlanID) == "" || len(config.PlanID) > 256 ||
		strings.TrimSpace(config.InboundTag) == "" || len(config.InboundTag) > 256 {
		return nil, errors.New("runtime requires an absolute statePath, planId, and inboundTag")
	}
	stateLock, err := lockRuntimeState(config.StatePath)
	if err != nil {
		return nil, err
	}
	prepared := false
	defer func() {
		if !prepared {
			_ = stateLock.Close()
		}
	}()
	previous, err := readRuntimeState(config.StatePath)
	if err != nil {
		return nil, err
	}
	// Archive before any new session can replace the previous saved snapshot.
	// Repeated failed starts write the same session filename, not duplicate records.
	if err := archiveRuntimeState(config.StatePath, previous); err != nil {
		return nil, err
	}
	var id [16]byte
	if _, err = rand.Read(id[:]); err != nil {
		return nil, err
	}
	prepared = true
	return &managedRuntime{
		config: *config, stateLock: stateLock,
		snapshot: RuntimeSnapshot{
			Version: 1,
			Session: RuntimeSession{ID: hex.EncodeToString(id[:]), PlanID: config.PlanID, StartedAtMs: time.Now().UnixMilli()},
		},
	}, nil
}

func lockRuntimeState(path string) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("runtime statePath must be absolute")
	}
	lockPath := path + ".lock"
	if info, err := os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() || err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("runtime state lock is unavailable")
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, errors.New("runtime state lock is unavailable")
	}
	if err := lockRuntimeFile(file); err != nil {
		_ = file.Close()
		return nil, errors.New("runtime state is in use")
	}
	return file, nil
}

func (r *managedRuntime) attach(server *core.Instance) error {
	manager, ok := server.GetFeature(stats.ManagerType()).(stats.Manager)
	policies, policyOK := server.GetFeature(policy.ManagerType()).(policy.Manager)
	inbounds, inboundOK := server.GetFeature(inbound.ManagerType()).(inbound.Manager)
	if !ok || !policyOK || !inboundOK || !policies.ForSystem().Stats.InboundUplink || !policies.ForSystem().Stats.InboundDownlink {
		return errors.New("runtime requires inbound uplink and downlink statistics")
	}
	if _, err := inbounds.GetHandler(context.Background(), r.config.InboundTag); err != nil {
		return errors.New("runtime inboundTag does not exist")
	}
	// Registering zero counters handles an idle inbound without treating disabled statistics as zero.
	for _, direction := range []string{"uplink", "downlink"} {
		counter, err := manager.GetOrRegisterCounter(r.counterName(direction))
		if err != nil || counter == nil {
			return errors.New("runtime requires a statistics manager")
		}
	}
	r.manager = manager
	return nil
}

func (r *managedRuntime) counterName(direction string) string {
	return "inbound>>>" + r.config.InboundTag + ">>>traffic>>>" + direction
}

func (r *managedRuntime) start() error {
	r.sample()
	if err := r.save(); err != nil {
		return err
	}
	r.stopTicker, r.tickDone = make(chan struct{}), make(chan struct{})
	go func() {
		defer close(r.tickDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.sample()
				_ = r.save()
			case <-r.stopTicker:
				return
			}
		}
	}()
	return nil
}

func (r *managedRuntime) sample() {
	r.snapshot.SampledAtMs = time.Now().UnixMilli()
	up, down := r.manager.GetCounter(r.counterName("uplink")), r.manager.GetCounter(r.counterName("downlink"))
	r.snapshot.Available = up != nil && down != nil
	if r.snapshot.Available {
		u, d := up.Value(), down.Value()
		r.snapshot.Available = u >= 0 && d >= 0
		if r.snapshot.Available {
			// Preserve raw Value semantics, including a nonnegative counter rollback.
			// Never reset counters or synthesize deltas/application totals here.
			r.snapshot.Session.Uplink, r.snapshot.Session.Downlink = u, d
		}
	}
	r.snapshot.Error = ""
	if !r.snapshot.Available {
		r.snapshot.Error = "counters_unavailable"
	}
}

func (r *managedRuntime) save() error {
	candidate := r.snapshot
	candidate.SavedAtMs = time.Now().UnixMilli()
	candidate.Error = ""
	if !candidate.Available {
		candidate.Error = "counters_unavailable"
	}
	if err := writeRuntimeState(r.config.StatePath, candidate); err != nil {
		r.snapshot.Error = "state_write_failed"
		return errors.New("runtime state_write_failed")
	}
	r.snapshot = candidate
	return nil
}

func (r *managedRuntime) stop() error {
	if r.stopTicker == nil {
		return nil
	}
	close(r.stopTicker)
	<-r.tickDone
	r.stopTicker = nil
	// The ticker has exited, so the final sample/write cannot race a periodic one.
	r.sample()
	r.snapshot.Session.EndedAtMs = time.Now().UnixMilli()
	return r.save()
}

func readRuntimeState(path string) (RuntimeSnapshot, error) {
	var state RuntimeSnapshot
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64*1024 {
		return state, errors.New("runtime state is not a readable regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return state, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
	if err := decoder.Decode(&state); err != nil {
		return state, errors.New("runtime state is invalid")
	}
	var extra any
	id, idErr := hex.DecodeString(state.Session.ID)
	if decoder.Decode(&extra) != io.EOF || state.Version != 1 || idErr != nil || len(id) != 16 ||
		state.Session.ID != strings.ToLower(state.Session.ID) ||
		strings.TrimSpace(state.Session.PlanID) == "" || len(state.Session.PlanID) > 256 ||
		state.Session.StartedAtMs <= 0 || state.Session.EndedAtMs < 0 || state.SampledAtMs <= 0 || state.SavedAtMs <= 0 ||
		state.Session.Uplink < 0 || state.Session.Downlink < 0 {
		return state, errors.New("runtime state is invalid")
	}
	return state, nil
}

func archiveRuntimeState(path string, previous RuntimeSnapshot) error {
	if previous.Version == 0 {
		return nil
	}
	directory := filepath.Join(filepath.Dir(path), "runtime-sessions")
	if err := os.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.New("runtime archive directory is unavailable")
	}
	if info, err := os.Lstat(directory); err != nil || !info.IsDir() {
		return errors.New("runtime archive directory is unavailable")
	}
	if err := writeRuntimeState(filepath.Join(directory, previous.Session.ID+".json"), previous); err != nil {
		return errors.New("runtime archive write failed")
	}
	return nil
}

func writeRuntimeState(path string, state RuntimeSnapshot) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() || err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("runtime state is not a regular file")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".runtime-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	return replaceRuntimeState(file.Name(), path)
}
