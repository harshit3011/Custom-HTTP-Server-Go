package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to bind to port: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error while accepting connection:", err)
			continue
		}

		fmt.Println("Connected!")

		bytesTemp := make([]byte, 1024)
		var bytesRead []byte

		for {
			bys, err := conn.Read(bytesTemp)
			if err != nil {
				if err.Error() != "EOF" {
					log.Println("Error while reading:", err)
				}
				break
			}

			bytesRead = append(bytesRead, bytesTemp[:bys]...)

			if bytes.Contains(bytesRead, []byte("\r\n\r\n")) {
				break
			}
		}

		if !bytes.Contains(bytesRead, []byte("\r\n\r\n")) {
			log.Println("Incomplete request headers")
			conn.Close()
			continue
		}

		index := bytes.Index(bytesRead, []byte("\r\n"))
		if index == -1 {
			log.Println("Invalid request: request line not found")
			conn.Close()
			continue
		}

		bytesReq := bytesRead[:index]
		lastIndex := bytes.Index(bytesRead, []byte("\r\n\r\n"))

		bytesHead := bytesRead[index+2 : lastIndex]
		vals := strings.Fields(string(bytesReq))

		if len(vals) != 3 {
			log.Println("Invalid request line")
			conn.Close()
			continue
		}

		method := vals[0]
		path := vals[1]
		version := vals[2]

		fmt.Printf("No. of bytes read: %d\n", len(bytesRead))
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("Path: %s\n", path)
		fmt.Printf("Version: %s\n", version)

		headers := make(map[string]string)

		if len(bytesHead) > 0 {
			headerLines := strings.Split(string(bytesHead), "\r\n")

			for _, line := range headerLines {
				headerName, headerValue, found := strings.Cut(line, ":")
				if !found || strings.TrimSpace(headerName) == "" {
					log.Println("Invalid header:", line)
					continue
				}

				headerName = strings.TrimSpace(headerName)
				headerValue = strings.TrimSpace(headerValue)

				headers[headerName] = headerValue
			}
		}

		for name, value := range headers {
			fmt.Printf("Header: %s = %s\n", name, value)
		}

		fmt.Printf("Request:\n%s", string(bytesRead))

		body:=""
		status:=200
		statusText:="OK"

		switch{
		case method=="GET" && path=="/":
			body="This is the home route"
		case method=="GET" && path=="/health":
			body="OK"
		case method=="GET" && path=="/about":
			body="This is my custom HTTP Server built in go"
		default:
			body="404 page not found"
			status=404
			statusText="Not Found"
		}
		if err:=writeResponse(body,status,statusText,conn);err!=nil{
			log.Printf("Error while writing the response: %v",err)
		}
		if err:=conn.Close();err!=nil{
			log.Printf("Error while closing the connection: %v",err)
		}
	}
}

func writeResponse(body string, status int, statusText string, conn net.Conn) error{
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Type: text/plain; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		status,
		statusText,
		len(body),
		body,
	)
	return writeAll(conn,[]byte(response))
}

func writeAll(conn net.Conn,data []byte) error{
	for len(data)>0{
		n,err:=conn.Write(data)
		if err!=nil{
			return err
		} 
		if n>0{
			data=data[n:]
		} else{
			return io.ErrShortWrite
		}
	}
	return nil
}