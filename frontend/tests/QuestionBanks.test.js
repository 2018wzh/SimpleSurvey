import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QuestionBanks from '../src/views/QuestionBanks.vue'

const apiMocks = vi.hoisted(() => ({
    getQuestionBanks: vi.fn(),
    getMyQuestions: vi.fn(),
    getUsers: vi.fn(),
    createQuestionBank: vi.fn(),
    updateQuestionBank: vi.fn(),
    addQuestionBankItem: vi.fn(),
    getQuestionVersions: vi.fn(),
    removeQuestionBankItem: vi.fn(),
    shareQuestionBank: vi.fn(),
    unshareQuestionBank: vi.fn()
}))

vi.mock('../src/api', () => ({
    default: apiMocks
}))

function flushPromises() {
    return new Promise(resolve => setTimeout(resolve, 0))
}

describe('QuestionBanks.vue', () => {
    beforeEach(() => {
        Object.values(apiMocks).forEach(mock => mock.mockReset())
    })

    it('creates a bank and manages items and sharing for an expanded bank', async () => {
        apiMocks.getQuestionBanks.mockResolvedValue({
            data: {
                data: {
                    items: [
                        {
                            id: 'bank-1',
                            name: '基础题库',
                            description: '常用题目',
                            visibility: 'private',
                            items: [{ questionId: 'q-1', pinnedVersionId: 'v-1' }],
                            sharedWith: [{ userId: 'u-1', permission: 'use' }]
                        }
                    ]
                }
            }
        })
        apiMocks.getMyQuestions.mockResolvedValue({
            data: {
                data: {
                    items: [
                        { id: 'q-1', questionKey: '年龄', currentVersion: 2 }
                    ]
                }
            }
        })
        apiMocks.getUsers.mockResolvedValue({
            data: {
                data: {
                    items: [
                        { id: 'u-1', username: 'alice' },
                        { id: 'u-2', username: 'bob' }
                    ]
                }
            }
        })
        apiMocks.getQuestionVersions.mockResolvedValue({
            data: {
                data: [
                    { id: 'v-1', version: 1, schema: { title: '年龄' } },
                    { id: 'v-2', version: 2, schema: { title: '年龄（新版）' } }
                ]
            }
        })
        apiMocks.createQuestionBank.mockResolvedValue({ data: { data: null } })
        apiMocks.updateQuestionBank.mockResolvedValue({ data: { data: null } })
        apiMocks.addQuestionBankItem.mockResolvedValue({ data: { data: null } })
        apiMocks.removeQuestionBankItem.mockResolvedValue({ data: { data: null } })
        apiMocks.shareQuestionBank.mockResolvedValue({ data: { data: null } })
        apiMocks.unshareQuestionBank.mockResolvedValue({ data: { data: null } })

        const wrapper = mount(QuestionBanks, {
            global: {
                stubs: {
                    'router-link': true
                }
            }
        })

        await flushPromises()
        await nextTick()

        expect(wrapper.text()).toContain('基础题库')
        expect(wrapper.text()).toContain('常用题目')

        const topInputs = wrapper.findAll('input')
        await topInputs[0].setValue('新题库')
        await topInputs[1].setValue('测试描述')
        const createButton = wrapper.findAll('button').find(button => button.text() === '创建')
        await createButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.createQuestionBank).toHaveBeenCalledWith({
            name: '新题库',
            description: '测试描述',
            visibility: 'private'
        })

        const bankHeader = wrapper.find('strong')
        expect(bankHeader.exists()).toBe(true)
        await bankHeader.trigger('click')
        await nextTick()

        expect(wrapper.text()).toContain('题库题目')
        expect(wrapper.text()).toContain('共享')

        const expandedInputs = wrapper.findAll('input')
        await expandedInputs[2].setValue('题库名称修改')
        await expandedInputs[3].setValue('题库说明修改')
        const selects = wrapper.findAll('select')
        await selects[4].setValue('u-2')
        const shareButton = wrapper.findAll('button').find(button => button.text() === '共享')
        await shareButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.shareQuestionBank).toHaveBeenCalledWith('bank-1', { targetUserId: 'u-2', permission: 'use' })

        const itemSelect = wrapper.findAll('select')[2]
        await itemSelect.setValue('q-1')
        await nextTick()
        const versionSelect = wrapper.findAll('select')[3]
        await versionSelect.setValue('v-2')
        const addButton = wrapper.findAll('button').find(button => button.text() === '添加题目')
        await addButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.addQuestionBankItem).toHaveBeenCalledWith('bank-1', { questionId: 'q-1', pinnedVersionId: 'v-2' })

        const updateButton = wrapper.findAll('button').find(button => button.text() === '更新')
        await updateButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.updateQuestionBank).toHaveBeenCalledWith('bank-1', expect.objectContaining({ name: '题库名称修改', description: '题库说明修改' }))
    })
})
