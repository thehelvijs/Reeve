package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// maxSlugLen keeps a slug short enough to type and to read in a URL bar.
const maxSlugLen = 60

// Slugify renders a name as a URL path segment: lowercase, runs of anything
// that is not a letter or digit collapsed to a single dash, no leading or
// trailing dash. It returns "" for a name with nothing usable in it, which the
// caller resolves to a fallback rather than storing an empty slug.
func Slugify(name string) string {
	var b strings.Builder
	dashPending := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dashPending && b.Len() > 0 {
				b.WriteByte('-')
			}
			dashPending = false
			b.WriteRune(r)
			continue
		}
		dashPending = true
	}
	slug := b.String()
	if len(slug) > maxSlugLen {
		slug = strings.TrimRight(slug[:maxSlugLen], "-")
	}
	return slug
}

// UniqueToolSlug returns base, or base-2, base-3, … until one is free. excludeID
// keeps a tool from colliding with itself when it is renamed.
func (db *DB) UniqueToolSlug(base, excludeID string) (string, error) {
	if base == "" {
		base = "tool"
	}
	candidate := base
	for n := 2; ; n++ {
		taken, err := db.toolSlugTaken(candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
}

func (db *DB) toolSlugTaken(slug, excludeID string) (bool, error) {
	var id string
	err := db.sql.QueryRow(`SELECT id FROM tools WHERE slug = ?`, slug).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return id != excludeID, nil
}

// GetToolBySlug returns the tool a /go/<slug> URL names, or ErrNotFound.
func (db *DB) GetToolBySlug(slug string) (Tool, error) {
	rows, err := db.sql.Query(toolSelect+` WHERE slug = ?`, slug)
	if err != nil {
		return Tool{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Tool{}, ErrNotFound
	}
	return db.scanTool(rows)
}
