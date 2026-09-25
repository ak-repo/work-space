package service

import (
	"container/heap"

	"github.com/ak-repo/order-delivery-engine/internal/models"
)

// DriverPriorityQueue is a max-heap: highest score = best driver
type DriverPriorityQueue []*models.DriverHeapItem

func (pq DriverPriorityQueue) Len() int { return len(pq) }

// We want the HIGHEST score at the top (max-heap)
func (pq DriverPriorityQueue) Less(i, j int) bool {
	return pq[i].Score > pq[j].Score
}

func (pq DriverPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *DriverPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*models.DriverHeapItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *DriverPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[:n-1]
	return item
}

// BuildDriverQueue scores all available drivers for an order pickup location
// and returns a max-heap ready to pop the best driver.
func BuildDriverQueue(drivers []models.Driver, pickupLat, pickupLng float64, maxRadiusKm, distanceWeight, capacityWeight, smoothing float64) *DriverPriorityQueue {
	pq := &DriverPriorityQueue{}
	heap.Init(pq)

	for _, d := range drivers {
		dist := HaversineKm(d.CurrentLat, d.CurrentLng, pickupLat, pickupLng)

		// Score formula:
		//   - Closer driver = higher score (inverse distance, capped at maxRadiusKm)
		//   - Fewer active orders = higher score
		//   - max maxRadiusKm radius considered
		if dist > maxRadiusKm {
			continue // skip drivers more than maxRadiusKm away
		}

		distScore := 1.0 / (dist + smoothing) // avoid division by zero
		capacityScore := float64(d.Capacity-d.ActiveOrders) / float64(d.Capacity)
		totalScore := (distScore * distanceWeight) + (capacityScore * capacityWeight)

		dCopy := d
		dCopy.Distance = dist
		dCopy.Score = totalScore

		heap.Push(pq, &models.DriverHeapItem{
			Driver: dCopy,
			Score:  totalScore,
		})
	}
	return pq
}
