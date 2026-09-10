package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTargetRevision(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec map[string]any
		want string
	}{{
		name: "default branch is not shown",
		spec: map[string]any{"source": map[string]any{"targetRevision": "main"}},
		want: "",
	}, {
		name: "master is not shown",
		spec: map[string]any{"source": map[string]any{"targetRevision": "master"}},
		want: "",
	}, {
		name: "HEAD is not shown",
		spec: map[string]any{"source": map[string]any{"targetRevision": "HEAD"}},
		want: "",
	}, {
		name: "unset targetRevision is not shown",
		spec: map[string]any{"source": map[string]any{"repoURL": "https://example.com/x.git"}},
		want: "",
	}, {
		name: "branch is shown",
		spec: map[string]any{"source": map[string]any{"targetRevision": "fix-acme-solver"}},
		want: "fix-acme-solver",
	}, {
		name: "commit hash is shown",
		spec: map[string]any{"source": map[string]any{"targetRevision": "9f2c1ab"}},
		want: "9f2c1ab",
	}, {
		name: "multi-source: all defaults are not shown",
		spec: map[string]any{"sources": []any{
			map[string]any{"targetRevision": "main"},
			map[string]any{"targetRevision": "HEAD"},
		}},
		want: "",
	}, {
		name: "multi-source: only the non-default is shown",
		spec: map[string]any{"sources": []any{
			map[string]any{"targetRevision": "main"},
			map[string]any{"targetRevision": "v2.1.0"},
		}},
		want: "v2.1.0",
	}, {
		name: "multi-source: duplicates collapse",
		spec: map[string]any{"sources": []any{
			map[string]any{"targetRevision": "release-1.4"},
			map[string]any{"targetRevision": "release-1.4"},
		}},
		want: "release-1.4",
	}, {
		name: "multi-source: distinct revisions are joined",
		spec: map[string]any{"sources": []any{
			map[string]any{"targetRevision": "release-1.4"},
			map[string]any{"targetRevision": "v2.1.0"},
		}},
		want: "release-1.4, v2.1.0",
	}, {
		name: "sources wins over source when both are set",
		spec: map[string]any{
			"source":  map[string]any{"targetRevision": "stale-branch"},
			"sources": []any{map[string]any{"targetRevision": "v2.1.0"}},
		},
		want: "v2.1.0",
	}, {
		name: "no source at all",
		spec: map[string]any{"project": "default"},
		want: "",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			u := &unstructured.Unstructured{Object: map[string]any{"spec": tc.spec}}
			if got := targetRevision(u); got != tc.want {
				t.Errorf("targetRevision() = %q, want %q", got, tc.want)
			}
		})
	}
}
