import { describe, expect, it } from 'vitest'
import {
    buildSurveyResponsePayload,
    getNextQuestionIndex,
    validateQuestionAnswer
} from '../src/utils/surveyRuntime'

describe('survey runtime utilities', () => {
    const questions = [
        {
            questionId: 'q1',
            questionVersionId: 'v1',
            type: 'SINGLE_CHOICE',
            title: '是否继续',
            isRequired: true,
            options: [
                { optionId: 'a', text: '继续' },
                { optionId: 'b', text: '跳过' }
            ],
            validation: {}
        },
        {
            questionId: 'q2',
            questionVersionId: 'v2',
            type: 'TEXT',
            title: '中间题',
            isRequired: false,
            validation: { minLength: 2, maxLength: 4 }
        },
        {
            questionId: 'q3',
            questionVersionId: 'v3',
            type: 'NUMBER',
            title: '年龄',
            isRequired: true,
            validation: { minVal: 18, maxVal: 60, numberType: 'integer' }
        },
        {
            questionId: 'q4',
            questionVersionId: 'v4',
            type: 'MULTIPLE_CHOICE',
            title: '喜欢的水果',
            isRequired: false,
            validation: { minSelect: 1, maxSelect: 2 }
        }
    ]

    it('validates the same rules used when filling a survey', () => {
        expect(validateQuestionAnswer(questions[0], '')).toBe('请选择一个选项')
        expect(validateQuestionAnswer(questions[1], 'a')).toBe('最少输入 2 个字')
        expect(validateQuestionAnswer(questions[1], 'abcdx')).toBe('最多输入 4 个字')
        expect(validateQuestionAnswer(questions[2], '')).toBe('请填写数字')
        expect(validateQuestionAnswer(questions[2], '17')).toBe('不能小于 18')
        expect(validateQuestionAnswer(questions[2], '18.5')).toBe('必须为整数')
        expect(validateQuestionAnswer(questions[2], '20')).toBe(true)
        expect(validateQuestionAnswer(questions[3], [])).toBe(true)
        expect(validateQuestionAnswer(questions[3], ['x'])).toBe(true)
    })

    it('supports sequential navigation and jump rules', () => {
        const answers = {
            q1: 'a',
            q3: 20,
            q4: ['x']
        }
        const logicRules = [
            {
                conditionQuestionId: 'q1',
                operator: 'EQUALS',
                conditionValue: 'a',
                action: 'JUMP_TO',
                actionDetails: { targetQuestionId: 'q3' }
            },
            {
                conditionQuestionId: 'q3',
                operator: 'GREATER_THAN',
                conditionValue: 18,
                action: 'JUMP_TO',
                actionDetails: { targetQuestionId: 'q4' }
            }
        ]

        expect(getNextQuestionIndex({ currentIndex: 0, questions, answers, logicRules })).toBe(2)
        expect(getNextQuestionIndex({ currentIndex: 2, questions, answers, logicRules })).toBe(3)
        expect(getNextQuestionIndex({ currentIndex: 3, questions, answers, logicRules })).toBe(null)
    })

    it('builds the response payload from visited questions only', () => {
        const payload = buildSurveyResponsePayload({
            questions,
            answers: {
                q1: 'a',
                q2: 'should-not-appear',
                q3: '20',
                q4: []
            },
            visitedOrder: [0],
            currentIndex: 2,
            isAnonymous: true,
            startTime: 0
        })

        expect(payload).toEqual({
            isAnonymous: true,
            answers: [
                { questionId: 'q1', questionVersionId: 'v1', value: 'a' },
                { questionId: 'q3', questionVersionId: 'v3', value: 20 }
            ],
            statistics: {
                completionTime: expect.any(Number)
            }
        })
    })
})
