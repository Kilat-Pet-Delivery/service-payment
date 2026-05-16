// Package handler contains HTTP handler unit tests.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-proto/dto"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/rail"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---- fakes ----

// fakeOwnership is a controllable stub for DestinationOwnership.
type fakeOwnership struct {
	result bool
	err    error
}

func (f *fakeOwnership) BelongsTo(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.result, f.err
}

// fakeCashOutRepo is a controllable stub for CashOutRepository.
type fakeCashOutRepo struct {
	balance    int64
	balanceErr error
	insertErr  error
}

func (f *fakeCashOutRepo) Insert(_ context.Context, _ *repository.CashOutModel) error {
	return f.insertErr
}

func (f *fakeCashOutRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string, _ *string) error {
	return nil
}

func (f *fakeCashOutRepo) MarkCompleted(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (f *fakeCashOutRepo) GetAvailableBalanceCents(_ context.Context, _ uuid.UUID) (int64, error) {
	return f.balance, f.balanceErr
}

// fakeRail is a no-op rail for unit tests.
type fakeRail struct{}

func (f *fakeRail) Submit(_ context.Context, _ uuid.UUID, _ int64, _ uuid.UUID) (string, error) {
	return "DN-fake-ref", nil
}

func (f *fakeRail) Status(_ context.Context, _ string) (rail.RailStatus, error) {
	return rail.RailStatusCompleted, nil
}

// ---- helpers ----

// newTestRouter wires a CashOutHandler and injects auth claims so we can call
// the endpoint without a real JWT.
func newTestRouter(h *CashOutHandler, runnerID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Inject auth context directly (bypass JWT signature validation in unit tests).
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, runnerID)
		c.Set(middleware.ContextKeyRole, auth.RoleRunner)
		c.Next()
	})
	apiV1 := r.Group("/api/v1")
	// Register only the /payouts group without the real AuthMiddleware or
	// RequireRole so we can inject claims above. The production path applies
	// these middlewares; the unit tests exercise handler logic only.
	apiV1.POST("/payouts/cash-out", h.CashOut)
	return r
}

func newHandler(repo *fakeCashOutRepo, own *fakeOwnership) *CashOutHandler {
	logger, _ := zap.NewDevelopment()
	return NewCashOutHandler(repo, own, &fakeRail{}, 30*time.Second, logger)
}

func cashOutBody(t *testing.T, amountCents int64, destID string) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(dto.CashOutRequest{AmountMyrCents: amountCents, DestinationID: destID})
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ---- tests ----

// TestCashOut_AmountExceedsBalance_Returns400 verifies that a request whose
// amount+fee exceeds the runner's available balance is rejected with 400.
func TestCashOut_AmountExceedsBalance_Returns400(t *testing.T) {
	runnerID := uuid.New()
	destID := uuid.New()

	repo := &fakeCashOutRepo{balance: 5000} // 50 MYR available
	own := &fakeOwnership{result: true}
	h := newHandler(repo, own)
	r := newTestRouter(h, runnerID)

	// Request 100 MYR (10000 cents) + 50 cent fee = 10050 cents > 5000 balance.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/cash-out", cashOutBody(t, 10000, destID.String()))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "amount exceeds available balance")
}

// TestCashOut_WrongDestinationOwner_Returns403 verifies that a request for a
// destination not owned by the authenticated runner is rejected with 403.
func TestCashOut_WrongDestinationOwner_Returns403(t *testing.T) {
	runnerID := uuid.New()
	destID := uuid.New()

	repo := &fakeCashOutRepo{balance: 999999}
	own := &fakeOwnership{result: false} // destination does not belong to runner
	h := newHandler(repo, own)
	r := newTestRouter(h, runnerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/cash-out", cashOutBody(t, 1000, destID.String()))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "destination does not belong to runner")
}

// TestCashOut_InvalidBody_Returns400 exercises DTO validation: a zero amount
// must be rejected before any DB access.
func TestCashOut_InvalidBody_Returns400(t *testing.T) {
	runnerID := uuid.New()
	destID := uuid.New()

	repo := &fakeCashOutRepo{}
	own := &fakeOwnership{}
	h := newHandler(repo, own)
	r := newTestRouter(h, runnerID)

	// amountMyrCents = 0 fails Validate().
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/cash-out", cashOutBody(t, 0, destID.String()))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCashOut_ValidRequest_Returns202_Unit verifies a well-formed request with
// sufficient balance returns 202 and a valid response body (unit-level, no DB).
func TestCashOut_ValidRequest_Returns202_Unit(t *testing.T) {
	runnerID := uuid.New()
	destID := uuid.New()

	repo := &fakeCashOutRepo{balance: 999999}
	own := &fakeOwnership{result: true}
	h := newHandler(repo, own)
	r := newTestRouter(h, runnerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/cash-out", cashOutBody(t, 1000, destID.String()))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp dto.CashOutResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.CashOutID)
	// 30s delay → ceil(30/60) = 1 minute
	assert.Equal(t, 1, resp.EtaMinutes)
}

// TestCashOut_EmptyDestinationID_Returns400 verifies that an empty destinationId
// is rejected by DTO validation before any DB access.
func TestCashOut_EmptyDestinationID_Returns400(t *testing.T) {
	runnerID := uuid.New()

	repo := &fakeCashOutRepo{}
	own := &fakeOwnership{}
	h := newHandler(repo, own)
	r := newTestRouter(h, runnerID)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/cash-out", cashOutBody(t, 1000, ""))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
