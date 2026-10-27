package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrAffiliateInvalidInviter = infraerrors.BadRequest("AFFILIATE_INVALID_INVITER", "inviter must be a different, active user and cannot create an invitation cycle")

// AdminBindInviter only establishes a relationship. It neither accrues historical
// rebates nor changes feature switches, rates, balances or quota.
func (s *AffiliateService) AdminBindInviter(ctx context.Context, userID, inviterID int64) error {
	if userID <= 0 || inviterID <= 0 || userID == inviterID {
		return ErrAffiliateInvalidInviter
	}
	if s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	bound, err := s.repo.BindInviter(ctx, userID, inviterID)
	if err != nil {
		return err
	}
	if !bound {
		return ErrAffiliateAlreadyBound
	}
	return nil
}
