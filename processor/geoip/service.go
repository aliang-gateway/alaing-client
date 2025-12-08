package geoip

import (
	"fmt"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// Service provides GeoIP lookup functionality using MaxMind GeoLite2 database
type Service struct {
	mu      sync.RWMutex
	reader  *geoip2.Reader
	enabled bool
	dbPath  string
}

var (
	defaultService *Service
	once           sync.Once
)

// GetService returns the singleton GeoIP service instance
func GetService() *Service {
	once.Do(func() {
		defaultService = &Service{
			enabled: false,
		}
	})
	return defaultService
}

// LoadDatabase loads the MaxMind GeoLite2 database from the specified path
func (s *Service) LoadDatabase(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reader, err := geoip2.Open(path)
	if err != nil {
		return fmt.Errorf("failed to load GeoIP database from %s: %w", path, err)
	}

	// Close old reader if exists
	if s.reader != nil {
		s.reader.Close()
	}

	s.reader = reader
	s.dbPath = path
	s.enabled = true

	return nil
}

// LookupCountry performs a country-level GeoIP lookup for the given IP address
func (s *Service) LookupCountry(ip net.IP) (*CountryInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.enabled || s.reader == nil {
		return nil, fmt.Errorf("GeoIP service not initialized")
	}

	record, err := s.reader.Country(ip)
	if err != nil {
		return nil, fmt.Errorf("GeoIP lookup failed for %s: %w", ip.String(), err)
	}

	return &CountryInfo{
		ISOCode: record.Country.IsoCode,
		Name:    record.Country.Names["en"],
	}, nil
}

// IsChina checks if the given IP address is located in China
func (s *Service) IsChina(ip net.IP) bool {
	country, err := s.LookupCountry(ip)
	if err != nil {
		return false
	}
	return country.ISOCode == "CN"
}

// IsEnabled returns whether the GeoIP service is currently enabled
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// GetDatabasePath returns the path to the currently loaded database
func (s *Service) GetDatabasePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dbPath
}

// Close closes the GeoIP database reader
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader != nil {
		err := s.reader.Close()
		s.reader = nil
		s.enabled = false
		return err
	}
	return nil
}

// Disable disables the GeoIP service without closing the database
func (s *Service) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
}

// Enable enables the GeoIP service if a database is loaded
func (s *Service) Enable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader == nil {
		return fmt.Errorf("cannot enable GeoIP service: no database loaded")
	}

	s.enabled = true
	return nil
}
