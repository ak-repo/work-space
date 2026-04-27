package main

import (
	"fmt"
	"log"
	"net/http"
	"order-delivery/internal"
)

func main() {
	// Wire up the dependency chain: store → service → handler
	s := internal.NewStore()
	svc := internal.NewService(s)
	h := internal.NewHandler(svc)

	mux := http.NewServeMux()
	internal.RegisterRoutes(mux, h)

	addr := ":8080"
	fmt.Println("Order & Delivery service running on http://localhost" + addr)
	fmt.Println()
	fmt.Println("  POST   /orders")
	fmt.Println("  GET    /orders/{id}")
	fmt.Println("  PUT    /orders/{id}/status")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
