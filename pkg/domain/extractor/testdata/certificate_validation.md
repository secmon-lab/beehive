# Updates to Certificate Validation Methods

The Certificate Authority Browser Forum has announced changes to acceptable validation methods for TLS certificates, effective March 2024.

## Sunsetted Validation Methods

Several validation methods are being retired:

- Phone Contact with DNS CAA Phone Contact: https://cabforum.org/working-groups/server/baseline-requirements/requirements/#322417-phone-contact-with-dns-caa-phone-contact
- Phone Contact with IP Address Contact: https://cabforum.org/working-groups/server/baseline-requirements/requirements/#32255-phone-contact-with-ip-address-contact
- IP Address validation: https://cabforum.org/working-groups/server/baseline-requirements/requirements/#32248-ip-address
- Reverse Address Lookup: https://cabforum.org/working-groups/server/baseline-requirements/requirements/#32253-reverse-address-lookup

## Recommended Alternatives

Certificate Authorities should transition to ACME (Automated Certificate Management Environment) protocol as documented in RFC 8555: https://datatracker.ietf.org/doc/html/rfc8555/

For more details, see the Chromium Security blog: https://blog.chromium.org/2023/10/unlocking-power-of-tls-certificate.html

## Technical Implementation

Validation typically uses DNS queries to authoritative servers (e.g., 8.8.8.8, 1.1.1.1) or direct connections to web servers on port 443. Example validation domains include:

- _acme-challenge.example.com
- validation.letsencrypt.org
- test.example.net (testing environment)

The validation process may connect to localhost (127.0.0.1) during development.

## References

- CA/Browser Forum homepage: https://cabforum.org
- Let's Encrypt documentation: https://letsencrypt.org/docs/
- Google Trust Services: https://pki.goog

**Contact**: Chrome Root Program Team
**Profile**: https://plus.google.com/116899029375914044550
