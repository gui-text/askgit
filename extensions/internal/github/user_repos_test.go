package github_test

import (
	"testing"

	"github.com/askgitdev/askgit/extensions/internal/tools"
)

func TestUserRepos(t *testing.T) {
	cleanup := newRecorder(t)
	defer cleanup()

	db := Connect(t, Memory)

	rows, err := db.Query("SELECT * FROM github_user_repos('josephjacks') LIMIT 10")
	if err != nil {
		t.Fatalf("failed to execute query: %v", err.Error())
	}
	defer rows.Close()

	colCount, content, err := tools.RowContent(rows)
	if err != nil {
		t.Fatalf("failed to retrieve row contents: %v", err.Error())
	}

	if expected := 30; colCount != expected {
		t.Fatalf("expected %d columns, got: %d", expected, colCount)
	}

	if expected := 10; len(content) != expected {
		t.Fatalf("expected %d rows, got: %d", expected, len(content))
	}
}
