package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/heritechie/bot-bca/internal/bankbot"
	bca "github.com/heritechie/bot-bca/internal/bca/klikbca"
	b "github.com/heritechie/bot-bca/internal/browser"
	"github.com/heritechie/bot-bca/internal/dto"
	"github.com/heritechie/bot-bca/internal/utils"
)

const (
	DEFAULT_KLIK_BCA_PORT = "8090"
)

type KlikBCAServer struct {
	http.Server
}

var klikBCAUsername string
var klikBCAPIN string

func NewKlikBCAServer() *http.Server {
	fileLines := utils.GetLinesStr("config.txt")

	var customPort *string
	for _, line := range fileLines {
		split := strings.Split(line, "=")
		key := split[0]
		val := split[1]

		if key == "KLIKBCA_USERNAME" {
			klikBCAUsername = strings.Trim(val, " ")
		}

		if key == "KLIKBCA_PIN" {
			klikBCAPIN = strings.Trim(val, " ")
		}

		if key == "KLIKBCA_SERVER_PORT" {
			customPort = &val
		}
	}

	if klikBCAUsername == "" || klikBCAPIN == "" {
		panic("KLIKBCA_USERNAME & KLKIKBCA_PIN belum di set di file config")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/ping", PongHandler)
	mux.HandleFunc("/logout", klikBCALogoutHandler)
	mux.HandleFunc("/norek", klikBCAGetAccountsHandler)
	mux.HandleFunc("/mutasi-rekening", klikBCAGetAccountStatementsHandler)

	portStr := fmt.Sprintf(":%s", DEFAULT_KLIK_BCA_PORT)

	if customPort != nil {
		portStr = strings.Trim(*customPort, " ")
	}

	serverPort := fmt.Sprintf(":%s", portStr)

	return &http.Server{
		Addr:         serverPort,
		Handler:      http.TimeoutHandler(mux, 45*time.Second, "Timeout!\n"),
		ReadTimeout:  45 * time.Second,
		WriteTimeout: 45 * time.Second,
	}
}

func klikBCAGetAccountsHandler(w http.ResponseWriter, _ *http.Request) {
	if len(b.LocalBrowser.CurrentPageList) == 0 {
		loginAccount := &bca.KlikBCAAccount{
			Username: klikBCAUsername,
			PIN:      klikBCAPIN,
		}

		klikBCA := bca.NewKlikBCA(loginAccount)
		page := stealth.MustPage(b.LocalBrowser.Instance)

		currentPage := &b.CurrentPage{
			Bankbot:     klikBCA,
			CurrentPage: *page,
		}

		b.LocalBrowser.CurrentPageList = append(b.LocalBrowser.CurrentPageList, currentPage)
	}

	currentP := b.LocalBrowser.CurrentPageList[0]
	bot := currentP.Bankbot
	p := &currentP.CurrentPage

	bot.NavigateToAuthenticatedPage(p)

	isOnLoginPage := bot.IsOnLoginPage(p)

	maxRetry := 3
	var alertMsg string
	for maxRetry > 0 && isOnLoginPage {
		wait := p.EachEvent(func(e *proto.PageJavascriptDialogOpening) (stop bool) {
			alertMsg = e.Message
			maxRetry = 0
			return true
		})
		bot.NavigateToLoginPage(p)
		bot.Login(p)
		wait()
		loginSessionIsActive, _ := bot.CheckLoginSessionIsActive(p)
		isOnLoginPage = !loginSessionIsActive
		maxRetry--
	}

	w.Header().Set("Content-Type", "application/json")
	accountList := []bankbot.BankAccount{}
	if alertMsg != "" {
		fmt.Fprintf(w, alertMsg)
		bot.SetLoginSession(false)
	} else if maxRetry == 0 {
		fmt.Fprintf(w, "Failed to access Klik BCA")
		bot.SetLoginSession(false)
	} else {
		bot.SetLoginSession(true)
		bot.SetCurrentUrl(bca.AUTHENTICATED_URL)
		accountList = bot.GetAccountList(p)
		response := dto.ResponseBankAccount{
			BaseResponse: dto.BaseResponse{
				Success: true,
			},
			Data: accountList,
		}

		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
	}
}

func klikBCAGetAccountStatementsHandler(w http.ResponseWriter, _ *http.Request) {
	if len(b.LocalBrowser.CurrentPageList) == 0 {
		loginAccount := &bca.KlikBCAAccount{
			Username: klikBCAUsername,
			PIN:      klikBCAPIN,
		}

		klikBCA := bca.NewKlikBCA(loginAccount)
		page := stealth.MustPage(b.LocalBrowser.Instance)

		currentPage := &b.CurrentPage{
			Bankbot:     klikBCA,
			CurrentPage: *page,
		}

		b.LocalBrowser.CurrentPageList = append(b.LocalBrowser.CurrentPageList, currentPage)
		currentPage.Bankbot.NavigateToAuthenticatedPage(page)
	}

	currentP := b.LocalBrowser.CurrentPageList[0]
	bot := currentP.Bankbot
	p := &currentP.CurrentPage

	loginSessionIsActive, _ := bot.CheckLoginSessionIsActive(p)

	isOnLoginPage := bot.IsOnLoginPage(p)

	utils.Log(fmt.Sprintf("*isOnLoginPage %v", isOnLoginPage))

	maxRetry := 3
	alertMsgChn := make(chan string)
	alertMsg := ""
	for maxRetry > 0 && isOnLoginPage && alertMsg == "" {
		if alertMsg == "" {
			w, h := p.MustHandleDialog()
			go func() {
				dialog := w()
				alertMsgChn <- dialog.Message
				h(false, "")
			}()

			bot.Login(p)

		}

		select {
		case alertMsg = <-alertMsgChn:
		case <-time.After(2 * time.Second):
			loginSessionIsActive, _ = bot.CheckLoginSessionIsActive(p)
			isOnLoginPage = !loginSessionIsActive
			maxRetry--
		}

	}

	w.Header().Set("Content-Type", "application/json")
	accountStatementResponse := dto.AccountStatement{}
	baseResponse := &dto.BaseResponse{
		Success: false,
	}

	if alertMsg != "" {
		baseResponse.Message = &alertMsg
		bot.SetLoginSession(false)
	} else if maxRetry == 0 {
		defaultErrMsg := "Failed to access Klik BCA"
		baseResponse.Message = &defaultErrMsg
		bot.SetLoginSession(false)
	} else {
		bot.SetLoginSession(true)
		bot.SetCurrentUrl(p.MustInfo().URL)
		bot.SetLoginSession(true)
		var (
			accountStatementList []bankbot.BankAccountStatement
			accountInfo          *bankbot.AccountStatementInfo
		)

		errMsg, accountStatementList, accountInfo, _ := bot.GetAccountStatementList(p)

		if errMsg != nil && strings.Contains(*errMsg, "out of service") {
			errMsg, accountStatementList, accountInfo, _ = bot.GetAccountStatementList(p)
			utils.Log(fmt.Sprintf("* %v", *errMsg))
		}

		baseResponse.Success = true
		baseResponse.Message = errMsg

		accountStatementResponse = dto.AccountStatement{
			StatementList: accountStatementList,
			BankAccount:   accountInfo,
		}

	}

	response := dto.ResponseAccountStatement{
		BaseResponse: *baseResponse,
		Data:         accountStatementResponse,
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

func klikBCALogoutHandler(w http.ResponseWriter, _ *http.Request) {
	if len(b.LocalBrowser.CurrentPageList) > 0 {
		currentP := b.LocalBrowser.CurrentPageList[0]
		bot := currentP.Bankbot
		loginSessionActive := bot.GetLoginSession()
		if loginSessionActive {
			p := &currentP.CurrentPage
			bot.Logout(p)
			bot.SetLoginSession(false)
			bot.SetCurrentUrl(bca.LOGIN_URL)
		}

	}

	message := "Logout successfully"

	response := dto.BaseResponse{
		Success: true,
		Message: &message,
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	w.Write(jsonResp)
}

func PongHandler(w http.ResponseWriter, _ *http.Request) {
	message := "pong"

	response := dto.BaseResponse{
		Success: true,
		Message: &message,
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}
