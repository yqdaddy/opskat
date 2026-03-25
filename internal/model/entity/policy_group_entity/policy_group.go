package policy_group_entity

import (
	"encoding/json"
	"errors"

	"github.com/opskat/opskat/internal/model/entity/policy"
)

// 策略类型常量
const (
	PolicyTypeCommand = "command"
	PolicyTypeQuery   = "query"
	PolicyTypeRedis   = "redis"
)

// PolicyGroup 权限组实体
type PolicyGroup struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"column:name;type:varchar(255);not null" json:"name"`
	Description string `gorm:"column:description;type:text" json:"description"`
	PolicyType  string `gorm:"column:policy_type;type:varchar(50);not null" json:"policyType"`
	Policy      string `gorm:"column:policy;type:text;not null" json:"policy"`
	Createtime  int64  `gorm:"column:createtime" json:"createtime"`
	Updatetime  int64  `gorm:"column:updatetime" json:"updatetime"`
}

// TableName GORM 表名
func (PolicyGroup) TableName() string {
	return "policy_groups"
}

// Validate 校验
func (pg *PolicyGroup) Validate() error {
	if pg.Name == "" {
		return errors.New("权限组名称不能为空")
	}
	switch pg.PolicyType {
	case PolicyTypeCommand, PolicyTypeQuery, PolicyTypeRedis:
	default:
		return errors.New("无效的策略类型")
	}
	return nil
}

// PolicyGroupItem 返回给前端的权限组项，含 Builtin 标识
type PolicyGroupItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PolicyType  string `json:"policyType"`
	Policy      string `json:"policy"`
	Builtin     bool   `json:"builtin"`
	Createtime  int64  `json:"createtime"`
	Updatetime  int64  `json:"updatetime"`
}

// ToItem 转为 PolicyGroupItem
func (pg *PolicyGroup) ToItem(builtin bool) *PolicyGroupItem {
	return &PolicyGroupItem{
		ID:          pg.ID,
		Name:        pg.Name,
		Description: pg.Description,
		PolicyType:  pg.PolicyType,
		Policy:      pg.Policy,
		Builtin:     builtin,
		Createtime:  pg.Createtime,
		Updatetime:  pg.Updatetime,
	}
}

// --- 内置权限组 ---

func mustMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// BuiltinGroups 返回所有内置权限组
func BuiltinGroups() []*PolicyGroup {
	return []*PolicyGroup{
		// SSH command 类型
		{
			ID:          policy.BuiltinLinuxReadOnly,
			Name:        "Linux 常用只读",
			Description: "常用 Linux 只读命令",
			PolicyType:  PolicyTypeCommand,
			Policy: mustMarshal(&policy.CommandPolicy{
				AllowList: []string{
					"ls *", "cat *", "head *", "tail *",
					"grep *", "find *", "pwd", "wc *",
					"whoami", "hostname", "uname *", "id", "date",
					"env", "printenv *", "which *", "file *", "stat *",
					"df *", "du *", "free *", "uptime",
					"ps *", "top -b -n 1 *",
					"netstat *", "ss *", "ip *", "ifconfig *",
					"mount", "lsblk *", "blkid *",
					"lsof *", "vmstat *", "iostat *",
					"systemctl status *", "journalctl *",
				},
			}),
		},
		{
			ID:          policy.BuiltinK8sReadOnly,
			Name:        "Kubernetes 只读",
			Description: "Kubernetes 只读操作命令",
			PolicyType:  PolicyTypeCommand,
			Policy: mustMarshal(&policy.CommandPolicy{
				AllowList: []string{
					"kubectl get *", "kubectl describe *", "kubectl logs *",
					"kubectl top *", "kubectl explain *",
					"kubectl api-resources *", "kubectl api-versions",
					"kubectl cluster-info *", "kubectl config view *",
					"kubectl config get-contexts *", "kubectl version *",
					"kubectl auth can-i *",
				},
			}),
		},
		{
			ID:          policy.BuiltinDockerReadOnly,
			Name:        "Docker 只读",
			Description: "Docker 只读操作命令",
			PolicyType:  PolicyTypeCommand,
			Policy: mustMarshal(&policy.CommandPolicy{
				AllowList: []string{
					"docker ps *", "docker images *", "docker logs *",
					"docker inspect *", "docker stats *", "docker top *",
					"docker port *", "docker diff *", "docker history *",
					"docker info", "docker version",
					"docker network ls *", "docker network inspect *",
					"docker volume ls *", "docker volume inspect *",
					"docker compose ps *", "docker compose logs *",
				},
			}),
		},
		{
			ID:          policy.BuiltinDangerousDeny,
			Name:        "高危命令拒绝",
			Description: "拒绝执行高危系统命令",
			PolicyType:  PolicyTypeCommand,
			Policy: mustMarshal(&policy.CommandPolicy{
				DenyList: []string{
					"rm -rf /*",
					"mkfs *",
					"dd *",
					"shutdown *",
					"reboot *",
					"poweroff *",
					"halt *",
				},
			}),
		},
		// Database query 类型
		{
			ID:          policy.BuiltinSQLReadOnly,
			Name:        "SQL 只读",
			Description: "只允许查询类 SQL 语句",
			PolicyType:  PolicyTypeQuery,
			Policy: mustMarshal(&policy.QueryPolicy{
				AllowTypes: []string{
					"SELECT", "SHOW", "DESCRIBE", "EXPLAIN", "USE",
				},
			}),
		},
		{
			ID:          policy.BuiltinSQLDangerousDeny,
			Name:        "SQL 高危拒绝",
			Description: "拒绝高危 SQL 操作",
			PolicyType:  PolicyTypeQuery,
			Policy: mustMarshal(&policy.QueryPolicy{
				DenyTypes: []string{
					"DROP TABLE", "DROP DATABASE", "TRUNCATE",
					"GRANT", "REVOKE",
					"CREATE USER", "DROP USER", "ALTER USER",
				},
				DenyFlags: []string{
					"no_where_delete",
					"no_where_update",
					"prepare",
				},
			}),
		},
		// Redis 类型
		{
			ID:          policy.BuiltinRedisReadOnly,
			Name:        "Redis 只读",
			Description: "只允许 Redis 只读命令",
			PolicyType:  PolicyTypeRedis,
			Policy: mustMarshal(&policy.RedisPolicy{
				AllowList: []string{
					"GET", "MGET", "STRLEN",
					"HGET", "HGETALL", "HKEYS", "HVALS", "HLEN", "HMGET", "HEXISTS",
					"LRANGE", "LLEN", "LINDEX",
					"SMEMBERS", "SCARD", "SISMEMBER",
					"ZRANGE", "ZCARD", "ZSCORE", "ZRANK", "ZCOUNT",
					"TYPE", "TTL", "PTTL", "EXISTS", "DBSIZE", "KEYS", "SCAN",
					"INFO", "PING",
				},
			}),
		},
		{
			ID:          policy.BuiltinRedisDangerousDeny,
			Name:        "Redis 高危拒绝",
			Description: "拒绝 Redis 高危命令",
			PolicyType:  PolicyTypeRedis,
			Policy: mustMarshal(&policy.RedisPolicy{
				DenyList: []string{
					"FLUSHDB", "FLUSHALL",
					"CONFIG SET *", "CONFIG RESETSTAT",
					"DEBUG *", "SHUTDOWN *",
					"SLAVEOF *", "REPLICAOF *",
					"ACL DELUSER *", "ACL SETUSER *",
					"SCRIPT FLUSH", "CLUSTER RESET *",
				},
			}),
		},
	}
}

// builtinMap 内置组缓存
var builtinMap map[int64]*PolicyGroup

func init() {
	builtinMap = make(map[int64]*PolicyGroup)
	for _, pg := range BuiltinGroups() {
		builtinMap[pg.ID] = pg
	}
}

// FindBuiltin 按 ID 查找内置权限组
func FindBuiltin(id int64) *PolicyGroup {
	return builtinMap[id]
}

// IsBuiltinID 检查 ID 是否为内置权限组
func IsBuiltinID(id int64) bool {
	return id < 0
}
