package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type FileRecord struct {
	ID                string    `json:"id"`
	UserID            string    `json:"-"`
	Name              string    `json:"name"`
	Size              int64     `json:"size"`
	ContentType       string    `json:"contentType"`
	URL               string    `json:"url"`
	PreviewURL        string    `json:"previewUrl,omitempty"`
	RawURL            string    `json:"rawUrl,omitempty"`
	UploadedAt        time.Time `json:"uploadedAt"`
	Views             int64     `json:"views,omitempty"`
	PasswordHash      string    `json:"-"`
	PasswordProtected bool      `json:"passwordProtected"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"passwordHash"`
	PasswordSalt string    `json:"passwordSalt"`
	CreatedAt    time.Time `json:"createdAt"`
	MaxUploadBytes  int64  `json:"maxUploadBytes"`
	MaxStorageBytes int64  `json:"maxStorageBytes"`
	UnlimitedStorage bool   `json:"unlimitedStorage"`
	GlowColor    string    `json:"glowColor"`
}

const (
	defaultMaxUploadBytes  int64 = 100 * 1024 * 1024
	defaultMaxStorageBytes int64 = 5120 * 1024 * 1024
	defaultGlowColor = "#ffffff"
)

type AppConfig struct {
	DatabaseType  string `json:"databaseType,omitempty"`
	SQLitePath    string `json:"sqlitePath,omitempty"`
	MySQLDSN      string `json:"mysqlDsn,omitempty"`
	MySQLHost     string `json:"mysqlHost,omitempty"`
	MySQLPort     string `json:"mysqlPort,omitempty"`
	MySQLDatabase string `json:"mysqlDatabase,omitempty"`
	MySQLUser     string `json:"mysqlUser,omitempty"`
	MySQLPassword string `json:"mysqlPassword,omitempty"`
	MySQLTLS      string `json:"mysqlTls,omitempty"`
	BaseURL       string `json:"baseUrl,omitempty"`
	Port          string `json:"port,omitempty"`
	CookieSecure  string `json:"cookieSecure,omitempty"`
	AdminEmail    string `json:"adminEmail,omitempty"`
	AdminPassword string `json:"adminPassword,omitempty"`
	UploadAPIKey  string `json:"uploadApiKey,omitempty"` // legacy/environment override
	AdminAPIKey   string `json:"adminApiKey,omitempty"` // legacy/environment override
	UploadAPIKeyHash string `json:"uploadApiKeyHash,omitempty"`
	AdminAPIKeyHash  string `json:"adminApiKeyHash,omitempty"`
	CORSOrigin    string `json:"corsOrigin,omitempty"`
	MaxUploadMB   string `json:"maxUploadMb,omitempty"`
	MaxStorageMB  string `json:"maxStorageMb,omitempty"`
	SMTPHost      string `json:"smtpHost,omitempty"`
	SMTPPort      string `json:"smtpPort,omitempty"`
	SMTPUser      string `json:"smtpUser,omitempty"`
	SMTPPassword  string `json:"smtpPassword,omitempty"`
	SMTPFrom      string `json:"smtpFrom,omitempty"`
	SMTPTLS       string `json:"smtpTls,omitempty"`
	BrandLogo     string `json:"brandLogo,omitempty"`
	EmbedSiteName  string `json:"embedSiteName,omitempty"`
	EmbedColor     string `json:"embedColor,omitempty"`
	EmbedTitle     string `json:"embedTitle,omitempty"`
	EmbedExtra     string `json:"embedExtra,omitempty"`
	EmbedCard      string `json:"embedCard,omitempty"`
	EmbedImage     string `json:"embedImage,omitempty"`
	EmbedShowSite  string `json:"embedShowSite,omitempty"`
	EmbedShowDate  string `json:"embedShowDate,omitempty"`
	EmbedShowSize  string `json:"embedShowSize,omitempty"`
	EmbedShowViews string `json:"embedShowViews,omitempty"`
	EmbedShowType  string `json:"embedShowType,omitempty"`
	EmbedShowDims  string `json:"embedShowDims,omitempty"`
	EmbedShowIcons string `json:"embedShowIcons,omitempty"`
}

func (c AppConfig) withDefaults() AppConfig {
	if c.DatabaseType == "" {
		c.DatabaseType = "mysql"
	}
	if c.MySQLPort == "" {
		c.MySQLPort = "3306"
	}
	if c.MySQLDatabase == "" {
		c.MySQLDatabase = "sharex"
	}
	if c.MySQLTLS == "" {
		c.MySQLTLS = "true"
	}
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:" + envOr("PORT", "8080")
	}
	if c.Port == "" {
		c.Port = envOr("PORT", "8080")
	}
	if c.CookieSecure == "" {
		if strings.HasPrefix(strings.ToLower(c.BaseURL), "https://") {
			c.CookieSecure = "true"
		} else {
			c.CookieSecure = "false"
		}
	}
	if c.AdminEmail == "" {
		c.AdminEmail = "admin@example.com"
	}
	if c.MaxUploadMB == "" {
		c.MaxUploadMB = "100"
	}
	if c.MaxStorageMB == "" {
		c.MaxStorageMB = "5120"
	}
	if c.SMTPPort == "" {
		c.SMTPPort = "587"
	}
	if c.SMTPFrom == "" {
		c.SMTPFrom = c.AdminEmail
	}
	if c.SMTPTLS == "" {
		c.SMTPTLS = "true"
	}
	return c
}

func (c AppConfig) databaseType() string {
	if strings.EqualFold(strings.TrimSpace(c.DatabaseType), "sqlite") {
		return "sqlite"
	}
	return "mysql"
}

func (c AppConfig) databaseDSN(dataDir string) string {
	if c.databaseType() == "sqlite" {
		path := strings.TrimSpace(c.SQLitePath)
		if path == "" {
			path = "sharex.db"
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(dataDir, path)
		}
		return path
	}
	return c.mysqlDSN()
}

func (c AppConfig) complete() bool {
	if c.databaseType() == "sqlite" {
		return c.AdminPassword != ""
	}
	databaseReady := c.MySQLDSN != "" || (c.MySQLHost != "" && c.MySQLUser != "" && c.MySQLPassword != "")
	return databaseReady && c.AdminPassword != ""
}

func configFromEnvironment() AppConfig {
	return AppConfig{
		DatabaseType:  os.Getenv("DATABASE_TYPE"),
		SQLitePath:    os.Getenv("SQLITE_PATH"),
		MySQLDSN:      os.Getenv("MYSQL_DSN"),
		MySQLHost:     os.Getenv("MYSQL_HOST"),
		MySQLPort:     os.Getenv("MYSQL_PORT"),
		MySQLDatabase: os.Getenv("MYSQL_DATABASE"),
		MySQLUser:     os.Getenv("MYSQL_USER"),
		MySQLPassword: os.Getenv("MYSQL_PASSWORD"),
		MySQLTLS:      os.Getenv("MYSQL_TLS"),
		BaseURL:       os.Getenv("BASE_URL"),
		Port:          os.Getenv("PORT"),
		CookieSecure:  os.Getenv("COOKIE_SECURE"),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		UploadAPIKey:  os.Getenv("UPLOAD_API_KEY"),
		AdminAPIKey:   os.Getenv("ADMIN_API_KEY"),
		CORSOrigin:    os.Getenv("CORS_ORIGIN"),
		MaxUploadMB:   os.Getenv("MAX_UPLOAD_MB"),
		MaxStorageMB:  os.Getenv("MAX_STORAGE_MB"),
		SMTPHost:      os.Getenv("SMTP_HOST"),
		SMTPPort:      os.Getenv("SMTP_PORT"),
		SMTPUser:      os.Getenv("SMTP_USER"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:      os.Getenv("SMTP_FROM"),
		SMTPTLS:       os.Getenv("SMTP_TLS"),
	}.withDefaults()
}

func applySMTPEnvironment(config AppConfig) AppConfig {
	if strings.TrimSpace(os.Getenv("SMTP_HOST")) == "" {
		return config
	}
	config.SMTPHost = strings.TrimSpace(os.Getenv("SMTP_HOST"))
	config.SMTPPort = strings.TrimSpace(os.Getenv("SMTP_PORT"))
	config.SMTPUser = strings.TrimSpace(os.Getenv("SMTP_USER"))
	config.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	config.SMTPFrom = strings.TrimSpace(os.Getenv("SMTP_FROM"))
	config.SMTPTLS = strings.TrimSpace(os.Getenv("SMTP_TLS"))
	return config.withDefaults()
}

func loadAppConfig(dataDir string) (AppConfig, bool, error) {
	if config := configFromEnvironment(); config.complete() {
		config = mergeStoredAPIKeys(dataDir, config)
		storedConfig := AppConfig{}
		configPath := filepath.Join(dataDir, "config.json")
		if data, err := os.ReadFile(configPath); err == nil {
			_ = json.Unmarshal(data, &storedConfig)
		} else if !errors.Is(err, os.ErrNotExist) {
			return AppConfig{}, false, fmt.Errorf("Setup-Konfiguration konnte nicht gelesen werden: %w", err)
		}
		keysChanged := false
		if config.UploadAPIKey == "" && config.UploadAPIKeyHash == "" {
			key, err := randomToken()
			if err != nil {
				return AppConfig{}, false, fmt.Errorf("Upload-Key konnte nicht erzeugt werden: %w", err)
			}
			config.UploadAPIKeyHash = hashAPIKey(key)
			keysChanged = true
		}
		if config.AdminAPIKey == "" && config.AdminAPIKeyHash == "" {
			key, err := randomToken()
			if err != nil {
				return AppConfig{}, false, fmt.Errorf("Admin-Key konnte nicht erzeugt werden: %w", err)
			}
			config.AdminAPIKeyHash = hashAPIKey(key)
			keysChanged = true
		}
		if keysChanged || storedConfig.UploadAPIKey != "" || storedConfig.AdminAPIKey != "" {
			storedConfig.UploadAPIKey = ""
			storedConfig.AdminAPIKey = ""
			storedConfig.UploadAPIKeyHash = apiKeyHash(config.UploadAPIKey, config.UploadAPIKeyHash)
			storedConfig.AdminAPIKeyHash = apiKeyHash(config.AdminAPIKey, config.AdminAPIKeyHash)
			if err := saveAppConfig(configPath, storedConfig); err != nil {
				return AppConfig{}, false, fmt.Errorf("API-Schlüssel konnten nicht sicher gespeichert werden: %w", err)
			}
		}
		return config, true, nil
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return AppConfig{}, false, nil
	}
	if err != nil {
		return AppConfig{}, false, fmt.Errorf("Setup-Konfiguration konnte nicht gelesen werden: %w", err)
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return AppConfig{}, false, fmt.Errorf("Setup-Konfiguration ist ungültig: %w", err)
	}
	keysMigrated := false
	if config.UploadAPIKeyHash == "" && config.UploadAPIKey != "" {
		config.UploadAPIKeyHash = hashAPIKey(config.UploadAPIKey)
		config.UploadAPIKey = ""
		keysMigrated = true
	}
	if config.AdminAPIKeyHash == "" && config.AdminAPIKey != "" {
		config.AdminAPIKeyHash = hashAPIKey(config.AdminAPIKey)
		config.AdminAPIKey = ""
		keysMigrated = true
	}
	if keysMigrated {
		if err := saveAppConfig(filepath.Join(dataDir, "config.json"), config); err != nil {
			return AppConfig{}, false, fmt.Errorf("API-Schlüssel konnten nicht sicher gespeichert werden: %w", err)
		}
	}
	config = config.withDefaults()
	config = applySMTPEnvironment(config)
	config = applyPublicURLEnvironment(config)
	return config, config.complete(), nil
}

func applyPublicURLEnvironment(config AppConfig) AppConfig {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/"); value != "" {
		config.BaseURL = value
	}
	if value := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); value != "" {
		config.CookieSecure = value
	}
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("CORS_ORIGIN")), "/"); value != "" {
		config.CORSOrigin = value
	}
	return config.withDefaults()
}

func mergeStoredAPIKeys(dataDir string, config AppConfig) AppConfig {
	data, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		return config
	}
	var stored AppConfig
	if err := json.Unmarshal(data, &stored); err != nil {
		return config
	}
	if config.UploadAPIKey == "" && config.UploadAPIKeyHash == "" {
		config.UploadAPIKey = stored.UploadAPIKey
		config.UploadAPIKeyHash = stored.UploadAPIKeyHash
	}
	if config.AdminAPIKey == "" && config.AdminAPIKeyHash == "" {
		config.AdminAPIKey = stored.AdminAPIKey
		config.AdminAPIKeyHash = stored.AdminAPIKeyHash
	}
	return config
}

func saveAppConfig(path string, config AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

type Store struct {
	db           *sql.DB
	mu           sync.RWMutex
	dataDir      string
	maxStorage   int64
	databaseType string
}

type AuthStore struct {
	db           *sql.DB
	databaseType string
}

type Mailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
	tls      bool
}

func newMailer(config AppConfig) *Mailer {
	config = config.withDefaults()
	if strings.TrimSpace(config.SMTPHost) == "" || strings.TrimSpace(config.SMTPFrom) == "" {
		return nil
	}
	return &Mailer{
		host:     strings.TrimSpace(config.SMTPHost),
		port:     strings.TrimSpace(config.SMTPPort),
		user:     config.SMTPUser,
		password: config.SMTPPassword,
		from:     strings.TrimSpace(config.SMTPFrom),
		tls:      !strings.EqualFold(config.SMTPTLS, "false") && config.SMTPTLS != "0",
	}
}

func (m *Mailer) send(to, subject, body string) error {
	if m == nil {
		return errors.New("SMTP ist nicht konfiguriert")
	}
	if strings.ContainsAny(to+subject+m.from, "\r\n") {
		return errors.New("ungültiger SMTP-Header")
	}
	message := []byte("From: " + m.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + mime.BEncoding.Encode("UTF-8", subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + body + "\r\n")
	address := net.JoinHostPort(m.host, m.port)
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.password, m.host)
	}
	if m.tls && m.port == "465" {
		return m.sendImplicitTLS(address, to, auth, message)
	}
	return smtp.SendMail(address, auth, m.from, []string{to}, message)
}

