package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	gopostalExpand "github.com/openvenues/gopostal/expand"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
	log.Logger = zerolog.Nop()
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestStringToBool(t *testing.T) {
	assert.True(t, stringToBool("true"))
	assert.True(t, stringToBool("1"))
	assert.True(t, stringToBool("t"))

	assert.False(t, stringToBool("false"))
	assert.False(t, stringToBool("0"))
	assert.False(t, stringToBool("invalid_string"))
	assert.False(t, stringToBool(""))
}

func TestParseAddressComponents(t *testing.T) {
	t.Run("Valid Component", func(t *testing.T) {
		queryParams := url.Values{
			"address_name": []string{"true"},
		}

		components, found := parseAddressComponents(queryParams)
		assert.True(t, found)
		// Explicitly cast to uint16
		assert.Equal(t, uint16(gopostalExpand.AddressName), components)
	})

	t.Run("Multiple Components", func(t *testing.T) {
		queryParams := url.Values{
			"address_street":  []string{"true"},
			"address_po_box":  []string{"1"},
			"address_invalid": []string{"true"}, // Should be ignored
		}

		components, found := parseAddressComponents(queryParams)
		assert.True(t, found)
		// Explicitly cast the combined bitmask to uint16
		expected := uint16(gopostalExpand.AddressStreet | gopostalExpand.AddressPoBox)
		assert.Equal(t, expected, components)
	})

	t.Run("False Parameter Evaluated Correctly", func(t *testing.T) {
		queryParams := url.Values{
			"address_street": []string{"false"},
		}

		components, found := parseAddressComponents(queryParams)
		assert.False(t, found)
		// Explicitly cast to uint16
		assert.Equal(t, uint16(gopostalExpand.AddressNone), components)
	})
}

func TestParseRoute(t *testing.T) {
	router := SetupRouter()

	t.Run("Basic Parsing", func(t *testing.T) {
		w := httptest.NewRecorder()
		// URL encode the address from your README
		address := url.QueryEscape("781 Franklin Ave Crown Heights Brooklyn NY 11216 USA")
		req, _ := http.NewRequest(http.MethodGet, "/parse?address="+address, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// The parse endpoint returns an array of label/value objects
		var response []map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		assert.NotEmpty(t, response)

		// Create a quick lookup map to easily assert the parsed components
		parsedMap := make(map[string]string)
		for _, component := range response {
			parsedMap[component["label"]] = component["value"]
		}

		// Verify libpostal correctly identified the parts of the address
		assert.Equal(t, "781", parsedMap["house_number"])
		assert.Equal(t, "franklin ave", parsedMap["road"])
		assert.Equal(t, "11216", parsedMap["postcode"])
		assert.Equal(t, "usa", parsedMap["country"])
	})

	t.Run("Parsing with Language and Country Parameters", func(t *testing.T) {
		w := httptest.NewRecorder()

		query := "?address=" + url.QueryEscape("Quatre-vingt-douze Ave des Champs-Élysées") + "&language=fr&country=fr"
		req, _ := http.NewRequest(http.MethodGet, "/parse"+query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		assert.NotEmpty(t, response)

		parsedMap := make(map[string]string)
		for _, component := range response {
			parsedMap[component["label"]] = component["value"]
		}

		// Verify it parsed the French address correctly
		assert.Equal(t, "quatre-vingt-douze", parsedMap["house_number"])
		assert.Equal(t, "ave des champs-élysées", parsedMap["road"])
	})

	t.Run("Empty Address", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/parse?address=", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		// Passing an empty address should safely return an empty array
		assert.Empty(t, response)
	})
}

func TestExpandRoute(t *testing.T) {
	router := SetupRouter()

	t.Run("Basic Expansion", func(t *testing.T) {
		w := httptest.NewRecorder()
		// URL encode the address to simulate a real client request
		address := url.QueryEscape("781 Franklin Ave")
		req, _ := http.NewRequest(http.MethodGet, "/expand?address="+address, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		assert.NotEmpty(t, response)

		// libpostal should normalize and expand "Ave" to "avenue"
		assert.Contains(t, response, "781 franklin avenue")
	})

	t.Run("Expansion with Parameters", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Test with extra parameters like turning off lowercase
		// (which is true by default in libpostal)
		query := "?address=" + url.QueryEscape("781 Franklin Ave") + "&lowercase=false"
		req, _ := http.NewRequest(http.MethodGet, "/expand"+query, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		assert.NotEmpty(t, response)

		// Because lowercase=false, the output should maintain capitalization
		assert.Contains(t, response, "781 Franklin Ave")
	})

	t.Run("Empty Address", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/expand?address=", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []string
		err := json.Unmarshal(w.Body.Bytes(), &response)

		assert.Nil(t, err)
		// Passing an empty address should safely return an empty JSON array
		assert.Empty(t, response)
	})
}

func TestHealthCheckRoute(t *testing.T) {
	router := SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)

	assert.Nil(t, err)
	assert.Equal(t, "ok", response["status"])
}

func TestRootRoute(t *testing.T) {
	router := SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "version")
}
