package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BankAccountModel stores a runner payout destination.
type BankAccountModel struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID              uuid.UUID  `gorm:"type:uuid;not null;index"`
	BankCode            string     `gorm:"type:varchar(20);not null"`
	AccountNumberMasked string     `gorm:"type:varchar(40);not null"`
	AccountHolderName   string     `gorm:"type:varchar(255);not null"`
	IsDefault           bool       `gorm:"not null;default:false;index"`
	DeletedAt           *time.Time `gorm:"type:timestamptz;index"`
	CreatedAt           time.Time  `gorm:"type:timestamptz;not null"`
}

func (BankAccountModel) TableName() string {
	return "bank_accounts"
}

// BankAccountRepository defines bank account persistence operations.
type BankAccountRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]BankAccountModel, error)
	Insert(ctx context.Context, account *BankAccountModel) error
	FindByID(ctx context.Context, id uuid.UUID) (*BankAccountModel, error)
	FindDefaultByUserID(ctx context.Context, userID uuid.UUID) (*BankAccountModel, error)
	SetDefault(ctx context.Context, userID, accountID uuid.UUID) error
	SoftDelete(ctx context.Context, userID, accountID uuid.UUID) error
	BelongsTo(ctx context.Context, accountID, userID uuid.UUID) (bool, error)
}

// GormBankAccountRepository is a GORM-backed bank account repository.
type GormBankAccountRepository struct {
	db *gorm.DB
}

// NewGormBankAccountRepository creates a bank account repository.
func NewGormBankAccountRepository(db *gorm.DB) *GormBankAccountRepository {
	return &GormBankAccountRepository{db: db}
}

func (r *GormBankAccountRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]BankAccountModel, error) {
	var accounts []BankAccountModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("is_default DESC, created_at DESC").
		Find(&accounts).Error
	return accounts, err
}

func (r *GormBankAccountRepository) Insert(ctx context.Context, account *BankAccountModel) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *GormBankAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*BankAccountModel, error) {
	var account BankAccountModel
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *GormBankAccountRepository) FindDefaultByUserID(ctx context.Context, userID uuid.UUID) (*BankAccountModel, error) {
	var account BankAccountModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ? AND deleted_at IS NULL", userID, true).
		First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *GormBankAccountRepository) SetDefault(ctx context.Context, userID, accountID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&BankAccountModel{}).
			Where("id = ? AND user_id = ? AND deleted_at IS NULL", accountID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return domain.ErrNotFound
		}
		if err := tx.Model(&BankAccountModel{}).
			Where("user_id = ? AND deleted_at IS NULL", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&BankAccountModel{}).
			Where("id = ?", accountID).
			Update("is_default", true).Error
	})
}

func (r *GormBankAccountRepository) SoftDelete(ctx context.Context, userID, accountID uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&BankAccountModel{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", accountID, userID).
		Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormBankAccountRepository) BelongsTo(ctx context.Context, accountID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&BankAccountModel{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", accountID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
