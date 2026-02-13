package user

// Storage interface defines the methods for user-related storage operations
type Storage interface {
	EnsureUser(telegramID int64) error
	SetHevyAPIKey(telegramID int64, key string) error
	GetHevyAPIKey(telegramID int64) (string, error)
	ClearHevyAPIKey(telegramID int64) error
}

// Manager handles user-related operations
type Manager struct {
	userStorage Storage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage) *Manager {
	return &Manager{
		userStorage: userStorage,
	}
}

// EnsureUser ensures a user exists in the storage
func (m *Manager) EnsureUser(telegramID int64) error {
	return m.userStorage.EnsureUser(telegramID)
}

// SetHevyAPIKey stores the Hevy API key for a user
func (m *Manager) SetHevyAPIKey(telegramID int64, key string) error {
	return m.userStorage.SetHevyAPIKey(telegramID, key)
}

// GetHevyAPIKey retrieves the Hevy API key for a user
func (m *Manager) GetHevyAPIKey(telegramID int64) (string, error) {
	return m.userStorage.GetHevyAPIKey(telegramID)
}

// ClearHevyAPIKey removes the Hevy API key for a user
func (m *Manager) ClearHevyAPIKey(telegramID int64) error {
	return m.userStorage.ClearHevyAPIKey(telegramID)
}

// IsHevyConnected checks if a user has a Hevy API key stored
func (m *Manager) IsHevyConnected(telegramID int64) (bool, error) {
	key, err := m.userStorage.GetHevyAPIKey(telegramID)
	if err != nil {
		return false, err
	}
	return key != "", nil
}
