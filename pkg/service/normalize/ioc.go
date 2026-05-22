// Package normalize canonicalises raw IoC values that LLM extraction or
// upstream feeds return. The goal is that `(Type, Value)` after this
// function uniquely identifies an indicator across observations.
package normalize

import (
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/refang"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"golang.org/x/net/idna"
)

// ErrUnsupportedType is returned when the input IoC type is not one we
// know how to normalize. Caller may skip such entries.
var ErrUnsupportedType = goerr.New("unsupported ioc type", goerr.T(errutil.TagInvalidInput))

// ErrInvalidValue indicates that the value cannot be normalized into a
// valid representation of the declared type.
var ErrInvalidValue = goerr.New("invalid ioc value", goerr.T(errutil.TagInvalidInput))

var (
	hexRE   = regexp.MustCompile(`^[a-f0-9]+$`)
	emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// IoC takes the raw type + value pair and returns the canonical form.
// The Value is what we hash to compute the IoCID.
func IoC(t types.IoCType, value string) (string, error) {
	v := refang.Refang(strings.TrimSpace(value))
	if v == "" {
		return "", ErrInvalidValue
	}
	switch t {
	case types.IoCTypeIPv4:
		return normalizeIP(v, false)
	case types.IoCTypeIPv6:
		return normalizeIP(v, true)
	case types.IoCTypeDomain:
		return normalizeDomain(v)
	case types.IoCTypeURL:
		return normalizeURL(v)
	case types.IoCTypeMD5, types.IoCTypeSHA1, types.IoCTypeSHA256, types.IoCTypeSHA512:
		return normalizeHash(t, v)
	case types.IoCTypeEmail:
		return normalizeEmail(v)
	default:
		return "", ErrUnsupportedType
	}
}

func normalizeIP(v string, wantIPv6 bool) (string, error) {
	addr, err := netip.ParseAddr(v)
	if err != nil {
		return "", goerr.Wrap(errors.Join(ErrInvalidValue, err), "parse ip", goerr.V("value", v))
	}
	addr = addr.Unmap()
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return "", ErrInvalidValue
	}
	if wantIPv6 && addr.Is4() {
		return "", ErrInvalidValue
	}
	if !wantIPv6 && addr.Is6() {
		return "", ErrInvalidValue
	}
	return addr.String(), nil
}

func normalizeDomain(v string) (string, error) {
	v = strings.TrimSuffix(v, ".")
	v = strings.ToLower(v)
	ascii, err := idna.Lookup.ToASCII(v)
	if err != nil {
		return "", goerr.Wrap(errors.Join(ErrInvalidValue, err), "idna",
			goerr.V("value", v))
	}
	if ascii == "" || !strings.Contains(ascii, ".") {
		return "", ErrInvalidValue
	}
	return ascii, nil
}

func normalizeURL(v string) (string, error) {
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ErrInvalidValue
	}
	u.Fragment = ""
	host, err := idna.Lookup.ToASCII(strings.ToLower(u.Host))
	if err == nil {
		u.Host = host
	} else {
		u.Host = strings.ToLower(u.Host)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	return u.String(), nil
}

func normalizeHash(t types.IoCType, v string) (string, error) {
	v = strings.ToLower(v)
	if !hexRE.MatchString(v) {
		return "", ErrInvalidValue
	}
	var want int
	switch t {
	case types.IoCTypeMD5:
		want = 32
	case types.IoCTypeSHA1:
		want = 40
	case types.IoCTypeSHA256:
		want = 64
	case types.IoCTypeSHA512:
		want = 128
	}
	if len(v) != want {
		return "", ErrInvalidValue
	}
	return v, nil
}

func normalizeEmail(v string) (string, error) {
	v = strings.ToLower(v)
	if !emailRE.MatchString(v) {
		return "", ErrInvalidValue
	}
	at := strings.LastIndex(v, "@")
	local, domain := v[:at], v[at+1:]
	dom, err := normalizeDomain(domain)
	if err != nil {
		return "", err
	}
	return local + "@" + dom, nil
}
