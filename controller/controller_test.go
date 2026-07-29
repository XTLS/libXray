package controller

import (
	"errors"
	"testing"
)

type rawConnStub struct {
	fd         uintptr
	controlErr error
}

func (conn rawConnStub) Control(callback func(fd uintptr)) error {
	if conn.controlErr != nil {
		return conn.controlErr
	}
	callback(conn.fd)
	return nil
}

func (rawConnStub) Read(func(fd uintptr) (done bool)) error {
	return nil
}

func (rawConnStub) Write(func(fd uintptr) (done bool)) error {
	return nil
}

func TestProtectSocketPropagatesControllerFailure(t *testing.T) {
	err := protectSocket(rawConnStub{fd: 42}, func(fd uintptr) bool {
		return fd != 42
	})
	if !errors.Is(err, errProtectSocket) {
		t.Fatalf("protectSocket() error = %v, want %v", err, errProtectSocket)
	}
}

func TestProtectSocketPropagatesRawConnFailure(t *testing.T) {
	controlErr := errors.New("control failed")
	err := protectSocket(
		rawConnStub{controlErr: controlErr},
		func(uintptr) bool { return true },
	)
	if !errors.Is(err, controlErr) {
		t.Fatalf("protectSocket() error = %v, want %v", err, controlErr)
	}
}

func TestProtectSocketSucceeds(t *testing.T) {
	err := protectSocket(rawConnStub{fd: 42}, func(fd uintptr) bool {
		return fd == 42
	})
	if err != nil {
		t.Fatalf("protectSocket() error = %v", err)
	}
}
