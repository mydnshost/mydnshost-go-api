package mydnshost_go_api

import "net/http"

// ApiKeyAuthenticator authenticates using a username (e-mail address) and API key
type ApiKeyAuthenticator struct {
	User string `json:"user"`
	Key  string `json:"key"`
}

func (a *ApiKeyAuthenticator) AddHeaders(r *http.Request) {
	r.Header["X-API-User"] = []string{a.User}
	r.Header["X-API-Key"] = []string{a.Key}
}

// DomainKeyAuthenticator authenticates using a domain name and a domain-specific API key
type DomainKeyAuthenticator struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
}

func (a *DomainKeyAuthenticator) AddHeaders(r *http.Request) {
	r.Header["X-Domain"] = []string{a.Domain}
	r.Header["X-Domain-Key"] = []string{a.Key}
}
