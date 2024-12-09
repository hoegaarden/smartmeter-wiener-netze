package smartmeter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/antchfx/htmlquery"
)

type Client struct {
	*wienerStadtwerkeClient
	*wienerNetzeClient
}

const (
	AuthURL     = "https://log.wien/auth/realms/logwien/protocol/openid-connect/"
	RedirectURL = "https://smartmeter-web.wienernetze.at/"

	ClientID = "wn-smartmeter"

	userAgent = "Please document your APIs! Thanks!"
)

func Login(username, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("setting up cookie jar: %w", err)
	}

	httpClient := httpClientWithReqHeaders(&http.Client{Jar: jar}, func() map[string]string {
		return map[string]string{"User-Agent": userAgent}
	})

	loginURL := AuthURL + "auth?" + url.Values{
		"client_id":     []string{ClientID},
		"redirect_uri":  []string{RedirectURL},
		"response_mode": []string{"fragment"},
		"response_type": []string{"code"},
		"scope":         []string{"openid"},
	}.Encode()

	loginPageRes, err := httpClient.Get(loginURL)
	if err != nil {
		return nil, fmt.Errorf("requesting login page: %w", err)
	}
	defer loginPageRes.Body.Close()

	userResp, err := fillForm(httpClient, loginPageRes, "username", username)
	if err != nil {
		return nil, fmt.Errorf("filling username form: %w", err)
	}
	defer userResp.Body.Close()

	passwordResp, err := fillForm(httpClient, userResp, "password", password)
	if err != nil {
		return nil, fmt.Errorf("filling password form: %w", err)
	}
	defer passwordResp.Body.Close()

	redirectLoc, err := passwordResp.Location()
	if err != nil {
		return nil, fmt.Errorf("getting redirect URL (wrong credentials?): %w", err)
	}

	fragment, err := url.ParseQuery(redirectLoc.Fragment)
	if err != nil {
		return nil, fmt.Errorf("parsing redirect URL's fragment: %w", err)
	}

	code := fragment.Get("code")
	if code == "" {
		return nil, fmt.Errorf("code not found in redirect URL's fragment")
	}

	tokenRes, err := httpClient.PostForm(AuthURL+"token", url.Values{
		"code":         []string{code},
		"grant_type":   []string{"authorization_code"},
		"client_id":    []string{ClientID},
		"redirect_uri": []string{RedirectURL},
	})
	if err != nil {
		return nil, fmt.Errorf("requesting login page: %w", err)
	}
	defer tokenRes.Body.Close()

	tokenData := &tokenData{}
	if err := json.NewDecoder(tokenRes.Body).Decode(tokenData); err != nil {
		return nil, fmt.Errorf("decoding token data: %w", err)
	}

	wstwClient := &wienerStadtwerkeClient{
		httpClient: httpClientWithReqHeaders(httpClient, func() map[string]string {
			return map[string]string{
				"Authorization":    "Bearer " + tokenData.AccessToken,
				"X-Gateway-APIKey": GatewayAPIKeyWienerStadtwerke,
			}
		}),
	}

	wnClient := &wienerNetzeClient{
		httpClient: httpClientWithReqHeaders(httpClient, func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer " + tokenData.AccessToken,
			}
		}),
	}

	return &Client{
		wienerStadtwerkeClient: wstwClient,
		wienerNetzeClient:      wnClient,
	}, nil
}

func fillForm(client *http.Client, lastResp *http.Response, paramName, paramValue string) (*http.Response, error) {
	doc, err := htmlquery.Parse(lastResp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing form page: %w", err)
	}

	formAction, err := htmlquery.Query(doc, "//form/@action")
	if err != nil || formAction == nil {
		if err == nil {
			err = fmt.Errorf("form action not found")
		}
		return nil, fmt.Errorf("extracting login form action: %w", err)
	}
	formActionURL := htmlquery.InnerText(formAction)
	if formActionURL == "" {
		return nil, fmt.Errorf("login action URL is empty")
	}

	formActionRes, err := withoutRedirect(client).PostForm(formActionURL, url.Values{
		paramName: []string{paramValue},
	})
	if err != nil {
		return nil, fmt.Errorf("submitting login form: %w", err)
	}

	return formActionRes, nil
}

type tokenData struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

type headersFunc = func() map[string]string

func httpClientWithReqHeaders(orgClient *http.Client, headersFunc headersFunc) *http.Client {
	inner := orgClient.Transport
	if inner == nil {
		inner = http.DefaultTransport
	}

	return withTransport(orgClient, &reqHeaderFuncRoundTripper{
		headersFunc: headersFunc,
		inner:       inner,
	})
}

type reqHeaderFuncRoundTripper struct {
	inner       http.RoundTripper
	headersFunc headersFunc
}

func (rt *reqHeaderFuncRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.headersFunc != nil {
		for k, v := range rt.headersFunc() {
			req.Header.Add(k, v)
		}
	}

	return rt.inner.RoundTrip(req)
}

func withoutRedirect(orgClient *http.Client) *http.Client {
	newClient := &http.Client{}
	if orgClient != nil {
		*newClient = *orgClient
	}
	newClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return newClient
}

func withTransport(orgClient *http.Client, transport http.RoundTripper) *http.Client {
	newClient := &http.Client{}
	if orgClient != nil {
		*newClient = *orgClient
	}
	newClient.Transport = transport
	return newClient
}
