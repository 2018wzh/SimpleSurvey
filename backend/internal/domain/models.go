package domain

import "time"

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "SINGLE_CHOICE"
	QuestionTypeMultipleChoice QuestionType = "MULTIPLE_CHOICE"
	QuestionTypeText           QuestionType = "TEXT"
	QuestionTypeNumber         QuestionType = "NUMBER"
)

type QuestionnaireStatus string

const (
	QuestionnaireStatusDraft     QuestionnaireStatus = "draft"
	QuestionnaireStatusPublished QuestionnaireStatus = "published"
	QuestionnaireStatusClosed    QuestionnaireStatus = "closed"
)

type LogicOperator string

const (
	LogicOperatorEquals      LogicOperator = "EQUALS"
	LogicOperatorContains    LogicOperator = "CONTAINS"
	LogicOperatorGreaterThan LogicOperator = "GREATER_THAN"
	LogicOperatorLessThan    LogicOperator = "LESS_THAN"
)

type LogicAction string

const (
	LogicActionJumpTo LogicAction = "JUMP_TO"
)

type QuestionVersionChangeType string

const (
	QuestionVersionChangeTypeCreate  QuestionVersionChangeType = "create"
	QuestionVersionChangeTypeEdit    QuestionVersionChangeType = "edit"
	QuestionVersionChangeTypeRestore QuestionVersionChangeType = "restore"
	QuestionVersionChangeTypeFork    QuestionVersionChangeType = "fork"
)

type QuestionBankVisibility string

const (
	QuestionBankVisibilityPrivate QuestionBankVisibility = "private"
	QuestionBankVisibilityTeam    QuestionBankVisibility = "team"
)

type QuestionBankPermission string

const (
	QuestionBankPermissionUse    QuestionBankPermission = "use"
	QuestionBankPermissionManage QuestionBankPermission = "manage"
)

type User struct {
	ID        string                 `json:"id" bson:"_id,omitempty"`
	Username  string                 `json:"username" bson:"username"`
	Password  string                 `json:"-" bson:"password"`
	CreatedAt time.Time              `json:"createdAt" bson:"createdAt"`
	Role      UserRole               `json:"role" bson:"role"`
	Status    UserStatus             `json:"status" bson:"status"`
	MetaInfo  map[string]interface{} `json:"metaInfo,omitempty" bson:"meta_info,omitempty"`
}

type QuestionnaireSettings struct {
	AllowAnonymous bool   `json:"allowAnonymous" bson:"allowAnonymous"`
	DuplicateCheck string `json:"duplicateCheck,omitempty" bson:"duplicateCheck,omitempty"`
	ThemeColor     string `json:"themeColor,omitempty" bson:"themeColor,omitempty"`
}

type QuestionOption struct {
	OptionID      string `json:"optionId" bson:"optionId"`
	Text          string `json:"text" bson:"text"`
	HasOtherInput bool   `json:"hasOtherInput,omitempty" bson:"hasOtherInput,omitempty"`
}

type QuestionValidation struct {
	MinSelect  *int     `json:"minSelect,omitempty" bson:"minSelect,omitempty"`
	MaxSelect  *int     `json:"maxSelect,omitempty" bson:"maxSelect,omitempty"`
	MinLength  *int     `json:"minLength,omitempty" bson:"minLength,omitempty"`
	MaxLength  *int     `json:"maxLength,omitempty" bson:"maxLength,omitempty"`
	NumberType string   `json:"numberType,omitempty" bson:"numberType,omitempty"`
	MinVal     *float64 `json:"minVal,omitempty" bson:"minVal,omitempty"`
	MaxVal     *float64 `json:"maxVal,omitempty" bson:"maxVal,omitempty"`
}

type QuestionSchema struct {
	Type       QuestionType           `json:"type" bson:"type"`
	Title      string                 `json:"title" bson:"title"`
	IsRequired bool                   `json:"isRequired" bson:"isRequired"`
	Meta       map[string]interface{} `json:"meta,omitempty" bson:"meta,omitempty"`
	Options    []QuestionOption       `json:"options,omitempty" bson:"options,omitempty"`
	Validation QuestionValidation     `json:"validation,omitempty" bson:"validation,omitempty"`
}

