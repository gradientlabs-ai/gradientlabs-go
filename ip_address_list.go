package client

import (
	"context"
	"encoding/json"
	"net/http"
)

// ListIPAddressesResponse contains the IP address ranges Gradient Labs uses.
type ListIPAddressesResponse struct {
	// API is the IP address ranges the API is served from, in CIDR format.
	API []string `json:"api"`

	// Egress is the IP address ranges Gradient Labs sends outbound requests
	// from, such as calls to your HTTP tools and resource sources, in CIDR
	// format.
	Egress []string `json:"egress"`
}

// ListIPAddresses lists the IP address ranges Gradient Labs uses, in CIDR
// format.
func (c *Client) ListIPAddresses(ctx context.Context) (*ListIPAddressesResponse, error) {
	rsp, err := c.makeRequest(ctx, http.MethodGet, "ip-addresses", nil)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if err := responseError(rsp); err != nil {
		return nil, err
	}

	var ips ListIPAddressesResponse
	if err := json.NewDecoder(rsp.Body).Decode(&ips); err != nil {
		return nil, err
	}
	return &ips, nil
}
