import { get, post } from '@/utils/request'

export interface Rental {
  id: number
  rental_no: string
  user_id: number
  peripheral_id: number
  device_no: string
  device_type: string
  deposit: number
  expected_return_at: string
  returned_at: string | null
  status: string
  damage_desc: string
  compensation: number
  handler_id: number
  handler_name: string
  debt_amount: number
  refund_amount: number
  remark: string
  created_at: string
}

export function listRentals(params: { page: number; page_size: number; status?: string; user_id?: number }) {
  return get<{ list: Rental[]; total: number }>('/rentals', params)
}

export function listMyRentals(params: { page: number; page_size: number; status?: string }) {
  return get<{ list: Rental[]; total: number }>('/rentals/mine', params)
}

export function createRental(data: { user_id: number; peripheral_id: number; deposit: number; expected_return_at: string; remark?: string }) {
  return post<Rental>('/rentals', data)
}

export function returnRental(id: number) {
  return post<Rental>(`/rentals/${id}/return`)
}

export function damageRental(id: number, data: { damage_desc: string; compensation: number }) {
  return post<Rental>(`/rentals/${id}/damage`, data)
}
