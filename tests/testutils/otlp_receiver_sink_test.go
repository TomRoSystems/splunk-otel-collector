// Copyright Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.uber.org/zap"
)

func TestNewOTLPReceiverSink(t *testing.T) {
	otlp := NewOTLPMetricsReceiverSink()
	require.NotNil(t, otlp)

	require.Empty(t, otlp.Endpoint)
	require.Nil(t, otlp.Host)
	require.Nil(t, otlp.Logger)
	require.Nil(t, otlp.receiver)
	require.Nil(t, otlp.sink)
}

func TestBuilderMethods(t *testing.T) {
	otlp := NewOTLPMetricsReceiverSink()

	withEndpoint := otlp.WithEndpoint("myendpoint")
	require.Equal(t, "myendpoint", withEndpoint.Endpoint)
	require.Empty(t, otlp.Endpoint)

	host := componenttest.NewNopHost()
	withHost := otlp.WithHost(host)
	require.Same(t, host, withHost.Host)
	require.Nil(t, otlp.Host)

	logger := zap.NewNop()
	withLogger := otlp.WithLogger(logger)
	require.Same(t, logger, withLogger.Logger)
	require.Nil(t, otlp.Logger)
}

func TestBuildDefaults(t *testing.T) {
	otlp, err := NewOTLPMetricsReceiverSink().Build()
	require.Error(t, err)
	assert.EqualError(t, err, "must provide an Endpoint for OTLPMetricsReceiverSink")
	assert.Nil(t, otlp)

	otlp, err = NewOTLPMetricsReceiverSink().WithEndpoint("myEndpoint").Build()
	require.NoError(t, err)
	assert.Equal(t, "myEndpoint", otlp.Endpoint)
	assert.NotNil(t, otlp.Host)
	assert.NotNil(t, otlp.Logger)
	assert.NotNil(t, otlp.receiver)
	assert.NotNil(t, otlp.sink)
}

func TestReceiverMethodsWithoutBuildingDisallowed(t *testing.T) {
	otlp := NewOTLPMetricsReceiverSink()

	err := otlp.Start()
	require.Error(t, err)
	require.EqualError(t, err, "cannot invoke Start() on an OTLPMetricsReceiverSink that hasn't been built")

	err = otlp.Shutdown()
	require.Error(t, err)
	require.EqualError(t, err, "cannot invoke Shutdown() on an OTLPMetricsReceiverSink that hasn't been built")

	metrics := otlp.AllMetrics()
	require.Nil(t, metrics)

	count := otlp.DataPointCount()
	require.Zero(t, count)

	// doesn't panic
	otlp.Reset()

	err = otlp.AssertAllMetricsReceived(t, ResourceMetrics{}, 0)
	require.Error(t, err)
	require.EqualError(t, err, "cannot invoke AssertAllMetricsReceived() on an OTLPMetricsReceiverSink that hasn't been built")
}

func otlpExporter(t *testing.T) component.MetricsExporter {
	exporterCfg := otlpexporter.Config{
		ExporterSettings: config.NewExporterSettings(config.NewIDWithName("otlp", "otlp")),
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: "localhost:4317",
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		}}
	otlpExporterFactory := otlpexporter.NewFactory()
	ctx := context.Background()
	createParams := component.ExporterCreateSettings{Logger: zap.NewNop()}
	exporter, err := otlpExporterFactory.CreateMetricsExporter(ctx, createParams, &exporterCfg)
	require.NoError(t, err)
	require.NotNil(t, exporter)
	err = exporter.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)
	return exporter
}

func TestOTLPReceiverMetricsAvailableToSink(t *testing.T) {
	otlp, err := NewOTLPMetricsReceiverSink().WithEndpoint("localhost:4317").Build()
	require.NoError(t, err)

	err = otlp.Start()
	defer func() {
		require.NoError(t, otlp.Shutdown())
	}()
	require.NoError(t, err)

	exporter := otlpExporter(t)
	defer exporter.Shutdown(context.Background())

	metrics := pdataMetrics()
	expectedCount := metrics.DataPointCount()
	err = exporter.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		return otlp.DataPointCount() == expectedCount
	}, 5*time.Second, 1*time.Millisecond)
}

func TestAssertAllMetricsReceivedHappyPath(t *testing.T) {
	otlp, err := NewOTLPMetricsReceiverSink().WithEndpoint("localhost:4317").Build()
	require.NoError(t, err)

	err = otlp.Start()
	defer func() {
		require.NoError(t, otlp.Shutdown())
	}()
	require.NoError(t, err)

	exporter := otlpExporter(t)
	defer exporter.Shutdown(context.Background())

	metrics := pdataMetrics()
	err = exporter.ConsumeMetrics(context.Background(), metrics)
	require.NoError(t, err)

	resourceMetrics, err := PDataToResourceMetrics(metrics)
	require.NoError(t, err)
	otlp.AssertAllMetricsReceived(t, resourceMetrics, 100*time.Millisecond)
}
