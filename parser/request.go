package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxBodySize = 1 << 20 // 1 MiB

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string][]string
	Body    io.Reader
}

func Parse(r io.Reader) (*Request, error) {
	reader := bufio.NewReader(r)

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimSuffix(line, "\r\n")
	parts := strings.Fields(line)

	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	headerMap := make(map[string][]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("incomplete request headers")
			}
			return nil, err
		}

		line = strings.TrimSuffix(line, "\r\n")

		if line == "" {
			break
		}

		lineBefore, lineAfter, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(lineBefore) == "" {
			return nil, fmt.Errorf("invalid header format")
		}

		headerName := strings.ToLower(strings.TrimSpace(lineBefore))
		headerValue := strings.TrimSpace(lineAfter)

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
		return nil, fmt.Errorf("transfer-encoding is not supported")
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
		return nil, fmt.Errorf("request body exceeds maximum size")
	}

	body := make([]byte, int(length))
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, fmt.Errorf("incomplete request body: %w", err)
	}

	req.Body = bytes.NewReader(body)

	return req, nil
}
