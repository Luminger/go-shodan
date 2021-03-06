package shodan

const (
	profilePath = "/account/profile"
)

// Profile holds account's information
type Profile struct {
	Member  bool   `json:"member"`
	Credits int    `json:"credits"`
	Name    string `json:"display_name"`
	Created string `json:"created"`
}

// GetAccountProfile returns information about the Shodan account linked to the API key
func (c *Client) GetAccountProfile() (*Profile, error) {
	url := c.buildBaseURL(profilePath, nil)

	var profile Profile
	err := c.executeRequest("GET", url, &profile, nil)

	return &profile, err
}
