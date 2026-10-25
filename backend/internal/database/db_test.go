package database

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestLogMigrationNotice(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	for _, notice := range []*pgconn.Notice{
		{Severity: "NOTICE", Message: "unrelated notice"},
		{Severity: "WARNING", Message: "unrelated warning"},
		{Severity: "WARNING", Message: "plugin URL migration: organization_id=org asset_id=asset attribute_key=plugin.igdb_games.games.url reason=invalid URL value; removed"},
	} {
		logMigrationNotice(nil, notice)
	}
	if strings.Count(output.String(), "\n") != 1 || !strings.Contains(output.String(), `"level":"WARN"`) || !strings.Contains(output.String(), "asset_id=asset") {
		t.Fatalf("unexpected migration logs: %s", output.String())
	}
}
