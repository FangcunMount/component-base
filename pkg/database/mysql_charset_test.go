package database

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// Exercise authentication and the actual initialization commands: charset is
// SET NAMES, never a server system variable. No real database or DDL is used.
func TestConnectorPreservesCharsetAndSessionInitialization(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := make(chan []string, 1)
	go func() {
		var statements []string
		defer func() { commands <- statements }()
		conn, e := listener.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		write := func(sequence byte, data []byte) bool {
			_, e := conn.Write(append([]byte{byte(len(data)), byte(len(data) >> 8), byte(len(data) >> 16), sequence}, data...))
			return e == nil
		}
		read := func() ([]byte, error) {
			h := make([]byte, 4)
			if _, e := io.ReadFull(conn, h); e != nil {
				return nil, e
			}
			n := int(h[0]) | int(h[1])<<8 | int(h[2])<<16
			data := make([]byte, n)
			_, e := io.ReadFull(conn, data)
			return data, e
		}
		caps := uint32(0x200 | 0x8000 | 0x80000)
		greeting := append([]byte{10}, []byte("8.0.36-test\x00")...)
		greeting = binary.LittleEndian.AppendUint32(greeting, 1)
		greeting = append(greeting, []byte("12345678\x00")...)
		greeting = append(greeting, byte(caps), byte(caps>>8), 45, 2, 0, byte(caps>>16), byte(caps>>24), 21)
		greeting = append(greeting, make([]byte, 10)...)
		greeting = append(greeting, []byte("123456789012\x00mysql_native_password\x00")...)
		if !write(0, greeting) {
			return
		}
		if _, e := read(); e != nil {
			return
		}
		ok := []byte{0, 0, 0, 2, 0, 0, 0}
		if !write(2, ok) {
			return
		}
		for {
			packet, e := read()
			if e != nil || len(packet) == 0 {
				return
			}
			if packet[0] == 1 {
				return
			}
			if packet[0] == 3 {
				statements = append(statements, string(packet[1:]))
			}
			if !write(1, ok) {
				return
			}
		}
	}()
	connector, err := NewMySQLConnector(&MySQLConfig{Host: listener.Addr().String(), Username: "test", Database: "test", TLSMode: "DISABLED", Location: "Etc/GMT-8", SessionTimeZone: "+08:00"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := connector.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	got := <-commands
	if len(got) != 2 || !strings.EqualFold(got[0], "SET NAMES utf8") || !strings.Contains(got[1], "time_zone") || !strings.Contains(got[1], "+08:00") {
		t.Fatalf("connection initialization = %q", got)
	}
}
