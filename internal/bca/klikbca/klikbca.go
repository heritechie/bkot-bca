package bca

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/heritechie/bot-bca/internal/bankbot"
	"github.com/heritechie/bot-bca/internal/browser"
	utls "github.com/heritechie/bot-bca/internal/utils"
)

const (
	BANKBOT_NAME                                 = "KLIK_BCA"
	BASE_URL                                     = "https://ibank.klikbca.com/"
	LOGIN_URL                                    = BASE_URL
	AUTHENTICATED_URL                            = BASE_URL + "authentication.do"
	SELECTOR_PAGE_HOME_LINK                      = "#gotohome > font > b > font > a"
	SELECTOR_INPUT_USERNAME                      = "#user_id"
	SELECTOR_INPUT_PIN                           = "#pswd"
	SELECTOR_FRAME_TOP                           = "html > frameset > frame:nth-child(1)"
	SELECTOR_FRAME_MENU                          = "html > frameset > frameset > frame:nth-child(1)"
	SELECTOR_FRAME_MAIN                          = "html > frameset > frameset > frame:nth-child(2)"
	SELECTOR_MENU_ACCOUNT_INFO                   = "body > table > tbody > tr > td:nth-child(2) > table > tbody > tr:nth-child(17) > td > a"
	SELECTOR_MENU_BACK_TO_MAIN_MENU              = "body > table > tbody > tr > td:nth-child(2) > table > tbody > tr:nth-child(8) > td > a"
	SELECTOR_SUBMENU_BALANCE_INQUIRY             = "body > table > tbody > tr > td:nth-child(2) > table > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(1) > td:nth-child(2) > font > a"
	SELECTOR_FRAME_MENU_CONTENT_ACCOUNT_LIST     = "body > table:nth-child(3) > tbody"
	SELECTOR_SUBMENU_ACCOUNT_STATEMENT           = "body > table > tbody > tr > td:nth-child(2) > table > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(2) > td:nth-child(2) > font > a"
	X_SELECTOR_SUBMENU_ACCOUNT_STATEMENT         = "/html/body/table/tbody/tr/td[2]/table/tbody/tr[4]/td/table/tbody/tr[2]/td[2]/font/a"
	SELECTOR_START_DATE                          = "//*[@id=\"startDt\"]"
	SELECTOR_BTN_VIEW_ACCOUNT_STATEMENT          = "body > table:nth-child(4) > tbody > tr:nth-child(2) > td > input[type=submit]:nth-child(1)"
	X_SELECTOR_BTN_VIEW_ACCOUNT_STATEMENT        = "/html/body/table[4]/tbody/tr[2]/td/input[1]"
	SELECTOR_TABLE_ACCOUNT_STATEMENT_LIST        = "body > table:nth-child(4) > tbody > tr:nth-child(2) > td > table > tbody"
	X_SELECTOR_TABLE_ACCOUNT_STATEMENT_LIST      = "/html/body/table[3]/tbody/tr[2]/td/table/tbody"
	SELECTOR_TABLE_ACCOUNT_ONACCOUNT_STATEMENT   = "body > table:nth-child(4) > tbody > tr:nth-child(1) > td > table > tbody"
	X_SELECTOR_TABLE_ACCOUNT_ONACCOUNT_STATEMENT = "/html/body/table[3]/tbody"
	SEPARATOR_ROW                                = "^^"
	SEPARATOR_COL                                = "^"
)

type KlikBCAAccount struct {
	Username string
	PIN      string
}

type KlikBCA struct {
	Name             string
	CurrentUrl       string
	BaseUrl          string
	LoginUrl         string
	AuthenticatedUrl string
	LoginAccount     *KlikBCAAccount
	LoginSession     *browser.Session
}

func NewKlikBCA(loginAccount *KlikBCAAccount) *KlikBCA {
	return &KlikBCA{
		Name:             BANKBOT_NAME,
		BaseUrl:          BASE_URL,
		LoginAccount:     loginAccount,
		AuthenticatedUrl: AUTHENTICATED_URL,
		LoginSession: &browser.Session{
			LoginIsActive: false,
			Credential: bankbot.Credential{
				Username: loginAccount.Username,
				PIN:      &loginAccount.PIN,
			},
		},
	}
}

