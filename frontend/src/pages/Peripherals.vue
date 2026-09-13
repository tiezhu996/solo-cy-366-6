<template>
  <div class="peripherals-page">
    <!-- 会员视图：我的欠款 + 我的在借/历史记录 -->
    <template v-if="!isStaffOrAdmin">
      <van-cell-group inset title="我的账户">
        <van-cell title="账户余额" :value="formatMoney(user?.balance)" />
        <van-cell title="我的欠款">
          <template #value>
            <span :class="{ 'debt-text': (user?.debt ?? 0) > 0 }">{{ formatMoney(user?.debt) }}</span>
          </template>
        </van-cell>
      </van-cell-group>
      <van-dropdown-menu>
        <van-dropdown-item v-model="myStatus" :options="rentalStatusOptions" @change="loadMine" />
      </van-dropdown-menu>
      <van-cell-group inset title="我的租借记录">
        <van-cell
          v-for="r in myList"
          :key="r.id"
          :title="`${typeText(r.device_type)} · ${r.device_no}`"
          :label="mineLabel(r)"
        >
          <template #value>
            <StatusBadge kind="rental" :status="r.status" />
          </template>
        </van-cell>
        <EmptyState v-if="!myList.length" description="暂无租借记录" />
      </van-cell-group>
      <van-pagination v-model="myPage" :total-items="myTotal" :items-per-page="pageSize" @change="loadMine" />
    </template>

    <!-- 店员/管理员视图 -->
    <template v-else>
      <van-tabs v-model:active="tab" sticky>
        <van-tab title="租借管理">
          <van-cell-group inset title="登记租借（押金从会员余额冻结）">
            <van-field v-model="rentForm.user_id" type="number" label="会员ID" placeholder="输入到场会员ID" />
            <van-field :model-value="rentForm.deviceLabel" label="租借设备" placeholder="选择可借设备" readonly @click="openDevicePicker" />
            <van-field v-model="rentForm.deposit" type="number" label="押金(元)" placeholder="如 100" />
            <van-field :model-value="rentForm.expectedLabel" label="预计归还" placeholder="选择归还日期" readonly @click="showReturnDate = true" />
            <van-field v-model="rentForm.remark" label="备注" placeholder="选填" />
          </van-cell-group>
          <div class="submit-btn"><van-button round block type="primary" @click="createRent">登记租借</van-button></div>

          <van-dropdown-menu>
            <van-dropdown-item v-model="rentalStatus" :options="rentalStatusOptions" @change="loadRentals" />
          </van-dropdown-menu>
          <van-cell-group inset title="租借记录">
            <van-cell
              v-for="r in rentalList"
              :key="r.id"
              :title="`#${r.id} ${typeText(r.device_type)} · ${r.device_no} · 会员${r.user_id}`"
              :label="staffLabel(r)"
            >
              <template #value>
                <StatusBadge kind="rental" :status="r.status" />
                <template v-if="r.status === RENTAL_STATUS.RENTING">
                  <van-button size="mini" type="success" class="op-btn" @click="doReturn(r)">归还</van-button>
                  <van-button size="mini" type="danger" plain class="op-btn" @click="openDamage(r)">损坏</van-button>
                </template>
              </template>
            </van-cell>
            <EmptyState v-if="!rentalList.length" description="暂无租借记录" />
          </van-cell-group>
          <van-pagination v-model="rentalPage" :total-items="rentalTotal" :items-per-page="pageSize" @change="loadRentals" />
        </van-tab>

        <van-tab title="设备管理">
          <van-cell-group inset title="登记设备">
            <van-field v-model="devForm.device_no" label="设备编号" placeholder="如 KB-003" />
            <van-field :model-value="devForm.typeLabel" label="设备类型" placeholder="选择类型" readonly @click="showType = true" />
            <van-field v-model="devForm.name" label="设备名称" placeholder="如 机械键盘 青轴" />
          </van-cell-group>
          <div class="submit-btn"><van-button round block type="primary" @click="createDev">登记设备</van-button></div>
          <van-cell-group inset title="设备列表">
            <van-cell v-for="d in deviceList" :key="d.id" :title="`${d.device_no} · ${d.name}`" :label="typeText(d.device_type)">
              <template #value>
                <StatusBadge kind="peripheral" :status="d.status" />
                <van-button v-if="d.status === PERIPHERAL_STATUS.AVAILABLE" size="mini" type="warning" plain class="op-btn" @click="setStatus(d, PERIPHERAL_STATUS.MAINTENANCE)">维护</van-button>
                <van-button v-if="d.status === PERIPHERAL_STATUS.MAINTENANCE" size="mini" type="success" plain class="op-btn" @click="setStatus(d, PERIPHERAL_STATUS.AVAILABLE)">恢复</van-button>
              </template>
            </van-cell>
            <EmptyState v-if="!deviceList.length" description="暂无设备" />
          </van-cell-group>
          <van-pagination v-model="devicePage" :total-items="deviceTotal" :items-per-page="pageSize" @change="loadDevices" />
        </van-tab>
      </van-tabs>
    </template>

    <!-- 可借设备选择 -->
    <van-popup v-model:show="showDevice" position="bottom">
      <van-picker :columns="deviceColumns" title="选择可借设备" @confirm="onDevicePick" @cancel="showDevice = false" />
    </van-popup>
    <!-- 设备类型选择 -->
    <van-popup v-model:show="showType" position="bottom">
      <van-picker :columns="typeColumns" title="选择设备类型" @confirm="onTypePick" @cancel="showType = false" />
    </van-popup>
    <!-- 预计归还日期 -->
    <van-popup v-model:show="showReturnDate" position="bottom">
      <van-date-picker v-model="returnDate" title="选择预计归还日期" :min-date="minDate" @confirm="onReturnDate" @cancel="showReturnDate = false" />
    </van-popup>
    <!-- 损坏赔付登记 -->
    <van-popup v-model:show="showDamage" position="bottom" round>
      <div class="damage-panel">
        <div class="damage-title">损坏赔付登记 · {{ damageTarget?.device_no }}</div>
        <van-cell-group inset>
          <van-field v-model="damageForm.damage_desc" label="损坏说明" type="textarea" rows="2" placeholder="描述损坏情况" />
          <van-field v-model="damageForm.compensation" type="number" label="赔偿金额(元)" :placeholder="`押金 ${damageTarget?.deposit ?? 0} 元，不足部分计入会员欠款`" />
        </van-cell-group>
        <div class="submit-btn">
          <van-button round block type="danger" @click="doDamage">确认登记</van-button>
        </div>
      </div>
    </van-popup>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { showConfirmDialog, showSuccessToast, showToast } from 'vant'
