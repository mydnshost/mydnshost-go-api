// Package mydnshost_go_api provides an API client for MyDNSHost.
package mydnshost_go_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiHost = "api.mydnshost.co.uk"
const apiVersion = "1.0"

type apiResponse struct {
	ResponseId string            `json:"respid"`
	Method     string            `json:"method"`
	Error      *string           `json:"error"`
	ErrorData  map[string]string `json:"errorData"`
	Response   *json.RawMessage  `json:"response"`
}

type apiRequest struct {
	Data interface{} `json:"data"`
}

// ClientAuthenticator adds authentication headers to outgoing requests to the API.
type ClientAuthenticator interface {
	AddHeaders(r *http.Request)
}

// Client is the client API for communicating with MyDNSHost. For most requests it will require a ClientAuthenticator
// to be provided that can supply credentials to the API.
type Client struct {
	Authenticator ClientAuthenticator
}

// PingResponse is the API response to a ping request, containing the time the request was sent.
type PingResponse struct {
	Time string `json:"time"`
}

// Ping sends a ping request to the API. It does not require authentication.
func (c *Client) Ping(ctx context.Context) (*PingResponse, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("ping/%d", time.Now().Unix()), nil)
	if err != nil {
		return nil, err
	}

	response := &PingResponse{}
	return response, json.Unmarshal(*res.Response, response)
}

// UserDataResponse is the API response to a user data request, describing the current user and access levels.
type UserDataResponse struct {
	User struct {
		Id       string `json:"id"`
		Email    string `json:"email"`
		RealName string `json:"realname"`
	} `json:"user"`
	Access struct {
		DomainsRead  bool `json:"domains_read"`
		DomainsWrite bool `json:"domains_write"`
		UserRead     bool `json:"user_read"`
		UserWrite    bool `json:"user_write"`
	} `json:"access"`
}

// UserData sends a request for details on the current user.
func (c *Client) UserData(ctx context.Context) (*UserDataResponse, error) {
	res, err := c.request(ctx, http.MethodGet, "userdata", nil)
	if err != nil {
		return nil, err
	}

	response := &UserDataResponse{}
	return response, json.Unmarshal(*res.Response, response)
}

// AccessLevel describes a level of access to a domain.
type AccessLevel string

const (
	LevelOwner AccessLevel = "owner"
	LevelAdmin AccessLevel = "admin"
	LevelWrite AccessLevel = "write"
	LevelRead AccessLevel = "read"
	LevelNone AccessLevel = "none"
)

// Domains lists all domains accessible by the current user, and gives the access level to each.
func (c *Client) Domains(ctx context.Context) (map[string]AccessLevel, error) {
	res, err := c.request(ctx, http.MethodGet, "domains", nil)
	if err != nil {
		return nil, err
	}

	response := make(map[string]AccessLevel)
	return response, json.Unmarshal(*res.Response, &response)
}

// Record contains the basic details of a DNS record.
type Record struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Content  string `json:"content,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Disabled *bool  `json:"disabled,omitempty"`
}

// ExistingRecord is a Record that has been stored by MyDNSHost and thus has an ID and change history.
type ExistingRecord struct {
	Record
	Id        int  `json:"id"`
	ChangedAt int  `json:"changed_at,omitempty"`
	ChangedBy *int `json:"changed_by,omitempty"`
}

// ChangedRecord is a record that has been modified by a call to ModifyRecords.
type ChangedRecord struct {
	ExistingRecord
	Updated bool `json:"updated,omitempty"`
	Deleted bool `json:"deleted,omitempty"`
}

// RecordsResponse lists all records for a domain, as well as the NS and SOA data for it.
type RecordsResponse struct {
	Records []ExistingRecord `json:"records"`
	HasNS   bool             `json:"hasNS"`
	Soa     struct {
		PrimaryNS    string `json:"primaryNS"`
		AdminAddress string `json:"adminAddress"`
		Serial       uint64 `json:"serial"`
		Refresh      uint64 `json:"refresh"`
		Retry        uint64 `json:"retry"`
		Expire       uint64 `json:"expire"`
		MinTTL       uint64 `json:"min_ttl"`
	} `json:"soa"`
}

// Records retrieves all records associated with the specified domain.
func (c *Client) Records(ctx context.Context, domain string) (*RecordsResponse, error) {
	res, err := c.request(ctx, http.MethodGet, fmt.Sprintf("domains/%s/records", domain), nil)
	if err != nil {
		return nil, err
	}

	response := &RecordsResponse{}
	return response, json.Unmarshal(*res.Response, response)
}

// RecordOperation is an operation performed on a record when calling ModifyRecords.
type RecordOperation json.RawMessage

// ModifyRecord changes an existing record with the given ID. Any field populated in the record will be updated.
func ModifyRecord(id int, record Record) RecordOperation {
	res, _ := json.Marshal(ExistingRecord{
		Record: record,
		Id:     id,
	})
	return res
}

// DeleteRecord deletes an existing record with the given ID.
func DeleteRecord(id int) RecordOperation {
	res, _ := json.Marshal(struct {
		Id     int  `json:"id"`
		Delete bool `json:"delete"`
	}{
		Id:     id,
		Delete: true,
	})
	return res
}

// CreateRecord creates a new record. All non-pointer fields of the given Record must be supplied.
func CreateRecord(record Record) RecordOperation {
	res, _ := json.Marshal(record)
	return res
}

// ModifyRecordsResponse lists all changed records as a result of a ModifyRecords request.
type ModifyRecordsResponse struct {
	Serial  uint64          `json:"serial"`
	Changed []ChangedRecord `json:"changed"`
}

// ModifyRecords performs one or more operations on the records of a domain, including adding, modifying and deleting.
func (c *Client) ModifyRecords(ctx context.Context, domain string, operations ...RecordOperation) (*ModifyRecordsResponse, error) {
	records := make([]json.RawMessage, len(operations))
	for i := range operations {
		records[i] = json.RawMessage(operations[i])
	}

	r := apiRequest{
		Data: struct {
			Records []json.RawMessage `json:"records"`
		}{
			Records: records,
		},
	}

	res, err := c.request(ctx, http.MethodPost, fmt.Sprintf("domains/%s/records", domain), r)
	if err != nil {
		return nil, err
	}

	response := &ModifyRecordsResponse{}
	return response, json.Unmarshal(*res.Response, response)
}

func (c *Client) request(ctx context.Context, method string, route string, body interface{}) (*apiResponse, error) {
	var reader io.Reader = nil
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("https://%s/%s/%s", apiHost, apiVersion, route), reader)
	if err != nil {
		return nil, err
	}

	if c.Authenticator != nil {
		c.Authenticator.AddHeaders(req)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	response := &apiResponse{}
	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s", *response.Error)
	}

	return response, nil
}
