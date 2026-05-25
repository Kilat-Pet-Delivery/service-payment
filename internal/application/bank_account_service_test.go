package application

import (
	"context"
	"errors"
	"testing"

	"github.com/Kilat-Pet-Delivery/lib-common/domain"
	"github.com/Kilat-Pet-Delivery/service-payment/internal/repository"
	"github.com/google/uuid"
)

type fakeBankAccountRepo struct {
	accounts map[uuid.UUID]*repository.BankAccountModel
}

func newFakeBankAccountRepo() *fakeBankAccountRepo {
	return &fakeBankAccountRepo{accounts: map[uuid.UUID]*repository.BankAccountModel{}}
}

func (r *fakeBankAccountRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]repository.BankAccountModel, error) {
	results := []repository.BankAccountModel{}
	for _, account := range r.accounts {
		if account.UserID == userID && account.DeletedAt == nil {
			results = append(results, *account)
		}
	}
	return results, nil
}

func (r *fakeBankAccountRepo) Insert(_ context.Context, account *repository.BankAccountModel) error {
	copy := *account
	r.accounts[account.ID] = &copy
	return nil
}

func (r *fakeBankAccountRepo) FindByID(_ context.Context, id uuid.UUID) (*repository.BankAccountModel, error) {
	account, ok := r.accounts[id]
	if !ok || account.DeletedAt != nil {
		return nil, domain.ErrNotFound
	}
	copy := *account
	return &copy, nil
}

func (r *fakeBankAccountRepo) FindDefaultByUserID(_ context.Context, userID uuid.UUID) (*repository.BankAccountModel, error) {
	for _, account := range r.accounts {
		if account.UserID == userID && account.IsDefault && account.DeletedAt == nil {
			copy := *account
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeBankAccountRepo) SetDefault(_ context.Context, userID, accountID uuid.UUID) error {
	account, ok := r.accounts[accountID]
	if !ok || account.UserID != userID || account.DeletedAt != nil {
		return domain.ErrNotFound
	}
	for _, item := range r.accounts {
		if item.UserID == userID {
			item.IsDefault = false
		}
	}
	account.IsDefault = true
	return nil
}

func (r *fakeBankAccountRepo) SoftDelete(_ context.Context, userID, accountID uuid.UUID) error {
	account, ok := r.accounts[accountID]
	if !ok || account.UserID != userID {
		return domain.ErrNotFound
	}
	now := account.CreatedAt
	account.DeletedAt = &now
	return nil
}

func (r *fakeBankAccountRepo) BelongsTo(_ context.Context, accountID, userID uuid.UUID) (bool, error) {
	account, ok := r.accounts[accountID]
	return ok && account.UserID == userID && account.DeletedAt == nil, nil
}

func Test_AddBankAccount_FirstOneBecomesDefault(t *testing.T) {
	service := NewBankAccountService(newFakeBankAccountRepo())
	userID := uuid.New()

	result, err := service.AddBankAccount(context.Background(), userID, AddBankAccountRequest{
		BankCode:          "maybank",
		AccountNumber:     "1234567890",
		AccountHolderName: "Runner One",
	})
	if err != nil {
		t.Fatalf("AddBankAccount returned error: %v", err)
	}
	if !result.IsDefault {
		t.Fatalf("expected first bank account to be default")
	}
	if result.AccountNumberMasked != "******7890" {
		t.Fatalf("expected masked account number, got %s", result.AccountNumberMasked)
	}
}

func Test_SetDefault_FlipsDefault_AndUnflipsOldDefault(t *testing.T) {
	repo := newFakeBankAccountRepo()
	service := NewBankAccountService(repo)
	userID := uuid.New()
	first, _ := service.AddBankAccount(context.Background(), userID, AddBankAccountRequest{BankCode: "MBB", AccountNumber: "11111111", AccountHolderName: "Runner"})
	second, _ := service.AddBankAccount(context.Background(), userID, AddBankAccountRequest{BankCode: "CIMB", AccountNumber: "22222222", AccountHolderName: "Runner"})

	if err := service.SetDefault(context.Background(), userID, second.ID); err != nil {
		t.Fatalf("SetDefault returned error: %v", err)
	}
	if repo.accounts[first.ID].IsDefault {
		t.Fatalf("expected old default to be unset")
	}
	if !repo.accounts[second.ID].IsDefault {
		t.Fatalf("expected new default to be set")
	}
}

func Test_DeleteDefault_Rejected_Returns409(t *testing.T) {
	service := NewBankAccountService(newFakeBankAccountRepo())
	userID := uuid.New()
	account, _ := service.AddBankAccount(context.Background(), userID, AddBankAccountRequest{BankCode: "MBB", AccountNumber: "11111111", AccountHolderName: "Runner"})

	err := service.DeleteBankAccount(context.Background(), userID, account.ID)
	if err == nil {
		t.Fatalf("expected deleting default to fail")
	}
	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != 409 {
		t.Fatalf("expected conflict domain error, got %v", err)
	}
}
