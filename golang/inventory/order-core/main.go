package main

import (
	"log"
	"net/http"
	"os"

	"order-core/handlers"
	"order-core/services"
)

func main() {
	inventoryBaseURL := os.Getenv("INVENTORY_BASE_URL")
	if inventoryBaseURL == "" {
		inventoryBaseURL = "http://localhost:8081"
	}

	inventoryClient := services.NewInventoryClient(inventoryBaseURL)
	orderService := services.NewOrderService(inventoryClient)
	orderHandler := handlers.NewOrderHandler(orderService)

	mux := http.NewServeMux()
	orderHandler.RegisterRoutes(mux)

	addr := ":8082"
	log.Printf("order-core listening on %s (inventory: %s)", addr, inventoryBaseURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
