package transfer

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxFileNameBytes is the longest file name accepted from a peer (the common filesystem limit).
const MaxFileNameBytes = 255

// ErrInvalidFileName is returned when a peer announces a file name that cannot be shown to the
// user or stored safely.
var ErrInvalidFileName = errors.New("invalid file name")

// ValidatePeerFileName checks a name announced by a remote peer. The name is attacker controlled
// and is shown in the consent prompt, so besides length and UTF-8 validity it rejects control
// (Cc), format (Cf, e.g. bidi overrides) and line/paragraph separator characters: they allow
// terminal-escape injection and extension spoofing ("invoice‮gpj.exe").
func ValidatePeerFileName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("%w: empty", ErrInvalidFileName)
	case len(name) > MaxFileNameBytes:
		return fmt.Errorf("%w: %d bytes (max %d)", ErrInvalidFileName, len(name), MaxFileNameBytes)
	case !utf8.ValidString(name):
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidFileName)
	}
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
			return fmt.Errorf("%w: contains control or formatting character U+%04X", ErrInvalidFileName, r)
		}
	}
	return nil
}

// windowsReserved are device names that Windows resolves regardless of extension ("CON.txt").
var windowsReserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

func isReservedDeviceName(stem string) bool {
	s := strings.ToUpper(stem)
	if windowsReserved[s] {
		return true
	}
	if len(s) == 4 && (strings.HasPrefix(s, "COM") || strings.HasPrefix(s, "LPT")) && s[3] >= '1' && s[3] <= '9' {
		return true
	}
	return false
}

// SanitizeFileName turns a peer-announced name into a single, portable path component that is
// safe to create inside the download directory. It never returns a path: separators of both
// platforms are cut (the last component wins), "." and ".." are rejected, characters that are
// illegal on Windows (including the NTFS stream separator ':') become '_', trailing dots and
// spaces are dropped, leading dots are removed so a peer cannot create dotfiles such as
// ".bashrc", and reserved Windows device names get a '_' prefix.
func SanitizeFileName(name string) (string, error) {
	if err := ValidatePeerFileName(name); err != nil {
		return "", err
	}
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "", fmt.Errorf("%w: no file name component", ErrInvalidFileName)
	}
	base := parts[len(parts)-1]
	if base == "." || base == ".." {
		return "", fmt.Errorf("%w: %q", ErrInvalidFileName, base)
	}
	base = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"|?*`, r) {
			return '_'
		}
		return r
	}, base)
	base = strings.TrimLeft(base, ". ")
	base = strings.TrimRight(base, ". ")
	if base == "" {
		return "", fmt.Errorf("%w: nothing left after sanitising", ErrInvalidFileName)
	}
	stem := base
	if i := strings.IndexByte(base, '.'); i >= 0 {
		stem = base[:i]
	}
	if isReservedDeviceName(strings.TrimRight(stem, " ")) {
		base = "_" + base
	}
	if len(base) > MaxFileNameBytes {
		return "", fmt.Errorf("%w: too long after sanitising", ErrInvalidFileName)
	}
	return base, nil
}
