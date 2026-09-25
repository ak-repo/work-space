import { EmptyState } from '../shared/EmptyState'
import { PriorityBadge, StatusBadge } from '../shared/Badge'
import type { Driver, Order, OrderStatus } from '../../types/api'

const orderFlow: OrderStatus[] = ['pending', 'assigned', 'picked_up', 'delivered']

interface OrdersTableProps {
  orders: Order[]
  drivers: Driver[]
  onShowOrder: (order: Order) => void
  onUpdateStatus: (id: string, status: OrderStatus) => Promise<void>
}

export function OrdersTable({ orders, drivers, onShowOrder, onUpdateStatus }: OrdersTableProps) {
  if (orders.length === 0) {
    return (
      <div className="table-wrap">
        <EmptyState icon="ti-inbox" message="No orders found" />
      </div>
    )
  }

  return (
    <div className="table-wrap scroll-table">
      <table>
        <colgroup>
          <col className="col-id" />
          <col className="col-name" />
          <col className="col-address" />
          <col className="col-address" />
          <col className="col-priority" />
          <col className="col-status" />
          <col className="col-driver" />
          <col className="col-actions" />
        </colgroup>
        <thead>
          <tr>
            <th>ID</th>
            <th>Customer</th>
            <th>Pickup</th>
            <th>Delivery</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Driver</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {orders.map((order) => {
            const driverName = order.driver_id ? drivers.find((driver) => driver.id === order.driver_id)?.name || 'Unknown' : 'Unassigned'
            const nextStatus = getNextStatus(order.status)
            return (
              <tr className="fade-in" key={order.id}>
                <td><span className="mono-id">{shortId(order.id)}</span></td>
                <td>{order.customer_name}</td>
                <td className="muted-cell">{order.pickup_address}</td>
                <td className="muted-cell">{order.delivery_address}</td>
                <td><PriorityBadge priority={order.priority} /></td>
                <td><StatusBadge status={order.status} /></td>
                <td className={order.driver_id ? '' : 'muted-cell'}>{driverName}</td>
                <td>
                  <div className="action-btns">
                    {nextStatus ? (
                      <button className="icon-btn blue" type="button" title={`Advance to ${nextStatus}`} onClick={() => void onUpdateStatus(order.id, nextStatus)}>
                        <i className="ti ti-circle-arrow-right" aria-hidden="true" />
                      </button>
                    ) : null}
                    <button className="icon-btn" type="button" title="View details" onClick={() => onShowOrder(order)}>
                      <i className="ti ti-eye" aria-hidden="true" />
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function getNextStatus(status: OrderStatus) {
  const index = orderFlow.indexOf(status)
  return index >= 0 && index < orderFlow.length - 1 ? orderFlow[index + 1] : undefined
}

function shortId(id: string) {
  return `${id.slice(0, 8)}...`
}
