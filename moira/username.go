package moira

import (
	"strings"
)

const KERBEROS_SUFFIX = "@mit.edu"

type Username string // e.g. 'dmz' or 'ziedaniel1@gmail.com'
type Email string    // eg 'dmz@mit.edu' or 'ziedaniel1@gmail.com'

func UsernameFromEmail(email Email) Username {
	return Username(strings.TrimSuffix(strings.ToLower(string(email)), KERBEROS_SUFFIX))
}

func (u Username) IsKerberos() bool {
	return !strings.ContainsRune(string(u), '@')
}

func (u Username) Email() Email {
	if u.IsKerberos() {
		return Email(u + KERBEROS_SUFFIX)
	} else {
		return Email(u)
	}
}
