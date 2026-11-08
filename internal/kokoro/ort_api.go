package kokoro

// ortAPIVersion is the ONNX Runtime API version Blup asks for: the one of
// the oldest runtime it downloads (1.23, the last with an Intel Mac build).
// Everything Blup calls exists since the first versions, and a newer
// runtime still serves an older API version.
const ortAPIVersion = 23

// The positions of the functions Blup calls in ONNX Runtime's OrtApi table
// (onnxruntime_c_api.h), for the loader without cgo (Windows). The table
// only grows at the end, so they hold for every version. A test checks
// them against the header with cgo.
const (
	ortGetErrorMessage                = 2
	ortCreateEnv                      = 3
	ortCreateSession                  = 7
	ortRun                            = 9
	ortCreateSessionOptions           = 10
	ortSetSessionGraphOptimizationLvl = 23
	ortCreateTensorAsOrtValue         = 48
	ortGetTensorMutableData           = 51
	ortGetTensorShapeElementCount     = 64
	ortGetTensorTypeAndShape          = 65
	ortGetAllocatorWithDefaultOptions = 78
	ortReleaseEnv                     = 92
	ortReleaseStatus                  = 93
	ortReleaseSession                 = 95
	ortReleaseValue                   = 96
	ortReleaseTensorTypeAndShapeInfo  = 99
	ortReleaseSessionOptions          = 100
)

// Values of ONNX Runtime's enums that Blup passes.
const (
	ortLoggingLevelError = 3
	ortEnableAll         = 99
	ortTensorFloat       = 1
	ortTensorInt64       = 7
)
