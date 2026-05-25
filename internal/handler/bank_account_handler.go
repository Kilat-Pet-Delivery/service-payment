package handler

import (
	"net/http"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-common/response"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/application"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BankAccountHandler handles runner bank account routes.
type BankAccountHandler struct {
	service *application.BankAccountService
}

// NewBankAccountHandler creates a bank account handler.
func NewBankAccountHandler(service *application.BankAccountService) *BankAccountHandler {
	return &BankAccountHandler{service: service}
}

func (h *BankAccountHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	me := r.Group("/me/bank-accounts")
	me.Use(middleware.AuthMiddleware(jwtManager), middleware.RequireRole(auth.RoleRunner))
	{
		me.GET("", h.List)
		me.POST("", h.Add)
		me.POST("/:id/set-default", h.SetDefault)
		me.DELETE("/:id", h.Delete)
	}
}

func (h *BankAccountHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.service.ListMyBankAccounts(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *BankAccountHandler) Add(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req application.AddBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.AddBankAccount(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

func (h *BankAccountHandler) SetDefault(c *gin.Context) {
	userID, accountID, ok := bankAccountRouteIDs(c)
	if !ok {
		return
	}
	if err := h.service.SetDefault(c.Request.Context(), userID, accountID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "default bank account updated"})
}

func (h *BankAccountHandler) Delete(c *gin.Context) {
	userID, accountID, ok := bankAccountRouteIDs(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBankAccount(c.Request.Context(), userID, accountID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func bankAccountRouteIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, uuid.Nil, false
	}
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid bank account ID")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, accountID, true
}
