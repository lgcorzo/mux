// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type routeRegexpOptions struct {
	strictSlash    bool
	useEncodedPath bool
}

type regexpType int

const (
	regexpTypePath regexpType = iota
	regexpTypeHost
	regexpTypePrefix
	regexpTypeQuery
)

// newRouteRegexp parses a route template and returns a routeRegexp,
// used to match a host, a path or a query string.
//
// It will extract named variables, assemble a regexp to be matched, create
// a "reverse" template to build URLs and compile regexps to validate variable
// values used in URL building.
//
// Previously we accepted only Python-like identifiers for variable
// names ([a-zA-Z_][a-zA-Z0-9_]*), but currently the only restriction is that
// name and pattern can't be empty, and names can't contain a colon.
func newRouteRegexp(tpl string, typ regexpType, options routeRegexpOptions) (*routeRegexp, error) {
	// Check if it is well-formed.
	idxs, errBraces := braceIndices(tpl)
	if errBraces != nil {
		return nil, errBraces
	}
	// Backup the original.
	template := tpl
	// Now let's parse it.
	defaultPattern := "[^/]+"
	if typ == regexpTypeQuery {
		defaultPattern = ".*"
	} else if typ == regexpTypeHost {
		defaultPattern = "[^.]+"
	}
	// Only match strict slash if not matching
	if typ != regexpTypePath {
		options.strictSlash = false
	}
	// Set a flag for strictSlash.
	endSlash := false
	if options.strictSlash && strings.HasSuffix(tpl, "/") {
		tpl = tpl[:len(tpl)-1]
		endSlash = true
	}
	varsN := make([]string, len(idxs)/2)
	varsR := make([]*regexp.Regexp, len(idxs)/2)

	var pattern, reverse strings.Builder
	pattern.WriteByte('^')

	var end, colonIdx, groupIdx int
	var err error
	var patt, param, name string
	for i := 0; i < len(idxs); i += 2 {
		// Set all values we are interested in.
		groupIdx = i / 2

		raw := tpl[end:idxs[i]]
		end = idxs[i+1]
		tag := tpl[idxs[i]:end]

		// trim braces from tag
		param = tag[1 : len(tag)-1]

		colonIdx = strings.Index(param, ":")
		if colonIdx == -1 {
			name = param
			patt = defaultPattern
		} else {
			name = param[0:colonIdx]
			patt = param[colonIdx+1:]
		}

		// Name or pattern can't be empty.
		if name == "" || patt == "" {
			return nil, fmt.Errorf("mux: missing name or pattern in %q", tag)
		}
		// Build the regexp pattern.
		groupName := varGroupName(groupIdx)

		pattern.WriteString(regexp.QuoteMeta(raw) + "(?P<" + groupName + ">" + patt + ")")

		// Build the reverse template.
		reverse.WriteString(raw + "%s")

		// Append variable name and compiled pattern.
		varsN[groupIdx] = name
		varsR[groupIdx], err = RegexpCompileFunc("^" + patt + "$")
		if err != nil {
			return nil, fmt.Errorf("mux: error compiling regex for %q: %w", tag, err)
		}
	}
	// Add the remaining.
	raw := tpl[end:]
	pattern.WriteString(regexp.QuoteMeta(raw))
	if options.strictSlash {
		pattern.WriteString("[/]?")
	}
	var templateKey string
	if typ == regexpTypeQuery {
		if eq := strings.IndexByte(template, '='); eq >= 0 {
			templateKey = template[:eq]
			if eq == len(template)-1 {
				pattern.WriteString(defaultPattern)
			}
		} else {
			templateKey = template
		}
	}
	if typ != regexpTypePrefix {
		pattern.WriteByte('$')
	}

	// Compile full regexp.
	patternStr := pattern.String()
	reg, errCompile := RegexpCompileFunc(patternStr)
	if errCompile != nil {
		return nil, errCompile
	}

	// Check for capturing groups which used to work in older versions
	if reg.NumSubexp() != len(idxs)/2 {
		panic(fmt.Sprintf("route %s contains capture groups in its regexp. ", template) +
			"Only non-capturing groups are accepted: e.g. (?:pattern) instead of (pattern)")
	}

	var wildcardHostPort bool
	if typ == regexpTypeHost {
		if !strings.Contains(patternStr, ":") {
			wildcardHostPort = true
		}
	}
	reverse.WriteString(raw)
	if endSlash {
		reverse.WriteByte('/')
	}

	// A template without variables compiles to an anchored, fully quoted
	// literal, so matching it reduces to a string comparison. raw is the whole
	// template here, already trimmed of a strictSlash trailing slash.
	isLiteral := len(idxs) == 0 && typ != regexpTypeQuery

	// Done!
	return &routeRegexp{
		template:         template,
		templateKey:      templateKey,
		regexpType:       typ,
		options:          options,
		regexp:           reg,
		pattern:          patternStr,
		isLiteral:        isLiteral,
		literal:          raw,
		matchesEmpty:     reg.MatchString(""),
		reverse:          reverse.String(),
		varsN:            varsN,
		varsR:            varsR,
		wildcardHostPort: wildcardHostPort,
	}, nil
}

