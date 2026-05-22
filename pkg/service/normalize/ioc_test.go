package normalize_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/normalize"
)

func TestIoC_IPv4(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeIPv4, "1.2.3.4")
	gt.NoError(t, err)
	gt.Equal(t, v, "1.2.3.4")
}

func TestIoC_IPv4_RejectsPrivate(t *testing.T) {
	_, err := normalize.IoC(types.IoCTypeIPv4, "127.0.0.1")
	gt.Error(t, err)
	_, err = normalize.IoC(types.IoCTypeIPv4, "10.0.0.1")
	gt.Error(t, err)
}

func TestIoC_IPv4_TypeMismatch(t *testing.T) {
	_, err := normalize.IoC(types.IoCTypeIPv4, "2001:db8::1")
	gt.Error(t, err)
}

func TestIoC_IPv6(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeIPv6, "2001:db8::1")
	gt.NoError(t, err)
	gt.Equal(t, v, "2001:db8::1")
}

func TestIoC_Domain(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeDomain, "Evil[.]Example.COM.")
	gt.NoError(t, err)
	gt.Equal(t, v, "evil.example.com")
}

func TestIoC_Domain_RejectsBare(t *testing.T) {
	_, err := normalize.IoC(types.IoCTypeDomain, "localhost")
	gt.Error(t, err)
}

func TestIoC_URL(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeURL, "hxxps://Evil[.]example/path?x=1#frag")
	gt.NoError(t, err)
	gt.Equal(t, v, "https://evil.example/path?x=1")
}

func TestIoC_Hash(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeMD5, "DEADBEEFDEADBEEFDEADBEEFDEADBEEF")
	gt.NoError(t, err)
	gt.Equal(t, v, "deadbeefdeadbeefdeadbeefdeadbeef")

	_, err = normalize.IoC(types.IoCTypeMD5, "tooshort")
	gt.Error(t, err)

	_, err = normalize.IoC(types.IoCTypeSHA256, "DEADBEEFDEADBEEFDEADBEEFDEADBEEF")
	gt.Error(t, err) // wrong length for sha256
}

func TestIoC_Email(t *testing.T) {
	v, err := normalize.IoC(types.IoCTypeEmail, "Bad@Evil[.]Example.com")
	gt.NoError(t, err)
	gt.Equal(t, v, "bad@evil.example.com")

	_, err = normalize.IoC(types.IoCTypeEmail, "not-an-email")
	gt.Error(t, err)
}

func TestIoC_Unsupported(t *testing.T) {
	_, err := normalize.IoC(types.IoCType("bogus"), "x")
	gt.Error(t, err)
}
