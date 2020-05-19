// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"

	//"github.com/pion/rtcp"
	//"sync/atomic"
	"sync"

	//"github.com/pion/webrtc/pkg/media/oggwriter"
	//"time"

	"encoding/base64"
	//"github.com/pion/rtcp"
	"github.com/pion/webrtc"
	"github.com/pion/webrtc/pkg/media"
	//"github.com/pion/webrtc/pkg/media/oggwriter"
	"io"
	"os"
	"strings"
	//"time"
)

var audioTrack2 *webrtc.Track
var track *webrtc.Track
var mediaEngine webrtc.MediaEngine
var api *webrtc.API
var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
	SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
}

var (
	// Mutex for room
	roomLock = sync.RWMutex{}
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Rooms
	rooms map[string][]*Client
	roomsJoinCount map[string]int32

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	requestOffer chan *RequestOffer

	requestCandidate chan *RequestCandidate
}

// client send -> requestOffer
type RequestOffer struct {
	client *Client
	sdp string
}

type RequestCandidate struct {
	client *Client
	candidate string
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		requestOffer: make(chan *RequestOffer),
		requestCandidate: make(chan *RequestCandidate),
		rooms: make(map[string][]*Client),
		roomsJoinCount: make(map[string]int32),
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
		case message := <-h.requestCandidate:
			fmt.Print("requestCandidate", message)
		case message := <-h.requestOffer:
			var err error

			// Append the Client to room
			roomLock.Lock()
			h.rooms[message.client.room] = append(h.rooms[message.client.room], message.client)
			roomLock.Unlock()

			// Create PeerConnection
			message.client.rtcConnection, err = api.NewPeerConnection(peerConnectionConfig)
			checkError(err)

			// Add Remote Track to self When second user is offer
			//if len(h.rooms[message.client.room]) == 2 {
			//	for _, remoteClient := range h.rooms[message.client.room] {
			//		if remoteClient != message.client {
			//			message.client.sendTrack, err = message.client.rtcConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
			//			message.client.rtcConnection.AddTrack(message.client.sendTrack)
			//		}
			//	}
			//} else {
			//	message.client.sendTrack, err = message.client.rtcConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
			//	message.client.rtcConnection.AddTrack(message.client.sendTrack)
			//}

			message.client.sendTrack, err = message.client.rtcConnection.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
			message.client.rtcConnection.AddTrack(message.client.sendTrack)

			if _, err = message.client.rtcConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
				panic(err)
			}



			// Create File IO
			//oggFile, err := oggwriter.New("output.ogg", 48000, 2)

			// Listen Connection Status
			message.client.rtcConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
				//fmt.Printf("Connection State has changed %s \n", connectionState.String())
			})

			message.client.rtcConnection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
				message.client.recvTrack = remoteTrack

				if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeOpus {
					for _, remoteClient := range h.rooms[message.client.room] {
						if remoteClient != message.client {
							go func() {
								rtpBuf := make([]byte, 1400)
								for {
									i, err := message.client.recvTrack.Read(rtpBuf)
									checkError(err)
									_, err = remoteClient.sendTrack.Write(rtpBuf[:i])

									if err != io.ErrClosedPipe {
										checkError(err)
									}
								}
							}()
						}
					}

					//var otherClient *Client
					//if len(h.rooms[message.client.room]) == 2 {
					//	for _, remoteClient := range h.rooms[message.client.room] {
					//		if remoteClient != message.client {
					//			otherClient = remoteClient
					//		}
					//	}
					//}

					//if otherClient != nil {
					//	fmt.Printf("하잇또")
					//	go func() {
					//		rtpBuf := make([]byte, 1400)
					//		for {
					//			i, err := message.client.remoteTrack.Read(rtpBuf)
					//			checkError(err)
					//			_, err = otherClient.recvTrack.Write(rtpBuf[:i])
					//
					//			if err != io.ErrClosedPipe {
					//				checkError(err)
					//			}
					//		}
					//	}()
					//
					//	go func() {
					//		rtpBuf2 := make([]byte, 1400)
					//		for {
					//			i2, err := otherClient.remoteTrack.Read(rtpBuf2)
					//			checkError(err)
					//			_, err = message.client.recvTrack.Write(rtpBuf2[:i2])
					//
					//			if err != io.ErrClosedPipe {
					//				checkError(err)
					//			}
					//		}
					//	}()
					//}


					//fmt.Print(remoteTrack.SSRC())

					// Save my Audio Pub Track to Client.track
					//message.client.track = audioTrack
					//
					//if len(h.rooms[message.client.room]) == 2 {
					//	for _, remoteClient := range h.rooms[message.client.room] {
					//		if remoteClient != message.client {
					//			fmt.Printf("Check Member")
					//			fmt.Printf("ANSWER2")
					//			remoteClient.rtcConnection.AddTrack(audioTrack)
					//			//message.client.rtcConnection.AddTrack(audioTrack)
					//		}
					//	}
					//}


					// Send my rtp to my Audio Pub Track

					//if len(h.rooms[message.client.room]) == 2 {
					//
					//	var targetTrack *webrtc.Track
					//	for _, remoteClient := range h.rooms[message.client.room] {
					//		if remoteClient != message.client {
					//			targetTrack = remoteClient.track
					//		}
					//	}
					//
					//	rtpBuf := make([]byte, 1400)
					//	for {
					//		i, err := remoteTrack.Read(rtpBuf)
					//		checkError(err)
					//		_, err = targetTrack.Write(rtpBuf[:i])
					//
					//		if err != io.ErrClosedPipe {
					//			checkError(err)
					//		}
					//	}
					//} else {
					//	rtpBuf := make([]byte, 1400)
					//	for {
					//		i, err := remoteTrack.Read(rtpBuf)
					//		checkError(err)
					//		_, err = message.client.track.Write(rtpBuf[:i])
					//
					//		if err != io.ErrClosedPipe {
					//			checkError(err)
					//		}
					//	}
					//}


					//codec := remoteTrack.Codec()
					//if codec.Name == webrtc.Opus {
					//	fmt.Println("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
					//	saveToDisk(oggFile, remoteTrack)
					//}
				}

				//for _, remoteClient := range h.rooms[message.client.room] {
				//	if message.client != remoteClient {
				//		fmt.Printf("ADD TRACK")
				//		remoteClient.rtcConnection.AddTrack(remoteTrack)
				//		message.client.rtcConnection.AddTrack(remoteClient.track)
				//	}
				//}

				//rtpBuf := make([]byte, 1400)
				//for {
				//	fmt.Printf("READ SUCCESS")
				//	i, err := track.Read(rtpBuf)
				//	checkError(err)
				//	_, err = track.Write(rtpBuf[:i])
				//
				//	if err != io.ErrClosedPipe {
				//		checkError(err)
				//	}
				//
			});

			checkError(message.client.rtcConnection.SetRemoteDescription(
				webrtc.SessionDescription{
					SDP: message.sdp,
					Type: webrtc.SDPTypeOffer,
				}))

			// Create answer
			answer, err := message.client.rtcConnection.CreateAnswer(nil)
			checkError(err)

			if err = message.client.rtcConnection.SetLocalDescription(answer); err != nil {
				panic(err)
			}

			fmt.Printf("CREATE ANSWER")

			responseMap := make(map[string] interface{})
			responseMap["type"] = "requestOffer"
			responseMap["sdp"] = answer.SDP

			response, _ := json.Marshal(responseMap)
			message.client.send <- []byte(response)

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

func createAnswer() string {
	return "test"
}

func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

func saveToDisk(i media.Writer, track *webrtc.Track) {
	defer func() {
		if err := i.Close(); err != nil {
			panic(err)
		}
	}()

	for {
		rtpPacket, err := track.ReadRTP()
		if err != nil {
			panic(err)
		}
		if err := i.WriteRTP(rtpPacket); err != nil {
			panic(err)
		}
	}
}
