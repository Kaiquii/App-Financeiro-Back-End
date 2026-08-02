package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// This value is already recorded in production and must never change.
const baselineRecordedChecksum = "72f1622409cda6a371dbf1de0de5575f6e6cf5d10928ebfec2ec1dd472b41c2a"

var baselineStatements = []string{
	`CREATE TABLE users (
		id BIGSERIAL PRIMARY KEY,
		name TEXT,
		email TEXT,
		password TEXT,
		role VARCHAR(20) DEFAULT 'user',
		avatar_url TEXT,
		access_blocked BOOLEAN DEFAULT FALSE,
		access_blocked_at TIMESTAMPTZ
	)`,
	`CREATE UNIQUE INDEX idx_users_email ON users (email)`,
	`CREATE TABLE password_reset_tokens (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		email TEXT,
		ip_address TEXT,
		code_hash TEXT,
		expires_at TIMESTAMPTZ,
		used BOOLEAN,
		created_at TIMESTAMPTZ
	)`,
	`CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id)`,
	`CREATE INDEX idx_password_reset_tokens_email ON password_reset_tokens (email)`,
	`CREATE INDEX idx_password_reset_tokens_ip_address ON password_reset_tokens (ip_address)`,
	`CREATE TABLE registration_codes (
		id BIGSERIAL PRIMARY KEY,
		email TEXT,
		code_hash TEXT,
		ip_address TEXT,
		expires_at TIMESTAMPTZ,
		used BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ
	)`,
	`CREATE INDEX idx_registration_codes_email ON registration_codes (email)`,
	`CREATE INDEX idx_registration_codes_ip_address ON registration_codes (ip_address)`,
	`CREATE TABLE expenses (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		series_id TEXT,
		category_id BIGINT,
		amount NUMERIC,
		description TEXT,
		notes TEXT,
		payment_source TEXT,
		date TIMESTAMPTZ,
		month BIGINT,
		year BIGINT,
		"type" TEXT,
		installments BIGINT,
		current_install BIGINT
	)`,
	`CREATE INDEX idx_expenses_series_id ON expenses (series_id)`,
	`CREATE TABLE incomes (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		source TEXT,
		amount NUMERIC,
		month BIGINT,
		year BIGINT
	)`,
	`CREATE TABLE categories (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		name TEXT
	)`,
	`CREATE TABLE conversations (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT,
		title TEXT,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	)`,
	`CREATE INDEX idx_conversations_user_id ON conversations (user_id)`,
	`CREATE TABLE messages (
		id BIGSERIAL PRIMARY KEY,
		conversation_id BIGINT,
		user_id BIGINT,
		role TEXT,
		content TEXT,
		tool_call TEXT,
		tool_result TEXT,
		created_at TIMESTAMPTZ
	)`,
	`CREATE INDEX idx_messages_conversation_id ON messages (conversation_id)`,
	`CREATE INDEX idx_messages_user_id ON messages (user_id)`,
	`CREATE TABLE app_versions (
		id BIGSERIAL PRIMARY KEY,
		platform VARCHAR(30),
		latest_version_name TEXT,
		latest_version_code BIGINT,
		min_required_version_code BIGINT,
		force_update BOOLEAN,
		play_store_url TEXT,
		message TEXT,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ
	)`,
	`CREATE INDEX idx_app_versions_platform ON app_versions (platform)`,
	`CREATE UNIQUE INDEX idx_app_versions_platform_code ON app_versions (platform, latest_version_code)`,
}

