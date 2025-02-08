package circuitbreaker

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// State представляет состояние Circuit Breaker
type State int

const (
	StateClosed   State = iota // Нормальное состояние, запросы проходят
	StateOpen                  // Состояние отказа, запросы блокируются
	StateHalfOpen              // Тестовое состояние, пропускается часть запросов
)

var (
	// ErrCircuitOpen возвращается, когда Circuit Breaker находится в открытом состоянии
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// Метрики для Circuit Breaker
	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of the circuit breaker (0: Closed, 1: Open, 2: Half-Open)",
		},
		[]string{"name", "pod_name", "namespace"},
	)

	circuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of failures detected by circuit breaker",
		},
		[]string{"name", "pod_name", "namespace"},
	)

	circuitBreakerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests passed through circuit breaker",
		},
		[]string{"name", "pod_name", "namespace", "status"},
	)

	// Kubernetes-специфичные метрики
	circuitBreakerPodHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_pod_health",
			Help: "Health status of the pod based on circuit breaker state (0: Unhealthy, 1: Healthy)",
		},
		[]string{"name", "pod_name", "namespace"},
	)

	circuitBreakerRecoveryTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "circuit_breaker_recovery_duration_seconds",
			Help:    "Time taken to recover from Open to Closed state",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"name", "pod_name", "namespace"},
	)
)

// Config содержит настройки для Circuit Breaker
type Config struct {
	Name             string        // Имя для идентификации в метриках
	FailureThreshold int           // Количество ошибок до перехода в состояние Open
	ResetTimeout     time.Duration // Время до перехода из Open в Half-Open
	HalfOpenMaxCalls int           // Максимальное количество запросов в состоянии Half-Open
	SuccessThreshold int           // Количество успешных запросов для перехода из Half-Open в Closed
	PodName          string        // Имя пода в Kubernetes
	Namespace        string        // Namespace в Kubernetes
}

// CircuitBreaker реализует паттерн Circuit Breaker
type CircuitBreaker struct {
	config Config
	state  State

	failures        int       // Счетчик последовательных ошибок
	lastStateChange time.Time // Время последнего изменения состояния
	successes       int       // Счетчик успешных запросов в Half-Open состоянии
	halfOpenCalls   int       // Счетчик запросов в Half-Open состоянии
	openStartTime   time.Time // Время перехода в состояние Open

	mu sync.RWMutex
}

// NewCircuitBreaker создает новый экземпляр Circuit Breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	if config.PodName == "" {
		config.PodName = os.Getenv("HOSTNAME")
	}
	if config.Namespace == "" {
		config.Namespace = os.Getenv("POD_NAMESPACE")
	}

	cb := &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}

	// Инициализация начального состояния в метриках
	labels := prometheus.Labels{
		"name":      config.Name,
		"pod_name":  config.PodName,
		"namespace": config.Namespace,
	}

	circuitBreakerState.With(labels).Set(float64(StateClosed))
	circuitBreakerPodHealth.With(labels).Set(1)

	return cb
}

// Execute выполняет функцию с учетом состояния Circuit Breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		circuitBreakerRequests.WithLabelValues(cb.config.Name, cb.config.PodName, cb.config.Namespace, "rejected").Inc()
		return ErrCircuitOpen
	}

	err := fn()
	cb.handleResult(err)

	if err != nil {
		circuitBreakerRequests.WithLabelValues(cb.config.Name, cb.config.PodName, cb.config.Namespace, "failure").Inc()
		return err
	}

	circuitBreakerRequests.WithLabelValues(cb.config.Name, cb.config.PodName, cb.config.Namespace, "success").Inc()
	return nil
}

// allowRequest проверяет, можно ли выполнить запрос
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) > cb.config.ResetTimeout {
			cb.toHalfOpen()
			return true
		}
		return false
	case StateHalfOpen:
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return false
		}
		cb.halfOpenCalls++
		return true
	default:
		return false
	}
}

// handleResult обрабатывает результат выполнения запроса
func (cb *CircuitBreaker) handleResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure обрабатывает ошибку выполнения запроса
func (cb *CircuitBreaker) onFailure() {
	circuitBreakerFailures.WithLabelValues(cb.config.Name, cb.config.PodName, cb.config.Namespace).Inc()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	}
}

// onSuccess обрабатывает успешное выполнение запроса
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.toClosed()
		}
	}
}

// toOpen переводит Circuit Breaker в состояние Open
func (cb *CircuitBreaker) toOpen() {
	cb.state = StateOpen
	cb.lastStateChange = time.Now()
	cb.openStartTime = time.Now()
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0

	labels := prometheus.Labels{
		"name":      cb.config.Name,
		"pod_name":  cb.config.PodName,
		"namespace": cb.config.Namespace,
	}
	circuitBreakerState.With(labels).Set(float64(StateOpen))
	circuitBreakerPodHealth.With(labels).Set(0)
}

// toHalfOpen переводит Circuit Breaker в состояние Half-Open
func (cb *CircuitBreaker) toHalfOpen() {
	cb.state = StateHalfOpen
	cb.lastStateChange = time.Now()
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0

	labels := prometheus.Labels{
		"name":      cb.config.Name,
		"pod_name":  cb.config.PodName,
		"namespace": cb.config.Namespace,
	}
	circuitBreakerState.With(labels).Set(float64(StateHalfOpen))
	circuitBreakerPodHealth.With(labels).Set(1) // В Half-Open состоянии под все еще может обрабатывать часть запросов
}

// toClosed переводит Circuit Breaker в состояние Closed
func (cb *CircuitBreaker) toClosed() {
	cb.state = StateClosed
	cb.lastStateChange = time.Now()
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0

	labels := prometheus.Labels{
		"name":      cb.config.Name,
		"pod_name":  cb.config.PodName,
		"namespace": cb.config.Namespace,
	}
	circuitBreakerState.With(labels).Set(float64(StateClosed))
	circuitBreakerPodHealth.With(labels).Set(1)

	// Если был переход из Open состояния, записываем время восстановления
	if !cb.openStartTime.IsZero() {
		recoveryTime := time.Since(cb.openStartTime).Seconds()
		circuitBreakerRecoveryTime.With(labels).Observe(recoveryTime)
		cb.openStartTime = time.Time{} // Сбрасываем время
	}
}

// State возвращает текущее состояние Circuit Breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// IsHealthy возвращает true, если Circuit Breaker находится в состоянии, позволяющем обрабатывать запросы
func (cb *CircuitBreaker) IsHealthy() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Считаем сервис здоровым, если он в Closed состоянии или в Half-Open с возможностью принимать запросы
	isHealthy := cb.state == StateClosed || (cb.state == StateHalfOpen && cb.halfOpenCalls < cb.config.HalfOpenMaxCalls)

	labels := prometheus.Labels{
		"name":      cb.config.Name,
		"pod_name":  cb.config.PodName,
		"namespace": cb.config.Namespace,
	}
	if isHealthy {
		circuitBreakerPodHealth.With(labels).Set(1)
	} else {
		circuitBreakerPodHealth.With(labels).Set(0)
	}

	return isHealthy
}

// String возвращает строковое представление состояния
func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}
