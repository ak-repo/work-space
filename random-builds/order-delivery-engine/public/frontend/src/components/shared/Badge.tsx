import type { DriverStatus, OrderStatus } from '../../types/api'

export function StatusBadge({ status }: { status: OrderStatus | DriverStatus }) {
  return <span className={`badge badge-${status}`}>{status.replace('_', ' ')}</span>
}

export function PriorityBadge({ priority }: { priority: number }) {
  return <span className={`badge priority-${priority}`}>P{priority}</span>
}
