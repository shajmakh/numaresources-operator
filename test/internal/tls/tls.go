/*
 * Copyright 2026 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tls

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"k8s.io/klog/v2"

	configv1 "github.com/openshift/api/config/v1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
)

// ErrTLSHandshakeRejected indicates the server rejected the TLS handshake.
var ErrTLSHandshakeRejected = errors.New("TLS handshake rejected by server")

func wrapTLSHandshakeError(err error) error {
	if err == nil {
		return nil
	}
	var alertErr *tls.AlertError
	if errors.As(err, &alertErr) {
		return fmt.Errorf("%w: %w", ErrTLSHandshakeRejected, err)
	}
	if strings.Contains(err.Error(), "handshake failure") ||
		strings.Contains(err.Error(), "protocol version") {
		return fmt.Errorf("%w: %w", ErrTLSHandshakeRejected, err)
	}
	return err
}

const dialTimeout = 5 * time.Second

var dialer = &net.Dialer{Timeout: dialTimeout}

func tlsDial(addr string, cfg *tls.Config) (*tls.ConnectionState, error) {
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	state := conn.ConnectionState()
	return &state, nil
}

// ProbeMaxTLSVersion checks whether the server at addr rejects
// TLS connections capped at the given maximum version.
func ProbeMaxTLSVersion(addr string, maxVersion uint16) error {
	klog.InfoS("probe with max TLS version", "addr", addr, "maxVersion", tls.VersionName(maxVersion))
	_, err := tlsDial(addr, &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         maxVersion,
	})
	return wrapTLSHandshakeError(err)
}

// ProbeTLSCipher checks whether the server at addr rejects
// TLS connections when the client only offers the given cipher (OpenSSL name).
// Since this is dedicated to invalidating ciphers, it is assumed that it is
// called on non TLS 1.3 configuration.
func ProbeTLSCipher(addr string, cipher string) error {
	cipherID, err := OpenSSLCipherToGoID(cipher)
	if err != nil {
		return err
	}
	klog.InfoS("probe with TLS cipher", "addr", addr, "cipher", cipher, "cipherID", cipherID)
	_, err = tlsDial(addr, &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS12,
		CipherSuites:       []uint16{cipherID},
	})
	return wrapTLSHandshakeError(err)
}

// ProbeTLSSettings connects to the server at addr and returns the negotiated
// TLS version and cipher suite.
func ProbeTLSSettings(addr string) (version uint16, cipherSuite uint16, err error) {
	klog.InfoS("probe TLS settings", "addr", addr)
	state, err := tlsDial(addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return 0, 0, err
	}
	klog.InfoS("negotiated TLS settings", "addr", addr, "version", tls.VersionName(state.Version), "cipher", tls.CipherSuiteName(state.CipherSuite))
	return state.Version, state.CipherSuite, nil
}

// OpenSSLCipherToGoID converts an OpenSSL cipher name to a Go crypto/tls
// cipher suite ID by going through the IANA name as an intermediate form.
func OpenSSLCipherToGoID(opensslName string) (uint16, error) {
	ianaNames := libgocrypto.OpenSSLToIANACipherSuites([]string{opensslName})
	if len(ianaNames) == 0 {
		return 0, fmt.Errorf("unknown OpenSSL cipher %q", opensslName)
	}
	ianaName := ianaNames[0]
	for _, cs := range tls.CipherSuites() {
		if cs.Name == ianaName {
			return cs.ID, nil
		}
	}
	for _, cs := range tls.InsecureCipherSuites() {
		if cs.Name == ianaName {
			return cs.ID, nil
		}
	}
	return 0, fmt.Errorf("no Go cipher suite for IANA name %q (from OpenSSL %q)", ianaName, opensslName)
}

// findDisallowedCipher returns the first TLS 1.2 cipher (OpenSSL name)
// from the broadest upstream profile (Old) that is not in the allowed set.
// TLS 1.3 ciphers are skipped because they are not individually configurable.
func FindDisallowedCipher(allowed []string) string {
	allowedSet := make(map[string]bool, len(allowed))
	for _, c := range allowed {
		allowedSet[c] = true
	}
	allCiphers := configv1.TLSProfiles[configv1.TLSProfileOldType].Ciphers
	for _, cipher := range allCiphers {
		if strings.HasPrefix(cipher, "TLS_") {
			continue
		}
		if !allowedSet[cipher] {
			return cipher
		}
	}
	return ""
}

func TLSVersionBelow(v uint16) (uint16, error) {
	switch v {
	case tls.VersionTLS13:
		return tls.VersionTLS12, nil
	case tls.VersionTLS12:
		return tls.VersionTLS11, nil
	case tls.VersionTLS11:
		return tls.VersionTLS10, nil
	default:
		return 0, fmt.Errorf("unknown TLS version %d", v)
	}
}
