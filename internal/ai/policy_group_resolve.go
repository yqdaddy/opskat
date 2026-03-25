package ai

import (
	"context"
	"encoding/json"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/model/entity/policy"
	"github.com/opskat/opskat/internal/model/entity/policy_group_entity"
	"github.com/opskat/opskat/internal/repository/policy_group_repo"
)

// resolveCommandGroups 解析引用的权限组，返回合并后的 allow/deny 规则
func resolveCommandGroups(ctx context.Context, groupIDs []int64) (allow, deny []string) {
	if len(groupIDs) == 0 {
		return
	}
	for _, pg := range fetchPolicyGroups(ctx, groupIDs) {
		if pg.PolicyType != policy_group_entity.PolicyTypeCommand {
			continue
		}
		var p policy.CommandPolicy
		if err := json.Unmarshal([]byte(pg.Policy), &p); err != nil {
			logger.Default().Warn("unmarshal policy group command policy", zap.Int64("id", pg.ID), zap.Error(err))
			continue
		}
		allow = append(allow, p.AllowList...)
		deny = append(deny, p.DenyList...)
	}
	return
}

// resolveQueryGroups 解析引用的 Query 权限组，返回合并后的规则
func resolveQueryGroups(ctx context.Context, groupIDs []int64) (allowTypes, denyTypes, denyFlags []string) {
	if len(groupIDs) == 0 {
		return
	}
	for _, pg := range fetchPolicyGroups(ctx, groupIDs) {
		if pg.PolicyType != policy_group_entity.PolicyTypeQuery {
			continue
		}
		var p policy.QueryPolicy
		if err := json.Unmarshal([]byte(pg.Policy), &p); err != nil {
			logger.Default().Warn("unmarshal policy group query policy", zap.Int64("id", pg.ID), zap.Error(err))
			continue
		}
		allowTypes = append(allowTypes, p.AllowTypes...)
		denyTypes = append(denyTypes, p.DenyTypes...)
		denyFlags = append(denyFlags, p.DenyFlags...)
	}
	return
}

// resolveRedisGroups 解析引用的 Redis 权限组，返回合并后的 allow/deny 规则
func resolveRedisGroups(ctx context.Context, groupIDs []int64) (allow, deny []string) {
	if len(groupIDs) == 0 {
		return
	}
	for _, pg := range fetchPolicyGroups(ctx, groupIDs) {
		if pg.PolicyType != policy_group_entity.PolicyTypeRedis {
			continue
		}
		var p policy.RedisPolicy
		if err := json.Unmarshal([]byte(pg.Policy), &p); err != nil {
			logger.Default().Warn("unmarshal policy group redis policy", zap.Int64("id", pg.ID), zap.Error(err))
			continue
		}
		allow = append(allow, p.AllowList...)
		deny = append(deny, p.DenyList...)
	}
	return
}

// fetchPolicyGroups 按 ID 列表获取权限组（内置组从代码，用户组从 DB）
func fetchPolicyGroups(ctx context.Context, ids []int64) []*policy_group_entity.PolicyGroup {
	var result []*policy_group_entity.PolicyGroup
	var dbIDs []int64

	for _, id := range ids {
		if policy_group_entity.IsBuiltinID(id) {
			if pg := policy_group_entity.FindBuiltin(id); pg != nil {
				result = append(result, pg)
			}
		} else {
			dbIDs = append(dbIDs, id)
		}
	}

	if len(dbIDs) > 0 {
		repo := policy_group_repo.PolicyGroup()
		if repo != nil {
			pgs, err := repo.ListByIDs(ctx, dbIDs)
			if err != nil {
				logger.Default().Warn("fetch policy groups from DB", zap.Error(err))
			} else {
				result = append(result, pgs...)
			}
		}
	}

	return result
}
