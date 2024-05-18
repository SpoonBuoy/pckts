package client

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type ClientStats struct {
	Rate uint64
}

// ClientController will be keeping track of clients with their clientId along with rate
type ClientController struct {
	ServerPort int
	Clients    map[int]*ClientStats
	ClientsMu  sync.Mutex
}

// returns new clientController
func NewClientController(totalClients int, srvPort int) *ClientController {
	c := make(map[int]*ClientStats)
	for i := 1; i <= totalClients; i++ {
		cs := ClientStats{Rate: 0}
		c[i] = &cs
	}
	return &ClientController{Clients: c, ServerPort: srvPort}
}

// Client logic
func (c *ClientController) StartClient(clientID int, controlURL string) {
	//connect to server on port 9000
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", c.ServerPort))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for {
		c.ClientsMu.Lock()
		//check the corresponding packet rate of client
		rate := c.Clients[clientID].Rate
		c.ClientsMu.Unlock()
		if rate > 0 {
			//divide second into appropriate tickers
			//tick duration will containg the ticker interval
			tickDuration := time.Second / time.Duration(rate)
			fmt.Printf("Tick %v for Client %d", tickDuration, clientID)
			if tickDuration <= 0 {
				//tickDuration should be greater than 0
				//otherwise it would crash
				fmt.Println("Tick Duration can't be less than 0, need to decrease the packet size")
				continue
			}
			ticker := time.NewTicker(tickDuration)
			for range ticker.C {
				//send packet with first byte as ClientID so that server identifies this client
				packet := fmt.Sprintf("%ddata", clientID)
				_, err := conn.Write([]byte(packet))
				if err != nil {
					fmt.Printf("error in writing: %s", err.Error())
					//set the client rate to 0
					c.ClientsMu.Lock()
					c.Clients[clientID].Rate = 0
					c.ClientsMu.Unlock()
					break
				}

				//if rate changes/updates during the time we need to create a new ticker then
				if rate != c.Clients[clientID].Rate {
					ticker.Stop()
					break
				}
			}
		}
	}
}
