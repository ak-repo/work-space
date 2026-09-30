package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"
)

type fixture struct {
	Title string
	State map[string]any
}

var fixtures = []fixture{
	{"Payment API outage", map[string]any{"message": "Payment API unavailable for many users", "service": "payments", "impact": "many users"}},
	{"Suspicious login", map[string]any{"message": "Suspicious login and leaked token reported", "account": "demo"}},
	{"SDK integration failure", map[string]any{"message": "Customer SDK integration returns API error after upgrade"}},
	{"Duplicate charge", map[string]any{"message": "Customer reports duplicate subscription charge and asks for refund"}},
	{"Vague request", map[string]any{"message": "Something is not right, please help"}},
}

func main() {
	count := flag.Int("count", 20, "number of events")
	base := flag.String("url", "http://localhost:8081", "server base URL")
	flag.Parse()
	c := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < *count; i++ {
		f := fixtures[rand.IntN(len(fixtures))]
		payload := map[string]any{"source": "simulator", "event_type": "demo", "title": f.Title, "state": f.State}
		b, _ := json.Marshal(payload)
		r, err := c.Post(*base+"/api/v1/events", "application/json", bytes.NewReader(b))
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		r.Body.Close()
		fmt.Printf("%02d %s -> %s\n", i+1, f.Title, r.Status)
		time.Sleep(250 * time.Millisecond)
	}
}
