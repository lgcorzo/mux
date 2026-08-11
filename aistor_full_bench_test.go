// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// This file mocks the whole AIStor server router as assembled by
// configureServerHandler, not just the S3 API layer, so that routing cost is
// measured over the route table a real request actually traverses. Registration
// order matters and is preserved: the S3 API router is registered last, so an
// object request walks every earlier layer first.
//
// Handlers are no-ops, and the admin and REST layers are generated with the
// route count and matcher mix counted from AIStor rather than transcribed
// literally — for routing cost only the shape matters (literal vs variable
// prefix, method, query and header matchers, and how many routes share a
// template). Counts as of the survey: 162 admin routes per API version over two
// versions plus 35 v4-only, 19 of which carry path variables and 50 a query.
const (
	mockReservedBucketPath = "/minio"
	mockAdminPathPrefix    = mockReservedBucketPath + "/admin"
	mockHealthPathPrefix   = mockReservedBucketPath + "/health"
	mockKMSPathPrefix      = mockReservedBucketPath + "/kms"
	mockConsolePathPrefix  = mockReservedBucketPath + "/console"
	mockMCPRoutePrefix     = mockReservedBucketPath + "/mcp"
	mockStorageRESTPrefix  = mockReservedBucketPath + "/storage"
	mockPeerRESTPrefix     = mockReservedBucketPath + "/peer"
	mockGridPath           = mockReservedBucketPath + "/grid/v1"
	mockGridLockPath       = mockReservedBucketPath + "/grid/lock/v1"

	mockTablesRouteRoot       = "/_iceberg/v1"
	mockDeltaSharingRouteRoot = "/_delta-sharing/v1"
	mockMemoryRouteRoot       = "/_mem/v1"
)

// registerMockDistErasureRouters mirrors registerDistErasureRouters: the storage
// and peer REST layers plus the two grid handles, all under literal prefixes.
func registerMockDistErasureRouters(router *Router) {
	storageRouter := router.PathPrefix(mockStorageRESTPrefix).Subrouter()
	for _, name := range []string{
		"health", "diskinfo", "nsscanner", "makevol", "makevolbulk", "listvols",
		"statvol", "deletevol", "listdir", "deletefile", "deleteversions",
		"renamefile", "renamedata", "checkparts", "readall", "readxl",
		"writemetadata", "updatemetadata", "deleteversion", "writeall",
		"readversion", "readfile", "readfilestream", "liststream", "statinfofile",
		"cleanabandoned",
	} {
		storageRouter.Methods(http.MethodPost).Path("/v60/" + name).HandlerFunc(nopHandler)
	}

	peerRouter := router.PathPrefix(mockPeerRESTPrefix).Subrouter()
	for _, name := range []string{
		"health", "verifybinary", "commitbinary", "signalservice", "backgroundhealstatus",
		"getlocks", "server-info", "proc-info", "mem-info", "sys-errors", "sys-services",
		"sys-config", "os-info", "disk-hw-info", "cpu-info", "net-hw-info", "get-all-bucket-stats",
		"get-bucket-stats", "get-sr-metrics", "load-bucket-metadata", "delete-bucket-metadata",
		"reload-pool-meta", "stop-rebalance", "load-rebalance-meta", "load-transition-tier-config",
		"speedtest", "driveSpeedTest", "devnull", "netperf",
	} {
		peerRouter.Methods(http.MethodPost).Path("/v52/" + name).HandlerFunc(nopHandler)
	}

	router.Handle(mockGridLockPath, http.HandlerFunc(nopHandler))
	router.Handle(mockGridPath, http.HandlerFunc(nopHandler))
}

// registerMockAdminRouter mirrors registerAdminRouter: common routes registered
// once per API version, then the v4-only block.
func registerMockAdminRouter(router *Router) {
	adminRouter := router.PathPrefix(mockAdminPathPrefix).Subrouter()

	for _, version := range []string{"/v4", "/v3"} {
		for i := 0; i < 162; i++ {
			method := http.MethodGet
			switch {
			case i%7 == 1, i%7 == 2:
				method = http.MethodPost
			case i%7 == 3:
				method = http.MethodPut
			case i%7 == 4 && i%11 == 0:
				method = http.MethodDelete
			}
			route := adminRouter.Methods(method)
			// 19 of 162 carry a path variable.
			if i%8 == 0 {
				route = route.Path(version + "/res" + strconv.Itoa(i) + "/{name}")
			} else {
				route = route.Path(version + "/op" + strconv.Itoa(i))
			}
			route = route.HandlerFunc(nopHandler)
			// 50 of 162 carry a query matcher.
			if i%3 == 0 {
				route.Queries("q"+strconv.Itoa(i), "")
			}
		}
	}

	for i := 0; i < 35; i++ {
		adminRouter.Methods(http.MethodGet).Path("/v4/latest" + strconv.Itoa(i)).HandlerFunc(nopHandler)
	}
}

