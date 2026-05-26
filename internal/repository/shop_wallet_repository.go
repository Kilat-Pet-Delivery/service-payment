package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShopWalletLedgerModel is an append-only shop wallet entry.
type ShopWalletLedgerModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	ShopID         uuid.UUID `gorm:"type:uuid;index;not null"`
	Direction      string    `gorm:"type:varchar(10);not null"`
	AmountMyrCents int64     `gorm:"not null"`
	SourceType     string    `gorm:"type:varchar(40);not null;uniqueIndex:idx_shop_ledger_source"`
	SourceID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_shop_ledger_source"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (ShopWalletLedgerModel) TableName() string { return "shop_wallet_ledger" }

// ShopWithdrawalModel tracks shop withdrawal requests.
type ShopWithdrawalModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	ShopID         uuid.UUID `gorm:"type:uuid;index;not null"`
	AmountMyrCents int64     `gorm:"not null"`
	BankAccountID  uuid.UUID `gorm:"type:uuid;not null"`
	Status         string    `gorm:"type:varchar(20);index;not null"`
	RequestedAt    time.Time `gorm:"not null"`
	PaidAt         *time.Time
	FailureReason  *string
	IdempotencyKey string `gorm:"type:varchar(128);uniqueIndex;not null"`
}

func (ShopWithdrawalModel) TableName() string { return "shop_withdrawals" }

// PaymentIdempotencyKeyModel stores payment idempotency hashes.
type PaymentIdempotencyKeyModel struct {
	Key         string    `gorm:"primaryKey;size:128"`
	RequestHash string    `gorm:"column:request_hash;not null;size:64"`
	CreatedAt   time.Time `gorm:"not null"`
	ExpiresAt   time.Time `gorm:"not null"`
}

func (PaymentIdempotencyKeyModel) TableName() string { return "idempotency_keys" }

// GormShopWalletRepository persists shop wallet state.
type GormShopWalletRepository struct {
	db *gorm.DB
}

func NewGormShopWalletRepository(db *gorm.DB) *GormShopWalletRepository {
	return &GormShopWalletRepository{db: db}
}

func (r *GormShopWalletRepository) GetBalance(ctx context.Context, shopID uuid.UUID) (int64, error) {
	var credit, debit int64
	if err := r.db.WithContext(ctx).Model(&ShopWalletLedgerModel{}).Where("shop_id = ? AND direction = ?", shopID, "credit").Select("COALESCE(SUM(amount_myr_cents), 0)").Scan(&credit).Error; err != nil {
		return 0, err
	}
	if err := r.db.WithContext(ctx).Model(&ShopWalletLedgerModel{}).Where("shop_id = ? AND direction = ?", shopID, "debit").Select("COALESCE(SUM(amount_myr_cents), 0)").Scan(&debit).Error; err != nil {
		return 0, err
	}
	return credit - debit, nil
}

func (r *GormShopWalletRepository) AppendLedger(ctx context.Context, entry ShopWalletLedgerModel) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(&entry).Error; err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *GormShopWalletRepository) ListLedger(ctx context.Context, shopID uuid.UUID, limit int) ([]ShopWalletLedgerModel, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []ShopWalletLedgerModel
	err := r.db.WithContext(ctx).Where("shop_id = ?", shopID).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *GormShopWalletRepository) Withdraw(ctx context.Context, shopID, bankAccountID uuid.UUID, amount int64, key, hash string) (*ShopWithdrawalModel, error) {
	var out *ShopWithdrawalModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing PaymentIdempotencyKeyModel
		err := tx.Where("key = ?", key).Take(&existing).Error
		if err == nil {
			if existing.RequestHash != hash {
				return domain.NewConflictError("idempotency key reused with different request")
			}
			var existingWithdrawal ShopWithdrawalModel
			if err := tx.Where("idempotency_key = ?", key).Take(&existingWithdrawal).Error; err != nil {
				return err
			}
			out = &existingWithdrawal
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		repo := &GormShopWalletRepository{db: tx}
		balance, err := repo.GetBalance(ctx, shopID)
		if err != nil {
			return err
		}
		if balance < amount {
			return domain.NewInvalidStateError("insufficient_funds", "withdrawal")
		}
		withdrawalID := uuid.New()
		now := time.Now().UTC()
		if err := tx.Create(&ShopWalletLedgerModel{
			ID:             uuid.New(),
			ShopID:         shopID,
			Direction:      "debit",
			AmountMyrCents: amount,
			SourceType:     "withdrawal",
			SourceID:       withdrawalID,
			CreatedAt:      now,
		}).Error; err != nil {
			return err
		}
		withdrawal := ShopWithdrawalModel{
			ID:             withdrawalID,
			ShopID:         shopID,
			AmountMyrCents: amount,
			BankAccountID:  bankAccountID,
			Status:         "pending",
			RequestedAt:    now,
			IdempotencyKey: key,
		}
		if err := tx.Create(&withdrawal).Error; err != nil {
			return err
		}
		if err := tx.Create(&PaymentIdempotencyKeyModel{Key: key, RequestHash: hash, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}).Error; err != nil {
			return err
		}
		out = &withdrawal
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("withdraw: %w", err)
	}
	return out, nil
}

func (r *GormShopWalletRepository) ListWithdrawals(ctx context.Context, shopID uuid.UUID, limit int) ([]ShopWithdrawalModel, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []ShopWithdrawalModel
	err := r.db.WithContext(ctx).Where("shop_id = ?", shopID).Order("requested_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func isUniqueViolation(err error) bool {
	return err != nil && (errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint failed"))
}
