package v

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type StringValidator interface {
	Min(int) StringValidator
	Max(int) StringValidator
	Length(int) StringValidator
	Regex(*regexp.Regexp) StringValidator
	StartsWith(string) StringValidator
	EndsWith(string) StringValidator
	Includes(string) StringValidator
	Uppercase() StringValidator
	Lowercase() StringValidator
	Email() StringValidator
	URL() StringValidator
	HTTP() StringValidator
	IP() StringValidator
	IPv4() StringValidator
	IPv6() StringValidator
	CIDR() StringValidator
	CIDRv4() StringValidator
	CIDRv6() StringValidator
	Time(string) StringValidator
}

type stringV struct {
	*GenericV[string, StringValidator]
}

func (v *stringV) Min(min int) StringValidator {
	return v.Check(func(value string) error {
		if len(value) < min {
			return fmt.Errorf("value %v is shorter than %v", value, min)
		}
		return nil
	})
}

func (v *stringV) Max(max int) StringValidator {
	return v.Check(func(value string) error {
		if len(value) > max {
			return fmt.Errorf("value %v is longer than %v", value, max)
		}
		return nil
	})
}

func (v *stringV) Length(num int) StringValidator {
	return v.Check(func(value string) error {
		if len(value) != num {
			return fmt.Errorf("value %v is not %v characters long", value, num)
		}
		return nil
	})
}

func (v *stringV) Regex(regex *regexp.Regexp) StringValidator {
	return v.Check(func(value string) error {
		if !regex.MatchString(value) {
			return fmt.Errorf("value %v does not match regex %v", value, regex)
		}
		return nil
	})
}

func (v *stringV) StartsWith(prefix string) StringValidator {
	return v.Check(func(value string) error {
		if !strings.HasPrefix(value, prefix) {
			return fmt.Errorf("value %v does not start with %v", value, prefix)
		}
		return nil
	})
}

func (v *stringV) EndsWith(suffix string) StringValidator {
	return v.Check(func(value string) error {
		if !strings.HasSuffix(value, suffix) {
			return fmt.Errorf("value %v does not end with %v", value, suffix)
		}
		return nil
	})
}

func (v *stringV) Includes(substring string) StringValidator {
	return v.Check(func(value string) error {
		if !strings.Contains(value, substring) {
			return fmt.Errorf("value %v does not contain %v", value, substring)
		}
		return nil
	})
}

func (v *stringV) Lowercase() StringValidator {
	return v.Check(func(value string) error {
		if strings.ToLower(value) != value {
			return fmt.Errorf("value %v is not lowercase", value)
		}
		return nil
	})
}

func (v *stringV) Uppercase() StringValidator {
	return v.Check(func(value string) error {
		if strings.ToUpper(value) != value {
			return fmt.Errorf("value %v is not uppercase", value)
		}
		return nil
	})
}

func (v *stringV) Email() StringValidator {
	return v.Check(func(value string) error {
		if _, err := mail.ParseAddress(value); err != nil {
			return fmt.Errorf("value %v is not a valid email address", value)
		}
		return nil
	})
}

func (v *stringV) URL() StringValidator {
	return v.Check(func(value string) error {
		if _, err := url.Parse(value); err != nil {
			return fmt.Errorf("value %v is not a valid URL", value)
		}
		return nil
	})
}

func (v *stringV) HTTP() StringValidator {
	return v.Check(func(value string) error {
		if _, err := url.ParseRequestURI(value); err != nil {
			return fmt.Errorf("value %v is not a valid HTTP URL", value)
		}
		return nil
	})
}

func (v *stringV) IP() StringValidator {
	return v.Check(func(value string) error {
		if ip := net.ParseIP(value); ip == nil {
			return fmt.Errorf("value %v is not a valid IP address", value)
		}
		return nil
	})
}

func (v *stringV) IPv4() StringValidator {
	return v.Check(func(value string) error {
		if ip := net.ParseIP(value); ip == nil || ip.To4() == nil {
			return fmt.Errorf("value %v is not a valid IPv4 address", value)
		}
		return nil
	})
}

func (v *stringV) IPv6() StringValidator {
	return v.Check(func(value string) error {
		if ip := net.ParseIP(value); ip == nil || ip.To4() != nil {
			return fmt.Errorf("value %v is not a valid IPv6 address", value)
		}
		return nil
	})
}

func (v *stringV) CIDR() StringValidator {
	return v.Check(func(value string) error {
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("value %v is not a valid CIDR", value)
		}
		return nil
	})
}

func (v *stringV) CIDRv4() StringValidator {
	return v.Check(func(value string) error {
		if ip, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("value %v is not a valid CIDRv4", value)
		} else if ip.To4() == nil {
			return fmt.Errorf("value %v is not a valid CIDRv4", value)
		}
		return nil
	})
}

func (v *stringV) CIDRv6() StringValidator {
	return v.Check(func(value string) error {
		if ip, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("value %v is not a valid CIDRv6", value)
		} else if ip.To4() != nil {
			return fmt.Errorf("value %v is not a valid CIDRv6", value)
		}
		return nil
	})
}

func (v *stringV) Time(layout string) StringValidator {
	return v.Check(func(value string) error {
		if _, err := time.Parse(layout, value); err != nil {
			return fmt.Errorf("value %v is not a valid time", value)
		}
		return nil
	})
}

func String[Source any](grp *Set[Source], get func(Source) string) StringValidator {
	stringV := &stringV{}
	stringV.GenericV = NewGenericV[string, StringValidator](stringV)
	grp.validations = append(grp.validations, func(source Source) error {
		value := get(source)
		err := stringV.Validate(value)
		return err
	})
	return stringV
}
