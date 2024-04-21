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

var remoteAddresses = map[string]string{
	"prod.gamev90.portalworldsgame.com":          "44.194.163.69",
	"ec2-54-92-129-196.compute-1.amazonaws.com":  "54.92.129.196",
	"ec2-54-198-33-148.compute-1.amazonaws.com":  "54.198.33.148",
	"ec2-54-162-199-227.compute-1.amazonaws.com": "54.162.199.227",
	"ec2-3-87-31-158.compute-1.amazonaws.com":    "3.87.31.158",
	"ec2-3-81-35-200.compute-1.amazonaws.com":    "3.81.35.200",
	"ec2-54-210-140-11.compute-1.amazonaws.com":  "54.210.140.11",
}

var remoteAddress = "44.194.163.69" // prod.gamev90.portalworldsgame.com

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

		_, err = dst.Write(buffer[:n])
		if err != nil {
			log.Fatal(err)
		}

		cmd := buffer[:4]
		raw := bson.Raw(buffer[4:n])
		if raw.Validate() == nil {
			log.Printf("Deserialized (%x): %s\n", cmd, raw)

			// Switch server
			if (cmd[0] == 0x6c || cmd[0] == 0x6d || cmd[0] == 0x6e || cmd[0] == 0x6f) && cmd[1] == 0x00 {
				remoteHostname := raw.Lookup("m0").Document().Lookup("IP").StringValue()
				remoteAddress = remoteAddresses[remoteHostname]
				log.Printf("Changing remote address to %s (%s)", remoteHostname, remoteAddress)
				break
			} else if cmd[0] == 0x9b && cmd[1] == 0x00 {
				remoteHostname := raw.Lookup("m1").Document().Lookup("IP").StringValue()
				remoteAddress = remoteAddresses[remoteHostname]
				log.Printf("Changing remote address to %s (%s)", remoteHostname, remoteAddress)
				break
			}

			// profile received
			if cmd[0] == 0x8a && cmd[1] == 0x0e && cmd[2] == 0x00 && cmd[3] == 0x00 {
				println("profile received")
			}
		}
	}
}

func handleConnection(localSocket net.Conn) {
	defer localSocket.Close()

	remoteSocket, err := net.Dial("tcp", remoteAddress+":10001")
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

	// file, err := os.OpenFile("proxy.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalf("error opening file: %v", err)
	// }
	// defer file.Close()

	// writer := io.MultiWriter(os.Stdout, file)
	// log.SetOutput(writer)

	for {
		connection, err := localSocket.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(connection)
	}
}
