import { useState, type FormEvent } from 'react'
import type { CreateDriverRequest } from '../../types/api'
import { Modal } from '../shared/Modal'

const initialForm: CreateDriverRequest = {
  name: '',
  phone: '',
  current_lat: 0,
  current_lng: 0,
  capacity: 3,
}

interface DriverFormModalProps {
  open: boolean
  onClose: () => void
  onSubmit: (body: CreateDriverRequest) => Promise<void>
}

export function DriverFormModal({ open, onClose, onSubmit }: DriverFormModalProps) {
  const [form, setForm] = useState<CreateDriverRequest>(initialForm)
  const [saving, setSaving] = useState(false)

  const update = (key: keyof CreateDriverRequest, value: string) => {
    setForm((current) => ({ ...current, [key]: ['current_lat', 'current_lng', 'capacity'].includes(key) ? Number(value) : value }))
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    try {
      await onSubmit(form)
      setForm(initialForm)
      onClose()
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={<><i className="ti ti-user-plus green-text" aria-hidden="true" /> New driver</>}>
      <form onSubmit={(event) => void submit(event)}>
        <div className="form-grid">
          <Field label="Name *" value={form.name} onChange={(value) => update('name', value)} required />
          <Field label="Phone *" value={form.phone} onChange={(value) => update('phone', value)} required />
          <Field label="Latitude *" type="number" value={form.current_lat} onChange={(value) => update('current_lat', value)} required />
          <Field label="Longitude *" type="number" value={form.current_lng} onChange={(value) => update('current_lng', value)} required />
          <Field label="Capacity" type="number" value={form.capacity} onChange={(value) => update('capacity', value)} min={1} max={10} />
        </div>
        <div className="form-actions">
          <button className="btn btn-ghost" type="button" onClick={onClose}>Cancel</button>
          <button className="btn btn-primary" type="submit" disabled={saving}>Create driver</button>
        </div>
      </form>
    </Modal>
  )
}

function Field({ label, value, onChange, type = 'text', required, min, max }: { label: string; value: string | number; onChange: (value: string) => void; type?: string; required?: boolean; min?: number; max?: number }) {
  return (
    <div className="form-group">
      <label>{label}</label>
      <input type={type} step={type === 'number' ? 'any' : undefined} min={min} max={max} value={value} onChange={(event) => onChange(event.target.value)} required={required} />
    </div>
  )
}
