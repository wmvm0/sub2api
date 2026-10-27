<template>
  <BaseDialog :show="true" :title="t('admin.users.inviterBinding.title')" @close="close">
    <div class="space-y-4">
      <p class="font-medium">{{ user.email }} <span class="text-gray-400">#{{ user.id }}</span></p>
      <p class="text-sm text-gray-500">{{ t('admin.users.inviterBinding.hint') }}</p>
      <p v-if="loading">{{ t('admin.users.inviterBinding.loading') }}</p>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <template v-if="binding?.inviter">
        <p>{{ t('admin.users.inviterBinding.bound') }}</p>
        <p class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
          {{ binding.inviter.username }} {{ binding.inviter.email }} #{{ binding.inviter.id }}
        </p>
      </template>
      <template v-else-if="binding && !loading">
        <form class="flex gap-2" @submit.prevent="search">
          <input v-model="query" class="input flex-1" :disabled="saving" :aria-label="t('admin.users.inviterBinding.search')" :placeholder="t('admin.users.inviterBinding.search')" />
          <button class="btn btn-secondary" :disabled="searching || saving || !query.trim()">{{ t('admin.users.inviterBinding.searchButton') }}</button>
        </form>
        <label class="block text-sm">
          {{ t('admin.users.inviterBinding.select') }}
          <select v-model="selectedId" class="input mt-2" :disabled="saving || searching" @change="confirmed = false">
            <option :value="0">{{ t('admin.users.inviterBinding.select') }}</option>
            <option v-for="candidate in candidates" :key="candidate.id" :value="candidate.id">
              {{ candidate.username }} {{ candidate.email }} #{{ candidate.id }}
            </option>
          </select>
        </label>
        <p v-if="searched && !searching && candidates.length === 0" class="text-sm text-gray-500">{{ t('admin.users.inviterBinding.empty') }}</p>
        <label class="flex items-start gap-2 text-sm">
          <input v-model="confirmed" type="checkbox" class="mt-1" :disabled="saving || !selectedId" />
          {{ t('admin.users.inviterBinding.confirm') }}
        </label>
        <button class="btn btn-primary" :disabled="saving || !confirmed || !selectedId || searching" @click="save">
          {{ saving ? t('admin.users.inviterBinding.loading') : t('admin.users.inviterBinding.save') }}
        </button>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getUserInviter, bindUserInviter, lookupUsers } from '@/api/admin/affiliates'
import type { SimpleUser, UserInviterBinding } from '@/api/admin/affiliates'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'

const props = defineProps<{ user: AdminUser }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const binding = ref<UserInviterBinding | null>(null)
const loading = ref(true)
const saving = ref(false)
const searching = ref(false)
const searched = ref(false)
const query = ref('')
const candidates = ref<SimpleUser[]>([])
const selectedId = ref(0)
const confirmed = ref(false)
const error = ref('')
let alive = true
onBeforeUnmount(() => { alive = false })
function close() { if (!saving.value) emit('close') }
function failure(e: unknown) {
  const reason = (e as { response?: { data?: { reason?: string } }; reason?: string })?.response?.data?.reason
    ?? (e as { reason?: string })?.reason
  if (reason === 'AFFILIATE_ALREADY_BOUND') return t('admin.users.inviterBinding.conflict')
  if (reason === 'AFFILIATE_INVALID_INVITER') return t('admin.users.inviterBinding.invalid')
  return t('admin.users.inviterBinding.failed')
}
onMounted(async () => {
  try { const result = await getUserInviter(props.user.id); if (alive) binding.value = result }
  catch (e) { if (alive) error.value = failure(e) }
  finally { if (alive) loading.value = false }
})
async function search() {
  if (searching.value || saving.value || !query.value.trim()) return
  searching.value = true; error.value = ''; selectedId.value = 0; confirmed.value = false
  try { const result = await lookupUsers(query.value.trim()); if (alive) candidates.value = result.filter(u => u.id !== props.user.id) }
  catch (e) { if (alive) { error.value = failure(e); candidates.value = [] } }
  finally { if (alive) { searching.value = false; searched.value = true } }
}
async function save() {
  if (saving.value || !confirmed.value || !selectedId.value || binding.value?.inviter) return
  saving.value = true; error.value = ''
  try {
    await bindUserInviter(props.user.id, selectedId.value)
    appStore.showSuccess(t('admin.users.inviterBinding.success'))
    emit('close')
  } catch (e) {
    if (alive) {
      error.value = failure(e)
      // Re-read after conflicts or a lost response: never invite the admin to
      // overwrite a binding that may already have committed.
      try { binding.value = await getUserInviter(props.user.id) } catch { /* keep the original error */ }
      confirmed.value = false
    }
  } finally { if (alive) saving.value = false }
}
</script>
