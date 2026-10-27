import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserInviterModal from '../UserInviterModal.vue'
import type { AdminUser } from '@/types'
const api = vi.hoisted(() => ({ get: vi.fn(), bind: vi.fn(), lookup: vi.fn(), success: vi.fn() }))
vi.mock('@/api/admin/affiliates', () => ({ getUserInviter: api.get, bindUserInviter: api.bind, lookupUsers: api.lookup }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: api.success }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const user = { id: 2, email: 'target@example.com' } as AdminUser
function render() { return mount(UserInviterModal, { props: { user }, global: { stubs: { BaseDialog: { template: '<div><slot /></div>' } } } }) }
beforeEach(() => { vi.resetAllMocks(); api.get.mockResolvedValue({ user_id: 2, inviter: null }); api.bind.mockResolvedValue(undefined) })
describe('UserInviterModal', () => {
 it('shows existing inviter without editable controls', async () => {
  api.get.mockResolvedValue({ user_id: 2, inviter: { id: 1, email: 'owner@example.com', username: 'owner' } })
  const w = render(); await flushPromises()
  expect(w.text()).toContain('owner@example.com'); expect(w.find('select').exists()).toBe(false)
 })
 it('searches, excludes self, requires confirmation and submits numeric IDs', async () => {
  api.lookup.mockResolvedValue([user, { id: 3, email: 'inviter@example.com', username: 'inviter' }])
  const w = render(); await flushPromises()
  await w.get('input:not([type=checkbox])').setValue('inviter'); await w.get('form').trigger('submit'); await flushPromises()
  expect(w.findAll('option')).toHaveLength(2)
  await w.get('select').setValue('3')
  expect(w.get('button.btn-primary').attributes('disabled')).toBeDefined()
  await w.get('input[type=checkbox]').setValue(true); await w.get('button.btn-primary').trigger('click'); await flushPromises()
  expect(api.bind).toHaveBeenCalledWith(2, 3); expect(w.emitted('close')).toHaveLength(1)
 })
 it('refreshes and locks after another admin binds concurrently', async () => {
  api.lookup.mockResolvedValue([{ id: 3, email: 'third@example.com', username: '' }])
  const w = render(); await flushPromises()
  await w.get('input:not([type=checkbox])').setValue('third'); await w.get('form').trigger('submit'); await flushPromises()
  await w.get('select').setValue('3'); await w.get('input[type=checkbox]').setValue(true)
  api.bind.mockRejectedValue({ reason: 'AFFILIATE_ALREADY_BOUND' })
  api.get.mockResolvedValue({ user_id: 2, inviter: { id: 4, email: 'winner@example.com', username: '' } })
  await w.get('button.btn-primary').trigger('click'); await flushPromises()
  expect(w.text()).toContain('winner@example.com'); expect(w.find('select').exists()).toBe(false); expect(w.emitted('close')).toBeUndefined()
 })
 it('does not offer editing when the initial status cannot be loaded', async () => {
  api.get.mockRejectedValue(new Error('network'))
  const w = render(); await flushPromises()
  expect(w.find('[role=alert]').exists()).toBe(true); expect(w.find('select').exists()).toBe(false)
 })
})
