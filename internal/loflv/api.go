package loflv

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type apiResponse struct {
	HTML string `json:"html"`
	Msg  string `json:"msg"`
}

// unwrapResponse extracts HTML content from the JSON API envelope.
// Tries "html" key first, falls back to "msg".
func unwrapResponse(body []byte) (string, error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode API response: %w", err)
	}
	if resp.HTML != "" {
		return resp.HTML, nil
	}
	if resp.Msg != "" {
		return resp.Msg, nil
	}
	return "", fmt.Errorf("empty API response")
}

// formData builds POST form values for a results request.
// The API expects only two fields: txt and p, where p bundles all query params.
// view: 1 = single event results, 99 = kopvērtējums (multi-day overall).
func formData(eventID int, view int) url.Values {
	p := "ev=" + strconv.Itoa(eventID) + "&t=1&r=" + strconv.Itoa(view) + "&d=0"
	return url.Values{
		"txt": {"1"},
		"p":   {p},
	}
}
