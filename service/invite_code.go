package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InviteCodeService subscription invitation code service
type InviteCodeService struct {
}

// NewInviteCodeService creates InviteCodeService
func NewInviteCodeService() *InviteCodeService {
	return &InviteCodeService{}
}

// Db returns the DB instance
func (s *InviteCodeService) Db() *gorm.DB {
	return DB
}

const base62Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrInviteCodeNotFound        = errors.New("invite code not found")
	ErrInviteCodeUsed            = errors.New("used invite code cannot be deleted")
	ErrInviteCodeAlreadyConsumed = errors.New("invite code already used or revoked")
)

// generateCode generates a 32-bit base62 random string
func (s *InviteCodeService) generateCode() string {
	b := make([]byte, 32)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(base62Charset))))
		b[i] = base62Charset[n.Int64()]
	}
	return string(b)
}

// Generate generates invitation codes and binds users (automatic code issuance process)
// If orderID is not empty, it will be bound in the same transaction
func (s *InviteCodeService) Generate(plan string, userID uint, boundOrderID string, expireDays int) (*model.InviteCode, error) {
	return s.GenerateWithDB(s.Db(), plan, userID, boundOrderID, expireDays)
}

// GenerateWithDB generates an invitation code on the specified DB connection (can be transaction tx).
// tx must be passed in the transaction callback, otherwise SQLite will cause "database is locked" deadlock due to database-level exclusive lock.
func (s *InviteCodeService) GenerateWithDB(db *gorm.DB, plan string, userID uint, boundOrderID string, expireDays int) (*model.InviteCode, error) {
	// Retry up to 10 times to avoid unique index conflicts
	var code *model.InviteCode
	for i := 0; i < 10; i++ {
		codeStr := s.generateCode()
		code = &model.InviteCode{
			Code:         codeStr,
			Plan:         plan,
			ExpireDays:   expireDays,
			ExpireAt:     time.Now().AddDate(0, 0, expireDays),
			Status:       "unused",
			BoundOrderID: boundOrderID,
		}
		err := db.Create(code).Error
		if err == nil {
			return code, nil
		}
		// Unique index conflict, try again
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			continue
		}
		return nil, fmt.Errorf("generate invite code: %w", err)
	}
	return nil, fmt.Errorf("generate invite code: failed after 10 retries")
}

// Activate Activate user subscription using invitation code (deferral strategy)
// Transaction: update code status + update user.subscription_expire_at
func (s *InviteCodeService) Activate(codeStr string, userID uint) (*model.InviteCode, error) {
	lockKey := fmt.Sprintf("invite-code-activate:user:%d", userID)
	Lock.Lock(lockKey)
	defer Lock.UnLock(lockKey)

	now := time.Now()
	ic := &model.InviteCode{}
	err := s.Db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", codeStr).First(ic).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteCodeNotFound
			}
			return fmt.Errorf("find invite code for activation: %w", err)
		}

		switch ic.Status {
		case "used":
			return ErrInviteCodeAlreadyConsumed
		case "revoked":
			return ErrInviteCodeAlreadyConsumed
		}
		if ic.ExpireAt.Before(now) {
			return fmt.Errorf("code expired")
		}

		// Atomic update code status (optimistic locking)
		res := tx.Model(&model.InviteCode{}).
			Where("id = ? AND status = ?", ic.Id, "unused").
			Updates(map[string]interface{}{
				"status":    "used",
				"used_by":   userID,
				"used_at":   &now,
				"expire_at": ic.ExpireAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrInviteCodeAlreadyConsumed
		}

		// Update user subscription expiration time (extension policy)
		user := &model.User{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", userID).First(user).Error; err != nil {
			return fmt.Errorf("load user for activation: %w", err)
		}
		var newExpire time.Time
		periodDuration := time.Duration(ic.ExpireDays*24) * time.Hour
		if user.SubscriptionExpireAt == nil || user.SubscriptionExpireAt.Before(now) {
			newExpire = now.Add(periodDuration)
		} else {
			newExpire = user.SubscriptionExpireAt.Add(periodDuration)
		}
		// Synchronously update expired_at
		expiredAt := newExpire.Unix()
		result := tx.Model(&model.User{}).Where("id = ?", userID).
			Updates(map[string]interface{}{
				"subscription_plan":      ic.Plan,
				"subscription_expire_at": &newExpire,
				"expired_at":             expiredAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("user not found")
		}

		ic.Status = "used"
		ic.UsedBy = userID
		ic.UsedAt = &now
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ic, nil
}

// Revoke invalid invitation code (administrators only)
func (s *InviteCodeService) Revoke(id uint) error {
	now := time.Now()
	res := s.Db().Model(&model.InviteCode{}).
		Where("id = ? AND status = ?", id, "unused").
		Updates(map[string]interface{}{
			"status":     "revoked",
			"revoked_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("code not found or already used/revoked")
	}
	return nil
}

// Delete permanently deletes an invite code by ID.
//
// Used codes are retained because they are redemption audit records and, for
// order-bound codes, are also used to make ClaimCode idempotent. Unused codes
// (expired or not) and revoked codes may be deleted by an administrator.
func (s *InviteCodeService) Delete(id uint) error {
	return s.Db().Transaction(func(tx *gorm.DB) error {
		ic := &model.InviteCode{}
		if err := tx.Where("id = ?", id).First(ic).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteCodeNotFound
			}
			return err
		}

		if ic.Status == "used" || ic.UsedBy != 0 || ic.UsedAt != nil {
			return ErrInviteCodeUsed
		}

		res := tx.Where("id = ? AND status <> ?", id, "used").
			Delete(&model.InviteCode{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// The status may have changed concurrently after it was read.
			return ErrInviteCodeUsed
		}
		return nil
	})
}

// List Query invitation code list by page
// Supports filtering by status / plan / used_by
type InviteCodeFilter struct {
	Status   string
	Plan     string
	UsedBy   uint
	Page     int
	PageSize int
}

func (s *InviteCodeService) List(filter InviteCodeFilter) ([]*model.InviteCode, int64, error) {
	query := s.Db().Model(&model.InviteCode{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Plan != "" {
		query = query.Where("plan = ?", filter.Plan)
	}
	if filter.UsedBy > 0 {
		query = query.Where("used_by = ?", filter.UsedBy)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count invite codes: %w", err)
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	var list []*model.InviteCode
	if err := query.Order("id desc").
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list invite codes: %w", err)
	}

	return list, total, nil
}

// InfoByCode query based on code
func (s *InviteCodeService) InfoByCode(code string) (*model.InviteCode, error) {
	ic := &model.InviteCode{}
	err := s.Db().Where("code = ?", code).First(ic).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInviteCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invite code by code: %w", err)
	}
	return ic, nil
}

// InfoByOrderID queries the generated invitation code based on the order number
func (s *InviteCodeService) InfoByOrderID(orderID string) (*model.InviteCode, error) {
	ic := &model.InviteCode{}
	err := s.Db().Where("bound_order_id = ?", orderID).First(ic).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInviteCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find invite code by order ID: %w", err)
	}
	return ic, nil
}
