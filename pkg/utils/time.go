// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"strings"
	"time"
)

// ToUnixMilliseconds converts a time.Time object to Unix time in milliseconds.
// This format is commonly used in APIs and databases.
//
// Parameters:
//   t (time.Time): The time object to convert.
//
// Returns:
//   int64: The Unix time in milliseconds.
func ToUnixMilliseconds(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// FromUnixMilliseconds converts Unix time in milliseconds back to a time.Time object.
//
// Parameters:
//   ms (int64): The Unix time in milliseconds.
//
// Returns:
//   time.Time: The corresponding time.Time object.
func FromUnixMilliseconds(ms int64) time.Time {
	return time.Unix(0, ms*int64(time.Millisecond))
}

// TimeIn returns the time `t` converted to the specified timezone location string (e.g., "America/New_York").
//
// Parameters:
//   t (time.Time): The time to convert.
//   location (string): The IANA Time Zone database name (e.g., "UTC", "America/New_York").
//
// Returns:
//   time.Time: The time in the new location.
//   error: An error if the location cannot be loaded.
func TimeIn(t time.Time, location string) (time.Time, error) {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load location '%s': %w", location, err)
	}
	return t.In(loc), nil
}

// FormatDuration formats a time.Duration into a more human-readable string,
// showing days, hours, minutes, and seconds.
//
// Parameters:
//   d (time.Duration): The duration to format.
//
// Returns:
//   string: A human-readable representation of the duration (e.g., "2d 4h 30m 15s").
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return "less than a second"
	}

	d = d.Round(time.Second)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}

	return strings.Join(parts, " ")
}

// TODO: Implement a simple cron-like scheduler for timed tasks.
// TODO: Implement utilities for time-series data analysis, such as windowing and aggregation functions.

//Personal.AI order the ending