// routeRegexp stores a regexp to match a host or path and information to
// collect and validate route variables.
type routeRegexp struct {
	// The unmodified template.
	template string
	// The query template key (substring of template up to "="). Only set for
	// regexpTypeQuery so getURLQuery can avoid splitting on every request.
	templateKey string
	// The type of match
	regexpType regexpType
	// Options for matching
	options routeRegexpOptions
	// Expanded regexp.
	regexp *regexp.Regexp
	// The source pattern of regexp. Routes that share a template compile to
	// the same pattern, which is what lets pathMemo reuse a match result
	// across them.
	pattern string
	// Set when the template has no variables, so the compiled regexp is an
	// anchored literal and matching reduces to a string comparison against
	// literal (the template, minus any strictSlash trailing slash).
	isLiteral bool
	literal   string
	// Whether regexp matches the empty string, so that matchQueryString can
	// answer without executing it when the query key is absent.
	matchesEmpty bool
	// Reverse template.
	reverse string
	// Variable names.
	varsN []string
	// Variable regexps (validators).
	varsR []*regexp.Regexp
	// Wildcard host-port (no strict port match in hostname)
	wildcardHostPort bool
}

// Match matches the regexp against the URL host or path.
func (r *routeRegexp) Match(req *http.Request, match *RouteMatch) bool {
	if r.regexpType == regexpTypeHost {
		host := getHost(req)
		if r.wildcardHostPort {
			// Don't be strict on the port match
			if i := strings.Index(host, ":"); i != -1 {
				host = host[:i]
			}
		}
		return r.matchString(host)
	}

	if r.regexpType == regexpTypeQuery {
		return r.matchQueryString(req)
	}

	path := r.subjectPath(req, match)
	if r.isLiteral {
		return r.matchLiteral(path)
	}
	if matched, ok := match.pathMemo.lookup(r.pattern, path); ok {
		return matched
	}
	matched := r.regexp.MatchString(path)
	match.pathMemo.store(r.pattern, path, matched)
	return matched
}

// subjectPath returns the request path this regexp matches against, reusing the
// escaped form across the routes of a single match.
func (r *routeRegexp) subjectPath(req *http.Request, match *RouteMatch) string {
	if r.options.useEncodedPath {
		return match.pathMemo.escapedPath(req.URL)
	}
	return req.URL.Path
}

// matchString reports whether s matches, using a string comparison when the
// compiled regexp is an anchored literal.
func (r *routeRegexp) matchString(s string) bool {
	if r.isLiteral {
		return r.matchLiteral(s)
	}
	return r.regexp.MatchString(s)
}

// matchLiteral is the regexp-free equivalent of matching a variable-free
// template: "^" + QuoteMeta(literal) + optional "[/]?" + optional "$".
func (r *routeRegexp) matchLiteral(s string) bool {
	switch {
	case r.regexpType == regexpTypePrefix:
		return strings.HasPrefix(s, r.literal)
	case s == r.literal:
		return true
	case r.options.strictSlash && len(s) == len(r.literal)+1 && s[len(s)-1] == '/':
		return s[:len(s)-1] == r.literal
	default:
		return false
	}
}

