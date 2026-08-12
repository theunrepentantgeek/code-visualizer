package git

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// mailmapEntry maps an old email to canonical (email, name).
type mailmapEntry struct {
	properEmail string
	properName  string
}

// mailmap is a lookup table built from a .mailmap file.
// Key is the old author email (lower-cased for case-insensitive matching).
type mailmap map[string]mailmapEntry

// loadMailmap reads <repoRoot>/.mailmap and returns a lookup table.
// If the file does not exist or cannot be read, an empty map is returned.
func loadMailmap(repoRoot string) mailmap {
	path := filepath.Join(repoRoot, ".mailmap")

	f, err := os.Open(path)
	if err != nil {
		return mailmap{}
	}

	defer f.Close()

	return parseMailmap(f)
}

// parseMailmap parses a .mailmap reader into a lookup table keyed by old email.
// Supported line forms (# comments and blank lines are skipped):
//
//	Proper Name <proper@email.com>                          — rename; key = proper email
//	<proper@email.com> <old@email.com>                      — re-email; keep name
//	Proper Name <proper@email.com> <old@email.com>          — rename + re-email by old email
//	Proper Name <proper@email.com> Old Name <old@email.com> — rename + re-email by old email
//
// Only old-email keying is implemented; old-name disambiguation is not needed
// for the authorship metric use-case.
func parseMailmap(r io.Reader) mailmap {
	mm := mailmap{}
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip inline comments.
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		properName, properEmail, oldEmail := parseMailmapLine(line)
		if properEmail == "" {
			continue
		}

		key := oldEmail
		if key == "" {
			key = properEmail
		}

		mm[strings.ToLower(key)] = mailmapEntry{
			properEmail: properEmail,
			properName:  properName,
		}
	}

	return mm
}

// parseMailmapLine extracts (properName, properEmail, oldEmail) from one line.
// oldEmail is empty when the line only specifies a proper identity with no
// explicit old-email override.
func parseMailmapLine(line string) (properName, properEmail, oldEmail string) {
	emails, _ := extractEmails(line)

	switch len(emails) {
	case 1:
		// "Proper Name <proper@email>" — proper email only; key = proper email.
		properEmail = emails[0]
		properName = strings.TrimSpace(textBefore(line, "<"))
	case 2:
		// "<proper@email> <old@email>" or "Name <proper@email> <old@email>" or
		// "Name <proper@email> OldName <old@email>".
		properEmail = emails[0]
		oldEmail = emails[1]
		properName = strings.TrimSpace(textBefore(line, "<"))
	}

	return
}

// extractEmails returns all <...> contents found in line (in order).
func extractEmails(line string) (emails []string, remainder string) {
	rest := line

	for {
		open := strings.Index(rest, "<")
		if open < 0 {
			remainder += rest
			break
		}

		close := strings.Index(rest[open:], ">")
		if close < 0 {
			remainder += rest
			break
		}

		remainder += rest[:open]
		emails = append(emails, strings.TrimSpace(rest[open+1:open+close]))
		rest = rest[open+close+1:]
	}

	return
}

// textBefore returns the text in s before the first occurrence of sep.
func textBefore(s, sep string) string {
	if idx := strings.Index(s, sep); idx >= 0 {
		return s[:idx]
	}

	return s
}

// apply returns the canonical (email, name) for (oldEmail, oldName) by
// looking up oldEmail in the map. Originals are returned when no entry exists.
func (mm mailmap) apply(email, name string) (string, string) {
	if len(mm) == 0 {
		return email, name
	}

	entry, ok := mm[strings.ToLower(email)]
	if !ok {
		return email, name
	}

	outEmail := email
	if entry.properEmail != "" {
		outEmail = entry.properEmail
	}

	outName := name
	if entry.properName != "" {
		outName = entry.properName
	}

	return outEmail, outName
}
