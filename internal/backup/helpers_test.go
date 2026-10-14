package backup

import (
	"io"
	"log/slog"
)

func slogText(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}
