package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/internal/service"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type createQuestionRequest struct {
	QuestionKey string                `json:"questionKey" binding:"required"`
	Schema      domain.QuestionSchema `json:"schema" binding:"required"`
	Tags        []string              `json:"tags"`
}

type createQuestionVersionRequest struct {
	BaseVersionID string                           `json:"baseVersionId"`
	ChangeType    domain.QuestionVersionChangeType `json:"changeType"`
	Note          string                           `json:"note"`
	Schema        domain.QuestionSchema            `json:"schema" binding:"required"`
}

type restoreQuestionVersionRequest struct {
	FromVersionID string `json:"fromVersionId" binding:"required"`
	Note          string `json:"note"`
}

type createQuestionBankRequest struct {
	Name        string                                `json:"name" binding:"required"`
	Description string                                `json:"description"`
	Visibility  domain.QuestionBankVisibility         `json:"visibility"`
	Items       []service.CreateQuestionBankItemInput `json:"items"`
}

type updateQuestionBankRequest struct {
	Name        string                        `json:"name" binding:"required"`
	Description string                        `json:"description"`
	Visibility  domain.QuestionBankVisibility `json:"visibility"`
}

type addQuestionBankItemRequest struct {
	QuestionID      string  `json:"questionId" binding:"required"`
	PinnedVersionID *string `json:"pinnedVersionId"`
	Order           int     `json:"order"`
}

type updateQuestionBankItemRequest struct {
	PinnedVersionID *string `json:"pinnedVersionId"`
	Order           *int    `json:"order"`
}

type shareQuestionBankRequest struct {
	TargetUserID string                        `json:"targetUserId" binding:"required"`
	Permission   domain.QuestionBankPermission `json:"permission"`
	ExpiresAt    *time.Time                    `json:"expiresAt"`
}

type createCrossTabRequest struct {
	RowQuestionID string `json:"rowQuestionId" binding:"required"`
	ColQuestionID string `json:"colQuestionId" binding:"required"`
	Filters       struct {
		DateRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"dateRange"`
		CompletionStatus string `json:"completionStatus"`
	} `json:"filters"`
}

func (h *Handler) CreateQuestion(c *gin.Context) {
	var req createQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	result, appErr := h.question.Create(c.Request.Context(), getRequiredUserID(c), service.CreateQuestionInput{
		QuestionKey: req.QuestionKey,
		Schema:      req.Schema,
		Tags:        req.Tags,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "创建成功", result)
}

func (h *Handler) CreateQuestionVersion(c *gin.Context) {
	var req createQuestionVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	result, appErr := h.question.CreateVersion(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.CreateQuestionVersionInput{
		BaseVersionID: req.BaseVersionID,
		ChangeType:    req.ChangeType,
		Note:          req.Note,
		Schema:        req.Schema,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "创建成功", result)
}

func (h *Handler) GetQuestionVersions(c *gin.Context) {
	items, appErr := h.question.ListVersions(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")))
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) RestoreQuestionVersion(c *gin.Context) {
	var req restoreQuestionVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	result, appErr := h.question.RestoreVersion(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.RestoreQuestionVersionInput{
		FromVersionID: req.FromVersionID,
		Note:          req.Note,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "创建成功", result)
}

func (h *Handler) GetQuestionUsages(c *gin.Context) {
	questionID := strings.TrimSpace(c.Param("id"))
	questionVersionID := strings.TrimSpace(c.Query("questionVersionId"))
	status := strings.TrimSpace(c.Query("status"))
	items, appErr := h.question.GetUsages(c.Request.Context(), questionID, questionVersionID, status)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) GetQuestionStats(c *gin.Context) {
	questionID := strings.TrimSpace(c.Param("id"))
	questionVersionID := strings.TrimSpace(c.Query("questionVersionId"))
	from := parseOptionalTime(c.Query("from"))
	to := parseOptionalTime(c.Query("to"))
	stats, appErr := h.question.GetStats(c.Request.Context(), questionID, service.QuestionStatsInput{
		QuestionVersionID: questionVersionID,
		From:              from,
		To:                to,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, stats)
}

func (h *Handler) CreateQuestionBank(c *gin.Context) {
	var req createQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	id, appErr := h.questionBank.Create(c.Request.Context(), getRequiredUserID(c), service.CreateQuestionBankInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Items:       req.Items,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "创建成功", gin.H{"id": id})
}

func (h *Handler) GetQuestionBanks(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	limit := parseInt(c.Query("limit"), 20)
	keyword := strings.TrimSpace(c.Query("keyword"))
	items, total, appErr := h.questionBank.List(c.Request.Context(), getRequiredUserID(c), domain.QuestionBankListFilter{Page: page, Limit: limit, Keyword: keyword})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) UpdateQuestionBank(c *gin.Context) {
	var req updateQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	bank, appErr := h.questionBank.UpdateBase(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.UpdateQuestionBankInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, bank)
}

func (h *Handler) AddQuestionBankItem(c *gin.Context) {
	var req addQuestionBankItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	items, appErr := h.questionBank.AddItem(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.AddQuestionBankItemInput{
		QuestionID:      req.QuestionID,
		PinnedVersionID: req.PinnedVersionID,
		Order:           req.Order,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) UpdateQuestionBankItem(c *gin.Context) {
	var req updateQuestionBankItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	items, appErr := h.questionBank.UpdateItem(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), strings.TrimSpace(c.Param("questionId")), service.UpdateQuestionBankItemInput{
		PinnedVersionID: req.PinnedVersionID,
		Order:           req.Order,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) RemoveQuestionBankItem(c *gin.Context) {
	items, appErr := h.questionBank.RemoveItem(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), strings.TrimSpace(c.Param("questionId")))
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *Handler) ShareQuestionBank(c *gin.Context) {
	var req shareQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	shares, appErr := h.questionBank.Share(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.ShareQuestionBankInput{
		TargetUserID: req.TargetUserID,
		Permission:   req.Permission,
		ExpiresAt:    req.ExpiresAt,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"sharedWith": shares})
}

func (h *Handler) UnshareQuestionBank(c *gin.Context) {
	shares, appErr := h.questionBank.Unshare(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), strings.TrimSpace(c.Param("targetUserId")))
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"sharedWith": shares})
}

func (h *Handler) CreateCrossTabReport(c *gin.Context) {
	var req createCrossTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}
	start, ok := parseDateTimeInput(req.Filters.DateRange.Start, false)
	if !ok {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"filters.dateRange.start": "时间格式错误，仅支持RFC3339"})
		return
	}
	end, ok := parseDateTimeInput(req.Filters.DateRange.End, true)
	if !ok {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"filters.dateRange.end": "时间格式错误，仅支持RFC3339"})
		return
	}
	report, appErr := h.questionnaire.BuildCrossTab(c.Request.Context(), getRequiredUserID(c), strings.TrimSpace(c.Param("id")), service.CrossTabInput{
		RowQuestionID: req.RowQuestionID,
		ColQuestionID: req.ColQuestionID,
		Filters: service.CrossTabFilters{
			DateRange:        service.CrossTabDateRange{Start: start, End: end},
			CompletionStatus: req.Filters.CompletionStatus,
		},
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, report)
}

func parseOptionalTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		t = t.UTC()
		return &t
	}
	return nil
}

func parseDateTimeInput(raw string, endOfDay bool) (*time.Time, bool) {
	_ = endOfDay
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		t = t.UTC()
		return &t, true
	}
	return nil, false
}
