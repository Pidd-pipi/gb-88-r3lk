package service

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestAuthServiceRegister(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := NewAuthService(newTestConfig(), repo, discardLogger())

	tests := []struct {
		name        string
		username    string
		password    string
		wantRole    string
		wantErr     bool
		wantErrCode int
	}{
		{name: "success creates dev account", username: "alice", password: "alice123", wantRole: RoleDev},
		{name: "duplicate username conflict", username: "alice", password: "alice123", wantErr: true, wantErrCode: constants.CodeConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, token, err := svc.Register(tt.username, tt.password)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Register(%q) expected error", tt.username)
				}
				var appErr *constants.AppError
				if tt.wantErrCode != 0 && errors.As(err, &appErr) {
					if appErr.Code != tt.wantErrCode {
						t.Fatalf("Register(%q) code = %d, want %d", tt.username, appErr.Code, tt.wantErrCode)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Register(%q) unexpected error: %v", tt.username, err)
			}
			if user == nil || user.ID == 0 || user.Role != tt.wantRole {
				t.Fatalf("Register(%q) user = %+v, want role %s", tt.username, user, tt.wantRole)
			}
			if token == "" {
				t.Fatalf("Register(%q) token is empty", tt.username)
			}
		})
	}
}

func TestAuthServiceLogin(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := NewAuthService(newTestConfig(), repo, discardLogger())
	created, _, err := svc.Register("bob", "bob123456")
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}

	tests := []struct {
		name        string
		username    string
		password    string
		wantUserID  uint
		wantErr     bool
		wantErrCode int
	}{
		{name: "valid credentials", username: "bob", password: "bob123456", wantUserID: created.ID},
		{name: "wrong password", username: "bob", password: "wrong-password", wantErr: true, wantErrCode: constants.CodeUnauthorized},
		{name: "unknown username", username: "nobody", password: "whatever1", wantErr: true, wantErrCode: constants.CodeUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, token, err := svc.Login(tt.username, tt.password)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Login(%q) expected error", tt.username)
				}
				var appErr *constants.AppError
				if errors.As(err, &appErr) && appErr.Code != tt.wantErrCode {
					t.Fatalf("Login(%q) code = %d, want %d", tt.username, appErr.Code, tt.wantErrCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login(%q) unexpected error: %v", tt.username, err)
			}
			if user.ID != tt.wantUserID || token == "" {
				t.Fatalf("Login(%q) user=%+v token=%q, want userID %d", tt.username, user, token, tt.wantUserID)
			}
		})
	}
}

func TestAuthServiceGetByID(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	svc := NewAuthService(newTestConfig(), repo, discardLogger())
	created, _, err := svc.Register("carol", "carol123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	tests := []struct {
		name    string
		id      uint
		wantErr bool
	}{
		{name: "existing user", id: created.ID},
		{name: "missing user", id: 9999, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetByID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetByID(%d) expected error", tt.id)
				}
				return
			}
			if err != nil || user.ID != tt.id {
				t.Fatalf("GetByID(%d) = %+v, %v", tt.id, user, err)
			}
		})
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
