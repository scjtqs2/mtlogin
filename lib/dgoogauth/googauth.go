package dgoogauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Much of this code assumes int == int64, which probably is not the case.

// ComputeCode computes the response code for a 64-bit challenge 'value' using the secret 'secret'.
// To avoid breaking compatibility with the previous API, it returns an invalid code (-1) when an error occurs,
// but does not silently ignore them (it forces a mismatch so the code will be rejected).
// 根据secret 和 时间戳 生成 token
func ComputeCode(secret string, value int64) int {

	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return -1
	}

	hash := hmac.New(sha1.New, key)
	err = binary.Write(hash, binary.BigEndian, value)
	if err != nil {
		return -1
	}
	h := hash.Sum(nil)

	offset := h[19] & 0x0f

	truncated := binary.BigEndian.Uint32(h[offset : offset+4])

	truncated &= 0x7fffffff
	code := truncated % 1000000

	return int(code)
}

// ErrInvalidCode indicate the supplied one-time code was not valid
var ErrInvalidCode = errors.New("invalid code")

// OTPConfig is a one-time-password configuration.  This object will be modified by calls to
// Authenticate and should be saved to ensure the codes are in fact only used
// once.
type OTPConfig struct {
	Secret        string // 80-bit base32 encoded string of the user's secret
	WindowSize    int    // valid range: technically 0..100 or so, but beyond 3-5 is probably bad security
	HotpCounter   int    // the current otp counter.  0 if the user uses time-based codes instead.
	DisallowReuse []int  // timestamps in the current window unavailable for re-use
	ScratchCodes  []int  // an array of 8-digit numeric codes that can be used to log in
	UTC           bool   // use UTC for the timestamp instead of local time
}

func (c *OTPConfig) checkScratchCodes(code int) bool {

	for i, v := range c.ScratchCodes {
		if code == v {
			// remove this code from the list of valid ones
			l := len(c.ScratchCodes) - 1
			c.ScratchCodes[i] = c.ScratchCodes[l] // copy last element over this element
			c.ScratchCodes = c.ScratchCodes[0:l]  // and trim the list length by 1
			return true
		}
	}

	return false
}

func (c *OTPConfig) checkHotpCode(code int) bool {

	for i := 0; i < c.WindowSize; i++ {
		if ComputeCode(c.Secret, int64(c.HotpCounter+i)) == code {
			c.HotpCounter += i + 1
			// We don't check for overflow here, which means you can only authenticate 2^63 times
			// After that, the counter is negative and the above 'if' test will fail.
			// This matches the behaviour of the PAM module.
			return true
		}
	}

	// we must always advance the counter if we tried to authenticate with it
	c.HotpCounter++
	return false
}

func (c *OTPConfig) checkTotpCode(t0, code int) bool {

	minT := t0 - (c.WindowSize / 2)
	maxT := t0 + (c.WindowSize / 2)
	for t := minT; t <= maxT; t++ {
		if ComputeCode(c.Secret, int64(t)) == code {

			if c.DisallowReuse != nil {
				for _, timeCode := range c.DisallowReuse {
					if timeCode == t {
						return false
					}
				}

				// code hasn't been used before
				c.DisallowReuse = append(c.DisallowReuse, t)

				// remove all time codes outside of the valid window
				sort.Ints(c.DisallowReuse)
				min := 0
				for c.DisallowReuse[min] < minT {
					min++
				}
				// FIXME: check we don't have an off-by-one error here
				c.DisallowReuse = c.DisallowReuse[min:]
			}

			return true
		}
	}

	return false
}

// Authenticate a one-time-password against the given OTPConfig
// Returns true/false if the authentication was successful.
// Returns error if the password is incorrectly formatted (not a zero-padded 6 or non-zero-padded 8 digit number).
func (c *OTPConfig) Authenticate(password string) (bool, error) {

	var scratch bool

	switch {
	case len(password) == 6 && password[0] >= '0' && password[0] <= '9':
		break
	case len(password) == 8 && password[0] >= '1' && password[0] <= '9':
		scratch = true
		break
	default:
		return false, ErrInvalidCode
	}

	code, err := strconv.Atoi(password)

	if err != nil {
		return false, ErrInvalidCode
	}

	if scratch {
		return c.checkScratchCodes(code), nil
	}

	// we have a counter value we can use
	if c.HotpCounter > 0 {
		return c.checkHotpCode(code), nil
	}

	var t0 int
	// assume we're on Time-based OTP
	if c.UTC {
		t0 = int(time.Now().UTC().Unix() / 30)
	} else {
		t0 = int(time.Now().Unix() / 30)
	}
	return c.checkTotpCode(t0, code), nil
}

// ProvisionURI generates a URI that can be turned into a QR code to configure
// a Google Authenticator mobile app.
func (c *OTPConfig) ProvisionURI(user string) string {
	return c.ProvisionURIWithIssuer(user, "")
}

// ProvisionURIWithIssuer generates a URI that can be turned into a QR code
// to configure a Google Authenticator mobile app. It respects the recommendations
// on how to avoid conflicting accounts.
//
// See https://github.com/google/google-authenticator/wiki/Conflicting-Accounts
func (c *OTPConfig) ProvisionURIWithIssuer(user string, issuer string) string {
	auth := "totp/"
	q := make(url.Values)
	if c.HotpCounter > 0 {
		auth = "hotp/"
		q.Add("counter", strconv.Itoa(c.HotpCounter))
	}
	q.Add("secret", c.Secret)
	if issuer != "" {
		q.Add("issuer", issuer)
		auth += issuer + ":"
	}

	return "otpauth://" + auth + user + "?" + q.Encode()
}

// Append extra 0s if the length of otp is less than 6
// If otp is "1234", it will return it as "001234"
func prefix0(otp string) string {
	if len(otp) == 6 {
		return otp
	}
	for i := (6 - len(otp)); i > 0; i-- {
		otp = "0" + otp
	}
	return otp
}

func getHOTPToken(secret string, interval int64) (string, error) {

	// Converts secret to base32 Encoding. Base32 encoding desires a 32-character
	// subset of the twenty-six letters A–Z and ten digits 0–9
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(interval))

	// Signing the value using HMAC-SHA1 Algorithm
	hash := hmac.New(sha1.New, key)
	hash.Write(bs)
	h := hash.Sum(nil)

	// We're going to use a subset of the generated hash.
	// Using the last nibble (half-byte) to choose the index to start from.
	// This number is always appropriate as it's maximum decimal 15, the hash will
	// have the maximum index 19 (20 bytes of SHA1) and we need 4 bytes.
	o := (h[19] & 15)

	var header uint32
	// Get 32 bit chunk from hash starting at the o
	r := bytes.NewReader(h[o : o+4])
	err = binary.Read(r, binary.BigEndian, &header)

	if err != nil {
		return "", err
	}
	// Ignore most significant bits as per RFC 4226.
	// Takes division from one million to generate a remainder less than < 7 digits
	h12 := (int(header) & 0x7fffffff) % 1000000

	// Converts number as a string
	otp := strconv.Itoa(int(h12))

	return prefix0(otp), nil
}

// GetTOTPToken 通过 secret 获取实时的token。每30秒刷新一次
func GetTOTPToken(secret string) (string, error) {
	// The TOTP token is just a HOTP token seeded with every 30 seconds.
	interval := time.Now().Unix() / 30
	return getHOTPToken(secret, interval)
}
