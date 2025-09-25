package auth_pipe

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"redis-cp-proxy/control_plane"
	"sync"
)

func HandleClient(client net.Conn) {
	defer client.Close()
	reader := bufio.NewReader(client)

	//first command to be AUTH (Redis)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Println("read error:", err)
		return
	}

	var username, password string
	if _, err := fmt.Sscanf(line, "*3\r\n$4\r\nAUTH\r\n$%*d\r\n%s\r\n$%*d\r\n%s\r\n", &username, &password); err != nil {
		client.Write([]byte("-ERR invalid AUTH format\r\n"))
		log.Println("failed to parse AUTH:", err)
		return
	}

	backendAddr, error := control_plane.GetBackendAddress(username, password)
	if error != nil {
		client.Write([]byte("-ERR invalid credentials\r\n"))
		return
	}

	client.Write([]byte("+OK\r\n"))
	backendConn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		log.Println("failed to connect backend:", err)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go proxyPipe(&wg, backendConn, client)
	go proxyPipe(&wg, client, backendConn)
	wg.Wait()
}

func proxyPipe(wg *sync.WaitGroup, src net.Conn, dst net.Conn) {
	defer wg.Done()
	io.Copy(dst, src)
}
