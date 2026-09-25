import { apiFetch, apiFetchResponse } from '../lib/api'
import type {
  AssignmentResult,
  BatchAssignmentResult,
  CreateOrderRequest,
  Order,
  OrderStatus,
} from '../types/api'

export function getOrders(status?: OrderStatus | '') {
  const query = status ? `?status=${encodeURIComponent(status)}` : ''
  return apiFetch<Order[]>(`/api/orders${query}`)
}

export function createOrder(body: CreateOrderRequest) {
  return apiFetch<AssignmentResult>('/api/orders', { method: 'POST', body })
}

export function updateOrderStatus(id: string, status: OrderStatus) {
  return apiFetch<Order>(`/api/orders/${id}/status`, { method: 'PATCH', body: { status } })
}

export async function batchAssignOrders(driverId: string, orderIds: string[]) {
  const response = await apiFetchResponse<BatchAssignmentResult>('/api/orders/batch-assign', {
    method: 'POST',
    body: { driver_id: driverId, order_ids: orderIds },
  })
  return response.data
}
