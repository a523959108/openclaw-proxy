package geoip

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/oschwald/geoip2-golang"
)

const (
	defaultDBURL         = "https://github.com/P3TERX/GeoLite.mmdb/releases/download/2025.03.25/GeoLite2-Country.mmdb"
	dbFileName           = "GeoLite2-Country.mmdb"
	defaultCheckInterval = 24 * time.Hour // Check for updates every 24 hours
)

// Manager handles GeoIP database operations
type Manager struct {
	db        *geoip2.Reader
	dbPath    string
	dbVersion *version.Version
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

var (
	instance *Manager
	once     sync.Once
)

// GetInstance returns the singleton GeoIP manager
func GetInstance() *Manager {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		instance = &Manager{
			ctx:    ctx,
			cancel: cancel,
			dbPath: filepath.Join(os.TempDir(), dbFileName),
		}
		// Initialize database
		go instance.init()
	})
	return instance
}

// init initializes the GeoIP database
func (m *Manager) init() {
	// Try to load existing database
	if _, err := os.Stat(m.dbPath); err == nil {
		if err := m.loadDB(); err == nil {
			log.Println("GeoIP database loaded successfully")
			go m.startAutoUpdate()
			return
		}
	}

	// Download database if not exists or failed to load
	log.Println("Downloading GeoIP database...")
	if err := m.downloadDB(); err != nil {
		log.Printf("Failed to download GeoIP database: %v, GEOIP rules will not work", err)
		return
	}

	if err := m.loadDB(); err != nil {
		log.Printf("Failed to load GeoIP database: %v, GEOIP rules will not work", err)
		return
	}

	log.Println("GeoIP database downloaded and loaded successfully")
	go m.startAutoUpdate()
}

// loadDB loads the database from file
func (m *Manager) loadDB() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		m.db.Close()
	}

	var err error
	m.db, err = geoip2.Open(m.dbPath)
	return err
}

// downloadDB downloads the latest GeoIP database
func (m *Manager) downloadDB() error {
	resp, err := http.Get(defaultDBURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(m.dbPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// startAutoUpdate starts automatic database update
func (m *Manager) startAutoUpdate() {
	ticker := time.NewTicker(defaultCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			log.Println("Checking for GeoIP database updates...")
			// For simplicity, just redownload the database periodically
			// In production, you would check etag or last-modified header first
			if err := m.downloadDB(); err != nil {
				log.Printf("Failed to update GeoIP database: %v", err)
				continue
			}
			if err := m.loadDB(); err != nil {
				log.Printf("Failed to load updated GeoIP database: %v", err)
				continue
			}
			log.Println("GeoIP database updated successfully")
		}
	}
}

// LookupCountry returns the ISO country code for an IP address
func (m *Manager) LookupCountry(ip net.IP) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return "", nil // No database available
	}

	record, err := m.db.Country(ip)
	if err != nil {
		return "", err
	}

	return record.Country.IsoCode, nil
}

// Stop stops the auto update process
func (m *Manager) Stop() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.db != nil {
		m.db.Close()
	}
}
