package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/krispaisarn/web-otp/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	instance *gorm.DB
	once     sync.Once
	initErr  error
)

func init() {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if pem := loadPEM(); pem != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			panic("db: TIDB_CA_CERT contains invalid PEM data")
		}
		cfg.RootCAs = pool
	}
	mysqlDriver.RegisterTLSConfig("tidb", cfg)
}

// Get returns the shared GORM instance, initializing and migrating on first call.
func Get() (*gorm.DB, error) {
	once.Do(func() {
		dsn := normalizeDSN(buildRawDSN())

		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			initErr = fmt.Errorf("opening database: %w", err)
			return
		}

		sql, _ := db.DB()
		sql.SetMaxOpenConns(10)
		sql.SetMaxIdleConns(5)

		if err := db.AutoMigrate(&models.OTP{}); err != nil {
			initErr = fmt.Errorf("schema migration: %w", err)
			return
		}

		instance = db
	})
	return instance, initErr
}

func buildRawDSN() string {
	if dsn := os.Getenv("TIDB_DSN"); dsn != "" {
		return dsn
	}
	port := os.Getenv("TIDB_PORT")
	if port == "" {
		port = "4000"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?tls=tidb&parseTime=true&loc=UTC",
		os.Getenv("TIDB_USER"),
		os.Getenv("TIDB_PASSWORD"),
		os.Getenv("TIDB_HOST"),
		port,
		os.Getenv("TIDB_DATABASE"),
	)
}

// normalizeDSN converts common formats to the Go MySQL driver format.
// Accepts: correct Go format, mysql:// URL, or bare user:pass@host:port/db.
func normalizeDSN(dsn string) string {
	if strings.Contains(dsn, "@tcp(") || strings.Contains(dsn, "@unix(") {
		return dsn
	}
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		user := u.User.Username()
		pass, _ := u.User.Password()
		host := u.Host
		if !strings.Contains(host, ":") {
			host += ":4000"
		}
		db := strings.TrimPrefix(u.Path, "/")
		return fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=tidb&parseTime=true&loc=UTC", user, pass, host, db)
	}
	if at := strings.LastIndex(dsn, "@"); at != -1 {
		creds := dsn[:at]
		rest := dsn[at+1:]
		slash := strings.Index(rest, "/")
		if slash == -1 {
			return dsn
		}
		hostPort := rest[:slash]
		remainder := rest[slash:]
		if !strings.Contains(hostPort, ":") {
			hostPort += ":4000"
		}
		if !strings.Contains(remainder, "?") {
			remainder += "?tls=tidb&parseTime=true&loc=UTC"
		}
		return fmt.Sprintf("%s@tcp(%s)%s", creds, hostPort, remainder)
	}
	return dsn
}

func loadPEM() string {
	if path := os.Getenv("TIDB_CA_CERT_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("db: reading TIDB_CA_CERT_FILE %q: %v", path, err))
		}
		return strings.TrimSpace(string(data))
	}
	return os.Getenv("TIDB_CA_CERT")
}