func (m *Mailer) sendImplicitTLS(address, to string, auth smtp.Auth, message []byte) error {
	connection, err := tls.Dial("tcp", address, &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(connection, m.host)
	if err != nil {
		connection.Close()
		return err
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

type loginAttempt struct {
	startedAt   time.Time
	failedCount int
}

type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

func newLoginLimiter() *LoginLimiter {
	return &LoginLimiter{attempts: make(map[string]loginAttempt)}
}

func (l *LoginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	attempt, ok := l.attempts[key]
	if !ok || now.Sub(attempt.startedAt) >= 15*time.Minute {
		l.attempts[key] = loginAttempt{startedAt: now, failedCount: 1}
		return true
	}
	if attempt.failedCount >= 10 {
		return false
	}
	attempt.failedCount++
	l.attempts[key] = attempt
	return true
}

func (l *LoginLimiter) success(key string) {
	l.mu.Lock()
	delete(l.attempts, key)
	l.mu.Unlock()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func newDatabase(databaseType, dsn string) (*sql.DB, error) {
	if strings.EqualFold(databaseType, "sqlite") {
		return newSQLiteDatabase(dsn)
	}
	return newMySQLDatabase(dsn)
}

func newMySQLDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL-Verbindung fehlgeschlagen: %w", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS files (
			id VARCHAR(32) NOT NULL PRIMARY KEY,
			user_id VARCHAR(64) NULL,
			name VARCHAR(255) NOT NULL,
			size BIGINT NOT NULL,
			content_type VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			uploaded_at DATETIME(6) NOT NULL,
			views BIGINT NOT NULL DEFAULT 0,
			password_hash VARCHAR(255) NULL,
			INDEX idx_files_uploaded_at (uploaded_at),
			INDEX idx_files_user_id (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS file_viewers (
			file_id VARCHAR(32) NOT NULL,
			visitor_hash CHAR(64) NOT NULL,
			viewed_at DATETIME(6) NOT NULL,
			PRIMARY KEY (file_id, visitor_hash),
			CONSTRAINT fk_file_viewers_file FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
			INDEX idx_file_viewers_viewed_at (viewed_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			role VARCHAR(32) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			password_salt VARCHAR(128) NOT NULL,
			created_at DATETIME(6) NOT NULL,
			max_upload_bytes BIGINT NOT NULL DEFAULT 104857600,
			max_storage_bytes BIGINT NOT NULL DEFAULT 5368709120,
			unlimited_storage BOOLEAN NOT NULL DEFAULT FALSE,
			glow_color VARCHAR(7) NOT NULL DEFAULT '#ffffff',
			INDEX idx_users_role (role)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token CHAR(64) NOT NULL PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			expires_at DATETIME(6) NOT NULL,
			created_at DATETIME(6) NOT NULL,
			CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_sessions_expires_at (expires_at),
			INDEX idx_sessions_user_id (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS password_resets (
			token_hash CHAR(64) NOT NULL PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			expires_at DATETIME(6) NOT NULL,
			created_at DATETIME(6) NOT NULL,
			CONSTRAINT fk_password_resets_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_password_resets_expires_at (expires_at),
			INDEX idx_password_resets_user_id (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS login_ip_history (
			user_id VARCHAR(64) NOT NULL,
			ip_address VARCHAR(64) NOT NULL,
			first_seen DATETIME(6) NOT NULL,
			last_seen DATETIME(6) NOT NULL,
			last_alerted_at DATETIME(6) NULL,
			PRIMARY KEY (user_id, ip_address),
			CONSTRAINT fk_login_ip_history_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_login_ip_history_last_seen (last_seen)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("MySQL-Schema konnte nicht angelegt werden: %w", err)
		}
	}
	for _, statement := range []string{
		`ALTER TABLE files ADD COLUMN user_id VARCHAR(64) NULL`,
		`ALTER TABLE files ADD INDEX idx_files_user_id (user_id)`,
		`ALTER TABLE files ADD COLUMN views BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE files ADD COLUMN password_hash VARCHAR(255) NULL`,
		`ALTER TABLE files ADD COLUMN imgur_url VARCHAR(512) NULL`,
		`ALTER TABLE users ADD COLUMN max_upload_bytes BIGINT NOT NULL DEFAULT 104857600`,
		`ALTER TABLE users ADD COLUMN max_storage_bytes BIGINT NOT NULL DEFAULT 5368709120`,
		`ALTER TABLE users ADD COLUMN unlimited_storage BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE users ADD COLUMN glow_color VARCHAR(7) NOT NULL DEFAULT '#ffffff'`,
	} {
		if _, err := db.Exec(statement); err != nil && !isDuplicateSchemaError(err) {
			db.Close()
			return nil, fmt.Errorf("MySQL-Schema-Migration fehlgeschlagen: %w", err)
		}
	}
	return db, nil
}

func newSQLiteDatabase(path string) (*sql.DB, error) {
	if directory := filepath.Dir(path); directory != "." {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, fmt.Errorf("SQLite-Verzeichnis konnte nicht angelegt werden: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("SQLite konnte nicht aktiviert werden: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("SQLite-Verbindung fehlgeschlagen: %w", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS files (id TEXT NOT NULL PRIMARY KEY, user_id TEXT NULL, name TEXT NOT NULL, size INTEGER NOT NULL, content_type TEXT NOT NULL, url TEXT NOT NULL, uploaded_at DATETIME NOT NULL, views INTEGER NOT NULL DEFAULT 0, password_hash TEXT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files (uploaded_at)`,
		`CREATE TABLE IF NOT EXISTS file_viewers (file_id TEXT NOT NULL, visitor_hash TEXT NOT NULL, viewed_at DATETIME NOT NULL, PRIMARY KEY (file_id, visitor_hash), FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS idx_file_viewers_viewed_at ON file_viewers (viewed_at)`,
		`CREATE TABLE IF NOT EXISTS users (id TEXT NOT NULL PRIMARY KEY, email TEXT NOT NULL UNIQUE, name TEXT NOT NULL, role TEXT NOT NULL, password_hash TEXT NOT NULL, password_salt TEXT NOT NULL, created_at DATETIME NOT NULL, max_upload_bytes INTEGER NOT NULL DEFAULT 104857600, max_storage_bytes INTEGER NOT NULL DEFAULT 5368709120, unlimited_storage INTEGER NOT NULL DEFAULT 0, glow_color TEXT NOT NULL DEFAULT '#ffffff')`,
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users (role)`,
		`CREATE TABLE IF NOT EXISTS sessions (token TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id)`,
		`CREATE TABLE IF NOT EXISTS password_resets (token_hash TEXT NOT NULL PRIMARY KEY, user_id TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS idx_password_resets_expires_at ON password_resets (expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets (user_id)`,
		`CREATE TABLE IF NOT EXISTS login_ip_history (user_id TEXT NOT NULL, ip_address TEXT NOT NULL, first_seen DATETIME NOT NULL, last_seen DATETIME NOT NULL, last_alerted_at DATETIME NULL, PRIMARY KEY (user_id, ip_address), FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS idx_login_ip_history_last_seen ON login_ip_history (last_seen)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return nil, fmt.Errorf("SQLite-Schema konnte nicht angelegt werden: %w", err)
		}
	}
	for _, statement := range []string{
		`ALTER TABLE files ADD COLUMN user_id TEXT NULL`,
		`ALTER TABLE files ADD COLUMN views INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE files ADD COLUMN password_hash TEXT NULL`,
		`ALTER TABLE files ADD COLUMN imgur_url TEXT NULL`,
		`ALTER TABLE users ADD COLUMN max_upload_bytes INTEGER NOT NULL DEFAULT 104857600`,
		`ALTER TABLE users ADD COLUMN max_storage_bytes INTEGER NOT NULL DEFAULT 5368709120`,
		`ALTER TABLE users ADD COLUMN unlimited_storage INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN glow_color TEXT NOT NULL DEFAULT '#ffffff'`,
	} {
		if _, err := db.Exec(statement); err != nil && !isSQLiteSchemaError(err) {
			db.Close()
			return nil, fmt.Errorf("SQLite-Schema-Migration fehlgeschlagen: %w", err)
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_files_user_id ON files (user_id)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("SQLite-Dateiindex konnte nicht angelegt werden: %w", err)
	}
	return db, nil
}

func isDuplicateSchemaError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == 1060 || mysqlErr.Number == 1061
}

func isSQLiteSchemaError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column name") || strings.Contains(message, "already exists")
}

var errStorageLimit = errors.New("storage limit reached")

func newStore(db *sql.DB, dataDir string, maxStorage int64, databaseType string) (*Store, error) {
	store := &Store{db: db, dataDir: dataDir, maxStorage: maxStorage, databaseType: databaseType}
	if err := os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755); err != nil {
		return nil, err
	}
	if err := migrateLegacyIndex(db, dataDir, databaseType); err != nil {
		return nil, err
	}
	return store, nil
}

func migrateLegacyIndex(db *sql.DB, dataDir, databaseType string) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "index.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var files map[string]FileRecord
	if len(data) > 0 {
		if err := json.Unmarshal(data, &files); err != nil {
			return fmt.Errorf("index.json lesen: %w", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	insertStatement := `INSERT IGNORE INTO files (id, name, size, content_type, url, uploaded_at) VALUES (?, ?, ?, ?, ?, ?)`
	if strings.EqualFold(databaseType, "sqlite") {
		insertStatement = `INSERT OR IGNORE INTO files (id, name, size, content_type, url, uploaded_at) VALUES (?, ?, ?, ?, ?, ?)`
	}
	for id, file := range files {
		if file.ID == "" {
			file.ID = id
		}
		if file.UploadedAt.IsZero() {
			file.UploadedAt = time.Now().UTC()
		}
		if _, err := tx.Exec(
			insertStatement,
			file.ID, file.Name, file.Size, file.ContentType, file.URL, file.UploadedAt,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("alte Datei %s migrieren: %w", file.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if len(files) > 0 {
		log.Printf("%d Dateien aus index.json nach %s migriert", len(files), databaseType)
	}
	return nil
}

func (s *Store) list() []FileRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, COALESCE(user_id, ''), name, size, content_type, url, uploaded_at, views, COALESCE(password_hash, '') FROM files ORDER BY uploaded_at DESC`)
	if err != nil {
		log.Printf("Dateiliste konnte nicht gelesen werden: %v", err)
		return []FileRecord{}
	}
	defer rows.Close()

	items := make([]FileRecord, 0)
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			log.Printf("Datei konnte nicht gelesen werden: %v", err)
			continue
		}
		items = append(items, file)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Dateiliste konnte nicht gelesen werden: %v", err)
	}
	return items
}

func (s *Store) listForUser(userID string) []FileRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, COALESCE(user_id, ''), name, size, content_type, url, uploaded_at, views, COALESCE(password_hash, '') FROM files WHERE user_id = ? ORDER BY uploaded_at DESC`, userID)
	if err != nil {
		log.Printf("Benutzer-Dateiliste konnte nicht gelesen werden: %v", err)
		return []FileRecord{}
	}
	defer rows.Close()

	items := make([]FileRecord, 0)
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			log.Printf("Benutzer-Datei konnte nicht gelesen werden: %v", err)
			continue
		}
		items = append(items, file)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Benutzer-Dateiliste konnte nicht gelesen werden: %v", err)
	}
	return items
}

func (s *Store) find(id string) (FileRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var file FileRecord
	err := s.db.QueryRow(`SELECT id, COALESCE(user_id, ''), name, size, content_type, url, uploaded_at, views, COALESCE(password_hash, '') FROM files WHERE id = ?`, id).Scan(&file.ID, &file.UserID, &file.Name, &file.Size, &file.ContentType, &file.URL, &file.UploadedAt, &file.Views, &file.PasswordHash)
	file.PasswordProtected = file.PasswordHash != ""
	return file, err == nil
}

func (s *Store) glowColorForFile(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var color string
	// Branding is controlled centrally by the primary administrator.
	if err := s.db.QueryRow(`SELECT glow_color FROM users WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1`).Scan(&color); err != nil {
		return defaultGlowColor
	}
	return normalizeGlowColor(color)
}

func scanFile(scanner rowScanner) (FileRecord, error) {
	var file FileRecord
	err := scanner.Scan(&file.ID, &file.UserID, &file.Name, &file.Size, &file.ContentType, &file.URL, &file.UploadedAt, &file.Views, &file.PasswordHash)
	file.PasswordProtected = file.PasswordHash != ""
	return file, err
}

func (s *Store) add(file FileRecord, source io.Reader, extension, userID string, storageLimit int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if storageLimit > 0 {
		var total int64
		query := `SELECT COALESCE(SUM(size), 0) FROM files`
		args := []any{}
		if userID != "" {
			query += ` WHERE user_id = ?`
			args = append(args, userID)
		}
		if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
			return err
		}
		if total+file.Size > storageLimit {
			return errStorageLimit
		}
	}

	path := filepath.Join(s.dataDir, "uploads", file.ID+extension)
	output, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, source); err != nil {
		output.Close()
		os.Remove(path)
		return err
	}
	if err := output.Close(); err != nil {
		os.Remove(path)
		return err
	}

	_, err = s.db.Exec(
		`INSERT INTO files (id, user_id, name, size, content_type, url, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		file.ID, nullableString(userID), file.Name, file.Size, file.ContentType, file.URL, file.UploadedAt,
	)
	if err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

func (s *Store) registerView(id, visitorHash string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Count each visitor (IP hash) only once per file.
	insertQuery := `INSERT OR IGNORE INTO file_viewers (file_id, visitor_hash, viewed_at) VALUES (?, ?, ?)`
	if s.databaseType != "sqlite" {
		insertQuery = `INSERT IGNORE INTO file_viewers (file_id, visitor_hash, viewed_at) VALUES (?, ?, ?)`
	}
	result, err := tx.Exec(insertQuery, id, visitorHash, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if inserted > 0 {
		if _, err := tx.Exec(`UPDATE files SET views = views + 1 WHERE id = ?`, id); err != nil {
			return 0, err
		}
	}
	var views int64
	if err := tx.QueryRow(`SELECT views FROM files WHERE id = ?`, id).Scan(&views); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return views, nil
}

func (s *Store) setPassword(id, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`UPDATE files SET password_hash = ? WHERE id = ?`, nullableString(passwordHash), id)
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s *Store) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var file FileRecord
	err := s.db.QueryRow(
		`SELECT id, COALESCE(user_id, ''), name, size, content_type, url, uploaded_at, views, COALESCE(password_hash, '') FROM files WHERE id = ?`,
		id,
	).Scan(&file.ID, &file.UserID, &file.Name, &file.Size, &file.ContentType, &file.URL, &file.UploadedAt, &file.Views, &file.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return os.ErrNotExist
	}
	if err != nil {
		return err
	}

	if _, err := s.db.Exec(`DELETE FROM files WHERE id = ?`, id); err != nil {
		return err
	}
	for _, candidate := range allowedExtensions(file.Name, file.ContentType) {
		path := filepath.Join(s.dataDir, "uploads", id+candidate)
		if err := os.Remove(path); err == nil || !errors.Is(err, os.ErrNotExist) {
			break
		}
	}
	return nil
}

func newAuthStore(db *sql.DB, dataDir, databaseType string) (*AuthStore, error) {
	auth := &AuthStore{db: db, databaseType: databaseType}
	if err := migrateLegacyUsers(db, dataDir, databaseType); err != nil {
		return nil, err
	}
	return auth, nil
}

func migrateLegacyUsers(db *sql.DB, dataDir, databaseType string) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "users.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var users map[string]User
	if len(data) > 0 {
		if err := json.Unmarshal(data, &users); err != nil {
			return fmt.Errorf("users.json lesen: %w", err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	insertStatement := `INSERT IGNORE INTO users (id, email, name, role, password_hash, password_salt, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	if strings.EqualFold(databaseType, "sqlite") {
		insertStatement = `INSERT OR IGNORE INTO users (id, email, name, role, password_hash, password_salt, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}
	for email, user := range users {
		if user.Email == "" {
			user.Email = email
		}
		if user.Name == "" {
			user.Name = user.Email
		}
		if user.Role == "" {
			user.Role = "user"
		}
		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now().UTC()
		}
		if user.PasswordHash == "" || user.PasswordSalt == "" {
			continue
		}
		if _, err := tx.Exec(
			insertStatement,
			user.ID, normalizeEmail(user.Email), user.Name, user.Role, user.PasswordHash, user.PasswordSalt, user.CreatedAt,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("alten Benutzer %s migrieren: %w", user.Email, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if len(users) > 0 {
		log.Printf("%d Benutzer aus users.json nach %s migriert", len(users), databaseType)
	}
	return nil
}

func (a *AuthStore) ensureAdmin(email, password string) error {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return errors.New("admin email und passwort dürfen nicht leer sein")
	}

	var id string
	err := a.db.QueryRow(`SELECT id FROM users WHERE role = 'admin' LIMIT 1`).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	salt, err := randomID()
	if err != nil {
		return err
	}
	admin := User{
		ID:           "usr_" + salt,
		Email:        email,
		Name:         "Administrator",
		Role:         "admin",
		PasswordSalt: salt,
		PasswordHash: mustHashPassword(password),
		CreatedAt:    time.Now().UTC(),
	}
	_, err = a.db.Exec(
		`INSERT INTO users (id, email, name, role, password_hash, password_salt, created_at, max_upload_bytes, max_storage_bytes, unlimited_storage) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		admin.ID, admin.Email, admin.Name, admin.Role, admin.PasswordHash, admin.PasswordSalt, admin.CreatedAt, defaultMaxUploadBytes, defaultMaxStorageBytes, false,
	)
	return err
}

func (a *AuthStore) resetAdminPassword(email, password string) error {
	if len([]rune(password)) < 12 || len([]rune(password)) > 256 {
		return errors.New("das Passwort muss zwischen 12 und 256 Zeichen haben")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var userID string
	err = a.db.QueryRow(`SELECT id FROM users WHERE role = 'admin' AND email = ? LIMIT 1`, normalizeEmail(email)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = a.db.QueryRow(`SELECT id FROM users WHERE role = 'admin' LIMIT 1`).Scan(&userID)
	}
	if err != nil {
		return err
	}
	if _, err := a.db.Exec(`UPDATE users SET password_hash = ?, password_salt = '' WHERE id = ?`, string(hash), userID); err != nil {
		return err
	}
	_, err = a.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (a *AuthStore) checkPassword(email, password string) (User, bool) {
	user, err := scanUser(a.db.QueryRow(
		`SELECT id, email, name, role, password_hash, password_salt, created_at, max_upload_bytes, max_storage_bytes, unlimited_storage, glow_color FROM users WHERE email = ?`,
		normalizeEmail(email),
	))
	if err != nil {
		return User{}, false
	}
	storedHash := strings.TrimSpace(user.PasswordHash)
	valid := false
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		valid = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
	} else {
		actual := legacyHashPassword(password, user.PasswordSalt)
		valid = subtle.ConstantTimeCompare([]byte(actual), []byte(storedHash)) == 1
		if valid {
			if upgraded, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); err == nil {
				_, _ = a.db.Exec(`UPDATE users SET password_hash = ?, password_salt = '' WHERE id = ?`, string(upgraded), user.ID)
			}
		}
	}
	return user, valid
}

func (a *AuthStore) findUserByEmail(email string) (User, bool) {
	user, err := scanUser(a.db.QueryRow(
		`SELECT id, email, name, role, '', '', created_at, max_upload_bytes, max_storage_bytes, unlimited_storage, glow_color FROM users WHERE email = ?`,
		normalizeEmail(email),
	))
	if err != nil {
		return User{}, false
	}
	return user, true
}

func (a *AuthStore) primaryAdmin() (User, bool) {
	user, err := scanUser(a.db.QueryRow(
		`SELECT id, email, name, role, password_hash, password_salt, created_at, max_upload_bytes, max_storage_bytes, unlimited_storage, glow_color
		 FROM users WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1`,
	))
	if err != nil {
		return User{}, false
	}
	return user, true
}

func (a *AuthStore) listUsers() ([]User, error) {
	rows, err := a.db.Query(`SELECT id, email, name, role, '', '', created_at, max_upload_bytes, max_storage_bytes, unlimited_storage, glow_color FROM users ORDER BY email ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (a *AuthStore) createUser(email, name, password string, maxUploadBytes, maxStorageBytes int64, unlimitedStorage bool) (User, error) {
	email = normalizeEmail(email)
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}
	if email == "" || password == "" {
		return User{}, errors.New("email und passwort dürfen nicht leer sein")
	}
	if len([]rune(password)) < 12 {
		return User{}, errors.New("das passwort muss mindestens 12 zeichen haben")
	}
	if maxUploadBytes < 1 || maxStorageBytes < 1 {
		return User{}, errors.New("limits müssen größer als null sein")
	}
	id, err := randomID()
	if err != nil {
		return User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	user := User{
		ID:               "usr_" + id,
		Email:            email,
		Name:             name,
		Role:             "user",
		PasswordHash:     string(hash),
		CreatedAt:        time.Now().UTC(),
		MaxUploadBytes:   maxUploadBytes,
		MaxStorageBytes:  maxStorageBytes,
		UnlimitedStorage: unlimitedStorage,
	}
	_, err = a.db.Exec(
		`INSERT INTO users (id, email, name, role, password_hash, password_salt, created_at, max_upload_bytes, max_storage_bytes, unlimited_storage) VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?)`,
		user.ID, user.Email, user.Name, user.Role, user.PasswordHash, user.CreatedAt, user.MaxUploadBytes, user.MaxStorageBytes, user.UnlimitedStorage,
	)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (a *AuthStore) updateUserLimits(userID string, maxUploadBytes, maxStorageBytes int64, unlimitedStorage bool) error {
	if maxUploadBytes < 1 || maxStorageBytes < 1 {
		return errors.New("limits müssen größer als null sein")
	}
	var existingID string
	if err := a.db.QueryRow(`SELECT id FROM users WHERE id = ?`, userID).Scan(&existingID); err != nil {
		return err
	}
	_, err := a.db.Exec(
		`UPDATE users SET max_upload_bytes = ?, max_storage_bytes = ?, unlimited_storage = ? WHERE id = ?`,
		maxUploadBytes, maxStorageBytes, unlimitedStorage, userID,
	)
	return err
}

func (a *AuthStore) updateGlowColor(userID, glowColor string) error {
	_, err := a.db.Exec(`UPDATE users SET glow_color = ? WHERE id = ?`, normalizeGlowColor(glowColor), userID)
	return err
}

func (a *AuthStore) updateProfile(userID, name, email string) (User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if length := len([]rune(name)); length < 2 || length > 80 {
		return User{}, errors.New("der Name muss zwischen 2 und 80 Zeichen haben")
	}
	if len(email) < 3 || len(email) > 254 {
		return User{}, errors.New("bitte eine gültige E-Mail-Adresse eingeben")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(parsed.Address, email) {
		return User{}, errors.New("bitte eine gültige E-Mail-Adresse eingeben")
	}
	var existingID string
	err = a.db.QueryRow(`SELECT id FROM users WHERE email = ? AND id <> ? LIMIT 1`, email, userID).Scan(&existingID)
	if err == nil {
		return User{}, errors.New("diese E-Mail-Adresse wird bereits verwendet")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	if _, err := a.db.Exec(`UPDATE users SET name = ?, email = ? WHERE id = ?`, name, email, userID); err != nil {
		return User{}, err
	}
	return scanUser(a.db.QueryRow(
		`SELECT id, email, name, role, password_hash, password_salt, created_at, max_upload_bytes, max_storage_bytes, unlimited_storage, glow_color FROM users WHERE id = ?`,
		userID,
	))
}

func (a *AuthStore) updatePassword(userID, currentPassword, newPassword string) error {
	var email string
	if err := a.db.QueryRow(`SELECT email FROM users WHERE id = ?`, userID).Scan(&email); err != nil {
		return err
	}
	user, valid := a.checkPassword(email, currentPassword)
	if !valid || user.ID != userID {
		return errors.New("das aktuelle Passwort ist falsch")
	}
	if currentPassword == newPassword {
		return errors.New("das neue Passwort muss sich vom aktuellen unterscheiden")
	}
	if length := len([]rune(newPassword)); length < 12 || length > 256 {
		return errors.New("das Passwort muss zwischen 12 und 256 Zeichen haben")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(`UPDATE users SET password_hash = ?, password_salt = '' WHERE id = ?`, string(hash), userID)
	return err
}

func (a *AuthStore) deleteOtherSessions(r *http.Request, userID string) {
	cookie, err := r.Cookie("drop_session")
	if err != nil || cookie.Value == "" {
		_, _ = a.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
		return
	}
	_, _ = a.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token <> ?`, userID, hashSessionToken(cookie.Value))
}

func scanUser(scanner rowScanner) (User, error) {
	var user User
	err := scanner.Scan(
		&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash, &user.PasswordSalt,
		&user.CreatedAt, &user.MaxUploadBytes, &user.MaxStorageBytes, &user.UnlimitedStorage, &user.GlowColor,
	)
	if user.GlowColor == "" {
		user.GlowColor = defaultGlowColor
	}
	return user, err
}

func normalizeGlowColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 7 || value[0] != '#' {
		return defaultGlowColor
	}
	if _, err := strconv.ParseUint(value[1:], 16, 24); err != nil {
		return defaultGlowColor
	}
	return value
}

func glowShadowColors(value string) (string, string) {
	value = normalizeGlowColor(value)
	parsed, _ := strconv.ParseUint(value[1:], 16, 24)
	red := (parsed >> 16) & 0xff
	green := (parsed >> 8) & 0xff
	blue := parsed & 0xff
	return fmt.Sprintf("rgba(%d,%d,%d,.95)", red, green, blue), fmt.Sprintf("rgba(%d,%d,%d,.42)", red, green, blue)
}

func (a *AuthStore) createPasswordReset(userID string) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(time.Hour)
	if _, err := a.db.Exec(`DELETE FROM password_resets WHERE expires_at <= ? OR user_id = ?`, now, userID); err != nil {
		return "", time.Time{}, err
	}
	_, err = a.db.Exec(
		`INSERT INTO password_resets (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		hashSessionToken(token), userID, expires, now,
	)
	return token, expires, err
}

func (a *AuthStore) resetPassword(token, password string) bool {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false
	}
	tx, err := a.db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRow(
		`SELECT user_id FROM password_resets WHERE token_hash = ? AND expires_at > ?`,
		hashSessionToken(token), time.Now().UTC(),
	).Scan(&userID)
	if err != nil {
		return false
	}
	if _, err := tx.Exec(`UPDATE users SET password_hash = ?, password_salt = '' WHERE id = ?`, string(hash), userID); err != nil {
		return false
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return false
	}
	if _, err := tx.Exec(`DELETE FROM password_resets WHERE user_id = ?`, userID); err != nil {
		return false
	}
	return tx.Commit() == nil
}

func (a *AuthStore) recordLoginIP(userID, ip string) (bool, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now().UTC()
	tx, err := a.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var firstSeen time.Time
	var lastAlerted sql.NullTime
	loginIPQuery := `SELECT first_seen, last_alerted_at FROM login_ip_history WHERE user_id = ? AND ip_address = ?`
	if strings.EqualFold(a.databaseType, "mysql") {
		loginIPQuery += " FOR UPDATE"
	}
	err = tx.QueryRow(
		loginIPQuery,
		userID, ip,
	).Scan(&firstSeen, &lastAlerted)
	if err == nil {
		_, err = tx.Exec(`UPDATE login_ip_history SET last_seen = ? WHERE user_id = ? AND ip_address = ?`, now, userID, ip)
		if err != nil {
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	var knownIPs int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM login_ip_history WHERE user_id = ?`, userID).Scan(&knownIPs); err != nil {
		return false, err
	}
	var lastAlert sql.NullTime
	if err := tx.QueryRow(`SELECT MAX(last_alerted_at) FROM login_ip_history WHERE user_id = ?`, userID).Scan(&lastAlert); err != nil {
		return false, err
	}
	shouldAlert := knownIPs > 0 && (!lastAlert.Valid || now.Sub(lastAlert.Time) >= 24*time.Hour)
	var alertedAt any
	if shouldAlert {
		alertedAt = now
	}
	if _, err := tx.Exec(
		`INSERT INTO login_ip_history (user_id, ip_address, first_seen, last_seen, last_alerted_at) VALUES (?, ?, ?, ?, ?)`,
		userID, ip, now, now, alertedAt,
	); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return shouldAlert, nil
}

func (a *AuthStore) createSession(userID string) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if _, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now); err != nil {
		return "", time.Time{}, err
	}
	_, err = a.db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		hashSessionToken(token), userID, expires, now,
	)
	return token, expires, err
}

func (a *AuthStore) userFromRequest(r *http.Request) (User, bool) {
	cookie, err := r.Cookie("drop_session")
	if err != nil || cookie.Value == "" {
		return User{}, false
	}

	var user User
	user, err = scanUser(a.db.QueryRow(
		`SELECT u.id, u.email, u.name, u.role, u.password_hash, u.password_salt, u.created_at, u.max_upload_bytes, u.max_storage_bytes, u.unlimited_storage, u.glow_color
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token = ? AND s.expires_at > ?`,
		hashSessionToken(cookie.Value), time.Now().UTC(),
	))
	if err != nil {
		return User{}, false
	}
	return user, true
}

func (a *AuthStore) deleteSession(r *http.Request) {
	cookie, err := r.Cookie("drop_session")
	if err != nil {
		return
	}
	_, _ = a.db.Exec(`DELETE FROM sessions WHERE token = ?`, hashSessionToken(cookie.Value))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func legacyHashPassword(password, salt string) string {
	digest := sha256.Sum256([]byte(salt + ":" + password))
	for i := 0; i < 120000; i++ {
		next := sha256.New()
		next.Write(digest[:])
		next.Write([]byte(salt))
		digest = sha256.Sum256(next.Sum(nil))
	}
	return hex.EncodeToString(digest[:])
}

func mustHashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}

func hashSessionToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// apiKeyPepper optionally comes from API_KEY_PEPPER. Empty keeps legacy SHA-256(key) hashes
// so existing installs and GitHub installer setups stay compatible until a pepper is set.
func apiKeyPepper() string {
	return strings.TrimSpace(os.Getenv("API_KEY_PEPPER"))
}

func hashAPIKey(value string) string {
	digest := sha256.Sum256([]byte(apiKeyPepper() + value))
	return hex.EncodeToString(digest[:])
}

func apiKeyHash(plain, storedHash string) string {
	if strings.TrimSpace(storedHash) != "" {
		return strings.TrimSpace(storedHash)
	}
	if strings.TrimSpace(plain) != "" {
		return hashAPIKey(plain)
	}
	return ""
}

func allowedExtensions(name, contentType string) []string {
	ext := filepath.Ext(name)
	if ext == "" {
		if guessed := extensionForType(contentType); guessed != "" {
			return []string{guessed, ""}
		}
		return []string{""}
	}
	return []string{strings.ToLower(ext), ""}
}

func extensionForType(contentType string) string {
	if value, err := mime.ExtensionsByType(contentType); err == nil && len(value) > 0 {
		return value[0]
	}
	return ""
}

type Server struct {
	stateMu       sync.RWMutex
	setupMu       sync.Mutex
	store         *Store
	auth          *AuthStore
	mailer        *Mailer
	configured    bool
	dataDir       string
	configPath    string
	baseURL       string
	reportEmail   string
	databaseType  string
	apiKey        string
	adminAPIKey   string
	corsOrigin    string
	frontend      http.Handler
	maxUpload     int64
	secureCookies bool
	brandLogoPath string
	embed         embedOptions
	loginLimiter  *LoginLimiter
	setupLimiter  *LoginLimiter
	resetLimiter  *LoginLimiter
	filePasswordLimiter *LoginLimiter
}

type serverSnapshot struct {
	store         *Store
	auth          *AuthStore
	mailer        *Mailer
	configured    bool
	baseURL       string
	reportEmail   string
	databaseType  string
	apiKey        string
	adminAPIKey   string
	corsOrigin    string
	maxUpload     int64
	secureCookies bool
	brandLogoPath string
	embed         embedOptions
}

func (s *Server) snapshot() serverSnapshot {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return serverSnapshot{
		store:         s.store,
		auth:          s.auth,
		mailer:        s.mailer,
		configured:    s.configured,
		baseURL:       s.baseURL,
		reportEmail:   s.reportEmail,
		databaseType:  s.databaseType,
		apiKey:        s.apiKey,
		adminAPIKey:   s.adminAPIKey,
		corsOrigin:    s.corsOrigin,
		maxUpload:     s.maxUpload,
		secureCookies: s.secureCookies,
		brandLogoPath: s.brandLogoPath,
		embed:         s.embed,
	}
}

func (s *Server) activate(config AppConfig, store *Store, auth *AuthStore) {
	config = config.withDefaults()
	maxUploadMB, _ := strconv.ParseInt(config.MaxUploadMB, 10, 64)
	if maxUploadMB < 1 {
		maxUploadMB = 100
	}
	secureCookies := !strings.EqualFold(config.CookieSecure, "false") && config.CookieSecure != "0"

	s.stateMu.Lock()
	s.store = store
	s.auth = auth
	s.mailer = newMailer(config)
	s.configured = true
	s.baseURL = config.BaseURL
	s.reportEmail = config.AdminEmail
	s.databaseType = config.databaseType()
	s.apiKey = apiKeyHash(config.UploadAPIKey, config.UploadAPIKeyHash)
	s.adminAPIKey = apiKeyHash(config.AdminAPIKey, config.AdminAPIKeyHash)
	s.corsOrigin = strings.TrimRight(config.CORSOrigin, "/")
	s.maxUpload = maxUploadMB * 1024 * 1024
	s.secureCookies = secureCookies
	s.brandLogoPath = brandLogoPath(s.dataDir, config.BrandLogo)
	s.embed = embedOptionsFromConfig(config)
	s.stateMu.Unlock()
}

func (s *Server) isConfigured() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.configured
}

func originFromURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (s *Server) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	snapshot := s.snapshot()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); forwardedProto == "http" || forwardedProto == "https" {
		scheme = forwardedProto
	}
	requestOrigin := scheme + "://" + r.Host
	if !snapshot.configured {
		return origin == requestOrigin
	}
	return origin == requestOrigin || origin == originFromURL(snapshot.baseURL) || (snapshot.corsOrigin != "" && origin == strings.TrimRight(snapshot.corsOrigin, "/"))
}

func (s *Server) withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := s.snapshot()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		if snapshot.secureCookies {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !s.originAllowed(r) {
				if r.Method == http.MethodOptions {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
					return
				}
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Admin-Key, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Add("Vary", "Origin")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureKeyMatch(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func (s *Server) apiKeyMatches(r *http.Request) bool {
	snapshot := s.snapshot()
	return secureKeyMatch(hashAPIKey(r.Header.Get("X-API-Key")), snapshot.apiKey) || secureKeyMatch(hashAPIKey(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), snapshot.apiKey)
}

func (s *Server) adminAPIKeyMatches(r *http.Request) bool {
	snapshot := s.snapshot()
	return secureKeyMatch(hashAPIKey(r.Header.Get("X-Admin-Key")), snapshot.adminAPIKey) || secureKeyMatch(hashAPIKey(strings.TrimPrefix(r.Header.Get("Authorization"), "Admin ")), snapshot.adminAPIKey)
}

func (s *Server) authenticate(r *http.Request) bool {
	if s.apiKeyMatches(r) {
		return true
	}
	snapshot := s.snapshot()
	if snapshot.auth == nil {
		return false
	}
	_, ok := snapshot.auth.userFromRequest(r)
	return ok
}

func (s *Server) hasAccess(r *http.Request) bool {
	if s.apiKeyMatches(r) {
		return true
	}
	snapshot := s.snapshot()
	if snapshot.auth == nil {
		return false
	}
	_, ok := snapshot.auth.userFromRequest(r)
	return ok
}

func (s *Server) isAdmin(r *http.Request) bool {
	if s.adminAPIKeyMatches(r) {
		return true
	}
	snapshot := s.snapshot()
	if snapshot.auth == nil {
		return false
	}
	user, ok := snapshot.auth.userFromRequest(r)
	return ok && user.Role == "admin"
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete) && !s.originAllowed(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
		return
	}
	if !s.isConfigured() && r.URL.Path != "/api/setup/status" && r.URL.Path != "/api/setup" && r.URL.Path != "/api/health" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
		return
	}
	snapshot := s.snapshot()
	switch {
	case r.URL.Path == "/api/setup/status" && r.Method == http.MethodGet:
		s.handleSetupStatus(w)
	case r.URL.Path == "/api/setup" && r.Method == http.MethodPost:
		s.handleSetup(w, r)
	case r.URL.Path == "/api/auth/login" && r.Method == http.MethodPost:
		s.handleLogin(w, r)
	case r.URL.Path == "/api/auth/forgot-password" && r.Method == http.MethodPost:
		s.handleForgotPassword(w, r)
	case r.URL.Path == "/api/auth/reset-password" && r.Method == http.MethodPost:
		s.handleResetPassword(w, r)
	case r.URL.Path == "/api/auth/logout" && r.Method == http.MethodPost:
		s.handleLogout(w, r)
	case r.URL.Path == "/api/auth/me" && r.Method == http.MethodGet:
		s.handleCurrentUser(w, r)
	case r.URL.Path == "/api/account/settings" && (r.Method == http.MethodGet || r.Method == http.MethodPut):
		if snapshot.auth == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		if _, ok := snapshot.auth.userFromRequest(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		if r.Method == http.MethodGet {
			s.handleAccountSettings(w, r)
		} else {
			s.handleUpdateAccountSettings(w, r)
		}
	case r.URL.Path == "/api/account/profile" && r.Method == http.MethodPut:
		if snapshot.auth == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		user, ok := snapshot.auth.userFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		s.handleUpdateProfile(w, r, user)
	case r.URL.Path == "/api/account/password" && r.Method == http.MethodPut:
		if snapshot.auth == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		user, ok := snapshot.auth.userFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		s.handleChangePassword(w, r, user)
	case r.URL.Path == "/api/branding/logo" && r.Method == http.MethodGet:
		s.handleBrandLogo(w, r)
	case r.URL.Path == "/api/health" && r.Method == http.MethodGet:
		s.handleHealth(w, r)
	case r.URL.Path == "/api/admin/users" && r.Method == http.MethodGet:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleAdminUsers(w)
	case r.URL.Path == "/api/admin/users" && r.Method == http.MethodPost:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleCreateAdminUser(w, r)
	case r.URL.Path == "/api/admin/keys" && r.Method == http.MethodPost:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleRotateAPIKey(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/admin/users/") && r.Method == http.MethodPost:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleUpdateAdminUserLimits(w, r)
	case r.URL.Path == "/api/admin/overview" && r.Method == http.MethodGet:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleAdminOverview(w)
	case r.URL.Path == "/api/admin/branding" && (r.Method == http.MethodGet || r.Method == http.MethodPut || r.Method == http.MethodDelete):
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		s.handleAdminBranding(w, r)
	case r.URL.Path == "/api/files" && r.Method == http.MethodGet:
		if !s.hasAccess(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		files := snapshot.store.list()
		if !s.apiKeyMatches(r) {
			if user, ok := snapshot.auth.userFromRequest(r); ok && user.Role != "admin" {
				files = snapshot.store.listForUser(user.ID)
			}
		}
		for index := range files {
			files[index] = withFileShareURLs(snapshot.baseURL, files[index])
		}
		writeJSON(w, http.StatusOK, map[string]any{"files": files})
	case strings.HasSuffix(r.URL.Path, "/views") && strings.HasPrefix(r.URL.Path, "/api/files/") && r.Method == http.MethodGet:
		if snapshot.store == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/views")
		id = strings.TrimSuffix(id, "/")
		if id == "" || id != filepath.Base(id) || strings.Contains(id, "..") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file id"})
			return
		}
		file, ok := snapshot.store.find(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": file.ID, "views": file.Views})
	case r.URL.Path == "/api/upload" && r.Method == http.MethodPost:
		if ok, reason := s.uploadReady(); !ok {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": reason})
			return
		}
		if !s.authenticate(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "api key or login required"})
			return
		}
		s.handleUpload(w, r)
	case strings.HasSuffix(r.URL.Path, "/password") && strings.HasPrefix(r.URL.Path, "/api/files/") && r.Method == http.MethodPut:
		if snapshot.auth == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		user, ok := snapshot.auth.userFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		s.handleSetFilePassword(w, r, user)
	case strings.HasPrefix(r.URL.Path, "/api/files/") && r.Method == http.MethodDelete:
		if !s.isAdmin(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/files/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file id"})
			return
		}
		if err := snapshot.store.remove(id); errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		} else if err != nil {
				log.Printf("Datei %s konnte nicht gelöscht werden: %v", id, err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "datei konnte nicht gelöscht werden"})
		} else {
			writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
		}
	default:
		if !s.isConfigured() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

type filePasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleSetFilePassword(w http.ResponseWriter, r *http.Request, user User) {
	snapshot := s.snapshot()
	if snapshot.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/files/"), "/password")
	if id == "" || id != filepath.Base(id) || strings.Contains(id, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file id"})
		return
	}
	file, ok := snapshot.store.find(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}
	if user.Role != "admin" && file.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "file access denied"})
		return
	}
	var request filePasswordRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Anfrage"})
		return
	}
	if request.Password != "" {
		passwordLength := len([]rune(request.Password))
		if passwordLength < 8 || passwordLength > 256 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Das Passwort muss zwischen 8 und 256 Zeichen lang sein."})
			return
		}
	}
	passwordHash := ""
	if request.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Dateipasswort %s konnte nicht gehasht werden: %v", id, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Passwort konnte nicht gespeichert werden"})
			return
		}
		passwordHash = string(hash)
	}
	if err := snapshot.store.setPassword(id, passwordHash); err != nil {
		log.Printf("Dateipasswort %s konnte nicht gespeichert werden: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Passwort konnte nicht gespeichert werden"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"passwordProtected": passwordHash != ""})
}

type setupRequest struct {
	DatabaseType  string `json:"databaseType"`
	MySQLDSN      string `json:"mysqlDsn"`
	MySQLHost     string `json:"mysqlHost"`
	MySQLPort     string `json:"mysqlPort"`
	MySQLDatabase string `json:"mysqlDatabase"`
	MySQLUser     string `json:"mysqlUser"`
	MySQLPassword string `json:"mysqlPassword"`
	MySQLTLS      string `json:"mysqlTls"`
	BaseURL       string `json:"baseUrl"`
	AdminEmail    string `json:"adminEmail"`
	AdminPassword string `json:"adminPassword"`
	CORSOrigin    string `json:"corsOrigin"`
	MaxUploadMB   string `json:"maxUploadMb"`
	MaxStorageMB  string `json:"maxStorageMb"`
	SMTPHost      string `json:"smtpHost"`
	SMTPPort      string `json:"smtpPort"`
	SMTPUser      string `json:"smtpUser"`
	SMTPPassword  string `json:"smtpPassword"`
	SMTPFrom      string `json:"smtpFrom"`
	SMTPTLS       string `json:"smtpTls"`
}

func (s *Server) handleSetupStatus(w http.ResponseWriter) {
	// Temporary UI preview: FORCE_SETUP_PREVIEW=true shows /setup without resetting the instance.
	if value := strings.TrimSpace(os.Getenv("FORCE_SETUP_PREVIEW")); value == "1" || strings.EqualFold(value, "true") {
		writeJSON(w, http.StatusOK, map[string]bool{"configured": false})
		return
	}
	if s.isConfigured() {
		writeJSON(w, http.StatusOK, map[string]bool{"configured": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"configured": false})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if s.isConfigured() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "setup bereits abgeschlossen"})
		return
	}
	limiterKey := clientIP(r)
	if s.setupLimiter != nil && !s.setupLimiter.allow(limiterKey) {
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "zu viele Setup-Versuche"})
		return
	}

	var input setupRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Setup-Daten"})
		return
	}
	databaseType := strings.ToLower(strings.TrimSpace(input.DatabaseType))
	if databaseType == "" {
		databaseType = "mysql"
	}
	if databaseType != "sqlite" && databaseType != "mysql" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unbekannter Datenbanktyp"})
		return
	}
	config := AppConfig{
		DatabaseType:  databaseType,
		MySQLDSN:      strings.TrimSpace(input.MySQLDSN),
		MySQLHost:     strings.TrimSpace(input.MySQLHost),
		MySQLPort:     strings.TrimSpace(input.MySQLPort),
		MySQLDatabase: strings.TrimSpace(input.MySQLDatabase),
		MySQLUser:     strings.TrimSpace(input.MySQLUser),
		MySQLPassword: input.MySQLPassword,
		MySQLTLS:      strings.TrimSpace(input.MySQLTLS),
		BaseURL:       strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
		Port:          envOr("PORT", "8080"),
		AdminEmail:    normalizeEmail(input.AdminEmail),
		AdminPassword: input.AdminPassword,
		CORSOrigin:    strings.TrimRight(strings.TrimSpace(input.CORSOrigin), "/"),
		MaxUploadMB:   strings.TrimSpace(input.MaxUploadMB),
		MaxStorageMB:  strings.TrimSpace(input.MaxStorageMB),
		SMTPHost:      strings.TrimSpace(input.SMTPHost),
		SMTPPort:      strings.TrimSpace(input.SMTPPort),
		SMTPUser:      strings.TrimSpace(input.SMTPUser),
		SMTPPassword:  input.SMTPPassword,
		SMTPFrom:      strings.TrimSpace(input.SMTPFrom),
		SMTPTLS:       strings.TrimSpace(input.SMTPTLS),
	}.withDefaults()

	if !config.complete() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Datenbankverbindung und Admin-Passwort müssen gesetzt sein"})
		return
	}
	if originFromURL(config.BaseURL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "BASE_URL ist ungültig"})
		return
	}
	if len([]rune(config.AdminPassword)) < 12 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Das Admin-Passwort muss mindestens 12 Zeichen haben"})
		return
	}
	uploadAPIKey, err := randomToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Upload-Key konnte nicht erzeugt werden"})
		return
	}
	adminAPIKey, err := randomToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Admin-Key konnte nicht erzeugt werden"})
		return
	}
	config.UploadAPIKey = ""
	config.AdminAPIKey = ""
	config.UploadAPIKeyHash = hashAPIKey(uploadAPIKey)
	config.AdminAPIKeyHash = hashAPIKey(adminAPIKey)
	if _, err := strconv.ParseInt(config.MaxUploadMB, 10, 64); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_UPLOAD_MB muss eine Zahl sein"})
		return
	}
	if _, err := strconv.ParseInt(config.MaxStorageMB, 10, 64); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_STORAGE_MB muss eine Zahl sein"})
		return
	}

	db, err := newDatabase(config.databaseType(), config.databaseDSN(s.dataDir))
	if err != nil {
		log.Printf("Web-Setup: Datenbankverbindung oder Schema fehlgeschlagen: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Datenbankverbindung oder Schema konnte nicht eingerichtet werden"})
		return
	}
	store, err := newStore(db, s.dataDir, parseStorageBytes(config.MaxStorageMB), config.databaseType())
	if err != nil {
		db.Close()
		log.Printf("Web-Setup: Dateispeicher konnte nicht eingerichtet werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dateispeicher konnte nicht eingerichtet werden"})
		return
	}
	auth, err := newAuthStore(db, s.dataDir, config.databaseType())
	if err != nil {
		db.Close()
		log.Printf("Web-Setup: Authentifizierung konnte nicht eingerichtet werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Authentifizierung konnte nicht eingerichtet werden"})
		return
	}
	if err := auth.ensureAdmin(config.AdminEmail, config.AdminPassword); err != nil {
		db.Close()
		log.Printf("Web-Setup: Admin konnte nicht angelegt werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Admin konnte nicht angelegt werden"})
		return
	}
	if err := saveAppConfig(s.configPath, config); err != nil {
		db.Close()
		log.Printf("Web-Setup: Konfiguration konnte nicht gespeichert werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Konfiguration konnte nicht gespeichert werden"})
		return
	}

	s.activate(config, store, auth)
	if s.setupLimiter != nil {
		s.setupLimiter.success(limiterKey)
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"configured":    true,
		"uploadApiKey":  uploadAPIKey,
		"adminApiKey":   adminAPIKey,
		"baseUrl":       config.BaseURL,
		"user":          publicUser(User{Email: config.AdminEmail, Name: "Administrator", Role: "admin"}),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Anmeldung"})
		return
	}
	limiterKey := clientIP(r)
	if s.loginLimiter != nil && !s.loginLimiter.allow(limiterKey) {
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "zu viele fehlgeschlagene Anmeldeversuche"})
		return
	}
	user, ok := snapshot.auth.checkPassword(input.Email, input.Password)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "E-Mail oder Passwort ist falsch"})
		return
	}
	if s.loginLimiter != nil {
		s.loginLimiter.success(limiterKey)
	}
	token, expires, err := snapshot.auth.createSession(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session konnte nicht erstellt werden"})
		return
	}
	s.setSessionCookie(w, token, expires, input.Remember)
	if snapshot.auth != nil {
		newIP, err := snapshot.auth.recordLoginIP(user.ID, clientIP(r))
		if err != nil {
			log.Printf("Login-IP für %s konnte nicht gespeichert werden: %v", user.Email, err)
		} else if newIP && snapshot.mailer != nil {
			body := fmt.Sprintf("Hallo %s,\n\nfür dein ShareX-Uploader-Konto wurde eine Anmeldung von einer bisher unbekannten IP-Adresse erkannt.\n\nE-Mail: %s\nIP-Adresse: %s\nZeitpunkt: %s\n\nWenn du das nicht warst, ändere bitte sofort dein Passwort und prüfe die Sicherheit deines Servers.\n\n%s", user.Name, user.Email, clientIP(r), time.Now().UTC().Format(time.RFC3339), strings.TrimRight(snapshot.baseURL, "/"))
			go func(mailer *Mailer, recipient, message string) {
				if err := mailer.send(recipient, "Sicherheitswarnung: neue Anmeldung", message); err != nil {
					log.Printf("Sicherheitsmail an %s konnte nicht versendet werden: %v", recipient, err)
				}
			}(snapshot.mailer, user.Email, body)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user)})
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	limiterKey := clientIP(r)
	if s.resetLimiter != nil && !s.resetLimiter.allow(limiterKey) {
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusAccepted, map[string]string{"message": "Wenn ein Konto mit dieser E-Mail existiert, wurde eine Nachricht versendet."})
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusAccepted, map[string]string{"message": "Wenn ein Konto mit dieser E-Mail existiert, wurde eine Nachricht versendet."})
		return
	}
	snapshot := s.snapshot()
	if snapshot.mailer != nil && snapshot.auth != nil {
		if user, ok := snapshot.auth.findUserByEmail(input.Email); ok {
			token, _, err := snapshot.auth.createPasswordReset(user.ID)
			if err != nil {
				log.Printf("Passwort-Reset für %s konnte nicht erzeugt werden: %v", user.Email, err)
			} else {
				resetURL := strings.TrimRight(snapshot.baseURL, "/") + "/reset-password?token=" + url.QueryEscape(token)
				body := fmt.Sprintf("Hallo %s,\n\nfür dein ShareX-Uploader-Konto wurde ein Passwort-Reset angefordert.\n\nÜber diesen Link kannst du ein neues Passwort setzen:\n%s\n\nDer Link ist 60 Minuten gültig und kann nur einmal verwendet werden.\n\nWenn du diese Anfrage nicht gestellt hast, kannst du diese Nachricht ignorieren.", user.Name, resetURL)
				go func(mailer *Mailer, recipient, message string) {
					if err := mailer.send(recipient, "Passwort zurücksetzen", message); err != nil {
						log.Printf("Passwort-Reset-Mail an %s konnte nicht versendet werden: %v", recipient, err)
					}
				}(snapshot.mailer, user.Email, body)
			}
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "Wenn ein Konto mit dieser E-Mail existiert, wurde eine Nachricht versendet."})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	limiterKey := clientIP(r)
	if s.resetLimiter != nil && !s.resetLimiter.allow(limiterKey) {
		w.Header().Set("Retry-After", "900")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "zu viele Reset-Versuche"})
		return
	}
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Reset-Daten"})
		return
	}
	if strings.TrimSpace(input.Token) == "" || len([]rune(input.Password)) < 12 || len([]rune(input.Password)) > 256 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Token oder Passwort ist ungültig"})
		return
	}
	snapshot := s.snapshot()
	if snapshot.auth == nil || !snapshot.auth.resetPassword(input.Token, input.Password) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Token ist ungültig oder abgelaufen"})
		return
	}
	if s.resetLimiter != nil {
		s.resetLimiter.success(limiterKey)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"reset": true})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(strings.TrimSpace(host))
	if remoteIP == nil {
		return "unknown"
	}
	// App is only published on 127.0.0.1 behind nginx/Docker. RemoteAddr is then the
	// docker bridge IP (private), not loopback — still trust proxy headers there.
	if remoteIP.IsLoopback() || remoteIP.IsPrivate() || remoteIP.IsLinkLocalUnicast() {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Real-IP"), ",")[0]); net.ParseIP(forwarded) != nil {
			return net.ParseIP(forwarded).String()
		}
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); net.ParseIP(forwarded) != nil {
			return net.ParseIP(forwarded).String()
		}
	}
	return remoteIP.String()
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	user, ok := snapshot.auth.userFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user)})
}

func (s *Server) handleAccountSettings(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	user, ok := snapshot.auth.userFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"glowColor": normalizeGlowColor(user.GlowColor)})
}

func (s *Server) handleUpdateAccountSettings(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	user, ok := snapshot.auth.userFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
		return
	}
	var input struct {
		GlowColor string `json:"glowColor"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Einstellungsdaten"})
		return
	}
	glowColor := normalizeGlowColor(input.GlowColor)
	if strings.TrimSpace(input.GlowColor) != "" && glowColor == defaultGlowColor && !strings.EqualFold(strings.TrimSpace(input.GlowColor), defaultGlowColor) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Glow-Farbe muss ein gültiger Hex-Farbwert sein"})
		return
	}
	if err := snapshot.auth.updateGlowColor(user.ID, glowColor); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Glow-Farbe konnte nicht gespeichert werden"})
		return
	}
	user.GlowColor = glowColor
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user), "glowColor": glowColor})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Profildaten"})
		return
	}
	updated, err := s.snapshot().auth.updateProfile(user.ID, input.Name, input.Email)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "bereits verwendet") {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(updated)})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Passwortdaten"})
		return
	}
	if input.CurrentPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "aktuelles Passwort ist erforderlich"})
		return
	}
	if input.NewPassword != input.ConfirmPassword {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "die neuen Passwörter stimmen nicht überein"})
		return
	}
	if err := s.snapshot().auth.updatePassword(user.ID, input.CurrentPassword, input.NewPassword); err != nil {
		status := http.StatusBadRequest
		if !strings.Contains(err.Error(), "Passwort") {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	s.snapshot().auth.deleteOtherSessions(r, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"updated": true, "message": "Passwort geändert"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	snapshot.auth.deleteSession(r)
	s.setSessionCookie(w, "", time.Unix(0, 0), false)
	writeJSON(w, http.StatusOK, map[string]bool{"loggedOut": true})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time, persistent bool) {
	snapshot := s.snapshot()
	maxAge := 0
	if persistent {
		maxAge = int(time.Until(expires).Seconds())
	}
	if token == "" {
		maxAge = -1
	}
	cookie := &http.Cookie{Name: "drop_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: snapshot.secureCookies, MaxAge: maxAge}
	if persistent || token == "" {
		cookie.Expires = expires
	}
	http.SetCookie(w, cookie)
}

func publicUser(user User) map[string]any {
	glowColor := normalizeGlowColor(user.GlowColor)
	return map[string]any{"id": user.ID, "email": user.Email, "name": user.Name, "role": user.Role, "maxUploadBytes": user.MaxUploadBytes, "maxStorageBytes": user.MaxStorageBytes, "unlimitedStorage": user.UnlimitedStorage, "glowColor": glowColor}
}

func (s *Server) databaseAlive() bool {
	// Test helper: set FORCE_DB_OFFLINE=true to simulate DB outage without deleting data.
	if value := strings.TrimSpace(os.Getenv("FORCE_DB_OFFLINE")); value == "1" || strings.EqualFold(value, "true") {
		return false
	}
	snapshot := s.snapshot()
	if snapshot.store == nil || snapshot.store.db == nil {
		return false
	}
	return snapshot.store.db.Ping() == nil
}

func (s *Server) uploadReady() (bool, string) {
	if !s.isConfigured() {
		return false, "setup erforderlich"
	}
	snapshot := s.snapshot()
	if snapshot.apiKey == "" {
		return false, "api-key-schutz nicht aktiv"
	}
	if !s.databaseAlive() {
		return false, "datenbank nicht erreichbar"
	}
	return true, ""
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if !s.isConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":         "setup_required",
			"api":            true,
			"apiKeyEnabled":  false,
			"database":       false,
			"databaseType":   "",
			"uploadReady":    false,
		})
		return
	}
	snapshot := s.snapshot()
	apiKeyEnabled := snapshot.apiKey != ""
	databaseOK := s.databaseAlive()
	uploadReady := apiKeyEnabled && databaseOK
	status := "ok"
	code := http.StatusOK
	if !uploadReady {
		status = "degraded"
	}
	writeJSON(w, code, map[string]any{
		"status":        status,
		"api":           true,
		"apiKeyEnabled": apiKeyEnabled,
		"database":      databaseOK,
		"databaseType":  snapshot.databaseType,
		"uploadReady":   uploadReady,
	})
}

func (s *Server) handleAdminOverview(w http.ResponseWriter) {
	snapshot := s.snapshot()
	files := snapshot.store.list()
	var totalBytes int64
	types := make(map[string]int)
	for _, file := range files {
		totalBytes += file.Size
		kind := "FILE"
		if strings.HasPrefix(file.ContentType, "image/") {
			kind = "IMAGE"
		} else if strings.HasPrefix(file.ContentType, "video/") {
			kind = "VIDEO"
		} else if file.ContentType == "application/pdf" {
			kind = "PDF"
		}
		types[kind]++
	}
	if len(files) > 10 {
		files = files[:10]
	}
	for index := range files {
		files[index] = withFileShareURLs(snapshot.baseURL, files[index])
	}
	apiKeyEnabled := snapshot.apiKey != ""
	databaseOK := s.databaseAlive()
	writeJSON(w, http.StatusOK, map[string]any{
		"fileCount":        len(snapshot.store.list()),
		"totalBytes":       totalBytes,
		"typeCount":        len(types),
		"types":            types,
		"recent":           files,
		"maxUploadBytes":   snapshot.maxUpload,
		"maxStorageBytes":  snapshot.store.maxStorage,
		"apiKeyEnabled":    apiKeyEnabled,
		"database":         snapshot.databaseType,
		"databaseOnline":   databaseOK,
		"uploadReady":      apiKeyEnabled && databaseOK,
	})
}

func brandLogoPath(dataDir, filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == ".." || filename == "" || filename == string(filepath.Separator) {
		return ""
	}
	return filepath.Join(dataDir, filename)
}

func brandLogoURL(path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return "/api/branding/logo?v=" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
}

func absoluteURL(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(strings.TrimSpace(base), "/") + path
}

func normalizeSiteName(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 64 {
		return string(runes[:64])
	}
	return value
}

func normalizeEmbedColor(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len(value) != 7 || value[0] != '#' {
		return "", false
	}
	for index := 1; index < 7; index++ {
		char := value[index]
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return "", false
		}
	}
	return strings.ToLower(value), true
}

func embedSiteName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "ShareX Uploader") {
		return "404ProjectNET"
	}
	return value
}

func embedColorOrDefault(value string) string {
	if color, ok := normalizeEmbedColor(value); ok && color != "" {
		return color
	}
	return "#718dff"
}

type embedOptions struct {
	siteName  string
	color     string
	title     string
	extra     string
	card      string
	image     string
	showSite  bool
	showDate  bool
	showSize  bool
	showViews bool
	showType  bool
	showDims  bool
	showIcons bool
}

func embedFlag(value string, defaultOn bool) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "0", "false", "off", "no":
		return false
	case "1", "true", "on", "yes":
		return true
	default:
		return defaultOn
	}
}

