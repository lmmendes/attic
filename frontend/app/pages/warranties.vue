<script setup lang="ts">
import type { Warranty } from '~/types/api'

definePageMeta({
  middleware: 'auth'
})

const days = ref(30)

const { data: warranties, status } = useApi<Warranty[]>(
  () => `/api/warranties/expiring?days=${days.value}`
)

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString()
}

function daysUntilExpiry(endDate?: string) {
  if (!endDate) return null
  const end = new Date(endDate)
  const now = new Date()
  const diff = Math.ceil((end.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  return diff
}

function getExpiryColor(endDate?: string) {
  const days = daysUntilExpiry(endDate)
  if (days === null) return 'neutral'
  if (days < 0) return 'error'
  if (days <= 7) return 'warning'
  if (days <= 30) return 'info'
  return 'success'
}

const dayOptions = [
  { label: 'Next 7 days', value: 7 },
  { label: 'Next 30 days', value: 30 },
  { label: 'Next 90 days', value: 90 },
  { label: 'Next 365 days', value: 365 }
]
</script>

<template>
  <UContainer>
    <div class="py-8">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold">Warranties</h1>
        <USelectMenu
          v-model="days"
          :items="dayOptions"
          value-key="value"
          class="w-48"
        />
      </div>

      <UCard>
        <UTable
          :data="warranties || []"
          :columns="[
            { accessorKey: 'asset_id', id: 'asset_id', header: 'Asset' },
            { accessorKey: 'provider', id: 'provider', header: 'Provider' },
            { accessorKey: 'policy_number', id: 'policy_number', header: 'Policy #' },
            { accessorKey: 'end_date', id: 'end_date', header: 'Expires' },
            { id: 'status', header: 'Status' }
          ]"
          :loading="status === 'pending'"
        >
          <template #asset_id-cell="{ row }">
            <NuxtLink
              :to="`/assets/${row.original.asset_id}`"
              class="text-primary hover:underline font-medium"
            >
              View Asset
            </NuxtLink>
          </template>
          <template #end_date-cell="{ row }">
            {{ formatDate(row.original.end_date) }}
          </template>
          <template #status-cell="{ row }">
            <UBadge :color="getExpiryColor(row.original.end_date)">
              <template v-if="daysUntilExpiry(row.original.end_date) !== null">
                <span v-if="daysUntilExpiry(row.original.end_date)! < 0">
                  Expired {{ Math.abs(daysUntilExpiry(row.original.end_date)!) }} days ago
                </span>
                <span v-else-if="daysUntilExpiry(row.original.end_date) === 0">
                  Expires today
                </span>
                <span v-else>
                  {{ daysUntilExpiry(row.original.end_date) }} days left
                </span>
              </template>
              <template v-else>
                No expiry
              </template>
            </UBadge>
          </template>
        </UTable>

        <div v-if="!warranties?.length && status !== 'pending'" class="text-center py-8 text-muted">
          <UIcon name="i-lucide-shield-check" class="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No warranties expiring in the next {{ days }} days</p>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>
