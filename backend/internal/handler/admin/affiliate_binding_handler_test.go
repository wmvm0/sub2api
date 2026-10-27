package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type bindingRepoStub struct {
	service.AffiliateRepository
	bound             bool
	err               error
	calls             int
	userID, inviterID int64
}

func (s *bindingRepoStub) BindInviter(_ context.Context, u, i int64) (bool, error) {
	s.calls++
	s.userID, s.inviterID = u, i
	return s.bound, s.err
}
func TestAdminBindInviter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, id, body string
		bound          bool
		err            error
		status, calls  int
	}{
		{"success", "2", `{"inviter_id":1}`, true, nil, 200, 1},
		{"already bound", "2", `{"inviter_id":1}`, false, nil, 409, 1},
		{"cycle or inactive", "2", `{"inviter_id":1}`, false, service.ErrAffiliateInvalidInviter, 400, 1},
		{"self", "2", `{"inviter_id":2}`, false, nil, 400, 0},
		{"invalid target", "0", `{"inviter_id":1}`, false, nil, 400, 0},
		{"missing inviter", "2", `{}`, false, nil, 400, 0},
		{"negative inviter", "2", `{"inviter_id":-1}`, false, nil, 400, 0},
		{"bad json", "2", `{`, false, nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &bindingRepoStub{bound: tc.bound, err: tc.err}
			h := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), nil)
			r := gin.New()
			r.POST("/users/:user_id/inviter", h.BindInviter)
			req := httptest.NewRequest(http.MethodPost, "/users/"+tc.id+"/inviter", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.Equal(t, tc.calls, repo.calls)
			if tc.calls > 0 {
				require.Equal(t, int64(2), repo.userID)
				require.Equal(t, int64(1), repo.inviterID)
			}
		})
	}
}

func TestAdminBindInviter_RequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &bindingRepoStub{bound: true}
	h := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), nil)
	r := gin.New()
	r.Use(gin.HandlerFunc(middleware.NewAdminAuthMiddleware(nil, nil, nil, nil)))
	r.POST("/users/:user_id/inviter", h.BindInviter)
	req := httptest.NewRequest(http.MethodPost, "/users/2/inviter", bytes.NewBufferString(`{"inviter_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Zero(t, repo.calls)
}
