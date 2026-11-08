//go:build cgo && (linux || darwin)

package kokoro

import "testing"

// The loader without cgo (Windows) calls ONNX Runtime by the positions in
// ort_api.go: they must match the header.
func TestORTTableMatchesHeader(t *testing.T) {
	h := headerAPI()
	want := map[string]uintptr{
		"GetErrorMessage":                  ortGetErrorMessage,
		"CreateEnv":                        ortCreateEnv,
		"CreateSession":                    ortCreateSession,
		"Run":                              ortRun,
		"CreateSessionOptions":             ortCreateSessionOptions,
		"SetSessionGraphOptimizationLevel": ortSetSessionGraphOptimizationLvl,
		"CreateTensorAsOrtValue":           ortCreateTensorAsOrtValue,
		"GetTensorMutableData":             ortGetTensorMutableData,
		"GetTensorShapeElementCount":       ortGetTensorShapeElementCount,
		"GetTensorTypeAndShape":            ortGetTensorTypeAndShape,
		"GetAllocatorWithDefaultOptions":   ortGetAllocatorWithDefaultOptions,
		"ReleaseEnv":                       ortReleaseEnv,
		"ReleaseStatus":                    ortReleaseStatus,
		"ReleaseSession":                   ortReleaseSession,
		"ReleaseValue":                     ortReleaseValue,
		"ReleaseTensorTypeAndShapeInfo":    ortReleaseTensorTypeAndShapeInfo,
		"ReleaseSessionOptions":            ortReleaseSessionOptions,
		"ORT_LOGGING_LEVEL_ERROR":          ortLoggingLevelError,
		"ORT_ENABLE_ALL":                   ortEnableAll,
		"FLOAT":                            ortTensorFloat,
		"INT64":                            ortTensorInt64,
	}
	for name, v := range want {
		if h[name] != v {
			t.Errorf("%s: header says %d, ort_api.go %d", name, h[name], v)
		}
	}
	if h["ORT_API_VERSION"] < ortAPIVersion {
		t.Errorf("the header (API %d) is older than the version asked for (%d)", h["ORT_API_VERSION"], ortAPIVersion)
	}
}
