/*
 * Copyright (c) 2013, 2014 Conformal Systems LLC <info@conformal.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoPreviousAppVersion describes an error where no previous
// application version was recorded.
var ErrNoPreviousAppVersion = errors.New("no previous application version")

// semanticAlphabet
const semanticAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

// These constants define the application version and follow the semantic
// versioning 2.0.0 spec (http://semver.org/).
const (
	appMajor uint = 0
	appMinor uint = 2
	appPatch uint = 0

	// appPreRelease MUST only contain characters from semanticAlphabet
	// per the semantic versioning spec.
	appPreRelease = "alpha"
)

// appVersionFilename describes the filename used to write the current
// application version, and the file read at startup to determine the
// version of the previous application run.
const appVersionFilename = "version.txt"

type appVersion struct {
	major      uint
	minor      uint
	patch      uint
	prerelease string
	metadata   string
}

var version = appVersion{
	major:      appMajor,
	minor:      appMinor,
	patch:      appPatch,
	prerelease: appPreRelease,
}

// ParseVersion parses and returns an appVersion based on a version
// string.
func ParseVersion(s string) appVersion {
	var v appVersion
	fmt.Sscanf(s, "%d.%d.%d-%s+%s", &v.major, &v.minor, &v.patch, &v.prerelease, &v.metadata)
	return v
}

// version returns the application version as a properly formed string per the
// semantic versioning 2.0.0 spec (http://semver.org/).
func (v appVersion) String() string {
	// Start with the major, minor, and path versions.
	version := fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch)

	// Append pre-release version if there is one.  The hyphen called for
	// by the semantic versioning spec is automatically appended and should
	// not be contained in the pre-release string.  The pre-release version
	// is not appended if it contains invalid characters.
	preRelease := normalizeVerString(appPreRelease)
	if preRelease != "" {
		version = fmt.Sprintf("%s-%s", version, preRelease)
	}

	// Append build metadata if there is any.  The plus called for
	// by the semantic versioning spec is automatically appended and should
	// not be contained in the build metadata string.  The build metadata
	// string is not appended if it contains invalid characters.
	build := normalizeVerString(appBuild)
	if build != "" {
		version = fmt.Sprintf("%s+%s", version, build)
	}

	return version
}

// NewerThan tests whether an application version v is newer than a a
// second version v2.
func (v appVersion) NewerThan(v2 appVersion) bool {
	switch {
	case v.major > v2.major:
		return true
	case v.major < v2.major:
		return false
	case v.minor > v2.minor:
		return true
	case v.minor < v2.minor:
		return false
	case v.patch > v2.patch:
		return true
	default:
		return false
	}
}

// Equal tests whether two application versions are equal.
func (v appVersion) Equal(v2 appVersion) bool {
	switch {
	case v.major != v2.major, v.minor != v2.minor, v.patch != v2.patch:
		return false
	default:
		return true
	}
}

// SaveToDataDir writes the current application version string to
// the version file so it can be read on a future application run.
func (v appVersion) SaveToDataDir(cfg *config) error {
	// TODO(jrick): when home dir becomes a config option, use correct
	// directory.
	hdir := btcguiHomeDir
	fi, err := os.Stat(hdir)
	if err != nil {
		if os.IsNotExist(err) {
			// Attempt data directory creation
			if err = os.MkdirAll(hdir, 0700); err != nil {
				return fmt.Errorf("cannot create data directory: %s", err)
			}
		} else {
			return fmt.Errorf("error checking data directory: %s", err)
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("data directory '%s' is not a directory", hdir)
		}
	}

	filename := filepath.Join(hdir, appVersionFilename)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "%s\n", v.String())
	return err
}

// appBuild is defined as a variable so it can be overridden during the build
// process with '-ldflags "-X main.appBuild foo' if needed.  It MUST only
// contain characters from semanticAlphabet per the semantic versioning spec.
var appBuild string

// normalizeVerString returns the passed string stripped of all characters which
// are not valid according to the semantic versioning guidelines for pre-release
// version and build metadata strings.  In particular they MUST only contain
// characters in semanticAlphabet.
func normalizeVerString(str string) string {
	var result bytes.Buffer
	for _, r := range str {
		if strings.ContainsRune(semanticAlphabet, r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetPreviousAppVersion returns the previously recorded application
// version, or ErrNoPreviousAppVersion if no version was recorded.
func GetPreviousAppVersion(cfg *config) (*appVersion, error) {
	// TODO(jrick): when home dir becomes a config option, use correct
	// directory.
	filename := filepath.Join(btcguiHomeDir, appVersionFilename)
	if !fileExists(filename) {
		return nil, ErrNoPreviousAppVersion
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	filebuf := bufio.NewReader(file)
	line, err := filebuf.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	verstr := string(line)
	ver := ParseVersion(verstr)
	return &ver, nil

}
