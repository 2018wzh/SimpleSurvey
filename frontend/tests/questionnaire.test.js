import { describe, expect, it } from 'vitest'
import {
    buildCreateQuestionnairePayload,
    buildValidation,
    makeQuestionSnapshot,
    snapshotsEqual,
    typeLabel
} from '../src/utils/questionnaire'

describe('questionnaire utilities', () => {
    it('renders the same labels used in the create/fill pages', () => {
        expect(typeLabel('SINGLE_CHOICE')).toBe('单选题')
        expect(typeLabel('MULTIPLE_CHOICE')).toBe('多选题')
        expect(typeLabel('TEXT')).toBe('文本填空')
        expect(typeLabel('NUMBER')).toBe('数字填空')
        expect(typeLabel('UNKNOWN')).toBe('UNKNOWN')
    })

    it('builds snapshots with filtered options and validation fields', () => {
        const question = {
            type: 'MULTIPLE_CHOICE',
            title: '你喜欢的水果',
            isRequired: true,
            options: [
                { optionId: 'o1', text: '苹果' },
                { optionId: 'o2', text: '  ' },
                { optionId: 'o3', text: '香蕉' }
            ],
            validation: { minSelect: 2, maxSelect: 3 },
            meta: { source: 'demo' }
        }

        expect(buildValidation(question)).toEqual({ minSelect: 2, maxSelect: 3 })
        expect(makeQuestionSnapshot(question)).toEqual({
            type: 'MULTIPLE_CHOICE',
            title: '你喜欢的水果',
            isRequired: true,
            meta: { source: 'demo' },
            options: [
                { optionId: 'o1', text: '苹果', hasOtherInput: false },
                { optionId: 'o3', text: '香蕉', hasOtherInput: false }
            ],
            validation: { minSelect: 2, maxSelect: 3 }
        })
    })

    it('builds the questionnaire payload with order, snapshots and valid jump rules', () => {
        const form = {
            title: '满意度调查',
            description: '用于验证创建问卷',
            settings: { allowAnonymous: true },
            questions: [
                {
                    questionId: 'q1',
                    questionVersionId: 'v1',
                    type: 'SINGLE_CHOICE',
                    title: '是否满意',
                    isRequired: true,
                    options: [
                        { optionId: 'yes', text: '满意' },
                        { optionId: 'no', text: '不满意' }
                    ],
                    validation: {},
                    meta: { group: 'base' }
                },
                {
                    questionId: 'q2',
                    questionVersionId: 'v2',
                    type: 'NUMBER',
                    title: '年龄',
                    isRequired: false,
                    options: [],
                    validation: { minVal: 0, maxVal: 120 },
                    meta: {}
                }
            ],
            logicRules: [
                { conditionQuestionId: 'q1', operator: 'EQUALS', conditionValue: 'yes', targetQuestionId: 'q2' },
                { conditionQuestionId: '', operator: 'EQUALS', conditionValue: 'x', targetQuestionId: 'q3' }
            ]
        }

        expect(snapshotsEqual(makeQuestionSnapshot(form.questions[0]), makeQuestionSnapshot(form.questions[0]))).toBe(true)

        expect(buildCreateQuestionnairePayload(form)).toEqual({
            title: '满意度调查',
            description: '用于验证创建问卷',
            settings: { allowAnonymous: true },
            questions: [
                {
                    questionId: 'q1',
                    questionVersionId: 'v1',
                    order: 0,
                    snapshot: {
                        type: 'SINGLE_CHOICE',
                        title: '是否满意',
                        isRequired: true,
                        meta: { group: 'base' },
                        options: [
                            { optionId: 'yes', text: '满意', hasOtherInput: false },
                            { optionId: 'no', text: '不满意', hasOtherInput: false }
                        ]
                    },
                    type: 'SINGLE_CHOICE',
                    title: '是否满意',
                    isRequired: true,
                    options: [
                        { optionId: 'yes', text: '满意', hasOtherInput: false },
                        { optionId: 'no', text: '不满意', hasOtherInput: false }
                    ],
                    validation: undefined,
                    meta: { group: 'base' }
                },
                {
                    questionId: 'q2',
                    questionVersionId: 'v2',
                    order: 1,
                    snapshot: {
                        type: 'NUMBER',
                        title: '年龄',
                        isRequired: false,
                        meta: {},
                        validation: { minVal: 0, maxVal: 120 }
                    },
                    type: 'NUMBER',
                    title: '年龄',
                    isRequired: false,
                    options: undefined,
                    validation: { minVal: 0, maxVal: 120 },
                    meta: {}
                }
            ],
            logicRules: [
                {
                    conditionQuestionId: 'q1',
                    operator: 'EQUALS',
                    conditionValue: 'yes',
                    action: 'JUMP_TO',
                    actionDetails: { targetQuestionId: 'q2' }
                }
            ]
        })
    })
})
