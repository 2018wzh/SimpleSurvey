import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import FillSurvey from '../src/views/FillSurvey.vue'

const apiMocks = vi.hoisted(() => ({
    getSurvey: vi.fn(),
    submitResponse: vi.fn()
}))

const routerMocks = vi.hoisted(() => ({
    route: { params: { id: 'survey-1' } },
    push: vi.fn()
}))

vi.mock('../src/api/index.js', () => ({
    default: apiMocks
}))

vi.mock('vue-router', () => ({
    useRoute: () => routerMocks.route,
    useRouter: () => ({ push: routerMocks.push })
}))

function flushPromises() {
    return new Promise(resolve => setTimeout(resolve, 0))
}

describe('FillSurvey.vue', () => {
    it('validates answers, jumps according to logic, and submits only visited answers', async () => {
        apiMocks.getSurvey.mockResolvedValue({
            data: {
                data: {
                    title: '路由与校验测试问卷',
                    description: '用于验证前端填写流程',
                    settings: { allowAnonymous: true },
                    questions: [
                        {
                            questionId: 'q1',
                            questionVersionId: 'v1',
                            type: 'SINGLE_CHOICE',
                            title: '是否跳转',
                            isRequired: true,
                            options: [
                                { optionId: 'go', text: '跳到最后一题' },
                                { optionId: 'stay', text: '继续顺序填写' }
                            ],
                            validation: {}
                        },
                        {
                            questionId: 'q2',
                            questionVersionId: 'v2',
                            type: 'TEXT',
                            title: '中间题',
                            isRequired: false,
                            validation: { minLength: 2 }
                        },
                        {
                            questionId: 'q3',
                            questionVersionId: 'v3',
                            type: 'NUMBER',
                            title: '年龄',
                            isRequired: true,
                            validation: { minVal: 18, maxVal: 99, numberType: 'integer' }
                        }
                    ],
                    logicRules: [
                        {
                            conditionQuestionId: 'q1',
                            operator: 'EQUALS',
                            conditionValue: 'go',
                            action: 'JUMP_TO',
                            actionDetails: { targetQuestionId: 'q3' }
                        }
                    ]
                }
            }
        })
        apiMocks.submitResponse.mockResolvedValue({ data: { data: null } })

        const wrapper = mount(FillSurvey, {
            global: {
                stubs: {
                    'router-link': true
                }
            }
        })

        await flushPromises()
        await nextTick()

        expect(wrapper.text()).toContain('路由与校验测试问卷')
        expect(wrapper.text()).toContain('是否跳转')

        await wrapper.find('button').trigger('click')
        await nextTick()
        expect(wrapper.text()).toContain('请选择一个选项')

        const radios = wrapper.findAll('input[type="radio"]')
        await radios[0].setValue(true)
        await wrapper.find('button').trigger('click')
        await flushPromises()
        await nextTick()

        expect(wrapper.text()).toContain('年龄')
        expect(wrapper.text()).not.toContain('中间题')

        const numberInput = wrapper.find('input[type="number"]')
        await numberInput.setValue('20')
        const submitButton = wrapper.findAll('button').find(button => button.text() === '提交')
        expect(submitButton).toBeTruthy()
        await submitButton.trigger('click')
        await flushPromises()
        await nextTick()

        expect(apiMocks.submitResponse).toHaveBeenCalledTimes(1)
        expect(apiMocks.submitResponse).toHaveBeenCalledWith('survey-1', {
            isAnonymous: false,
            answers: [
                { questionId: 'q1', questionVersionId: 'v1', value: 'go' },
                { questionId: 'q3', questionVersionId: 'v3', value: 20 }
            ],
            statistics: {
                completionTime: expect.any(Number)
            }
        })
        expect(wrapper.text()).toContain('提交成功！')
    })
})
