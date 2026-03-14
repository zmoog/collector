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

// DeviceInfo contains per-channel device metadata from the Shelly Cloud device list.
// Multi-channel devices (e.g. S4PL-00416EU) appear as separate entries per channel,
// each with its own name and room, so this struct represents one channel.
type DeviceInfo struct {
	// ID is the channel-scoped identifier, e.g. "98a3167ba5d8_1" for channel 1.
	ID string
	// BaseID is the physical device ID used for status API calls, e.g. "98a3167ba5d8".
	BaseID        string
	Name          string
	Type          string
	Gen           int
	Channel       int
	ChannelsCount int
	RoomID        int
	CloudOnline   bool
}

// Room contains room metadata.
type Room struct {
	ID   int
	Name string
}

// DeviceStatus holds the parsed metrics for a single physical device,
// normalised across Gen1 and Gen2+ formats.
type DeviceStatus struct {
	// Gen2+: one entry per switch component, keyed by channel index ("0", "1", …)
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

type deviceListResponse struct {
	IsOk bool           `json:"isok"`
	Data deviceListData `json:"data"`
}

type deviceListData struct {
	Devices map[string]deviceEntry `json:"devices"`
}

type deviceEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Gen           int    `json:"gen"`
	Channel       int    `json:"channel"`
	ChannelsCount int    `json:"channels_count"`
	RoomID        int    `json:"room_id"`
	CloudOnline   bool   `json:"cloud_online"`
}

type roomListResponse struct {
	IsOk bool         `json:"isok"`
	Data roomListData `json:"data"`
}

type roomListData struct {
	Rooms map[string]roomEntry `json:"rooms"`
}

type roomEntry struct {
	Name string `json:"name"`
}

type deviceStatusResponse struct {
	IsOk   bool             `json:"isok"`
	Errors json.RawMessage  `json:"errors"`
	Data   deviceStatusData `json:"data"`
}

type deviceStatusData struct {
	Online       bool                       `json:"online"`
	DeviceStatus map[string]json.RawMessage `json:"device_status"`
}

// ListDevices returns all channel entries from the Shelly Cloud account
// together with a room ID→Room map for name resolution.
// The room map may be empty if the rooms endpoint returns an error.
func (c *Client) ListDevices() ([]DeviceInfo, map[int]Room, error) {
	body, err := c.get("/interface/device/list", url.Values{"auth_key": {c.authKey}})
	if err != nil {
		return nil, nil, err
	}

	var dlr deviceListResponse
	if err := json.Unmarshal(body, &dlr); err != nil {
		return nil, nil, fmt.Errorf("parse device list: %w", err)
	}
	if !dlr.IsOk {
		return nil, nil, fmt.Errorf("Shelly Cloud API error on device list")
	}

	rooms, err := c.listRooms()
	if err != nil {
		// Non-fatal: proceed without room names.
		rooms = map[int]Room{}
	}

	devices := make([]DeviceInfo, 0, len(dlr.Data.Devices))
	for _, entry := range dlr.Data.Devices {
		devices = append(devices, DeviceInfo{
			ID:            entry.ID,
			BaseID:        baseDeviceID(entry.ID),
			Name:          entry.Name,
			Type:          entry.Type,
			Gen:           entry.Gen,
			Channel:       entry.Channel,
			ChannelsCount: entry.ChannelsCount,
			RoomID:        entry.RoomID,
			CloudOnline:   entry.CloudOnline,
		})
	}

	return devices, rooms, nil
}

func (c *Client) listRooms() (map[int]Room, error) {
	body, err := c.get("/interface/room/list", url.Values{"auth_key": {c.authKey}})
	if err != nil {
		return nil, err
	}

	var rlr roomListResponse
	if err := json.Unmarshal(body, &rlr); err != nil {
		return nil, fmt.Errorf("parse room list: %w", err)
	}
	if !rlr.IsOk {
		return nil, fmt.Errorf("Shelly Cloud API error on room list")
	}

	rooms := make(map[int]Room, len(rlr.Data.Rooms))
	for idStr, entry := range rlr.Data.Rooms {
		id, _ := strconv.Atoi(idStr)
		rooms[id] = Room{ID: id, Name: entry.Name}
	}

	return rooms, nil
}

// GetDeviceStatus fetches and parses the current status for a physical device ID.
func (c *Client) GetDeviceStatus(deviceID string) (*DeviceStatus, error) {
	body, err := c.get("/device/status", url.Values{
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
		return nil, fmt.Errorf("Shelly Cloud API error on device status for %s: %s", deviceID, string(dsr.Errors))
	}
	if !dsr.Data.Online {
		return nil, nil
	}

	return parseDeviceStatus(dsr.Data.DeviceStatus)
}

// parseDeviceStatus detects Gen1 vs Gen2+ from the raw JSON keys.
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
			status.Switches[strings.TrimPrefix(key, "switch:")] = sw

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
				// Some Gen1 models nest this; ignore parse failures.
				status.Temperature = 0
			}
		}
	}

	return status, nil
}

// baseDeviceID strips the channel suffix from a device ID.
// "98a3167ba5d8_1" → "98a3167ba5d8", "80646f83ea3b" → "80646f83ea3b".
func baseDeviceID(id string) string {
	if idx := strings.LastIndex(id, "_"); idx != -1 {
		if _, err := strconv.Atoi(id[idx+1:]); err == nil {
			return id[:idx]
		}
	}
	return id
}

func (c *Client) get(path string, params url.Values) ([]byte, error) {
	u := c.serverURL + path + "?" + params.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
