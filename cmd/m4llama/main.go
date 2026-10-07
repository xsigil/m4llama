package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"m4llama/internal/infrastructure/sqlite3"
	"m4llama/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("[*] Booting m4llama...")

	db, err := sqlite3.NewDB(ctx, "app.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// インフラの初期化
	txManager := sqlite3.NewTxManager(db)
	sampleRepo := sqlite3.NewSampleRepository(db)

	// Usecase の初期化と実行
	appUsecase := usecase.NewSampleUsecase(sampleRepo, txManager)
	if err := appUsecase.Execute(ctx, "sample-1", "Initial Value"); err != nil {
		fmt.Fprintf(os.Stderr, "usecase error: %v\n", err)
	}

	<-ctx.Done()
	fmt.Println("\n[*] Gracefully shutting down...")
}
