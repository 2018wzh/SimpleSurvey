package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2018wzh/SimpleSurvey/backend/internal/domain"
	"github.com/2018wzh/SimpleSurvey/backend/internal/service"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/apperror"
	"github.com/2018wzh/SimpleSurvey/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	identity      *service.IdentityService
	questionnaire *service.QuestionnaireService
	admin         *service.AdminService
}

func NewHandler(identity *service.IdentityService, questionnaire *service.QuestionnaireService, admin *service.AdminService) *Handler {
	return &Handler{identity: identity, questionnaire: questionnaire, admin: admin}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type createQuestionnaireRequest struct {
	Title       string                       `json:"title" binding:"required"`
	Description string                       `json:"description"`
	Settings    domain.QuestionnaireSettings `json:"settings"`
	Questions   []domain.Question            `json:"questions" binding:"required,min=1"`
	LogicRules  []domain.LogicRule           `json:"logicRules"`
}

type updateStatusRequest struct {
	Status   domain.QuestionnaireStatus `json:"status" binding:"required"`
	Deadline *time.Time                 `json:"deadline"`
}

type submitResponseRequest struct {
	IsAnonymous bool            `json:"isAnonymous"`
	Answers     []domain.Answer `json:"answers" binding:"required,min=1"`
	Statistics  struct {
		CompletionTime int `json:"completionTime"`
	} `json:"statistics"`
}

type updateUserRoleRequest struct {
	Role domain.UserRole `json:"role" binding:"required"`
}

type updateUserStatusRequest struct {
	Status domain.UserStatus `json:"status" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID, appErr := h.identity.Register(c.Request.Context(), req.Username, req.Password)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "注册成功", gin.H{"userId": userID})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	tokens, appErr := h.identity.Login(c.Request.Context(), req.Username, req.Password)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, tokens)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req refreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	tokens, appErr := h.identity.Refresh(c.Request.Context(), req.RefreshToken)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, tokens)
}

func (h *Handler) CreateQuestionnaire(c *gin.Context) {
	var req createQuestionnaireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID := getRequiredUserID(c)
	id, appErr := h.questionnaire.Create(c.Request.Context(), userID, service.CreateQuestionnaireInput{
		Title:       req.Title,
		Description: req.Description,
		Settings:    req.Settings,
		Questions:   req.Questions,
		LogicRules:  req.LogicRules,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "创建成功", gin.H{"id": id})
}

func (h *Handler) GetQuestionnaires(c *gin.Context) {
	userID := getRequiredUserID(c)
	page := parseInt(c.Query("page"), 1)
	limit := parseInt(c.Query("limit"), 20)
	status := strings.TrimSpace(c.Query("status"))
	sortBy := strings.TrimSpace(c.Query("sortBy"))

	items, total, appErr := h.questionnaire.ListMine(c.Request.Context(), userID, domain.QuestionnaireListFilter{
		Page:   page,
		Limit:  limit,
		Status: status,
		SortBy: sortBy,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) UpdateQuestionnaireStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID := getRequiredUserID(c)
	qid := c.Param("id")
	appErr := h.questionnaire.UpdateStatus(c.Request.Context(), userID, qid, service.UpdateQuestionnaireStatusInput{
		Status:   req.Status,
		Deadline: req.Deadline,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"id": qid, "status": req.Status})
}

func (h *Handler) GetQuestionnaireStats(c *gin.Context) {
	userID := getRequiredUserID(c)
	qid := c.Param("id")
	stats, appErr := h.questionnaire.GetStats(c.Request.Context(), userID, qid)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, stats)
}

func (h *Handler) GetQuestionnaireResponses(c *gin.Context) {
	userID := getRequiredUserID(c)
	qid := c.Param("id")
	page := parseInt(c.Query("page"), 1)
	limit := parseInt(c.Query("limit"), 20)
	questionID := strings.TrimSpace(c.Query("questionId"))

	items, total, appErr := h.questionnaire.GetResponses(c.Request.Context(), userID, qid, domain.ResponseListFilter{
		Page:       page,
		Limit:      limit,
		QuestionID: questionID,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) GetSurvey(c *gin.Context) {
	qid := c.Param("id")
	userID := getOptionalUserID(c)
	survey, appErr := h.questionnaire.GetSurveyForFill(c.Request.Context(), qid, userID)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Success(c, survey)
}

func (h *Handler) SubmitResponse(c *gin.Context) {
	qid := c.Param("id")
	var req submitResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID := getOptionalUserID(c)
	appErr := h.questionnaire.SubmitResponse(c.Request.Context(), qid, userID, service.SubmitResponseInput{
		IsAnonymous: req.IsAnonymous,
		Answers:     req.Answers,
		Statistics: domain.ResponseStatistics{
			CompletionTime: req.Statistics.CompletionTime,
		},
	}, c.ClientIP())
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}
	response.Created(c, "提交成功", nil)
}

func (h *Handler) AdminListUsers(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	limit := parseInt(c.Query("limit"), 20)
	status := strings.TrimSpace(c.Query("status"))
	role := strings.TrimSpace(c.Query("role"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	items, total, appErr := h.admin.ListUsers(c.Request.Context(), domain.UserListFilter{
		Page:    page,
		Limit:   limit,
		Status:  status,
		Role:    role,
		Keyword: keyword,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) AdminUpdateUserRole(c *gin.Context) {
	var req updateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID := strings.TrimSpace(c.Param("id"))
	appErr := h.admin.UpdateUserRole(c.Request.Context(), userID, req.Role)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{"id": userID, "role": req.Role})
}

func (h *Handler) AdminUpdateUserStatus(c *gin.Context) {
	var req updateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	userID := strings.TrimSpace(c.Param("id"))
	appErr := h.admin.UpdateUserStatus(c.Request.Context(), userID, req.Status)
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{"id": userID, "status": req.Status})
}

func (h *Handler) AdminListQuestionnaires(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	limit := parseInt(c.Query("limit"), 20)
	status := strings.TrimSpace(c.Query("status"))
	sortBy := strings.TrimSpace(c.Query("sortBy"))
	creatorID := strings.TrimSpace(c.Query("creatorId"))

	items, total, appErr := h.admin.ListQuestionnaires(c.Request.Context(), domain.QuestionnaireAdminListFilter{
		Page:      page,
		Limit:     limit,
		Status:    status,
		SortBy:    sortBy,
		CreatorID: creatorID,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) AdminUpdateQuestionnaireStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "请求参数错误", gin.H{"error": err.Error()})
		return
	}

	qid := strings.TrimSpace(c.Param("id"))
	appErr := h.admin.UpdateQuestionnaireStatus(c.Request.Context(), qid, service.UpdateQuestionnaireStatusInput{
		Status:   req.Status,
		Deadline: req.Deadline,
	})
	if appErr != nil {
		h.writeAppError(c, appErr)
		return
	}

	response.Success(c, gin.H{"id": qid, "status": req.Status})
}

func (h *Handler) writeAppError(c *gin.Context, appErr *apperror.AppError) {
	response.Error(c, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
}

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
