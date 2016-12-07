package dev

import (
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

func TestHTTP1(t *testing.T) {

	t.Logf("TestHTTP1: starting")

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerHTTP(t, addr)
	if listenErr != nil {
		t.Errorf("could not spawn bogus HTTP server: %v", listenErr)
	}
	t.Logf("TestHTTP1: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "http", "lab1", "localhost"+addr, "", "", "", "", false, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	go Spawner(tab, logger, requestCh, repo, repo, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func spawnServerHTTP(t *testing.T, addr string) (*testServer, error) {

	t.Logf("spawnServerHTTP: will listen on %s", addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	t.Logf("spawnServerHTTP: listening on %s", addr)

	s := &testServer{listener: ln, done: make(chan int)}

	m := http.NewServeMux()
	m.HandleFunc("/", rootHandler) // default handler

	go func() {
		err := http.Serve(ln, m)
		t.Logf("spawnServerHTTP: http.Serve: exited: %v", err)
		close(s.done)
	}()

	return s, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello web client\n")
}
