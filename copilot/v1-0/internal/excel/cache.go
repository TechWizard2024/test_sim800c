package excel

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ExcelCache struct {
	reader     *ExcelReader
	cache      map[string][]USSDCode
	lastUpdate time.Time
	mu         sync.RWMutex
	ttl        time.Duration
	logger     *logrus.Logger
}

func NewExcelCache(reader *ExcelReader, ttlMinutes int, logger *logrus.Logger) *ExcelCache {
	return &ExcelCache{
		reader: reader,
		cache:  make(map[string][]USSDCode),
		ttl:    time.Duration(ttlMinutes) * time.Minute,
		logger: logger,
	}
}

func (c *ExcelCache) GetConsultCodes(carrier string) []USSDCode {
	c.mu.RLock()

	// Vérifier si le cache est encore valide
	if time.Since(c.lastUpdate) < c.ttl {
		if codes, ok := c.cache["consult_"+carrier]; ok {
			c.mu.RUnlock()
			return codes
		}
	}
	c.mu.RUnlock()

	// Recharger
	c.mu.Lock()
	defer c.mu.Unlock()

	codes := c.reader.GetConsultCodes(carrier)
	c.cache["consult_"+carrier] = codes
	c.lastUpdate = time.Now()

	return codes
}

func (c *ExcelCache) GetServiceNCodes(carrier string) []USSDCode {
	c.mu.RLock()

	if time.Since(c.lastUpdate) < c.ttl {
		if codes, ok := c.cache["service_"+carrier]; ok {
			c.mu.RUnlock()
			return codes
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	codes := c.reader.GetServiceNCodes(carrier)
	c.cache["service_"+carrier] = codes
	c.lastUpdate = time.Now()

	return codes
}

func (c *ExcelCache) GetByUSSDCode(code string) (bool, USSDCode) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.reader.GetByUSSDCode(code)
}

func (c *ExcelCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string][]USSDCode)
	c.lastUpdate = time.Time{}
	c.logger.Info("Cache Excel invalidé")
}

func (c *ExcelCache) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.reader.Load(); err != nil {
		return err
	}

	c.cache = make(map[string][]USSDCode)
	c.lastUpdate = time.Now()

	return nil
}
