// alonabot-adapter-onebot11 命令运行官方 OneBot 11 Adapter。
package main

import (
	"log/slog"
	"os"

	"github.com/AlonaBot/AlonaBot/adapters/onebot11"
	"github.com/AlonaBot/AlonaBot/sdk"
)

func main() {
	if err := sdk.RunAdapter(onebot11.New()); err != nil {
		slog.Error("onebot adapter stopped with an error", "error", err)
		os.Exit(1)
	}
}
