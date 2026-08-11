package mux

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func Test_newRouteRegexp_Errors(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"/{}", `mux: missing name or pattern in "{}"`},
		{"/{:.}", `mux: missing name or pattern in "{:.}"`},
		{"/{a:}", `mux: missing name or pattern in "{a:}"`},
		{"/{id:abc(}", `mux: error compiling regex for "{id:abc(}":`},
	}

	for _, tc := range tests {
		t.Run("Test case for "+tc.in, func(t *testing.T) {
			_, err := newRouteRegexp(tc.in, 0, routeRegexpOptions{})
			if err != nil {
				if strings.HasPrefix(err.Error(), tc.out) {
					return
				}
				t.Errorf("Resulting error does not contain %q as expected, error: %s", tc.out, err)
			} else {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// Test_routeRegexpMatchMatchesRegexp asserts that the fast paths in
// routeRegexp.Match agree with executing the compiled regexp, for templates
// with and without variables. Variable-free templates skip the regexp and
// compare strings instead, so the subjects below include every character
// regexp.QuoteMeta is responsible for escaping.
func Test_routeRegexpMatchMatchesRegexp(t *testing.T) {
	templates := []string{
		"/foo", "/foo/", "/", "", "/a.b", "/a+b", "/a*b", "/a?b", "/a(b)c",
		"/a[b]c", "/a{{}}b", "/a|b", "/a^b", "/a$b", "/a\\b", "/.*",
		"/{v}", "/{v:.+}", "/x/{v}/y", "/a.b/{v}",
	}
	subjects := []string{
		"", "/", "/foo", "/foo/", "/foo//", "/FOO", "/a.b", "/axb", "/a.b/",
		"/a+b", "/ab", "/aab", "/a*b", "/a?b", "/a(b)c", "/abc", "/a[b]c",
		"/a|b", "/a^b", "/a$b", "/a\\b", "/.*", "/anything", "/x/1/y", "/x//y",
		"/a.b/c", "/a\nb", "/foo\n",
	}
	for _, typ := range []regexpType{regexpTypePath, regexpTypePrefix} {
		for _, strictSlash := range []bool{false, true} {
			for _, tpl := range templates {
				opts := routeRegexpOptions{strictSlash: strictSlash}
				rr, err := newRouteRegexp(tpl, typ, opts)
				if err != nil {
					t.Fatalf("newRouteRegexp(%q, %v, %+v): %v", tpl, typ, opts, err)
				}
				for _, subject := range subjects {
					req := newRequestForPath(t, subject)
					want := rr.regexp.MatchString(subject)
					got := rr.Match(req, &RouteMatch{})
					if got != want {
						t.Errorf("template %q type=%v strictSlash=%v subject %q: Match = %v, regexp %q = %v",
							tpl, typ, strictSlash, subject, got, rr.pattern, want)
					}
				}
			}
		}
	}
}

// Test_routeRegexpMatchHostMatchesRegexp is the host-template counterpart of
// Test_routeRegexpMatchMatchesRegexp.
func Test_routeRegexpMatchHostMatchesRegexp(t *testing.T) {
	// A template carrying a port clears wildcardHostPort, so both the trimming
	// and non-trimming branches are covered.
	templates := []string{
		"example.com", "a-b.example.com", "{sub}.example.com", "{sub:.+}.example.com",
		"example.com:8080", "{sub}.example.com:8080",
	}
	hosts := []string{
		"example.com", "EXAMPLE.COM", "examplexcom", "a.example.com", "a.b.example.com", "",
		"example.com:8080", "a.example.com:8080", "example.com:80", ":8080", "example.com:",
	}
	for _, tpl := range templates {
		rr, err := newRouteRegexp(tpl, regexpTypeHost, routeRegexpOptions{})
		if err != nil {
			t.Fatalf("newRouteRegexp(%q): %v", tpl, err)
		}
		for _, host := range hosts {
			req := newRequestForPath(t, "/")
			req.Host = host
			subject := host
			if rr.wildcardHostPort {
				if i := strings.Index(subject, ":"); i != -1 {
					subject = subject[:i]
				}
			}
			want := rr.regexp.MatchString(subject)
			if got := rr.Match(req, &RouteMatch{}); got != want {
				t.Errorf("template %q host %q: Match = %v, regexp %q = %v", tpl, host, got, rr.pattern, want)
			}
		}
	}
}

// Test_matchQueryStringMatchesRegexp asserts that the two query shortcuts —
// answering from matchesEmpty when the key is absent, and skipping the scan for
// an empty RawQuery — agree with executing the regexp against the subject the
// unshortcut code would have built.
func Test_matchQueryStringMatchesRegexp(t *testing.T) {
	// getURLQueryUnshortcut is getURLQuery without the empty-RawQuery return,
	// so the comparison covers that shortcut too.
	getURLQueryUnshortcut := func(r *routeRegexp, req *http.Request) string {
		if val, ok := findFirstQueryKey(req.URL.RawQuery, r.templateKey); ok {
			return r.templateKey + "=" + val
		}
		return ""
	}

	templates := []string{
		"acl=", "foo=bar", "list-type=2", "id={id:.*}", "x={v:.+}", "=",
		"{k}=v", "{k}=", "versions=", "select-type=2", "events={events:.*}",
	}
	rawQueries := []string{
		"", "acl", "acl=", "acl=1", "foo=bar", "foo=baz", "id=", "id=7",
		"list-type=2", "list-type=1", "a=1&acl=", "acl=&a=1", "x=", "x=1",
		"=", "=v", "k=v", "&", ";", "&&", "%zz=1", "versions=&metadata=true",
		"events=x", "select=&select-type=2", "id=1;id=2",
	}
	for _, tpl := range templates {
		rr, err := newRouteRegexp(tpl, regexpTypeQuery, routeRegexpOptions{})
		if err != nil {
			t.Fatalf("newRouteRegexp(%q): %v", tpl, err)
		}
		for _, raw := range rawQueries {
			req := &http.Request{URL: &url.URL{Path: "/", RawQuery: raw}, Host: "localhost"}
			want := rr.regexp.MatchString(getURLQueryUnshortcut(rr, req))
			if got := rr.matchQueryString(req); got != want {
				t.Errorf("template %q RawQuery %q: matchQueryString = %v, want %v (pattern %q, matchesEmpty %v)",
					tpl, raw, got, want, rr.pattern, rr.matchesEmpty)
			}
		}
	}
}

// Test_pathMemoSubjectChange covers a route whose inherited prefix matcher and
// own path matcher disagree on useEncodedPath, so a single match evaluates two
// different subjects and the memo entries must not carry across them.
func Test_pathMemoSubjectChange(t *testing.T) {
	router := NewRouter().SkipClean(true)
	sub := router.PathPrefix("/{bucket}").Subrouter()
	sub.UseEncodedPath()
	sub.Path("/{object}").HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	// Parsed rather than assembled so that URL.RawPath carries the escaping and
	// EscapedPath differs from Path.
	req, err := http.NewRequest(http.MethodGet, "http://localhost/mybucket/a%2Fb", nil)
	if err != nil {
		t.Fatal(err)
	}
	var match RouteMatch
	if !router.Match(req, &match) {
		t.Fatalf("expected a match, got MatchErr %v", match.MatchErr)
	}
	if got, want := match.Vars["bucket"], "mybucket"; got != want {
		t.Errorf("bucket = %q, want %q", got, want)
	}
	if got, want := match.Vars["object"], "a%2Fb"; got != want {
		t.Errorf("object = %q, want %q", got, want)
	}
}

// Test_pathMemoSameTemplateDifferentSubject covers two sibling routes with an
// identical path template that disagree on useEncodedPath. They compile to the
// same pattern but match different subjects, so a memo entry recorded for one
// must not answer for the other.
func Test_pathMemoSameTemplateDifferentSubject(t *testing.T) {
	router := NewRouter().SkipClean(true)

	// "/x/a/b" has two segments after the prefix, so this route cannot match
	// the decoded path and is evaluated first, seeding the memo with a miss.
	plain := router.PathPrefix("/x").Subrouter()
	plain.Path("/{v}").Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).Name("plain")

	// The same template against the escaped path, where "%2F" keeps it to one
	// segment and the route does match.
	encoded := router.PathPrefix("/x").Subrouter()
	encoded.UseEncodedPath()
	encoded.Path("/{v}").Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).Name("encoded")

	req, err := http.NewRequest(http.MethodGet, "http://localhost/x/a%2Fb", nil)
	if err != nil {
		t.Fatal(err)
	}
	var match RouteMatch
	if !router.Match(req, &match) {
		t.Fatalf("expected the encoded route to match, got MatchErr %v", match.MatchErr)
	}
	if got := match.Route.GetName(); got != "encoded" {
		t.Errorf("matched route = %q, want %q", got, "encoded")
	}
	if got, want := match.Vars["v"], "a%2Fb"; got != want {
		t.Errorf("v = %q, want %q", got, want)
	}
}

// Test_pathMemoStaleEntryAfterSubjectChange exercises the memo reset. Pattern P
// is evaluated against the decoded path, then a different pattern against the
// escaped path, then P again against the escaped path. Unless the entries
// recorded for the first subject are dropped when the subject changes, the last
// lookup finds P's stale result and the matching route is skipped.
func Test_pathMemoStaleEntryAfterSubjectChange(t *testing.T) {
	router := NewRouter().SkipClean(true)

	// Evaluates P against the decoded "/x/a/b", which has two segments and so
	// does not match.
	plain := router.PathPrefix("/x").Subrouter()
	plain.Path("/{v}").Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).Name("plain")

	// A different pattern, matched against the escaped path, which changes the
	// memo's subject. The method mismatch keeps the route from winning while
	// still letting its path matcher run.
	other := router.PathPrefix("/x").Subrouter()
	other.UseEncodedPath()
	other.Methods(http.MethodPost).Path("/{a:.+}").
		Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).Name("other")

	// P again, now against the escaped path, where it does match.
	encoded := router.PathPrefix("/x").Subrouter()
	encoded.UseEncodedPath()
	encoded.Path("/{v}").Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).Name("encoded")

	req, err := http.NewRequest(http.MethodGet, "http://localhost/x/a%2Fb", nil)
	if err != nil {
		t.Fatal(err)
	}
	var match RouteMatch
	if !router.Match(req, &match) {
		t.Fatalf("expected the encoded route to match, got MatchErr %v", match.MatchErr)
	}
	if got := match.Route.GetName(); got != "encoded" {
		t.Errorf("matched route = %q, want %q", got, "encoded")
	}
}