var baselineColumns = map[string]map[string]string{
	"users": {
		"id": "int8", "name": "text", "email": "text", "password": "text", "role": "varchar",
		"avatar_url": "text", "access_blocked": "bool", "access_blocked_at": "timestamptz",
	},
	"password_reset_tokens": {
		"id": "int8", "user_id": "int8", "email": "text", "ip_address": "text", "code_hash": "text",
		"expires_at": "timestamptz", "used": "bool", "created_at": "timestamptz",
	},
	"registration_codes": {
		"id": "int8", "email": "text", "code_hash": "text", "ip_address": "text",
		"expires_at": "timestamptz", "used": "bool", "created_at": "timestamptz",
	},
	"expenses": {
		"id": "int8", "user_id": "int8", "series_id": "text", "category_id": "int8", "amount": "numeric",
		"description": "text", "notes": "text", "payment_source": "text", "date": "timestamptz",
		"month": "int8", "year": "int8", "type": "text", "installments": "int8", "current_install": "int8",
	},
	"incomes": {
		"id": "int8", "user_id": "int8", "source": "text", "amount": "numeric", "month": "int8", "year": "int8",
	},
	"categories": {
		"id": "int8", "user_id": "int8", "name": "text",
	},
	"conversations": {
		"id": "int8", "user_id": "int8", "title": "text", "created_at": "timestamptz", "updated_at": "timestamptz",
	},
	"messages": {
		"id": "int8", "conversation_id": "int8", "user_id": "int8", "role": "text", "content": "text",
		"tool_call": "text", "tool_result": "text", "created_at": "timestamptz",
	},
	"app_versions": {
		"id": "int8", "platform": "varchar", "latest_version_name": "text", "latest_version_code": "int8",
		"min_required_version_code": "int8", "force_update": "bool", "play_store_url": "text",
		"message": "text", "created_at": "timestamptz", "updated_at": "timestamptz",
	},
}

var baselineIndexes = map[string][]string{
	"users":                 {"idx_users_email"},
	"password_reset_tokens": {"idx_password_reset_tokens_email", "idx_password_reset_tokens_ip_address", "idx_password_reset_tokens_user_id"},
	"registration_codes":    {"idx_registration_codes_email", "idx_registration_codes_ip_address"},
	"expenses":              {"idx_expenses_series_id"},
	"conversations":         {"idx_conversations_user_id"},
	"messages":              {"idx_messages_conversation_id", "idx_messages_user_id"},
	"app_versions":          {"idx_app_versions_platform", "idx_app_versions_platform_code"},
}

func migrateBaselineSchema(db *gorm.DB) error {
	for _, statement := range baselineStatements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func baselineDefinitionChecksum() string {
	hash := sha256.New()
	for _, statement := range baselineStatements {
		fmt.Fprintln(hash, statement)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func applicationSchemaExists(db *gorm.DB) (bool, error) {
	for table := range baselineColumns {
		var exists bool
		if err := db.Raw("SELECT to_regclass(?) IS NOT NULL", "public."+table).Scan(&exists).Error; err != nil {
			return false, fmt.Errorf("falha ao verificar tabela %s: %w", table, err)
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}

func validateBaselineSchema(db *gorm.DB) error {
	tables := make([]string, 0, len(baselineColumns))
	for table := range baselineColumns {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	for _, table := range tables {
		var tableExists bool
		if err := db.Raw("SELECT to_regclass(?) IS NOT NULL", "public."+table).Scan(&tableExists).Error; err != nil {
			return fmt.Errorf("falha ao verificar tabela %s: %w", table, err)
		}
		if !tableExists {
			return fmt.Errorf("tabela obrigatoria ausente: %s", table)
		}

		for column, expectedType := range baselineColumns[table] {
			var actualType string
			if err := db.Raw(`
				SELECT udt_name
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = ? AND column_name = ?
			`, table, column).Scan(&actualType).Error; err != nil {
				return fmt.Errorf("falha ao verificar coluna %s.%s: %w", table, column, err)
			}
			if actualType == "" {
				return fmt.Errorf("coluna obrigatoria ausente: %s.%s", table, column)
			}
			if actualType != expectedType {
				return fmt.Errorf("tipo inesperado em %s.%s: esperado %s, encontrado %s", table, column, expectedType, actualType)
			}
		}
	}

	for table, indexes := range baselineIndexes {
		for _, index := range indexes {
			var count int64
			if err := db.Raw(`
				SELECT COUNT(*)
				FROM pg_indexes
				WHERE schemaname = 'public' AND tablename = ? AND indexname = ?
			`, table, index).Scan(&count).Error; err != nil {
				return fmt.Errorf("falha ao verificar indice %s: %w", index, err)
			}
			if count != 1 {
				return fmt.Errorf("indice obrigatorio ausente: %s", index)
			}
		}
	}
	return nil
}
