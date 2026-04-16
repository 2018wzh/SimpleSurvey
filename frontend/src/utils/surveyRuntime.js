export function validateQuestionAnswer(question, value) {
    const validation = question?.validation || {}

    if (question.isRequired) {
        if (question.type === 'SINGLE_CHOICE' && !value) return '请选择一个选项'
        if (question.type === 'MULTIPLE_CHOICE' && (!value || value.length === 0)) return '请至少选择一个选项'
        if (question.type === 'TEXT' && (!value || !value.trim())) return '请填写内容'
        if (question.type === 'NUMBER' && (value === null || value === '' || value === undefined)) return '请填写数字'
    }

    if (question.type === 'MULTIPLE_CHOICE' && value && value.length > 0) {
        if (validation.minSelect && value.length < validation.minSelect) return `至少选择 ${validation.minSelect} 个选项`
        if (validation.maxSelect && value.length > validation.maxSelect) return `最多选择 ${validation.maxSelect} 个选项`
    }

    if (question.type === 'TEXT' && value) {
        if (validation.minLength && value.length < validation.minLength) return `最少输入 ${validation.minLength} 个字`
        if (validation.maxLength && value.length > validation.maxLength) return `最多输入 ${validation.maxLength} 个字`
    }

    if (question.type === 'NUMBER' && value !== null && value !== '' && value !== undefined) {
        const numberValue = Number(value)
        if (Number.isNaN(numberValue)) return '请输入有效数字'
        if (validation.numberType === 'integer' && !Number.isInteger(numberValue)) return '必须为整数'
        if (validation.minVal != null && numberValue < validation.minVal) return `不能小于 ${validation.minVal}`
        if (validation.maxVal != null && numberValue > validation.maxVal) return `不能大于 ${validation.maxVal}`
    }

    return true
}

export function getNextQuestionIndex({ currentIndex, questions, answers, logicRules }) {
    const currentQuestion = questions[currentIndex]
    if (!currentQuestion) return null

    const value = answers[currentQuestion.questionId]
    const rules = logicRules || []

    for (const rule of rules) {
        if (rule.conditionQuestionId !== currentQuestion.questionId) continue
        if (rule.action !== 'JUMP_TO') continue

        let matched = false
        const ruleValue = rule.conditionValue

        if (rule.operator === 'EQUALS') {
            if (currentQuestion.type === 'SINGLE_CHOICE') matched = value === ruleValue
            else if (currentQuestion.type === 'NUMBER') matched = Number(value) === Number(ruleValue)
            else matched = value === ruleValue
        } else if (rule.operator === 'CONTAINS') {
            if (Array.isArray(value)) matched = value.includes(ruleValue)
            else if (typeof value === 'string') matched = value.includes(ruleValue)
        } else if (rule.operator === 'GREATER_THAN') {
            if (currentQuestion.type === 'NUMBER' && value !== null && value !== '') matched = Number(value) > Number(ruleValue)
        } else if (rule.operator === 'LESS_THAN') {
            if (currentQuestion.type === 'NUMBER' && value !== null && value !== '') matched = Number(value) < Number(ruleValue)
        }

        if (matched && rule.actionDetails && rule.actionDetails.targetQuestionId) {
            const targetIndex = questions.findIndex(question => question.questionId === rule.actionDetails.targetQuestionId)
            if (targetIndex >= 0) return targetIndex
        }
    }

    const nextIndex = currentIndex + 1
    return nextIndex < questions.length ? nextIndex : null
}

export function buildSurveyResponsePayload({ questions = [], answers = {}, visitedOrder = [], currentIndex = 0, isAnonymous = false, startTime = Date.now() }) {
    const visitedIndices = [...new Set([...visitedOrder, currentIndex])]
    const answerList = []

    for (const index of visitedIndices) {
        const question = questions[index]
        if (!question) continue

        let value = answers[question.questionId]
        if (question.type === 'NUMBER' && value !== null && value !== '' && value !== undefined) {
            value = Number(value)
        }

        if (value === '' || value === null || value === undefined) continue
        if (Array.isArray(value) && value.length === 0) continue

        answerList.push({
            questionId: question.questionId,
            questionVersionId: question.questionVersionId,
            value
        })
    }

    return {
        isAnonymous,
        answers: answerList,
        statistics: {
            completionTime: Math.round((Date.now() - startTime) / 1000)
        }
    }
}
