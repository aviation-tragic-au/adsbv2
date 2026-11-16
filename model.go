// Package adsbv2 holds readsb's "v2" aircraft JSON, the format served by
// ADS-B Exchange's API and the community APIs that mirror it (ADSB.lol,
// adsb.fi, airplanes.live), so every source decodes into the same types.
// AussieADSB's readsb re-api sends the same aircraft objects in a different
// envelope.
//
// Field meanings follow readsb's README-json.md:
// https://github.com/wiedehopf/readsb/blob/dev/README-json.md
package adsbv2

import (
	"encoding/json"
	"strings"
)

// DefaultUserAgent identifies a client to the community APIs, which ask API
// users to send one. Clients built on this package let the host replace it
// with its own.
const DefaultUserAgent = "adsbv2-go (+https://github.com/aviation-tragic-au/adsbv2)"

// Response is a v2 query result.
type Response struct {
	Aircraft []*Aircraft `json:"ac"`
	Message  string      `json:"msg"`
	// Now is when the response was generated, in milliseconds since the
	// Unix epoch.
	Now   int64 `json:"now"`
	Total int64 `json:"total"`
	// CTime and PTime are the server's cache time and processing time in
	// milliseconds, when it sends them.
	CTime int64 `json:"ctime"`
	PTime int64 `json:"ptime"`
}

// Aircraft is one aircraft in a v2 response. readsb omits whatever it
// doesn't currently know, so optional numbers are pointers: nil means not
// sent, not zero.
type Aircraft struct {
	// Hex is the 24-bit ICAO address, lower case. A leading '~' marks a
	// non-ICAO address (TIS-B, for example).
	Hex string `json:"hex"`
	// RecordType is where the data came from; see the RecordType
	// constants.
	RecordType   string `json:"type"`
	Flight       string `json:"flight"`
	Registration string `json:"r"`
	Type         string `json:"t"`
	Description  string `json:"desc"`
	Operator     string `json:"ownOp"`
	Year         string `json:"year"`
	// Category is the ADS-B emitter category; see the Category constants.
	Category  string `json:"category"`
	Squawk    string `json:"squawk"`
	Emergency string `json:"emergency"`
	// DBFlags is a bit field from the aggregator's database; see
	// IsMilitary and the other flag methods.
	DBFlags int `json:"dbFlags"`

	// AltBaro is feet, or the string "ground"; use Altitude.
	AltBaro json.RawMessage `json:"alt_baro"`
	AltGeom *int            `json:"alt_geom"`

	GS   *float64 `json:"gs"`
	IAS  *float64 `json:"ias"`
	TAS  *float64 `json:"tas"`
	Mach *float64 `json:"mach"`

	// Track is the direction of movement over the ground, in degrees.
	Track     *float64 `json:"track"`
	TrackRate *float64 `json:"track_rate"`
	Roll      *float64 `json:"roll"`
	// TrueHeading and MagHeading are where the nose points. On the ground
	// many aircraft send a heading instead of Track, which is what makes
	// pushback detectable (moving opposite to the heading).
	TrueHeading *float64 `json:"true_heading"`
	MagHeading  *float64 `json:"mag_heading"`
	BaroRate    *int     `json:"baro_rate"`
	GeomRate    *int     `json:"geom_rate"`

	// Wind and temperatures, derived by the aggregator from speeds and
	// headings; only some aircraft give enough to work them out.
	WindDirection  *float64 `json:"wd"`
	WindSpeed      *float64 `json:"ws"`
	OutsideAirTemp *float64 `json:"oat"`
	TotalAirTemp   *float64 `json:"tat"`

	// NavQNH is the altimeter setting the crew has set, in hPa.
	NavQNH *float64 `json:"nav_qnh"`
	// NavAltitudeMCP and NavAltitudeFMS are the altitude selected on the
	// autopilot (MCP/FCU) or in the flight management system: where a
	// climb or descent will level off. See SelectedAltitude.
	NavAltitudeMCP *int     `json:"nav_altitude_mcp"`
	NavAltitudeFMS *int     `json:"nav_altitude_fms"`
	NavHeading     *float64 `json:"nav_heading"`
	// NavModes are the engaged autopilot modes; see the NavMode
	// constants.
	NavModes []string `json:"nav_modes"`

	Lat *float64 `json:"lat"`
	Lon *float64 `json:"lon"`

	// Integrity and accuracy, as the transponder reports them.
	NIC     *int   `json:"nic"`
	RC      *int   `json:"rc"`
	Version *int   `json:"version"`
	NICBaro *int   `json:"nic_baro"`
	NACP    *int   `json:"nac_p"`
	NACV    *int   `json:"nac_v"`
	SIL     *int   `json:"sil"`
	SILType string `json:"sil_type"`
	GVA     *int   `json:"gva"`
	SDA     *int   `json:"sda"`
	Alert   *int   `json:"alert"`
	SPI     *int   `json:"spi"`

	// MLAT and TISB list the fields that came from multilateration or
	// TIS-B rather than the aircraft itself.
	MLAT []string `json:"mlat"`
	TISB []string `json:"tisb"`

	Messages int64 `json:"messages"`
	// SeenPos and Seen are seconds since the last position and the last
	// message of any kind.
	SeenPos *float64 `json:"seen_pos"`
	Seen    *float64 `json:"seen"`
	RSSI    *float64 `json:"rssi"`
}

