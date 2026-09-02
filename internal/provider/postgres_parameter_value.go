package provider

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// unitUndefined is the API unit for numeric parameters that have no unit.
const unitUndefined = "UNDEFINED"

// parameterUnit maps an API unit enum to its postgresql.conf suffix and its
// multiplier to the family base unit: bytes for memory, microseconds for time.
type parameterUnit struct {
	apiName string
	suffix  string
	memory  bool
	factor  float64
}

// parameterUnits is ordered from smallest to largest within each family.
var parameterUnits = []parameterUnit{
	{apiName: "BYTES", suffix: "B", memory: true, factor: 1},
	{apiName: "KILOBYTES", suffix: "kB", memory: true, factor: 1 << 10},
	{apiName: "MEGABYTES", suffix: "MB", memory: true, factor: 1 << 20},
	{apiName: "GIGABYTES", suffix: "GB", memory: true, factor: 1 << 30},
	{apiName: "TERABYTES", suffix: "TB", memory: true, factor: 1 << 40},
	{apiName: "MICROSECONDS", suffix: "us", memory: false, factor: 1},
	{apiName: "MILLISECONDS", suffix: "ms", memory: false, factor: 1e3},
	{apiName: "SECONDS", suffix: "s", memory: false, factor: 1e6},
	{apiName: "MINUTES", suffix: "min", memory: false, factor: 60e6},
	{apiName: "HOURS", suffix: "h", memory: false, factor: 3600e6},
	{apiName: "DAYS", suffix: "d", memory: false, factor: 86400e6},
}

const validUnitSuffixes = "B, kB, MB, GB, TB, us, ms, s, min, h, d"

func unitByAPIName(name string) (parameterUnit, bool) {
	for _, u := range parameterUnits {
		if u.apiName == name {
			return u, true
		}
	}
	return parameterUnit{}, false
}

func unitBySuffix(suffix string) (parameterUnit, bool) {
	for _, u := range parameterUnits {
		if u.suffix == suffix {
			return u, true
		}
	}
	return parameterUnit{}, false
}

// largestDividingUnit returns the largest unit of the family with factor <= maxFactor
// that divides base evenly. base is a whole number of the family's smallest unit.
func largestDividingUnit(memory bool, base, maxFactor float64) parameterUnit {
	var best parameterUnit
	for _, u := range parameterUnits {
		if u.memory == memory && u.factor <= maxFactor && math.Mod(math.Abs(base), u.factor) == 0 {
			best = u
		}
	}
	return best
}

func familyName(u parameterUnit) string {
	if u.memory {
		return "memory"
	}
	return "time"
}

// parameterCatalogEntry is one parameter as reported by getPostgresParameters.
type parameterCatalogEntry struct {
	info    tsClient.ParameterInfo
	numeric bool
	// unit is the API unit enum for numeric parameters; unitUndefined for plain numbers.
	unit     string
	value    float64
	strValue string
	// boolean is true for string parameters whose allowed values are exactly {"on", "off"}.
	boolean bool
}

// parsedParameterValue is a config value ready to be sent to setPostgresParameters.
type parsedParameterValue struct {
	numeric bool
	value   float64
	unit    string
	str     string
}

func buildParameterCatalog(p *tsClient.PostgresParameters) map[string]parameterCatalogEntry {
	catalog := make(map[string]parameterCatalogEntry, len(p.StringParameters)+len(p.NumericParameters))
	for _, sp := range p.StringParameters {
		catalog[sp.Info.Name] = parameterCatalogEntry{info: sp.Info, strValue: sp.CurrentValue, boolean: isBooleanAllowedValues(sp.AllowedValues)}
	}
	for _, np := range p.NumericParameters {
		unit := np.Unit
		if unit == "" {
			unit = unitUndefined
		}
		catalog[np.Info.Name] = parameterCatalogEntry{info: np.Info, numeric: true, unit: unit, value: np.CurrentValue}
	}
	return catalog
}

// isBooleanAllowedValues reports whether allowed is exactly {"on", "off"}, in any order.
func isBooleanAllowedValues(allowed []string) bool {
	if len(allowed) != 2 {
		return false
	}
	return (allowed[0] == "on" && allowed[1] == "off") || (allowed[0] == "off" && allowed[1] == "on")
}

