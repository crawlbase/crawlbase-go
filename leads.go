package crawlbase

import "context"

// LeadsAPI is the (legacy) domain-scoped email-extraction client.
// Closed to new sign-ups since Oct 2024 but still operational for
// existing customers. New integrations should use the email-extractor
// scraper on the Crawling API instead.
//
// See https://crawlbase.com/docs/leads-api for parameters.
type LeadsAPI struct {
	*baseClient
}

// NewLeadsAPI constructs a Leads API client with the given token.
// Returns ErrTokenRequired on an empty token.
func NewLeadsAPI(token string) (*LeadsAPI, error) {
	bc, err := newBaseClient(token, "leads")
	if err != nil {
		return nil, err
	}
	return &LeadsAPI{baseClient: bc}, nil
}

// GetFromDomain extracts the public email addresses associated with the
// given domain. Pass nil for options to send only the domain.
func (a *LeadsAPI) GetFromDomain(domain string, options map[string]string) (*Response, error) {
	return a.GetFromDomainWithContext(context.Background(), domain, options)
}

// GetFromDomainWithContext is GetFromDomain with cancellation / deadline
// / trace propagation.
func (a *LeadsAPI) GetFromDomainWithContext(ctx context.Context, domain string, options map[string]string) (*Response, error) {
	if options == nil {
		options = map[string]string{}
	}
	options["domain"] = domain
	// Pass empty targetURL — the Leads API uses the `domain` param, not `url`.
	return a.request(ctx, "GET", "", options, nil, "")
}
