import { useState, type FormEvent } from 'react'
import type { CreateOrderRequest } from '../../types/api'
import { Modal } from '../shared/Modal'

function emptyOrderForm(): CreateOrderRequest {
  return {
    customer_name: '',
    customer_phone: '',
    pickup_lat: 0,
    pickup_lng: 0,
    pickup_address: '',
    delivery_lat: 0,
    delivery_lng: 0,
    delivery_address: '',
    priority: 2,
    notes: '',
  }
}

interface OrderFormModalProps {
  open: boolean
  onClose: () => void
  onSubmit: (body: CreateOrderRequest) => Promise<void>
}

export function OrderFormModal({ open, onClose, onSubmit }: OrderFormModalProps) {
  const [form, setForm] = useState<CreateOrderRequest>(emptyOrderForm())
  const [saving, setSaving] = useState(false)

  const update = (key: keyof CreateOrderRequest, value: string) => {
    setForm((current) => ({ ...current, [key]: ['pickup_lat', 'pickup_lng', 'delivery_lat', 'delivery_lng', 'priority'].includes(key) ? Number(value) : value }))
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      await onSubmit(form)
      setForm(emptyOrderForm())
      onClose()
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={<><i className="ti ti-package blue-text" aria-hidden="true" /> New order</>}>
      <form onSubmit={(event) => void submit(event)}>
        <div className="form-grid">
          <Field label="Customer name *" value={form.customer_name} onChange={(value) => update('customer_name', value)} required />
          <Field label="Customer phone" value={form.customer_phone} onChange={(value) => update('customer_phone', value)} />
          <Field label="Pickup address *" value={form.pickup_address} onChange={(value) => update('pickup_address', value)} required wide />
          <Field label="Pickup lat *" type="number" value={form.pickup_lat} onChange={(value) => update('pickup_lat', value)} required />
          <Field label="Pickup lng *" type="number" value={form.pickup_lng} onChange={(value) => update('pickup_lng', value)} required />
          <Field label="Delivery address *" value={form.delivery_address} onChange={(value) => update('delivery_address', value)} required wide />
          <Field label="Delivery lat *" type="number" value={form.delivery_lat} onChange={(value) => update('delivery_lat', value)} required />
          <Field label="Delivery lng *" type="number" value={form.delivery_lng} onChange={(value) => update('delivery_lng', value)} required />
          <div className="form-group">
            <label>Priority</label>
            <select value={form.priority} onChange={(event) => update('priority', event.target.value)}>
              <option value="1">Normal (1)</option>
              <option value="2">Express (2)</option>
              <option value="3">Premium (3)</option>
            </select>
          </div>
          <div className="form-group form-wide">
            <label>Notes</label>
            <textarea value={form.notes} onChange={(event) => update('notes', event.target.value)} placeholder="Any special instructions..." />
          </div>
        </div>
        <div className="form-actions">
          <button className="btn btn-ghost" type="button" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" type="submit" disabled={saving}>Create order</button>
        </div>
      </form>
    </Modal>
  )
}

function Field({ label, value, onChange, type = 'text', required, wide }: { label: string; value: string | number; onChange: (value: string) => void; type?: string; required?: boolean; wide?: boolean }) {
  return (
    <div className={`form-group ${wide ? 'form-wide' : ''}`}>
      <label>{label}</label>
      <input type={type} step={type === 'number' ? 'any' : undefined} value={value} onChange={(event) => onChange(event.target.value)} required={required} />
    </div>
  )
}
