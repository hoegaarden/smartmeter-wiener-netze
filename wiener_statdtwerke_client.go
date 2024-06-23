package smartmeter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// some magic API Key :shrug:
	GatewayAPIKeyWienerStadtwerke = "afb0be74-6455-44f5-a34d-6994223020ba"

	APIBaseWienerStadtwerke = "https://api.wstw.at/gateway/WN_SMART_METER_PORTAL_API_B2C/1.0/"
)

type wienerStadtwerkeClient struct {
	httpClient *http.Client
}

func (c *wienerStadtwerkeClient) BaseInfo() (BaseInfo, error) {
	url := APIBaseWienerStadtwerke + "zaehlpunkt/baseInformation"
	baseInfo := BaseInfo{}

	res, err := c.httpClient.Get(url)
	if err != nil {
		return baseInfo, fmt.Errorf("gathering base information: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&baseInfo); err != nil {
		return baseInfo, fmt.Errorf("decoding base information: %w", err)
	}

	return baseInfo, nil
}

func (c *wienerStadtwerkeClient) Meters() ([]Meter, error) {
	url := APIBaseWienerStadtwerke + "zaehlpunkte"

	res, err := c.httpClient.Get(url)
	if err != nil {
		return []Meter{}, fmt.Errorf("gathering meters: %w", err)
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	meters := []Meter{}
	i := 0
	for {
		res := []metersResponse{}

		err := dec.Decode(&res)
		if err == io.EOF {
			break
		}
		if err != nil {
			return []Meter{}, fmt.Errorf("decoding meter response %d: %w", i, err)
		}

		for _, r := range res {
			for _, meter := range r.Meters {
				meter.CustomerID = r.CustomerID
				meters = append(meters, meter)
			}
		}

		i += 1
	}

	return meters, nil
}

type BaseInfo struct {
	HasSmartMeter         bool          `json:"hasSmartMeter"`
	IsDeleted             bool          `json:"isDeleted"`
	DataDeletionTimestamp *string       `json:"dataDeletionTimestampUTC"`
	Meter                 BaseInfoMeter `json:"zaehlpunkt"`
}

type BaseInfoMeter struct {
	Name    string `json:"zaehlpunktName"`
	ID      string `json:"zaehlpunktnummer"`
	Type    string `json:"zaehlpunktAnlagentyp"`
	Adress  string `json:"adresse"`
	ZIPCode string `json:"postleitzahl"`
}

type metersResponse struct {
	Name       string  `json:"bezeichnung"`
	CustomerID string  `json:"geschaeftspartner"`
	Meters     []Meter `json:"zaehlpunkte"`
}

type Meter struct {
	ID                      string  `json:"zaehlpunktnummer"`
	CustomerID              string  `json:"geschaeftspartner"`
	Label                   string  `json:"customLabel"`
	EquipmentID             string  `json:"equipmentNumber"`
	DeviceID                string  `json:"geraetNumber"`
	IsSmartMeter            bool    `json:"isSmartMeter"`
	IsDefault               bool    `json:"isDefault"`
	IsActive                bool    `json:"isActive"`
	IsDataDeleted           bool    `json:"isDataDeleted"`
	IsSmartMeterMarketReady bool    `json:"isSmartMeterMarketReady"`
	DataDeletionTimestamp   *string `json:"dataDeletionTimestampUTC"`
	Location                struct {
		VStelle           string `json:"vstelle"` // no idea what that is
		Street            string `json:"strasse"`
		StreetNumber      string `json:"hausnummer"`
		MeterStreetNumber string `json:"anlagehausnummer"`
		ZIPCode           string `json:"postleitzahl"`
		City              string `json:"ort"`
		Latitude          string `json:"breitengrad"`
		Longitude         string `json:"laengengrad"`
	} `json:"verbrauchsstelle"`
	Installation struct {
		Type string `json:"typ"`
	} `json:"anlage"`
	Contracts []struct {
		MoveInDate  *string `json:"einzugsdatum"` // null or '2024-04-04'
		MoveOutDate *string `json:"auszugsdatum"` // null or '2024-04-04'
	} `json:"vertraege"`
	IdexStatus struct {
		Granularity struct {
			Status       string `json:"status"`
			CanBeChanged bool   `json:"canBeChanged"`
		} `json:"granularity"`
		CustomerInterface struct {
			Status       string `json:"status"`
			CanBeChanged bool   `json:"canBeChanged"`
		} `json:"customerInterface"`
		Display struct {
			IsLocked     bool `json:"isLocked"`
			CanBeChanged bool `json:"canBeChanged"`
		} `json:"display"`
		DisplayProfile struct {
			DisplayProfile string `json:"displayProfile"`
			CanBeChanged   bool   `json:"canBeChanged"`
		} `json:"displayProfile"`
	} `json:"idexStatus"` // is this really 'idex' or 'index'?
	OptOutDetails struct {
		IsOptOut bool `json:"isOptOut"`
	} `json:"optOutDetails"`
	SharingInfo struct {
		IsOwner bool `json:"isOwner"`
	} `json:"zpSharingInfo"`
}
