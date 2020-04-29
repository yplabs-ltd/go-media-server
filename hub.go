// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"encoding/json"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	requestOffer chan *RequestOfferReceive
}

type RequestOfferReceive struct {
	client *Client
	offer []byte
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		requestOffer: make(chan *RequestOfferReceive),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			client.send <- createNeedOffer()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.requestOffer:
			fmt.Printf(string(message.offer))
			_ = message.offer
		case message := <-h.broadcast:
			fmt.Println(message)
			for client := range h.clients {
				select {
				case client.send <- message:
					fmt.Println("GOOD!")
				default:
					fmt.Println("NO")
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func createNeedOffer() []byte {
	response := make(map[string]string)
	response["type"] = "NeedOffer"

	jsonString, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	return jsonString
}