import StatusBadge from '@/components/StatusBadge.vue'
import EmptyState from '@/components/EmptyState.vue'
import { listPeripherals, createPeripheral, updatePeripheralStatus, type Peripheral } from '@/api/peripheral'
import { listRentals, listMyRentals, createRental, returnRental, damageRental, type Rental } from '@/api/rental'
import { formatMoney, formatTime } from '@/utils/format'
import { useAuth } from '@/hooks/useAuth'
import { PERIPHERAL_STATUS, PERIPHERAL_TYPE_TEXT, RENTAL_STATUS } from '@/constants'

const { user, isStaffOrAdmin, fetchProfile } = useAuth()
const pageSize = 10
const tab = ref(0)

const rentalStatusOptions = [
  { text: '全部状态', value: '' },
  { text: '在借', value: 'renting' },
  { text: '已归还', value: 'returned' },
  { text: '损坏已赔付', value: 'damaged' },
]
const typeColumns = [
  { text: '键盘', value: 'keyboard' },
  { text: '鼠标', value: 'mouse' },
  { text: '耳机', value: 'headset' },
]

function typeText(t: string): string {
  return PERIPHERAL_TYPE_TEXT[t] || t
}

// ---------- 会员侧 ----------
const myList = ref<Rental[]>([])
const myTotal = ref(0)
const myPage = ref(1)
const myStatus = ref('')

async function loadMine() {
  const data = await listMyRentals({ page: myPage.value, page_size: pageSize, status: myStatus.value || undefined })
  myList.value = data.list
  myTotal.value = data.total
}

function mineLabel(r: Rental): string {
  const parts = [`押金 ${formatMoney(r.deposit)}`, `预计归还 ${formatTime(r.expected_return_at)}`]
  if (r.status === RENTAL_STATUS.RETURNED) {
    parts.push(`已退押金 ${formatMoney(r.refund_amount)}`)
  }
  if (r.status === RENTAL_STATUS.DAMAGED) {
    parts.push(`赔偿 ${formatMoney(r.compensation)}`)
    if (r.debt_amount > 0) parts.push(`计入欠款 ${formatMoney(r.debt_amount)}`)
  }
  return parts.join(' · ')
}

// ---------- 店员侧：租借 ----------
const rentalList = ref<Rental[]>([])
const rentalTotal = ref(0)
const rentalPage = ref(1)
const rentalStatus = ref('')
const rentForm = ref({ user_id: '', peripheral_id: 0, deviceLabel: '', deposit: '', expectedLabel: '', expected_return_at: '', remark: '' })
const showDevice = ref(false)
const showReturnDate = ref(false)
const returnDate = ref<string[]>([])
const minDate = new Date()
const deviceColumns = ref<{ text: string; value: number }[]>([])

async function loadRentals() {
  const data = await listRentals({ page: rentalPage.value, page_size: pageSize, status: rentalStatus.value || undefined })
  rentalList.value = data.list
  rentalTotal.value = data.total
}

async function openDevicePicker() {
  const data = await listPeripherals({ page: 1, page_size: 100, status: 'available' })
  deviceColumns.value = data.list.map((d) => ({ text: `${d.device_no} · ${d.name}（${typeText(d.device_type)}）`, value: d.id }))
  if (!deviceColumns.value.length) {
    showToast('暂无可借设备')
    return
  }
  showDevice.value = true
}

function onDevicePick({ selectedOptions }: any) {
  const opt = selectedOptions[0]
  rentForm.value.peripheral_id = opt.value
  rentForm.value.deviceLabel = opt.text
  showDevice.value = false
}

