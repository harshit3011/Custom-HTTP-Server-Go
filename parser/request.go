package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	maxBodySize        = 1 << 20  // 1 MiB
	maxRequestLineSize = 8 << 10  // 8 KiB
	maxHeaderLineSize  = 8 << 10  // 8 KiB
	maxHeaderBytes     = 32 << 10 // 32 KiB total
	maxHeaderCount     = 100
)

var (
	ErrLineTooLarge       = errors.New("line exceeds maximum size")
	ErrHeadersTooLarge    = errors.New("request headers exceed maximum size")
	ErrTooManyHeaders     = errors.New("too many request headers")
	ErrURITooLong         = errors.New("request line too long")
	ErrUnsupportedVersion = errors.New("unsupported HTTP version")
	ErrBodyTooLarge       = errors.New("request body exceeds maximum size")
	ErrUnsupportedTE      = errors.New("transfer-encoding is not supported")
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string][]string
	Body    io.Reader
}

func Parse(r io.Reader) (*Request, error) {
	reader := bufio.NewReader(r)
	line, err := readLine(reader, maxRequestLineSize)
	if err != nil {
		if errors.Is(err, ErrLineTooLarge) {
			return nil, ErrURITooLong
		}
		return nil, fmt.Errorf("reading request line: %w", err)
	}

	line = strings.TrimSuffix(line, "\r\n")
	parts := strings.Fields(line)

	if len(parts) != 3 || !isToken(parts[0]) || parts[1] == "" {
		return nil, fmt.Errorf("invalid request line")
	}

	if parts[2] != "HTTP/1.0" && parts[2] != "HTTP/1.1" {
		return nil, ErrUnsupportedVersion
	}

	headerMap := make(map[string][]string)
	headerBytes := 0
	headerCount := 0
	for {
		line, err := readLine(reader, maxHeaderLineSize)
		if err != nil {
			if errors.Is(err, ErrLineTooLarge) {
				return nil, ErrHeadersTooLarge
			}
			return nil, fmt.Errorf("reading request header: %w", err)
		}

		headerBytes += len(line)
		if headerBytes > maxHeaderBytes {
			return nil, ErrHeadersTooLarge
		}

		line = strings.TrimSuffix(line, "\r\n")

		if line == "" {
			break
		}

		headerCount++
		if headerCount > maxHeaderCount {
			return nil, ErrTooManyHeaders
		}

		name, value, found := strings.Cut(line, ":")
		if !found || !isToken(name) {
			return nil, fmt.Errorf("invalid header format")
		}

		headerName := strings.ToLower(name)
		headerValue := strings.TrimSpace(value)

		if !validHeaderValue(headerValue) {
			return nil, fmt.Errorf("invalid header value")
		}

		headerMap[headerName] = append(headerMap[headerName], headerValue)
	}

	req := &Request{
		Method:  parts[0],
		Target:  parts[1],
		Version: parts[2],
		Headers: headerMap,
		Body:    bytes.NewReader(nil),
	}

	if _, ok := headerMap["transfer-encoding"]; ok {
		return nil, ErrUnsupportedTE
	}

	contentLengths := headerMap["content-length"]
	if len(contentLengths) == 0 {
		return req, nil
	}

	if len(contentLengths) != 1 {
		return nil, fmt.Errorf("multiple content-length headers")
	}

	length, err := strconv.ParseInt(contentLengths[0], 10, 64)
	if err != nil || length < 0 {
		return nil, fmt.Errorf("invalid content-length")
	}

	if length > maxBodySize {
		return nil, ErrBodyTooLarge
	}

	body := make([]byte, int(length))
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, fmt.Errorf("incomplete request body: %w", err)
	}

	req.Body = bytes.NewReader(body)

	return req, nil
}

func readLine(reader *bufio.Reader, maxBytes int) (string, error) {
	var line []byte

	for {
		part, err := reader.ReadSlice('\n')
		if len(line)+len(part) > maxBytes {
			return "", ErrLineTooLarge
		}

		line = append(line, part...)

		if err == bufio.ErrBufferFull {
			continue
		}

		if err != nil {
			return "", err
		}

		if len(line) < 2 ||
			line[len(line)-2] != '\r' ||
			line[len(line)-1] != '\n' {
			return "", fmt.Errorf("invalid line ending")
		}

		return string(line), nil
	}
}

func isToken(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') {
			continue
		}

		if strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			continue
		}

		return false
	}

	return true
}

func validHeaderValue(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}

	return true
}
