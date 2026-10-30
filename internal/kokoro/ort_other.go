//go:build !(cgo && (linux || darwin))

package kokoro

// Without cgo (the blup CLI) or on Windows (still to come: there the
// library is loaded without cgo), the local voice is not available.

type session struct{}

func openSession(lib, model string) (*session, error) { return nil, ErrUnsupported }

func (s *session) run(ids []int64, style []float32, speed float32) ([]float32, error) {
	return nil, ErrUnsupported
}

func (s *session) close() {}
