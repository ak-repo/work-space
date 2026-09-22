import { useState } from 'react'
import type { Driver, Order } from '../../types/api'
import { StatusBadge } from '../shared/Badge'
import { EmptyState } from '../shared/EmptyState'

interface DriversTableProps {
  drivers: Driver[]
  orders: Order[]
  onShowDriver: (driver: Driver) => void
  onUpdateLocation: (id: string, lat: number, lng: number) => Promise<void>
}

export function DriversTable({ drivers, onShowDriver, onUpdateLocation }: DriversTableProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [lat, setLat] = useState('')
  const [lng, setLng] = useState('')
  const [savingId, setSavingId] = useState<string | null>(null)

  if (drivers.length === 0) {
    return (
      <div className="table-wrap">
        <EmptyState icon="ti-users" message="No drivers found" />
      </div>
    )
  }

  const startEdit = (driver: Driver) => {
    setEditingId(driver.id)
    setLat(String(driver.current_lat))
    setLng(String(driver.current_lng))
  }

  const saveEdit = async (id: string) => {
    setSavingId(id)
    try {
      await onUpdateLocation(id, Number(lat), Number(lng))
      setEditingId(null)
    } finally {
      setSavingId(null)
    }
  }

  return (
    <div className="table-wrap scroll-table">
      <table>
        <colgroup>
          <col className="col-id" />
          <col className="col-name" />
          <col className="col-phone" />
          <col className="col-status" />
          <col className="col-location" />
          <col className="col-small" />
          <col className="col-small" />
          <col className="col-actions" />
        </colgroup>
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Phone</th>
            <th>Status</th>
            <th>Location</th>
            <th>Cap.</th>
            <th>Active</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {drivers.map((driver) => (
            <tr className="fade-in" key={driver.id}>
              <td><span className="mono-id">{shortId(driver.id)}</span></td>
              <td>{driver.name}</td>
              <td className="muted-cell">{driver.phone}</td>
              <td><StatusBadge status={driver.status} /></td>
              <td>
                {editingId === driver.id ? (
                  <div className="loc-edit">
                    <input type="number" step="any" value={lat} onChange={(event) => setLat(event.target.value)} />
                    <input type="number" step="any" value={lng} onChange={(event) => setLng(event.target.value)} />
                    <button className="btn btn-sm btn-success" type="button" disabled={savingId === driver.id} onClick={() => void saveEdit(driver.id)}>
                      <i className="ti ti-check" aria-hidden="true" />
                    </button>
                    <button className="btn btn-sm btn-ghost" type="button" onClick={() => setEditingId(null)}>
                      <i className="ti ti-x" aria-hidden="true" />
                    </button>
                  </div>
                ) : (
                  <div className="loc-edit">
                    <span className="coords">{driver.current_lat.toFixed(4)}, {driver.current_lng.toFixed(4)}</span>
                    <button className="icon-btn blue" type="button" title="Edit location" onClick={() => startEdit(driver)}>
                      <i className="ti ti-pencil" aria-hidden="true" />
                    </button>
                  </div>
                )}
              </td>
              <td className="center-mono">{driver.capacity}</td>
              <td className={`center-mono ${driver.active_orders > 0 ? 'active-count' : 'muted-cell'}`}>{driver.active_orders}</td>
              <td>
                <button className="icon-btn" type="button" title="View orders" onClick={() => onShowDriver(driver)}>
                  <i className="ti ti-list" aria-hidden="true" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function shortId(id: string) {
  return `${id.slice(0, 8)}...`
}
