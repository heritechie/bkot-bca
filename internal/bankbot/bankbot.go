package bankbot

import (
	"github.com/go-rod/rod"
)

type BankAccount struct {
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	Currency      string `json:"currency"`
	Balance       string `json:"balance"`
}

type AccountStatementInfo struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Currency      string `json:"currency"`
	Period        string `json:"period"`
}

type AccountStatementSummary struct {
	StartBalance string `json:"start_balance"`
	EndBalance   string `json:"end_balance"`
	TotalDebit   string `json:"total_debit"`
	TotalCredit  string `json:"total_credit"`
}

type BankAccountStatement struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	Branch      string `json:"branch"`
	Amount      string `json:"amount"`
	Type        string `json:"type"`
	Balance     string `json:"balance"`
}

type BankBot interface {
	NavigateToAuthenticatedPage(*rod.Page) *rod.Page
	NavigateToLoginPage(*rod.Page) *rod.Page
	GetAccountList(*rod.Page) []BankAccount
	GetAccountStatementList(*rod.Page) (*string, []BankAccountStatement, *AccountStatementInfo, *AccountStatementSummary)
	CheckLoginSessionIsActive(*rod.Page) (bool, *string)
	IsOnLoginPage(*rod.Page) bool
	Login(*rod.Page)
	Logout(*rod.Page)
	SetLoginSession(bool)
	GetLoginSession() bool
	SetCurrentUrl(url string)
	GetCurrentUrl() string
	GetName() string
}

func (b *BankAccount) GetBankAccountDataResponse() *BankAccount {
	return b
}

func (b *BankAccountStatement) GetBankAccountStatementDataResponse() *BankAccountStatement {
	return b
}
