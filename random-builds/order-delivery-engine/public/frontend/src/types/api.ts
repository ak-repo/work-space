export type OrderStatus = 'pending' | 'assigned' | 'picked_up' | 'delivered' | 'cancelled'

export type DriverStatus = 'available' | 'busy' | 'unavailable'

export type ToastKind = 'success' | 'danger' | 'warning' | 'info'

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
  message?: string
}

export interface Order {
  id: string
  customer_name: string
  customer_phone: string
  pickup_lat: number
  pickup_lng: number
  pickup_address: string
  delivery_lat: number
  delivery_lng: number
  delivery_address: string
  status: OrderStatus
  driver_id?: string | null
  priority: 1 | 2 | 3
  notes: string
  created_at: string
  updated_at: string
  assigned_at?: string | null
  delivered_at?: string | null
}

export interface Driver {
  id: string
  name: string
  phone: string
  status: DriverStatus
  current_lat: number
  current_lng: number
  capacity: number
  active_orders: number
  created_at: string
  updated_at: string
  score?: number
  distance_km?: number
}

export interface CreateOrderRequest {
  customer_name: string
  customer_phone: string
  pickup_lat: number
  pickup_lng: number
  pickup_address: string
  delivery_lat: number
  delivery_lng: number
  delivery_address: string
  priority: number
  notes: string
}

export interface CreateDriverRequest {
  name: string
  phone: string
  current_lat: number
  current_lng: number
  capacity: number
}

export interface AssignmentResult {
  order: Order
  driver?: Partial<Driver>
  distance_km?: number
  score?: number
}

export interface BatchAssignmentItem {
  order_id: string
  status: 'assigned' | 'skipped' | 'failed'
  reason?: string
}

export interface BatchAssignmentResult {
  driver_id: string
  results: BatchAssignmentItem[]
  assigned: number
  skipped: number
  failed: number
}

export interface StatusUpdate {
  type: 'order_update' | 'driver_update' | string
  order_id?: string
  driver_id?: string
  status: string
  message: string
  data?: unknown
}

export interface DashboardStats {
  availableDrivers: number
  busyDrivers: number
  pendingOrders: number
  totalOrders: number
  deliveredOrders: number
  inTransitOrders: number
}

export interface ToastItem {
  id: string
  message: string
  kind: ToastKind
}
