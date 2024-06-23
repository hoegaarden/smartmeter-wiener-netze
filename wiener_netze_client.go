package smartmeter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	APIBaseWienerNetze = "https://service.wienernetze.at"
)

type wienerNetzeClient struct {
	httpClient *http.Client
}

func (c *wienerNetzeClient) Profile() (Profile, error) {
	url := APIBaseWienerNetze + "/rest/smp/1.0/w/user/profile"
	profile := Profile{}

	res, err := c.httpClient.Get(url)
	if err != nil {
		return profile, fmt.Errorf("gathering profile: %w", err)
	}
	defer res.Body.Close()

	allProfiles := []Profile{}
	dec := json.NewDecoder(res.Body)
	i := 0
	for {
		res := Profile{}

		err := dec.Decode(&res)
		if err == io.EOF {
			break
		}
		if err != nil {
			return profile, fmt.Errorf("decoding profile response %d: %w", i, err)
		}

		allProfiles = append(allProfiles, res)
		i += 1
	}

	if l := len(allProfiles); l != 1 {
		return profile, fmt.Errorf("expected to find 1 profile, but found %d", l)
	}

	return allProfiles[0], nil
}

func (c *wienerNetzeClient) Export(customerID, meterID string, start, end time.Time) (Export, error) {
	startStr := start.Format(time.RFC3339)
	endStr := end.Format(time.RFC3339)
	url := APIBaseWienerNetze + "/sm/api/user/messwerte/bewegungsdaten?" + url.Values{
		"rolle":             []string{"V002"}, // magic string :shrug:
		"geschaeftspartner": []string{customerID},
		"zaehlpunktnummer":  []string{meterID},
		"zeitpunktVon":      []string{startStr},
		"zeitpunktBis":      []string{endStr},
	}.Encode()

	res, err := c.httpClient.Get(url)
	if err != nil {
		return Export{}, fmt.Errorf("requesting data export: %w", err)
	}
	defer res.Body.Close()

	var data Export
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return Export{}, fmt.Errorf("decoding data export: %w", err)
	}

	return data, nil
}

type Export struct {
	Descriptor struct {
		CustomerID  string `json:"geschaeftspartnernummer"`
		MeterID     string `json:"zaehlpunktnummer"`
		Role        string `json:"rolle"`
		Aggregation string `json:"aggregat"`
		Granularity string `json:"granularitaet"`
		Unit        string `json:"einheit"`
	} `json:"descriptor"`
	Values []struct {
		Value     float64         `json:"wert"`
		From      RFC3339DateTime `json:"zeitpunktVon"`
		To        RFC3339DateTime `json:"zeitpunktBis"`
		Estimated bool            `json:"geschaetzt"`
	} `json:"values"`
}

type RFC3339DateTime struct {
	time.Time
}

func (dt *RFC3339DateTime) UnmarshalJSON(b []byte) error {
	date, err := time.Parse(`"`+time.RFC3339+`"`, string(b))
	if err != nil {
		return err
	}
	dt.Time = date
	return nil
}

type Profile struct {
	ID           int64   `json:"id"`
	Salutation   string  `json:"salutation"`
	FirstName    string  `json:"firstname"`
	LastName     string  `json:"lastname"`
	Email        string  `json:"email"`
	Approval     *string `json:"zustimmung"` // not sure about the type of this
	Registration struct {
		Key         string `json:"registrationKey"`
		MeterID     string `json:"zaehlpunkt"`
		Status      string `json:"status"`
		CompletedAt string `json:"completedAt"`
	} `json:"registration"`
	DefaultCostumerRegistration struct {
		ID          int64  `json:"id"`
		Key         string `json:"registrationKey"`
		Status      string `json:"status"`
		CustomerID  string `json:"geschaeftspartner"`
		CompletedAt string `json:"completedAt"`
	} `json:"defaultGeschaeftspartnerRegistration"`
}
