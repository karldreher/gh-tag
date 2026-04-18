package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// parseBrowserURL parses and validates s as an https URL with a non-empty host.
// Returns an error if the URL is malformed, uses a non-https scheme, or has
// no host — guarding against open-redirect or protocol-injection via tag names
// or repo URLs that do not conform to expectations.
func parseBrowserURL(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", s, err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("URL %q must use https scheme", s)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("URL %q has no host", s)
	}
	return u, nil
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
	target, err := parseBrowserURL(repoURL + "/" + path)
	if err != nil {
		return err
	}
	return openURL(target)
}

// openURL opens the given URL in the default browser using the platform's
// native launcher. The URL must be pre-validated (via parseBrowserURL) to
// prevent unvalidated strings from reaching the shell.
func openURL(u *url.URL) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u.String())
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", u.String())
	default:
		cmd = exec.Command("xdg-open", u.String())
	}
	return cmd.Run()
}
