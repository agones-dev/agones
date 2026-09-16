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
		wantErr  bool
	}{
		{"numeric vs string ordering", "1.9.0", "1.10.0", true, false}, // the bug this PR fixes
		{"equal versions", "1.61.0", "1.61.0", true, false},
		{"differing segment counts, v shorter", "1.61", "1.61.0", true, false},
		{"differing segment counts, v longer but equal prefix", "1.61.0", "1.61", true, false},
		{"v greater than target", "1.62.0", "1.61.0", false, false},
		{"v less than target", "1.5.0", "1.61.0", true, false},
		// These two used to assert that an unparseable segment silently
		// coerced to 0. That's the exact bug flagged in review: a malformed
		// or typo'd shortcode version (e.g. "1.x.0") would then compare as
		// <= almost anything and its block would be deleted. versionLTE now
		// fails closed and returns an error instead.
		{"unparseable segment in v returns error", "1.x.0", "1.1.0", false, true},
		{"unparseable segment in target returns error", "1.1.0", "1.x.0", false, true},
		{"v-prefixed version returns error", "v1.61.0", "1.61.0", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := versionLTE(c.v, c.target)
			if c.wantErr {
				if err == nil {
					t.Fatalf("versionLTE(%q, %q) expected an error, got nil (result=%v)", c.v, c.target, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("versionLTE(%q, %q) unexpected error: %v", c.v, c.target, err)
			}
			if got != c.expected {
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

	// Regression test for the "high" review comment: when the expiry
	// block's open and close tags share a line with surrounding prose,
	// only the shortcode span and its content should be dropped — the
	// prefix and suffix text must survive.
	t.Run("single-line expiry block with surrounding prose keeps prefix and suffix", func(t *testing.T) {
		in := "before\nprefix {{% feature expiryVersion=\"1.50.0\" %}}drop{{% /feature %}} suffix\nafter\n"
		want := "before\nprefix  suffix\nafter\n"
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

	// Regression test for the "high" review comment: when the publish
	// block's *closing* tag shares a line with content (open tag on its
	// own line, close tag on a later line with trailing/leading content),
	// only the closing tag should be stripped — the content on that line
	// must be kept, not dropped along with the whole line.
	t.Run("publish block: content shares line with close tag is kept", func(t *testing.T) {
		in := "before\n{{% feature publishVersion=\"1.50.0\" %}}\nkeep this{{% /feature %}}\nafter\n"
		want := "before\nkeep this\nafter\n"
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

	t.Run("malformed expiry version aborts with error", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader(
			"keep\n{{% feature expiryVersion=\"1.x.0\" %}}\ndrop me\n{{% /feature %}}\nkeep2\n"))
		if _, _, err := removeBlocks(scanner, testTargetVersion, mdExt); err == nil {
			t.Error("expected removeBlocks to return an error for a malformed expiryVersion, got nil")
		}
	})

	t.Run("malformed publish version aborts with error", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader(
			"keep\n{{% feature publishVersion=\"1.x.0\" %}}\nkeep this\n{{% /feature %}}\nkeep2\n"))
		if _, _, err := removeBlocks(scanner, testTargetVersion, mdExt); err == nil {
			t.Error("expected removeBlocks to return an error for a malformed publishVersion, got nil")
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
