export function typeLabel(t) {
    return {
        SINGLE_CHOICE: '单选题',
        MULTIPLE_CHOICE: '多选题',
        TEXT: '文本填空',
        NUMBER: '数字填空'
    }[t] || t
}

export function buildValidation(question) {
    const validation = {}

    if (question.type === 'MULTIPLE_CHOICE') {
        if (question.validation?.minSelect) validation.minSelect = question.validation.minSelect
        if (question.validation?.maxSelect) validation.maxSelect = question.validation.maxSelect
    }

    if (question.type === 'TEXT') {
        if (question.validation?.minLength) validation.minLength = question.validation.minLength
        if (question.validation?.maxLength) validation.maxLength = question.validation.maxLength
    }

    if (question.type === 'NUMBER') {
        if (question.validation?.minVal !== null && question.validation?.minVal !== undefined && question.validation?.minVal !== '') {
            validation.minVal = question.validation.minVal
        }
        if (question.validation?.maxVal !== null && question.validation?.maxVal !== undefined && question.validation?.maxVal !== '') {
            validation.maxVal = question.validation.maxVal
        }
        if (question.integerOnly) validation.numberType = 'integer'
    }

    return validation
}

export function makeQuestionSnapshot(question) {
    const snapshot = {
        type: question.type,
        title: question.title,
        isRequired: question.isRequired,
        meta: question.meta || {}
    }

    if (question.type === 'SINGLE_CHOICE' || question.type === 'MULTIPLE_CHOICE') {
        snapshot.options = (question.options || [])
            .filter(option => (option.text || '').trim())
            .map(option => ({
                optionId: option.optionId,
                text: option.text,
                hasOtherInput: false
            }))
    }

    const validation = buildValidation(question)
    if (Object.keys(validation).length > 0) {
        snapshot.validation = validation
    }

    return snapshot
}

export function snapshotsEqual(left, right) {
    return JSON.stringify(left) === JSON.stringify(right)
}

export function buildCreateQuestionnairePayload(form) {
    return {
        title: form.title,
        description: form.description,
        settings: { ...(form.settings || {}) },
        questions: (form.questions || []).map((question, index) => {
            const snapshot = makeQuestionSnapshot(question)
            return {
                questionId: question.questionId,
                questionVersionId: question.questionVersionId,
                order: index,
                snapshot,
                type: question.type,
                title: question.title,
                isRequired: question.isRequired,
                options: (question.type === 'SINGLE_CHOICE' || question.type === 'MULTIPLE_CHOICE') ? snapshot.options : undefined,
                validation: snapshot.validation,
                meta: question.meta || {}
            }
        }),
        logicRules: (form.logicRules || [])
            .filter(rule => rule.conditionQuestionId && rule.targetQuestionId)
            .map(rule => ({
                conditionQuestionId: rule.conditionQuestionId,
                operator: rule.operator,
                conditionValue: rule.conditionValue,
                action: 'JUMP_TO',
                actionDetails: { targetQuestionId: rule.targetQuestionId }
            }))
    }
}
