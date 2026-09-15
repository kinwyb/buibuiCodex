package cdm

import (
	"path/filepath"
	"testing"

	"github.com/kinwyb/buibuiCodex/core/db"
)

func Test_history(t *testing.T) {
	codexRoot := "/Users/wangyingbin/Downloads/kanflux/buibuiCodex/codex/root"
	sessionID := "wxcom:work_aibAnLYMK4TgdOkkhCCYZabAtOGE64ouXFV_007928"
	agent := "codex"
	sqlite, err := db.NewSQLiteStorage(filepath.Join("/Users/wangyingbin/Downloads/kanflux/buibuiCodex", "buibui.db"))
	if err != nil {
		panic(err)
	}
	dataStorage := db.NewData(sqlite)
	history := History{
		codexRoot: codexRoot,
		sess:      dataStorage.Session(),
		maxNTurn:  2,
	}
	res := history.History(sessionID, agent)
	t.Log(res)
}
