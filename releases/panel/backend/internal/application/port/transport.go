package port

import "github.com/rs/zerolog"

// Logger — структурный логгер (адаптер вокруг zerolog).
type Logger interface {
	Z() *zerolog.Logger
}
