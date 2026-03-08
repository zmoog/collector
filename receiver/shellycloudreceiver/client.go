package shellycloudreceiver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client is the Shelly Cloud API client.
type Client struct {
	httpClient *http.Client
	serverURL  string
	authKey    string
}

func newClient(serverURL, authKey string) *Client {
	return &Client{
		httpClient: &http.Client{},
		serverURL:  strings.TrimRight(serverURL, "/"),
		authKey:    authKey,
	}
}

// DeviceInfo contains basic device metadata from the Shelly Cloud.
type DeviceInfo struct {
	ID     string
	Name   string
	Type   string
	RoomID int
	Online bool
}

// Room contains room metadata.
type Room struct {
	ID   int
	Name string
}

// DeviceStatus holds the parsed metrics for a single device,
// normalised across Gen1 and Gen2 formats.
type DeviceStatus struct {
	// Gen2: one entry per switch component, keyed by channel index ("0", "1", …)
	Switches map[string]SwitchStatus
	// Gen1: one entry per meter channel
	Meters []Gen1Meter
	// Gen1: one entry per relay channel
	Relays []Gen1Relay
	// Gen1: device-level temperature (°C)
	Temperature float64
}

type SwitchStatus struct {
	Output      bool        `json:"output"`
	APower      float64     `json:"apower"`
	Voltage     float64     `json:"voltage"`
	Freq        float64     `json:"freq"`
	Current     float64     `json:"current"`
	AEnergy     AEnergy     `json:"aenergy"`
	Temperature Temperature `json:"temperature"`
}

type AEnergy struct {
	Total float64 `json:"total"` // Wh
}

type Temperature struct {
	TC float64 `json:"tC"`
}

type Gen1Meter struct {
	Power   float64 `json:"power"`    // W
	IsValid bool    `json:"is_valid"`
	Total   float64 `json:"total"` // Wh
}

type Gen1Relay struct {
	IsOn bool `json:"ison"`
}

// --- internal API response types ---

type deviceCollectionResponse struct {
	IsOk bool                 `json:"isok"`
	Data deviceCollectionData `json:"data"`
}

type deviceCollectionData struct {
	DevicesStatus map[string]deviceEntry `json:"devices_status"`
}

type deviceEntry struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	RoomID int    `json:"room_id"`
	Online bool   `json:"online"`
}

type roomCollectionResponse struct {
	IsOk bool             `json:"isok"`
	Data roomCollectionData `json:"data"`
}

type roomCollectionData struct {
	Rooms map[string]roomEntry `json:"rooms"`
}

type roomEntry struct {
	Name string `json:"name"`
}

type deviceStatusResponse struct {
	IsOk bool             `json:"isok"`
	Data deviceStatusData `json:"data"`
}

type deviceStatusData struct {
	DeviceStatus map[string]json.RawMessage `json:"device_status"`
}

// ListDevices returns all devices registered in the Shelly Cloud account
// together with a room ID→Room map for name resolution.
func (c *Client) ListDevices() ([]DeviceInfo, map[int]Room, error) {
	body, err := c.post("/interface/device/collection", url.Values{"auth_key": {c.authKey}})
	if err != nil {
		return nil, nil, err
	}

	var dcr deviceCollectionResponse
	if err := json.Unmarshal(body, &dcr); err != nil {
		return nil, nil, fmt.Errorf("parse device collection: %w", err)
	}
	if !dcr.IsOk {
		return nil, nil, fmt.Errorf("Shelly Cloud API error on device collection")
	}

	rooms, err := c.listRooms()
	if err != nil {
		return nil, nil, err
	}

	devices := make([]DeviceInfo, 0, len(dcr.Data.DevicesStatus))
	for id, entry := range dcr.Data.DevicesStatus {
		devices = append(devices, DeviceInfo{
			ID:     id,
			Name:   entry.Name,
			Type:   entry.Type,
			RoomID: entry.RoomID,
			Online: entry.Online,
		})
	}

	return devices, rooms, nil
}

func (c *Client) listRooms() (map[int]Room, error) {
	body, err := c.post("/interface/room/collection", url.Values{"auth_key": {c.authKey}})
	if err != nil {
		return nil, err
	}

	var rcr roomCollectionResponse
	if err := json.Unmarshal(body, &rcr); err != nil {
		return nil, fmt.Errorf("parse room collection: %w", err)
	}
	if !rcr.IsOk {
		return nil, fmt.Errorf("Shelly Cloud API error on room collection")
	}

	rooms := make(map[int]Room, len(rcr.Data.Rooms))
	for idStr, entry := range rcr.Data.Rooms {
		id, _ := strconv.Atoi(idStr)
		rooms[id] = Room{ID: id, Name: entry.Name}
	}

	return rooms, nil
}

// GetDeviceStatus fetches and parses the current status for the given device ID.
func (c *Client) GetDeviceStatus(deviceID string) (*DeviceStatus, error) {
	body, err := c.post("/device/status", url.Values{
		"auth_key": {c.authKey},
		"id":       {deviceID},
	})
	if err != nil {
		return nil, err
	}

	var dsr deviceStatusResponse
	if err := json.Unmarshal(body, &dsr); err != nil {
		return nil, fmt.Errorf("parse device status: %w", err)
	}
	if !dsr.IsOk {
		return nil, fmt.Errorf("Shelly Cloud API error on device status for %s", deviceID)
	}

	return parseDeviceStatus(dsr.Data.DeviceStatus)
}

// parseDeviceStatus detects Gen1 vs Gen2 from the raw JSON keys and
// returns a unified DeviceStatus.
func parseDeviceStatus(raw map[string]json.RawMessage) (*DeviceStatus, error) {
	status := &DeviceStatus{
		Switches: make(map[string]SwitchStatus),
	}

	for key, value := range raw {
		switch {
		case strings.HasPrefix(key, "switch:"):
			var sw SwitchStatus
			if err := json.Unmarshal(value, &sw); err != nil {
				return nil, fmt.Errorf("parse switch %s: %w", key, err)
			}
			channel := strings.TrimPrefix(key, "switch:")
			status.Switches[channel] = sw

		case key == "meters":
			if err := json.Unmarshal(value, &status.Meters); err != nil {
				return nil, fmt.Errorf("parse Gen1 meters: %w", err)
			}

		case key == "relays":
			if err := json.Unmarshal(value, &status.Relays); err != nil {
				return nil, fmt.Errorf("parse Gen1 relays: %w", err)
			}

		case key == "temperature":
			// Gen1 reports device temperature as a bare float.
			if err := json.Unmarshal(value, &status.Temperature); err != nil {
				// Some Gen1 models use an object; ignore parse failures here.
				status.Temperature = 0
			}
		}
	}

	return status, nil
}

func (c *Client) post(path string, data url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.serverURL+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
