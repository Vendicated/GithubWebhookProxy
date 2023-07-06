package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
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

func makeUniqueAvatarUrl(avatarUrl string) string {
	req, err := http.NewRequest("GET", avatarUrl, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Range", "bytes=200-250")
	res, err := http.DefaultClient.Do(req)

	if err != nil || res.StatusCode >= 300 {
		return ""
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}

	// Ugly but parsing URL and adding via "proper" method might alter the order of parameters
	pre := "?"
	if strings.Contains(avatarUrl, "?") {
		pre = "&"
	}
	avatarUrl += pre + "hash=" + url.QueryEscape(base64.StdEncoding.EncodeToString(body))

	return avatarUrl
}

func webhook(w http.ResponseWriter, r *http.Request) {
	if !verifyIp(r.Header.Get("X-Forwarded-For")) {
		w.WriteHeader(403)
		return
	}

	webhookUrl := r.URL.Query().Get("url")
	if webhookUrl == "" {
		w.WriteHeader(400)
		_, _ = fmt.Fprintln(w, "'webhook' parameter must be specified")
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		w.WriteHeader(400)
		return
	}

	body := r.Body
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

	if event == "watch" {
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
	}

	if payload.Sender.AvatarURL != "" {
		avatarUrl := makeUniqueAvatarUrl(payload.Sender.AvatarURL)
		if avatarUrl != "" {
			// Ugly but the alternative is unmarshalling to generic map which makes the rest of the code ugly (since we need to marshal it back with all fields)
			bytes.ReplaceAll(data, []byte(payload.Sender.AvatarURL), []byte(avatarUrl))
		}
	}

	body = io.NopCloser(bytes.NewReader(data))

	post, err := http.NewRequest("POST", webhookUrl, body)
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
