package main

import (
	"fmt"

	b "github.com/heritechie/bot-bca/internal/browser"
	srv "github.com/heritechie/bot-bca/internal/server"
	"github.com/heritechie/bot-bca/internal/utils"
)

func init() {
	b.Init()
}

func main() {
	klikBCAServer := srv.NewKlikBCAServer()
	utils.Log(fmt.Sprintf("Starting KlikBCA Server on port %s\n", klikBCAServer.Addr))
	utils.Log(fmt.Sprintf("http://localhost%s/ping\n", klikBCAServer.Addr))
	utils.Log(fmt.Sprintf("http://localhost%s/mutasi-rekening\n", klikBCAServer.Addr))
	utils.Log(fmt.Sprintf("http://localhost%s/logout\n", klikBCAServer.Addr))
	if err := klikBCAServer.ListenAndServe(); err != nil {
		utils.Log(fmt.Sprintf("Error starting KlikBCA Server: %s\n", err.Error()))
	}
}
