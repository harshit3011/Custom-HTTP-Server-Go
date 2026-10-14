
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/harshit3011/Custom-HTTP-Server-Go/middleware"
	"github.com/harshit3011/Custom-HTTP-Server-Go/parser"
	"github.com/harshit3011/Custom-HTTP-Server-Go/response"
	"github.com/harshit3011/Custom-HTTP-Server-Go/router"
)

const (
	maxConcurrentConnections = 100
	readTimeout              = 5 * time.Second
	writeTimeout             = 5 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
	defer listener.Close()

	r := router.New()

	if err := r.Handle("GET", "/", func(w *response.ResponseWriter, req *parser.Request) {
		w.Text(200, "Welcome to my custom HTTP server!\n")
	}); err != nil {
		log.Fatal(err)
	}

	if err := r.Handle("GET", "/health", func(w *response.ResponseWriter, req *parser.Request) {
		w.Text(200, "OK\n")
	}); err != nil {
		log.Fatal(err)
	}

	if err := r.Handle("GET", "/about", func(w *response.ResponseWriter, req *parser.Request) {
		if err := w.JSON(200, map[string]string{
			"message": "My custom HTTP server built in Go",
		}); err != nil {
			log.Printf("failed to encode response: %v", err)
			w.Text(500, "Internal Server Error\n")
		}
	}); err != nil {
		log.Fatal(err)
	}

	metrics := &middleware.Metrics{}

	r.Use(
		middleware.Logging,
		middleware.RequestID,
		metrics.Middleware,
		middleware.Recover,
	)

	semaphore := make(chan struct{}, maxConcurrentConnections)
	var wg sync.WaitGroup

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	log.Println("Custom HTTP server listening on :8080")

acceptLoop:
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}

			log.Printf("failed to accept connection: %v", err)
			continue
		}

		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			_ = conn.Close()
			break acceptLoop
		}

		if ctx.Err() != nil {
			<-semaphore
			_ = conn.Close()
			break
		}

		wg.Add(1)

		go func(conn net.Conn) {
			defer wg.Done()
			defer func() {
				<-semaphore
			}()

			handleConnection(conn, r)
		}(conn)
	}

	log.Println("Server shutting down; waiting for active connections...")
	wg.Wait()

	snapshot := metrics.Snapshot()
	log.Printf(
		"metrics: requests=%d server_errors=%d total_duration=%s",
		snapshot.Requests,
		snapshot.ServerErrors,
		snapshot.TotalDuration,
	)

	log.Println("Server shut down cleanly")
}

func handleConnection(conn net.Conn, r *router.Router) {
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("failed to set read deadline: %v", err)
		return
	}

	req, err := parser.Parse(conn)
	if err != nil {
		status, statusText := statusForParseError(err)

		log.Printf("request parsing failed: status=%d error=%v", status, err)

		w := response.NewResponseWriter()
		w.Text(status, statusText+"\n")

		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			log.Printf("failed to set write deadline: %v", err)
			return
		}

		if err := writeResponse(w, conn); err != nil {
			log.Printf("failed to write error response: %v", err)
		}

		return
	}

	w := r.Serve(req)

	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		log.Printf("failed to set write deadline: %v", err)
		return
	}

	if err := writeResponse(w, conn); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func statusForParseError(err error) (int, string) {
	switch {
	case errors.Is(err, parser.ErrURITooLong):
		return 414, "URI Too Long"

	case errors.Is(err, parser.ErrBodyTooLarge):
		return 413, "Content Too Large"

	case errors.Is(err, parser.ErrHeadersTooLarge),
		errors.Is(err, parser.ErrTooManyHeaders):
		return 431, "Request Header Fields Too Large"

	case errors.Is(err, parser.ErrUnsupportedTE):
		return 501, "Not Implemented"

	case errors.Is(err, parser.ErrUnsupportedVersion):
		return 505, "HTTP Version Not Supported"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return 408, "Request Timeout"
	}

	return 400, "Bad Request"
}

func writeResponse(w *response.ResponseWriter, conn net.Conn) error {
	var buf bytes.Buffer

	statusCode := w.StatusCode()
	statusText := w.StatusText()

	if _, err := fmt.Fprintf(
		&buf,
		"HTTP/1.1 %d %s\r\n",
		statusCode,
		statusText,
	); err != nil {
		return err
	}

	headers := w.Header()

	if err := headers.Set(
		"Content-Length",
		strconv.Itoa(len(w.BodyBytes())),
	); err != nil {
		return err
	}

	if err := headers.Set("Connection", "close"); err != nil {
		return err
	}

	for name, value := range headers {
		if !isValidResponseHeader(name, value) {
			return fmt.Errorf("invalid response header: %q", name)
		}

		if _, err := fmt.Fprintf(&buf, "%s: %s\r\n", name, value); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("\r\n"); err != nil {
		return err
	}

	if _, err := buf.Write(w.BodyBytes()); err != nil {
		return err
	}

	return writeAll(conn, buf.Bytes())
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}

		if n == 0 {
			return io.ErrShortWrite
		}

		data = data[n:]
	}

	return nil
}

func isValidResponseHeader(name, value string) bool {
	if name == "" {
		return false
	}

	for i := 0; i < len(name); i++ {
		if !isTokenChar(name[i]) {
			return false
		}
	}

	for i := 0; i < len(value); i++ {
		c := value[i]

		if (c < 32 && c != '\t') || c == 127 {
			return false
		}
	}

	return !strings.ContainsAny(name, "\r\n")
}

func isTokenChar(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}

	if c >= 'A' && c <= 'Z' {
		return true
	}

	if c >= '0' && c <= '9' {
		return true
	}

	return strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))
}
