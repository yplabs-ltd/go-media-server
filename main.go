
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func getUserInformation(token string) User {
	req, err := http.NewRequest("GET", "http://api.yplabs.net/account/v2/admin/user/", nil)
	req.Header.Add("Authorization", "Token " + token)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	bytes, _ := ioutil.ReadAll(resp.Body)
	var objmap map[string]interface{}
	json.Unmarshal(bytes, &objmap)
	resp.Body.Close()

	return User {
		email: objmap["email"].(string),
		nickname: objmap["nickname"].(string),
		sex: objmap["sex"].(string),
	}
}

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("HI")
		token := r.URL.Query().Get("token")
		room := r.URL.Query().Get("room")

		user := getUserInformation(token)
		serveWs(hub, user, room, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}