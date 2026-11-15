package version

import "testing"

func TestResolve(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		injected string
		module   string
		want     string
	}{
		{name: "local build", injected: developmentVersion, module: "(devel)", want: "dev"},
		{name: "missing metadata", want: "dev"},
		{name: "release", injected: "0.1.0", module: "(devel)", want: "v0.1.0"},
		{name: "prefixed release", injected: "v0.1.0", want: "v0.1.0"},
		{name: "snapshot", injected: "0.1.0-SNAPSHOT-abc123", want: "v0.1.0-SNAPSHOT-abc123"},
		{name: "go install tag", injected: developmentVersion, module: "v0.2.0", want: "v0.2.0"},
		{name: "go install commit", injected: developmentVersion, module: "v0.0.0-20261115000000-abcdef123456", want: "v0.0.0-20261115000000-abcdef123456"},
		{name: "local VCS build", injected: developmentVersion, module: "v0.0.0-20261115000000-abcdef123456+dirty", want: "v0.0.0-20261115000000-abcdef123456+dirty"},
		{name: "injected version wins", injected: "0.3.0", module: "v0.2.0", want: "v0.3.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := resolve(test.injected, test.module); got != test.want {
				t.Fatalf("resolve(%q, %q) = %q, want %q", test.injected, test.module, got, test.want)
			}
		})
	}
}
