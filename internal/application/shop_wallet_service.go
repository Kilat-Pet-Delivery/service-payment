package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ShopWalletService coordinates shop wallet use cases.
type ShopWalletService struct {
	repo   *repository.GormShopWalletRepository
	logger *zap.Logger
}

func NewShopWalletService(repo *repository.GormShopWalletRepository, logger *zap.Logger) *ShopWalletService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ShopWalletService{repo: repo, logger: logger}
}

type ShopWalletDTO struct {
	ShopID          uuid.UUID                          `json:"shop_id"`
	BalanceMyrCents int64                              `json:"balance_myr_cents"`
	Ledger          []repository.ShopWalletLedgerModel `json:"ledger"`
}

type WithdrawRequest struct {
	AmountMyrCents int64     `json:"amount_myr_cents" binding:"required"`
	BankAccountID  uuid.UUID `json:"bank_account_id" binding:"required"`
}

func (s *ShopWalletService) GetWallet(ctx context.Context, shopID uuid.UUID) (*ShopWalletDTO, error) {
	balance, err := s.repo.GetBalance(ctx, shopID)
	if err != nil {
		return nil, err
	}
	ledger, err := s.repo.ListLedger(ctx, shopID, 20)
	if err != nil {
		return nil, err
	}
	return &ShopWalletDTO{ShopID: shopID, BalanceMyrCents: balance, Ledger: ledger}, nil
}

func (s *ShopWalletService) ListLedger(ctx context.Context, shopID uuid.UUID, limit int) ([]repository.ShopWalletLedgerModel, error) {
	return s.repo.ListLedger(ctx, shopID, limit)
}

func (s *ShopWalletService) Withdraw(ctx context.Context, shopID uuid.UUID, req WithdrawRequest, key string) (*repository.ShopWithdrawalModel, error) {
	if strings.TrimSpace(key) == "" {
		return nil, domain.NewValidationError("Idempotency-Key header is required")
	}
	if req.AmountMyrCents <= 0 {
		return nil, domain.NewValidationError("amount_myr_cents must be positive")
	}
	if req.AmountMyrCents > 1000000 {
		return nil, domain.NewInvalidStateError("daily_cap_exceeded", "withdrawal")
	}
	if req.BankAccountID == uuid.Nil {
		return nil, domain.NewValidationError("bank_account_id is required")
	}
	return s.repo.Withdraw(ctx, shopID, req.BankAccountID, req.AmountMyrCents, key, hashWithdraw(shopID, req))
}

func (s *ShopWalletService) ListWithdrawals(ctx context.Context, shopID uuid.UUID, limit int) ([]repository.ShopWithdrawalModel, error) {
	return s.repo.ListWithdrawals(ctx, shopID, limit)
}

func (s *ShopWalletService) HandleBookingDelivered(ctx context.Context, bookingID, shopID uuid.UUID, amountMyrCents int64) error {
	if shopID == uuid.Nil || bookingID == uuid.Nil || amountMyrCents <= 0 {
		return nil
	}
	return s.repo.AppendLedger(ctx, repository.ShopWalletLedgerModel{
		ID:             uuid.New(),
		ShopID:         shopID,
		Direction:      "credit",
		AmountMyrCents: amountMyrCents,
		SourceType:     "booking_delivered",
		SourceID:       bookingID,
		CreatedAt:      time.Now().UTC(),
	})
}

func hashWithdraw(shopID uuid.UUID, req WithdrawRequest) string {
	raw := shopID.String() + "|" + req.BankAccountID.String() + "|" + strconv.FormatInt(req.AmountMyrCents, 10)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
