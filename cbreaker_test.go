package cbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/piotrpersona/cbreaker"
	"github.com/stretchr/testify/assert"
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

		cb := cbreaker.NewBreaker[int](
			cbreaker.WithThreshold(3),
			cbreaker.WithOpenTimeout(time.Second),
			cbreaker.WithRetryThreshold(1),
			cbreaker.WithStateChangeCallback(stateCallback),
		)

		res, err := cb.Try(func() (int, error) {
			return 348, nil
		})
		require.NoError(t, err)
		require.Equal(t, res, 348)

		for i := 0; i < 5; i++ {
			_, err = cb.Try(func() (int, error) {
				return 0, errors.New("error")
			})
			assert.Error(t, err)
		}
		assert.Equal(t, cb.State(), cbreaker.StateOpen)

		time.Sleep(time.Second * 2)

		_, err = cb.Try(func() (int, error) {
			return 0, errors.New("error")
		})
		assert.Error(t, err)
		require.Equal(t, cb.State(), cbreaker.StateHalfOpen)

		_, err = cb.Try(func() (int, error) {
			return 0, errors.New("error")
		})
		require.Error(t, err)
		require.Equal(t, cb.State(), cbreaker.StateOpen)
		time.Sleep(time.Second * 2)

		_, err = cb.Try(func() (int, error) {
			return 0, nil
		})
		_, err = cb.Try(func() (int, error) {
			return 0, nil
		})
		require.NoError(t, err)
		require.Equal(t, cb.State(), cbreaker.StateClosed)

		require.Equal(t, stateTransitions, [][]cbreaker.State{
			{cbreaker.StateClosed, cbreaker.StateOpen},
			{cbreaker.StateOpen, cbreaker.StateHalfOpen},
			{cbreaker.StateHalfOpen, cbreaker.StateOpen},
			{cbreaker.StateOpen, cbreaker.StateHalfOpen},
			{cbreaker.StateHalfOpen, cbreaker.StateClosed},
		})
	})

	t.Run("Test NoRetBreaker", func(t *testing.T) {
		t.Parallel()

		cb := cbreaker.NewNoRetBreaker()
		err := cb.Try(func() error {
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, cb.State(), cbreaker.StateClosed)
	})
}
