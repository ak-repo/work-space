package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"order-core/models"
)

var (
	ErrInventoryUnavailable = errors.New("inventory service unavailable")
	ErrStockUnavailable     = errors.New("stock unavailable")
)

type InventoryClient struct {
	baseURL    string
	httpClient *http.Client
}

type checkRequest struct {
	Items []models.Item `json:"items"`
}

type checkResponse struct {
	AllAvailable bool `json:"all_available"`
}

type reserveRequest struct {
	OrderID string        `json:"order_id"`
	Items   []models.Item `json:"items"`
}

type reserveResponse struct {
	ReservationID string `json:"reservation_id"`
}

type reservationActionRequest struct {
	ReservationID string `json:"reservation_id"`
}

func NewInventoryClient(baseURL string) *InventoryClient {
	return &InventoryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (c *InventoryClient) Check(ctx context.Context, items []models.Item) error {
	var resp checkResponse
	status, err := c.doJSON(ctx, http.MethodPost, "/inventory/check", checkRequest{Items: items}, &resp)
	if err != nil {
		return err
	}
	if status >= 400 {
		if status == http.StatusConflict || status == http.StatusBadRequest {
			return ErrStockUnavailable
		}
		return ErrInventoryUnavailable
	}
	if !resp.AllAvailable {
		return ErrStockUnavailable
	}
	return nil
}

func (c *InventoryClient) Reserve(ctx context.Context, orderID string, items []models.Item) (string, error) {
	var resp reserveResponse
	status, err := c.doJSON(ctx, http.MethodPost, "/inventory/reserve", reserveRequest{OrderID: orderID, Items: items}, &resp)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		if status == http.StatusConflict {
			return "", ErrStockUnavailable
		}
		return "", ErrInventoryUnavailable
	}
	if resp.ReservationID == "" {
		return "", ErrInventoryUnavailable
	}
	return resp.ReservationID, nil
}

func (c *InventoryClient) Confirm(ctx context.Context, reservationID string) error {
	status, err := c.doJSON(ctx, http.MethodPost, "/inventory/confirm", reservationActionRequest{ReservationID: reservationID}, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return ErrInventoryUnavailable
	}
	return nil
}

func (c *InventoryClient) Release(ctx context.Context, reservationID string) error {
	status, err := c.doJSON(ctx, http.MethodPost, "/inventory/release", reservationActionRequest{ReservationID: reservationID}, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return ErrInventoryUnavailable
	}
	return nil
}

func (c *InventoryClient) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) (int, error) {
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, ErrInventoryUnavailable
	}
	defer resp.Body.Close()

	if responseBody != nil {
		_ = json.NewDecoder(resp.Body).Decode(responseBody)
	}

	return resp.StatusCode, nil
}