type QuestionEntity struct {
	ID               string    `json:"id" bson:"_id,omitempty"`
	QuestionKey      string    `json:"questionKey" bson:"questionKey"`
	OwnerID          string    `json:"ownerId" bson:"ownerId"`
	CurrentVersion   int       `json:"currentVersion" bson:"currentVersion"`
	CurrentVersionID string    `json:"currentVersionId" bson:"currentVersionId"`
	Tags             []string  `json:"tags,omitempty" bson:"tags,omitempty"`
	CreatedAt        time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt" bson:"updatedAt"`
	IsArchived       bool      `json:"isArchived" bson:"isArchived"`
}

type QuestionVersion struct {
	ID              string                    `json:"id" bson:"_id,omitempty"`
	QuestionID      string                    `json:"questionId" bson:"questionId"`
	Version         int                       `json:"version" bson:"version"`
	ParentVersionID *string                   `json:"parentVersionId,omitempty" bson:"parentVersionId,omitempty"`
	ChangeType      QuestionVersionChangeType `json:"changeType" bson:"changeType"`
	Schema          QuestionSchema            `json:"schema" bson:"schema"`
	CreatedBy       string                    `json:"createdBy" bson:"createdBy"`
	CreatedAt       time.Time                 `json:"createdAt" bson:"createdAt"`
	Note            string                    `json:"note,omitempty" bson:"note,omitempty"`
}

type QuestionBankShare struct {
	UserID     string                 `json:"userId" bson:"userId"`
	Permission QuestionBankPermission `json:"permission" bson:"permission"`
	GrantedBy  string                 `json:"grantedBy" bson:"grantedBy"`
	GrantedAt  time.Time              `json:"grantedAt" bson:"grantedAt"`
	ExpiresAt  *time.Time             `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
}

type QuestionBankItem struct {
	QuestionID      string    `json:"questionId" bson:"questionId"`
	PinnedVersionID *string   `json:"pinnedVersionId,omitempty" bson:"pinnedVersionId,omitempty"`
	AddedBy         string    `json:"addedBy" bson:"addedBy"`
	AddedAt         time.Time `json:"addedAt" bson:"addedAt"`
	Order           int       `json:"order" bson:"order"`
}

type QuestionBank struct {
	ID          string                 `json:"id" bson:"_id,omitempty"`
	Name        string                 `json:"name" bson:"name"`
	OwnerID     string                 `json:"ownerId" bson:"ownerId"`
	Description string                 `json:"description,omitempty" bson:"description,omitempty"`
	Visibility  QuestionBankVisibility `json:"visibility" bson:"visibility"`
	SharedWith  []QuestionBankShare    `json:"sharedWith,omitempty" bson:"sharedWith,omitempty"`
	Items       []QuestionBankItem     `json:"items,omitempty" bson:"items,omitempty"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
}

type QuestionBankListFilter struct {
	Page    int
	Limit   int
	Keyword string
}

type QuestionUsage struct {
	QuestionnaireID    string              `json:"questionnaireId"`
	QuestionnaireTitle string              `json:"questionnaireTitle"`
	Status             QuestionnaireStatus `json:"status"`
	QuestionVersionID  string              `json:"questionVersionId"`
}

type QuestionCrossStats struct {
	QuestionID        string         `json:"questionId"`
	QuestionVersionID string         `json:"questionVersionId,omitempty"`
	Type              QuestionType   `json:"type"`
	TotalAnswered     int            `json:"totalAnswered"`
	OptionCounts      map[string]int `json:"optionCounts,omitempty"`
	AverageValue      *float64       `json:"averageValue,omitempty"`
	TextAnswers       []string       `json:"textAnswers,omitempty"`
}

