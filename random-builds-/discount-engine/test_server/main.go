package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ak-repo/discount-engine/internal/engine"
)

var testOrders = []engine.OrderInput{
	{
		// No discount applies
		OrderTotal:   40,
		CustomerType: "regular",
	},
	{
		// rule_3 applies (5% over $50)
		OrderTotal:   60,
		CustomerType: "regular",
	},
	{
		// rule_4 applies ($10 off for regular over $75)
		OrderTotal:   80,
		CustomerType: "regular",
	},
	{
		// rule_1 applies (10% over $100)
		OrderTotal:   120,
		CustomerType: "regular",
	},
	{
		// rule_2 applies ($20 off premium)
		OrderTotal:   60,
		CustomerType: "premium",
	},
	{
		// rule_1 vs rule_3 conflict → rule_1 wins
		OrderTotal:   150,
		CustomerType: "regular",
	},
	{
		// rule_1 vs rule_2 → max discount logic decides
		OrderTotal:   120,
		CustomerType: "premium",
	},
}

func main() {

	client := &http.Client{}

	for _, order := range testOrders {

		body, err := json.Marshal(order)
		if err != nil {
			log.Println("marshal error:", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPost,
			"http://localhost:8000/apply-discount",
			bytes.NewBuffer(body),
		)
		if err != nil {
			log.Println("request error:", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Println("http error:", err)
			continue
		}

		defer resp.Body.Close()

		var data engine.Response
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			log.Println("decode error:", err)
			continue
		}

		log.Printf("response: %+v\n", data)
	}
}
