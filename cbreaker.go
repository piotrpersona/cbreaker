package cbreaker

import (
	"sync"
	"sync/atomic"
	"time"
)

// State represents circuit state.
type State uint32

const (
	// StateOpen indicates that circuit is open and will return previously cached error.
	StateOpen State = 0
	// StateClosed indicates that the circuit is closed and will call callback function.
	StateClosed State = 1
	// StateHalfOpen means that the circuit was previously open but it will allow a few requests to be called.
	StateHalfOpen State = 2
)

// String return State name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return ""
	}
}

// StateChangeCallback is a callback to acknowledge state transition of a circuit.
// For example it can be used for logging.
type StateChangeCallback func(current, new State)

// Breaker is a default circuit breaker implementation.
type Breaker[T any] struct {
	state uint32

	threshold  uint32
	currentTry uint32

	openTime time.Time

	mu         sync.RWMutex
	openResult T
	openErr    error

	openTimeout    time.Duration
	currentRetry   uint32
	retryThreshold uint32

	stateChangeCallback StateChangeCallback
}

type configuration struct {
	threshold           uint32
	openTimeout         time.Duration
	retryThreshold      uint32
	stateChangeCallback StateChangeCallback
}

// Option modifies Breaker configuration.
type Option func(*configuration)

// WithThreshold sets the threshold value after which the circuit becomes Open.
func WithThreshold(threshold uint32) Option {
	return func(c *configuration) {
		c.threshold = threshold
	}
}

// WithOpenTimeout sets timeout after which the circuit becomes HalfOpen.
func WithOpenTimeout(timeout time.Duration) Option {
	return func(c *configuration) {
		c.openTimeout = timeout
	}
}

// WithRetryThreshold sets threshold value after which the circuit becomes Open from state HalfOpen.
func WithRetryThreshold(threshold uint32) Option {
	return func(c *configuration) {
		c.retryThreshold = threshold
	}
}

// WithStateChangeCallback sets StateChangeCallback to record state transitions.
func WithStateChangeCallback(callback StateChangeCallback) Option {
	return func(c *configuration) {
		c.stateChangeCallback = callback
	}
}

// NewBreaker returns circuit breaker object.
func NewBreaker[T any](opts ...Option) *Breaker[T] {
	cfg := &configuration{
		threshold:           1,
		openTimeout:         time.Minute,
		retryThreshold:      1,
		stateChangeCallback: nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Breaker[T]{
		state:               1,
		threshold:           cfg.threshold,
		openErr:             nil,
		currentTry:          0,
		openTimeout:         cfg.openTimeout,
		currentRetry:        0,
		retryThreshold:      cfg.retryThreshold,
		stateChangeCallback: cfg.stateChangeCallback,
	}
}

// Try will call callback. If Try returns `threshold` times error the circuit becomes open.
// After `openTimeout` expires the circuit becomes half-open and will retry callback until
// success or after `retryThreshold` is reached. In case of success it will become closed, otherwise it becomes open.
func (b *Breaker[T]) Try(callback func() (T, error)) (T, error) {
	state := b.State()
	switch state {
	case StateClosed:
		result, err := callback()
		if err == nil {
			return result, nil
		}
		b.try()
		if b.shouldOpen() {
			b.openCircuit(result, err)
		}
		return result, err
	case StateOpen:
		if b.shouldHalfOpen() {
			b.halfOpenCircuit()
		}
		res, err := b.getPreviousResult()
		return res, err
	case StateHalfOpen:
		result, err := callback()
		if err == nil {
			b.closeCircuit()
			return result, nil
		}
		b.retry()
		if b.shouldOpen() {
			b.openCircuit(result, err)
		}
		return result, err
	default:
		var result T
		return result, nil
	}
}

// State returns corcuit breaker current State.
func (b *Breaker[T]) State() State {
	return State(atomic.LoadUint32(&b.state))
}

func (b *Breaker[T]) shouldOpen() bool {
	return b.State() == StateClosed && atomic.LoadUint32(&b.currentTry) == b.threshold ||
		b.State() == StateHalfOpen && atomic.LoadUint32(&b.currentRetry) == b.retryThreshold
}

func (b *Breaker[T]) try() {
	atomic.AddUint32(&b.currentTry, 1)
}

func (b *Breaker[T]) changeState(desired State) {
	current := b.State()
	atomic.StoreUint32(&b.state, uint32(desired))
	b.recordStateTransition(current, desired)
}

func (b *Breaker[T]) recordStateTransition(current, desired State) {
	if b.stateChangeCallback != nil {
		b.stateChangeCallback(current, desired)
	}
}

func (b *Breaker[T]) openCircuit(result T, err error) {
	b.changeState(StateOpen)

	b.mu.Lock()
	b.openResult = result
	b.openErr = err
	b.openTime = time.Now()
	b.mu.Unlock()
}

func (b *Breaker[T]) shouldHalfOpen() bool {
	return time.Now().After(b.openTime.Add(b.openTimeout))
}

func (b *Breaker[T]) halfOpenCircuit() {
	b.changeState(StateHalfOpen)
}

func (b *Breaker[T]) getPreviousResult() (T, error) {
	b.mu.RLock()
	res, err := b.openResult, b.openErr
	b.mu.RUnlock()
	return res, err
}

func (b *Breaker[T]) retry() {
	atomic.AddUint32(&b.currentRetry, 1)
}

func (b *Breaker[T]) closeCircuit() {
	b.mu.Lock()
	var res T
	b.openResult = res
	b.openErr = nil
	b.openTime = time.Time{}
	b.mu.Unlock()

	atomic.StoreUint32(&b.currentTry, 0)
	atomic.StoreUint32(&b.currentRetry, 0)

	b.changeState(StateClosed)
}

// NoRetBreaker is a circuit breaker that returns only an error.
type NoRetBreaker struct {
	breaker *Breaker[struct{}]
}

// NewNoRetBreaker will initialize circuit breaker with options.
func NewNoRetBreaker(opts ...Option) *NoRetBreaker {
	return &NoRetBreaker{
		breaker: NewBreaker[struct{}](opts...),
	}
}

// Try will try calling a callback. In case of any error it will work same
// as Breaker.
func (b *NoRetBreaker) Try(callback func() error) error {
	_, err := b.breaker.Try(func() (struct{}, error) {
		return struct{}{}, callback()
	})
	return err
}

// State returns current state.
func (b *NoRetBreaker) State() State {
	return b.breaker.State()
}
