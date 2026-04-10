package runtime

import "net/url"

// Pythonista3 produces URLs for the Pythonista 3 iOS app.
// The URL scheme pythonista3://importscript?url=<script-url>&name=<filename>
// instructs the app to import and run the remote script.
type Pythonista3 struct{}

// QRCodeURL converts a raw script URL into a Pythonista 3 deep-link URL.
func (p *Pythonista3) QRCodeURL(publicURL string) string {
	u := url.URL{
		Scheme: "pythonista3",
		Host:   "importscript",
		RawQuery: url.Values{
			"url": []string{publicURL},
		}.Encode(),
	}
	return u.String()
}