func (klikBCA *KlikBCA) NavigateToAuthenticatedPage(page *rod.Page) *rod.Page {
	utls.Log("> NavigateToAuthenticatedPage")
	return page.MustNavigate(klikBCA.AuthenticatedUrl).MustWaitLoad()
}

func (klikBCA *KlikBCA) NavigateToLoginPage(page *rod.Page) *rod.Page {
	utls.Log("> NavigateToLoginPage")
	return page.MustNavigate(klikBCA.LoginUrl).MustWaitLoad()
}

func (klikBCA *KlikBCA) GetAccountList(page *rod.Page) []bankbot.BankAccount {
	utls.Log("> GetAccountList")
	actionClickBalanceInquiry(page)
	accountListStr := getAccountListString(page)

	if isOutOfServiceResponse(accountListStr) {
		utls.Log("Failed Get Account List\n")

		maxRetry := 3
		for maxRetry > 0 && isOutOfServiceResponse(accountListStr) {
			utls.Log("Retry Get Balance Info process\n")
			loginSessionIsAtive, _ := klikBCA.CheckLoginSessionIsActive(page)
			if loginSessionIsAtive {
				actionBackToMainMenu(page)
				actionClickBalanceInquiry(page)
				accountListStr = getAccountListString(page)
			}
			maxRetry--
		}

		if maxRetry != 0 {
			return parseAccountListData(accountListStr)
		}

		return []bankbot.BankAccount{}
	}

	actionBackToMainMenu(page)

	return parseAccountListData(accountListStr)
}

func (klikBCA *KlikBCA) GetAccountStatementList(page *rod.Page) (*string, []bankbot.BankAccountStatement, *bankbot.AccountStatementInfo, *bankbot.AccountStatementSummary) {
	utls.Log("> GetAccountStatementList")
	actionClickAccountStatement(page)
	actionBackToMainMenu(page)
	// actionSelectStartFirstDate(page)
	actionClickViewAccountStatement(page)
	accountStatementListStr, msg := getAccountStatementListString(page)
	accountStatementStr := getAccountOnAccountStatementPage(page)
	// utls.Log("> accountStatementListStr", *accountStatementListStr)
	// utls.Log("> accountStatementStr", accountStatementStr)

	if msg != nil {
		return msg, []bankbot.BankAccountStatement{}, nil, nil
	}

	return nil, parseAccountStatementListData(*accountStatementListStr), parseAccountInfo(accountStatementStr), nil
}

func (klikBCA *KlikBCA) PrintAccountList(accountList []bankbot.BankAccount) {
	utls.Log("> PrintAccountList")

	b, err := json.MarshalIndent(accountList, "", "  ")
	if err != nil {
		utls.Log(err)
	}
	utls.Log(string(b))
}

func (klikBCA *KlikBCA) CheckLoginSessionIsActive(page *rod.Page) (bool, *string) {
	utls.Log("> CheckLoginSessionIsActive")
	var loginSessionIsActive bool
	page.Race().Element(SELECTOR_FRAME_TOP).MustHandle(func(e *rod.Element) {
		topFrameEl := e.MustFrame()
		if topFrameEl != nil {
			utls.Log(">Check for Logout Button")
			logOutBtnEl := topFrameEl.MustElement("#gotohome > font > b > a")
			logOutText := strings.ToLower(logOutBtnEl.MustText())
			utls.Log(fmt.Sprintf("*Button Text: %s\n", logOutText))
			utls.Log("*Login session is active")
			loginSessionIsActive = true
		}

	}).Element(SELECTOR_PAGE_HOME_LINK).MustHandle(func(e *rod.Element) {
		utls.Log(">Check for Home Button")
		homeBtlText := strings.ToLower(e.MustText())
		utls.Log(fmt.Sprintf("*Button Text: %s\n", homeBtlText))
		loginSessionIsActive = false
		utls.Log("*Login session is not active")
	}).MustDo()

	return loginSessionIsActive, nil
}

func (klikBCA *KlikBCA) Login(page *rod.Page) {
	utls.Log("> Login")
	page.MustNavigate(klikBCA.BaseUrl).MustWaitLoad()
	page.MustElement(SELECTOR_INPUT_USERNAME).MustInput(klikBCA.LoginAccount.Username).MustType()
	page.MustElement(SELECTOR_INPUT_PIN).MustInput(klikBCA.LoginAccount.PIN).MustType().Type(input.Enter)
}

