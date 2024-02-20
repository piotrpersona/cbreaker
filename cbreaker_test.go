package cbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/piotrpersona/cbreaker"
	"github.com/stretchr/testify/require"
)

func TestCbreaker(t *testing.T) {
	t.Parallel()

	t.Run("state transitions", func(t *testing.T) {
		t.Parallel()

		stateTransitions := make([][]cbreaker.State, 0)
		stateCallback := func(current, newState cbreaker.State) {
			stateTransitions = append(stateTransitions, []cbreaker.State{current, newState})
			t.Logf("state transition %s -> %s", current, newState)
		}

		breaker := cbreaker.NewBreaker[int](
			cbreaker.WithThreshold(3),
			cbreaker.WithOpenTimeout(time.Second),
			cbreaker.WithRetryThreshold(1),
			cbreaker.WithStateChangeCallback(stateCallback),
		)

		res, err := breaker.Try(func() (int, error) {
			return 348, nil
		})
		require.NoError(t, err)
		require.Equal(t, 348, res)

		for i := 0; i < 5; i++ {
			_, err = breaker.Try(func() (int, error) {
				return 0, errors.New("error")
			})
			require.Error(t, err)
		}
		require.Equal(t, cbreaker.StateOpen, breaker.State())

		time.Sleep(time.Second * 2)

		_, err = breaker.Try(func() (int, error) {
			return 0, errors.New("error")
		})
		require.Error(t, err)
		require.Equal(t, cbreaker.StateHalfOpen, breaker.State())

		_, err = breaker.Try(func() (int, error) {
			return 0, errors.New("error")
		})
		require.Error(t, err)
		require.Equal(t, cbreaker.StateOpen, breaker.State())
		time.Sleep(time.Second * 2)

		_, _ = breaker.Try(func() (int, error) {
			return 0, nil
		})
		_, err = breaker.Try(func() (int, error) {
			return 0, nil
		})
		require.NoError(t, err)
		require.Equal(t, cbreaker.StateClosed, breaker.State())

		require.Equal(t, [][]cbreaker.State{
			{cbreaker.StateClosed, cbreaker.StateOpen},
			{cbreaker.StateOpen, cbreaker.StateHalfOpen},
			{cbreaker.StateHalfOpen, cbreaker.StateOpen},
			{cbreaker.StateOpen, cbreaker.StateHalfOpen},
			{cbreaker.StateHalfOpen, cbreaker.StateClosed},
		}, stateTransitions)
	})

	t.Run("Test NoRetBreaker", func(t *testing.T) {
		t.Parallel()

		breaker := cbreaker.NewNoRetBreaker()
		err := breaker.Try(func() error {
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, cbreaker.StateClosed, breaker.State())
	})
}