func registerMockHealthCheckRouter(router *Router) {
	healthRouter := router.PathPrefix(mockHealthPathPrefix).Subrouter()
	for _, p := range []string{"/cluster", "/cluster/read", "/live", "/ready"} {
		healthRouter.Methods(http.MethodGet, http.MethodHead).Path(p).HandlerFunc(nopHandler)
	}
	healthRouter.Methods(http.MethodOptions).HandlerFunc(nopHandler)
}

func registerMockMetricsRouter(router *Router) {
	metricsRouter := router.NewRoute().PathPrefix(mockReservedBucketPath + "/").Subrouter()
	for _, p := range []string{
		"/prometheus/metrics", "/v2/metrics/cluster", "/v2/metrics/bucket",
		"/v2/metrics/node", "/v2/metrics/resource",
	} {
		metricsRouter.Handle(p, http.HandlerFunc(nopHandler))
	}
	metricsRouter.Methods(http.MethodGet).Path("/metrics/v3{pathComps:.*}").Handler(http.HandlerFunc(nopHandler))
	metricsRouter.Methods(http.MethodOptions).HandlerFunc(nopHandler)
}

// registerMockSTSRouter mirrors registerSTSRouter. Its prefix is "/", so every
// request in the server reaches these routes, and the first two are gated by a
// MatcherFunc that inspects headers.
func registerMockSTSRouter(router *Router) {
	stsRouter := router.NewRoute().PathPrefix("/").Subrouter()

	stsRouter.Methods(http.MethodPost).MatcherFunc(func(r *http.Request, _ *RouteMatch) bool {
		return r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" &&
			r.Header.Get("Authorization") != "" && len(r.URL.RawQuery) == 0
	}).HandlerFunc(nopHandler)
	stsRouter.Methods(http.MethodPost).MatcherFunc(func(r *http.Request, _ *RouteMatch) bool {
		return r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" &&
			len(r.URL.RawQuery) == 0
	}).HandlerFunc(nopHandler)

	for _, action := range []string{"AssumeRoleWithClientGrants", "AssumeRoleWithWebIdentity", "AssumeRoleWithLDAPIdentity", "AssumeRoleWithCertificate", "AssumeRoleWithCustomToken"} {
		stsRouter.Methods(http.MethodPost).HandlerFunc(nopHandler).
			Queries("Action", action).
			Queries("Version", "2011-06-15").
			Queries("Token", "{Token:.*}")
	}
}

func registerMockKMSRouter(router *Router) {
	kmsRouter := router.PathPrefix(mockKMSPathPrefix).Subrouter()
	kmsRouter.Methods(http.MethodPost).Path("/v1/enable").HandlerFunc(nopHandler)
	for _, p := range []string{"/status", "/metrics", "/apis", "/version"} {
		kmsRouter.Methods(http.MethodGet).Path("/v1" + p).HandlerFunc(nopHandler)
	}
	for _, p := range []string{"/key/create", "/key/delete", "/key/import"} {
		kmsRouter.Methods(http.MethodPost).Path("/v1"+p).HandlerFunc(nopHandler).Queries("key-id", "{key-id:.*}")
	}
	kmsRouter.Methods(http.MethodGet).Path("/v1/key/list").HandlerFunc(nopHandler).Queries("pattern", "{pattern:.*}")
}

func registerMockConsoleRouter(router *Router) {
	router.PathPrefix(mockConsolePathPrefix).Subrouter().
		Methods(http.MethodGet).Path("/login-cli").HandlerFunc(nopHandler)
}

func registerMockMCPRouter(router *Router) {
	router.PathPrefix(mockMCPRoutePrefix).Handler(http.HandlerFunc(nopHandler))
}

