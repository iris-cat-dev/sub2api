<template>
  <AppLayout>
    <div class="mx-auto min-w-0 max-w-[1600px] space-y-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <button class="mb-2 inline-flex items-center gap-2 text-sm text-gray-500 transition-colors hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-400" @click="router.push('/admin/groups')">
            <Icon name="arrowLeft" size="sm" />
            {{ t('admin.groupModelPricing.back') }}
          </button>
          <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ t('admin.groupModelPricing.title') }}
          </h1>
          <p v-if="group" class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">
            {{ group.name }} · {{ t(`admin.groups.platforms.${group.platform}`) }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2 self-start sm:self-auto">
          <button
            class="btn btn-secondary whitespace-nowrap"
            :disabled="loading || !group || syncingWhitelist"
            @click="syncWhitelist"
          >
            <Icon name="sync" size="sm" class="mr-1.5" :class="{ 'animate-spin': syncingWhitelist }" />
            {{ syncingWhitelist ? t('admin.groupModelPricing.syncingWhitelist') : t('admin.groupModelPricing.syncWhitelist') }}
          </button>
          <button class="btn btn-primary whitespace-nowrap" :disabled="loading || !group || hasNewRow || syncingWhitelist" @click="addRow">
            <Icon name="plus" size="sm" class="mr-1.5" />
            {{ t('admin.groupModelPricing.addRow') }}
          </button>
        </div>
      </div>

      <div v-if="group" class="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-xl border px-4 py-3 text-sm leading-5" :class="group.long_context_pricing_enabled ? 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300' : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-300'">
        <span class="shrink-0 font-medium">
          {{ group.long_context_pricing_enabled ? t('admin.groupModelPricing.longContextEnabled') : t('admin.groupModelPricing.longContextDisabled') }}
        </span>
        <span class="opacity-80">{{ t('admin.groupModelPricing.longContextHint') }}</span>
      </div>

      <div v-if="loading" class="flex min-h-48 items-center justify-center text-gray-500">
        {{ t('admin.groupModelPricing.loading') }}
      </div>

      <div v-else-if="rows.length === 0" class="rounded-xl border border-dashed border-gray-300 px-6 py-16 text-center dark:border-dark-600">
        <p class="text-gray-500 dark:text-gray-400">{{ t('admin.groupModelPricing.empty') }}</p>
      </div>

      <div v-else class="min-w-0 overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="max-w-full overflow-x-auto">
          <table class="w-full min-w-[1160px] table-fixed border-collapse text-left text-sm">
            <colgroup>
              <col class="w-[260px]" />
              <col class="w-[130px]" />
              <col class="w-[155px]" />
              <col />
              <col class="w-[132px]" />
            </colgroup>
            <thead class="bg-gray-50/95 text-xs font-semibold text-gray-500 dark:bg-dark-900 dark:text-gray-400">
            <tr>
              <th class="border-b border-gray-200 px-4 py-3.5 dark:border-dark-700">{{ t('admin.groupModelPricing.models') }}</th>
              <th class="border-b border-gray-200 px-3 py-3.5 dark:border-dark-700">{{ t('admin.groupModelPricing.billingMode') }}</th>
              <th class="border-b border-gray-200 px-3 py-3.5 dark:border-dark-700">{{ t('admin.groupModelPricing.multiplier') }}</th>
              <th class="border-b border-gray-200 px-4 py-3.5 dark:border-dark-700">{{ t('admin.groupModelPricing.prices') }}</th>
              <th class="border-b border-gray-200 px-3 py-3.5 text-center dark:border-dark-700">{{ t('admin.groupModelPricing.actions') }}</th>
            </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
            <tr v-for="row in rows" :key="row.key" class="align-top transition-colors hover:bg-gray-50/70 dark:hover:bg-dark-700/50">
              <td class="p-4">
                <textarea v-model="row.modelsText" rows="5" class="input h-[142px] resize-none rounded-lg px-3 py-2 leading-5" :placeholder="t('admin.groupModelPricing.modelsPlaceholder')" @change="loadOfficial(row)" />
                <p class="mt-1.5 break-words text-xs leading-4 text-gray-400">{{ t('admin.groupModelPricing.modelsHint') }}</p>
              </td>
              <td class="px-3 py-4">
                <select v-model="row.billing_mode" class="input rounded-lg px-3 py-2">
                  <option v-for="mode in billingModes" :key="mode.value" :value="mode.value">{{ t(mode.labelKey) }}</option>
                </select>
              </td>
              <td class="px-3 py-4">
                <input v-model="row.multiplier" type="number" min="0.000001" step="0.01" class="input rounded-lg px-3 py-2 font-mono tabular-nums" :class="{ 'border-red-500': !isValidMultiplier(row.multiplier) }" @input="applyMultiplier(row)" />
                <p v-if="!isValidMultiplier(row.multiplier)" class="mt-1.5 break-words text-xs leading-4 text-red-500">{{ t('admin.groupModelPricing.multiplierInvalid') }}</p>
                <p v-else class="mt-1.5 break-words text-xs leading-4 text-gray-400">{{ t('admin.groupModelPricing.multiplierHint') }}</p>
              </td>
              <td class="px-4 py-4">
                <div class="grid grid-cols-4 gap-x-3 gap-y-3">
                  <label v-for="column in priceColumns" :key="column.key" class="block min-w-0">
                    <span class="mb-1 flex min-w-0 items-baseline justify-between gap-1.5">
                      <span class="truncate text-xs font-medium text-gray-700 dark:text-gray-300">{{ t(column.labelKey) }}</span>
                      <span class="shrink-0 text-[10px] text-gray-400">{{ column.unit }}</span>
                    </span>
                    <span class="mb-1 block min-h-4 truncate text-[11px] leading-4 text-gray-400" :title="officialPriceText(row, column)">
                      <template v-if="column.official">
                        {{ t('admin.groupModelPricing.official') }}:
                        <span v-if="row.officialLoading">{{ t('admin.groupModelPricing.loadingShort') }}</span>
                        <span v-else-if="row.officialError" class="text-red-500">{{ t('admin.groupModelPricing.officialLoadFailed') }}</span>
                        <span v-else class="font-mono tabular-nums">{{ formatOfficial(row, column.key) }}</span>
                      </template>
                      <template v-else>{{ t('admin.groupModelPricing.noOfficialPrice') }}</template>
                    </span>
                    <input v-model="row[column.key]" type="number" min="0" step="any" class="input h-9 rounded-lg px-2 py-1.5 text-right font-mono text-xs tabular-nums" :aria-label="t(column.labelKey)" />
                  </label>
                </div>
              </td>
              <td class="px-3 py-4">
                <div class="flex flex-col gap-2">
                  <button class="btn btn-primary btn-sm w-full justify-center whitespace-nowrap" :disabled="row.saving || !canSave(row)" @click="saveRow(row)">
                    <Icon name="check" size="sm" class="mr-1 shrink-0" />
                    {{ row.saving ? t('admin.groupModelPricing.saving') : t('admin.groupModelPricing.save') }}
                  </button>
                  <button class="btn btn-secondary btn-sm w-full justify-center whitespace-nowrap text-red-600 dark:text-red-400" :disabled="row.saving" @click="deleteRow(row)">
                    <Icon name="trash" size="sm" class="mr-1 shrink-0" />
                    {{ t('admin.groupModelPricing.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

import { adminAPI } from '@/api/admin'
import type { ChannelModelPricing } from '@/api/admin/channels'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { mTokToPerToken, perTokenToMTok, toNullableNumber } from '@/components/admin/channel/types'
import type { BillingMode } from '@/constants/channel'
import { useAppStore } from '@/stores/app'
import type { AdminGroup } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  applyOfficialPriceMultiplier,
  findUnsyncedAllowlistModels,
  inferOfficialPriceMultiplier,
  modelPricingTokenFields,
  type ModelPricingTokenField,
  type ModelPricingTokenValues,
  type OfficialModelPricing
} from './groupModelPricing'

interface PricingRow extends ModelPricingTokenValues {
  key: string
  index: number
  isNew: boolean
  raw: ChannelModelPricing
  modelsText: string
  billing_mode: BillingMode
  per_request_price: number | string | null
  multiplier: number | string
  official: OfficialModelPricing
  officialLoading: boolean
  officialError: boolean
  saving: boolean
}

const priceColumns: Array<{
  key: ModelPricingTokenField | 'per_request_price'
  labelKey: string
  unit: string
  official: boolean
}> = [
  { key: 'input_price', labelKey: 'admin.groupModelPricing.inputPrice', unit: '¥/MTok', official: true },
  { key: 'output_price', labelKey: 'admin.groupModelPricing.outputPrice', unit: '¥/MTok', official: true },
  { key: 'cache_write_price', labelKey: 'admin.groupModelPricing.cacheWrite5m', unit: '¥/MTok', official: true },
  { key: 'cache_write_1h_price', labelKey: 'admin.groupModelPricing.cacheWrite1h', unit: '¥/MTok', official: true },
  { key: 'cache_read_price', labelKey: 'admin.groupModelPricing.cacheRead', unit: '¥/MTok', official: true },
  { key: 'image_input_price', labelKey: 'admin.groupModelPricing.imageInput', unit: '¥/MTok', official: true },
  { key: 'image_output_price', labelKey: 'admin.groupModelPricing.imageOutput', unit: '¥/MTok', official: true },
  { key: 'per_request_price', labelKey: 'admin.groupModelPricing.perRequest', unit: '¥/request', official: false }
]

const billingModes: Array<{ value: BillingMode; labelKey: string }> = [
  { value: 'token', labelKey: 'admin.groupModelPricing.modeToken' },
  { value: 'per_request', labelKey: 'admin.groupModelPricing.modePerRequest' },
  { value: 'image', labelKey: 'admin.groupModelPricing.modeImage' },
  { value: 'video', labelKey: 'admin.groupModelPricing.modeVideo' }
]

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const group = ref<AdminGroup | null>(null)
const rows = ref<PricingRow[]>([])
const loading = ref(true)
const syncingWhitelist = ref(false)
let newRowSequence = 0

const groupId = computed(() => Number(route.params.id))
const hasNewRow = computed(() => rows.value.some(row => row.isNew))

function parseModels(value: string): string[] {
  return [...new Set(value.split(/[\n,]+/).map(model => model.trim()).filter(Boolean))]
}

function createRow(pricing: ChannelModelPricing, index: number, isNew = false): PricingRow {
  return {
    key: isNew ? `new-${++newRowSequence}` : `saved-${index}`,
    index,
    isNew,
    raw: pricing,
    modelsText: (pricing.models || []).join('\n'),
    billing_mode: pricing.billing_mode || 'token',
    input_price: perTokenToMTok(pricing.input_price),
    output_price: perTokenToMTok(pricing.output_price),
    cache_write_price: perTokenToMTok(pricing.cache_write_price),
    cache_write_1h_price: perTokenToMTok(pricing.cache_write_1h_price),
    cache_read_price: perTokenToMTok(pricing.cache_read_price),
    image_input_price: perTokenToMTok(pricing.image_input_price),
    image_output_price: perTokenToMTok(pricing.image_output_price),
    per_request_price: pricing.per_request_price,
    multiplier: pricing.official_price_multiplier ?? 1,
    official: {},
    officialLoading: false,
    officialError: false,
    saving: false
  }
}

function emptyPricing(): ChannelModelPricing {
  return {
    platform: group.value?.platform || '',
    models: [],
    billing_mode: 'token',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_write_1h_price: null,
    cache_read_price: null,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    time_pricing: null
  }
}

function rebuildRows(updatedGroup: AdminGroup, drafts: PricingRow[] = []): void {
  group.value = updatedGroup
  const savedRows = (updatedGroup.model_pricing || []).map((pricing, index) => createRow(pricing, index))
  const nextIndex = savedRows.length
  for (const draft of drafts) draft.index = nextIndex
  rows.value = [...savedRows, ...drafts]
  for (const row of rows.value) void loadOfficial(row)
}

function firstModel(row: PricingRow): string {
  return parseModels(row.modelsText)[0] || ''
}

async function loadOfficial(row: PricingRow): Promise<void> {
  const model = firstModel(row)
  row.official = {}
  row.officialError = false
  row.officialLoading = false
  if (!model) return

  row.officialLoading = true
  try {
    const pricing = await adminAPI.channels.getModelDefaultPricing(model)
    if (firstModel(row) !== model) return
    if (!pricing.found) return

    const official: OfficialModelPricing = {}
    for (const field of modelPricingTokenFields) {
      const value = pricing[field]
      if (value !== undefined && value !== null) official[field] = perTokenToMTok(value)
    }
    row.official = official
    if (row.raw.official_price_multiplier == null && Number(row.multiplier) === 1) {
      row.multiplier = inferOfficialPriceMultiplier(row, official) ?? row.multiplier
    }
  } catch {
    if (firstModel(row) === model) row.officialError = true
  } finally {
    if (firstModel(row) === model) row.officialLoading = false
  }
}

function isValidMultiplier(value: number | string): boolean {
  const numeric = Number(value)
  return Number.isFinite(numeric) && numeric > 0
}

function applyMultiplier(row: PricingRow): void {
  const multiplier = Number(row.multiplier)
  if (!isValidMultiplier(row.multiplier)) return
  const next = applyOfficialPriceMultiplier(row, row.official, multiplier)
  for (const field of modelPricingTokenFields) row[field] = next[field]
}

function formatOfficial(row: PricingRow, field: ModelPricingTokenField | 'per_request_price'): string {
  if (field === 'per_request_price') return '—'
  const value = row.official[field]
  return value === undefined || value === null ? '—' : `¥${value} / MTok`
}

function officialPriceText(row: PricingRow, column: (typeof priceColumns)[number]): string {
  if (!column.official) return t('admin.groupModelPricing.noOfficialPrice')
  if (row.officialLoading) return t('admin.groupModelPricing.loadingShort')
  if (row.officialError) return t('admin.groupModelPricing.officialLoadFailed')
  return `${t('admin.groupModelPricing.official')}: ${formatOfficial(row, column.key)}`
}

function canSave(row: PricingRow): boolean {
  return parseModels(row.modelsText).length > 0 && isValidMultiplier(row.multiplier)
}

function rowToPricing(row: PricingRow): ChannelModelPricing {
  return {
    ...row.raw,
    platform: group.value?.platform || row.raw.platform,
    models: parseModels(row.modelsText),
    billing_mode: row.billing_mode,
    input_price: mTokToPerToken(row.input_price),
    output_price: mTokToPerToken(row.output_price),
    cache_write_price: mTokToPerToken(row.cache_write_price),
    cache_write_1h_price: mTokToPerToken(row.cache_write_1h_price),
    cache_read_price: mTokToPerToken(row.cache_read_price),
    image_input_price: mTokToPerToken(row.image_input_price),
    image_output_price: mTokToPerToken(row.image_output_price),
    per_request_price: toNullableNumber(row.per_request_price),
    official_price_multiplier: Number(row.multiplier)
  }
}

function addRow(): void {
  if (!group.value || hasNewRow.value) return
  rows.value.push(createRow(emptyPricing(), group.value.model_pricing?.length || 0, true))
}

async function syncWhitelist(): Promise<void> {
  if (!group.value || syncingWhitelist.value) return

  const allowlistModels = group.value.model_allowlist?.models || []
  if (allowlistModels.length === 0) {
    appStore.showWarning(t('admin.groupModelPricing.noWhitelistModels'))
    return
  }

  const pricedModels = rows.value.flatMap(row => parseModels(row.modelsText))
  const missingModels = findUnsyncedAllowlistModels(allowlistModels, pricedModels)
  if (missingModels.length === 0) {
    appStore.showInfo(t('admin.groupModelPricing.whitelistAlreadySynced'))
    return
  }

  syncingWhitelist.value = true
  try {
    const nextIndex = group.value.model_pricing?.length || 0
    const drafts = missingModels.map(model => {
      const row = createRow(emptyPricing(), nextIndex, true)
      row.modelsText = model
      return row
    })

    await Promise.all(drafts.map(async row => {
      await loadOfficial(row)
      applyMultiplier(row)
    }))
    rows.value.push(...drafts)
    appStore.showSuccess(t('admin.groupModelPricing.whitelistSynced', { count: drafts.length }))
  } finally {
    syncingWhitelist.value = false
  }
}

async function saveRow(row: PricingRow): Promise<void> {
  if (!group.value || !canSave(row)) return
  row.saving = true
  const drafts = rows.value.filter(candidate => candidate.isNew && candidate !== row)
  try {
    const updated = await adminAPI.groups.saveModelPricingEntry(group.value.id, row.index, rowToPricing(row))
    rebuildRows(updated, drafts)
    appStore.showSuccess(t('admin.groupModelPricing.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.groupModelPricing.saveFailed')))
    row.saving = false
  }
}

async function deleteRow(row: PricingRow): Promise<void> {
  if (!group.value) return
  if (row.isNew) {
    rows.value = rows.value.filter(candidate => candidate !== row)
    return
  }

  row.saving = true
  const drafts = rows.value.filter(candidate => candidate.isNew)
  try {
    const updated = await adminAPI.groups.deleteModelPricingEntry(group.value.id, row.index)
    rebuildRows(updated, drafts)
    appStore.showSuccess(t('admin.groupModelPricing.deleteSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.groupModelPricing.deleteFailed')))
    row.saving = false
  }
}

onMounted(async () => {
  if (!Number.isSafeInteger(groupId.value) || groupId.value <= 0) {
    appStore.showError(t('admin.groupModelPricing.loadFailed'))
    await router.replace('/admin/groups')
    return
  }

  try {
    rebuildRows(await adminAPI.groups.getById(groupId.value))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.groupModelPricing.loadFailed')))
  } finally {
    loading.value = false
  }
})
</script>


[Showing lines 1-300 of 307. Use :301 to continue]