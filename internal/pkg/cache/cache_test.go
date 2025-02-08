package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	// Тест добавления и получения значения
	t.Run("Set and Get", func(t *testing.T) {
		key := "test_key"
		value := []byte("test_value")

		cache.Set(key, value)

		if got, exists := cache.Get(key); !exists {
			t.Error("Expected value to exist in cache")
		} else if string(got) != string(value) {
			t.Errorf("Expected %s, got %s", string(value), string(got))
		}
	})

	// Тест истечения срока жизни значения
	t.Run("Expiration", func(t *testing.T) {
		key := "test_expiration"
		value := []byte("test_value")

		cache.Set(key, value)
		time.Sleep(150 * time.Millisecond)

		if _, exists := cache.Get(key); exists {
			t.Error("Expected value to be expired")
		}
	})

	// Тест удаления значения
	t.Run("Delete", func(t *testing.T) {
		key := "test_delete"
		value := []byte("test_value")

		cache.Set(key, value)
		cache.Delete(key)

		if _, exists := cache.Get(key); exists {
			t.Error("Expected value to be deleted")
		}
	})

	// Тест автоматической очистки
	t.Run("Cleanup", func(t *testing.T) {
		key := "test_cleanup"
		value := []byte("test_value")

		cache.Set(key, value)
		time.Sleep(150 * time.Millisecond)

		if _, exists := cache.Get(key); exists {
			t.Error("Expected value to be cleaned up")
		}
	})
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewCache(1 * time.Second)
	done := make(chan bool)

	// Тест параллельного доступа
	t.Run("Concurrent access", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			go func(id int) {
				key := "test_key"
				value := []byte("test_value")

				cache.Set(key, value)
				if _, exists := cache.Get(key); !exists {
					t.Error("Expected value to exist in cache")
				}

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
