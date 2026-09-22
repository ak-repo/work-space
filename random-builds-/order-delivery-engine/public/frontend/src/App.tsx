import { useState } from 'react'
import { BatchAssignPage } from './components/batch/BatchAssignPage'
import { DriverFormModal } from './components/drivers/DriverFormModal'
import { DriversPage } from './components/drivers/DriversPage'
import { Sidebar } from './components/layout/Sidebar'
import { Topbar } from './components/layout/Topbar'
import { OrderFormModal } from './components/orders/OrderFormModal'
import { OrdersPage } from './components/orders/OrdersPage'
import { DetailModal, type DetailTarget } from './components/shared/DetailModal'
import { ToastViewport } from './components/shared/ToastViewport'
import { useDispatchDashboard } from './hooks/useDispatchDashboard'
import type { CreateDriverRequest, CreateOrderRequest } from './types/api'
import type { PageKey } from './types/ui'

function App() {
  const dashboard = useDispatchDashboard()
  const [activePage, setActivePage] = useState<PageKey>('orders')
  const [orderModalOpen, setOrderModalOpen] = useState(false)
  const [driverModalOpen, setDriverModalOpen] = useState(false)
  const [detailTarget, setDetailTarget] = useState<DetailTarget | null>(null)

  const createOrder = async (body: CreateOrderRequest) => {
    await dashboard.addOrder(body)
  }

  const createDriver = async (body: CreateDriverRequest) => {
    await dashboard.addDriver(body)
  }

  return (
    <>
      <div className="shell">
        <Sidebar activePage={activePage} pendingOrders={dashboard.stats.pendingOrders} onPageChange={setActivePage} />
        <div className="main-wrap">
          <Topbar page={activePage} stats={dashboard.stats} live={dashboard.live} />
          <main className="content" aria-busy={dashboard.loading}>
            {activePage === 'orders' ? (
              <OrdersPage
                orders={dashboard.orders}
                drivers={dashboard.drivers}
                stats={dashboard.stats}
                statusFilter={dashboard.statusFilter}
                onStatusFilter={dashboard.setStatusFilter}
                onNewOrder={() => setOrderModalOpen(true)}
                onShowOrder={(order) => setDetailTarget({ type: 'order', order })}
                onUpdateStatus={dashboard.setOrderStatus}
              />
            ) : null}
            {activePage === 'drivers' ? (
              <DriversPage
                drivers={dashboard.drivers}
                orders={dashboard.orders}
                onNewDriver={() => setDriverModalOpen(true)}
                onShowDriver={(driver) => setDetailTarget({ type: 'driver', driver })}
                onUpdateLocation={dashboard.setDriverLocation}
              />
            ) : null}
            {activePage === 'batch' ? (
              <BatchAssignPage orders={dashboard.orders} drivers={dashboard.drivers} onAssign={dashboard.assignBatch} onRefresh={dashboard.refresh} />
            ) : null}
          </main>
        </div>
      </div>

      <OrderFormModal open={orderModalOpen} onClose={() => setOrderModalOpen(false)} onSubmit={createOrder} />
      <DriverFormModal open={driverModalOpen} onClose={() => setDriverModalOpen(false)} onSubmit={createDriver} />
      <DetailModal target={detailTarget} drivers={dashboard.drivers} orders={dashboard.orders} onClose={() => setDetailTarget(null)} />
      <ToastViewport toasts={dashboard.toasts} />
    </>
  )
}

export default App