func embedFlagStore(on, defaultOn bool) string {
	if on == defaultOn {
		return ""
	}
	if on {
		return "1"
	}
	return "0"
}

func clipRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if max > 0 && len(runes) > max {
		return string(runes[:max])
	}
	return value
}

func embedCardMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "compact") {
		return "compact"
	}
	return "large"
}

func embedImageMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "file", "logo", "none":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func embedMetaIcon(kind string) string {
	switch kind {
	case "type-IMAGE":
		return "🖼️"
	case "type-VIDEO":
		return "🎬"
	case "type-AUDIO":
		return "🎵"
	case "type-PDF":
		return "📄"
	case "type":
		return "📁"
	case "date":
		return "📅"
	case "size":
		return "📦"
	case "dims":
		return "📐"
	case "views":
		return "👁️"
	default:
		return ""
	}
}

func withEmbedIcon(enabled bool, kind, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if !enabled {
		return text
	}
	icon := embedMetaIcon(kind)
	if icon == "" {
		return text
	}
	return icon + " " + text
}

func embedFileKind(file FileRecord) string {
	typeName := strings.ToLower(file.ContentType)
	name := strings.ToLower(file.Name)
	switch {
	case strings.HasPrefix(typeName, "image/"):
		return "IMAGE"
	case strings.HasPrefix(typeName, "video/"):
		return "VIDEO"
	case strings.HasPrefix(typeName, "audio/"):
		return "AUDIO"
	case typeName == "application/pdf" || strings.HasSuffix(name, ".pdf"):
		return "PDF"
	default:
		return "FILE"
	}
}