// pathMemo memoizes the path-matching work of a single Router.Match call.
//
// Two things make routers re-run the same path regexp against the same subject
// many times per request. Routes are commonly registered against one path
// template and discriminated by method, query or header matchers — the AIStor
// S3 API has ~40 such routes for PUT alone. And every route in a subrouter
// carries a copy of its ancestors' matchers, so each one re-evaluates the
// prefixes that were already matched to reach it. Both collapse into a string
// comparison here.
//
// Ancestor and leaf patterns alternate as the route list is walked, so the memo
// holds several entries; the count only needs to cover the distinct path
// patterns along one router nesting chain.
type pathMemo struct {
	// Escaped path, memoized against the URL it was derived from.
	url         *url.URL
	encodedPath string

	// The subject every entry was evaluated against. Routes almost always
	// agree on it, so it is held once rather than per entry; a route that
	// disagrees (a differing useEncodedPath) resets the entries.
	subject string
	entries [4]pathMemoEntry
	next    uint8
}

// pathMemoEntry needs no validity flag: every compiled pattern starts with "^",
// so a zero-valued entry cannot be mistaken for a stored one.
type pathMemoEntry struct {
	pattern string
	matched bool
}

func (m *pathMemo) escapedPath(u *url.URL) string {
	if m.url != u {
		m.url = u
		m.encodedPath = u.EscapedPath()
	}
	return m.encodedPath
}

func (m *pathMemo) lookup(pattern, subject string) (matched, ok bool) {
	if m.subject != subject {
		return false, false
	}
	for i := range m.entries {
		if e := &m.entries[i]; e.pattern == pattern {
			return e.matched, true
		}
	}
	return false, false
}

func (m *pathMemo) store(pattern, subject string, matched bool) {
	if m.subject != subject {
		m.subject = subject
		m.entries = [len(m.entries)]pathMemoEntry{}
		m.next = 0
	}
	m.entries[m.next] = pathMemoEntry{pattern: pattern, matched: matched}
	m.next = (m.next + 1) % uint8(len(m.entries))
}

// url builds a URL part using the given values.
func (r *routeRegexp) url(values map[string]string) (string, error) {
	urlValues := make([]interface{}, len(r.varsN))
	for k, v := range r.varsN {
		value, ok := values[v]
		if !ok {
			return "", fmt.Errorf("mux: missing route variable %q", v)
		}
		if r.regexpType == regexpTypeQuery {
			value = url.QueryEscape(value)
		}
		urlValues[k] = value
	}
	rv := fmt.Sprintf(r.reverse, urlValues...)
	if !r.regexp.MatchString(rv) {
		// The URL is checked against the full regexp, instead of checking
		// individual variables. This is faster but to provide a good error
		// message, we check individual regexps if the URL doesn't match.
		for k, v := range r.varsN {
			if !r.varsR[k].MatchString(values[v]) {
				return "", fmt.Errorf(
					"mux: variable %q doesn't match, expected %q", values[v],
					r.varsR[k].String())
			}
		}
	}
	return rv, nil
}

// getURLQuery returns a single query parameter from a request URL.
// For a URL with foo=bar&baz=ding, we return only the relevant key
// value pair for the routeRegexp.
func (r *routeRegexp) getURLQuery(req *http.Request) string {
	if r.regexpType != regexpTypeQuery || req.URL.RawQuery == "" {
		return ""
	}
	val, ok := findFirstQueryKey(req.URL.RawQuery, r.templateKey)
	if ok {
		return r.templateKey + "=" + val
	}
	return ""
}

// findFirstQueryKey returns the same result as (*url.URL).Query()[key][0].
// If key was not found, empty string and false is returned.
func findFirstQueryKey(rawQuery, key string) (value string, ok bool) {
	for len(rawQuery) > 0 {
		foundKey := rawQuery
		if i := strings.IndexAny(foundKey, "&;"); i >= 0 {
			foundKey, rawQuery = foundKey[:i], foundKey[i+1:]
		} else {
			rawQuery = ""
		}
		if len(foundKey) == 0 {
			continue
		}
		var rawValue string
		if i := strings.IndexByte(foundKey, '='); i >= 0 {
			foundKey, rawValue = foundKey[:i], foundKey[i+1:]
		}
		if len(foundKey) < len(key) {
			// Cannot possibly be key.
			continue
		}
		keyString, err := url.QueryUnescape(foundKey)
		if err != nil {
			continue
		}
		if keyString != key {
			continue
		}
		valueString, err := url.QueryUnescape(rawValue)
		if err != nil {
			continue
		}
		return valueString, true
	}
	return "", false
}