// Record types: where an aircraft's data came from.
const (
	RecordTypeADSBICAO      = "adsb_icao"      // a Mode S or ADS-B transponder with a 24-bit ICAO address
	RecordTypeADSBICAONT    = "adsb_icao_nt"   // an ADS-B "non-transponder" emitter (a ground vehicle, say) with an ICAO address
	RecordTypeADSRICAO      = "adsr_icao"      // ADS-B rebroadcast from another data link (UAT), ICAO address
	RecordTypeTISBICAO      = "tisb_icao"      // TIS-B about a non-ADS-B target with an ICAO address
	RecordTypeADSC          = "adsc"           // ADS-C, from satellite downlinks
	RecordTypeMLAT          = "mlat"           // multilateration: position worked out from arrival times at several receivers
	RecordTypeOther         = "other"          // Basestation/SBS data of unknown quality
	RecordTypeModeS         = "mode_s"         // Mode S with no position
	RecordTypeADSBOther     = "adsb_other"     // ADS-B with a non-ICAO (anonymised) address
	RecordTypeADSROther     = "adsr_other"     // ADS-B rebroadcast with a non-ICAO address
	RecordTypeTISBOther     = "tisb_other"     // TIS-B about a target with a non-ICAO address
	RecordTypeTISBTrackFile = "tisb_trackfile" // TIS-B using a radar track/file identifier
)

// Emergency states.
const (
	EmergencyNone      = "none"
	EmergencyGeneral   = "general"
	EmergencyLifeguard = "lifeguard"
	EmergencyMinFuel   = "minfuel"
	EmergencyNORDO     = "nordo"
	EmergencyUnlawful  = "unlawful"
	EmergencyDowned    = "downed"
	EmergencyReserved  = "reserved"
)

// Autopilot modes, as listed in NavModes.
const (
	NavModeAltHold   = "althold"
	NavModeApproach  = "approach"
	NavModeAutopilot = "autopilot"
	NavModeLNAV      = "lnav"
	NavModeTCAS      = "tcas"
	NavModeVNAV      = "vnav"
)

// ADS-B emitter categories.
const (
	CategoryNone             = "A0" // no category information
	CategoryLight            = "A1" // under 15,500 lb
	CategorySmall            = "A2" // 15,500 to 75,000 lb
	CategoryLarge            = "A3" // 75,000 to 300,000 lb
	CategoryHighVortexLarge  = "A4" // high wake vortex, 75,000 to 300,000 lb (a B757)
	CategoryHeavy            = "A5" // over 300,000 lb
	CategoryHighPerformance  = "A6" // over 5 g and 400 kt
	CategoryRotorcraft       = "A7"
	CategoryGlider           = "B1"
	CategoryLighterThanAir   = "B2"
	CategoryParachutist      = "B3"
	CategoryUltralight       = "B4"
	CategoryUAV              = "B6"
	CategorySpacecraft       = "B7"
	CategoryEmergencyVehicle = "C1"
	CategoryServiceVehicle   = "C2"
	CategoryPointObstacle    = "C3"
)

