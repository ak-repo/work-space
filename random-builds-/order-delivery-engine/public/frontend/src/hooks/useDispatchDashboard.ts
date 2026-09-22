import { useCallback, useEffect, useRef, useState } from 'react'
import { createDriver, getDrivers, updateDriverLocation } from '../services/drivers'
import { batchAssignOrders, createOrder, getOrders, updateOrderStatus } from '../services/orders'
import { connectEvents } from '../services/events'
import type {
  AssignmentResult,
  BatchAssignmentResult,
  CreateDriverRequest,
  CreateOrderRequest,
  DashboardStats,
  Driver,
  Order,
  OrderStatus,
  ToastKind,
  ToastItem,
} from '../types/api'

const refetchDelayMs = 100

export function useDispatchDashboard() {
  const [orders, setOrders] = useState<Order[]>([])
  const [drivers, setDrivers] = useState<Driver[]>([])
  const [statusFilter, setStatusFilter] = useState<OrderStatus | ''>('')
  const [loading, setLoading] = useState(true)
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const [live, setLive] = useState(false)
  const filterRef = useRef<OrderStatus | ''>('')
  const refreshTimer = useRef<number | undefined>(undefined)
  const reconnectTimer = useRef<number | undefined>(undefined)

  const showToast = useCallback((message: string, kind: ToastKind = 'info') => {
    const id = crypto.randomUUID()
    setToasts((current) => [...current, { id, message, kind }])
    window.setTimeout(() => {
      setToasts((current) => current.filter((toast) => toast.id !== id))
    }, 3500)
  }, [])

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const [nextOrders, nextDrivers] = await Promise.all([getOrders(filterRef.current), getDrivers()])
      setOrders(nextOrders)
      setDrivers(nextDrivers)
    } catch (error) {
      showToast(error instanceof Error ? error.message : 'Failed to load dashboard data', 'danger')
    } finally {
      setLoading(false)
    }
  }, [showToast])

  const scheduleRefresh = useCallback(() => {
    window.clearTimeout(refreshTimer.current)
    refreshTimer.current = window.setTimeout(() => {
      void refresh()
    }, refetchDelayMs)
  }, [refresh])

  const applyStatusFilter = useCallback(
    (status: OrderStatus | '') => {
      filterRef.current = status
      setStatusFilter(status)
      void refresh()
    },
    [refresh],
  )

  const addOrder = useCallback(
    async (body: CreateOrderRequest): Promise<AssignmentResult> => {
      const result = await createOrder(body)
      const assigned = Boolean(result.driver?.id)
      showToast(assigned ? 'Order created and assigned' : 'Order created; no driver assigned yet', assigned ? 'success' : 'warning')
      await refresh()
      return result
    },
    [refresh, showToast],
  )

  const addDriver = useCallback(
    async (body: CreateDriverRequest) => {
      const driver = await createDriver(body)
      showToast('Driver added', 'success')
      await refresh()
      return driver
    },
    [refresh, showToast],
  )

  const setOrderStatus = useCallback(
    async (id: string, status: OrderStatus) => {
      await updateOrderStatus(id, status)
      showToast(`Order -> ${status.replace('_', ' ')}`, 'success')
      await refresh()
    },
    [refresh, showToast],
  )

  const setDriverLocation = useCallback(
    async (id: string, lat: number, lng: number) => {
      await updateDriverLocation(id, lat, lng)
      showToast('Location updated', 'success')
      await refresh()
    },
    [refresh, showToast],
  )

  const assignBatch = useCallback(
    async (driverId: string, orderIds: string[]): Promise<BatchAssignmentResult | undefined> => {
      if (!driverId || orderIds.length === 0) {
        showToast('Select at least one order and a driver', 'warning')
        return undefined
      }
      const result = await batchAssignOrders(driverId, orderIds)
      if (!result) {
        throw new Error('Batch assignment returned no result')
      }
      showToast(`${result.assigned} assigned, ${result.skipped} skipped, ${result.failed} failed`, result.failed ? 'warning' : 'success')
      await refresh()
      return result
    },
    [refresh, showToast],
  )

  useEffect(() => {
    const id = window.setTimeout(() => {
      void refresh()
    }, 0)
    return () => window.clearTimeout(id)
  }, [refresh])

  useEffect(() => {
    let cleanup: (() => void) | undefined
    let active = true

    const connect = () => {
      cleanup = connectEvents(
        (update) => {
          setLive(true)
          if (update.message) {
            showToast(update.message, 'info')
          }
          if (update.type === 'order_update' || update.type === 'driver_update') {
            scheduleRefresh()
          }
        },
        () => {
          setLive(false)
          if (active) {
            reconnectTimer.current = window.setTimeout(connect, 3000)
          }
        },
      )
    }

    connect()

    return () => {
      active = false
      cleanup?.()
      window.clearTimeout(refreshTimer.current)
      window.clearTimeout(reconnectTimer.current)
    }
  }, [scheduleRefresh, showToast])

  const stats: DashboardStats = {
    availableDrivers: drivers.filter((driver) => driver.status === 'available').length,
    busyDrivers: drivers.filter((driver) => driver.status === 'busy').length,
    pendingOrders: orders.filter((order) => order.status === 'pending').length,
    totalOrders: orders.length,
    deliveredOrders: orders.filter((order) => order.status === 'delivered').length,
    inTransitOrders: orders.filter((order) => order.status === 'assigned' || order.status === 'picked_up').length,
  }

  return {
    orders,
    drivers,
    statusFilter,
    stats,
    loading,
    live,
    toasts,
    refresh,
    showToast,
    setStatusFilter: applyStatusFilter,
    addOrder,
    addDriver,
    setOrderStatus,
    setDriverLocation,
    assignBatch,
  }
}