var valueWithUnitRe = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*([A-Za-z]+)$`)

// parseFinite is strconv.ParseFloat without NaN and Inf, which the API cannot encode.
func parseFinite(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// parseParameterValue interprets a postgresql.conf style value for the given parameter.
func parseParameterValue(entry parameterCatalogEntry, raw string) (parsedParameterValue, error) {
	raw = strings.TrimSpace(raw)
	if !entry.numeric {
		if !entry.boolean {
			return parsedParameterValue{str: raw}, nil
		}
		switch strings.ToLower(raw) {
		case "on", "true", "yes", "1":
			return parsedParameterValue{str: "on"}, nil
		case "off", "false", "no", "0":
			return parsedParameterValue{str: "off"}, nil
		default:
			return parsedParameterValue{}, fmt.Errorf("%q is not a boolean; use on or off", raw)
		}
	}
	entryUnit, known := unitByAPIName(entry.unit)
	if !known {
		// UNDEFINED, and any unit this provider does not know, take bare numbers.
		v, ok := parseFinite(raw)
		if !ok {
			return parsedParameterValue{}, fmt.Errorf("%q is not a number", raw)
		}
		return parsedParameterValue{numeric: true, value: v, unit: entry.unit}, nil
	}

	// Bare numbers are ambiguous for unit parameters. Postgres reads 0 and negatives
	// as special values in the default unit, so only those are accepted bare.
	if v, ok := parseFinite(raw); ok {
		if v > 0 {
			return parsedParameterValue{}, fmt.Errorf("%q needs an explicit unit, for example 64%s", raw, entryUnit.suffix)
		}
		return parsedParameterValue{numeric: true, value: v, unit: entry.unit}, nil
	}

	m := valueWithUnitRe.FindStringSubmatch(raw)
	if m == nil {
		return parsedParameterValue{}, fmt.Errorf("%q is not a number followed by a unit (valid units: %s)", raw, validUnitSuffixes)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return parsedParameterValue{}, fmt.Errorf("%q is not a number", m[1])
	}
	u, ok := unitBySuffix(m[2])
	if !ok {
		return parsedParameterValue{}, fmt.Errorf("unknown unit %q (valid units: %s)", m[2], validUnitSuffixes)
	}
	if u.memory != entryUnit.memory {
		return parsedParameterValue{}, fmt.Errorf("unit %q is a %s unit but the parameter takes %s units", m[2], familyName(u), familyName(entryUnit))
	}

	// The API rejects fractional values with units, so express them in a smaller unit.
	base := v * u.factor
	rounded := math.Round(base)
	if math.Abs(base-rounded) > math.Max(1e-6, math.Abs(base)*1e-12) {
		smallest := "B"
		if !u.memory {
			smallest = "us"
		}
		return parsedParameterValue{}, fmt.Errorf("%q cannot be expressed as a whole number of %s", raw, smallest)
	}
	best := largestDividingUnit(u.memory, rounded, u.factor)
	return parsedParameterValue{numeric: true, value: rounded / best.factor, unit: best.apiName}, nil
}

// formatParameterValue renders the running value in postgresql.conf syntax, in the largest
// unit that divides it evenly, so imported values read the way people write them.
func formatParameterValue(entry parameterCatalogEntry) string {
	if !entry.numeric {
		return entry.strValue
	}
	s := strconv.FormatFloat(entry.value, 'f', -1, 64)
	u, ok := unitByAPIName(entry.unit)
	if !ok {
		return s
	}
	// Zero renders in the API's reported unit; every unit divides zero.
	if entry.value == 0 {
		return "0" + u.suffix
	}

	base := math.Round(entry.value * u.factor)
	best := largestDividingUnit(u.memory, base, math.Inf(1))
	sign := ""
	if base < 0 {
		sign = "-"
	}
	return sign + strconv.FormatFloat(math.Abs(base)/best.factor, 'f', -1, 64) + best.suffix
}

// parsedValueEqual reports whether p denotes the entry's running value.
func parsedValueEqual(entry parameterCatalogEntry, p parsedParameterValue) bool {
	if !p.numeric {
		return p.str == entry.strValue
	}
	pu, pok := unitByAPIName(p.unit)
	eu, eok := unitByAPIName(entry.unit)
	if !pok || !eok {
		// UNDEFINED and unknown units compare as bare numbers.
		return p.unit == entry.unit && p.value == entry.value
	}
	return p.value*pu.factor == entry.value*eu.factor
}

// parameterValueEqual reports whether raw denotes the entry's running value.
func parameterValueEqual(entry parameterCatalogEntry, raw string) bool {
	p, err := parseParameterValue(entry, raw)
	return err == nil && parsedValueEqual(entry, p)
}
