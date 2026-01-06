package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestConfig(t *testing.T) {
	t.Run("zero value config", func(t *testing.T) {
		cfg := Config{}
		assert.False(t, cfg.Enabled)
		assert.Empty(t, cfg.Endpoint)
		assert.Zero(t, cfg.SampleRate)
		assert.False(t, cfg.Insecure)
	})

	t.Run("configured config", func(t *testing.T) {
		cfg := Config{
			Enabled:    true,
			Endpoint:   "localhost:4318",
			SampleRate: 0.5,
			Insecure:   true,
		}
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "localhost:4318", cfg.Endpoint)
		assert.Equal(t, 0.5, cfg.SampleRate)
		assert.True(t, cfg.Insecure)
	})
}

func TestNewProvider_Disabled(t *testing.T) {
	cfg := Config{Enabled: false}

	provider, err := NewProvider(context.Background(), cfg, "test-service", "1.0.0")
	assert.NoError(t, err)
	assert.Nil(t, provider)
}

func TestProvider_Shutdown_Nil(t *testing.T) {
	var provider *Provider
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestProvider_Shutdown_NilTP(t *testing.T) {
	provider := &Provider{tp: nil}
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestProvider_Tracer_Nil(t *testing.T) {
	var provider *Provider
	tracer := provider.Tracer("test")
	require.NotNil(t, tracer)
}

func TestProvider_Tracer_NilTP(t *testing.T) {
	provider := &Provider{tp: nil}
	tracer := provider.Tracer("test")
	require.NotNil(t, tracer)
}

func TestTracer(t *testing.T) {
	tracer := Tracer("test-tracer")
	require.NotNil(t, tracer)
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)
	require.NotNil(t, span)
	// Background context should return a non-recording span
	assert.False(t, span.IsRecording())
}

func TestSpanFromContext_WithSpan(t *testing.T) {
	ctx := context.Background()
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	retrievedSpan := SpanFromContext(ctx)
	require.NotNil(t, retrievedSpan)
	assert.Equal(t, span.SpanContext().SpanID(), retrievedSpan.SpanContext().SpanID())
}
