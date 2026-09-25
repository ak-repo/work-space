import { useState } from 'react'
import type { Driver, Order } from '../../types/api'

interface BatchAssignPageProps {
  orders: Order[]
  drivers: Driver[]
  onAssign: (driverId: string, orderIds: string[]) => Promise<unknown>
  onRefresh: () => Promise<void>
}

export function BatchAssignPage({ orders, drivers, onAssign, onRefresh }: BatchAssignPageProps) {
  const [selectedOrders, setSelectedOrders] = useState<string[]>([])
  const [driverId, setDriverId] = useState('')
  const [saving, setSaving] = useState(false)
  const pendingOrders = orders.filter((order) => order.status === 'pending')
  const availableDrivers = drivers.filter((driver) => driver.status === 'available' && driver.active_orders < driver.capacity)

  const assign = async () => {
    setSaving(true)
    try {
      await onAssign(driverId, selectedOrders)
      setSelectedOrders([])
      setDriverId('')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="page active" aria-label="Batch assign">
      <div className="page-header">
        <div>
          <div className="page-title">Batch assign</div>
          <div className="page-sub">Assign multiple pending orders to a driver</div>
        </div>
        <button className="btn btn-ghost btn-sm" type="button" onClick={() => void onRefresh()}>
          <i className="ti ti-refresh" aria-hidden="true" /> Refresh
        </button>
      </div>

      <div className="batch-panel">
        <div className="batch-grid">
          <div className="batch-box">
            <label><i className="ti ti-package" aria-hidden="true" /> Pending orders <span>(hold Ctrl/Cmd for multi)</span></label>
            <select multiple value={selectedOrders} onChange={(event) => setSelectedOrders(Array.from(event.target.selectedOptions, (option) => option.value))}>
              {pendingOrders.map((order) => (
                <option key={order.id} value={order.id}>{order.id.slice(0, 8)} · {order.customer_name} - {order.pickup_address}</option>
              ))}
            </select>
          </div>
          <div className="batch-box">
            <label><i className="ti ti-user" aria-hidden="true" /> Available driver</label>
            <select className="ctrl driver-select" value={driverId} onChange={(event) => setDriverId(event.target.value)}>
              <option value="">- select driver -</option>
              {availableDrivers.map((driver) => (
                <option key={driver.id} value={driver.id}>{driver.name} ({driver.id.slice(0, 8)}) · {driver.active_orders}/{driver.capacity} active</option>
              ))}
            </select>
          </div>
        </div>
        <button className="btn btn-success" type="button" disabled={saving} onClick={() => void assign()}>
          <i className="ti ti-bolt" aria-hidden="true" /> Assign selected orders
        </button>
      </div>
    </section>
  )
}
