package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"strconv"
)

// GetInviter is an admin-only view; do not expose another user's email in a
// general user endpoint. The numeric ID remains visible if the inviter is deleted.
func (h *AffiliateHandler) GetInviter(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	if _, err = h.adminService.GetUser(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	profile, err := h.affiliateService.EnsureUserAffiliate(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var inviter *AffiliateUserSummary
	if profile.InviterID != nil {
		inviter = &AffiliateUserSummary{ID: *profile.InviterID}
		u, lookupErr := h.adminService.GetUserIncludeDeleted(c.Request.Context(), *profile.InviterID)
		if lookupErr != nil {
			response.ErrorFrom(c, lookupErr)
			return
		}
		inviter.Email, inviter.Username = u.Email, u.Username
	}
	response.Success(c, gin.H{"user_id": userID, "inviter": inviter})
}

// BindInviter is registered exclusively under the authenticated, audited admin
// router. The repository atomically refuses to overwrite an existing inviter.
func (h *AffiliateHandler) BindInviter(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	var req struct {
		InviterID int64 `json:"inviter_id" binding:"required,gt=0"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid inviter_id")
		return
	}
	middleware.SetAuditAction(c, "admin.affiliates.bind_inviter")
	middleware.SetAuditExtra(c, map[string]any{"target_user_id": userID, "inviter_id": req.InviterID})
	if err = h.affiliateService.AdminBindInviter(c.Request.Context(), userID, req.InviterID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "inviter_id": req.InviterID})
}
