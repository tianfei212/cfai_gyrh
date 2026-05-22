#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-backend/data/gyrh.db}"
BACKUP_PATH="${2:-backend/data/gyrh_bak_20260522-1.db}"

if [ ! -f "$DB_PATH" ]; then
  echo "数据库不存在: $DB_PATH" >&2
  exit 1
fi

if [ -e "$BACKUP_PATH" ]; then
  echo "备份文件已存在，避免覆盖: $BACKUP_PATH" >&2
  exit 1
fi

echo "备份数据库: $DB_PATH -> $BACKUP_PATH"
sqlite3 "$DB_PATH" ".backup '$BACKUP_PATH'"

echo "升级数据库结构..."
sqlite3 "$DB_PATH" <<'SQL'
PRAGMA foreign_keys = OFF;

BEGIN IMMEDIATE;

CREATE TABLE IF NOT EXISTS background_categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_name TEXT NOT NULL,
  child_name TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS background_category_bindings (
  category_id INTEGER NOT NULL,
  background_prompt_id INTEGER NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (category_id, background_prompt_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_background_categories_parent_child
ON background_categories(parent_name, child_name);

CREATE INDEX IF NOT EXISTS idx_background_category_bindings_background
ON background_category_bindings(background_prompt_id);

INSERT OR IGNORE INTO background_categories (parent_name, child_name, created_at, updated_at)
VALUES ('default', 'default', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT OR IGNORE INTO background_category_bindings (category_id, background_prompt_id, created_at)
SELECT dc.id, bp.id, CURRENT_TIMESTAMP
FROM background_prompts bp
JOIN background_categories dc
  ON dc.parent_name = 'default'
 AND dc.child_name = 'default'
WHERE NOT EXISTS (
  SELECT 1
  FROM background_category_bindings b
  WHERE b.background_prompt_id = bp.id
);

COMMIT;

PRAGMA foreign_keys = ON;
SQL

echo "校验升级结果..."
sqlite3 "$DB_PATH" <<'SQL'
.headers on
.mode column

SELECT name
FROM sqlite_master
WHERE type = 'table'
  AND name IN ('background_categories', 'background_category_bindings')
ORDER BY name;

SELECT id, parent_name, child_name
FROM background_categories
WHERE parent_name = 'default'
  AND child_name = 'default';

SELECT COUNT(*) AS total_backgrounds
FROM background_prompts;

SELECT COUNT(DISTINCT background_prompt_id) AS bound_backgrounds
FROM background_category_bindings;

SELECT COUNT(*) AS unbound_backgrounds
FROM background_prompts bp
WHERE NOT EXISTS (
  SELECT 1
  FROM background_category_bindings b
  WHERE b.background_prompt_id = bp.id
);
SQL

unbound_count="$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM background_prompts bp WHERE NOT EXISTS (SELECT 1 FROM background_category_bindings b WHERE b.background_prompt_id = bp.id);")"
if [ "$unbound_count" != "0" ]; then
  echo "升级校验失败：仍有 $unbound_count 条背景图没有分类绑定" >&2
  exit 1
fi

echo "升级完成。unbound_backgrounds = 0"
