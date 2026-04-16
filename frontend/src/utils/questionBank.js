export function buildQuestionSchema(draft) {
    const schema = {
        type: draft.type,
        title: draft.title,
        isRequired: draft.isRequired,
        meta: draft.meta || {}
    }

    if (draft.type === 'SINGLE_CHOICE' || draft.type === 'MULTIPLE_CHOICE') {
        const options = (draft.options || [])
            .filter(option => (option.text || '').trim())
            .map(option => ({ optionId: option.optionId, text: option.text }))

        if (options.length < 2) {
            throw new Error('选择题至少需要2个选项')
        }

        schema.options = options
    }

    const validation = draft.validation || {}
    const normalizedValidation = {}

    if (draft.type === 'MULTIPLE_CHOICE') {
        if (validation.minSelect) normalizedValidation.minSelect = validation.minSelect
        if (validation.maxSelect) normalizedValidation.maxSelect = validation.maxSelect
    }

    if (draft.type === 'TEXT') {
        if (validation.minLength) normalizedValidation.minLength = validation.minLength
        if (validation.maxLength) normalizedValidation.maxLength = validation.maxLength
    }

    if (draft.type === 'NUMBER') {
        if (validation.minVal !== null && validation.minVal !== undefined && validation.minVal !== '') {
            normalizedValidation.minVal = validation.minVal
        }
        if (validation.maxVal !== null && validation.maxVal !== undefined && validation.maxVal !== '') {
            normalizedValidation.maxVal = validation.maxVal
        }
        if (draft.integerOnly || validation.numberType === 'integer') {
            normalizedValidation.numberType = 'integer'
        }
    }

    if (Object.keys(normalizedValidation).length > 0) {
        schema.validation = normalizedValidation
    }

    return schema
}
