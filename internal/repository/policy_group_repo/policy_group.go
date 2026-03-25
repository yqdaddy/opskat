package policy_group_repo

import (
	"context"

	"github.com/opskat/opskat/internal/model/entity/policy_group_entity"

	"github.com/cago-frame/cago/database/db"
)

// PolicyGroupRepo 权限组数据访问接口
type PolicyGroupRepo interface {
	Find(ctx context.Context, id int64) (*policy_group_entity.PolicyGroup, error)
	List(ctx context.Context) ([]*policy_group_entity.PolicyGroup, error)
	ListByType(ctx context.Context, policyType string) ([]*policy_group_entity.PolicyGroup, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*policy_group_entity.PolicyGroup, error)
	Create(ctx context.Context, pg *policy_group_entity.PolicyGroup) error
	Update(ctx context.Context, pg *policy_group_entity.PolicyGroup) error
	Delete(ctx context.Context, id int64) error
}

var instance PolicyGroupRepo

// RegisterPolicyGroup 注册实现
func RegisterPolicyGroup(repo PolicyGroupRepo) {
	instance = repo
}

// PolicyGroup 获取全局实例
func PolicyGroup() PolicyGroupRepo {
	return instance
}

// policyGroupRepo 默认实现
type policyGroupRepo struct{}

// NewPolicyGroup 创建默认实现
func NewPolicyGroup() PolicyGroupRepo {
	return &policyGroupRepo{}
}

func (r *policyGroupRepo) Find(ctx context.Context, id int64) (*policy_group_entity.PolicyGroup, error) {
	var pg policy_group_entity.PolicyGroup
	if err := db.Ctx(ctx).Where("id = ?", id).First(&pg).Error; err != nil {
		return nil, err
	}
	return &pg, nil
}

func (r *policyGroupRepo) List(ctx context.Context) ([]*policy_group_entity.PolicyGroup, error) {
	var pgs []*policy_group_entity.PolicyGroup
	if err := db.Ctx(ctx).Order("createtime DESC").Find(&pgs).Error; err != nil {
		return nil, err
	}
	return pgs, nil
}

func (r *policyGroupRepo) ListByType(ctx context.Context, policyType string) ([]*policy_group_entity.PolicyGroup, error) {
	var pgs []*policy_group_entity.PolicyGroup
	if err := db.Ctx(ctx).Where("policy_type = ?", policyType).Order("createtime DESC").Find(&pgs).Error; err != nil {
		return nil, err
	}
	return pgs, nil
}

func (r *policyGroupRepo) ListByIDs(ctx context.Context, ids []int64) ([]*policy_group_entity.PolicyGroup, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var pgs []*policy_group_entity.PolicyGroup
	if err := db.Ctx(ctx).Where("id IN ?", ids).Find(&pgs).Error; err != nil {
		return nil, err
	}
	return pgs, nil
}

func (r *policyGroupRepo) Create(ctx context.Context, pg *policy_group_entity.PolicyGroup) error {
	return db.Ctx(ctx).Create(pg).Error
}

func (r *policyGroupRepo) Update(ctx context.Context, pg *policy_group_entity.PolicyGroup) error {
	return db.Ctx(ctx).Save(pg).Error
}

func (r *policyGroupRepo) Delete(ctx context.Context, id int64) error {
	return db.Ctx(ctx).Delete(&policy_group_entity.PolicyGroup{}, id).Error
}