// registerMockReservedPrefixRouters mirrors the tables, delta sharing and memory
// routers, which are registered on apiRouter ahead of the bucket subrouters.
func registerMockReservedPrefixRouters(apiRouter *Router) {
	tables := apiRouter.PathPrefix(mockTablesRouteRoot).Subrouter()
	for _, p := range []string{"/config", "/namespaces", "/namespaces/{ns}", "/namespaces/{ns}/tables", "/namespaces/{ns}/tables/{table}"} {
		tables.Methods(http.MethodGet).Path(p).HandlerFunc(nopHandler)
		tables.Methods(http.MethodPost).Path(p).HandlerFunc(nopHandler)
	}

	ds := apiRouter.PathPrefix(mockDeltaSharingRouteRoot).Subrouter()
	for _, p := range []string{
		"/shares", "/shares/{share}", "/shares/{share}/schemas",
		"/shares/{share}/schemas/{schema}/tables", "/shares/{share}/all-tables",
		"/health", "/oauth/token",
	} {
		ds.Methods(http.MethodGet).Path(p).HandlerFunc(nopHandler)
		ds.Methods(http.MethodPost).Path(p).HandlerFunc(nopHandler)
	}

	mem := apiRouter.PathPrefix(mockMemoryRouteRoot).Subrouter()
	for _, p := range []string{"/agents", "/agents/{id}", "/secrets", "/secrets/{id}", "/search", "/bio"} {
		mem.Methods(http.MethodGet).Path(p).HandlerFunc(nopHandler)
		mem.Methods(http.MethodPost).Path(p).HandlerFunc(nopHandler)
	}
}

// registerMockAPIRouter mirrors registerAPIRouter, including the bucket DNS
// style (virtual host) subrouter per configured domain. The full S3 route set is
// registered on every one of those subrouters, exactly as AIStor does, so N
// domains multiply the table.
func registerMockAPIRouter(router *Router, domains []string, kubernetes bool) {
	apiRouter := router.PathPrefix("/").Subrouter()

	registerMockReservedPrefixRouters(apiRouter)

	var routers []*Router
	for _, domainName := range domains {
		if kubernetes {
			domain := domainName
			routers = append(routers, apiRouter.MatcherFunc(func(r *http.Request, _ *RouteMatch) bool {
				host := getHost(r)
				if i := strings.IndexByte(host, ':'); i >= 0 {
					host = host[:i]
				}
				return host != "minio."+domain && host != "aistor."+domain
			}).Host("{bucket:.+}."+domainName).Subrouter())
		} else {
			routers = append(routers, apiRouter.Host("{bucket:.+}."+domainName).Subrouter())
		}
	}
	routers = append(routers, apiRouter.PathPrefix("/{bucket:[^_][^/]*}").Subrouter())

	for _, r := range routers {
		registerMockS3Routes(r)
	}

	apiRouter.Methods(http.MethodGet).Path("/").HandlerFunc(nopHandler).Queries("events", "{events:.*}")
	apiRouter.Methods(http.MethodGet).Path("/").HandlerFunc(nopHandler)
	apiRouter.Methods(http.MethodGet).Path("//").HandlerFunc(nopHandler)
	apiRouter.Methods(http.MethodOptions).HandlerFunc(nopHandler)
}

// newAIStorFullRouter assembles the mock server router in configureServerHandler
// order. domains drives the bucket DNS style subrouters; an empty slice yields a
// path-style-only server.
func newAIStorFullRouter(domains []string, kubernetes bool) *Router {
	router := NewRouter().SkipClean(true).UseEncodedPath()
	registerMockDistErasureRouters(router)
	registerMockAdminRouter(router)
	registerMockHealthCheckRouter(router)
	registerMockMetricsRouter(router)
	registerMockSTSRouter(router)
	registerMockKMSRouter(router)
	registerMockConsoleRouter(router)
	registerMockMCPRouter(router)
	registerMockAPIRouter(router, domains, kubernetes)
	return router
}

func countRoutes(router *Router) int {
	n := 0
	router.Walk(func(*Route, *Router, []*Route) error {
		n++
		return nil
	})
	return n
}

type fullBenchCase struct {
	name    string
	method  string
	host    string
	target  string
	headers map[string]string
}