func applyEmbedTemplate(template, fallback string, values map[string]string) string {
	text := strings.TrimSpace(template)
	if text == "" {
		text = fallback
	}
	replacer := strings.NewReplacer(
		"{name}", values["name"],
		"{site}", values["site"],
		"{type}", values["type"],
		"{size}", values["size"],
	)
	out := clipRunes(replacer.Replace(text), 120)
	if out == "" {
		return fallback
	}
	return out
}

func embedOptionsFromConfig(config AppConfig) embedOptions {
	color := ""
	if parsed, ok := normalizeEmbedColor(config.EmbedColor); ok {
		color = parsed
	}
	return embedOptions{
		siteName:  embedSiteName(config.EmbedSiteName),
		color:     color,
		title:     clipRunes(config.EmbedTitle, 120),
		extra:     clipRunes(config.EmbedExtra, 200),
		card:      embedCardMode(config.EmbedCard),
		image:     embedImageMode(config.EmbedImage),
		showSite:  embedFlag(config.EmbedShowSite, true),
		showDate:  embedFlag(config.EmbedShowDate, true),
		showSize:  embedFlag(config.EmbedShowSize, true),
		showViews: embedFlag(config.EmbedShowViews, true),
		showType:  embedFlag(config.EmbedShowType, false),
		showDims:  embedFlag(config.EmbedShowDims, true),
		showIcons: embedFlag(config.EmbedShowIcons, true),
	}
}

