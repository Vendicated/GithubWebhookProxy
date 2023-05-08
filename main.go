package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

var webhookIpNets []*net.IPNet

func init() {
	// https://api.github.com/meta
	for _, cidr := range []string{
		"192.30.252.0/22",
		"185.199.108.0/22",
		"140.82.112.0/20",
		"143.55.64.0/20",
		"2a0a:a440::/29",
		"2606:50c0::/32",
	} {
		_, ipNet, _ := net.ParseCIDR(cidr)
		webhookIpNets = append(webhookIpNets, ipNet)
	}
}

type Payload struct {
	Action     string     `json:"action"`
	Sender     Sender     `json:"sender"`
	Repository Repository `json:"repository"`
}
type Sender struct {
	Login string `json:"login"`
}

type Repository struct {
	Id int `json:"id"`
}

var starGazers = make(map[string]bool)

func mainPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func verifyIp(ipString string) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false
	}

	for _, network := range webhookIpNets {
		if network.Contains(ip) {
			return true
		}
	}
	
	return false
}

func webhook(w http.ResponseWriter, r *http.Request) {
	if !verifyIp(r.Header.Get("X-Forwarded-For")) {
		w.WriteHeader(403)
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		w.WriteHeader(400)
		_, _ = fmt.Fprintln(w, "'webhook' parameter must be specified")
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	body := r.Body

	if event == "watch" {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		var payload Payload
		err = json.Unmarshal(data, &payload)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		if payload.Action == "started" {
			key := payload.Sender.Login + strconv.Itoa(payload.Repository.Id)
			if starGazers[key] {
				w.WriteHeader(200)
				return
			} else {
				starGazers[key] = true
				time.AfterFunc(15*time.Minute, func() {
					delete(starGazers, key)
				})
			}
		}
		body = io.NopCloser(bytes.NewReader(data))
	}

	post, err := http.NewRequest("POST", url, body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	post.Header = r.Header.Clone()
	res, err := http.DefaultClient.Do(post)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}

func main() {
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/webhook", webhook)

	log.Fatal(http.ListenAndServe(":1337", nil))
}
