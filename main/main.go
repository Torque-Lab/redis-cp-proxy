package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"redis-cp-proxy/auth_pipe"
	"redis-cp-proxy/control_plane"
	_ "redis-cp-proxy/env_config"
	"time"
)

var proxy_port = os.Getenv("PROXY_PORT")

func main() {
	certPath := "/etc/ssl/certs/tls.crt"
	keyPath := "/etc/ssl/certs/tls.key"
	go control_plane.StartUpdateServer()
	// TLS logic
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Println("failed to load cert:", err)
		return
	}
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		PreferServerCipherSuites: true,
	}

	listener, err := net.Listen("tcp", ":"+proxy_port)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Listening on: %s\n", proxy_port)
	tlsListener := tls.NewListener(listener, tlsConfig)

	for {
		conn, err := tlsListener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		if tc, ok := conn.(*tls.Conn); ok {
			tc.SetDeadline(time.Now().Add(15 * time.Second))
			if err := tc.Handshake(); err != nil {
				log.Println("TLS handshake failed:", err)
				conn.Close()
				continue
			}
			tc.SetDeadline(time.Time{})
		}

		go auth_pipe.HandleClient(conn)
	}
}