func applyEmbedOptions(config *AppConfig, options embedOptions) {
	config.EmbedSiteName = normalizeSiteName(options.siteName)
	config.EmbedColor = embedColorOrDefault(options.color)
	config.EmbedTitle = clipRunes(options.title, 120)
	config.EmbedExtra = clipRunes(options.extra, 200)
	config.EmbedCard = options.card
	config.EmbedImage = options.image
	config.EmbedShowSite = embedFlagStore(options.showSite, true)
	config.EmbedShowDate = embedFlagStore(options.showDate, true)
	config.EmbedShowSize = embedFlagStore(options.showSize, true)
	config.EmbedShowViews = embedFlagStore(options.showViews, true)
	config.EmbedShowType = embedFlagStore(options.showType, false)
	config.EmbedShowDims = embedFlagStore(options.showDims, true)
	config.EmbedShowIcons = embedFlagStore(options.showIcons, true)
}

func brandingJSON(snap serverSnapshot) map[string]any {
	embed := snap.embed
	return map[string]any{
		"logoUrl":         brandLogoURL(snap.brandLogoPath),
		"siteName":        embed.siteName,
		"embedColor":      embedColorOrDefault(embed.color),
		"embedTitle":      embed.title,
		"embedExtra":      embed.extra,
		"embedCard":       embed.card,
		"embedImage":      embed.image,
		"showSiteName":    embed.showSite,
		"showDate":        embed.showDate,
		"showSize":        embed.showSize,
		"showViews":       embed.showViews,
		"showType":        embed.showType,
		"showDimensions":  embed.showDims,
		"showIcons":       embed.showIcons,
	}
}

func (s *Server) handleBrandLogo(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	if snapshot.brandLogoPath == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(snapshot.brandLogoPath); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, snapshot.brandLogoPath)
}

