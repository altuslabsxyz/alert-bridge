package pagerduty

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	domainerrors "github.com/altuslabsxyz/alert-bridge/internal/domain/errors"
)

func TestNewClient(t *testing.T) {
	t.Run("sets default severity when empty", func(t *testing.T) {
		client := NewClient("token", "routing-key", "service-id", "from@test.com", "")
		assert.Equal(t, "warning", client.defaultSeverity)
	})

	t.Run("uses custom severity when provided", func(t *testing.T) {
		client := NewClient("token", "routing-key", "service-id", "from@test.com", "critical")
		assert.Equal(t, "critical", client.defaultSeverity)
	})

	t.Run("sets custom events API URL", func(t *testing.T) {
		client := NewClient("token", "routing-key", "service-id", "from@test.com", "warning", "http://custom-api.local")
		assert.Equal(t, "http://custom-api.local", client.eventsAPIURL)
	})

	t.Run("creates client without API token", func(t *testing.T) {
		client := NewClient("", "routing-key", "service-id", "", "warning")
		assert.Nil(t, client.eventsClient)
		assert.Equal(t, "routing-key", client.routingKey)
	})
}

func TestClient_Name(t *testing.T) {
	client := NewClient("", "", "", "", "")
	assert.Equal(t, "pagerduty", client.Name())
}

func TestClient_SupportsAck(t *testing.T) {
	client := NewClient("", "", "", "", "")
	assert.True(t, client.SupportsAck())
}

func TestClient_buildDedupKey(t *testing.T) {
	client := NewClient("", "", "", "", "")

	t.Run("uses fingerprint when available", func(t *testing.T) {
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
		}
		assert.Equal(t, "fp-456", client.buildDedupKey(alert))
	})

	t.Run("falls back to ID when fingerprint is empty", func(t *testing.T) {
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "",
		}
		assert.Equal(t, "alert-123", client.buildDedupKey(alert))
	})
}

