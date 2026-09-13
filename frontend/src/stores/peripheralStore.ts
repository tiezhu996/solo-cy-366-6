import { defineStore } from 'pinia'
import { listPeripherals, type Peripheral } from '@/api/peripheral'
import { listRentals, listMyRentals, type Rental } from '@/api/rental'

export const usePeripheralStore = defineStore('peripheral', () => {
  async function fetchAvailable() {
    return listPeripherals({ page: 1, page_size: 100, status: 'available' })
  }
  async function fetchRentals(params: { page: number; page_size: number; status?: string }) {
    return listRentals(params)
  }
  async function fetchMine(params: { page: number; page_size: number; status?: string }) {
    return listMyRentals(params)
  }
  function rentingByUser(list: Rental[], userId: number): Rental[] {
    return list.filter((r) => r.user_id === userId && r.status === 'renting')
  }
  function deviceLabel(p: Peripheral): string {
    return `${p.device_no} · ${p.name}`
  }
  return { fetchAvailable, fetchRentals, fetchMine, rentingByUser, deviceLabel }
})
