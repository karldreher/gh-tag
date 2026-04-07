package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// browserURL is a validated https URL that is safe to open in a browser.
// The only way to obtain a browserURL is through newBrowserURL, which
// enforces scheme and host constraints. openURL accepts only this type,
// making it impossible to call with an unvalidated string.
type browserURL string

// newBrowserURL parses and validates s as an https URL with a non-empty host.
// Returns an error if the URL is malformed, uses a non-https scheme, or has
// no host — guarding against open-redirect or protocol-injection via tag names
// or repo URLs that do not conform to expectations.
func newBrowserURL(s string) (browserURL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", s, err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("URL %q must use https scheme", s)
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL %q has no host", s)
	}
	return browserURL(s), nil
}

// openInBrowser opens a path relative to the current GitHub repository in the
// default browser. It resolves the repo's canonical URL via gh, then appends
// the path — e.g. "releases/tag/v1.2.3" becomes
// https://github.com/<owner>/<repo>/releases/tag/v1.2.3.
//
// gh browse is intentionally not used here: it treats its argument as a file
// path within the repo tree, not a URL path, which produces wrong URLs for
// non-file destinations like /tags.
func openInBrowser(path string) error {
	out, err := exec.Command("gh", "repo", "view", "--json", "url", "-q", ".url").Output()
	if err != nil {
		return fmt.Errorf("getting repo URL: %w", err)
	}
	repoURL := strings.TrimSpace(string(out))
	target, err := newBrowserURL(repoURL + "/" + path)
	if err != nil {
		return err
	}
	return openURL(target)
}

// openURL opens the given URL in the default browser using the platform's
// native launcher. It accepts only a validated browserURL to prevent
// unvalidated strings from reaching the shell.
func openURL(u browserURL) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", string(u))
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", string(u))
	default:
		cmd = exec.Command("xdg-open", string(u))
	}
	return cmd.Run()
}
