package policy_group_svc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/model/entity/policy_group_entity"
	"github.com/opskat/opskat/internal/repository/policy_group_repo"
)

// PolicyGroupSvc 权限组业务接口
type PolicyGroupSvc interface {
	List(ctx context.Context, policyType string) ([]*policy_group_entity.PolicyGroupItem, error)
	Get(ctx context.Context, id int64) (*policy_group_entity.PolicyGroupItem, error)
	Create(ctx context.Context, pg *policy_group_entity.PolicyGroup) error
	Update(ctx context.Context, pg *policy_group_entity.PolicyGroup) error
	Delete(ctx context.Context, id int64) error
	Copy(ctx context.Context, id int64, name string) (*policy_group_entity.PolicyGroup, error)
}

type policyGroupSvc struct{}

var defaultSvc = &policyGroupSvc{}

// PolicyGroup 获取 PolicyGroupSvc 实例
func PolicyGroup() PolicyGroupSvc {
	return defaultSvc
}

func (s *policyGroupSvc) List(ctx context.Context, policyType string) ([]*policy_group_entity.PolicyGroupItem, error) {
	// 内置组
	var items []*policy_group_entity.PolicyGroupItem
	for _, pg := range policy_group_entity.BuiltinGroups() {
		if policyType != "" && pg.PolicyType != policyType {
			continue
		}
		items = append(items, pg.ToItem(true))
	}

	// 用户自定义组
	var userGroups []*policy_group_entity.PolicyGroup
	var err error
	if policyType != "" {
		userGroups, err = policy_group_repo.PolicyGroup().ListByType(ctx, policyType)
	} else {
		userGroups, err = policy_group_repo.PolicyGroup().List(ctx)
	}
	if err != nil {
		// DB 查询失败（如表未创建）不影响内置组返回
		logger.Default().Warn("list user policy groups", zap.Error(err))
		return items, nil
	}
	for _, pg := range userGroups {
		items = append(items, pg.ToItem(false))
	}

	return items, nil
}

func (s *policyGroupSvc) Get(ctx context.Context, id int64) (*policy_group_entity.PolicyGroupItem, error) {
	if policy_group_entity.IsBuiltinID(id) {
		pg := policy_group_entity.FindBuiltin(id)
		if pg == nil {
			return nil, errors.New("内置权限组不存在")
		}
		return pg.ToItem(true), nil
	}
	pg, err := policy_group_repo.PolicyGroup().Find(ctx, id)
	if err != nil {
		return nil, err
	}
	return pg.ToItem(false), nil
}

func (s *policyGroupSvc) Create(ctx context.Context, pg *policy_group_entity.PolicyGroup) error {
	if err := pg.Validate(); err != nil {
		return err
	}
	now := time.Now().Unix()
	pg.Createtime = now
	pg.Updatetime = now
	return policy_group_repo.PolicyGroup().Create(ctx, pg)
}

func (s *policyGroupSvc) Update(ctx context.Context, pg *policy_group_entity.PolicyGroup) error {
	if policy_group_entity.IsBuiltinID(pg.ID) {
		return errors.New("内置权限组不可修改")
	}
	if err := pg.Validate(); err != nil {
		return err
	}
	pg.Updatetime = time.Now().Unix()
	return policy_group_repo.PolicyGroup().Update(ctx, pg)
}

func (s *policyGroupSvc) Delete(ctx context.Context, id int64) error {
	if policy_group_entity.IsBuiltinID(id) {
		return errors.New("内置权限组不可删除")
	}
	return policy_group_repo.PolicyGroup().Delete(ctx, id)
}

func (s *policyGroupSvc) Copy(ctx context.Context, id int64, name string) (*policy_group_entity.PolicyGroup, error) {
	var source *policy_group_entity.PolicyGroup
	if policy_group_entity.IsBuiltinID(id) {
		source = policy_group_entity.FindBuiltin(id)
		if source == nil {
			return nil, errors.New("内置权限组不存在")
		}
	} else {
		var err error
		source, err = policy_group_repo.PolicyGroup().Find(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	if name == "" {
		name = source.Name + " (副本)"
	}

	// 深拷贝 policy JSON（确保无共享引用），移除 groups 字段避免嵌套引用
	policyJSON := removePolicyGroups(source.Policy)

	newPG := &policy_group_entity.PolicyGroup{
		Name:        name,
		Description: source.Description,
		PolicyType:  source.PolicyType,
		Policy:      policyJSON,
	}
	if err := s.Create(ctx, newPG); err != nil {
		return nil, err
	}
	return newPG, nil
}

// removePolicyGroups 从策略 JSON 中移除 groups 字段（权限组自身不应引用其他组）
func removePolicyGroups(policyJSON string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(policyJSON), &m); err != nil {
		return policyJSON
	}
	delete(m, "groups")
	data, err := json.Marshal(m)
	if err != nil {
		return policyJSON
	}
	return string(data)
}
