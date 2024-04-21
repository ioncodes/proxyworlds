package main

import (
	"io"
	"log"
	"net"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

type ConnectionType int8

const (
	LOCAL_TO_REMOTE ConnectionType = 0
	REMOTE_TO_LOCAL ConnectionType = 1
)

func handleData(src, dst net.Conn, wg *sync.WaitGroup, connectionType ConnectionType) {
	defer wg.Done()

	buffer := make([]byte, 1024*1024)

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		} else if err == io.EOF {
			break
		}

		if connectionType == LOCAL_TO_REMOTE {
			log.Printf("Relaying packet to remote (%d bytes): %x", n, buffer[:n])
		} else {
			log.Printf("Relaying packet to process (%d bytes): %x", n, buffer[:n])
		}

		var raw bson.Raw = buffer[4:n]

		if raw.Validate() == nil {
			log.Println(raw)
		}

		_, err = dst.Write(buffer[:n])
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleConnection(localSocket net.Conn) {
	defer localSocket.Close()

	remoteSocket, err := net.Dial("tcp", "44.194.163.69:10001")
	if err != nil {
		log.Fatal(err)
	}
	defer remoteSocket.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go handleData(localSocket, remoteSocket, &wg, LOCAL_TO_REMOTE)
	go handleData(remoteSocket, localSocket, &wg, REMOTE_TO_LOCAL)

	wg.Wait()
}

func main() {
	localSocket, err := net.Listen("tcp", "127.0.0.1:10001")
	if err != nil {
		log.Fatal(err)
	}
	defer localSocket.Close()

	for {
		connection, err := localSocket.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(connection)
	}
}