func (klikBCA *KlikBCA) Logout(page *rod.Page) {
	utls.Log("> Logout")
	page.Race().Element(SELECTOR_FRAME_TOP).MustHandle(func(e *rod.Element) {
		topFrameEl := e.MustFrame()
		if topFrameEl != nil {
			utls.Log(">Check for Logout Button")
			logOutBtnEl := topFrameEl.MustElement("#gotohome > font > b > a")
			logOutText := strings.ToLower(logOutBtnEl.MustText())
			utls.Log(fmt.Sprintf("*Button Text: %s\n", logOutText))
			logOutBtnEl.MustClick()
			utls.Log("*Logout Successfully")
		}
	}).Element(SELECTOR_PAGE_HOME_LINK).MustHandle(func(e *rod.Element) {
		utls.Log("*Already on login page")
	}).MustDo()
}

func (klikBCA *KlikBCA) IsOnLoginPage(page *rod.Page) bool {
	var isOnLoginPage bool
	page.Race().Element(SELECTOR_FRAME_TOP).MustHandle(func(e *rod.Element) {
		topFrameEl := e.MustFrame()
		if topFrameEl != nil {
			utls.Log(">Check for Logout Button")
			logOutBtnEl := topFrameEl.MustElement("#gotohome > font > b > a")
			logOutText := strings.ToLower(logOutBtnEl.MustText())
			utls.Log(fmt.Sprintf("*Button Text: %s\n", logOutText))
			utls.Log("*Login session is active")
			isOnLoginPage = false
		}
	}).Element(SELECTOR_PAGE_HOME_LINK).MustHandle(func(e *rod.Element) {
		utls.Log(">Check for Home Button")
		homeBtlText := strings.ToLower(e.MustText())
		utls.Log(fmt.Sprintf("*Button Text: %s\n", homeBtlText))
		isOnLoginPage = true
		utls.Log("*Login session is not active")
	}).MustDo()

	return isOnLoginPage
}

func (klikBCA *KlikBCA) SetLoginSession(isActive bool) {
	klikBCA.LoginSession.LoginIsActive = isActive
}

func (klikBCA *KlikBCA) GetLoginSession() bool {
	return klikBCA.LoginSession.LoginIsActive
}

func (klikBCA *KlikBCA) SetCurrentUrl(url string) {
	klikBCA.CurrentUrl = url
}

func (klikBCA *KlikBCA) GetCurrentUrl() string {
	return klikBCA.CurrentUrl
}

func (klikBCA *KlikBCA) GetName() string {
	return klikBCA.Name
}

func getTopFrame(page *rod.Page) *rod.Page {
	var topFrame *rod.Page
	err := rod.Try(func() {
		topFrame = page.Timeout(2 * time.Second).MustElement(SELECTOR_FRAME_TOP).MustFrame()
	})
	if errors.Is(err, context.DeadlineExceeded) {
		utls.Log("*Timeout error, top frame not found")
		return nil
	} else if err != nil {
		utls.Log("*Other types of error, top frame not found")
		return nil
	}
	return topFrame
}

func getLogOutButtonEl(page *rod.Page) *rod.Element {
	topFrame := getTopFrame(page)
	topFrame.MustWaitLoad()
	if topFrame == nil {
		return nil
	}
	logOutBtnEl := topFrame.MustElement("#gotohome > font > b > a")
	return logOutBtnEl
}

func actionClickBalanceInquiry(page *rod.Page) {
	menuFrame := getMenuFrame(page)
	menuFrame.MustWaitLoad()
	accountInfoMenuEl := menuFrame.MustElement(SELECTOR_MENU_ACCOUNT_INFO)
	utls.Log(fmt.Sprintf(">> Click Menu %s\n", accountInfoMenuEl.MustText()))
	accountInfoMenuEl.MustClick()
	balanceInqSubMenuEl := menuFrame.MustElement(SELECTOR_SUBMENU_BALANCE_INQUIRY)
	utls.Log(fmt.Sprintf(">>> Click Menu %s\n", balanceInqSubMenuEl.MustText()))
	balanceInqSubMenuEl.MustClick()
}

