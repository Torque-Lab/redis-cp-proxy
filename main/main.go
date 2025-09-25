package main

import (
	"crypto/tls"
	"log"
	"net"
	"redis-cp-proxy/auth_pipe"
	"redis-cp-proxy/control_plane"
	"time"
)

func main() {

	go control_plane.StartUpdateServer()
	// TLS logic
	cert, err := tls.LoadX509KeyPair("wildcard.crt", "wildcard.key")
	if err != nil {
		log.Println("failed to load cert:", err)
	}
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		PreferServerCipherSuites: true,
	}

	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		log.Fatal(err)
	}
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
