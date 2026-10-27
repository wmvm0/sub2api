//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateBinding_ImmutableAndCycleSafe(t *testing.T) {
	ctx := context.Background()
	repo := NewAffiliateRepository(integrationEntClient, integrationDB)
	newUser := func() int64 {
		u := mustCreateUser(t, integrationEntClient, &service.User{Email: fmt.Sprintf("binding-%d@example.com", time.Now().UnixNano()), PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive, Concurrency: 1})
		return u.ID
	}
	a, b, c := newUser(), newUser(), newUser()
	// Only one request may win, and the winner's count increments once.
	start := make(chan struct{})
	var wg sync.WaitGroup
	outcomes := make(chan bool, 2)
	errs := make(chan error, 2)
	for _, inviter := range []int64{a, b} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			ok, err := repo.BindInviter(ctx, c, id)
			outcomes <- ok
			errs <- err
		}(inviter)
	}
	close(start)
	wg.Wait()
	close(outcomes)
	close(errs)
	wins := 0
	for ok := range outcomes {
		if ok {
			wins++
		}
	}
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, 1, wins)
	profile, err := repo.EnsureUserAffiliate(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, profile.InviterID)
	winner := *profile.InviterID
	require.Equal(t, 1, querySingleInt(t, ctx, integrationEntClient, "SELECT SUM(aff_count) FROM user_affiliates WHERE user_id IN ($1,$2)", a, b))
	ok, err := repo.BindInviter(ctx, c, winner)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.BindInviter(ctx, winner, c)
	require.ErrorIs(t, err, service.ErrAffiliateInvalidInviter)
	require.False(t, ok)
	ok, err = repo.BindInviter(ctx, a, a)
	require.ErrorIs(t, err, service.ErrAffiliateInvalidInviter)
	require.False(t, ok)
	require.Zero(t, querySingleInt(t, ctx, integrationEntClient, "SELECT COUNT(*) FROM user_affiliate_ledger WHERE user_id IN ($1,$2,$3)", a, b, c))
	// Concurrent reverse binding must not form a cycle.
	d, e := newUser(), newUser()
	start = make(chan struct{})
	outcomes = make(chan bool, 2)
	errs = make(chan error, 2)
	for _, pair := range [][2]int64{{d, e}, {e, d}} {
		wg.Add(1)
		go func(p [2]int64) {
			defer wg.Done()
			<-start
			ok, err := repo.BindInviter(ctx, p[0], p[1])
			outcomes <- ok
			errs <- err
		}(pair)
	}
	close(start)
	wg.Wait()
	close(outcomes)
	close(errs)
	wins = 0
	for ok := range outcomes {
		if ok {
			wins++
		}
	}
	require.Equal(t, 1, wins)
	failures := 0
	for err := range errs {
		if err != nil {
			failures++
			require.ErrorIs(t, err, service.ErrAffiliateInvalidInviter)
		}
	}
	require.Equal(t, 1, failures)
	// Nonexistent and disabled inviters must fail without binding the target.
	target, inactive := newUser(), newUser()
	_, err = integrationDB.ExecContext(ctx, "UPDATE users SET status='disabled' WHERE id=$1", inactive)
	require.NoError(t, err)
	for _, id := range []int64{inactive, 9223372036854775806} {
		ok, err = repo.BindInviter(ctx, target, id)
		require.ErrorIs(t, err, service.ErrAffiliateInvalidInviter)
		require.False(t, ok)
	}
}
