package main

import (
	"net"
	"os"
	"sync"
)

const (
	TransferTypeBroadcast = iota
	TransferTypeData
	TransferTypeAck
	TransferTypeRemove
)

var wg sync.WaitGroup
var forwardingTable ForwardingTable

func main() {

	forwardingTable = ForwardingTable{
		entries: make(map[string]Hop),
	}

	entityAddrList := getEntityAddrs()
	for _, addr := range entityAddrList {
		println("Entity address: ", addr)
	}
	sendAddrList := prepSendAddr(entityAddrList)

	listenAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:5000")
	if err != nil {
		println("Failed to resolve listening address: ", err.Error())
		os.Exit(0)
	}

	socket, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		println("Failed to establish socket: ", err.Error())
		os.Exit(0)
	}

	wg.Add(1)
	go receiveData(socket, sendAddrList, entityAddrList)
	wg.Wait()
}

func receiveData(socket *net.UDPConn, sendAddrList []*net.UDPAddr, entityAddrList []string) {
	defer wg.Done()

	for {
		buffer := make([]byte, 65000)

		n, senderAddr, err := socket.ReadFromUDP(buffer)
		if err != nil {
			println("Error reading from client: ", err.Error())
			os.Exit(0)
		}
		if senderAddrStr := senderAddr.String(); senderAddrStr != entityAddrList[0] && senderAddrStr != entityAddrList[1] {

			dataBuffer := make([]byte, n)
			copy(dataBuffer, buffer[:n])

			source, transferType, dest := decode(dataBuffer)
			switch transferType {
			case TransferTypeBroadcast:
				forwardingTable.AddRow(source, senderAddr)
				go broadcastData(socket, dataBuffer, senderAddr, sendAddrList)

			case TransferTypeData:
				nextHop, _ := forwardingTable.GetRow(dest)
				go sendDirectly(socket, dataBuffer, nextHop.IPAddress)

			case TransferTypeAck:
				forwardingTable.AddRow(source, senderAddr)
				nextHop, _ := forwardingTable.GetRow(dest)
				go sendDirectly(socket, dataBuffer, nextHop.IPAddress)

			case TransferTypeRemove:
				forwardingTable.RemoveRow(source)
				go broadcastData(socket, dataBuffer, senderAddr, sendAddrList)
			}
		}
	}
}

func broadcastData(socket *net.UDPConn, data []byte, senderAddr *net.UDPAddr, sendAddrList []*net.UDPAddr) {
	sendList := createSendingList(senderAddr, sendAddrList)
	for _, addr := range sendList {
		_, err := socket.WriteToUDP(data, addr)
		if err != nil {
			println("Error sending data: ", err.Error())
			continue
		}
	}
}

func sendDirectly(socket *net.UDPConn, buffer []byte, nextHop *net.UDPAddr) {
	_, err := socket.WriteToUDP(buffer, nextHop)
	if err != nil {
		println("Error sending data: ", err.Error())
	}
}
