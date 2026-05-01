package main

import (
	"fmt"
	"log/slog"
	"mail-assistant/internal/config"
	"mail-assistant/internal/embedding"
	"mail-assistant/internal/embedding/gigachat"
	"os"

	"github.com/Marlliton/slogpretty"
)

func main() {
	logger := slog.New(slogpretty.New(os.Stdout, &slogpretty.Options{
		Level:      slog.LevelDebug,
		AddSource:  false,
		Colorful:   true,
		Multiline:  true,
		TimeFormat: "[02.01.06 15:04:05]",
	}))
	slog.SetDefault(logger)

	c, err := config.New()
	if err != nil {
		slog.Error("[Config] failed to load config", "err", err)
		os.Exit(1)
	}

	emb := gigachat.New(
		c.TokenAuthURL,
		c.EmbeddingURL,
		c.TokenAuthKey,
	)
	res, err := emb.Get([]embedding.Chunk{"Hello", "DAmn"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	fmt.Println(len(res))
}