// fullBenchCases spans the layers a real deployment serves. The vhost cases use
// a Host header under the configured domain so the bucket is taken from the host
// rather than the path.
var fullBenchCases = []fullBenchCase{
	{name: "PathStyle_GetObject", method: http.MethodGet, host: "minio.example.net", target: "/testbucket/path/to/my-object.dat"},
	{name: "PathStyle_PutObject", method: http.MethodPut, host: "minio.example.net", target: "/testbucket/path/to/my-object.dat"},
	{name: "PathStyle_HeadObject", method: http.MethodHead, host: "minio.example.net", target: "/testbucket/path/to/my-object.dat"},
	{name: "PathStyle_DeleteObject", method: http.MethodDelete, host: "minio.example.net", target: "/testbucket/path/to/my-object.dat"},
	{name: "PathStyle_ListObjectsV2", method: http.MethodGet, host: "minio.example.net", target: "/testbucket?list-type=2"},

	{name: "VHost_GetObject", method: http.MethodGet, host: "testbucket.s3.example.com", target: "/path/to/my-object.dat"},
	{name: "VHost_PutObject", method: http.MethodPut, host: "testbucket.s3.example.com", target: "/path/to/my-object.dat"},
	{name: "VHost_HeadObject", method: http.MethodHead, host: "testbucket.s3.example.com", target: "/path/to/my-object.dat"},
	{name: "VHost_DeleteObject", method: http.MethodDelete, host: "testbucket.s3.example.com", target: "/path/to/my-object.dat"},
	{name: "VHost_ListObjectsV2", method: http.MethodGet, host: "testbucket.s3.example.com", target: "/?list-type=2"},
	{name: "VHost_PutObjectPart", method: http.MethodPut, host: "testbucket.s3.example.com", target: "/path/to/my-object.dat?partNumber=3&uploadId=abc"},
	{name: "VHost_WithPort", method: http.MethodGet, host: "testbucket.s3.example.com:9000", target: "/path/to/my-object.dat"},

	{name: "Admin", method: http.MethodGet, host: "minio.example.net", target: "/minio/admin/v4/op5"},
	{name: "HealthLive", method: http.MethodGet, host: "minio.example.net", target: "/minio/health/live"},
	{name: "MetricsV3", method: http.MethodGet, host: "minio.example.net", target: "/minio/metrics/v3/cluster"},
	{name: "StorageREST", method: http.MethodPost, host: "minio.example.net", target: "/minio/storage/v60/readall"},
	{name: "NotFound", method: http.MethodPatch, host: "minio.example.net", target: "/testbucket/path/to/my-object.dat"},
}

func (c fullBenchCase) request(tb testing.TB) *http.Request {
	tb.Helper()
	req, err := http.NewRequest(c.method, c.target, nil)
	if err != nil {
		tb.Fatal(err)
	}
	req.Host = c.host
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return req
}

var fullBenchDomains = []string{"s3.example.com"}

// assertBenchCaseMatches checks that a case still reaches a route before it is
// timed, so a change to the mock router cannot quietly turn a benchmark into a
// measurement of different work.
func assertBenchCaseMatches(b *testing.B, router *Router, req *http.Request, name string) {
	b.Helper()
	var match RouteMatch
	matched := router.Match(req, &match)
	if name == "NotFound" {
		if matched || match.MatchErr != ErrMethodMismatch {
			b.Fatalf("%s: expected a method mismatch, got matched=%v err=%v", name, matched, match.MatchErr)
		}
		return
	}
	if !matched || match.MatchErr != nil {
		b.Fatalf("%s: no route matched (err %v)", name, match.MatchErr)
	}
}

func BenchmarkAIStorFullRouter(b *testing.B) {
	router := newAIStorFullRouter(fullBenchDomains, false)
	b.Logf("total routes: %d", countRoutes(router))
	for _, tc := range fullBenchCases {
		req := tc.request(b)
		assertBenchCaseMatches(b, router, req, tc.name)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var match RouteMatch
				router.Match(req, &match)
			}
		})
	}
}

// BenchmarkAIStorFullRouterDomains shows how the table grows with each
// configured domain, since the whole S3 route set is registered per domain.
func BenchmarkAIStorFullRouterDomains(b *testing.B) {
	for _, n := range []int{0, 1, 3} {
		domains := make([]string, n)
		for i := range domains {
			domains[i] = fmt.Sprintf("s3-%d.example.com", i)
		}
		router := newAIStorFullRouter(domains, false)
		for _, tc := range fullBenchCases {
			if tc.name != "PathStyle_PutObject" && tc.name != "VHost_PutObject" {
				continue
			}
			if tc.name == "VHost_PutObject" && n == 0 {
				continue
			}
			req := tc.request(b)
			if tc.name == "VHost_PutObject" {
				req.Host = "testbucket.s3-0.example.com"
			}
			assertBenchCaseMatches(b, router, req, tc.name)
			b.Run(fmt.Sprintf("domains=%d/%s", n, tc.name), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var match RouteMatch
					router.Match(req, &match)
				}
			})
		}
	}
}
