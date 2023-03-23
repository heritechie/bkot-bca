package dto

import (
	bot "github.com/heritechie/bot-bca/internal/bankbot"
)

// type DataResponse interface {
// 	GetBankAccountDataResponse() *bot.BankAccount
// 	GetBankAccountStatementDataResponse() *bot.BankAccountStatement
// }

type BaseResponse struct {
	Success bool    `json:"success"`
	Message *string `json:"message"`
}

type ResponseBankAccount struct {
	BaseResponse
	Data []bot.BankAccount `json:"data"`
}

type AccountStatement struct {
	BankAccount   *bot.AccountStatementInfo    `json:"account"`
	StatementList []bot.BankAccountStatement   `json:"statements"`
	Summary       *bot.AccountStatementSummary `json:"summary"`
}

type ResponseAccountStatement struct {
	BaseResponse
	Data AccountStatement `json:"data"`
}
