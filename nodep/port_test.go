package nodep

import (
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestGetFreePorts(t *testing.T) {
	excluded := []int{18587, 18587, 9000}
	ports, err := GetFreePorts(8, excluded)
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 8 {
		t.Fatalf("ports = %v, want 8", ports)
	}
	seen := map[int]bool{18587: true, 9000: true}
	for _, port := range ports {
		if port < 1 || port > 65535 || seen[port] {
			t.Fatalf("invalid, excluded or duplicate port: %d", port)
		}
		seen[port] = true
	}
}

func TestGetFreePortsHoldsExcludedAndSelectedListeners(t *testing.T) {
	listeners := []*portListener{{port: 18587}, {port: 50001}, {port: 50002}}
	calls := 0
	ports, err := getFreePorts(2, []int{18587}, func() (net.Listener, error) {
		for _, previous := range listeners[:calls] {
			if previous.closed {
				t.Fatal("listener released before selection completed")
			}
		}
		listener := listeners[calls]
		calls++
		return listener, nil
	})
	if err != nil || !reflect.DeepEqual(ports, []int{50001, 50002}) {
		t.Fatalf("ports = %v, error = %v", ports, err)
	}
	for _, listener := range listeners {
		if !listener.closed {
			t.Fatal("listener not released after selection")
		}
	}
}

func TestGetFreePortsClosesListenersOnError(t *testing.T) {
	for _, first := range []int{18587, 50001} {
		listener := &portListener{port: first}
		failure := errors.New("no more ports")
		calls := 0
		_, err := getFreePorts(2, []int{18587}, func() (net.Listener, error) {
			calls++
			if calls == 1 {
				return listener, nil
			}
			return nil, failure
		})
		if !errors.Is(err, failure) || !listener.closed {
			t.Fatalf("error = %v, closed = %v", err, listener.closed)
		}
	}
}

func TestGetFreePortsRejectsImpossibleRequestsBeforeListening(t *testing.T) {
	for _, request := range []struct {
		count   int
		exclude []int
	}{
		{count: -1},
		{count: 65536},
		{count: 65535, exclude: []int{18587}},
		{count: 1, exclude: []int{0}},
		{count: 1, exclude: []int{-1}},
		{count: 1, exclude: []int{65536}},
	} {
		_, err := getFreePorts(request.count, request.exclude, func() (net.Listener, error) {
			t.Fatal("invalid request opened a listener")
			return nil, nil
		})
		if err == nil {
			t.Fatalf("request %+v succeeded", request)
		}
	}
}

func TestGetFreePortsZeroDoesNotListen(t *testing.T) {
	ports, err := getFreePorts(0, nil, func() (net.Listener, error) {
		t.Fatal("zero-port request opened a listener")
		return nil, nil
	})
	if err != nil || len(ports) != 0 {
		t.Fatalf("ports = %v, error = %v", ports, err)
	}
}

type portListener struct {
	port   int
	closed bool
}

func (l *portListener) Accept() (net.Conn, error) { return nil, errors.New("unused") }
func (l *portListener) Addr() net.Addr            { return &net.TCPAddr{Port: l.port} }
func (l *portListener) Close() error {
	l.closed = true
	return nil
}
