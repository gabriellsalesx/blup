//go:build cgo && (linux || darwin)

package kokoro

/*
#cgo CFLAGS: -I${SRCDIR}/ort
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>
#include "onnxruntime_c_api.h"

// ONNX Runtime is loaded at run time from the downloaded library (it is not
// linked into Blup): only OrtGetApiBase is looked up, the rest comes from
// the OrtApi table.
static const OrtApi* api;

static const char* blup_ort_load(const char* path) {
	if (api) return NULL;
	void* lib = dlopen(path, RTLD_NOW | RTLD_LOCAL);
	if (!lib) return dlerror();
	typedef const OrtApiBase* (*GetBase)(void);
	GetBase get = (GetBase)dlsym(lib, "OrtGetApiBase");
	if (!get) return "OrtGetApiBase not found";
	api = get()->GetApi(ORT_API_VERSION);
	if (!api) return "ONNX Runtime too old";
	return NULL;
}

// blup_err turns a status into a message to free (NULL when ok).
static char* blup_err(OrtStatus* s) {
	if (!s) return NULL;
	char* m = strdup(api->GetErrorMessage(s));
	api->ReleaseStatus(s);
	return m;
}

static char* blup_open(const char* model, OrtEnv** env, OrtSession** session) {
	char* e = blup_err(api->CreateEnv(ORT_LOGGING_LEVEL_ERROR, "blup", env));
	if (e) return e;
	OrtSessionOptions* opts;
	if ((e = blup_err(api->CreateSessionOptions(&opts)))) return e;
	free(blup_err(api->SetSessionGraphOptimizationLevel(opts, ORT_ENABLE_ALL)));
	e = blup_err(api->CreateSession(*env, model, opts, session));
	api->ReleaseSessionOptions(opts);
	return e;
}

static void blup_close(OrtEnv* env, OrtSession* session) {
	if (session) api->ReleaseSession(session);
	if (env) api->ReleaseEnv(env);
}

// blup_run runs Kokoro: tokens (padded), the voice's style row and the
// speed in; the waveform (24 kHz float samples, malloc'd) out.
static char* blup_run(OrtSession* session, int64_t* ids, int64_t n, float* style, float speed, float** out, int64_t* outLen) {
	OrtMemoryInfo* mem;
	char* e = blup_err(api->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, &mem));
	if (e) return e;
	OrtValue* in[3] = {NULL, NULL, NULL};
	OrtValue* res = NULL;
	int64_t idShape[2] = {1, n}, styleShape[2] = {1, 256}, speedShape[1] = {1};
	if ((e = blup_err(api->CreateTensorWithDataAsOrtValue(mem, ids, n * sizeof(int64_t), idShape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &in[0])))) goto done;
	if ((e = blup_err(api->CreateTensorWithDataAsOrtValue(mem, style, 256 * sizeof(float), styleShape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &in[1])))) goto done;
	if ((e = blup_err(api->CreateTensorWithDataAsOrtValue(mem, &speed, sizeof(float), speedShape, 1, ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &in[2])))) goto done;
	const char* names[3] = {"input_ids", "style", "speed"};
	const char* outNames[1] = {"waveform"};
	if ((e = blup_err(api->Run(session, NULL, names, (const OrtValue* const*)in, 3, outNames, 1, &res)))) goto done;
	OrtTensorTypeAndShapeInfo* info;
	if ((e = blup_err(api->GetTensorTypeAndShape(res, &info)))) goto done;
	size_t count;
	e = blup_err(api->GetTensorShapeElementCount(info, &count));
	api->ReleaseTensorTypeAndShapeInfo(info);
	if (e) goto done;
	float* data;
	if ((e = blup_err(api->GetTensorMutableData(res, (void**)&data)))) goto done;
	*out = malloc(count * sizeof(float));
	memcpy(*out, data, count * sizeof(float));
	*outLen = (int64_t)count;
done:
	for (int i = 0; i < 3; i++) if (in[i]) api->ReleaseValue(in[i]);
	if (res) api->ReleaseValue(res);
	api->ReleaseMemoryInfo(mem);
	return e;
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// session is a loaded Kokoro model.
type session struct {
	env *C.OrtEnv
	s   *C.OrtSession
}

var loadMu sync.Mutex

// openSession loads ONNX Runtime from lib and the model from model.
func openSession(lib, model string) (*session, error) {
	loadMu.Lock()
	defer loadMu.Unlock()
	cl := C.CString(lib)
	defer C.free(unsafe.Pointer(cl))
	if msg := C.blup_ort_load(cl); msg != nil {
		return nil, errors.New("onnxruntime: " + C.GoString(msg))
	}
	cm := C.CString(model)
	defer C.free(unsafe.Pointer(cm))
	s := &session{}
	if msg := C.blup_open(cm, &s.env, &s.s); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		C.blup_close(s.env, s.s)
		return nil, errors.New("onnxruntime: " + C.GoString(msg))
	}
	return s, nil
}

// run turns token ids (already padded) into 24 kHz samples.
func (s *session) run(ids []int64, style []float32, speed float32) ([]float32, error) {
	if len(ids) == 0 || len(style) != 256 {
		return nil, errors.New("kokoro: bad input")
	}
	// The inputs are copied to C memory: ONNX Runtime keeps pointers to them.
	cids := (*C.int64_t)(C.malloc(C.size_t(len(ids) * 8)))
	defer C.free(unsafe.Pointer(cids))
	copy(unsafe.Slice((*int64)(unsafe.Pointer(cids)), len(ids)), ids)
	cstyle := (*C.float)(C.malloc(256 * 4))
	defer C.free(unsafe.Pointer(cstyle))
	copy(unsafe.Slice((*float32)(unsafe.Pointer(cstyle)), 256), style)
	var out *C.float
	var n C.int64_t
	if msg := C.blup_run(s.s, cids, C.int64_t(len(ids)), cstyle, C.float(speed), &out, &n); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return nil, errors.New("onnxruntime: " + C.GoString(msg))
	}
	defer C.free(unsafe.Pointer(out))
	return append([]float32(nil), unsafe.Slice((*float32)(unsafe.Pointer(out)), int(n))...), nil
}

func (s *session) close() { C.blup_close(s.env, s.s) }
