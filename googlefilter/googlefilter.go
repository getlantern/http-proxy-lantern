// This package filters incoming requests for requests to Google search domains
// as well as associated redirects to a Google captcha page. It then stores
// that data in an effort to diagnose what triggers the captcha.

package googlefilter

import (
	"net/http"
	"regexp"

	"github.com/getlantern/http-proxy-lantern/v2/zerologger"
	"github.com/getlantern/ops"
	"github.com/getlantern/proxy/filters"
)

var (
	log = zerologger.Named("googlefilter")

	// DefaultSearchRegex is the default regex for google search domains.
	DefaultSearchRegex = `^(www.)?google\..+`

	// DefaultCaptchaRegex is the default regex for the redirect from Google
	// search domains to a captcha page.
	DefaultCaptchaRegex = `^ipv4.google\..+`
)

// googleFilter filters requests for Google search domains and associated
// redirects to a Google captcha page.
type googleFilter struct {
	searchRegex  *regexp.Regexp
	captchaRegex *regexp.Regexp
}

// New creates a new filter for checking for redirects from Google search to a
// captcha.
func New(searchRegex string, captchaRegex string) filters.Filter {
	return &googleFilter{
		searchRegex:  regexp.MustCompile(searchRegex),
		captchaRegex: regexp.MustCompile(captchaRegex),
	}
}

func (f *googleFilter) Apply(ctx filters.Context, req *http.Request, next filters.Next) (*http.Response, filters.Context, error) {
	f.recordActivity(req)
	return next(ctx, req)
}

func (f *googleFilter) recordActivity(req *http.Request) (sawSearch bool, sawCaptcha bool) {
	if f.searchRegex.MatchString(req.Host) {
		op := ops.Begin("google_search")
		log.Tracef("Saw google search")
		op.End()
		return true, false
	}
	if f.captchaRegex.MatchString(req.Host) {
		op := ops.Begin("google_captcha")
		log.Tracef("Saw google captcha")
		op.End()
		return false, true
	}
	return false, false
}