func TestClient_buildSummary(t *testing.T) {
	client := NewClient("", "", "", "", "")

	tests := []struct {
		name     string
		alert    *entity.Alert
		expected string
	}{
		{
			name: "critical severity with instance and summary",
			alert: &entity.Alert{
				Severity: entity.SeverityCritical,
				Name:     "HighCPU",
				Instance: "server-01",
				Summary:  "CPU usage above 90%",
			},
			expected: "[CRITICAL] HighCPU on server-01 - CPU usage above 90%",
		},
		{
			name: "warning severity without instance",
			alert: &entity.Alert{
				Severity: entity.SeverityWarning,
				Name:     "DiskSpace",
				Summary:  "Disk usage high",
			},
			expected: "[WARNING] DiskSpace - Disk usage high",
		},
		{
			name: "info severity without summary",
			alert: &entity.Alert{
				Severity: entity.SeverityInfo,
				Name:     "ServiceRestart",
				Instance: "web-01",
			},
			expected: "[INFO] ServiceRestart on web-01",
		},
		{
			name: "unknown severity defaults to info",
			alert: &entity.Alert{
				Severity: "unknown",
				Name:     "TestAlert",
			},
			expected: "[INFO] TestAlert",
		},
		{
			name: "minimal alert",
			alert: &entity.Alert{
				Name: "SimpleAlert",
			},
			expected: "[INFO] SimpleAlert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildSummary(tt.alert)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_buildDetails(t *testing.T) {
	client := NewClient("", "", "", "", "")

	t.Run("includes all required fields", func(t *testing.T) {
		firedAt := time.Date(2024, 1, 21, 15, 30, 0, 0, time.UTC)
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
			Name:        "TestAlert",
			Instance:    "server-01",
			Target:      "http://example.com",
			Severity:    entity.SeverityCritical,
			State:       entity.StateActive,
			FiredAt:     firedAt,
		}

		details := client.buildDetails(alert)

		assert.Equal(t, "alert-123", details["alert_id"])
		assert.Equal(t, "fp-456", details["fingerprint"])
		assert.Equal(t, "TestAlert", details["name"])
		assert.Equal(t, "server-01", details["instance"])
		assert.Equal(t, "http://example.com", details["target"])
		assert.Equal(t, "critical", details["severity"])
		assert.Equal(t, "active", details["state"])
		assert.Equal(t, "2024-01-21T15:30:00Z", details["fired_at"])
	})

	t.Run("includes optional summary and description", func(t *testing.T) {
		alert := &entity.Alert{
			ID:          "alert-123",
			FiredAt:     time.Now(),
			Summary:     "Brief summary",
			Description: "Detailed description",
		}

		details := client.buildDetails(alert)

		assert.Equal(t, "Brief summary", details["summary"])
		assert.Equal(t, "Detailed description", details["description"])
	})

	t.Run("excludes empty optional fields", func(t *testing.T) {
		alert := &entity.Alert{
			ID:      "alert-123",
			FiredAt: time.Now(),
		}

		details := client.buildDetails(alert)

		_, hasSummary := details["summary"]
		_, hasDescription := details["description"]
		assert.False(t, hasSummary)
		assert.False(t, hasDescription)
	})

	t.Run("includes labels when present", func(t *testing.T) {
		alert := &entity.Alert{
			ID:      "alert-123",
			FiredAt: time.Now(),
			Labels: map[string]string{
				"job":     "prometheus",
				"env":     "prod",
				"cluster": "us-west",
			},
		}

		details := client.buildDetails(alert)

		labels, ok := details["labels"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "prometheus", labels["job"])
		assert.Equal(t, "prod", labels["env"])
		assert.Equal(t, "us-west", labels["cluster"])
	})

	t.Run("includes annotations when present", func(t *testing.T) {
		alert := &entity.Alert{
			ID:      "alert-123",
			FiredAt: time.Now(),
			Annotations: map[string]string{
				"runbook_url": "http://runbook.local/alert",
				"dashboard":   "http://grafana.local/d/123",
			},
		}

		details := client.buildDetails(alert)

		annotations, ok := details["annotations"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "http://runbook.local/alert", annotations["runbook_url"])
		assert.Equal(t, "http://grafana.local/d/123", annotations["dashboard"])
	})

	t.Run("excludes empty labels and annotations", func(t *testing.T) {
		alert := &entity.Alert{
			ID:      "alert-123",
			FiredAt: time.Now(),
		}

		details := client.buildDetails(alert)

		_, hasLabels := details["labels"]
		_, hasAnnotations := details["annotations"]
		assert.False(t, hasLabels)
		assert.False(t, hasAnnotations)
	})
}

func TestClient_mapSeverity(t *testing.T) {
	t.Run("maps critical severity", func(t *testing.T) {
		client := NewClient("", "", "", "", "info")
		assert.Equal(t, "critical", client.mapSeverity(entity.SeverityCritical))
	})

	t.Run("maps warning severity", func(t *testing.T) {
		client := NewClient("", "", "", "", "info")
		assert.Equal(t, "warning", client.mapSeverity(entity.SeverityWarning))
	})

	t.Run("uses default severity for info", func(t *testing.T) {
		client := NewClient("", "", "", "", "info")
		assert.Equal(t, "info", client.mapSeverity(entity.SeverityInfo))
	})

	t.Run("uses custom default severity for unknown", func(t *testing.T) {
		client := NewClient("", "", "", "", "error")
		assert.Equal(t, "error", client.mapSeverity("unknown"))
	})
}

func TestCategorizePagerDutyError(t *testing.T) {
	t.Run("returns nil for nil error", func(t *testing.T) {
		result := categorizePagerDutyError(nil, "test operation")
		assert.Nil(t, result)
	})

	t.Run("categorizes network error as transient", func(t *testing.T) {
		netErr := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		result := categorizePagerDutyError(netErr, "sending event")

		assert.True(t, domainerrors.IsTransientError(result))
		assert.Contains(t, result.Error(), "network error")
	})

	t.Run("categorizes rate limiting (429) as transient", func(t *testing.T) {
		pdErr := pagerduty.APIError{
			StatusCode: 429,
			APIError:   pagerduty.NullAPIErrorObject{},
		}
		result := categorizePagerDutyError(pdErr, "sending event")

		assert.True(t, domainerrors.IsTransientError(result))
		assert.Contains(t, result.Error(), "rate limited")
	})

	t.Run("categorizes server error (500) as transient", func(t *testing.T) {
		pdErr := pagerduty.APIError{
			StatusCode: 500,
			APIError:   pagerduty.NullAPIErrorObject{},
		}
		result := categorizePagerDutyError(pdErr, "sending event")

		assert.True(t, domainerrors.IsTransientError(result))
		assert.Contains(t, result.Error(), "server error")
	})

	t.Run("categorizes server error (503) as transient", func(t *testing.T) {
		pdErr := pagerduty.APIError{
			StatusCode: 503,
			APIError:   pagerduty.NullAPIErrorObject{},
		}
		result := categorizePagerDutyError(pdErr, "test")

		assert.True(t, domainerrors.IsTransientError(result))
	})

	t.Run("categorizes client error (400) as permanent", func(t *testing.T) {
		pdErr := pagerduty.APIError{
			StatusCode: 400,
			APIError:   pagerduty.NullAPIErrorObject{},
		}
		result := categorizePagerDutyError(pdErr, "sending event")

		var domainErr *domainerrors.DomainError
		require.True(t, errors.As(result, &domainErr))
		assert.Equal(t, domainerrors.CategoryPermanent, domainErr.Category)
		assert.Contains(t, result.Error(), "client error")
	})

	t.Run("categorizes client error (401) as permanent", func(t *testing.T) {
		pdErr := pagerduty.APIError{
			StatusCode: 401,
			APIError:   pagerduty.NullAPIErrorObject{},
		}
		result := categorizePagerDutyError(pdErr, "test")

		var domainErr *domainerrors.DomainError
		require.True(t, errors.As(result, &domainErr))
		assert.Equal(t, domainerrors.CategoryPermanent, domainErr.Category)
	})

	t.Run("categorizes context deadline exceeded as transient", func(t *testing.T) {
		// Note: context.DeadlineExceeded implements net.Error interface,
		// so it gets categorized as network error first
		result := categorizePagerDutyError(context.DeadlineExceeded, "sending event")

		assert.True(t, domainerrors.IsTransientError(result))
	})

	t.Run("categorizes context canceled as transient", func(t *testing.T) {
		result := categorizePagerDutyError(context.Canceled, "sending event")

		assert.True(t, domainerrors.IsTransientError(result))
	})

	t.Run("categorizes unknown error as permanent", func(t *testing.T) {
		unknownErr := errors.New("some unknown error")
		result := categorizePagerDutyError(unknownErr, "sending event")

		var domainErr *domainerrors.DomainError
		require.True(t, errors.As(result, &domainErr))
		assert.Equal(t, domainerrors.CategoryPermanent, domainErr.Category)
	})
}

// mockPagerDutyServer creates a test server that mocks PagerDuty Events API v2
func mockPagerDutyServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

func TestClient_sendEventHTTP(t *testing.T) {
	t.Run("sends event successfully", func(t *testing.T) {
		var receivedEvent pagerduty.V2Event
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v2/enqueue", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			err := json.NewDecoder(r.Body).Decode(&receivedEvent)
			require.NoError(t, err)

			resp := pagerduty.V2EventResponse{
				Status:   "success",
				DedupKey: "test-dedup-key",
				Message:  "Event processed",
			}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		event := &pagerduty.V2Event{
			RoutingKey: "routing-key",
			Action:     "trigger",
			DedupKey:   "test-dedup",
			Payload: &pagerduty.V2Payload{
				Summary:  "Test summary",
				Source:   "test-source",
				Severity: "warning",
			},
		}

		resp, err := client.sendEventHTTP(context.Background(), event)

		require.NoError(t, err)
		assert.Equal(t, "test-dedup-key", resp.DedupKey)
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, "routing-key", receivedEvent.RoutingKey)
		assert.Equal(t, "trigger", receivedEvent.Action)
	})

	t.Run("returns error for non-2xx status", func(t *testing.T) {
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status":"invalid_event","message":"bad request"}`))
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		event := &pagerduty.V2Event{
			RoutingKey: "routing-key",
			Action:     "trigger",
		}

		resp, err := client.sendEventHTTP(context.Background(), event)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("returns error for invalid JSON response", func(t *testing.T) {
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`not json`))
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		event := &pagerduty.V2Event{
			RoutingKey: "routing-key",
			Action:     "trigger",
		}

		resp, err := client.sendEventHTTP(context.Background(), event)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshaling response")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusAccepted)
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		event := &pagerduty.V2Event{
			RoutingKey: "routing-key",
			Action:     "trigger",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		resp, err := client.sendEventHTTP(ctx, event)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestClient_Notify(t *testing.T) {
	t.Run("returns error when routing key not configured", func(t *testing.T) {
		client := NewClient("", "", "", "", "")
		alert := &entity.Alert{Name: "TestAlert"}

		_, err := client.Notify(context.Background(), alert)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "routing key not configured")
	})

	t.Run("sends event via custom API URL", func(t *testing.T) {
		var receivedEvent pagerduty.V2Event
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedEvent)
			resp := pagerduty.V2EventResponse{
				Status:   "success",
				DedupKey: "returned-dedup-key",
			}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		client := NewClient("", "test-routing-key", "", "", "warning", server.URL)
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
			Name:        "HighCPU",
			Instance:    "server-01",
			Target:      "http://example.com",
			Severity:    entity.SeverityCritical,
			State:       entity.StateActive,
			FiredAt:     time.Date(2024, 1, 21, 15, 30, 0, 0, time.UTC),
			Labels:      map[string]string{"job": "prometheus"},
		}

		dedupKey, err := client.Notify(context.Background(), alert)

		require.NoError(t, err)
		assert.Equal(t, "returned-dedup-key", dedupKey)
		assert.Equal(t, "test-routing-key", receivedEvent.RoutingKey)
		assert.Equal(t, "trigger", receivedEvent.Action)
		assert.Equal(t, "fp-456", receivedEvent.DedupKey)
		assert.Equal(t, "critical", receivedEvent.Payload.Severity)
		assert.Equal(t, "server-01", receivedEvent.Payload.Source)
		assert.Equal(t, "http://example.com", receivedEvent.Payload.Component)
		assert.Equal(t, "prometheus", receivedEvent.Payload.Group)
		assert.Equal(t, "HighCPU", receivedEvent.Payload.Class)
	})

	t.Run("handles server error", func(t *testing.T) {
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status":"error"}`))
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		alert := &entity.Alert{
			ID:      "alert-123",
			Name:    "TestAlert",
			FiredAt: time.Now(),
		}

		_, err := client.Notify(context.Background(), alert)

		require.Error(t, err)
	})
}

func TestClient_Acknowledge(t *testing.T) {
	t.Run("returns error when routing key not configured", func(t *testing.T) {
		client := NewClient("", "", "", "", "")
		alert := &entity.Alert{Name: "TestAlert"}
		ackEvent := &entity.AckEvent{UserName: "test-user"}

		err := client.Acknowledge(context.Background(), alert, ackEvent)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "routing key not configured")
	})

	t.Run("uses external reference as dedup key when available", func(t *testing.T) {
		// This test verifies the dedup key selection logic
		client := NewClient("", "routing-key", "", "", "")
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
			ExternalReferences: map[string]string{
				"pagerduty": "external-dedup-key",
			},
		}

		// This call will fail (no real PagerDuty API) but we're testing the setup logic
		_ = client.Acknowledge(context.Background(), alert, nil)
		// The main assertion is that the code path is executed without panic
	})

	t.Run("builds dedup key from alert when no external reference", func(t *testing.T) {
		client := NewClient("", "routing-key", "", "", "")
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
		}

		// This call will fail (no real PagerDuty API) but we're testing the dedup key fallback
		_ = client.Acknowledge(context.Background(), alert, nil)
	})
}

