package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Header map[string]string

func (h Header) Set(name, value string) error {
	if !isToken(name) {
		return fmt.Errorf("Invalid header name")
	}

	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return fmt.Errorf("Invalid header values")
		}
	}
	h[name] = value
	return nil
}

func (h Header) Get(name string) string {
	for key, value := range h {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

type ResponseWriter struct {
	statusCode int
	headers    Header
	body bytes.Buffer
	wroteHeader bool
}

func NewResponseWriter() *ResponseWriter{
	w:=&ResponseWriter{
		statusCode: 200,
		headers: make(Header),
	}
	_ = w.headers.Set("Content-Type","text/plain; charset=utf-8")
	return w
}


func (w *ResponseWriter) Header() Header {
	return w.headers
}

func (w *ResponseWriter) WriteHeader(status int){
	if w.wroteHeader{
		return
	}

	if status<200 || status>599{
		status=500
	}
	w.statusCode=status
	w.wroteHeader=true
}

func (w *ResponseWriter) Write(p []byte) (int, error){
	if !w.wroteHeader{
		w.WriteHeader(200)
	}
	return w.body.Write(p)
}


func (w *ResponseWriter) Text(status int, body string) {
	_ = w.headers.Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (w *ResponseWriter) JSON(status int, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_ = w.headers.Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

func (w *ResponseWriter) StatusCode() int {
	return w.statusCode
}

func (w *ResponseWriter) BodyBytes() []byte {
	return w.body.Bytes()
}

func (w *ResponseWriter) Headers() Header {
	return w.headers
}

func (w *ResponseWriter) Reset(status int, contentType, body string) {
	w.statusCode = status
	w.wroteHeader = true
	w.body.Reset()
	w.headers = make(Header)
	_ = w.headers.Set("Content-Type", contentType)
	_, _ = w.body.WriteString(body)
}

func (w *ResponseWriter) StatusText() string {
	switch w.statusCode {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 408:
		return "Request Timeout"
	case 413:
		return "Content Too Large"
	case 414:
		return "URI Too Long"
	case 431:
		return "Request Header Fields Too Large"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 505:
		return "HTTP Version Not Supported"
	default:
		return "Unknown Status"
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
