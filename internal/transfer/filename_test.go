package transfer

import (
	"errors"
	"strings"
	"testing"
)

// Detects: hostile names reaching the filesystem or the consent prompt (F-01, F-05).
func TestValidatePeerFileNameRejectsUnsafeNames(t *testing.T) {
	bad := map[string]string{
		"empty":             "",
		"too long":          strings.Repeat("a", MaxFileNameBytes+1),
		"invalid utf8":      "a\xffb",
		"NUL":               "a\x00b",
		"ESC sequence":      "report\x1b[2J.pdf",
		"carriage return":   "a\rb",
		"newline":           "a\nb",
		"bidi override":     "invoice‮gpj.exe",
		"zero width joiner": "a‍b",
		"line separator":    "a b",
	}
	for name, in := range bad {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePeerFileName(in); !errors.Is(err, ErrInvalidFileName) {
				t.Fatalf("ValidatePeerFileName(%q) = %v, want ErrInvalidFileName", in, err)
			}
		})
	}
	for _, ok := range []string{"report.pdf", "relatório final.docx", "日本語.txt", strings.Repeat("a", MaxFileNameBytes)} {
		if err := ValidatePeerFileName(ok); err != nil {
			t.Errorf("ValidatePeerFileName(%q) = %v, want nil", ok, err)
		}
	}
}

// Detects: path traversal, hidden files, reserved device names and NTFS streams surviving
// sanitisation. The result must always be one plain path component.
func TestSanitizeFileName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"report.pdf", "report.pdf"},
		{"../../etc/passwd", "passwd"},
		{`..\..\windows\system32\evil.dll`, "evil.dll"},
		{"/abs/path/file.txt", "file.txt"},
		{"dir/", "dir"},
		{".bashrc", "bashrc"},
		{"...hidden", "hidden"},
		{"name.txt.", "name.txt"},
		{"name.txt  ", "name.txt"},
		{"file.txt:evil", "file.txt_evil"},
		{`a<b>c"d|e?f*g`, "a_b_c_d_e_f_g"},
		{"CON", "_CON"},
		{"nul.txt", "_nul.txt"},
		{"com1", "_com1"},
		{"LPT9.log", "_LPT9.log"},
		{"console.txt", "console.txt"},
		{"a/b\\c/d.txt", "d.txt"},
	}
	for _, tc := range tests {
		got, err := SanitizeFileName(tc.in)
		if err != nil {
			t.Errorf("SanitizeFileName(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if strings.ContainsAny(got, `/\:`) || got == "." || got == ".." {
			t.Errorf("SanitizeFileName(%q) = %q is not a plain component", tc.in, got)
		}
	}
	for _, in := range []string{"", ".", "..", "../..", "/", "...", "a\x00b", "x‮y"} {
		if got, err := SanitizeFileName(in); err == nil {
			t.Errorf("SanitizeFileName(%q) = %q, want an error", in, got)
		}
	}
}

func FuzzSanitizeFileName(f *testing.F) {
	for _, s := range []string{"a", "../x", "CON", ".bashrc", "a:b", "x/y\\z", "‮"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		got, err := SanitizeFileName(in)
		if err != nil {
			return
		}
		if got == "" || got == "." || got == ".." || len(got) > MaxFileNameBytes ||
			strings.ContainsAny(got, "/\\:<>\"|?*\x00") || strings.HasPrefix(got, ".") ||
			strings.HasSuffix(got, ".") || strings.HasSuffix(got, " ") {
			t.Fatalf("SanitizeFileName(%q) = %q is unsafe", in, got)
		}
	})
}