func actionClickAccountStatement(page *rod.Page) {
	menuFrame := getMenuFrame(page)
	menuFrame.MustWaitLoad()
	accountInfoMenuEl := menuFrame.MustElement(SELECTOR_MENU_ACCOUNT_INFO)
	utls.Log(fmt.Sprintf(">> Click Menu %s\n", accountInfoMenuEl.MustText()))
	accountInfoMenuEl.MustClick()
	balanceInqSubMenuEl := menuFrame.MustElementX(X_SELECTOR_SUBMENU_ACCOUNT_STATEMENT)
	utls.Log(fmt.Sprintf(">>> Click Menu %s\n", balanceInqSubMenuEl.MustText()))
	balanceInqSubMenuEl.MustClick()
}

func actionSelectStartFirstDate(page *rod.Page) {
	utls.Log(fmt.Sprintf(">> Select First Date \n"))
	mainFrame := page.MustElement(SELECTOR_FRAME_MAIN).MustFrame()
	mainFrame.MustWaitLoad()
	startDateEl := mainFrame.MustElementX(SELECTOR_START_DATE)
	startDateEl.MustSelect("01")
}

func actionClickViewAccountStatement(page *rod.Page) {
	mainFrame := page.MustElement(SELECTOR_FRAME_MAIN).MustFrame()
	mainFrame.MustWaitLoad()
	// wait := mainFrame.EachEvent(func(e *proto.NetworkDataReceived) (stop bool) {
	// 	tableContainerEl := mainFrame.MustElement("body > table:nth-child(3)")
	// 	txtTable := strings.ToLower(tableContainerEl.MustText())

	// 	if !strings.Contains(txtTable, "transaksi gagal") {
	// 		btnEl := mainFrame.MustElement(SELECTOR_BTN_VIEW_ACCOUNT_STATEMENT)
	// 		utls.Log(fmt.Sprintf(">> Click %s\n", btnEl.MustText())
	// 		btnEl.MustClick()
	// 	}
	// 	actionBackToMainMenu(page)
	// 	return true
	// })
	// wait()

	btnEl := mainFrame.MustElement(SELECTOR_BTN_VIEW_ACCOUNT_STATEMENT)
	utls.Log(fmt.Sprintf(">> Click %s\n", btnEl.MustText()))
	btnEl.MustClick()

}

func getAccountListString(page *rod.Page) string {
	mainFrame := page.MustElement(SELECTOR_FRAME_MAIN).MustFrame()
	mainFrame.MustWaitLoad()
	var accountListStr string
	mainFrame.Race().Element(SELECTOR_FRAME_MENU_CONTENT_ACCOUNT_LIST).MustHandle(func(e *rod.Element) {
		accountListStr = e.MustEval(jsGetAccountList()).Str()

	}).MustDo()

	return accountListStr
}

func getAccountStatementListString(page *rod.Page) (*string, *string) {
	mainFrame := page.MustElement(SELECTOR_FRAME_MAIN).MustFrame()
	mainFrame.MustWaitLoad()
	// var accountStatementListStr string
	// mainFrame.Race().Element(SELECTOR_TABLE_ACCOUNT_STATEMENT_LIST).MustHandle(func(e *rod.Element) {
	// 	accountStatementListStr = e.MustEval(jsGetAccountStatementList()).Str()
	// }).MustDo()

	accountStatementEl := mainFrame.MustElement(SELECTOR_TABLE_ACCOUNT_STATEMENT_LIST)
	txt := strings.ToLower(accountStatementEl.MustText())

	if strings.Contains(txt, "transaksi gagal") || strings.Contains(txt, "no transaction") || strings.Contains(txt, "tidak ada transaksi") || strings.Contains(txt, "tidak dapat diproses") || strings.Contains(txt, "out of service") {
		retStr := ""
		return &retStr, &txt
	}
	accountStatementListStr := accountStatementEl.MustEval(jsGetAccountStatementList()).Str()

	return &accountStatementListStr, nil
}

func getAccountOnAccountStatementPage(page *rod.Page) string {
	mainFrame := page.MustElement(SELECTOR_FRAME_MAIN).MustFrame()
	mainFrame.MustWaitLoad()
	// var accountStatementStr string
	// mainFrame.Race().Element(SELECTOR_TABLE_ACCOUNT_ONACCOUNT_STATEMENT).MustHandle(func(e *rod.Element) {
	// 	accountStatementStr = e.MustEval(jsGetAccount()).Str()
	// }).MustDo()

	accountStatementEl := mainFrame.MustElement(SELECTOR_TABLE_ACCOUNT_ONACCOUNT_STATEMENT)
	accountStatementStr := accountStatementEl.MustEval(jsGetAccount()).Str()

	return accountStatementStr
}

