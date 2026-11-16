# adsbv2

[![Lint](https://github.com/aviation-tragic-au/adsbv2/actions/workflows/lint.yml/badge.svg)](https://github.com/aviation-tragic-au/adsbv2/actions/workflows/lint.yml) [![Test](https://github.com/aviation-tragic-au/adsbv2/actions/workflows/test.yml/badge.svg)](https://github.com/aviation-tragic-au/adsbv2/actions/workflows/test.yml)

Go types for [readsb](https://github.com/wiedehopf/readsb)'s "v2" aircraft
JSON: the format served by ADS-B Exchange's API and the community APIs that
mirror it (ADSB.lol, adsb.fi, airplanes.live). Every source decodes into
the same `Aircraft`, so a program can switch or combine sources without
converting between models.

It has no dependencies. The API clients built on it, such as
[adsbexchange-client](https://github.com/aviation-tragic-au/adsbexchange-client),
use it for their results.

## Install

```sh
go get github.com/aviation-tragic-au/adsbv2
```

## Usage

```go
res, err := http.Get("https://api.adsb.lol/v2/lat/-27.38/lon/153.12/dist/25")
if err != nil {
    log.Fatal(err)
}
defer res.Body.Close()

out, err := adsbv2.Decode(res)
if errors.Is(err, adsbv2.ErrRateLimited) {
    var rl *adsbv2.RateLimitError
    errors.As(err, &rl)
    log.Printf("rate limited; retry after %s", rl.RetryAfter)
    return
}
if err != nil {
    log.Fatal(err)
}

for _, ac := range out.Aircraft {
    feet, onGround := ac.Altitude()
    switch {
    case onGround:
        log.Printf("%s on the ground", ac.Callsign())
    case feet != nil:
        log.Printf("%s at %d ft", ac.Callsign(), *feet)
    }
}
```

readsb leaves out whatever it doesn't know, so optional numbers are
pointers: nil means not sent, not zero.

### Helpers

| Method | |
|---|---|
| `Altitude()` | Barometric altitude in feet, or on the ground (`alt_baro` is a number or the string `"ground"`) |
| `Callsign()` | `flight` trimmed and upper-cased, without readsb's `@` padding |
| `VerticalRate()` | Barometric rate, falling back to geometric |
| `SelectedAltitude()` | Where a climb or descent will level off (MCP/FCU, then FMS) |
| `HasPosition()` | Whether a latitude and longitude were sent |
| `IsMilitary()`, `IsInteresting()`, `IsPIA()`, `IsLADD()` | The aggregator's `dbFlags` |

`Decode` and `CheckStatus` map a 429 to a `*RateLimitError` (matching
`ErrRateLimited`, with the `Retry-After` delay) and any other non-200 to a
`*StatusError`. The `RecordType…`, `Category…`, `Emergency…` and
`NavMode…` constants name the values readsb sends. `DefaultUserAgent` is a
User-Agent for the community APIs, which ask clients to send one.

Field meanings follow readsb's
[README-json.md](https://github.com/wiedehopf/readsb/blob/dev/README-json.md).

## License

Copyright 2026 Dean Stalker

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
