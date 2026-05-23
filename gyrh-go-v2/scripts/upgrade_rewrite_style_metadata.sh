#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-backend/data/gyrh.db}"
BACKUP_PATH="${2:-${DB_PATH%.db}_bak_rewrite_style_$(date +%Y%m%d%H%M%S).db}"

if [ ! -f "$DB_PATH" ]; then
  echo "数据库不存在: $DB_PATH" >&2
  exit 1
fi

if [ -e "$BACKUP_PATH" ]; then
  echo "备份文件已存在，避免覆盖: $BACKUP_PATH" >&2
  exit 1
fi

require_table() {
  local table="$1"
  local exists
  exists="$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='$table';")"
  if [ "$exists" != "1" ]; then
    echo "数据库缺少必要表: $table" >&2
    exit 1
  fi
}

column_exists() {
  local table="$1"
  local column="$2"
  sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM pragma_table_info('$table') WHERE name='$column';"
}

add_column_if_missing() {
  local table="$1"
  local column="$2"
  local definition="$3"
  if [ "$(column_exists "$table" "$column")" = "0" ]; then
    echo "新增字段: $table.$column"
    sqlite3 "$DB_PATH" "ALTER TABLE $table ADD COLUMN $definition;"
  else
    echo "字段已存在: $table.$column"
  fi
}

require_table "generated_images"
require_table "image_rewrite_tasks"

echo "备份数据库: $DB_PATH -> $BACKUP_PATH"
sqlite3 "$DB_PATH" ".backup '$BACKUP_PATH'"

echo "升级转绘风格元数据字段..."
add_column_if_missing "generated_images" "asset_id" "asset_id TEXT NOT NULL DEFAULT ''"
add_column_if_missing "generated_images" "provider" "provider TEXT NOT NULL DEFAULT ''"
add_column_if_missing "generated_images" "status" "status TEXT NOT NULL DEFAULT 'succeeded'"
add_column_if_missing "generated_images" "background_prompt_id" "background_prompt_id INTEGER NOT NULL DEFAULT 0"
add_column_if_missing "generated_images" "image_width" "image_width INTEGER NOT NULL DEFAULT 0"
add_column_if_missing "generated_images" "image_height" "image_height INTEGER NOT NULL DEFAULT 0"
add_column_if_missing "image_rewrite_tasks" "style_name" "style_name TEXT NOT NULL DEFAULT ''"

echo "回填兼容字段..."
sqlite3 "$DB_PATH" <<'SQL'
BEGIN IMMEDIATE;

UPDATE generated_images
SET asset_id = path
WHERE asset_id = ''
  AND path != '';

UPDATE generated_images
SET provider = style_transform
WHERE provider = ''
  AND style_transform IN ('google', 'wan', '302-gpt-image');

UPDATE generated_images
SET status = 'succeeded'
WHERE status = '';

COMMIT;
SQL

echo "校验升级结果..."
sqlite3 "$DB_PATH" <<'SQL'
.headers on
.mode column

SELECT name
FROM pragma_table_info('generated_images')
WHERE name IN ('style_transform', 'provider', 'asset_id', 'status', 'background_prompt_id', 'image_width', 'image_height')
ORDER BY name;

SELECT name
FROM pragma_table_info('image_rewrite_tasks')
WHERE name = 'style_name';

SELECT COUNT(*) AS generated_total
FROM generated_images;

SELECT COUNT(*) AS provider_style_values
FROM generated_images
WHERE style_transform IN ('google', 'wan', '302-gpt-image');

SELECT id, name, provider, style_transform, status, created_at
FROM generated_images
ORDER BY id DESC
LIMIT 10;
SQL

provider_style_count="$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM generated_images WHERE style_transform IN ('google', 'wan', '302-gpt-image');")"
if [ "$provider_style_count" != "0" ]; then
  echo "注意：仍有 $provider_style_count 条历史记录的 style_transform 是 provider 值。"
  echo "脚本不会自动猜测历史风格；如需修正，请按业务确认后手动执行类似："
  echo "sqlite3 \"$DB_PATH\" \"UPDATE generated_images SET style_transform='黑白线稿' WHERE id=<图片ID>;\""
fi

echo "升级完成。"
