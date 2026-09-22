import { apiFetch, apiFetchResponse } from '../lib/api'
import type { CreateDriverRequest, Driver } from '../types/api'

export function getDrivers() {
  return apiFetch<Driver[]>('/api/drivers')
}

export function createDriver(body: CreateDriverRequest) {
  return apiFetch<Driver>('/api/drivers', { method: 'POST', body })
}

export async function updateDriverLocation(id: string, lat: number, lng: number) {
  await apiFetchResponse<never>(`/api/drivers/${id}/location`, { method: 'PUT', body: { lat, lng } })
}
