package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gosocketio "github.com/ambelovsky/gosf-socketio"
	"github.com/ambelovsky/gosf-socketio/transport"
)

const STREAMLABS_API_TOKEN = "STREAMLABS_API_TOKEN"

type Response struct {
	SocketToken string `json:"socket_token"`
}

func getSocketToken() string {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://streamlabs.com/api/v2.0/socket/token", nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+STREAMLABS_API_TOKEN)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatalln(err)
	}

	var response Response
	json.Unmarshal(body, &response)

	return response.SocketToken
}

func main() {
	token := getSocketToken()

	websocketTransport := transport.GetDefaultWebsocketTransport()
	websocketTransport.PingInterval = 5 * time.Second

	client, err := gosocketio.Dial(gosocketio.GetUrl("sockets.streamlabs.com", 443, true)+"&token="+token, websocketTransport)

	if err != nil {
		log.Printf("failed to subscribe to create client: %v", err)
		return
	}

	err = client.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		log.Printf("Streamlabs connected")
	})

	if err != nil {
		log.Printf("failed to subscribe to connection: %v", err)
		return
	}

	err = client.On("event", func(c *gosocketio.Channel, m map[string]interface{}) {

		message, ok := m["message"].([]interface{})
		if !ok || len(message) == 0 {
			return
		}

		details, ok := message[0].(map[string]interface{})
		if !ok {
			return
		}

		amount := details["amount"]
		currency := details["currency"]
		eventType := m["type"]
		log.Printf("Type: %v, Amount: %v, Currency: %v", eventType, amount, currency)
	})

	if err != nil {
		log.Printf("failed to subscribe to event: %v", err)
		return
	}

	err = client.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		log.Printf("Streamlabs disconnected")
	})

	if err != nil {
		log.Printf("failed to subscribe to disconnection: %v", err)
		return
	}

	// Wait for a termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
