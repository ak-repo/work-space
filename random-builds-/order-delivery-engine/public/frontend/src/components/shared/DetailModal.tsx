import type { Driver, Order } from '../../types/api'
import type { ReactNode } from 'react'
import { PriorityBadge, StatusBadge } from './Badge'
import { Modal } from './Modal'

export type DetailTarget = { type: 'order'; order: Order } | { type: 'driver'; driver: Driver }

interface DetailModalProps {
  target: DetailTarget | null
  drivers: Driver[]
  orders: Order[]
  onClose: () => void
}

export function DetailModal({ target, drivers, orders, onClose }: DetailModalProps) {
  if (!target) {
    return null
  }

  if (target.type === 'order') {
    const order = target.order
    const driverName = order.driver_id ? drivers.find((driver) => driver.id === order.driver_id)?.name || 'Unknown' : 'Unassigned'
    return (
      <Modal open onClose={onClose} title={<><i className="ti ti-package blue-text" aria-hidden="true" /> Order <span className="mono-id">{order.id.slice(0, 12)}...</span></>}>
        <div className="details-grid">
          <Detail label="Customer" value={order.customer_name} />
          <Detail label="Phone" value={order.customer_phone || '-'} />
          <Detail label="Status"><StatusBadge status={order.status} /></Detail>
          <Detail label="Priority"><PriorityBadge priority={order.priority} /></Detail>
          <Detail label="Pickup address" value={order.pickup_address} wide />
          <Detail label="Pickup coords" value={`${order.pickup_lat}, ${order.pickup_lng}`} mono />
          <Detail label="Delivery address" value={order.delivery_address} wide />
          <Detail label="Delivery coords" value={`${order.delivery_lat}, ${order.delivery_lng}`} mono />
          <Detail label="Assigned driver" value={driverName} />
          <Detail label="Notes" value={order.notes || '-'} wide />
        </div>
        <div className="form-actions"><button className="btn btn-ghost" type="button" onClick={onClose}>Close</button></div>
      </Modal>
    )
  }

  const driver = target.driver
  const activeOrders = orders.filter((order) => order.driver_id === driver.id && order.status !== 'delivered' && order.status !== 'cancelled')

  return (
    <Modal open onClose={onClose} title={<><i className="ti ti-user green-text" aria-hidden="true" /> {driver.name}</>}>
      <div className="details-grid detail-section">
        <Detail label="Phone" value={driver.phone} />
        <Detail label="Status"><StatusBadge status={driver.status} /></Detail>
        <Detail label="Location" value={`${driver.current_lat.toFixed(4)}, ${driver.current_lng.toFixed(4)}`} mono />
        <Detail label="Capacity" value={`${driver.active_orders} / ${driver.capacity}`} />
      </div>
      {activeOrders.length ? (
        <>
          <span className="section-tag">Active orders</span>
          {activeOrders.map((order) => (
            <div className="detail-order" key={order.id}>
              <span className="mono-id">{order.id.slice(0, 10)}...</span>
              <span>{order.customer_name}</span>
              <StatusBadge status={order.status} />
            </div>
          ))}
        </>
      ) : (
        <div className="empty-detail">No active orders</div>
      )}
      <div className="form-actions"><button className="btn btn-ghost" type="button" onClick={onClose}>Close</button></div>
    </Modal>
  )
}

function Detail({ label, value, children, wide, mono }: { label: string; value?: string; children?: ReactNode; wide?: boolean; mono?: boolean }) {
  return (
    <div className={`detail-row ${wide ? 'detail-wide' : ''}`}>
      <span className="detail-lbl">{label}</span>
      <span className={`detail-val ${mono ? 'mono-value' : ''}`}>{children ?? value}</span>
    </div>
  )
}
