import type { Driver, Order, OrderStatus, DashboardStats } from '../../types/api'
import { OrdersTable } from './OrdersTable'

const filters: Array<{ label: string; value: OrderStatus | '' }> = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'Assigned', value: 'assigned' },
  { label: 'Picked up', value: 'picked_up' },
  { label: 'Delivered', value: 'delivered' },
  { label: 'Cancelled', value: 'cancelled' },
]

interface OrdersPageProps {
  orders: Order[]
  drivers: Driver[]
  stats: DashboardStats
  statusFilter: OrderStatus | ''
  onStatusFilter: (status: OrderStatus | '') => void
  onNewOrder: () => void
  onShowOrder: (order: Order) => void
  onUpdateStatus: (id: string, status: OrderStatus) => Promise<void>
}

export function OrdersPage({ orders, drivers, stats, statusFilter, onStatusFilter, onNewOrder, onShowOrder, onUpdateStatus }: OrdersPageProps) {
  return (
    <section className="page active" aria-label="Orders">
      <div className="kpi-row">
        <Kpi color="blue" label="Total orders" value={stats.totalOrders} sub="all time" />
        <Kpi color="amber" label="Pending" value={stats.pendingOrders} sub="awaiting assignment" />
        <Kpi color="green" label="Delivered" value={stats.deliveredOrders} sub="completed" />
        <Kpi color="purple" label="In transit" value={stats.inTransitOrders} sub="assigned + picked up" />
      </div>

      <div className="page-header">
        <div>
          <div className="page-title">Orders</div>
          <div className="page-sub">Manage delivery orders</div>
        </div>
        <button className="btn btn-primary" type="button" onClick={onNewOrder}>
          <i className="ti ti-plus" aria-hidden="true" /> New order
        </button>
      </div>

      <div className="filter-bar">
        <span className="filter-label">Status</span>
        <div className="status-pills">
          {filters.map((filter) => (
            <button className={`s-pill ${statusFilter === filter.value ? 'active' : ''}`} type="button" key={filter.label} onClick={() => onStatusFilter(filter.value)}>
              {filter.label}
            </button>
          ))}
        </div>
      </div>

      <OrdersTable orders={orders} drivers={drivers} onShowOrder={onShowOrder} onUpdateStatus={onUpdateStatus} />
    </section>
  )
}

function Kpi({ color, label, value, sub }: { color: string; label: string; value: number; sub: string }) {
  return (
    <div className={`kpi ${color}`}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value">{value}</div>
      <div className="kpi-sub">{sub}</div>
    </div>
  )
}
