package main

import (
	"log"
	"net/http"

	"inventory-core/handlers"
	"inventory-core/models"
	"inventory-core/services"
)

func main() {
	products := []models.Product{
		{ID: "p-1", Name: "Laptop"},
		{ID: "p-2", Name: "Keyboard"},
		{ID: "p-3", Name: "Mouse"},
	}
	stock := []models.Stock{
		{ProductID: "p-1", AvailableQty: 10},
		{ProductID: "p-2", AvailableQty: 25},
		{ProductID: "p-3", AvailableQty: 50},
	}

	inventoryService := services.NewInventoryService(products, stock)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)

	mux := http.NewServeMux()
	inventoryHandler.RegisterRoutes(mux)

	addr := ":8081"
	log.Printf("inventory-core listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