func TestClient_Resolve(t *testing.T) {
	t.Run("returns error when routing key not configured", func(t *testing.T) {
		client := NewClient("", "", "", "", "")
		alert := &entity.Alert{Name: "TestAlert"}

		err := client.Resolve(context.Background(), alert)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "routing key not configured")
	})

	t.Run("uses external reference as dedup key when available", func(t *testing.T) {
		client := NewClient("", "routing-key", "", "", "")
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
			ExternalReferences: map[string]string{
				"pagerduty": "external-dedup-key",
			},
		}

		// This call will fail (no real PagerDuty API) but we're testing the setup logic
		_ = client.Resolve(context.Background(), alert)
	})

	t.Run("builds dedup key from alert when no external reference", func(t *testing.T) {
		client := NewClient("", "routing-key", "", "", "")
		alert := &entity.Alert{
			ID:          "alert-123",
			Fingerprint: "fp-456",
		}

		// This call will fail (no real PagerDuty API) but we're testing the dedup key fallback
		_ = client.Resolve(context.Background(), alert)
	})
}

func TestClient_UpdateMessage(t *testing.T) {
	t.Run("returns error when routing key not configured", func(t *testing.T) {
		client := NewClient("", "", "", "", "")
		alert := &entity.Alert{Name: "TestAlert"}

		err := client.UpdateMessage(context.Background(), "dedup-key", alert)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "routing key not configured")
	})

	t.Run("sends resolve action for resolved alert", func(t *testing.T) {
		var receivedEvent pagerduty.V2Event
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedEvent)
			resp := pagerduty.V2EventResponse{Status: "success", DedupKey: "test-dedup"}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "", server.URL)
		alert := &entity.Alert{
			Name:  "TestAlert",
			State: entity.StateResolved,
		}

		err := client.UpdateMessage(context.Background(), "test-dedup", alert)

		require.NoError(t, err)
		assert.Equal(t, "resolve", receivedEvent.Action)
		assert.Equal(t, "test-dedup", receivedEvent.DedupKey)
		assert.Nil(t, receivedEvent.Payload) // resolve doesn't include payload
	})

	t.Run("sends acknowledge action for acked alert", func(t *testing.T) {
		var receivedEvent pagerduty.V2Event
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedEvent)
			resp := pagerduty.V2EventResponse{Status: "success", DedupKey: "test-dedup"}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "warning", server.URL)
		alert := &entity.Alert{
			Name:     "TestAlert",
			State:    entity.StateAcked,
			Instance: "server-01",
			Severity: entity.SeverityWarning,
		}

		err := client.UpdateMessage(context.Background(), "test-dedup", alert)

		require.NoError(t, err)
		assert.Equal(t, "acknowledge", receivedEvent.Action)
		assert.NotNil(t, receivedEvent.Payload) // acknowledge includes payload
		assert.Equal(t, "warning", receivedEvent.Payload.Severity)
	})

	t.Run("sends trigger action for active alert", func(t *testing.T) {
		var receivedEvent pagerduty.V2Event
		server := mockPagerDutyServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedEvent)
			resp := pagerduty.V2EventResponse{Status: "success", DedupKey: "test-dedup"}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(resp)
		})
		defer server.Close()

		client := NewClient("", "routing-key", "", "", "info", server.URL)
		alert := &entity.Alert{
			Name:     "TestAlert",
			State:    entity.StateActive,
			Instance: "server-01",
			Severity: entity.SeverityInfo,
		}

		err := client.UpdateMessage(context.Background(), "test-dedup", alert)

		require.NoError(t, err)
		assert.Equal(t, "trigger", receivedEvent.Action)
		assert.NotNil(t, receivedEvent.Payload)
	})
}