func getMenuFrame(page *rod.Page) *rod.Page {
	return page.MustElement(SELECTOR_FRAME_MENU).MustFrame()
}

func parseAccountListData(accountListStr string) []bankbot.BankAccount {
	utls.Log(fmt.Sprintf("*Success Get Account List\n"))
	accountList := strings.Split(accountListStr, SEPARATOR_ROW)
	_, accountList = accountList[0], accountList[1:]

	var bankAccountList []bankbot.BankAccount
	for _, data := range accountList {
		d := strings.Split(data, SEPARATOR_COL)
		bankAccount := bankbot.BankAccount{
			AccountNumber: d[0],
			AccountType:   d[1],
			Currency:      d[2],
			Balance:       d[3],
		}
		bankAccountList = append(bankAccountList, bankAccount)
	}

	return bankAccountList
}

func parseAccountStatementListData(accountStatementListStr string) []bankbot.BankAccountStatement {
	utls.Log(fmt.Sprintf("*Success Get Account Statement List\n"))
	accountStatementList := strings.Split(accountStatementListStr, SEPARATOR_ROW)

	_, accountStatementList = accountStatementList[0], accountStatementList[1:]

	var bankAccountStatementList []bankbot.BankAccountStatement
	for _, data := range accountStatementList {
		d := strings.Split(data, SEPARATOR_COL)

		if len(d) > 5 {
			bankStatement := bankbot.BankAccountStatement{
				Date:        d[0],
				Description: d[1],
				Branch:      d[2],
				Amount:      d[3],
				Type:        d[4],
				Balance:     d[5],
			}
			bankAccountStatementList = append(bankAccountStatementList, bankStatement)
		}

	}

	return bankAccountStatementList
}

func parseAccountInfo(accountInfoStr string) *bankbot.AccountStatementInfo {
	utls.Log(fmt.Sprintf("*Success Get Account Info\n"))
	accountInfo := strings.Split(accountInfoStr, SEPARATOR_ROW)

	return &bankbot.AccountStatementInfo{
		AccountNumber: strings.Split(accountInfo[1], ":")[1],
		Name:          strings.Split(accountInfo[2], ":")[1],
		Period:        strings.Split(accountInfo[3], ":")[1],
		Currency:      strings.Split(accountInfo[4], ":")[1],
	}
}

func actionBackToMainMenu(page *rod.Page) {
	menuFrame := getMenuFrame(page)
	menuFrame.MustWaitLoad()
	backToMainMenuEl := menuFrame.MustElement(SELECTOR_MENU_BACK_TO_MAIN_MENU)
	utls.Log(fmt.Sprintf("> Click Menu %s\n", backToMainMenuEl.MustText()))
	backToMainMenuEl.MustClick()
}

func isOutOfServiceResponse(responseStr string) bool {
	outOfServiceResponse := false
	if strings.Contains(responseStr, "out of service") {
		outOfServiceResponse = true
	}
	return outOfServiceResponse
}

func getHomeButtonEl(page *rod.Page) *rod.Element {
	return page.MustElement(SELECTOR_PAGE_HOME_LINK)
}

func jsGetAccountList() string {
	return fmt.Sprintf(`() => {
		let result = ""
		for (let row of this.rows) 
		{
			let data = ""
			for(let cell of row.cells) 
			{
				data += (cell.innerText.trim() + "%s")
			}
			result += (data.slice(0,-1) + "%s")
		}
		return result.slice(0,-1)
	}`, SEPARATOR_COL, SEPARATOR_ROW)
}

func jsGetAccountStatementList() string {
	return fmt.Sprintf(`() => {
		let result = ""
		for (let row of this.rows) 
		{
			let data = ""
			for(let cell of row.cells) 
			{
				data += (cell.innerText.trim() + "%s")
			}
			result += (data.slice(0,-1) + "%s")
		}
		return result.slice(0,-1)
	}`, SEPARATOR_COL, SEPARATOR_ROW)
}

func jsGetAccount() string {
	return fmt.Sprintf(`() => {
		let result = ""
		for (let row of this.rows) 
		{
			let data = ""
			for(let cell of row.cells) 
			{
				data += cell.innerText.trim()
			}
			result += (data + "%s")
		}
		return result.slice(0,-1)
	}`, SEPARATOR_ROW)
}