type CrossTabCell struct {
	Row        string  `json:"row"`
	Col        string  `json:"col"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type CrossTabReport struct {
	RowQuestionID string         `json:"rowQuestionId"`
	ColQuestionID string         `json:"colQuestionId"`
	TotalSample   int            `json:"totalSample"`
	Matrix        []CrossTabCell `json:"matrix"`
}

type Question struct {
	QuestionID        string                 `json:"questionId" bson:"questionId"`
	QuestionVersionID string                 `json:"questionVersionId" bson:"questionVersionId"`
	Order             int                    `json:"order,omitempty" bson:"order,omitempty"`
	Snapshot          *QuestionSchema        `json:"snapshot,omitempty" bson:"snapshot,omitempty"`
	Type              QuestionType           `json:"type" bson:"type"`
	Title             string                 `json:"title" bson:"title"`
	IsRequired        bool                   `json:"isRequired" bson:"isRequired"`
	Meta              map[string]interface{} `json:"meta,omitempty" bson:"meta,omitempty"`
	Options           []QuestionOption       `json:"options,omitempty" bson:"options,omitempty"`
	Validation        QuestionValidation     `json:"validation,omitempty" bson:"validation,omitempty"`
}

type LogicRule struct {
	ConditionQuestionID string                 `json:"conditionQuestionId" bson:"conditionQuestionId"`
	Operator            LogicOperator          `json:"operator" bson:"operator"`
	ConditionValue      interface{}            `json:"conditionValue" bson:"conditionValue"`
	Action              LogicAction            `json:"action" bson:"action"`
	ActionDetails       map[string]interface{} `json:"actionDetails" bson:"actionDetails"`
}

type Questionnaire struct {
	ID          string                `json:"id" bson:"_id,omitempty"`
	CreatorID   string                `json:"creatorId" bson:"creatorId"`
	Title       string                `json:"title" bson:"title"`
	Description string                `json:"description" bson:"description"`
	Settings    QuestionnaireSettings `json:"settings" bson:"settings"`
	Questions   []Question            `json:"questions" bson:"questions"`
	LogicRules  []LogicRule           `json:"logicRules,omitempty" bson:"logicRules,omitempty"`
	Status      QuestionnaireStatus   `json:"status" bson:"status"`
	Deadline    *time.Time            `json:"deadline,omitempty" bson:"deadline,omitempty"`
	CreatedAt   time.Time             `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt" bson:"updatedAt"`
	IsDeleted   bool                  `json:"isDeleted" bson:"isDeleted"`
}

type Answer struct {
	QuestionID        string      `json:"questionId" bson:"questionId"`
	QuestionVersionID string      `json:"questionVersionId" bson:"questionVersionId"`
	Value             interface{} `json:"value" bson:"value"`
}

type ResponseStatistics struct {
	CompletionTime int    `json:"completionTime,omitempty" bson:"completionTime,omitempty"`
	IPAddress      string `json:"ipAddress,omitempty" bson:"ipAddress,omitempty"`
}

type SurveyResponse struct {
	ID              string             `json:"id" bson:"_id,omitempty"`
	QuestionnaireID string             `json:"questionnaireId" bson:"questionnaireId"`
	IsAnonymous     bool               `json:"isAnonymous" bson:"isAnonymous"`
	UserID          *string            `json:"userId,omitempty" bson:"userId,omitempty"`
	Answers         []Answer           `json:"answers" bson:"answers"`
	SubmittedAt     time.Time          `json:"submittedAt" bson:"submittedAt"`
	Statistics      ResponseStatistics `json:"statistics,omitempty" bson:"statistics,omitempty"`
}

type QuestionnaireListFilter struct {
	Page   int
	Limit  int
	Status string
	SortBy string
}

type QuestionnaireAdminListFilter struct {
	Page      int
	Limit     int
	Status    string
	SortBy    string
	CreatorID string
}

type UserListFilter struct {
	Page    int
	Limit   int
	Status  string
	Role    string
	Keyword string
}

type ResponseListFilter struct {
	Page              int
	Limit             int
	QuestionID        string
	QuestionVersionID string
}

type QuestionStat struct {
	QuestionID        string         `json:"questionId"`
	QuestionVersionID string         `json:"questionVersionId"`
	Type              QuestionType   `json:"type"`
	TotalAnswered     int            `json:"totalAnswered"`
	OptionCounts      map[string]int `json:"optionCounts,omitempty"`
	AverageValue      *float64       `json:"averageValue,omitempty"`
	TextAnswers       []string       `json:"textAnswers,omitempty"`
}

type QuestionnaireStats struct {
	TotalResponses int64          `json:"totalResponses"`
	QuestionStats  []QuestionStat `json:"questionStats"`
}
