package browser

import (
	"github.com/heritechie/bot-bca/internal/bankbot"
)

type Session struct {
	Credential    bankbot.Credential
	LoginIsActive bool
}

func NewSession(credential bankbot.Credential, loginIsActive bool) *Session {
	return &Session{
		Credential:    credential,
		LoginIsActive: loginIsActive,
	}
}
