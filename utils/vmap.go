package utils

import (
	"strconv"
	"strings"
)

// VersionedMap is a CRDT where each key contains a version history of prior values.
// The value of the key is the value with the latest version.  VersionMaps can be combined
// such that they always converge to the same values for all keys.
type VersionedMap struct {
	values map[string][]mapEntry
}

type mapEntry struct {
	value   string
	version int64
}

func NewVersionedMap() *VersionedMap {
	return &VersionedMap{
		values: make(map[string][]mapEntry),
	}
}

func (v *VersionedMap) currentVersion(key string) int64 {
	next := int64(0)
	for _, mapEntry := range v.values[key] {
		if mapEntry.version > next {
			next = mapEntry.version
		}
	}
	return next
}

func (v *VersionedMap) nextVersion(key string) int64 {
	return v.currentVersion(key) + 1
}

func (v *VersionedMap) SetVersion(key, value string, version int64) {
	entries := v.values[key]
	v.values[key] = append(entries, mapEntry{
		value:   value,
		version: version,
	})
}

func (v *VersionedMap) UnSetVersion(key string, version int64) {
	entries := v.values[key]
	v.values[key] = append(entries, mapEntry{
		value:   "",
		version: version,
	})
}

func (v *VersionedMap) Set(key, value string) {
	v.SetVersion(key, value, v.nextVersion(key))
}

func (v *VersionedMap) UnSet(key string) {
	v.UnSetVersion(key, v.nextVersion(key))
}

func (v *VersionedMap) Get(key string) string {
	entries := v.values[key]
	maxEntry := mapEntry{}
	for _, entry := range entries {
		// value is max(version)
		if entry.version > maxEntry.version {
			maxEntry = entry
		}

		// if there is a conflict, prefer setting a value over unsetting one
		// as well the largest value as a tie-breaker if two sets conflict.
		if entry.version == maxEntry.version && entry.value > maxEntry.value {
			maxEntry = entry
		}

	}
	return maxEntry.value
}

func (v *VersionedMap) Keys() []string {
	keys := []string{}
	for k := range v.values {
		keys = append(keys, k)
	}
	return keys
}

func (v *VersionedMap) LatestVersion() int64 {
	latest := int64(0)
	for _, entries := range v.values {
		for _, mapEntry := range entries {
			if mapEntry.version > latest {
				latest = mapEntry.version
			}
		}
	}
	return latest
}

func (v *VersionedMap) Merge(other *VersionedMap) {
	for k, entries := range other.values {
		v.values[k] = append(v.values[k], entries...)
	}
}

func (v *VersionedMap) MarshalMap() map[string]string {
	result := make(map[string]string)
	for key, entries := range v.values {
		for _, mapEntry := range entries {
			op := "s"
			if mapEntry.value == "" {
				op = "u"
			}
			mapKey := strings.Join([]string{key, op, strconv.FormatInt(mapEntry.version, 10)}, ":")
			result[mapKey] = mapEntry.value
		}

	}
	return result
}

func (v *VersionedMap) UnmarshalMap(serialized map[string]string) error {

	for key, val := range serialized {
		parts := strings.Split(key, ":")
		version, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return err
		}
		if parts[1] == "s" {
			v.SetVersion(parts[0], val, version)
		} else {
			v.UnSetVersion(parts[0], version)
		}
	}
	return nil
}

// MarshalExpiredMap returns historical entries that have been
// superseded by newer values
func (v *VersionedMap) MarshalExpiredMap(age int64) map[string]string {
	result := make(map[string]string)
	for key, entries := range v.values {
		currentVersion := v.currentVersion(key)
		for _, mapEntry := range entries {
			if mapEntry.version >= currentVersion-age {
				continue
			}
			op := "s"
			if mapEntry.value == "" {
				op = "u"
			}
			mapKey := strings.Join([]string{key, op, strconv.FormatInt(mapEntry.version, 10)}, ":")
			result[mapKey] = mapEntry.value
		}

	}
	return result
}
