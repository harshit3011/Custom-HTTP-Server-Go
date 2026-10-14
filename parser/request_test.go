package parser

import (
	"io"
	"strings"
	"testing"
)

func TestParse_RequestWithoutBody(t *testing.T) {
	input := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"

	req, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if req.Method != "GET" {
		t.Fatalf("Method = %q, want %q", req.Method, "GET")
	}
	if req.Target != "/" {
		t.Fatalf("Target = %q, want %q", req.Target, "/")
	}
	if req.Version != "HTTP/1.1" {
		t.Fatalf("Version = %q, want %q", req.Version, "HTTP/1.1")
	}

	if got := req.Headers["host"]; len(got) != 1 || got[0] != "localhost" {
		t.Fatalf("Headers[host] = %v, want [localhost]", got)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("Body = %q, want empty", string(body))
	}
}

func TestParse_RequestWithBody(t *testing.T) {
	input := "POST /submit HTTP/1.1\r\nContent-Length: 5\r\nContent-Type: text/plain\r\n\r\nhello"

	req, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("Body = %q, want %q", string(body), "hello")
	}

	if got := req.Headers["content-type"]; len(got) != 1 || got[0] != "text/plain" {
		t.Fatalf("Headers[content-type] = %v, want [text/plain]", got)
	}
}

func TestParse_InvalidRequestLine(t *testing.T) {
	_, err := Parse(strings.NewReader("GET /\r\n"))
	if err == nil || err.Error() != "invalid request line" {
		t.Fatalf("Parse() error = %v, want %q", err, "invalid request line")
	}
}

func TestParse_UnsupportedTransferEncoding(t *testing.T) {
	input := "GET / HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n"

	_, err := Parse(strings.NewReader(input))
	if err == nil || err.Error() != "transfer-encoding is not supported" {
		t.Fatalf("Parse() error = %v, want %q", err, "transfer-encoding is not supported")
	}
}

func TestParse_InvalidHeaderFormat(t *testing.T) {
	input := "GET / HTTP/1.1\r\nBadHeader\r\n\r\n"

	_, err := Parse(strings.NewReader(input))
	if err == nil || err.Error() != "invalid header format" {
		t.Fatalf("Parse() error = %v, want %q", err, "invalid header format")
	}
}

func TestParse_MultipleContentLengthHeaders(t *testing.T) {
	input := "POST / HTTP/1.1\r\nContent-Length: 3\r\nContent-Length: 4\r\n\r\nabc"

	_, err := Parse(strings.NewReader(input))
	if err == nil || err.Error() != "multiple content-length headers" {
		t.Fatalf("Parse() error = %v, want %q", err, "multiple content-length headers")
	}
}
