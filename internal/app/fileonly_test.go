package app

import (
	"os"
	"path/filepath"
	"testing"

	"finallyshell/internal/config"
	"finallyshell/internal/sshtest"

	"github.com/egoist/mygo/ui"
)

// connectFileOnly opens a file-only session to srv's host and waits until
// it is connected and its home directory is listed.
func connectFileOnly(t *testing.T, a *App, p *fakePlatform, srv *sshtest.Server, home string) *Session {
	t.Helper()
	h := &config.Host{ID: "srv", Name: "files", Addr: srv.Host, Port: srv.Port, User: "tester", Auth: config.AuthPassword}
	a.hosts.Add(h)
	a.passwords[h.ID] = sshtest.Password
	a.openSessionFileOnly(h)
	s := a.current()
	if s == nil || s.Host != h {
		t.Fatal("no session opened")
	}
	if !s.isFileOnly() {
		t.Fatal("the session is not file-only")
	}
	if answerHostKeys(t, p, s); s.status != statusConnected {
		t.Fatal(describeError(s.err))
	}
	p.until(t, "the home directory", func() bool { return s.files.cwd != "" && !s.files.loading })
	if s.files.cwd != home {
		t.Errorf("cwd %q, want %q", s.files.cwd, home)
	}
	return s
}

// TestFileOnlySession connects to a test server without opening a shell:
// the session is connected, SFTP lists the home directory, the file panel
// is the tab's main area, and the server saw no shell.
func TestFileOnlySession(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := sshtest.StartSFTP(t, home)
	a, p := testApp(t)
	t.Cleanup(a.CloseAll)
	a.NewTab()
	tab := a.current()
	s := connectFileOnly(t, a, p, srv, home)
	if s != tab || len(a.sessions) != 1 {
		t.Fatalf("the new tab did not become the session: %d rows", len(a.sessions))
	}
	if srv.Shells() != 0 {
		t.Errorf("the server opened %d shells, want none", srv.Shells())
	}
	if s.term() != nil {
		t.Error("a file-only session has a terminal")
	}
	if len(s.panes.panes()) != 0 {
		t.Error("a file-only session has panes")
	}
	if s.sftp == nil {
		t.Fatal("a file-only session has no SFTP client")
	}
	// The file panel is the tab's main area: the table lists the file.
	tt := ui.NewTester(a.View, 1280, 820)
	p.tt = tt
	if _, ok := tt.Find("hello.txt"); !ok {
		t.Error("the file table does not show hello.txt")
	}
	// Reconnecting keeps the session file-only, and still opens no shell.
	a.reconnect(s)
	if answerHostKeys(t, p, s); s.status != statusConnected {
		t.Fatal(describeError(s.err))
	}
	if !s.isFileOnly() {
		t.Fatal("reconnecting lost the file-only mode")
	}
	p.until(t, "the home directory again", func() bool { return s.files.cwd != "" && !s.files.loading })
	if srv.Shells() != 0 {
		t.Errorf("reconnecting opened %d shells, want none", srv.Shells())
	}
}

// TestFileOnlySessionFromTree opens a file-only session from the
// connection tree's entry point, in a tab of its own.
func TestFileOnlySessionFromTree(t *testing.T) {
	home := t.TempDir()
	srv := sshtest.StartSFTP(t, home)
	a, p := testApp(t)
	t.Cleanup(a.CloseAll)
	h := &config.Host{ID: "srv", Name: "files", Addr: srv.Host, Port: srv.Port, User: "tester", Auth: config.AuthPassword}
	a.hosts.Add(h)
	a.passwords[h.ID] = sshtest.Password
	a.connectFromFileOnly(&managerState{}, h)
	s := a.current()
	if s == nil || s.Host != h || !s.isFileOnly() {
		t.Fatal("no file-only session opened from the tree")
	}
	if answerHostKeys(t, p, s); s.status != statusConnected {
		t.Fatal(describeError(s.err))
	}
	p.until(t, "the home directory", func() bool { return s.files.cwd != "" && !s.files.loading })
	if srv.Shells() != 0 {
		t.Errorf("the server opened %d shells, want none", srv.Shells())
	}
}
