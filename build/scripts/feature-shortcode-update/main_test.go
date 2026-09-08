// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestVersionLTE(t *testing.T) {
	cases := []struct {
		name     string
		v        string
		target   string
		expected bool
	}{
		{"numeric vs string ordering", "1.9.0", "1.10.0", true}, // the bug this PR fixes
		{"equal versions", "1.61.0", "1.61.0", true},
		{"differing segment counts, v shorter", "1.61", "1.61.0", true},
		{"differing segment counts, v longer but equal prefix", "1.61.0", "1.61", true},
		{"v greater than target", "1.62.0", "1.61.0", false},
		{"v less than target", "1.5.0", "1.61.0", true},
		{"unparseable segment in v treated as 0", "1.x.0", "1.1.0", true},
		{"unparseable segment in target treated as 0", "1.1.0", "1.x.0", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := versionLTE(c.v, c.target); got != c.expected {
				t.Errorf("versionLTE(%q, %q) = %v, want %v", c.v, c.target, got, c.expected)
			}
		})
	}
}

// testTargetVersion is the release version used across all removeBlocks
// table tests below. It's fixed rather than a parameter because every
// case here is about block-handling behavior at a given version, not
// about varying the version itself — TestVersionLTE above already covers
// version-comparison edge cases directly.
const testTargetVersion = "1.61.0"

func runRemoveBlocks(t *testing.T, input, ext string) (string, bool) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(input))
	out, changed, err := removeBlocks(scanner, testTargetVersion, ext)
	if err != nil {
		t.Fatalf("removeBlocks returned error: %v", err)
	}
	return out, changed
}

func TestRemoveBlocksMarkdown(t *testing.T) {
	t.Run("resolved expiry block removed with content", func(t *testing.T) {
		in := "keep\n{{% feature expiryVersion=\"1.50.0\" %}}\ndrop me\n{{% /feature %}}\nkeep2\n"
		want := "keep\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("resolved publish block unwrapped, content kept", func(t *testing.T) {
		in := "keep\n{{% feature publishVersion=\"1.50.0\" %}}\nkeep this\n{{% /feature %}}\nkeep2\n"
		want := "keep\nkeep this\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("future-version expiry block left alone", func(t *testing.T) {
		in := "keep\n{{% feature expiryVersion=\"9.99.0\" %}}\nkeep this too\n{{% /feature %}}\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if changed || out != in {
			t.Errorf("got changed=%v out=%q, want changed=false out=%q", changed, out, in)
		}
	})

	t.Run("future-version publish block left alone", func(t *testing.T) {
		in := "keep\n{{% feature publishVersion=\"9.99.0\" %}}\nkeep this too\n{{% /feature %}}\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if changed || out != in {
			t.Errorf("got changed=%v out=%q, want changed=false out=%q", changed, out, in)
		}
	})

	t.Run("whole file is a single resolved expiry block", func(t *testing.T) {
		in := "{{% feature expiryVersion=\"1.50.0\" %}}\nall of it\n{{% /feature %}}\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != "" {
			t.Errorf("got changed=%v out=%q, want changed=true out=\"\"", changed, out)
		}
	})

	t.Run("single-line expiry block does not swallow following content", func(t *testing.T) {
		in := "a\n{{% feature expiryVersion=\"1.50.0\" %}}x{{% /feature %}}\nb\nc\n"
		want := "a\nb\nc\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("single-line publish block keeps its content", func(t *testing.T) {
		in := "before\n{{% feature publishVersion=\"1.5.0\" %}}text{{% /feature %}}\nafter\n"
		want := "before\ntext\nafter\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("angle-bracket delimiter style", func(t *testing.T) {
		in := "keep\n{{< feature expiryVersion=\"1.50.0\" >}}\ndrop me\n{{< /feature >}}\nkeep2\n"
		want := "keep\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("escaped shortcode form is not rewritten", func(t *testing.T) {
		in := "keep\n{{%/* feature expiryVersion=\"1.50.0\" */%}}\nkeep this\n{{%/* /feature */%}}\nkeep2\n"
		out, changed := runRemoveBlocks(t, in, mdExt)
		if changed || out != in {
			t.Errorf("escaped shortcode should be left untouched: got changed=%v out=%q", changed, out)
		}
	})
}

func TestRemoveBlocksHTML(t *testing.T) {
	t.Run("expiry removed on html", func(t *testing.T) {
		in := "<p>keep</p>\n{{% feature expiryVersion=\"1.50.0\" %}}\n<p>drop</p>\n{{% /feature %}}\n<p>keep2</p>\n"
		want := "<p>keep</p>\n<p>keep2</p>\n"
		out, changed := runRemoveBlocks(t, in, htmlExt)
		if !changed || out != want {
			t.Errorf("got changed=%v out=%q, want changed=true out=%q", changed, out, want)
		}
	})

	t.Run("publish wrapper preserved on html", func(t *testing.T) {
		// .html files never unwrap publishVersion - this is what keeps
		// generated files like agones_crd_api_reference.html stable and
		// `make test-gen-api-docs` green.
		in := "<p>keep</p>\n{{% feature publishVersion=\"1.50.0\" %}}\n<p>keep this</p>\n{{% /feature %}}\n<p>keep2</p>\n"
		out, changed := runRemoveBlocks(t, in, htmlExt)
		if changed || out != in {
			t.Errorf("publish block on .html should be untouched: got changed=%v out=%q", changed, out)
		}
	})
}
