import type { Driver, Order } from '../../types/api'
import { DriversTable } from './DriversTable'

interface DriversPageProps {
  drivers: Driver[]
  orders: Order[]
  onNewDriver: () => void
  onShowDriver: (driver: Driver) => void
  onUpdateLocation: (id: string, lat: number, lng: number) => Promise<void>
}

export function DriversPage({ drivers, orders, onNewDriver, onShowDriver, onUpdateLocation }: DriversPageProps) {
  return (
    <section className="page active" aria-label="Drivers">
      <div className="page-header">
        <div>
          <div className="page-title">Drivers</div>
          <div className="page-sub">Fleet management</div>
        </div>
        <button className="btn btn-primary" type="button" onClick={onNewDriver}>
          <i className="ti ti-plus" aria-hidden="true" /> New driver
        </button>
      </div>
      <DriversTable drivers={drivers} orders={orders} onShowDriver={onShowDriver} onUpdateLocation={onUpdateLocation} />
    </section>
  )
}