function toRFC3339(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const off = -d.getTimezoneOffset()
  const sign = off >= 0 ? '+' : '-'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:00${sign}${pad(Math.floor(Math.abs(off) / 60))}:${pad(Math.abs(off) % 60)}`
}

function onReturnDate({ selectedValues }: any) {
  const [y, m, d] = selectedValues.map(Number)
  const dt = new Date(y, m - 1, d, 22, 0, 0)
  rentForm.value.expected_return_at = toRFC3339(dt)
  rentForm.value.expectedLabel = `${y}-${m}-${d} 22:00`
  showReturnDate.value = false
}

async function createRent() {
  const userId = Number(rentForm.value.user_id)
  const deposit = Number(rentForm.value.deposit)
  if (!userId || !rentForm.value.peripheral_id || !rentForm.value.expected_return_at) {
    showToast('请填写会员ID、选择设备与预计归还时间')
    return
  }
  if (Number.isNaN(deposit) || deposit < 0) {
    showToast('押金金额无效')
    return
  }
  await createRental({ user_id: userId, peripheral_id: rentForm.value.peripheral_id, deposit, expected_return_at: rentForm.value.expected_return_at, remark: rentForm.value.remark })
  showSuccessToast('租借登记成功')
  rentForm.value = { user_id: '', peripheral_id: 0, deviceLabel: '', deposit: '', expectedLabel: '', expected_return_at: '', remark: '' }
  loadRentals()
  loadDevices()
}

async function doReturn(r: Rental) {
  await showConfirmDialog({ title: '完好归还', message: `确认设备 ${r.device_no} 完好归还？押金 ${formatMoney(r.deposit)} 将释放回会员余额。` })
  await returnRental(r.id)
  showSuccessToast('归还成功，押金已释放')
  loadRentals()
  loadDevices()
}

const showDamage = ref(false)
const damageTarget = ref<Rental | null>(null)
const damageForm = ref({ damage_desc: '', compensation: '' })

function openDamage(r: Rental) {
  damageTarget.value = r
  damageForm.value = { damage_desc: '', compensation: '' }
  showDamage.value = true
}

async function doDamage() {
  const target = damageTarget.value
  if (!target) return
  const compensation = Number(damageForm.value.compensation)
  if (!damageForm.value.damage_desc) {
    showToast('请填写损坏说明')
    return
  }
  if (Number.isNaN(compensation) || compensation < 0) {
    showToast('赔偿金额无效')
    return
  }
  await damageRental(target.id, { damage_desc: damageForm.value.damage_desc, compensation })
  showDamage.value = false
  showSuccessToast('损坏赔付登记成功')
  loadRentals()
  loadDevices()
}

function staffLabel(r: Rental): string {
  const parts = [`押金 ${formatMoney(r.deposit)}`, `预计 ${formatTime(r.expected_return_at)}`]
  if (r.status === RENTAL_STATUS.RETURNED) {
    parts.push(`退还 ${formatMoney(r.refund_amount)}`, `处理人 ${r.handler_name || '-'}`)
  }
  if (r.status === RENTAL_STATUS.DAMAGED) {
    parts.push(`赔偿 ${formatMoney(r.compensation)}`, `欠款 ${formatMoney(r.debt_amount)}`, `处理人 ${r.handler_name || '-'}`)
  }
  return parts.join(' · ')
}

// ---------- 店员侧：设备 ----------
const deviceList = ref<Peripheral[]>([])
const deviceTotal = ref(0)
const devicePage = ref(1)
const devForm = ref({ device_no: '', device_type: '', typeLabel: '', name: '' })
const showType = ref(false)

async function loadDevices() {
  const data = await listPeripherals({ page: devicePage.value, page_size: pageSize })
  deviceList.value = data.list
  deviceTotal.value = data.total
}

function onTypePick({ selectedOptions }: any) {
  const opt = selectedOptions[0]
  devForm.value.device_type = opt.value
  devForm.value.typeLabel = opt.text
  showType.value = false
}

async function createDev() {
  if (!devForm.value.device_no || !devForm.value.device_type || !devForm.value.name) {
    showToast('请填写设备编号、类型与名称')
    return
  }
  await createPeripheral({ device_no: devForm.value.device_no, device_type: devForm.value.device_type, name: devForm.value.name })
  showSuccessToast('设备登记成功')
  devForm.value = { device_no: '', device_type: '', typeLabel: '', name: '' }
  loadDevices()
}

async function setStatus(d: Peripheral, status: string) {
  await updatePeripheralStatus(d.id, status)
  showSuccessToast('状态已更新')
  loadDevices()
}

onMounted(() => {
  if (isStaffOrAdmin.value) {
    loadRentals()
    loadDevices()
  } else {
    fetchProfile()
    loadMine()
  }
})
</script>

<style scoped>
.submit-btn { margin: 12px 16px; }
.op-btn { margin-left: 6px; }
.debt-text { color: #ee0a24; font-weight: 600; }
.damage-panel { padding: 16px 0 8px; }
.damage-title { padding: 0 16px 12px; font-weight: 600; }
</style>
