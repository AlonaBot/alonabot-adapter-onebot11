package onebot11

import (
	"context"
	"log/slog"
	"strconv"

	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// enrichMentions 补充可选展示名；查询失败时保留提及和整条消息。
func (i *instance) enrichMentions(ctx context.Context, post onebotPost, chain msg.Chain) {
	names := map[string]string{i.account.ID: i.Account().DisplayName}
	if post.UserID != nil {
		name := post.Sender.Card
		if name == "" {
			name = post.Sender.Nickname
		}
		if name != "" {
			names[strconv.FormatInt(*post.UserID, 10)] = name
		}
	}
	for n, segment := range chain {
		mention, ok := segment.(msg.MentionSegment)
		if !ok || mention.Display != "" {
			continue
		}
		name, known := names[mention.ExternalUserID]
		if !known {
			var user struct {
				Card     string `json:"card"`
				Nickname string `json:"nickname"`
			}
			params := map[string]any{"user_id": mention.ExternalUserID, "no_cache": false}
			action := "get_stranger_info"
			if post.GroupID != nil {
				action = "get_group_member_info"
				params["group_id"] = *post.GroupID
			}
			if err := i.call(ctx, action, params, &user); err != nil {
				slog.Warn("onebot mention display lookup failed", "adapter_instance_id", i.id, "user_id", mention.ExternalUserID, "error", i.config.safeError(err))
			} else {
				name = user.Card
				if name == "" {
					name = user.Nickname
				}
				if name == "" {
					slog.Warn("onebot mention display lookup returned no name", "adapter_instance_id", i.id, "user_id", mention.ExternalUserID)
				}
			}
			names[mention.ExternalUserID] = name
		}
		mention.Display = name
		chain[n] = mention
	}
}
