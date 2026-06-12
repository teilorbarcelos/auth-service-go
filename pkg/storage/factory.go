package storage

import "fmt"

func NewStorageProvider(driver string, bucket string) (StorageProvider, error) {
	switch driver {
	default:
		return nil, fmt.Errorf("driver de storage não suportado: %s", driver)
	}
}
