package util

import (
	"fmt"
	"log"
	"log/slog"
)

func Warning(err error, format string, args ...any) {
	if err != nil {
		log.Printf("%s: %s", fmt.Sprintf(format, args...), err)
	}
}

func Fail(err error, format string, args ...any) {
	if err != nil {
		log.Fatalf("%s: %s", fmt.Sprintf(format, args...), err)
	}
}

func Annotate(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func Warn(err error, msg string, args ...any) {
	if err == nil {
		return
	}
	slog.Warn(msg,
		append(args,
			"error", err.Error(),
		)...,
	)
}