// Test_pathMemoDeepNesting checks that nesting deeper than the memo can hold
// still routes correctly; exceeding its capacity may only cost extra regexp
// executions, never change the outcome.
func Test_pathMemoDeepNesting(t *testing.T) {
	router := NewRouter()
	r := router
	for _, seg := range []string{"a", "b", "c", "d", "e", "f"} {
		r = r.PathPrefix("/{" + seg + "}").Subrouter()
	}
	r.Path("/{leaf}").HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	req := newRequestForPath(t, "/1/2/3/4/5/6/7")
	var match RouteMatch
	if !router.Match(req, &match) {
		t.Fatalf("expected a match, got MatchErr %v", match.MatchErr)
	}
	want := map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6", "leaf": "7"}
	if !reflect.DeepEqual(match.Vars, want) {
		t.Errorf("Vars = %v, want %v", match.Vars, want)
	}
}

// newRequestForPath builds a request whose URL.Path is exactly path. The URL is
// assembled rather than parsed so that subjects containing "?" or "#" reach the
// matcher intact, and left relative so getHost reports Request.Host.
func newRequestForPath(t *testing.T, path string) *http.Request {
	t.Helper()
	return &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: path},
		Host:   "localhost",
		Header: make(http.Header),
	}
}

func Test_findFirstQueryKey(t *testing.T) {
	tests := []string{
		"a=1&b=2",
		"a=1&a=2&a=banana",
		"ascii=%3Ckey%3A+0x90%3E",
		"a=1;b=2",
		"a=1&a=2;a=banana",
		"a==",
		"a=%2",
		"a=20&%20%3F&=%23+%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09:%2F@$%27%28%29%2A%2C%3B&a=30",
		"a=1& ?&=#+%!<>#\"{}|\\^[]`☺\t:/@$'()*,;&a=5",
		"a=xxxxxxxxxxxxxxxx&b=YYYYYYYYYYYYYYY&c=ppppppppppppppppppp&f=ttttttttttttttttt&a=uuuuuuuuuuuuu",
	}
	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			// Check against url.ParseQuery, ignoring the error.
			all, _ := url.ParseQuery(query)
			for key, want := range all {
				t.Run(key, func(t *testing.T) {
					got, ok := findFirstQueryKey(query, key)
					if !ok {
						t.Error("Did not get expected key", key)
					}
					if !reflect.DeepEqual(got, want[0]) {
						t.Errorf("findFirstQueryKey(%s,%s) = %v, want %v", query, key, got, want[0])
					}
				})
			}
		})
	}
}

