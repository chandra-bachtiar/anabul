-- name: InsertAgentLog :exec
INSERT INTO agent_logs (channel, user_message, tool_calls, reply)
VALUES ($1, $2, $3, $4);

-- name: ListRecentAgentLogs :many
SELECT id, channel, user_message, tool_calls, reply, created_at
FROM agent_logs
ORDER BY id DESC
LIMIT $1;

-- name: InsertChatMessage :exec
INSERT INTO chat_messages (channel, role, content)
VALUES ($1, $2, $3);

-- name: ListRecentChatMessages :many
SELECT id, channel, role, content, created_at
FROM chat_messages
WHERE channel = $1
ORDER BY id DESC
LIMIT $2;

-- name: CreatePendingAction :exec
INSERT INTO pending_actions (id, tool, args, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetPendingAction :one
SELECT id, tool, args, expires_at, created_at
FROM pending_actions
WHERE id = $1;

-- name: DeletePendingAction :exec
DELETE FROM pending_actions WHERE id = $1;

-- name: DeleteExpiredPendingActions :exec
DELETE FROM pending_actions WHERE expires_at < now();
