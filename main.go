package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: snap <file>")
		os.Exit(1)
	}

	path := os.Args[1]
	if err := run(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("directories not supported")
	}

	ip, err := localip()
	if err != nil {
		return err
	}

	port, listener, err := listen()
	if err != nil {
		return err
	}

	name := filepath.Base(absPath)
	url := fmt.Sprintf("http://%s:%d", ip, port)

	done := make(chan struct{})

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:system-ui;background:#0a0a0a;color:#fff}
a{display:block;padding:1rem 2rem;background:#fff;color:#000;text-decoration:none;border-radius:8px;font-weight:500}
</style>
</head>
<body>
<a href="/file" download="%s">download %s</a>
</body>
</html>`, name, name, name)
	})

	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(absPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))

		http.ServeContent(w, r, name, info.ModTime(), file)

		if r.Header.Get("Range") == "" {
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  300 * time.Second,
	}

	go server.Serve(listener)

	fmt.Printf("\n  %s\n\n", url)
	qrterminal.GenerateWithConfig(url, qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stdout,
		HalfBlocks: true,
		BlackChar:  qrterminal.BLACK_BLACK,
		WhiteChar:  qrterminal.WHITE_WHITE,
		QuietZone:  2,
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-done:
		time.Sleep(2 * time.Second)
	case <-sig:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	return nil
}

func localip() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func listen() (int, net.Listener, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, err
	}
	return listener.Addr().(*net.TCPAddr).Port, listener, nil
}
