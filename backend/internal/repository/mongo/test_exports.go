package mongo

import "github.com/2018wzh/SimpleSurvey/backend/internal/domain"

func RoundTripQuestionForTest(entity domain.QuestionEntity) (domain.QuestionEntity, error) {
	doc, err := toQuestionDoc(entity)
	if err != nil {
		return domain.QuestionEntity{}, err
	}
	return toDomainQuestion(doc), nil
}

func ToQuestionDocMetaForTest(entity domain.QuestionEntity) (string, int, error) {
	doc, err := toQuestionDoc(entity)
	if err != nil {
		return "", 0, err
	}
	return doc.QuestionKey, doc.CurrentVersion, nil
}

func RoundTripQuestionVersionForTest(version domain.QuestionVersion) (domain.QuestionVersion, error) {
	doc, err := toQuestionVersionDoc(version)
	if err != nil {
		return domain.QuestionVersion{}, err
	}
	return toDomainQuestionVersion(doc), nil
}

func ValidateToQuestionDocForTest(entity domain.QuestionEntity) error {
	_, err := toQuestionDoc(entity)
	return err
}

func ValidateToQuestionVersionDocForTest(version domain.QuestionVersion) error {
	_, err := toQuestionVersionDoc(version)
	return err
}

func RoundTripQuestionBankForTest(bank domain.QuestionBank) (domain.QuestionBank, error) {
	doc, err := toQuestionBankDoc(bank)
	if err != nil {
		return domain.QuestionBank{}, err
	}
	return toDomainQuestionBank(doc), nil
}

func ToQuestionBankDocMetaForTest(bank domain.QuestionBank) (string, int, int, error) {
	doc, err := toQuestionBankDoc(bank)
	if err != nil {
		return "", 0, 0, err
	}
	return doc.Name, len(doc.Items), len(doc.SharedWith), nil
}

func ValidateQuestionBankShareDocsForTest(shares []domain.QuestionBankShare) error {
	_, err := toQuestionBankShareDocs(shares)
	return err
}

func ValidateQuestionBankItemDocsForTest(items []domain.QuestionBankItem) error {
	_, err := toQuestionBankItemDocs(items)
	return err
}
