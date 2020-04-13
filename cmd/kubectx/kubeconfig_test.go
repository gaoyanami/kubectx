package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)


func withTestVar(key, value string) func() {
	// TODO(ahmetb) this method is currently duplicated
	// consider extracting to internal/testutil or something

	orig, ok := os.LookupEnv(key)
	os.Setenv(key, value)
	return func() {
		if ok {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	}
}

func Test_homeDir(t *testing.T) {
	type env struct{ k, v string }
	cases := []struct {
		name string
		envs []env
		want string
	}{
		{
			name: "XDG_CACHE_HOME precedence",
			envs: []env{
				{"XDG_CACHE_HOME", "xdg"},
				{"HOME", "home"},
			},
			want: "xdg",
		},
		{
			name: "HOME over USERPROFILE",
			envs: []env{
				{"HOME", "home"},
				{"USERPROFILE", "up"},
			},
			want: "home",
		},
		{
			name: "only USERPROFILE available",
			envs: []env{
				{"XDG_CACHE_HOME", ""},
				{"HOME", ""},
				{"USERPROFILE", "up"},
			},
			want: "up",
		},
		{
			name: "none available",
			envs: []env{
				{"XDG_CACHE_HOME", ""},
				{"HOME", ""},
				{"USERPROFILE", ""},
			},
			want: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(tt *testing.T) {
			var unsets []func()
			for _, e := range c.envs {
				unsets = append(unsets, withTestVar(e.k, e.v))
			}

			got := homeDir()
			if got != c.want {
				t.Errorf("expected:%q got:%q", c.want, got)
			}
			for _, u := range unsets {
				u()
			}
		})
	}
}

func Test_kubeconfigPath(t *testing.T) {
	defer withTestVar("HOME", "/x/y/z")()

	expected := filepath.FromSlash("/x/y/z/.kube/config")
	got, err := kubeconfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("got=%q expected=%q", got, expected)
	}
}

func Test_kubeconfigPath_noEnvVars(t *testing.T) {
	defer withTestVar("XDG_CACHE_HOME", "")()
	defer withTestVar("HOME", "")()
	defer withTestVar("USERPROFILE", "")()

	_, err := kubeconfigPath()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func Test_kubeconfigPath_envOvveride(t *testing.T) {
	defer withTestVar("KUBECONFIG", "foo")()

	v, err := kubeconfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if expected := "foo"; v != expected {
		t.Fatalf("expected=%q, got=%q", expected, v)
	}
}

func Test_kubeconfigPath_envOvverideDoesNotSupportPathSeparator(t *testing.T) {
	path := strings.Join([]string{"file1", "file2"}, string(os.PathListSeparator))
	defer withTestVar("KUBECONFIG", path)()

	_, err := kubeconfigPath()
	if err == nil {
		t.Fatal("expected error")
	}
}

func testfile(t *testing.T, contents string) (path string, cleanup func()) {
	t.Helper()

	f, err := ioutil.TempFile(os.TempDir(), "test-file")
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	path = f.Name()
	if _, err := f.Write([]byte(contents)); err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	return path, func() {
		f.Close()
		os.Remove(path)
	}
}
