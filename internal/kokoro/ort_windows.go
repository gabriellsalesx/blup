//go:build windows && amd64

package kokoro

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ONNX Runtime on Windows, without cgo (the Studio for Windows is built
// without it). The DLL is loaded at run time from the downloaded folder:
// only OrtGetApiBase is looked up, the rest comes from the OrtApi table,
// whose functions are called with syscall.SyscallN (64-bit Windows has a
// single C calling convention). Only pointers ONNX Runtime keeps past a
// call point to its own memory: the tensors are allocated by it.

var (
	loadMu sync.Mutex
	api    uintptr // const OrtApi*, once loaded
)

// vcRuntime are the Visual C++ runtime DLLs ONNX Runtime needs. The tools
// package carries them for the systems that lack them; they are loaded
// first, by full path, so ONNX Runtime uses these copies and not an older
// one that may be elsewhere on the system.
var vcRuntime = []string{"vcruntime140.dll", "vcruntime140_1.dll", "msvcp140.dll", "msvcp140_1.dll"}

func loadORT(lib string) error {
	if api != 0 {
		return nil
	}
	if lib == "" {
		return errors.New("no library given")
	}
	// lib is <dir>/ort/onnxruntime.dll; the tools are in <dir>/tools.
	tools := filepath.Join(filepath.Dir(filepath.Dir(lib)), "tools")
	for _, name := range vcRuntime {
		p := filepath.Join(tools, name)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if _, err := windows.LoadLibraryEx(p, 0, windows.LOAD_WITH_ALTERED_SEARCH_PATH); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	h, err := windows.LoadLibraryEx(lib, 0, windows.LOAD_WITH_ALTERED_SEARCH_PATH)
	if err != nil {
		return err
	}
	get, err := windows.GetProcAddress(h, "OrtGetApiBase")
	if err != nil {
		return errors.New("OrtGetApiBase not found")
	}
	base, _, _ := syscall.SyscallN(get)
	if base == 0 {
		return errors.New("OrtGetApiBase failed")
	}
	// OrtApiBase is {GetApi, GetVersionString}.
	a, _, _ := syscall.SyscallN(*(*uintptr)(ptr(base)), ortAPIVersion)
	if a == 0 {
		return errors.New("ONNX Runtime too old")
	}
	api = a
	return nil
}

// ptr turns an address ONNX Runtime gave (its memory, never Go's) into a
// pointer.
func ptr(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

// call runs function fn of the OrtApi table. Go pointers passed as
// uintptr stay alive and in place during the call (uintptrescapes).
//
//go:uintptrescapes
func call(fn uintptr, args ...uintptr) uintptr {
	f := *(*uintptr)(ptr(api + fn*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(f, args...)
	return r
}

// status turns an OrtStatus* into an error (nil when ok) and frees it.
func status(s uintptr) error {
	if s == 0 {
		return nil
	}
	msg := cString(call(ortGetErrorMessage, s))
	call(ortReleaseStatus, s)
	return errors.New("onnxruntime: " + msg)
}

func cString(p uintptr) string {
	if p == 0 {
		return ""
	}
	var b []byte
	for ; *(*byte)(ptr(p)) != 0; p++ {
		b = append(b, *(*byte)(ptr(p)))
	}
	return string(b)
}

func cStr(s string) *byte {
	b, _ := windows.BytePtrFromString(s)
	return b
}

// session is a loaded Kokoro model.
type session struct {
	env, s uintptr // OrtEnv*, OrtSession*
}

// openSession loads ONNX Runtime from lib and the model from model.
func openSession(lib, model string) (*session, error) {
	loadMu.Lock()
	defer loadMu.Unlock()
	if err := loadORT(lib); err != nil {
		return nil, errors.New("onnxruntime: " + err.Error())
	}
	path, err := windows.UTF16PtrFromString(model)
	if err != nil {
		return nil, err
	}
	s := &session{}
	if err := status(call(ortCreateEnv, ortLoggingLevelError, uintptr(unsafe.Pointer(cStr("blup"))), uintptr(unsafe.Pointer(&s.env)))); err != nil {
		return nil, err
	}
	var opts uintptr
	if err := status(call(ortCreateSessionOptions, uintptr(unsafe.Pointer(&opts)))); err != nil {
		s.close()
		return nil, err
	}
	status(call(ortSetSessionGraphOptimizationLvl, opts, ortEnableAll))
	err = status(call(ortCreateSession, s.env, uintptr(unsafe.Pointer(path)), opts, uintptr(unsafe.Pointer(&s.s))))
	call(ortReleaseSessionOptions, opts)
	if err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

// run turns token ids (already padded) into 24 kHz samples.
func (s *session) run(ids []int64, style []float32, speed float32) ([]float32, error) {
	if len(ids) == 0 || len(style) != 256 {
		return nil, errors.New("kokoro: bad input")
	}
	var alloc uintptr // ONNX Runtime's default allocator: not to be freed
	if err := status(call(ortGetAllocatorWithDefaultOptions, uintptr(unsafe.Pointer(&alloc)))); err != nil {
		return nil, err
	}
	var in [3]uintptr
	defer func() {
		for _, v := range in {
			if v != 0 {
				call(ortReleaseValue, v)
			}
		}
	}()
	// tensor makes input i, allocated by ONNX Runtime, and returns its data.
	tensor := func(i int, shape []int64, typ uintptr) (unsafe.Pointer, error) {
		if err := status(call(ortCreateTensorAsOrtValue, alloc, uintptr(unsafe.Pointer(&shape[0])), uintptr(len(shape)), typ, uintptr(unsafe.Pointer(&in[i])))); err != nil {
			return nil, err
		}
		var data uintptr
		if err := status(call(ortGetTensorMutableData, in[i], uintptr(unsafe.Pointer(&data)))); err != nil {
			return nil, err
		}
		return ptr(data), nil
	}
	p, err := tensor(0, []int64{1, int64(len(ids))}, ortTensorInt64)
	if err != nil {
		return nil, err
	}
	copy(unsafe.Slice((*int64)(p), len(ids)), ids)
	if p, err = tensor(1, []int64{1, 256}, ortTensorFloat); err != nil {
		return nil, err
	}
	copy(unsafe.Slice((*float32)(p), 256), style)
	if p, err = tensor(2, []int64{1}, ortTensorFloat); err != nil {
		return nil, err
	}
	*(*float32)(p) = speed

	names := []*byte{cStr("input_ids"), cStr("style"), cStr("speed")}
	outNames := []*byte{cStr("waveform")}
	var res uintptr
	if err := status(call(ortRun, s.s, 0, uintptr(unsafe.Pointer(&names[0])), uintptr(unsafe.Pointer(&in[0])), 3,
		uintptr(unsafe.Pointer(&outNames[0])), 1, uintptr(unsafe.Pointer(&res)))); err != nil {
		return nil, err
	}
	defer call(ortReleaseValue, res)
	var info uintptr
	if err := status(call(ortGetTensorTypeAndShape, res, uintptr(unsafe.Pointer(&info)))); err != nil {
		return nil, err
	}
	var count uintptr
	err = status(call(ortGetTensorShapeElementCount, info, uintptr(unsafe.Pointer(&count))))
	call(ortReleaseTensorTypeAndShapeInfo, info)
	if err != nil {
		return nil, err
	}
	var data uintptr
	if err := status(call(ortGetTensorMutableData, res, uintptr(unsafe.Pointer(&data)))); err != nil {
		return nil, err
	}
	return append([]float32(nil), unsafe.Slice((*float32)(ptr(data)), int(count))...), nil
}

func (s *session) close() {
	if s.s != 0 {
		call(ortReleaseSession, s.s)
	}
	if s.env != 0 {
		call(ortReleaseEnv, s.env)
	}
}
