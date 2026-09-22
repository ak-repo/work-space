import type { PageKey } from '../../types/ui'

interface SidebarProps {
  activePage: PageKey
  pendingOrders: number
  onPageChange: (page: PageKey) => void
}

export function Sidebar({ activePage, pendingOrders, onPageChange }: SidebarProps) {
  return (
    <aside className="sidebar" role="navigation" aria-label="Main navigation">
      <div className="sidebar-logo">
        <i className="ti ti-truck-delivery" aria-hidden="true" />
      </div>
      <NavItem active={activePage === 'orders'} icon="ti-package" label="Orders" onClick={() => onPageChange('orders')} badge={pendingOrders} />
      <NavItem active={activePage === 'drivers'} icon="ti-users" label="Drivers" onClick={() => onPageChange('drivers')} />
      <div className="nav-divider" />
      <NavItem active={activePage === 'batch'} icon="ti-stack-2" label="Batch" onClick={() => onPageChange('batch')} />
    </aside>
  )
}

function NavItem({ active, icon, label, onClick, badge }: { active: boolean; icon: string; label: string; onClick: () => void; badge?: number }) {
  return (
    <button className={`nav-item ${active ? 'active' : ''}`} type="button" onClick={onClick} title={label}>
      <i className={`ti ${icon}`} aria-hidden="true" />
      <span>{label}</span>
      {badge ? <span className="nav-badge">{badge}</span> : null}
    </button>
  )
}
