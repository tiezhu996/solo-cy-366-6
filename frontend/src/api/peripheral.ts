import { get, post, put } from '@/utils/request'

export interface Peripheral {
  id: number
  device_no: string
  device_type: string
  name: string
  status: string
}

export function listPeripherals(params: { page: number; page_size: number; device_type?: string; status?: string }) {
  return get<{ list: Peripheral[]; total: number }>('/peripherals', params)
}

export function createPeripheral(data: { device_no: string; device_type: string; name: string }) {
  return post<Peripheral>('/peripherals', data)
}

export function updatePeripheralStatus(id: number, status: string) {
  return put<Peripheral>(`/peripherals/${id}/status`, { status })
}
