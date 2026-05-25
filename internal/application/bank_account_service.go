package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/repository"
	"github.com/google/uuid"
)

// BankAccountDTO is the API representation of a payout bank account.
type BankAccountDTO struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	BankCode            string    `json:"bank_code"`
	AccountNumberMasked string    `json:"account_number_masked"`
	AccountHolderName   string    `json:"account_holder_name"`
	IsDefault           bool      `json:"is_default"`
	CreatedAt           time.Time `json:"created_at"`
}

// AddBankAccountRequest creates a payout destination.
type AddBankAccountRequest struct {
	BankCode          string `json:"bank_code" binding:"required"`
	AccountNumber     string `json:"account_number" binding:"required"`
	AccountHolderName string `json:"account_holder_name" binding:"required"`
}

// BankAccountService handles runner bank account use cases.
type BankAccountService struct {
	repo repository.BankAccountRepository
}

// NewBankAccountService creates a bank account service.
func NewBankAccountService(repo repository.BankAccountRepository) *BankAccountService {
	return &BankAccountService{repo: repo}
}

func (s *BankAccountService) ListMyBankAccounts(ctx context.Context, userID uuid.UUID) ([]BankAccountDTO, error) {
	accounts, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	results := make([]BankAccountDTO, len(accounts))
	for i := range accounts {
		results[i] = toBankAccountDTO(&accounts[i])
	}
	return results, nil
}

func (s *BankAccountService) AddBankAccount(ctx context.Context, userID uuid.UUID, req AddBankAccountRequest) (*BankAccountDTO, error) {
	bankCode := strings.ToUpper(strings.TrimSpace(req.BankCode))
	accountNumber := strings.TrimSpace(req.AccountNumber)
	holderName := strings.TrimSpace(req.AccountHolderName)
	if bankCode == "" || accountNumber == "" || holderName == "" {
		return nil, domain.NewValidationError("bank code, account number, and account holder name are required")
	}

	existing, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	account := &repository.BankAccountModel{
		ID:                  uuid.New(),
		UserID:              userID,
		BankCode:            bankCode,
		AccountNumberMasked: maskAccountNumber(accountNumber),
		AccountHolderName:   holderName,
		IsDefault:           len(existing) == 0,
		CreatedAt:           time.Now().UTC(),
	}
	if err := s.repo.Insert(ctx, account); err != nil {
		return nil, err
	}
	result := toBankAccountDTO(account)
	return &result, nil
}

func (s *BankAccountService) SetDefault(ctx context.Context, userID, accountID uuid.UUID) error {
	if err := s.repo.SetDefault(ctx, userID, accountID); err != nil {
		if err == domain.ErrNotFound {
			return domain.NewNotFoundError("BankAccount", accountID.String())
		}
		return err
	}
	return nil
}

func (s *BankAccountService) DeleteBankAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.NewNotFoundError("BankAccount", accountID.String())
		}
		return err
	}
	if account.UserID != userID {
		return domain.NewForbiddenError("bank account does not belong to caller")
	}
	if account.IsDefault {
		return domain.NewConflictError("default bank account cannot be deleted")
	}
	if err := s.repo.SoftDelete(ctx, userID, accountID); err != nil {
		return fmt.Errorf("delete bank account: %w", err)
	}
	return nil
}

func toBankAccountDTO(account *repository.BankAccountModel) BankAccountDTO {
	return BankAccountDTO{
		ID:                  account.ID,
		UserID:              account.UserID,
		BankCode:            account.BankCode,
		AccountNumberMasked: account.AccountNumberMasked,
		AccountHolderName:   account.AccountHolderName,
		IsDefault:           account.IsDefault,
		CreatedAt:           account.CreatedAt,
	}
}

func maskAccountNumber(accountNumber string) string {
	cleaned := strings.ReplaceAll(strings.TrimSpace(accountNumber), " ", "")
	if len(cleaned) <= 4 {
		return strings.Repeat("*", len(cleaned))
	}
	return strings.Repeat("*", len(cleaned)-4) + cleaned[len(cleaned)-4:]
}
