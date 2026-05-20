package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client sends requests to the backend bot API.
// Responsibilities: build request, send, decode response, map status codes to typed errors.
// No business logic, no retries, no circuit breaking.
type Client struct {
	baseURL   string
	botSecret string
	http      *http.Client
}

// New creates a Client with a 10-second request timeout.
func New(baseURL, botSecret string) *Client {
	return &Client{
		baseURL:   baseURL,
		botSecret: botSecret,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// GetDirections returns all active directions.
// NOTE: requires backend route GET /api/v1/bot/directions (bot auth).
func (c *Client) GetDirections(ctx context.Context) ([]Direction, error) {
	var out []Direction
	if err := c.get(ctx, "/api/v1/bot/directions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDoctors returns active doctors filtered by direction.
// NOTE: requires backend route GET /api/v1/bot/doctors?direction_id=X (bot auth + direction filter).
func (c *Client) GetDoctors(ctx context.Context, directionID int64) ([]Doctor, error) {
	params := url.Values{"direction_id": {strconv.FormatInt(directionID, 10)}}
	var out []Doctor
	if err := c.get(ctx, "/api/v1/bot/doctors", params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetServicesByDoctor returns active services for a given doctor.
// NOTE: requires backend route GET /api/v1/bot/doctors/{id}/services (bot auth).
func (c *Client) GetServicesByDoctor(ctx context.Context, doctorID int64) ([]Service, error) {
	path := fmt.Sprintf("/api/v1/bot/doctors/%d/services", doctorID)
	var out []Service
	if err := c.get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAvailability returns available slots for a doctor+service in a date range.
func (c *Client) GetAvailability(ctx context.Context, doctorID, serviceID int64, dateFrom, dateTo string) (*AvailabilityResponse, error) {
	params := url.Values{
		"doctor_id":  {strconv.FormatInt(doctorID, 10)},
		"service_id": {strconv.FormatInt(serviceID, 10)},
		"date_from":  {dateFrom},
		"date_to":    {dateTo},
	}
	var out AvailabilityResponse
	if err := c.get(ctx, "/api/v1/bot/availability", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateAppointment books an appointment via the bot endpoint.
// Returns ErrSlotTaken on 409 — caller is responsible for resetting session.
func (c *Client) CreateAppointment(ctx context.Context, input CreateAppointmentInput) (*AppointmentResult, error) {
	var out AppointmentResult
	if err := c.post(ctx, "/api/v1/bot/appointments", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelAppointment cancels an existing appointment.
func (c *Client) CancelAppointment(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v1/bot/appointments/%d/cancel", id)
	return c.post(ctx, path, nil, nil)
}

// — internal helpers —

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ErrTemporary
	}
	req.Header.Set("X-Bot-Token", c.botSecret)
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return ErrTemporary
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return ErrTemporary
	}
	req.Header.Set("X-Bot-Token", c.botSecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, out)
}

// do executes one request and maps the HTTP response to a typed error.
// No retry logic — a single attempt only.
func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		// network error, context timeout, TLS failure
		return ErrTemporary
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if out == nil {
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return ErrTemporary
		}
		return nil
	case http.StatusNoContent:
		return nil
	case http.StatusConflict: // 409 — slot already taken
		return ErrSlotTaken
	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403 — bad BOT_API_SECRET
		return ErrUnauthorized
	case http.StatusNotFound: // 404
		return ErrNotFound
	default: // 4xx (other), 5xx
		return ErrTemporary
	}
}
