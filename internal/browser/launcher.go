package browser

import (
	"github.com/go-rod/rod/lib/launcher"
)

func GetRemoteLauncher() *launcher.Launcher {
	l := launcher.MustNewManaged("")
	l.Devtools(false)
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")
	l.Set("disable-gpu").Delete("disable-gpu")
	return l
}

func GetLauncher(headlessMode *bool) *launcher.Launcher {
	path, _ := launcher.LookPath()
	var l *launcher.Launcher
	if *headlessMode == false {
		l = launcher.New().Bin(path)
	} else {
		l = launcher.New().Bin(path)
	}

	l.Headless(*headlessMode)
	return l
}