func (s *Server) handleAdminBranding(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, brandingJSON(s.snapshot()))
		return
	}

	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Branding wird über Umgebungsvariablen verwaltet"})
		return
	}
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Konfiguration konnte nicht gelesen werden"})
		return
	}

	if r.Method == http.MethodPut && strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		var input struct {
			SiteName        string `json:"siteName"`
			EmbedColor      string `json:"embedColor"`
			EmbedTitle      string `json:"embedTitle"`
			EmbedExtra      string `json:"embedExtra"`
			EmbedCard       string `json:"embedCard"`
			EmbedImage      string `json:"embedImage"`
			ShowSiteName    *bool  `json:"showSiteName"`
			ShowDate        *bool  `json:"showDate"`
			ShowSize        *bool  `json:"showSize"`
			ShowViews       *bool  `json:"showViews"`
			ShowType        *bool  `json:"showType"`
			ShowDimensions  *bool  `json:"showDimensions"`
			ShowIcons       *bool  `json:"showIcons"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Discord-Einstellungen"})
			return
		}
		color, ok := normalizeEmbedColor(input.EmbedColor)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Farbe muss im Format #RRGGBB sein"})
			return
		}
		options := embedOptionsFromConfig(config)
		options.siteName = input.SiteName
		options.color = color
		options.title = input.EmbedTitle
		options.extra = input.EmbedExtra
		options.card = embedCardMode(input.EmbedCard)
		options.image = embedImageMode(input.EmbedImage)
		if input.ShowSiteName != nil {
			options.showSite = *input.ShowSiteName
		}
		if input.ShowDate != nil {
			options.showDate = *input.ShowDate
		}
		if input.ShowSize != nil {
			options.showSize = *input.ShowSize
		}
		if input.ShowViews != nil {
			options.showViews = *input.ShowViews
		}
		if input.ShowType != nil {
			options.showType = *input.ShowType
		}
		if input.ShowDimensions != nil {
			options.showDims = *input.ShowDimensions
		}
		if input.ShowIcons != nil {
			options.showIcons = *input.ShowIcons
		}
		applyEmbedOptions(&config, options)
		if err := saveAppConfig(s.configPath, config); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Discord-Einstellungen konnten nicht gespeichert werden"})
			return
		}
		s.stateMu.Lock()
		s.embed = embedOptionsFromConfig(config)
		s.stateMu.Unlock()
		writeJSON(w, http.StatusOK, brandingJSON(s.snapshot()))
		return
	}

	if r.Method == http.MethodDelete {
		oldPath := brandLogoPath(s.dataDir, config.BrandLogo)
		config.BrandLogo = ""
		if err := saveAppConfig(s.configPath, config); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo-Konfiguration konnte nicht gespeichert werden"})
			return
		}
		if oldPath != "" {
			_ = os.Remove(oldPath)
		}
		s.stateMu.Lock()
		s.brandLogoPath = ""
		s.stateMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"logoUrl": ""})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024+1024)
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Logo ist zu groß oder ungültig"})
		return
	}
	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bitte ein Logo auswählen"})
		return
	}
	defer file.Close()
	if header.Size < 1 || header.Size > 5*1024*1024 {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Das Logo darf maximal 5 MB groß sein"})
		return
	}

	headerBytes := make([]byte, 512)
	read, readErr := file.Read(headerBytes)
	if readErr != nil && readErr != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Logo konnte nicht gelesen werden"})
		return
	}
	contentType := http.DetectContentType(headerBytes[:read])
	extensions := map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif"}
	ext, ok := extensions[contentType]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Erlaubt sind PNG-, JPG- oder GIF-Dateien"})
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Logo konnte nicht verarbeitet werden"})
		return
	}
	if _, _, err := image.DecodeConfig(file); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Die Bilddatei ist ungültig"})
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Logo konnte nicht verarbeitet werden"})
		return
	}
	logoID, err := randomUploadID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo-ID konnte nicht erzeugt werden"})
		return
	}
	filename := "branding-" + logoID + "." + ext
	temp, err := os.CreateTemp(s.dataDir, ".branding-logo-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo konnte nicht gespeichert werden"})
		return
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(temp, io.LimitReader(file, 5*1024*1024+1)); err != nil {
		_ = temp.Close()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo konnte nicht gespeichert werden"})
		return
	}
	if err := temp.Close(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo konnte nicht gespeichert werden"})
		return
	}
	targetPath := filepath.Join(s.dataDir, filename)
	if err := os.Rename(tempPath, targetPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo konnte nicht aktiviert werden"})
		return
	}

	oldPath := brandLogoPath(s.dataDir, config.BrandLogo)
	config.BrandLogo = filename
	if err := saveAppConfig(s.configPath, config); err != nil {
		_ = os.Remove(targetPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Logo-Konfiguration konnte nicht gespeichert werden"})
		return
	}
	if oldPath != "" && oldPath != targetPath {
		_ = os.Remove(oldPath)
	}
	s.stateMu.Lock()
	s.brandLogoPath = targetPath
	s.stateMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"logoUrl": brandLogoURL(targetPath)})
}

func (s *Server) handleRotateAPIKey(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Schlüssel-Anfrage"})
		return
	}
	keyType := strings.TrimSpace(strings.ToLower(input.Type))
	if keyType != "upload" && keyType != "admin" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unbekannter Schlüsseltyp"})
		return
	}

	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if !s.isConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup erforderlich"})
		return
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "API-Schlüssel werden über Umgebungsvariablen verwaltet"})
		return
	}
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Konfiguration für API-Key-Rotation konnte nicht gelesen werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Konfiguration konnte nicht gelesen werden"})
		return
	}
	key, err := randomToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "neuer API-Key konnte nicht erzeugt werden"})
		return
	}
	if keyType == "upload" {
		config.UploadAPIKey = ""
		config.UploadAPIKeyHash = hashAPIKey(key)
	} else {
		config.AdminAPIKey = ""
		config.AdminAPIKeyHash = hashAPIKey(key)
	}
	if err := saveAppConfig(s.configPath, config); err != nil {
		log.Printf("API-Key konnte nicht gespeichert werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "API-Key konnte nicht gespeichert werden"})
		return
	}

	s.stateMu.Lock()
	if keyType == "upload" {
		s.apiKey = config.UploadAPIKeyHash
	} else {
		s.adminAPIKey = config.AdminAPIKeyHash
	}
	s.stateMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"type": keyType, "key": key, "baseUrl": s.snapshot().baseURL})
}

type adminUserRequest struct {
	Email            string `json:"email"`
	Name             string `json:"name"`
	Password         string `json:"password"`
	MaxUploadMB     string `json:"maxUploadMb"`
	MaxStorageMB    string `json:"maxStorageMb"`
	UnlimitedStorage bool   `json:"unlimitedStorage"`
}

func (s *Server) handleAdminUsers(w http.ResponseWriter) {
	snapshot := s.snapshot()
	users, err := snapshot.auth.listUsers()
	if err != nil {
		log.Printf("Benutzerliste konnte nicht gelesen werden: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "benutzerliste konnte nicht gelesen werden"})
		return
	}
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		items = append(items, publicUser(user))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": items})
}

func (s *Server) handleCreateAdminUser(w http.ResponseWriter, r *http.Request) {
	var input adminUserRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Benutzerdaten"})
		return
	}
	maxUpload, err := parseLimitMB(input.MaxUploadMB, defaultMaxUploadBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_UPLOAD_MB muss eine gültige Zahl sein"})
		return
	}
	maxStorage, err := parseLimitMB(input.MaxStorageMB, defaultMaxStorageBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_STORAGE_MB muss eine gültige Zahl sein"})
		return
	}
	snapshot := s.snapshot()
	user, err := snapshot.auth.createUser(input.Email, input.Name, input.Password, maxUpload, maxStorage, input.UnlimitedStorage)
	if err != nil {
		log.Printf("Benutzer konnte nicht angelegt werden: %v", err)
		if isDuplicateEntryError(err) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Diese E-Mail-Adresse ist bereits vergeben"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": publicUser(user)})
}

func (s *Server) handleUpdateAdminUserLimits(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if userID == "" || strings.ContainsAny(userID, "/\\") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Benutzer-ID"})
		return
	}
	var input adminUserRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ungültige Limit-Daten"})
		return
	}
	maxUpload, err := parseLimitMB(input.MaxUploadMB, defaultMaxUploadBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_UPLOAD_MB muss eine gültige Zahl sein"})
		return
	}
	maxStorage, err := parseLimitMB(input.MaxStorageMB, defaultMaxStorageBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "MAX_STORAGE_MB muss eine gültige Zahl sein"})
		return
	}
	snapshot := s.snapshot()
	if err := snapshot.auth.updateUserLimits(userID, maxUpload, maxStorage, input.UnlimitedStorage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Benutzer nicht gefunden"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Benutzerlimits konnten nicht gespeichert werden"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

func isDuplicateEntryError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func applyUserUploadLimits(user User, maxUpload, storageLimit int64) (int64, int64) {
	if user.MaxUploadBytes > 0 {
		maxUpload = user.MaxUploadBytes
	}
	if user.UnlimitedStorage {
		storageLimit = 0
	} else if user.MaxStorageBytes > 0 {
		storageLimit = user.MaxStorageBytes
	}
	return maxUpload, storageLimit
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	ownerID := ""
	maxUpload := snapshot.maxUpload
	storageLimit := snapshot.store.maxStorage

	user, hasSession := User{}, false
	if snapshot.auth != nil {
		user, hasSession = snapshot.auth.userFromRequest(r)
	}

	switch {
	case hasSession:
		// Browser session always wins for per-user limits.
		ownerID = user.ID
		maxUpload, storageLimit = applyUserUploadLimits(user, maxUpload, storageLimit)
	case s.apiKeyMatches(r):
		// ShareX / API-Key: use primary admin limits so Admin-Benutzerlimits greifen.
		if snapshot.auth != nil {
			if admin, ok := snapshot.auth.primaryAdmin(); ok {
				maxUpload, storageLimit = applyUserUploadLimits(admin, maxUpload, storageLimit)
				ownerID = admin.ID
			}
		}
	default:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
		return
	}

	// Multipart overhead buffer so the declared file limit is reachable.
	bodyLimit := maxUpload + (1 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	if err := r.ParseMultipartForm(bodyLimit); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "dein Upload-Limit wurde überschritten"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "multipart-Feld 'file' fehlt"})
		return
	}
	defer file.Close()
	if header.Size > 0 && header.Size > maxUpload {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": fmt.Sprintf("dein Upload-Limit wurde überschritten (max. %s)", formatFileSize(maxUpload))})
		return
	}

	id, err := randomUploadID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "konnte id nicht erzeugen"})
		return
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if len(ext) > 10 || strings.ContainsAny(ext, "/\\") {
		ext = ""
	}
	now := time.Now().UTC()
	base := strings.TrimRight(snapshot.baseURL, "/")
	fileRecord := FileRecord{
		ID:          id,
		Name:        filepath.Base(header.Filename),
		Size:        header.Size,
		ContentType: contentType,
		URL:         fmt.Sprintf("%s/uploads/%s%s", base, id, ext),
		UploadedAt:  now,
	}
	fileRecord.PreviewURL = filePreviewURL(base, fileRecord)
	fileRecord.RawURL = fileRawURL(base, fileRecord)
	if err := snapshot.store.add(fileRecord, file, ext, ownerID, storageLimit); err != nil {
			log.Printf("Upload %s konnte nicht gespeichert werden: %v", fileRecord.ID, err)
			if errors.Is(err, errStorageLimit) {
				writeJSON(w, http.StatusInsufficientStorage, map[string]string{"error": "speicherlimit erreicht"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload konnte nicht gespeichert werden"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          fileRecord.ID,
		"name":        fileRecord.Name,
		"size":        fileRecord.Size,
		"contentType": fileRecord.ContentType,
		"url":         fileRecord.URL,
		"previewUrl":  fileRecord.PreviewURL,
		"rawUrl":      fileRecord.RawURL,
		"deleteUrl":   base + "/api/files/" + fileRecord.ID,
		"uploadedAt":  fileRecord.UploadedAt,
		"views":       fileRecord.Views,
	})
}

func (s *Server) routes() http.Handler {
	api := http.HandlerFunc(s.handleAPI)
	return s.withSecurityHeaders(s.withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			api.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/view/"):
			s.handlePreview(w, r)
		case strings.HasPrefix(r.URL.Path, "/raw/"):
			s.handleRawView(w, r)
		case strings.HasPrefix(r.URL.Path, "/i/"):
			s.handleDirectMedia(w, r)
		case strings.HasPrefix(r.URL.Path, "/uploads/"):
			s.handleUploadFile(w, r)
		case r.URL.Path == "/setup":
			if s.isConfigured() {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			r.URL.Path = "/"
			s.frontend.ServeHTTP(w, r)
		default:
			path := r.URL.Path
			switch path {
			case "/", "/login", "/logout", "/forgot-password", "/reset-password", "/dashboard", "/galerie", "/einstellungen", "/admin":
				r.URL.Path = "/"
				s.frontend.ServeHTTP(w, r)
			default:
				// Static assets (js/css/images) stay with the file server.
				if strings.Contains(filepath.Base(path), ".") {
					s.frontend.ServeHTTP(w, r)
					return
				}
				writeNotFoundPage(w, r)
			}
		}
	})))
}

func filePreviewURL(baseURL string, file FileRecord) string {
	return strings.TrimRight(baseURL, "/") + "/view/" + url.PathEscape(file.ID)
}

func fileRawURL(baseURL string, file FileRecord) string {
	// Discord needs /raw/{id}.png (bytes). /raw/{id} without extension stays the HTML viewer.
	ext := strings.ToLower(filepath.Ext(file.URL))
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(file.Name))
	}
	if len(ext) > 10 || strings.ContainsAny(ext, "/\\") {
		ext = ""
	}
	return strings.TrimRight(baseURL, "/") + "/raw/" + url.PathEscape(file.ID) + ext
}

func isDirectMediaExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg",
		".mp4", ".webm", ".mov", ".mkv",
		".mp3", ".wav", ".ogg", ".flac",
		".pdf":
		return true
	default:
		return false
	}
}

func isSocialMediaBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	bots := []string{
		"discordbot", "discord",
		"telegrambot", "telegram",
		"slackbot", "slack-imgproxy",
		"facebookexternalhit", "facebot",
		"twitterbot", "linkedinbot",
		"whatsapp", "viber", "skypeuripreview",
		"embedly", "quora link preview",
		"pinterest", "redditbot", "applebot",
		"googlebot", "bingbot", "yandexbot",
	}
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

func withFileShareURLs(baseURL string, file FileRecord) FileRecord {
	file.PreviewURL = filePreviewURL(baseURL, file)
	file.RawURL = fileRawURL(baseURL, file)
	return file
}

func imageSize(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width < 1 || config.Height < 1 {
		return 0, 0
	}
	return config.Width, config.Height
}

func imageDimensions(path string) string {
	width, height := imageSize(path)
	if width < 1 || height < 1 {
		return ""
	}
	return fmt.Sprintf("%d×%d", width, height)
}

func openGraphImageMeta(imageURL, alt, contentType string, width, height int) string {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return ""
	}
	parts := []string{
		fmt.Sprintf(`<meta property="og:image" content="%s">`, html.EscapeString(imageURL)),
		fmt.Sprintf(`<meta property="og:image:alt" content="%s">`, html.EscapeString(alt)),
		fmt.Sprintf(`<meta name="twitter:image" content="%s">`, html.EscapeString(imageURL)),
	}
	if contentType != "" {
		parts = append(parts, fmt.Sprintf(`<meta property="og:image:type" content="%s">`, html.EscapeString(contentType)))
	}
	if width > 0 && height > 0 {
		parts = append(parts,
			fmt.Sprintf(`<meta property="og:image:width" content="%d">`, width),
			fmt.Sprintf(`<meta property="og:image:height" content="%d">`, height),
		)
	}
	return strings.Join(parts, "")
}

func formatViewCount(views int64) string {
	if views == 1 {
		return "1 view"
	}
	return fmt.Sprintf("%d views", views)
}

func formatAufrufCount(views int64) string {
	if views == 1 {
		return "1 Aufruf"
	}
	return fmt.Sprintf("%d Aufrufe", views)
}

func previewMetaItem(icon, value string) string {
	return previewMetaItemID("", icon, value)
}

func previewMetaItemID(id, icon, value string) string {
	idAttr := ""
	if id != "" {
		idAttr = fmt.Sprintf(` id="%s"`, html.EscapeString(id))
	}
	return fmt.Sprintf(`<span class="meta-item"%s><span class="meta-icon" aria-hidden="true">%s</span><span class="meta-text">%s</span></span>`, idAttr, icon, html.EscapeString(value))
}

const previewSizeIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 3h12l3 3v15H4V3z"/><path d="M8 3v6h8V3M8 15h8v6H8z"/></svg>`
const previewDimensionsIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="8" cy="9" r="1.2"/><path d="m4.5 17 4.5-4 3.2 2.8 2.4-2.2 4.9 4.4"/></svg>`
const previewViewsIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"/><circle cx="12" cy="12" r="2.6"/></svg>`
const previewTimeIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="4" y="5" width="16" height="15" rx="2"/><path d="M8 3v4M16 3v4M4 9h16M8 13h.01M12 13h.01M16 13h.01M8 16h.01M12 16h.01"/></svg>`

func formatUploadTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	months := [...]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."}
	local := value.Local()
	return fmt.Sprintf("%d. %s, %02d:%02d", local.Day(), months[local.Month()-1], local.Hour(), local.Minute())
}

func viewVisitorHash(fileID string, r *http.Request) string {
	value := "drop-view\x00" + fileID + "\x00" + clientIP(r)
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

const fileAccessCookieLifetime = 12 * time.Hour

func fileAccessCookieName(fileID string) string {
	digest := sha256.Sum256([]byte("drop-file-access\x00" + fileID))
	return "drop_file_access_" + hex.EncodeToString(digest[:])[:24]
}

func fileAccessSecret(snapshot serverSnapshot) string {
	return snapshot.adminAPIKey + "\x00" + snapshot.apiKey
}

func fileAccessSignature(snapshot serverSnapshot, fileID, expires string) []byte {
	mac := hmac.New(sha256.New, []byte(fileAccessSecret(snapshot)))
	_, _ = io.WriteString(mac, "file-access\x00"+fileID+"\x00"+expires)
	return mac.Sum(nil)
}

func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}

func (s *Server) fileAccessCookieValid(r *http.Request, snapshot serverSnapshot, fileID string) bool {
	cookie, err := r.Cookie(fileAccessCookieName(fileID))
	if err != nil || cookie.Value == "" {
		return false
	}
	expires, signature, ok := strings.Cut(cookie.Value, ".")
	if !ok || expires == "" || signature == "" {
		return false
	}
	expiresAt, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || expiresAt <= time.Now().Unix() {
		return false
	}
	expected, err := hex.DecodeString(signature)
	if err != nil || !hmac.Equal(expected, fileAccessSignature(snapshot, fileID, expires)) {
		return false
	}
	return true
}

func (s *Server) grantFileAccess(w http.ResponseWriter, r *http.Request, snapshot serverSnapshot, fileID string) {
	expires := strconv.FormatInt(time.Now().Add(fileAccessCookieLifetime).Unix(), 10)
	value := expires + "." + hex.EncodeToString(fileAccessSignature(snapshot, fileID, expires))
	http.SetCookie(w, &http.Cookie{
		Name:     fileAccessCookieName(fileID),
		Value:    value,
		Path:     "/",
		MaxAge:   int(fileAccessCookieLifetime / time.Second),
		Expires:  time.Now().Add(fileAccessCookieLifetime),
		HttpOnly: true,
		Secure:   snapshot.secureCookies && requestIsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func writeFilePasswordPage(w http.ResponseWriter, r *http.Request, status int, fileName string, invalid bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	action := html.EscapeString(r.URL.EscapedPath())
	if action == "" {
		action = "/"
	}
	errorMessage := ""
	if invalid {
		errorMessage = `<div class="password-error">Das Passwort ist nicht korrekt.</div>`
	}
	const pageTemplate = `<!doctype html>
 <html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" type="image/png" href="/drop-logo.png"><title>Passwort erforderlich</title><style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;background:#080f1d;color:#e8eef8}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:radial-gradient(circle at 50% 15%,rgba(78,101,181,.16),transparent 38%),#080f1d}.password-card{width:min(430px,100%);padding:38px 34px;text-align:center;border:1px solid rgba(77,103,153,.55);border-radius:18px;background:rgba(13,25,43,.9);box-shadow:0 24px 80px rgba(0,0,0,.36)}.lock{display:grid;place-items:center;width:54px;height:54px;margin:0 auto 20px;border:1px solid rgba(142,162,255,.52);border-radius:16px;color:#b6c2ff;background:rgba(75,96,190,.2);font-size:25px}.eyebrow{margin:0 0 12px;color:#8ea4ff;font-size:11px;font-weight:800;letter-spacing:.2em}.password-card h1{margin:0;color:#f1f4fb;font-size:clamp(24px,5vw,32px);letter-spacing:-.045em}.password-card p{margin:13px auto 24px;color:#96a8c2;font-size:13px;line-height:1.7}.password-card form{display:grid;gap:12px;text-align:left}.password-card label{color:#b6c4d9;font-size:12px;font-weight:700}.password-card input{display:block;width:100%;margin-top:7px;padding:13px 14px;border:1px solid #30486f;border-radius:9px;outline:0;color:#edf2ff;background:#0b192c;font:inherit}.password-card input:focus{border-color:#8da3ff;box-shadow:0 0 0 3px rgba(117,139,255,.16)}.password-card button{margin-top:4px;padding:13px 16px;border:0;border-radius:9px;color:#08101d;background:#91a7ff;font:inherit;font-weight:800;cursor:pointer}.password-error{margin:-2px 0 2px;padding:10px 12px;border:1px solid rgba(255,121,106,.42);border-radius:8px;color:#ffb2a5;background:rgba(155,51,48,.18);font-size:12px}
</style></head><body><main class="password-card"><div class="lock" aria-hidden="true"><svg viewBox="0 0 24 24" width="25" height="25" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="10" width="14" height="10" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/><path d="M12 14v2"/></svg></div><p class="eyebrow">GESCHÜTZTE DATEI</p><h1>Passwort erforderlich</h1><p>Diese Datei ist geschützt. Gib das korrekte Passwort ein, um sie zu öffnen.</p>{{ERROR}}<form method="post" action="{{ACTION}}"><label>Passwort<input type="password" name="password" autocomplete="current-password" required autofocus></label><button type="submit">Datei öffnen</button></form></main></body></html>`
	page := strings.Replace(pageTemplate, "{{ERROR}}", errorMessage, 1)
	page = strings.Replace(page, "{{ACTION}}", action, 1)
	_, _ = io.WriteString(w, page)
	_ = fileName
}

func (s *Server) requireFileAccess(w http.ResponseWriter, r *http.Request, snapshot serverSnapshot, file FileRecord) bool {
	if file.PasswordHash == "" {
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return false
		}
		return true
	}
	if snapshot.auth != nil {
		if user, ok := snapshot.auth.userFromRequest(r); ok && (user.Role == "admin" || user.ID == file.UserID) {
			if r.Method == http.MethodPost {
				writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return false
			}
			return true
		}
	}
	if s.fileAccessCookieValid(r, snapshot, file.ID) {
		return r.Method != http.MethodPost
	}
	if r.Method == http.MethodPost {
		limiterKey := clientIP(r) + "\x00" + file.ID
		if s.filePasswordLimiter != nil && !s.filePasswordLimiter.allow(limiterKey) {
			w.Header().Set("Retry-After", "900")
			writeFilePasswordPage(w, r, http.StatusTooManyRequests, file.Name, true)
			return false
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
		if err := r.ParseForm(); err == nil {
			password := r.FormValue("password")
			if bcrypt.CompareHashAndPassword([]byte(file.PasswordHash), []byte(password)) == nil {
				if s.filePasswordLimiter != nil {
					s.filePasswordLimiter.success(limiterKey)
				}
				s.grantFileAccess(w, r, snapshot, file.ID)
				http.Redirect(w, r, r.URL.EscapedPath(), http.StatusSeeOther)
				return false
			}
		}
		writeFilePasswordPage(w, r, http.StatusUnauthorized, file.Name, true)
		return false
	}
	writeFilePasswordPage(w, r, http.StatusUnauthorized, file.Name, false)
	return false
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/view/")
	if id == "" || id != filepath.Base(id) || strings.Contains(id, "..") {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	snapshot := s.snapshot()
	if snapshot.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
		return
	}
	file, ok := snapshot.store.find(id)
	if !ok {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	ext := filepath.Ext(file.URL)
	path := filepath.Join(snapshot.store.dataDir, "uploads", file.ID+ext)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	if !s.requireFileAccess(w, r, snapshot, file) {
		return
	}
	if r.Method == http.MethodGet {
		if views, err := snapshot.store.registerView(file.ID, viewVisitorHash(file.ID, r)); err == nil {
			file.Views = views
		}
	}

	directURL := strings.TrimRight(snapshot.baseURL, "/") + "/uploads/" + url.PathEscape(file.ID+ext)
	previewURL := filePreviewURL(snapshot.baseURL, file)
	rawURL := fileRawURL(snapshot.baseURL, file)
	title := html.EscapeString(file.Name)
	metadataParts := []string{}
	if uploadedAt := formatUploadTime(file.UploadedAt); uploadedAt != "" {
		metadataParts = append(metadataParts, previewMetaItem(previewTimeIcon, uploadedAt))
	}
	metadataParts = append(metadataParts, previewMetaItem(previewSizeIcon, formatFileSize(file.Size)))
	if dimensions := imageDimensions(path); dimensions != "" {
		metadataParts = append(metadataParts, previewMetaItem(previewDimensionsIcon, dimensions))
	}
	metadataParts = append(metadataParts, previewMetaItemID("file-views", previewViewsIcon, formatAufrufCount(file.Views)))
	metadataHTML := strings.Join(metadataParts, "")
	contentType := html.EscapeString(file.ContentType)
	glowStrong, glowSoft := glowShadowColors(snapshot.store.glowColorForFile(file.ID))
	reportRecipient := url.PathEscape(strings.TrimSpace(snapshot.reportEmail))
	reportURL := "mailto:" + reportRecipient + "?subject=" + url.QueryEscape("Datei melden: "+file.Name) + "&body=" + url.QueryEscape("Bitte prüfe diese Datei: "+previewURL)
	brandingURL := brandLogoURL(snapshot.brandLogoPath)
	if brandingURL == "" {
		brandingURL = "/drop-logo.png"
	}
	embed := snapshot.embed
	siteName := embedSiteName(embed.siteName)
	embedColor := embedColorOrDefault(embed.color)
	iconURL := absoluteURL(snapshot.baseURL, brandingURL)
	iconAbs := html.EscapeString(iconURL)
	kindLabel := embedFileKind(file)
	uploadedAt := formatUploadTime(file.UploadedAt)
	sizeText := formatFileSize(file.Size)
	dimsText := imageDimensions(path)
	viewsText := formatAufrufCount(file.Views)
	ogTitle := applyEmbedTemplate(embed.title, file.Name, map[string]string{
		"name": file.Name,
		"site": siteName,
		"type": kindLabel,
		"size": sizeText,
	})
	ogParts := []string{}
	if extra := strings.TrimSpace(embed.extra); extra != "" {
		ogParts = append(ogParts, extra)
	}
	if embed.showType {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "type-"+kindLabel, kindLabel))
	}
	if embed.showDate && uploadedAt != "" {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "date", uploadedAt))
	}
	if embed.showSize {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "size", sizeText))
	}
	if embed.showDims && dimsText != "" {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "dims", dimsText))
	}
	if embed.showViews {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "views", viewsText))
	}
	ogDescription := strings.Join(ogParts, " · ")
	if ogDescription == "" {
		ogDescription = siteName
	}
	isImage := strings.HasPrefix(file.ContentType, "image/")
	imageURL := iconURL
	switch embed.image {
	case "none":
		imageURL = ""
	case "file":
		if isImage || strings.HasPrefix(file.ContentType, "video/") {
			imageURL = directURL
		}
	case "auto":
		if isImage {
			imageURL = directURL
		}
	}
	cardType := "summary"
	if embed.card == "large" && imageURL != "" {
		cardType = "summary_large_image"
	}
	media := fmt.Sprintf(`<div class="file-icon">%s</div>`, html.EscapeString(strings.ToUpper(file.ContentType)))
	imgW, imgH := 0, 0
	if isImage {
		imgW, imgH = imageSize(path)
	}
	metaMedia := openGraphImageMeta(imageURL, ogTitle, file.ContentType, imgW, imgH)
	if isImage {
		media = fmt.Sprintf(`<img class="preview-media" src="%s" alt="%s">`, html.EscapeString(directURL), title)
	} else if strings.HasPrefix(file.ContentType, "video/") {
		media = fmt.Sprintf(`<video class="preview-media" controls preload="metadata" src="%s"></video>`, html.EscapeString(directURL))
		metaMedia += fmt.Sprintf(`<meta property="og:video" content="%s"><meta property="og:video:type" content="%s">`, html.EscapeString(directURL), contentType)
	} else if strings.HasPrefix(file.ContentType, "audio/") {
		media = fmt.Sprintf(`<audio class="preview-audio" controls src="%s"></audio>`, html.EscapeString(directURL))
	}
	if embed.showSite {
		metaMedia += fmt.Sprintf(`<meta property="og:site_name" content="%s">`, html.EscapeString(siteName))
	}
	metaMedia += fmt.Sprintf(`<meta name="theme-color" content="%s">`, html.EscapeString(embedColor))
	ogTitleHTML := html.EscapeString(ogTitle)
	ogDescriptionHTML := html.EscapeString(ogDescription)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; media-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	viewsPollScript := fmt.Sprintf(`<script>(function(){var el=document.getElementById('file-views');if(!el)return;var text=el.querySelector('.meta-text');if(!text)return;var id=%s;function label(n){n=Number(n)||0;return n===1?'1 Aufruf':(n+' Aufrufe')}function refresh(){fetch('/api/files/'+encodeURIComponent(id)+'/views',{credentials:'omit',cache:'no-store'}).then(function(r){return r.ok?r.json():Promise.reject()}).then(function(data){if(data&&data.views!=null)text.textContent=label(data.views)}).catch(function(){})}refresh();setInterval(refresh,8000);document.addEventListener('visibilitychange',function(){if(!document.hidden)refresh()});})();</script>`, strconv.Quote(file.ID))
	page := fmt.Sprintf(`<!doctype html>
 <html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" href="%s">
 <title>%s · %s</title><meta name="description" content="%s"><meta property="og:type" content="website"><meta property="og:title" content="%s"><meta property="og:description" content="%s"><meta property="og:url" content="%s">%s<meta name="twitter:card" content="%s"><style>
 :root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;background:#091221;color:#e8eef8}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:#091221}.view-shell{width:min(1400px,100%%);min-height:calc(100vh - 48px);display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center}.brand{color:#aebcff;font-size:13px;font-weight:800;letter-spacing:.18em}.brand span{color:#f1a477}h1{margin:14px 0 6px;max-width:100%%;overflow-wrap:anywhere;font-size:clamp(24px,5vw,42px);letter-spacing:-.04em}p{color:#91a3bd;line-height:1.6}.meta{display:flex;align-items:center;justify-content:center;flex-wrap:wrap;gap:16px;color:#a9b9d1;font-size:13px;margin:14px 0 0}.meta-item{display:inline-flex;align-items:center;gap:7px;white-space:nowrap;line-height:1.25}.meta-icon{display:inline-flex;align-items:center;justify-content:center;width:16px;height:16px;color:#b4c5df}.meta-icon svg{display:block;width:16px;height:16px}.media-stage{width:100%%;display:grid;place-items:center;margin:20px 0}.preview-media{display:block;max-width:100%%;max-height:calc(100vh - 230px);object-fit:contain;border-radius:12px;background:#07101d;box-shadow:0 0 24px %s,0 0 48px %s,0 24px 80px rgba(0,0,0,.4)}.preview-audio{width:min(760px,100%%)}.file-icon{display:grid;place-items:center;min-height:180px;padding:36px;border:1px dashed #36517e;border-radius:12px;color:#8fa5ff;font-size:12px;letter-spacing:.1em}.actions{display:flex;justify-content:center;flex-wrap:wrap;gap:10px;margin-top:10px}a{display:inline-flex;align-items:center;justify-content:center;gap:7px;width:160px;padding:11px 15px;border-radius:8px;color:#b6c4db;background:#172943;border:1px solid #2d4569;text-decoration:none;font-weight:700;font-size:13px;transition:background-color .18s ease,border-color .18s ease,color .18s ease,transform .18s ease,box-shadow .18s ease}.button-icon{display:block;width:16px;height:16px;flex:0 0 16px}a:hover{color:#fff;background:#2a4270;border-color:#6d87c9;box-shadow:0 8px 20px rgba(45,69,105,.28);transform:translateY(-1px)}a.secondary{color:#b6c4db;background:#172943;border:1px solid #2d4569}a.secondary:hover{color:#fff;background:#2a4270;border-color:#6d87c9;box-shadow:0 8px 20px rgba(45,69,105,.28)}a:focus-visible{outline:2px solid #aebcff;outline-offset:3px}@media(max-width:520px){body{padding:16px}.view-shell{min-height:calc(100vh - 32px)}.meta{gap:11px;font-size:12px}.preview-media{max-height:calc(100vh - 210px)}}
 </style></head><body><main class="view-shell"><div class="media-stage">%s</div><p class="meta">%s</p><div class="actions"><a href="%s" download><svg class="button-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12"/><path d="m7 10 5 5 5-5"/><path d="M5 21h14"/></svg><span>Herunterladen</span></a><a class="secondary" href="%s"><svg class="button-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M14 4h6v6"/><path d="M20 4 11 13"/><path d="M18 13v6a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h6"/></svg><span>Original öffnen</span></a><a class="secondary" href="%s"><svg class="button-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m12 3 9 17H3L12 3Z"/><path d="M12 9v5"/><path d="M12 17h.01"/></svg><span>Datei melden</span></a></div></main><div style="position:fixed;right:20px;bottom:16px;display:flex;align-items:center;gap:7px;color:#91a3bd;font-size:11px;letter-spacing:.03em"><span>Powered by %s</span><img src="%s" alt="" style="width:22px;height:22px;object-fit:contain;filter:drop-shadow(0 3px 8px rgba(70,170,255,.28))"></div>%s</body></html>`, iconAbs, ogTitleHTML, html.EscapeString(siteName), ogDescriptionHTML, ogTitleHTML, ogDescriptionHTML, html.EscapeString(previewURL), metaMedia, cardType, glowStrong, glowSoft, media, metadataHTML, html.EscapeString(directURL), html.EscapeString(rawURL), html.EscapeString(reportURL), html.EscapeString(siteName), html.EscapeString(brandingURL), viewsPollScript)
	_, _ = io.WriteString(w, page)
}

func (s *Server) handleRawView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/raw/")
	name = filepath.Base(name)
	if name == "" || name == "." || strings.Contains(name, "..") {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	requestExt := strings.ToLower(filepath.Ext(name))
	id := strings.TrimSuffix(name, filepath.Ext(name))
	if id == "" || id != filepath.Base(id) {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	snapshot := s.snapshot()
	if snapshot.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
		return
	}
	file, ok := snapshot.store.find(id)
	if !ok {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	storedExt := filepath.Ext(file.URL)
	if storedExt == "" {
		storedExt = filepath.Ext(file.Name)
	}
	path := filepath.Join(snapshot.store.dataDir, "uploads", file.ID+storedExt)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	if !s.requireFileAccess(w, r, snapshot, file) {
		return
	}

	// Same share URL works for both:
	// - Discord/Telegram bots hitting /raw/{id}.png get pure image bytes (no branded card)
	// - Normal browsers always get the HTML raw viewer page
	if isDirectMediaExt(requestExt) && isSocialMediaBot(r.UserAgent()) {
		contentType := file.ContentType
		if contentType == "" {
			if detected := mime.TypeByExtension(storedExt); detected != "" {
				contentType = detected
			} else {
				contentType = "application/octet-stream"
			}
		}
		safeName := strings.NewReplacer("\\", "_", "\"", "_", "\r", "_", "\n", "_").Replace(file.Name)
		if safeName == "" {
			safeName = file.ID + storedExt
		}
		w.Header().Set("Content-Type", contentType)
		disposition := "attachment"
		if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") || contentType == "application/pdf" {
			disposition = "inline"
		}
		w.Header().Set("Content-Disposition", disposition+"; filename=\""+safeName+"\"")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		if file.PasswordProtected {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeFile(w, r, path)
		return
	}

	if r.Method == http.MethodGet {
		if views, err := snapshot.store.registerView(file.ID, viewVisitorHash(file.ID, r)); err == nil {
			file.Views = views
		}
	}

	directURL := strings.TrimRight(snapshot.baseURL, "/") + "/uploads/" + url.PathEscape(file.ID+storedExt)
	rawPageURL := fileRawURL(snapshot.baseURL, file)
	title := html.EscapeString(file.Name)
	contentType := html.EscapeString(file.ContentType)
	brandingURL := brandLogoURL(snapshot.brandLogoPath)
	if brandingURL == "" {
		brandingURL = "/drop-logo.png"
	}
	embed := snapshot.embed
	siteName := embedSiteName(embed.siteName)
	embedColor := embedColorOrDefault(embed.color)
	kindLabel := embedFileKind(file)
	sizeText := formatFileSize(file.Size)
	uploadedAt := formatUploadTime(file.UploadedAt)
	viewsText := formatAufrufCount(file.Views)
	ogTitle := applyEmbedTemplate(embed.title, file.Name, map[string]string{
		"name": file.Name,
		"site": siteName,
		"type": kindLabel,
		"size": sizeText,
	})
	ogParts := []string{}
	if extra := strings.TrimSpace(embed.extra); extra != "" {
		ogParts = append(ogParts, extra)
	}
	if embed.showType {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "type-"+kindLabel, kindLabel))
	}
	if embed.showDate && uploadedAt != "" {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "date", uploadedAt))
	}
	if embed.showSize {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "size", sizeText))
	}
	if embed.showViews {
		ogParts = append(ogParts, withEmbedIcon(embed.showIcons, "views", viewsText))
	}
	ogDescription := strings.Join(ogParts, " · ")
	if ogDescription == "" {
		ogDescription = siteName
	}
	isImage := strings.HasPrefix(file.ContentType, "image/")
	imageURL := absoluteURL(snapshot.baseURL, brandingURL)
	switch embed.image {
	case "none":
		imageURL = ""
	case "file":
		if isImage || strings.HasPrefix(file.ContentType, "video/") {
			imageURL = directURL
		}
	default: // auto
		if isImage {
			imageURL = directURL
		}
	}
	imgW, imgH := 0, 0
	if isImage {
		imgW, imgH = imageSize(path)
	}
	metaMedia := openGraphImageMeta(imageURL, ogTitle, file.ContentType, imgW, imgH)
	if strings.HasPrefix(file.ContentType, "video/") {
		metaMedia += fmt.Sprintf(`<meta property="og:video" content="%s"><meta property="og:video:type" content="%s">`, html.EscapeString(directURL), contentType)
	}
	if embed.showSite {
		metaMedia += fmt.Sprintf(`<meta property="og:site_name" content="%s">`, html.EscapeString(siteName))
	}
	metaMedia += fmt.Sprintf(`<meta name="theme-color" content="%s">`, html.EscapeString(embedColor))
	cardType := "summary"
	if embed.card == "large" && imageURL != "" {
		cardType = "summary_large_image"
	}
	media := fmt.Sprintf(`<div class="file-icon">%s</div>`, contentType)
	mediaSource := html.EscapeString(directURL)
	if isImage {
		media = fmt.Sprintf(`<img src="%s" alt="%s">`, mediaSource, title)
	} else if strings.HasPrefix(file.ContentType, "video/") {
		media = fmt.Sprintf(`<video controls preload="metadata" src="%s"></video>`, mediaSource)
	} else if strings.HasPrefix(file.ContentType, "audio/") {
		media = fmt.Sprintf(`<audio controls src="%s"></audio>`, mediaSource)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if file.PasswordProtected {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self'; media-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	page := fmt.Sprintf(`<!doctype html>
 <html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" type="image/png" href="%s"><title>%s · %s</title><meta name="description" content="%s"><meta property="og:type" content="website"><meta property="og:title" content="%s"><meta property="og:description" content="%s"><meta property="og:url" content="%s">%s<meta name="twitter:card" content="%s"><style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;background:#091221;color:#e8eef8}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:12px;background:#091221}.raw-view{width:min(1760px,calc(100vw - 24px));min-height:calc(100vh - 24px);display:flex;align-items:center;justify-content:center;position:relative;margin:0 auto}.raw-view img,.raw-view video{display:block;max-width:100%%;max-height:calc(100vh - 24px);width:auto;height:auto;object-fit:contain;border-radius:12px;box-shadow:0 24px 80px rgba(0,0,0,.4)}.raw-view audio{width:min(760px,100%%)}.file-icon{padding:36px;border:1px dashed #36517e;border-radius:12px;color:#8fa5ff}.filename{position:absolute;left:0;right:0;bottom:0;text-align:center;color:#91a3bd;font-size:13px;overflow-wrap:anywhere}.download{display:none;position:absolute;top:0;right:0;padding:10px 14px;border-radius:8px;color:#fff;background:#536de0;text-decoration:none;font-weight:700;font-size:13px}@media(max-width:520px){body{padding:8px}.raw-view{width:calc(100vw - 16px);min-height:calc(100vh - 16px)}.raw-view img,.raw-view video{max-width:100%%;max-height:calc(100vh - 16px)}}
</style></head><body><main class="raw-view"><a class="download" href="%s" download>Download</a>%s</main><div style="position:fixed;right:20px;bottom:16px;display:flex;align-items:center;gap:7px;color:#91a3bd;font-size:11px;letter-spacing:.03em"><span>Powered by %s</span><img src="%s" alt="" style="width:22px;height:22px;object-fit:contain;filter:drop-shadow(0 3px 8px rgba(70,170,255,.28))"></div></body></html>`,
		html.EscapeString(brandingURL),
		html.EscapeString(ogTitle), html.EscapeString(siteName), html.EscapeString(ogDescription),
		html.EscapeString(ogTitle), html.EscapeString(ogDescription), html.EscapeString(rawPageURL),
		metaMedia, cardType,
		mediaSource, media, html.EscapeString(siteName), html.EscapeString(brandingURL))
	_, _ = io.WriteString(w, page)
}

func (s *Server) serveStoredUpload(w http.ResponseWriter, r *http.Request, name string) {
	name = filepath.Base(name)
	if name == "" || name == "." || strings.Contains(name, "..") {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	snapshot := s.snapshot()
	if snapshot.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "setup required"})
		return
	}
	requestExt := filepath.Ext(name)
	fileID := strings.TrimSuffix(name, requestExt)
	file, found := snapshot.store.find(fileID)
	storedName := name
	if found {
		storedExt := filepath.Ext(file.URL)
		if storedExt == "" {
			storedExt = filepath.Ext(file.Name)
		}
		if storedExt != "" {
			storedName = file.ID + storedExt
		}
	}
	path := filepath.Join(snapshot.store.dataDir, "uploads", storedName)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeFileUnavailablePage(w, r, http.StatusNotFound)
		return
	}
	if r.Method == http.MethodPost && !found {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if found && !s.requireFileAccess(w, r, snapshot, file) {
		return
	}
	displayName := storedName
	if found && file.Name != "" {
		displayName = file.Name
	}
	safeName := strings.NewReplacer("\\", "_", "\"", "_", "\r", "_", "\n", "_").Replace(displayName)
	contentType := "application/octet-stream"
	if found && file.ContentType != "" {
		contentType = file.ContentType
	} else if detected := mime.TypeByExtension(filepath.Ext(storedName)); detected != "" {
		contentType = detected
	}
	w.Header().Set("Content-Type", contentType)
	disposition := "attachment"
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") || contentType == "application/pdf" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+"; filename=\""+safeName+"\"")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if found && file.PasswordProtected {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleDirectMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	s.serveStoredUpload(w, r, strings.TrimPrefix(r.URL.Path, "/i/"))
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	s.serveStoredUpload(w, r, strings.TrimPrefix(r.URL.Path, "/uploads/"))
}

func randomID() (string, error) {
	buffer := make([]byte, 6)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func randomUploadID() (string, error) {
	// Kompakter UUID-Stil mit 96 Bit Zufall ohne Original-Dateinamen.
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	value := hex.EncodeToString(buffer)
	return value[:8] + "-" + value[8:12] + "-" + value[12:16] + "-" + value[16:20] + "-" + value[20:], nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeErrorHTMLPage(w http.ResponseWriter, r *http.Request, status int, title, heading, message, actions string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	page := fmt.Sprintf(`<!doctype html>
<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="icon" type="image/png" href="/drop-logo.png"><title>%s</title><style>
:root{color-scheme:dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,sans-serif;background:#080f1d;color:#e8eef8}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:radial-gradient(circle at 50%% 15%%,rgba(78,101,181,.16),transparent 38%%),#080f1d}.error-shell{width:min(520px,100%%);padding:40px 34px;text-align:center;border:1px solid rgba(77,103,153,.55);border-radius:18px;background:rgba(13,25,43,.88);box-shadow:0 24px 80px rgba(0,0,0,.36)}.error-code{margin:0 0 19px;color:#8398ff;font-size:12px;font-weight:800;letter-spacing:.22em}.error-icon{display:grid;place-items:center;width:52px;height:52px;margin:0 auto 20px;border:1px solid rgba(255,141,117,.42);border-radius:15px;color:#ffad9a;background:rgba(153,56,48,.16);font-size:25px;font-weight:700}.error-shell h1{margin:0;color:#f1f4fb;font-size:clamp(25px,5vw,34px);letter-spacing:-.045em}.error-shell p{margin:14px auto 0;max-width:390px;color:#96a8c2;font-size:13px;line-height:1.7}.error-actions{display:flex;justify-content:center;gap:10px;flex-wrap:wrap;margin-top:24px}.error-actions a{display:inline-flex;align-items:center;gap:8px;min-height:34px;padding:8px 14px;border-radius:8px;text-decoration:none;font-size:12px;font-weight:700}.error-actions .primary{color:#dce5ff;border:1px solid #4f67b8;background:#243968}.error-actions .secondary{color:#aab9ff;border:1px solid #304568;background:#142546}.error-path{display:inline-block;margin-top:18px;padding:6px 10px;border-radius:6px;color:#8091ab;background:rgba(16,30,50,.9);font-size:11px}@media(max-width:520px){body{padding:16px}.error-shell{padding:32px 22px}}
</style></head><body><main class="error-shell"><div class="error-icon" aria-hidden="true">!</div><p class="error-code">FEHLER %d</p><h1>%s</h1><p>%s</p>%s</main></body></html>`, html.EscapeString(title), status, html.EscapeString(heading), html.EscapeString(message), actions)
	_, _ = io.WriteString(w, page)
}

func writeFileUnavailablePage(w http.ResponseWriter, r *http.Request, status int) {
	writeErrorHTMLPage(w, r, status, "404 · Datei nicht verfügbar", "Datei nicht verfügbar", "Diese Datei wurde gelöscht oder ist auf dem Server nicht mehr verfügbar.", "")
}

func writeNotFoundPage(w http.ResponseWriter, r *http.Request) {
	path := html.EscapeString(r.URL.Path)
	actions := fmt.Sprintf(`<div class="error-actions"><a class="primary" href="/dashboard">Zum Dashboard →</a></div><code class="error-path">%s</code>`, path)
	writeErrorHTMLPage(w, r, http.StatusNotFound, "404 · Seite nicht gefunden", "Seite nicht gefunden.", "Diese Adresse gibt es hier nicht oder sie wurde verschoben.", actions)
}

func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes)
	for i, unit := range units {
		value /= 1024
		// Switch early at 1000 so ~1022.9 MB becomes ~1.0 GB.
		if value < 1000 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseStorageBytes(value string) int64 {
	megabytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil || megabytes < 1 || megabytes > 1024*1024*1024 {
		megabytes = 5120
	}
	return megabytes * 1024 * 1024
}

func parseLimitMB(value string, fallback int64) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	megabytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil || megabytes < 1 || megabytes > 1024*1024*1024 {
		return 0, errors.New("ungültiges limit")
	}
	return megabytes * 1024 * 1024, nil
}

func (c AppConfig) mysqlDSN() string {
	c = c.withDefaults()
	if c.MySQLDSN != "" {
		return c.MySQLDSN
	}
	host := c.MySQLHost
	config := mysqlDriver.NewConfig()
	config.User = c.MySQLUser
	config.Passwd = c.MySQLPassword
	config.Net = "tcp"
	config.Addr = host + ":" + c.MySQLPort
	config.DBName = c.MySQLDatabase
	config.ParseTime = true
	config.Collation = "utf8mb4_unicode_ci"
	config.Loc = time.UTC
	tlsMode := c.MySQLTLS
	if tlsMode == "" {
		tlsMode = "true"
		for _, localHost := range []string{"127.0.0.1", "localhost", "::1", "host.docker.internal", "mysql"} {
			if strings.EqualFold(host, localHost) {
				tlsMode = "false"
				break
			}
		}
	}
	config.TLSConfig = tlsMode
	return config.FormatDSN()
}

func main() {
	dataDir := envOr("DATA_DIR", "../data")
	frontendDir := envOr("FRONTEND_DIR", "../frontend/dist")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		log.Fatal(err)
	}
	config, configured, err := loadAppConfig(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	config = config.withDefaults()
	maxUploadMB, _ := strconv.ParseInt(config.MaxUploadMB, 10, 64)
	if maxUploadMB < 1 {
		maxUploadMB = 100
	}
	secureCookies := !strings.EqualFold(config.CookieSecure, "false") && config.CookieSecure != "0"
	server := &Server{
		dataDir:       dataDir,
		configPath:    filepath.Join(dataDir, "config.json"),
		baseURL:       config.BaseURL,
		reportEmail:   config.AdminEmail,
		databaseType:  config.databaseType(),
		apiKey:        config.UploadAPIKey,
		adminAPIKey:   config.AdminAPIKey,
		corsOrigin:    strings.TrimRight(config.CORSOrigin, "/"),
		frontend:      http.FileServer(http.Dir(frontendDir)),
		maxUpload:     maxUploadMB * 1024 * 1024,
		secureCookies: secureCookies,
		loginLimiter:  newLoginLimiter(),
		setupLimiter:  newLoginLimiter(),
		resetLimiter:  newLoginLimiter(),
		filePasswordLimiter: newLoginLimiter(),
		mailer:        newMailer(config),
	}
	if configured {
		db, err := newDatabase(config.databaseType(), config.databaseDSN(dataDir))
		if err != nil {
			log.Fatal(err)
		}
		store, err := newStore(db, dataDir, parseStorageBytes(config.MaxStorageMB), config.databaseType())
		if err != nil {
			db.Close()
			log.Fatal(err)
		}
		auth, err := newAuthStore(db, dataDir, config.databaseType())
		if err != nil {
			db.Close()
			log.Fatal(err)
		}
		if err := auth.ensureAdmin(config.AdminEmail, config.AdminPassword); err != nil {
			db.Close()
			log.Fatal(err)
		}
		if resetPassword := os.Getenv("RESET_ADMIN_PASSWORD"); resetPassword != "" {
			if err := auth.resetAdminPassword(config.AdminEmail, resetPassword); err != nil {
				db.Close()
				log.Fatalf("Admin-Passwort konnte nicht zurückgesetzt werden: %v", err)
			}
			log.Println("Admin-Passwort wurde einmalig zurückgesetzt; Sessions wurden abgemeldet.")
			return
		}
		server.activate(config, store, auth)
		defer db.Close()
		log.Printf("ShareX Uploader läuft hinter %s mit %s", config.BaseURL, config.databaseType())
	} else {
		log.Printf("ShareX Uploader wartet auf Einrichtung unter %s/setup", strings.TrimRight(config.BaseURL, "/"))
	}

	log.Fatal(http.ListenAndServe(":"+config.Port, server.routes()))
}

