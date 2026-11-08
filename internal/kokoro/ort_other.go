//go:build !(cgo && (linux || darwin)) && !(windows && amd64)

package kokoro

// Without cgo on Linux and macOS (the blup CLI), or on a system with no
// loader yet, the local voice is not available.

type session struct{}

func openSession(lib, model string) (*session, error) { return nil, ErrUnsupported }

func (s *session) run(ids []int64, style []float32, speed float32) ([]float32, error) {
	return nil, ErrUnsupported
}

func (s *session) close() {}
