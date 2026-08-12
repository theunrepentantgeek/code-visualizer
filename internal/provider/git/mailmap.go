package git

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// mailmapEntry maps an old (email, name) pair to canonical (email, name).
// Both fields may be empty to mean "keep original".
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
//	Proper Name <proper@email.com>                        — rename; keep old email
//	<proper@email.com> <old@email.com>                    — re-email; keep name
//	Proper Name <proper@email.com> <old@email.com>        — rename + re-email by old email
//	Proper Name <proper@email.com> Old Name <old@email.com> — rename + re-email by old email+name
//
// Only old-email keying is implemented (old-name matching is not needed for
// the authorship metric use-case and is expensive to maintain separately).
func parseMailmap(r interface{ Read([]byte) (int, error) }) mailmap {
	mm := mailmap{}

	sc := bufio.NewScanner(r.(interface {
		Read([]byte) (int, error)
	}))

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
		if oldEmail == "" && properEmail == "" {
			continue
		}

		key := oldEmail
		if key == "" {
			key = properEmail
		}

		key = strings.ToLower(key)

		mm[key] = mailmapEntry{
			properEmail: properEmail,
			properName:  properName,
		}
	}

	return mm
}

// parseMailmapLine extracts (properName, properEmail, oldEmail) from one line.
// oldEmail is empty when the line only specifies a proper identity (no old email).
func parseMailmapLine(line string) (properName, properEmail, oldEmail string) {
	// Each <...> is an email field. The line has 1 or 2 angle-bracket groups.
	emails, remainder := extractEmails(line)

	switch len(emails) {
	case 1:
		// Form: "Proper Name <proper@email>" — only proper email, no old email.
		properEmail = emails[0]
		properName = strings.TrimSpace(remainder)
	case 2:
		// Could be:
		//   "<proper@email> <old@email>"
		//   "Proper Name <proper@email> <old@email>"
		//   "Proper Name <proper@email> Old Name <old@email>"
		//
		// emails[0] is always the PROPER email (leftmost <...>).
		// emails[1] is always the OLD email (rightmost <...>).
		properEmail = emails[0]
		oldEmail = emails[1]

		// The remainder (text between/around the angle brackets) may contain
		// the proper name (before the first <...>) and an old name (between
		// the two <...> groups), both of which we ignore for old-email keying.
		properName = strings.TrimSpace(textBefore(line, "<"))
	default:
		// Malformed or unsupported; skip.
	}

	return
}

// extractEmails returns all <...> contents found in line (in order) and the
// line with those angle-bracket groups removed.
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
		email := rest[open+1 : open+close]
		emails = append(emails, strings.TrimSpace(email))
		rest = rest[open+close+1:]
	}

	return
}

// textBefore returns the text in s before the first occurrence of sep.
func textBefore(s, sep string) string {
	idx := strings.Index(s, sep)
	if idx < 0 {
		return s
	}

	return s[:idx]
}

// apply returns the canonical (email, name) for (oldEmail, oldName) by
// looking up oldEmail in the map. If no entry is found, the originals
// are returned unchanged.
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
