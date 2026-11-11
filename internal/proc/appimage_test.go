package proc

import (
	"slices"
	"testing"
)

func TestOutsideAppImage(t *testing.T) {
	env := OutsideAppImage([]string{"HOME=/home/a", "APPDIR=/tmp/.mount_x", "LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib", "GST_PLUGIN_PATH_1_0=/x", "APPIMAGE=/a.AppImage", "WEBKIT_INJECTED_BUNDLE_PATH=/y", "PATH=/tmp/.mount_x/usr/bin:/usr/bin", "XDG_DATA_DIRS=/tmp/.mount_x/usr/share:/usr/share"})
	if !slices.Equal(env, []string{"HOME=/home/a", "PATH=/usr/bin", "XDG_DATA_DIRS=/usr/share"}) {
		t.Fatalf("got %v", env)
	}
	// The person's own library folder stays; a folder that only starts
	// like the mount's name is not inside it.
	env = OutsideAppImage([]string{"APPDIR=/tmp/.mount_x", "LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib:/opt/cuda/lib64:/tmp/.mount_xy/lib"})
	if !slices.Equal(env, []string{"LD_LIBRARY_PATH=/opt/cuda/lib64:/tmp/.mount_xy/lib"}) {
		t.Fatalf("got %v", env)
	}
	// Outside an AppImage nothing changes.
	in := []string{"HOME=/home/a", "LD_LIBRARY_PATH=/opt/lib", "GST_DEBUG=1"}
	if got := OutsideAppImage(in); !slices.Equal(got, in) {
		t.Fatalf("got %v", got)
	}
}
