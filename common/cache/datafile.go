package cache

const (
	// UnifiedDataDBFile is the single SQLite data file used for local state.
	UnifiedDataDBFile = "aliang.db"
)

// GetUnifiedDataDBPath returns the canonical path for the shared local SQLite database.
func GetUnifiedDataDBPath() (string, error) {
	return GetCacheFile(UnifiedDataDBFile)
}
