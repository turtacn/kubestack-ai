package rollback

import (
	"context"
	"fmt"
	"time"
)

// TargetInfo describes the target system for snapshot
type TargetInfo struct {
	Type string // redis, mysql, etc.
	ID   string // instance id or connection string
	// Add connection details as needed
}

// StateSnapshot 状态快照
type StateSnapshot struct {
	ID        string                 // 快照唯一ID
	PlanID    string                 // 关联的执行计划ID
	CreatedAt time.Time              // 创建时间
	TargetType string                 // 目标类型(redis/mysql/kafka等)
	TargetID  string                 // 目标实例ID
	State     map[string]interface{} // 状态数据
	Metadata  map[string]string      // 元数据
}

// SnapshotCollector 快照采集器接口
type SnapshotCollector interface {
	// Collect 采集当前状态
	// 输入: 目标连接信息
	// 输出: 状态快照
	Collect(ctx context.Context, target *TargetInfo) (*StateSnapshot, error)

	// SupportedTypes 支持的目标类型
	SupportedTypes() []string
}

// SnapshotStore 快照存储接口
type SnapshotStore interface {
	Save(ctx context.Context, snapshot *StateSnapshot) error
	Load(ctx context.Context, snapshotID string) (*StateSnapshot, error)
	Delete(ctx context.Context, snapshotID string) error
	ListByPlan(ctx context.Context, planID string) ([]*StateSnapshot, error)
}

// InMemorySnapshotStore Basic in-memory store for now
type InMemorySnapshotStore struct {
	snapshots map[string]*StateSnapshot
}

func NewInMemorySnapshotStore() *InMemorySnapshotStore {
	return &InMemorySnapshotStore{
		snapshots: make(map[string]*StateSnapshot),
	}
}

func (s *InMemorySnapshotStore) Save(ctx context.Context, snapshot *StateSnapshot) error {
	s.snapshots[snapshot.ID] = snapshot
	return nil
}

func (s *InMemorySnapshotStore) Load(ctx context.Context, snapshotID string) (*StateSnapshot, error) {
	if snap, ok := s.snapshots[snapshotID]; ok {
		return snap, nil
	}
	return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
}

func (s *InMemorySnapshotStore) Delete(ctx context.Context, snapshotID string) error {
	delete(s.snapshots, snapshotID)
	return nil
}

func (s *InMemorySnapshotStore) ListByPlan(ctx context.Context, planID string) ([]*StateSnapshot, error) {
	var result []*StateSnapshot
	for _, snap := range s.snapshots {
		if snap.PlanID == planID {
			result = append(result, snap)
		}
	}
	return result, nil
}
