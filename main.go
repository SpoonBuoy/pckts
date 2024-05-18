package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"pckts/client"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// config
var (
	totalClients       = 3
	ListenPort         = 9000
	ServeDashboardPort = ":8080"
)

//ClientStats -- will contain the data related to client and will be sent to the frontend dashboard

type ClientStats struct {
	//total packets received by the client till now
	TotalPacketsReceived uint64
	//packets received in one second
	PacketsReceivedInSec uint64
	//rate -- will be equal to the packets received in one second  -- we can remove it altogether
	Rate uint64
}

// ServerStats
type ServerStats struct {
	//aggregate packet rate of all clients
	TotalPacketRate uint64
	//total packets received from all clients
	TotalPacketsReceived uint64
}

// Our server
type Server struct {
	//Will have client controller
	ClientController *client.ClientController
	//no necessarily need to be here
	ClientStats map[int]*ClientStats
	//Since ClientStats is not thread safe
	ClientStatsMu sync.Mutex
	//Listener
	Listener net.Listener
	//Upgrader for ws
	Upgrader websocket.Upgrader
	//ServerStats
	Stats ServerStats
	Mu    sync.Mutex
}

// Returns new instance of server
func NewServer(cc *client.ClientController) *Server {
	return &Server{
		ClientController: cc,
		ClientStats:      make(map[int]*ClientStats),
		ClientStatsMu:    sync.Mutex{},
		Upgrader:         websocket.Upgrader{},
		Stats:            ServerStats{},
		Mu:               sync.Mutex{},
	}
}

// starts the server at some port
func (s *Server) StartServer(addr int) {
	for i := 1; i <= totalClients; i++ {
		s.ClientStats[i] = &ClientStats{}
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", addr))
	if err != nil {
		panic(err)
	}
	s.Listener = listener

}

// accept loop for incoming connections
func (s *Server) AcceptLoop() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			continue
		}
		go s.handleClient(conn)
	}
}

// Handle client will be used to handle the incoming packet from client
func (s *Server) handleClient(conn net.Conn) {
	defer conn.Close()
	//buffer to read the packet
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
		//ClientId would be the first byte off the packet
		clientID, _ := strconv.Atoi(string(buffer[:1]))
		//fmt.Printf("%s\n", buffer[:1])
		s.ClientStatsMu.Lock()
		//increment packetrecieved by one
		s.ClientStats[clientID].PacketsReceivedInSec++
		s.ClientStats[clientID].TotalPacketsReceived++
		s.ClientStatsMu.Unlock()

		//increment total packets received by server
		s.Mu.Lock()
		s.Stats.TotalPacketsReceived++
		s.Mu.Unlock()
	}
}

// statsHandler will be used for sending the client stats to the frontend dashboard using websocket every second
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		s.ClientStatsMu.Lock()
		for i := 1; i <= totalClients; i++ {
			//since rate is equal to packets received in one second
			//this is redundant but for the sake of completion
			//i am having a rate field as well
			s.ClientStats[i].Rate = s.ClientStats[i].PacketsReceivedInSec
		}

		stats := make(map[string]interface{})
		for i := 1; i <= totalClients; i++ {
			key := fmt.Sprintf("client%d", i)
			stats[key] = s.ClientStats[i]
			//add each client rate to the server packet rate
			s.Stats.TotalPacketRate += s.ClientStats[i].Rate
		}
		//server stats
		stats["server"] = s.Stats
		s.ClientStatsMu.Unlock()
		//marshall struct to byte
		data, _ := json.Marshal(stats)
		conn.WriteMessage(websocket.TextMessage, data)
		s.ClientStatsMu.Lock()
		//we have to set the PacketsRecievedInSec to 0 so that it contains only in the next second
		for i := 1; i <= 3; i++ {
			s.ClientStats[i].PacketsReceivedInSec = 0
		}
		s.ClientStatsMu.Unlock()
		//Set the server packet rate to 0 so that it contains only packets received in next second
		s.Mu.Lock()
		s.Stats.TotalPacketRate = 0
		s.Mu.Unlock()
		time.Sleep(1 * time.Second)

	}
}

// control handler will listen to the update packet rate for any client
// from the frontend dashboard using websocket
func (s *Server) controlHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		//got an update message
		_, msg, err := conn.ReadMessage()
		fmt.Println("Reading from ws")
		if err != nil {
			return
		}

		var message map[string]int
		json.Unmarshal(msg, &message)

		//read the message contents
		clientID := message["clientId"]
		rate := uint64(message["rate"])
		fmt.Printf("rate change for %d with %d \n", clientID, rate)

		//update the rate for that particular client in ClientController
		//the start client in ClientController will handle it further
		s.ClientController.ClientsMu.Lock()
		s.ClientController.Clients[clientID].Rate = rate
		s.ClientController.ClientsMu.Unlock()

	}
}

func main() {
	//new client controller with totalNumberOfClients
	c := client.NewClientController(totalClients)

	//new server
	s := NewServer(c)

	//start server at ListenPort
	go s.StartServer(ListenPort)
	//wait for server to start
	time.Sleep(time.Second * 1)
	//spawn accept loop
	go s.AcceptLoop()
	//wait for accept loop to spawn
	time.Sleep(time.Second * 2)

	//spawn client threads
	for i := 1; i <= totalClients; i++ {
		go c.StartClient(i, "ws://localhost:8080/control")
	}

	//routes for dashboard communication
	http.HandleFunc("/ws", s.statsHandler)
	http.HandleFunc("/control", s.controlHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	//serve dashboard on dashboard port
	http.ListenAndServe(ServeDashboardPort, nil)
}
