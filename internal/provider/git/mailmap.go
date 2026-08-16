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
		addMailmapLine(mm, sc.Text())
	}

	if err := sc.Err(); err != nil {
		return mailmap{}
	}

	return mm
}

func addMailmapLine(mm mailmap, rawLine string) {
	line := strings.TrimSpace(rawLine)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	if idx := strings.Index(line, " #"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}

	properName, properEmail, oldEmail := parseMailmapLine(line)
	if properEmail == "" {
		return
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

// parseMailmapLine extracts (properName, properEmail, oldEmail) from one line.
// oldEmail is empty when the line only specifies a proper identity with no
// explicit old-email override.
func parseMailmapLine(line string) (properName, properEmail, oldEmail string) {
	emails, _ := extractEmails(line)
	if len(emails) == 0 {
		return "", "", ""
	}

	properName = strings.TrimSpace(textBefore(line, "<"))

	properEmail = emails[0]
	if len(emails) == 1 {
		return properName, properEmail, ""
	}

	return properName, properEmail, emails[1]
}

// extractEmails returns all <...> contents found in line (in order).
func extractEmails(line string) ([]string, string) {
	rest := line

	var emails []string

	var remainder strings.Builder

	for {
		open := strings.Index(rest, "<")
		if open < 0 {
			remainder.WriteString(rest)

			return emails, remainder.String()
		}

		emailAndRest := rest[open+1:]

		email, remaining, found := strings.Cut(emailAndRest, ">")
		if !found {
			remainder.WriteString(rest)

			return emails, remainder.String()
		}

		remainder.WriteString(rest[:open])

		emails = append(emails, strings.TrimSpace(email))
		rest = remaining
	}
}

// textBefore returns the text in s before the first occurrence of sep.
func textBefore(s, sep string) string {
	before, _, found := strings.Cut(s, sep)
	if found {
		return before
	}

	return s
}

// apply returns the canonical (email, name) for (oldEmail, oldName) by
// looking up oldEmail in the map. Originals are returned when no entry exists.
func (mm mailmap) apply(email, name string) (outEmail, outName string) {
	if len(mm) == 0 {
		return email, name
	}

	entry, ok := mm[strings.ToLower(email)]
	if !ok {
		return email, name
	}

	outEmail = email
	if entry.properEmail != "" {
		outEmail = entry.properEmail
	}

	outName = name
	if entry.properName != "" {
		outName = entry.properName
	}

	return outEmail, outName
}
