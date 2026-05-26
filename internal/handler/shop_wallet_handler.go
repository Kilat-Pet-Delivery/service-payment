package handler

import (
	"net/http"
	"strconv"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-common/response"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/application"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ShopWalletHandler exposes shop wallet endpoints.
type ShopWalletHandler struct {
	service *application.ShopWalletService
}

func NewShopWalletHandler(service *application.ShopWalletService) *ShopWalletHandler {
	return &ShopWalletHandler{service: service}
}

func (h *ShopWalletHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	authMW := middleware.AuthMiddleware(jwtManager)
	shops := r.Group("/payments/shops/:id")
	shops.Use(authMW)
	{
		shops.GET("/wallet", h.GetWallet)
		shops.GET("/wallet/ledger", h.ListLedger)
		shops.POST("/withdraw", h.Withdraw)
		shops.GET("/withdrawals", h.ListWithdrawals)
	}
}

func (h *ShopWalletHandler) GetWallet(c *gin.Context) {
	shopID, ok := parseShopID(c)
	if !ok {
		return
	}
	result, err := h.service.GetWallet(c.Request.Context(), shopID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ShopWalletHandler) ListLedger(c *gin.Context) {
	shopID, ok := parseShopID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	result, err := h.service.ListLedger(c.Request.Context(), shopID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ShopWalletHandler) Withdraw(c *gin.Context) {
	shopID, ok := parseShopID(c)
	if !ok {
		return
	}
	var req application.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Withdraw(c.Request.Context(), shopID, req, c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ShopWalletHandler) ListWithdrawals(c *gin.Context) {
	shopID, ok := parseShopID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	result, err := h.service.ListWithdrawals(c.Request.Context(), shopID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func parseShopID(c *gin.Context) (uuid.UUID, bool) {
	if _, ok := middleware.GetUserID(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	shopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid shop id")
		return uuid.Nil, false
	}
	return shopID, true
}
