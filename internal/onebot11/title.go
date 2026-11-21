package onebot11

import (
	"context"
	"log/slog"
)

type groupTitle struct {
	ready chan struct{}
	name  string
}

// chatTitle 在进程内只查询每个群一次；失败结果也缓存，取消仍可结束等待。
func (i *instance) chatTitle(ctx context.Context, post onebotPost) string {
	if post.GroupID == nil {
		return post.Sender.Nickname
	}
	groupID := *post.GroupID
	i.owner.mu.Lock()
	title, exists := i.owner.titles[groupID]
	if !exists {
		title = &groupTitle{ready: make(chan struct{})}
		i.owner.titles[groupID] = title
	}
	i.owner.mu.Unlock()
	if exists {
		select {
		case <-ctx.Done():
			return ""
		case <-title.ready:
			return title.name
		}
	}
	defer close(title.ready) // 发布查询结果后，其他实例才读取 name。
	var response struct {
		GroupName string `json:"group_name"`
	}
	if err := i.call(ctx, "get_group_info", map[string]any{"group_id": groupID, "no_cache": false}, &response); err != nil {
		slog.Warn("onebot chat title lookup failed", "adapter_instance_id", i.id, "group_id", groupID, "error", i.config.safeError(err))
		return ""
	}
	title.name = response.GroupName
	return title.name
}
