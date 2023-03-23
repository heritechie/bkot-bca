package browser

import (
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/heritechie/bot-bca/internal/bankbot"
	"github.com/heritechie/bot-bca/internal/utils"
)

type CurrentPage struct {
	Bankbot     bankbot.BankBot
	CurrentPage rod.Page
}

type Browser struct {
	Launcher        *launcher.Launcher
	Instance        *rod.Browser
	CurrentPageList []*CurrentPage
}

var LocalBrowser Browser

func Init() {
	utils.LogToFile = true
	headlessMode := true
	fileLines := utils.GetLinesStr("config.txt")

	for _, line := range fileLines {
		split := strings.Split(line, "=")
		key := split[0]
		val := split[1]

		if key == "HEADLESS_MODE" {
			trimmedVal := strings.Trim(val, " ")
			if trimmedVal == "false" || trimmedVal == "FALSE" {
				headlessMode = false
			}

		}

		if key == "LOG_TO_FILE" && (val == "false" || val == "FALSE") {
			utils.LogToFile = false
		}

		if key == "LOG_FILEPATH" && val != "" {
			utils.LogFilePath = &val
		}

	}
	browser := &Browser{
		Launcher: GetLauncher(&headlessMode),
	}

	utils.Log("Initialize browser")
	LocalBrowser.Instance = rod.New().ControlURL(browser.Launcher.MustLaunch()).MustConnect()
}
