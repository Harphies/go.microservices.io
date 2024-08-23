package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"sync"
	"time"
)

type IPInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

type CountryInfo struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	Cca2 string `json:"cca2"`
}

var (
	countryCache      = make(map[string]string)
	countryCacheMutex sync.RWMutex
)

func GetUserLocationFromIPAddress(r *http.Request, logger *zap.Logger, countriesAPIEndpoint, ipInfoEndpoint string) (string, error) {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	fmt.Println("IP:", forwardedFor)
	ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	if ip == "" {
		ip = "31.49.235.239" // find a suitable default
	}
	location, err := getLocationFromIP(ip, ipInfoEndpoint, logger)
	if err != nil {
		return "", err
	}
	country, err := getCountryName(location.Country, countriesAPIEndpoint, logger)
	userLocation := fmt.Sprintf("%s, %s, %s", location.City, location.Region, country)
	return userLocation, err
}

func getLocationFromIP(ip, ipInfoEndpoint string, logger *zap.Logger) (*IPInfo, error) {
	ipInfoUrl := fmt.Sprintf("%s/%s/json", ipInfoEndpoint, ip)
	fmt.Println("ipInfoUrl:", ipInfoUrl)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := HTTPRequest(ctx, logger, http.MethodGet, ipInfoUrl, "", nil, nil, nil)
	logger.Error("getLocationFromIP()", zap.Error(err))
	if err != nil {
		return nil, err
	}

	var ipInfo IPInfo
	err = json.Unmarshal(resp, &ipInfo)
	if err != nil {
		return nil, err
	}

	return &ipInfo, nil
}

func getCountryName(countryCode, countriesAPIEndpoint string, logger *zap.Logger) (string, error) {
	// check the in-memory cache for all countries fetched
	countryCacheMutex.RLock()
	name, ok := countryCache[countryCode]
	countryCacheMutex.RUnlock()

	if ok {
		return name, nil
	}

	// If not in cache, fetch individually
	endpoint := fmt.Sprintf("%s/alpha/%s", countriesAPIEndpoint, countryCode)
	queryParams := map[string]string{
		"fields": "name",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := HTTPRequest(ctx, logger, http.MethodGet, endpoint, "", "", queryParams, nil)

	var countryInfo CountryInfo
	err = json.Unmarshal(resp, &countryInfo)
	if err != nil {
		return countryCode, err
	}

	if countryInfo.Name.Common != "" {
		countryCacheMutex.Lock()
		countryCache[countryCode] = countryInfo.Name.Common
		countryCacheMutex.Unlock()
		return countryInfo.Name.Common, nil
	}

	return countryCode, nil
}

// LoadAllCountries get list of all countries and their name and put in-memory(cache)
func LoadAllCountries(countriesAPIEndpoint string, logger *zap.Logger) error {
	endpoint := fmt.Sprintf("%s/all", countriesAPIEndpoint)
	queryParams := map[string]string{
		"fields": "name,cca2",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := HTTPRequest(ctx, logger, http.MethodGet, endpoint, "", "", queryParams, nil)

	var countries []CountryInfo
	err = json.Unmarshal(resp, &countries)
	if err != nil {
		return err
	}

	newCache := make(map[string]string)
	for _, country := range countries {
		newCache[country.Cca2] = country.Name.Common
	}

	countryCacheMutex.Lock()
	countryCache = newCache
	countryCacheMutex.Unlock()

	return nil
}