func Benchmark_findQueryKey(b *testing.B) {
	tests := []string{
		"a=1&b=2",
		"ascii=%3Ckey%3A+0x90%3E",
		"a=20&%20%3F&=%23+%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09:%2F@$%27%28%29%2A%2C%3B&a=30",
		"a=xxxxxxxxxxxxxxxx&bbb=YYYYYYYYYYYYYYY&cccc=ppppppppppppppppppp&ddddd=ttttttttttttttttt&a=uuuuuuuuuuuuu",
		"a=;b=;c=;d=;e=;f=;g=;h=;i=,j=;k=",
	}
	for i, query := range tests {
		b.Run(strconv.Itoa(i), func(b *testing.B) {
			// Check against url.ParseQuery, ignoring the error.
			all, _ := url.ParseQuery(query)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for key := range all {
					_, _ = findFirstQueryKey(query, key)
				}
			}
		})
	}
}

func Benchmark_findQueryKeyGoLib(b *testing.B) {
	tests := []string{
		"a=1&b=2",
		"ascii=%3Ckey%3A+0x90%3E",
		"a=20&%20%3F&=%23+%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09:%2F@$%27%28%29%2A%2C%3B&a=30",
		"a=xxxxxxxxxxxxxxxx&bbb=YYYYYYYYYYYYYYY&cccc=ppppppppppppppppppp&ddddd=ttttttttttttttttt&a=uuuuuuuuuuuuu",
		"a=;b=;c=;d=;e=;f=;g=;h=;i=,j=;k=",
	}
	for i, query := range tests {
		b.Run(strconv.Itoa(i), func(b *testing.B) {
			// Check against url.ParseQuery, ignoring the error.
			all, _ := url.ParseQuery(query)
			var u url.URL
			u.RawQuery = query
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for key := range all {
					v := u.Query()[key]
					if len(v) > 0 {
						_ = v[0]
					}
				}
			}
		})
	}
}
