// Package baseservice contains structs and initialization functions for
// "service-like" objects that provide commonly needed facilities so that they
// don't have to be redefined on every struct. The word "service" is used quite
// loosely here in that it may be applied to many long-lived object that aren't
// strictly services (e.g. adapters).
package baseservice

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"log/slog"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"gitlab.com/navyx/ai/maos/maos-core/util"
)

// BaseService is a struct that's meant to be embedded on "service-like" objects
// (e.g. client, producer, queue maintainer) and which provides a number of
// convenient properties that are widely needed so that they don't have to be
// defined on every individual service and can easily be copied from each other.
//
// An initial Archetype should be defined near the program's entrypoint
// (currently in Client), and then each service should invoke Init along with
// the archetype to initialize its own base service. This is often done in the
// service's constructor, but if it doesn't have one, it's the job of the caller
// which instantiates it to invoke Init.
type BaseService struct {
	Logger *slog.Logger

	// Rand is a random source safe for concurrent access and seeded with a
	// cryptographically random seed to ensure good distribution between nodes
	// and services. The random source itself is _not_ cryptographically secure,
	// and therefore should not be used anywhere security-related. This is a
	// deliberate choice because Go's non-crypto rand source is about twenty
	// times faster, and so far none of our uses of random require cryptographic
	// secure randomness.
	Rand *rand.Rand

	// Name is a name of the service. It should generally be used to prefix all
	// log lines the service emits.
	Name string
}

// CancellableSleep sleeps for the given duration, but returns early if context
// has been cancelled.
func (s *BaseService) CancellableSleep(ctx context.Context, sleepDuration time.Duration) {
	timer := time.NewTimer(sleepDuration)

	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}

// CancellableSleep sleeps for the given duration, but returns early if context
// has been cancelled.
//
// This variant returns a channel that should be waited on and which will be
// closed when either the sleep or context is done.
func (s *BaseService) CancellableSleepC(ctx context.Context, sleepDuration time.Duration) <-chan struct{} {
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		s.CancellableSleep(ctx, sleepDuration)
	}()

	return doneChan
}

// CancellableSleepRandomBetween sleeps for a random duration between the given
// bounds (max bound is exclusive), but returns early if context has been
// cancelled.
func (s *BaseService) CancellableSleepRandomBetween(ctx context.Context, sleepDurationMin, sleepDurationMax time.Duration) {
	s.CancellableSleep(ctx, time.Duration(IntBetween(s.Rand, int(sleepDurationMin), int(sleepDurationMax))))
}

// CancellableSleepRandomBetween sleeps for a random duration between the given
// bounds (max bound is exclusive), but returns early if context has been
// cancelled.
//
// This variant returns a channel that should be waited on and which will be
// closed when either the sleep or context is done.
func (s *BaseService) CancellableSleepRandomBetweenC(ctx context.Context, sleepDurationMin, sleepDurationMax time.Duration) <-chan struct{} {
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		s.CancellableSleep(ctx, time.Duration(IntBetween(s.Rand, int(sleepDurationMin), int(sleepDurationMax))))
	}()

	return doneChan
}

// MaxAttemptsBeforeResetDefault is the number of attempts during exponential
// backoff after which attempts is reset so that sleep durations aren't flung
// into a ridiculously distant future. This constant is typically injected into
// the CancellableSleepExponentialBackoff function. It could technically take
// another value instead, but shouldn't unless there's a good reason to do so.
const MaxAttemptsBeforeResetDefault = 10

// ExponentialBackoff returns a duration for a reasonable exponential backoff
// interval for a service based on the given attempt number, which can then be
// fed into CancellableSleep to perform the sleep. Uses a 2**N second algorithm,
// +/- 10% random jitter. Sleep is cancelled if the given context is cancelled.
//
// Attempt should start at one for the first backoff/failure.
func (s *BaseService) ExponentialBackoff(attempt, maxAttemptsBeforeReset int) time.Duration {
	retrySeconds := s.exponentialBackoffSecondsWithoutJitter(attempt, maxAttemptsBeforeReset)

	// Jitter number of seconds +/- 10%.
	retrySeconds += retrySeconds * (s.Rand.Float64()*0.2 - 0.1)

	return util.SecondsAsDuration(retrySeconds)
}

func (s *BaseService) exponentialBackoffSecondsWithoutJitter(attempt, maxAttemptsBeforeReset int) float64 {
	// It's easier for callers and more intuitive if attempt starts at one, but
	// subtract one before sending it the exponent so we start at only one
	// second of sleep instead of two.
	attempt--

	// We use a different exponential backoff algorithm here compared to the
	// default retry policy (2**N versus N**4) because it results in more
	// retries sooner. When it comes to exponential backoffs in services we
	// never want to sleep for hours/days, unlike with failed jobs.
	return math.Pow(2, float64(attempt%maxAttemptsBeforeReset))
}

func (s *BaseService) GetBaseService() *BaseService {
	return s
}

// withBaseService is an interface to a struct that embeds BaseService. An
// implementation is provided automatically by BaseService, and it's largely
// meant for internal use.
type withBaseService interface {
	GetBaseService() *BaseService
}

// Init initializes a base service from an archetype. It returns the same
// service that was passed into it for convenience.
func Init[TService withBaseService](logger *slog.Logger, service TService) TService {
	baseService := service.GetBaseService()

	baseService.Logger = logger
	baseService.Name = reflect.TypeOf(service).Elem().Name()
	baseService.Rand = rand.New(newCryptoSeededConcurrentSafeSource())

	return service
}

// TimeGenerator generates a current time in UTC. In test environments it's
// implemented by riverinternaltest.timeStub which lets the current time be
// stubbed. Otherwise, it's implemented as UnStubbableTimeGenerator which
// doesn't allow stubbing.
type TimeGenerator interface {
	// NowUTC returns the current time. This may be a stubbed time if the time
	// has been actively stubbed in a test.
	NowUTC() time.Time

	// NowUTCOrNil returns if the currently stubbed time _if_ the current time
	// is stubbed, and returns nil otherwise. This is generally useful in cases
	// where a component may want to use a stubbed time if the time is stubbed,
	// but to fall back to a database time default otherwise.
	NowUTCOrNil() *time.Time

	// StubNowUTC stubs the current time. It will panic if invoked outside of
	// tests. Returns the same time passed as parameter for convenience.
	StubNowUTC(nowUTC time.Time) time.Time
}

// UnStubbableTimeGenerator is a TimeGenerator implementation that can't be
// stubbed. It's always the generator used outside of tests.
type UnStubbableTimeGenerator struct{}

func (g *UnStubbableTimeGenerator) NowUTC() time.Time       { return time.Now() }
func (g *UnStubbableTimeGenerator) NowUTCOrNil() *time.Time { return nil }
func (g *UnStubbableTimeGenerator) StubNowUTC(nowUTC time.Time) time.Time {
	panic("time not stubbable outside tests")
}

// IntBetween generates a random number in the range of [min, max).
func IntBetween(rand *rand.Rand, min, max int) int {
	return rand.Intn(max-min) + min
}

func newCryptoSeededConcurrentSafeSource() rand.Source {
	var seed int64

	if err := binary.Read(cryptorand.Reader, binary.BigEndian, &seed); err != nil {
		panic(err)
	}

	return &concurrentSafeSource{innerSource: rand.NewSource(seed)}
}

type concurrentSafeSource struct {
	innerSource rand.Source
	mu          sync.Mutex
}

func (s *concurrentSafeSource) Int63() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.innerSource.Int63()
}

func (s *concurrentSafeSource) Seed(seed int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.innerSource.Seed(seed)
}
