package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StateClosed(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test",
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	})

	// Проверяем начальное состояние
	if cb.State() != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.State())
	}

	// Проверяем успешные запросы
	for i := 0; i < 5; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
	}

	// Проверяем, что состояние не изменилось
	if cb.State() != StateClosed {
		t.Errorf("Expected state to remain Closed after successes, got %v", cb.State())
	}
}

func TestCircuitBreaker_StateOpen(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test",
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	})

	// Вызываем ошибки до достижения порога
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("Expected test error, got: %v", err)
		}
	}

	// Проверяем переход в состояние Open
	if cb.State() != StateOpen {
		t.Errorf("Expected state to be Open after failures, got %v", cb.State())
	}

	// Проверяем, что запросы отклоняются
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Expected circuit open error, got: %v", err)
	}
}

func TestCircuitBreaker_StateHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test",
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	})

	// Переводим в состояние Open
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return testErr
		})
	}

	// Ждем перехода в Half-Open
	time.Sleep(150 * time.Millisecond)

	// Проверяем успешные запросы в Half-Open
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected success in half-open state, got error: %v", err)
		}
	}

	// Проверяем переход в Closed
	if cb.State() != StateClosed {
		t.Errorf("Expected state to be Closed after successes in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test",
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	})

	// Переводим в состояние Open
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return testErr
		})
	}

	// Ждем перехода в Half-Open
	time.Sleep(150 * time.Millisecond)

	// Проверяем, что ошибка в Half-Open возвращает в Open
	err := cb.Execute(context.Background(), func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected test error in half-open state, got: %v", err)
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state to be Open after failure in half-open, got %v", cb.State())
	}
}
