import { describe, expect, it } from 'vitest'
import { buildQuestionSchema } from '../src/utils/questionBank'

describe('question bank schema builder', () => {
    it('builds question schemas for reusable questions and versions', () => {
        expect(
            buildQuestionSchema({
                type: 'SINGLE_CHOICE',
                title: '你的性别',
                isRequired: true,
                options: [
                    { optionId: 'm', text: '男' },
                    { optionId: 'f', text: '女' },
                    { optionId: 'empty', text: '  ' }
                ],
                validation: {},
                meta: { tag: 'demo' }
            })
        ).toEqual({
            type: 'SINGLE_CHOICE',
            title: '你的性别',
            isRequired: true,
            meta: { tag: 'demo' },
            options: [
                { optionId: 'm', text: '男' },
                { optionId: 'f', text: '女' }
            ]
        })

        expect(
            buildQuestionSchema({
                type: 'NUMBER',
                title: '年龄',
                isRequired: false,
                options: [],
                integerOnly: true,
                validation: { minVal: 0, maxVal: 120 }
            })
        ).toEqual({
            type: 'NUMBER',
            title: '年龄',
            isRequired: false,
            meta: {},
            validation: { minVal: 0, maxVal: 120, numberType: 'integer' }
        })
    })

    it('rejects choice questions with fewer than two options', () => {
        expect(() =>
            buildQuestionSchema({
                type: 'MULTIPLE_CHOICE',
                title: '水果',
                isRequired: false,
                options: [{ optionId: 'only', text: '苹果' }],
                validation: {}
            })
        ).toThrow('选择题至少需要2个选项')
    })

    it('keeps text validation limits for questionnaire authoring', () => {
        expect(
            buildQuestionSchema({
                type: 'TEXT',
                title: '一句话介绍自己',
                isRequired: false,
                options: [],
                validation: { minLength: 2, maxLength: 20 }
            })
        ).toEqual({
            type: 'TEXT',
            title: '一句话介绍自己',
            isRequired: false,
            meta: {},
            validation: { minLength: 2, maxLength: 20 }
        })
    })
})