// CategoryDescription describes each emitter category.
var CategoryDescription = map[string]string{
	CategoryNone:             "No category information",
	CategoryLight:            "Light aircraft",
	CategorySmall:            "Small aircraft",
	CategoryLarge:            "Large aircraft",
	CategoryHighVortexLarge:  "High wake vortex aircraft",
	CategoryHeavy:            "Heavy aircraft",
	CategoryHighPerformance:  "High performance aircraft",
	CategoryRotorcraft:       "Rotorcraft",
	CategoryGlider:           "Glider or sailplane",
	CategoryLighterThanAir:   "Lighter than air",
	CategoryParachutist:      "Parachutist or skydiver",
	CategoryUltralight:       "Ultralight",
	CategoryUAV:              "Unmanned aerial vehicle",
	CategorySpacecraft:       "Spacecraft",
	CategoryEmergencyVehicle: "Surface emergency vehicle",
	CategoryServiceVehicle:   "Surface service vehicle",
	CategoryPointObstacle:    "Point obstacle",
}

// DBFlags bits.
const (
	FlagMilitary    = 1 << 0
	FlagInteresting = 1 << 1
	FlagPIA         = 1 << 2 // FAA Privacy ICAO Address program
	FlagLADD        = 1 << 3 // FAA Limiting Aircraft Data Displayed program
)

// IsMilitary reports whether the aggregator's database marks the aircraft
// as military.
func (a *Aircraft) IsMilitary() bool { return a.DBFlags&FlagMilitary != 0 }

// IsInteresting reports whether the aggregator's database marks the
// aircraft as interesting.
func (a *Aircraft) IsInteresting() bool { return a.DBFlags&FlagInteresting != 0 }

// IsPIA reports whether the aircraft uses a privacy ICAO address.
func (a *Aircraft) IsPIA() bool { return a.DBFlags&FlagPIA != 0 }

// IsLADD reports whether the aircraft is on the FAA's LADD list.
func (a *Aircraft) IsLADD() bool { return a.DBFlags&FlagLADD != 0 }

// Altitude decodes AltBaro: onGround when readsb reports "ground", otherwise
// the barometric altitude in feet (nil when unknown).
func (a *Aircraft) Altitude() (feet *int, onGround bool) {
	if len(a.AltBaro) == 0 || string(a.AltBaro) == "null" {
		return nil, false
	}

	var s string
	if json.Unmarshal(a.AltBaro, &s) == nil {
		return nil, s == "ground"
	}

	var n float64
	if json.Unmarshal(a.AltBaro, &n) == nil {
		ft := int(n)
		return &ft, false
	}

	return nil, false
}

// VerticalRate prefers the barometric rate, falling back to geometric.
func (a *Aircraft) VerticalRate() *int {
	if a.BaroRate != nil {
		return a.BaroRate
	}
	return a.GeomRate
}

// SelectedAltitude is the altitude the aircraft will level off at,
// preferring the autopilot's (MCP/FCU) selection, which is what's actually
// flown, over the FMS's. Nil when the aircraft sends neither.
func (a *Aircraft) SelectedAltitude() *int {
	if a.NavAltitudeMCP != nil {
		return a.NavAltitudeMCP
	}
	return a.NavAltitudeFMS
}

// Callsign is the trimmed, upper-cased flight field. readsb pads callsign
// characters it hasn't decoded yet with '@', so those are dropped too; an
// all-'@' callsign comes back empty.
func (a *Aircraft) Callsign() string {
	return strings.ToUpper(strings.Trim(a.Flight, " @"))
}

// HasPosition reports whether the aircraft has sent a position.
func (a *Aircraft) HasPosition() bool {
	return a.Lat != nil && a.Lon != nil
}
