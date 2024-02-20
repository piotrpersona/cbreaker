# cbreaker

[![Go Reference](https://pkg.go.dev/badge/github.com/piotrpersona/cbreaker.svg)](https://pkg.go.dev/github.com/piotrpersona/cbreaker)
![Tests passing](https://github.com/piotrpersona/cbreaker/actions/workflows/test.yml/badge.svg)
![Lint passing](https://github.com/piotrpersona/cbreaker/actions/workflows/lint.yml/badge.svg)


Actively maintained implementation of circuit breaker in Golang with generics support.

## Install

```sh
go get github.com/piotrpersona/cbreaker
```

## Usage

```go
cb := cbreaker.NewBreaker[int]()

res, err := cb.Try(func() (int, error) {
    // call
    return 123, nil
})
```

If result is not needed:
```go
cb := cbreaker.NewNoRetBreaker()

err := cb.Try(func() error {
    // call
    return nil
})
```

## Configure

> Note: Circuit breaker object won't automatically retry in half-open state.

```go
breaker := cbreaker.NewBreaker[int](
    // sets thershold after which the circuit becomes open
    cbreaker.WithThreshold(3),
    // sets timeout after which the circuit become half-open
    cbreaker.WithOpenTimeout(time.Second),
    // sets maximum number of retries in half-open state
    cbreaker.WithRetryThreshold(1),
    // registers stateChangeCallback
    cbreaker.WithStateChangeCallback(func(current, newState cbreaker.State) {
        log.Printf("state transition: %s -> %s", current, newState)
    }),
)
```