func (r *routeRegexp) matchQueryString(req *http.Request) bool {
	query := r.getURLQuery(req)
	if query == "" {
		// The key is absent, so the subject is the empty string whatever the
		// request holds and the result was settled at compile time. Routers
		// commonly register many query-discriminated routes against one path,
		// and a request naming none of those keys reaches every one of them.
		return r.matchesEmpty
	}
	return r.regexp.MatchString(query)
}

// braceIndices returns the first level curly brace indices from a string.
// It returns an error in case of unbalanced braces.
func braceIndices(s string) ([]int, error) {
	var level, idx int
	var idxs []int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if level++; level == 1 {
				idx = i
			}
		case '}':
			if level--; level == 0 {
				idxs = append(idxs, idx, i+1)
			} else if level < 0 {
				return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
			}
		}
	}
	if level != 0 {
		return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
	}
	return idxs, nil
}

// varGroupName builds a capturing group name for the indexed variable.
func varGroupName(idx int) string {
	return "v" + strconv.Itoa(idx)
}

// ----------------------------------------------------------------------------
// routeRegexpGroup
// ----------------------------------------------------------------------------

// routeRegexpGroup groups the route matchers that carry variables.
type routeRegexpGroup struct {
	host    *routeRegexp
	path    *routeRegexp
	queries []*routeRegexp
}

// setMatch extracts the variables from the URL once a route matches.
func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
	// Store host variables.
	if v.host != nil {
		if len(v.host.varsN) > 0 {
			host := getHost(req)
			if v.host.wildcardHostPort {
				// Don't be strict on the port match
				if i := strings.Index(host, ":"); i != -1 {
					host = host[:i]
				}
			}
			matches := v.host.regexp.FindStringSubmatchIndex(host)
			if len(matches) > 0 {
				m.Vars = extractVars(host, matches, v.host.varsN, m.Vars)
			}
		}
	}
	path := req.URL.Path
	if r.useEncodedPath {
		path = m.pathMemo.escapedPath(req.URL)
	}
	// Store path variables.
	if v.path != nil {
		if len(v.path.varsN) > 0 {
			matches := v.path.regexp.FindStringSubmatchIndex(path)
			if len(matches) > 0 {
				m.Vars = extractVars(path, matches, v.path.varsN, m.Vars)
			}
		}
		// Check if we should redirect.
		if v.path.options.strictSlash {
			p1 := strings.HasSuffix(path, "/")
			p2 := strings.HasSuffix(v.path.template, "/")
			if p1 != p2 {
				p := req.URL.Path
				if p1 {
					p = p[:len(p)-1]
				} else {
					p += "/"
				}
				u := replaceURLPath(req.URL, p)
				m.Handler = http.RedirectHandler(u, http.StatusMovedPermanently)
			}
		}
	}
	// Store query string variables.
	for _, q := range v.queries {
		if len(q.varsN) > 0 {
			queryURL := q.getURLQuery(req)
			matches := q.regexp.FindStringSubmatchIndex(queryURL)
			if len(matches) > 0 {
				m.Vars = extractVars(queryURL, matches, q.varsN, m.Vars)
			}
		}
	}
}

// getHost tries its best to return the request host.
// According to section 14.23 of RFC 2616 the Host header
// can include the port number if the default value of 80 is not used.
func getHost(r *http.Request) string {
	if r.URL.IsAbs() {
		return r.URL.Host
	}
	return r.Host
}

func extractVars(input string, matches []int, names []string, output map[string]string) map[string]string {
	for i, name := range names {
		if output == nil {
			output = make(map[string]string, len(names))
		}
		output[name] = input[matches[2*i+2]:matches[2*i+3]]
	}
	return output
}
