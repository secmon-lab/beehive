package types

import "slices"

// IoCType is the kind of indicator. CVE / ASN are intentionally excluded from
// MVP because they are not used as direct match targets; only types that can
// be looked up against observed traffic / artefacts are kept.
type IoCType string

const (
	IoCTypeIPv4   IoCType = "ipv4"
	IoCTypeIPv6   IoCType = "ipv6"
	IoCTypeDomain IoCType = "domain"
	IoCTypeURL    IoCType = "url"
	IoCTypeMD5    IoCType = "md5"
	IoCTypeSHA1   IoCType = "sha1"
	IoCTypeSHA256 IoCType = "sha256"
	IoCTypeSHA512 IoCType = "sha512"
	IoCTypeEmail  IoCType = "email"
)

func (t IoCType) String() string { return string(t) }

// AllIoCTypes returns the supported IoC type set in a stable order. Useful
// for prompt construction and validation.
func AllIoCTypes() []IoCType {
	return []IoCType{
		IoCTypeIPv4, IoCTypeIPv6,
		IoCTypeDomain, IoCTypeURL,
		IoCTypeMD5, IoCTypeSHA1, IoCTypeSHA256, IoCTypeSHA512,
		IoCTypeEmail,
	}
}

func (t IoCType) Valid() bool {
	return slices.Contains(AllIoCTypes(), t)
}
