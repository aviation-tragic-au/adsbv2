package adsbv2

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallsign(t *testing.T) {
	for flight, want := range map[string]string{
		"QFA615  ": "QFA615",
		"vep":      "VEP",
		"@@@@@@@@": "",
		"QFA6@@@@": "QFA6",
		"":         "",
	} {
		assert.Equal(t, want, (&Aircraft{Flight: flight}).Callsign(), flight)
	}
}

func TestSelectedAltitude(t *testing.T) {
	mcp, fms := 5000, 11000
	assert.Equal(t, &mcp, (&Aircraft{NavAltitudeMCP: &mcp, NavAltitudeFMS: &fms}).SelectedAltitude())
	assert.Equal(t, &fms, (&Aircraft{NavAltitudeFMS: &fms}).SelectedAltitude())
	assert.Nil(t, (&Aircraft{}).SelectedAltitude())
}

func TestVerticalRate(t *testing.T) {
	baro, geom := -800, -768
	assert.Equal(t, &baro, (&Aircraft{BaroRate: &baro, GeomRate: &geom}).VerticalRate())
	assert.Equal(t, &geom, (&Aircraft{GeomRate: &geom}).VerticalRate())
	assert.Nil(t, (&Aircraft{}).VerticalRate())
}

func TestAltitude(t *testing.T) {
	tests := []struct {
		raw      string
		feet     *int
		onGround bool
	}{
		{raw: `"ground"`, onGround: true},
		{raw: `37000`, feet: ptr(37000)},
		{raw: `1250.0`, feet: ptr(1250)},
		{raw: `null`},
		{raw: ``},
		{raw: `"something"`},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			feet, onGround := (&Aircraft{AltBaro: json.RawMessage(tt.raw)}).Altitude()
			assert.Equal(t, tt.feet, feet)
			assert.Equal(t, tt.onGround, onGround)
		})
	}
}

// TestDecodeAircraft decodes a real-shaped response: one airborne aircraft
// with most fields, and one on the ground, whose alt_baro is a string.
func TestDecodeAircraft(t *testing.T) {
	body := `{"ac":[
		{"hex":"7c6b2d","type":"adsb_icao","flight":"QFA615  ","r":"VH-VZS","t":"B738",
		 "alt_baro":37000,"alt_geom":37600,"gs":452.1,"ias":270,"tas":462,"mach":0.788,
		 "wd":265,"ws":48,"oat":-52,"tat":-25,"track":205.3,"roll":-0.5,"mag_heading":198.1,
		 "baro_rate":0,"squawk":"3041","emergency":"none","category":"A3",
		 "nav_qnh":1013.6,"nav_altitude_mcp":37008,"nav_heading":196.9,"nav_modes":["autopilot","vnav","lnav"],
		 "lat":-27.1,"lon":153.2,"nic":8,"rc":186,"seen_pos":0.4,"version":2,"nac_p":10,"sil":3,
		 "sil_type":"perhour","mlat":[],"tisb":[],"messages":12345,"seen":0.1,"rssi":-18.2,"dbFlags":0},
		{"hex":"7c4920","type":"adsb_icao","flight":"VOZ1234 ","alt_baro":"ground","gs":12,"true_heading":10,"dbFlags":1}
	],"msg":"No error","now":1763258400000,"total":2,"ctime":1763258400000,"ptime":3}`

	res := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
	out, err := Decode(res)
	require.NoError(t, err)
	require.Len(t, out.Aircraft, 2)
	assert.EqualValues(t, 2, out.Total)
	assert.EqualValues(t, 3, out.PTime)

	air := out.Aircraft[0]
	assert.Equal(t, "QFA615", air.Callsign())
	assert.Equal(t, RecordTypeADSBICAO, air.RecordType)
	assert.Equal(t, CategoryLarge, air.Category)
	assert.InDelta(t, 0.788, *air.Mach, 1e-9)
	assert.InDelta(t, 1013.6, *air.NavQNH, 1e-9)
	assert.Equal(t, []string{NavModeAutopilot, NavModeVNAV, NavModeLNAV}, air.NavModes)
	assert.True(t, air.HasPosition())
	assert.False(t, air.IsMilitary())
	feet, onGround := air.Altitude()
	assert.Equal(t, 37000, *feet)
	assert.False(t, onGround)

	ground := out.Aircraft[1]
	feet, onGround = ground.Altitude()
	assert.Nil(t, feet)
	assert.True(t, onGround)
	assert.False(t, ground.HasPosition())
	assert.True(t, ground.IsMilitary())
}

func TestCheckStatus(t *testing.T) {
	now := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		status     int
		header     string
		body       string
		rateLimit  bool
		retryAfter time.Duration
		statusErr  bool
	}{
		{name: "200", status: 200},
		{name: "429 no header", status: 429, rateLimit: true},
		{name: "429 seconds", status: 429, header: "30", rateLimit: true, retryAfter: 30 * time.Second},
		{name: "429 zero", status: 429, header: "0", rateLimit: true},
		{name: "429 junk", status: 429, header: "soon", rateLimit: true},
		{name: "500", status: 500, body: " upstream down \n", statusErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &http.Response{
				StatusCode: tt.status,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			if tt.header != "" {
				res.Header.Set("Retry-After", tt.header)
			}

			err := CheckStatus(res)

			switch {
			case tt.rateLimit:
				require.ErrorIs(t, err, ErrRateLimited)
				var rl *RateLimitError
				require.True(t, errors.As(err, &rl))
				assert.Equal(t, tt.retryAfter, rl.RetryAfter)
			case tt.statusErr:
				var se *StatusError
				require.True(t, errors.As(err, &se))
				assert.Equal(t, tt.status, se.StatusCode)
				assert.Equal(t, "upstream down", se.Body)
				assert.NotErrorIs(t, err, ErrRateLimited)
			default:
				require.NoError(t, err)
			}
		})
	}

	t.Run("Retry-After as a date", func(t *testing.T) {
		at := now.Add(90 * time.Second).Format(http.TimeFormat)
		assert.Equal(t, 90*time.Second, retryAfter(at, now))
		assert.Zero(t, retryAfter(now.Add(-time.Minute).Format(http.TimeFormat), now))
	})
}

func TestDecodeErrors(t *testing.T) {
	_, err := Decode(&http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not json"))})
	require.ErrorContains(t, err, "unable to decode response")

	_, err = Decode(&http.Response{StatusCode: 429, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))})
	require.ErrorIs(t, err, ErrRateLimited)
}

func ptr(v int) *int { return &v }
