import type { DashboardStats } from '../../types/api'
import type { PageKey } from '../../types/ui'

const titles: Record<PageKey, string> = {
  orders: 'Orders',
  drivers: 'Drivers',
  batch: 'Batch Assign',
}

export function Topbar({ page, stats, live }: { page: PageKey; stats: DashboardStats; live: boolean }) {
  return (
    <header className="topbar">
      <div className="topbar-left">
        <span className={`live-dot ${live ? '' : 'muted'}`} title={live ? 'Live' : 'Connecting'} />
        <span className="topbar-title">{titles[page]}</span>
        <span className="topbar-sep">/</span>
        <span className="topbar-sub">Dispatch Control</span>
      </div>
      <div className="topbar-right">
        <StatPill color="green" icon="ti-user-check" value={stats.availableDrivers} label="" title="Available drivers" />
        <StatPill color="amber" icon="ti-loader" value={stats.busyDrivers} label="" title="Busy drivers" />
        <StatPill color="blue" icon="ti-clock" value={stats.pendingOrders} label="pending" title="Pending orders" />
        <StatPill color="purple" icon="ti-package" value={stats.totalOrders} label="total" title="Total orders" />
      </div>
    </header>
  )
}

function StatPill({ color, icon, value, label, title }: { color: string; icon: string; value: number; label: string; title: string }) {
  return (
    <div className={`stat-pill ${color}`} title={title}>
      <i className={`ti ${icon}`} aria-hidden="true" />
      <span>{value}</span>
      {label}
    </div>
  )
}
