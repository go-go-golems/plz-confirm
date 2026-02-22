package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-go-golems/plz-confirm/internal/server"
	"github.com/go-go-golems/plz-confirm/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	st := store.New()
	srv := server.New(st)
	if err := srv.ListenAndServe(ctx, server.Options{Addr: ":3900"}); err != nil {
		log.Fatal(err)
	}
}
