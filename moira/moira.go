package moira

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"gopkg.in/ldap.v2"
)

// nfsgroup MUST match [a-z0-9-] (no LDAP quoting is done)
func GetMoiraNFSGroupMemberStrings(nfsgroup string) ([]string, error) {
	l, err := ldap.DialTLS("tcp", "ldap.mit.edu:636", &tls.Config{ServerName: "ldap.mit.edu"})
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer l.Close()

	// ldapsearch -LLL -x -H ldap://ldap.mit.edu:389 -b "ou=lists,ou=moira,dc=mit,dc=edu" "cn=${nfsgroup}" member
	sr, err := l.Search(ldap.NewSearchRequest(
		"ou=lists,ou=moira,dc=mit,dc=edu",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases /*sizelimit*/, 0 /*timelimit*/, 0 /*typesonly*/, false,
		"(cn="+nfsgroup+")",
		[]string{"member"},
		/*"control"*/ nil,
	))
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if l := len(sr.Entries); l != 1 {
		return nil, fmt.Errorf("expected exactly one list, found %d", l)
	}
	return sr.Entries[0].GetAttributeValues("member"), nil
}

func extractPart(prefix, suffix, str string) (string, bool) {
	// the first condition is necessary in case of potentially overlapping prefix & suffix
	if len(str) >= len(prefix)+len(suffix) && strings.HasPrefix(str, prefix) && strings.HasSuffix(str, suffix) {
		return str[len(prefix) : len(str)-len(suffix)], true
	} else {
		return "", false
	}
}

func GetMoiraNFSGroupMembers(nfsgroup string) ([]Username, error) {
	members, err := GetMoiraNFSGroupMemberStrings(nfsgroup)
	if err != nil {
		return nil, err
	}

	usernames := []Username{}
	for _, member := range members {
		if kerberos, ok := extractPart("uid=", ",OU=users,OU=moira,dc=MIT,dc=EDU", member); ok {
			usernames = append(usernames, UsernameFromKerberos(kerberos))
		} else if email, ok := extractPart("cn=", ",OU=strings,OU=moira,dc=MIT,dc=EDU", member); ok {
			usernames = append(usernames, UsernameFromEmail(Email(email)))
		}
		// ignore other entries
	}

	return usernames, nil
}

func IsAuthorized(authorize string, user Username) error {
	users, err := GetMoiraNFSGroupMembers(authorize)
	if err != nil {
		return err
	}

	for _, u := range users {
		if u == user {
			return nil
		}
	}

	return fmt.Errorf("authenticated as %q, but not authorized because not on moira list %q", user.Email(), authorize)
}
