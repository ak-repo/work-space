package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ak-repo/discount-engine/internal/engine"
	"github.com/ak-repo/discount-engine/internal/store"
)

func main() {

	rules, err := store.LoadRules("config/rules.json")
	if err != nil {
		log.Fatal(err)
	}

	discountEngine := engine.NewEngine(rules)

	// route
	http.HandleFunc("/apply-discount", func(w http.ResponseWriter, r *http.Request) {
		var req engine.OrderInput

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.CustomerType == "" || req.OrderTotal == 0 {
			log.Println(req)
			http.Error(w, "order details missing", http.StatusBadRequest)
			return
		}

		applied, finalTotal := discountEngine.ApplyBestDiscount(req)

		resp := engine.Response{
			AppliedRule: applied.RuleId,
			Discount:    applied.DiscountAmount,
			FinalTotal:  finalTotal,
		}

		json.NewEncoder(w).Encode(resp)

	})

	// Server
	log.Println("Server running on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))

}
