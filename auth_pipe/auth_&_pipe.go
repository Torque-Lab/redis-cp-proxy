package auth_pipe

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"redis-cp-proxy/control_plane"
	"strings"
	"sync"
)

func HandleClient(client net.Conn) {
	defer client.Close()
	reader := bufio.NewReader(client)

	cmd, err := readRESPArray(reader)
	if err != nil {
		log.Println("parse error:", err)
		client.Write([]byte("-ERR invalid RESP format\r\n"))
		return
	}

	if len(cmd) == 0 {
		client.Write([]byte("-ERR empty command\r\n"))
		return
	}

	command := strings.ToUpper(string(cmd[0]))
	var username, password string
	switch command {
	case "AUTH":
		if len(cmd) == 2 {
			password = string(cmd[1])
			fmt.Println("AUTH with password:", password)
		} else if len(cmd) == 3 {
			username = string(cmd[1])
			password = string(cmd[2])
			fmt.Println("AUTH with username:", username, "password:", password)
		} else {
			client.Write([]byte("-ERR wrong number of arguments for 'AUTH'\r\n"))
			return
		}
	default:
		client.Write([]byte("-ERR unknown command\r\n"))
	}
	backendAddr, error := control_plane.GetBackendAddress(username, password)
	if error != nil {
		client.Write([]byte("-ERR invalid credentials\r\n"))
		return
	}
	if backendAddr == "" {
		client.Write([]byte("-ERR invalid credentials\r\n"))
		return
	}
	backendConnRedis, err := net.Dial("tcp", backendAddr)
	if err != nil {
		client.Write([]byte("-ERR failed to connect backend\r\n"))
		log.Println("failed to connect backend:", err)
		return
	}
	//sending ok to real client after connecting real backend redis
	client.Write([]byte("+OK\r\n"))
	defer backendConnRedis.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go proxyPipe(client, backendConnRedis, &wg) // copy client -> backend
	go proxyPipe(backendConnRedis, client, &wg) // copy backend -> client
	wg.Wait()
}

func proxyPipe(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	_, _ = io.Copy(dst, src)
	if tcpConn, ok := dst.(*net.TCPConn); ok {
		_ = tcpConn.CloseWrite()
	}
	if tcpConn, ok := src.(*net.TCPConn); ok {
		_ = tcpConn.CloseRead()
	}
}

func readRESPArray(r *bufio.Reader) ([][]byte, error) {
	header, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(header) < 3 || header[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %q", header)
	}

	var count int
	_, err = fmt.Sscanf(string(header), "*%d", &count)
	if err != nil {
		return nil, fmt.Errorf("invalid array header: %v", err)
	}

	result := make([][]byte, 0, count)

	for i := 0; i < count; i++ {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if len(line) < 3 || line[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got: %q", line)
		}

		var length int
		_, err = fmt.Sscanf(string(line), "$%d", &length)
		if err != nil {
			return nil, fmt.Errorf("invalid bulk string header: %v", err)
		}

		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("failed to read bulk data: %v", err)
		}
		crlf := make([]byte, 2)
		if _, err := io.ReadFull(r, crlf); err != nil {
			return nil, fmt.Errorf("failed to read CRLF: %v", err)
		}

		result = append(result, data)
	}
	return result, nil
}
