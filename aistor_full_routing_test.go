// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"flag"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
)

// fullRoutingConfigs covers the server layouts AIStor produces from
// globalDomainNames and IsKubernetes(): path style only, one and two bucket DNS
// style domains, and the Kubernetes variant whose MatcherFunc excludes the
// reserved minio.<domain> and aistor.<domain> hosts.
var fullRoutingConfigs = []struct {
	name       string
	domains    []string
	kubernetes bool
}{
	{name: "pathstyle-only", domains: nil},
	{name: "one-domain", domains: []string{"s3.example.com"}},
	{name: "two-domains", domains: []string{"s3.example.com", "s3.example.net"}},
	{name: "kubernetes", domains: []string{"s3.example.com"}, kubernetes: true},
}

// fullRoutingCorpus exercises bucket DNS style addressing alongside path style
// and the non-S3 layers. Host matters as much as target here: the same target
// resolves to a different route depending on whether the host names a bucket.
var fullRoutingCorpus = []struct{ method, host, target string }{
	// Path style.
	{http.MethodGet, "minio.example.org", "/"},
	{http.MethodGet, "minio.example.org", "/testbucket"},
	{http.MethodGet, "minio.example.org", "/testbucket/obj"},
	{http.MethodPut, "minio.example.org", "/testbucket/obj"},
	{http.MethodHead, "minio.example.org", "/testbucket/obj"},
	{http.MethodDelete, "minio.example.org", "/testbucket/obj"},
	{http.MethodGet, "minio.example.org", "/testbucket?list-type=2"},
	{http.MethodPost, "minio.example.org", "/testbucket?delete="},

	// Bucket DNS style against the first domain.
	{http.MethodGet, "testbucket.s3.example.com", "/"},
	{http.MethodGet, "testbucket.s3.example.com", "/obj"},
	{http.MethodPut, "testbucket.s3.example.com", "/obj"},
	{http.MethodHead, "testbucket.s3.example.com", "/obj"},
	{http.MethodDelete, "testbucket.s3.example.com", "/obj"},
	{http.MethodGet, "testbucket.s3.example.com", "/a/b/c.dat"},
	{http.MethodGet, "testbucket.s3.example.com", "/?list-type=2"},
	{http.MethodGet, "testbucket.s3.example.com", "/obj?acl="},
	{http.MethodPut, "testbucket.s3.example.com", "/obj?partNumber=2&uploadId=xyz"},
	{http.MethodPost, "testbucket.s3.example.com", "/obj?uploads="},
	{http.MethodGet, "testbucket.s3.example.com", "/obj?uploadId=xyz"},
	{http.MethodPost, "testbucket.s3.example.com", "/?delete="},
	{http.MethodOptions, "testbucket.s3.example.com", "/obj"},
	{http.MethodPatch, "testbucket.s3.example.com", "/obj"},

	// Ports must not defeat host matching.
	{http.MethodGet, "testbucket.s3.example.com:9000", "/obj"},
	{http.MethodPut, "testbucket.s3.example.com:443", "/obj"},

	// Dotted and nested bucket names, which {bucket:.+} accepts.
	{http.MethodGet, "my.dotted.bucket.s3.example.com", "/obj"},
	{http.MethodGet, "a.s3.example.com", "/obj"},

	// The second domain, and a host under neither domain.
	{http.MethodGet, "testbucket.s3.example.net", "/obj"},
	{http.MethodGet, "testbucket.s3.example.org", "/obj"},
	{http.MethodGet, "s3.example.com", "/testbucket/obj"},

	// Reserved hosts the Kubernetes matcher excludes from bucket DNS style so
	// that they stay path style.
	{http.MethodGet, "minio.s3.example.com", "/testbucket/obj"},
	{http.MethodGet, "aistor.s3.example.com", "/testbucket/obj"},
	{http.MethodGet, "minio.s3.example.com", "/obj"},

	// Reserved path prefixes must not be taken for buckets.
	{http.MethodGet, "minio.example.org", "/_iceberg/v1/config"},
	{http.MethodGet, "minio.example.org", "/_delta-sharing/v1/shares"},
	{http.MethodGet, "minio.example.org", "/_mem/v1/agents"},
	{http.MethodGet, "testbucket.s3.example.com", "/_iceberg/v1/config"},

	// Non-S3 layers.
	{http.MethodGet, "minio.example.org", "/minio/admin/v4/op5"},
	{http.MethodGet, "minio.example.org", "/minio/admin/v3/op5"},
	{http.MethodGet, "minio.example.org", "/minio/admin/v4/op0?q0="},
	{http.MethodGet, "minio.example.org", "/minio/health/live"},
	{http.MethodHead, "minio.example.org", "/minio/health/cluster"},
	{http.MethodGet, "minio.example.org", "/minio/metrics/v3/cluster"},
	{http.MethodGet, "minio.example.org", "/minio/prometheus/metrics"},
	{http.MethodPost, "minio.example.org", "/minio/storage/v60/readall"},
	{http.MethodPost, "minio.example.org", "/minio/peer/v52/health"},
	{http.MethodGet, "minio.example.org", "/minio/kms/v1/status"},
	{http.MethodGet, "minio.example.org", "/minio/console/login-cli"},
	{http.MethodGet, "minio.example.org", "/minio/mcp/anything"},
	{http.MethodPost, "minio.example.org", "/?Action=AssumeRoleWithWebIdentity&Version=2011-06-15&Token=t"},

	// Same non-S3 targets under a bucket host, where the bucket subrouter is
	// tried first.
	{http.MethodGet, "testbucket.s3.example.com", "/minio/health/live"},
	{http.MethodGet, "testbucket.s3.example.com", "/minio/admin/v4/op5"},
}

// dumpFullRouting renders the routing decision for every corpus entry across
// every server layout.
func dumpFullRouting(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, cfg := range fullRoutingConfigs {
		router := newAIStorFullRouter(cfg.domains, cfg.kubernetes)
		nameRoutesByIndex(router)
		fmt.Fprintf(&b, "### %s (domains=%v kubernetes=%v routes=%d)\n", cfg.name, cfg.domains, cfg.kubernetes, countRoutes(router))
		for _, c := range fullRoutingCorpus {
			req, err := http.NewRequest(c.method, c.target, nil)
			if err != nil {
				t.Fatalf("%s %s: %v", c.method, c.target, err)
			}
			req.Host = c.host

			var match RouteMatch
			ok := router.Match(req, &match)

			name := "<none>"
			if match.Route != nil {
				name = match.Route.name
			}
			vars := make([]string, 0, len(match.Vars))
			for k, v := range match.Vars {
				vars = append(vars, k+"="+v)
			}
			sort.Strings(vars)
			errStr := "nil"
			if match.MatchErr != nil {
				errStr = match.MatchErr.Error()
			}
			fmt.Fprintf(&b, "%-7s %-34s %-42s ok=%-5v route=%s vars=[%s] err=%s\n",
				c.method, c.host, c.target, ok, name, strings.Join(vars, " "), errStr)
		}
	}
	return b.String()
}

// TestAIStorFullRouting pins routing across the whole mocked server, and in
// particular across bucket DNS style addressing: which route wins, the bucket
// taken from the host, port trimming, dotted bucket names, the Kubernetes
// reserved-host exclusion, and the interaction between a bucket host and the
// reserved path prefixes.
//
// The golden output was generated against the pre-optimization revision
// (166a2ea), so a diff means routing behaviour changed rather than only its
// cost. Routes are identified by registration index, so any edit to the mock
// router shifts every index and the golden must be regenerated wholesale.
func TestAIStorFullRouting(t *testing.T) {
	compareGolden(t, "full routing", dumpFullRouting(t), strings.TrimPrefix(aistorFullRoutingGolden, "\n"))
}

// updateAIStorGolden makes the golden tests print their regenerated output
// instead of a line-by-line diff. Any edit to the mock routers shifts every
// route index, so the golden has to be replaced wholesale.
var updateAIStorGolden = flag.Bool("update-aistor-golden", false,
	"print regenerated AIStor routing goldens instead of diffing them")

// compareGolden reports how got differs from want, or prints got in full when
// the goldens are being regenerated.
func compareGolden(t *testing.T, name, got, want string) {
	t.Helper()
	if got == want {
		return
	}
	if *updateAIStorGolden {
		// Printed rather than logged so the block is unindented and can be
		// pasted straight back into the golden constant.
		fmt.Print(got)
		t.Fatalf("%s: golden is stale; the regenerated block was printed above", name)
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			t.Errorf("%s line %d:\n got: %s\nwant: %s", name, i+1, g, w)
		}
	}
}

const aistorFullRoutingGolden = `
### pathstyle-only (domains=[] kubernetes=false routes=610)
GET     minio.example.org                  /                                          ok=true  route=r607 vars=[] err=nil
GET     minio.example.org                  /testbucket                                ok=true  route=r604 vars=[bucket=testbucket] err=nil
GET     minio.example.org                  /testbucket/obj                            ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     minio.example.org                  /testbucket/obj                            ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
HEAD    minio.example.org                  /testbucket/obj                            ok=true  route=r497 vars=[bucket=testbucket object=obj] err=nil
DELETE  minio.example.org                  /testbucket/obj                            ok=true  route=r531 vars=[bucket=testbucket object=obj] err=nil
GET     minio.example.org                  /testbucket?list-type=2                    ok=true  route=r558 vars=[bucket=testbucket] err=nil
POST    minio.example.org                  /testbucket?delete=                        ok=true  route=r583 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /                                          ok=true  route=r607 vars=[] err=nil
GET     testbucket.s3.example.com          /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
PUT     testbucket.s3.example.com          /obj                                       ok=true  route=r580 vars=[bucket=obj] err=nil
HEAD    testbucket.s3.example.com          /obj                                       ok=true  route=r581 vars=[bucket=obj] err=nil
DELETE  testbucket.s3.example.com          /obj                                       ok=true  route=r589 vars=[bucket=obj] err=nil
GET     testbucket.s3.example.com          /a/b/c.dat                                 ok=true  route=r522 vars=[bucket=a object=b/c.dat] err=nil
GET     testbucket.s3.example.com          /?list-type=2                              ok=true  route=r607 vars=[] err=nil
GET     testbucket.s3.example.com          /obj?acl=                                  ok=true  route=r547 vars=[bucket=obj] err=nil
PUT     testbucket.s3.example.com          /obj?partNumber=2&uploadId=xyz             ok=true  route=r580 vars=[bucket=obj] err=nil
POST    testbucket.s3.example.com          /obj?uploads=                              ok=false route=<none> vars=[] err=method is not allowed
GET     testbucket.s3.example.com          /obj?uploadId=xyz                          ok=true  route=r604 vars=[bucket=obj] err=nil
POST    testbucket.s3.example.com          /?delete=                                  ok=false route=<none> vars=[] err=method is not allowed
OPTIONS testbucket.s3.example.com          /obj                                       ok=true  route=r605 vars=[bucket=obj] err=nil
PATCH   testbucket.s3.example.com          /obj                                       ok=false route=<none> vars=[] err=method is not allowed
GET     testbucket.s3.example.com:9000     /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
PUT     testbucket.s3.example.com:443      /obj                                       ok=true  route=r580 vars=[bucket=obj] err=nil
GET     my.dotted.bucket.s3.example.com    /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
GET     a.s3.example.com                   /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
GET     testbucket.s3.example.net          /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
GET     testbucket.s3.example.org          /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
GET     s3.example.com                     /testbucket/obj                            ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /testbucket/obj                            ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
GET     aistor.s3.example.com              /testbucket/obj                            ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /obj                                       ok=true  route=r604 vars=[bucket=obj] err=nil
GET     minio.example.org                  /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /_delta-sharing/v1/shares                  ok=true  route=r467 vars=[] err=nil
GET     minio.example.org                  /_mem/v1/agents                            ok=true  route=r482 vars=[] err=nil
GET     testbucket.s3.example.com          /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v3/op5                        ok=true  route=r227 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op0?q0=                    ok=true  route=r522 vars=[bucket=minio object=admin/v4/op0] err=nil
GET     minio.example.org                  /minio/health/live                         ok=true  route=r422 vars=[] err=nil
HEAD    minio.example.org                  /minio/health/cluster                      ok=true  route=r420 vars=[] err=nil
GET     minio.example.org                  /minio/metrics/v3/cluster                  ok=true  route=r431 vars=[pathComps=/cluster] err=nil
GET     minio.example.org                  /minio/prometheus/metrics                  ok=true  route=r426 vars=[] err=nil
POST    minio.example.org                  /minio/storage/v60/readall                 ok=true  route=r015 vars=[] err=nil
POST    minio.example.org                  /minio/peer/v52/health                     ok=true  route=r028 vars=[] err=nil
GET     minio.example.org                  /minio/kms/v1/status                       ok=true  route=r443 vars=[] err=nil
GET     minio.example.org                  /minio/console/login-cli                   ok=true  route=r452 vars=[] err=nil
GET     minio.example.org                  /minio/mcp/anything                        ok=true  route=r453 vars=[] err=nil
POST    minio.example.org                  /?Action=AssumeRoleWithWebIdentity&Version=2011-06-15&Token=t ok=true  route=r437 vars=[Token=t] err=nil
GET     testbucket.s3.example.com          /minio/health/live                         ok=true  route=r422 vars=[] err=nil
GET     testbucket.s3.example.com          /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
### one-domain (domains=[s3.example.com] kubernetes=false routes=722)
GET     minio.example.org                  /                                          ok=true  route=r719 vars=[] err=nil
GET     minio.example.org                  /testbucket                                ok=true  route=r716 vars=[bucket=testbucket] err=nil
GET     minio.example.org                  /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
PUT     minio.example.org                  /testbucket/obj                            ok=true  route=r642 vars=[bucket=testbucket object=obj] err=nil
HEAD    minio.example.org                  /testbucket/obj                            ok=true  route=r609 vars=[bucket=testbucket object=obj] err=nil
DELETE  minio.example.org                  /testbucket/obj                            ok=true  route=r643 vars=[bucket=testbucket object=obj] err=nil
GET     minio.example.org                  /testbucket?list-type=2                    ok=true  route=r670 vars=[bucket=testbucket] err=nil
POST    minio.example.org                  /testbucket?delete=                        ok=true  route=r695 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /                                          ok=true  route=r604 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
HEAD    testbucket.s3.example.com          /obj                                       ok=true  route=r497 vars=[bucket=testbucket object=obj] err=nil
DELETE  testbucket.s3.example.com          /obj                                       ok=true  route=r531 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /a/b/c.dat                                 ok=true  route=r522 vars=[bucket=testbucket object=a/b/c.dat] err=nil
GET     testbucket.s3.example.com          /?list-type=2                              ok=true  route=r558 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj?acl=                                  ok=true  route=r505 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj?partNumber=2&uploadId=xyz             ok=true  route=r500 vars=[bucket=testbucket object=obj partNumber=2 uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /obj?uploads=                              ok=true  route=r503 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /obj?uploadId=xyz                          ok=true  route=r501 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /?delete=                                  ok=true  route=r583 vars=[bucket=testbucket] err=nil
OPTIONS testbucket.s3.example.com          /obj                                       ok=true  route=r605 vars=[bucket=testbucket] err=nil
PATCH   testbucket.s3.example.com          /obj                                       ok=false route=<none> vars=[] err=method is not allowed
GET     testbucket.s3.example.com:9000     /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com:443      /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
GET     my.dotted.bucket.s3.example.com    /obj                                       ok=true  route=r522 vars=[bucket=my.dotted.bucket object=obj] err=nil
GET     a.s3.example.com                   /obj                                       ok=true  route=r522 vars=[bucket=a object=obj] err=nil
GET     testbucket.s3.example.net          /obj                                       ok=true  route=r716 vars=[bucket=obj] err=nil
GET     testbucket.s3.example.org          /obj                                       ok=true  route=r716 vars=[bucket=obj] err=nil
GET     s3.example.com                     /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /testbucket/obj                            ok=true  route=r522 vars=[bucket=minio object=testbucket/obj] err=nil
GET     aistor.s3.example.com              /testbucket/obj                            ok=true  route=r522 vars=[bucket=aistor object=testbucket/obj] err=nil
GET     minio.s3.example.com               /obj                                       ok=true  route=r522 vars=[bucket=minio object=obj] err=nil
GET     minio.example.org                  /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /_delta-sharing/v1/shares                  ok=true  route=r467 vars=[] err=nil
GET     minio.example.org                  /_mem/v1/agents                            ok=true  route=r482 vars=[] err=nil
GET     testbucket.s3.example.com          /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v3/op5                        ok=true  route=r227 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op0?q0=                    ok=true  route=r634 vars=[bucket=minio object=admin/v4/op0] err=nil
GET     minio.example.org                  /minio/health/live                         ok=true  route=r422 vars=[] err=nil
HEAD    minio.example.org                  /minio/health/cluster                      ok=true  route=r420 vars=[] err=nil
GET     minio.example.org                  /minio/metrics/v3/cluster                  ok=true  route=r431 vars=[pathComps=/cluster] err=nil
GET     minio.example.org                  /minio/prometheus/metrics                  ok=true  route=r426 vars=[] err=nil
POST    minio.example.org                  /minio/storage/v60/readall                 ok=true  route=r015 vars=[] err=nil
POST    minio.example.org                  /minio/peer/v52/health                     ok=true  route=r028 vars=[] err=nil
GET     minio.example.org                  /minio/kms/v1/status                       ok=true  route=r443 vars=[] err=nil
GET     minio.example.org                  /minio/console/login-cli                   ok=true  route=r452 vars=[] err=nil
GET     minio.example.org                  /minio/mcp/anything                        ok=true  route=r453 vars=[] err=nil
POST    minio.example.org                  /?Action=AssumeRoleWithWebIdentity&Version=2011-06-15&Token=t ok=true  route=r437 vars=[Token=t] err=nil
GET     testbucket.s3.example.com          /minio/health/live                         ok=true  route=r422 vars=[] err=nil
GET     testbucket.s3.example.com          /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
### two-domains (domains=[s3.example.com s3.example.net] kubernetes=false routes=834)
GET     minio.example.org                  /                                          ok=true  route=r831 vars=[] err=nil
GET     minio.example.org                  /testbucket                                ok=true  route=r828 vars=[bucket=testbucket] err=nil
GET     minio.example.org                  /testbucket/obj                            ok=true  route=r746 vars=[bucket=testbucket object=obj] err=nil
PUT     minio.example.org                  /testbucket/obj                            ok=true  route=r754 vars=[bucket=testbucket object=obj] err=nil
HEAD    minio.example.org                  /testbucket/obj                            ok=true  route=r721 vars=[bucket=testbucket object=obj] err=nil
DELETE  minio.example.org                  /testbucket/obj                            ok=true  route=r755 vars=[bucket=testbucket object=obj] err=nil
GET     minio.example.org                  /testbucket?list-type=2                    ok=true  route=r782 vars=[bucket=testbucket] err=nil
POST    minio.example.org                  /testbucket?delete=                        ok=true  route=r807 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /                                          ok=true  route=r604 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
HEAD    testbucket.s3.example.com          /obj                                       ok=true  route=r497 vars=[bucket=testbucket object=obj] err=nil
DELETE  testbucket.s3.example.com          /obj                                       ok=true  route=r531 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /a/b/c.dat                                 ok=true  route=r522 vars=[bucket=testbucket object=a/b/c.dat] err=nil
GET     testbucket.s3.example.com          /?list-type=2                              ok=true  route=r558 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj?acl=                                  ok=true  route=r505 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj?partNumber=2&uploadId=xyz             ok=true  route=r500 vars=[bucket=testbucket object=obj partNumber=2 uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /obj?uploads=                              ok=true  route=r503 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /obj?uploadId=xyz                          ok=true  route=r501 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /?delete=                                  ok=true  route=r583 vars=[bucket=testbucket] err=nil
OPTIONS testbucket.s3.example.com          /obj                                       ok=true  route=r605 vars=[bucket=testbucket] err=nil
PATCH   testbucket.s3.example.com          /obj                                       ok=false route=<none> vars=[] err=method is not allowed
GET     testbucket.s3.example.com:9000     /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com:443      /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
GET     my.dotted.bucket.s3.example.com    /obj                                       ok=true  route=r522 vars=[bucket=my.dotted.bucket object=obj] err=nil
GET     a.s3.example.com                   /obj                                       ok=true  route=r522 vars=[bucket=a object=obj] err=nil
GET     testbucket.s3.example.net          /obj                                       ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.org          /obj                                       ok=true  route=r828 vars=[bucket=obj] err=nil
GET     s3.example.com                     /testbucket/obj                            ok=true  route=r746 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /testbucket/obj                            ok=true  route=r522 vars=[bucket=minio object=testbucket/obj] err=nil
GET     aistor.s3.example.com              /testbucket/obj                            ok=true  route=r522 vars=[bucket=aistor object=testbucket/obj] err=nil
GET     minio.s3.example.com               /obj                                       ok=true  route=r522 vars=[bucket=minio object=obj] err=nil
GET     minio.example.org                  /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /_delta-sharing/v1/shares                  ok=true  route=r467 vars=[] err=nil
GET     minio.example.org                  /_mem/v1/agents                            ok=true  route=r482 vars=[] err=nil
GET     testbucket.s3.example.com          /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v3/op5                        ok=true  route=r227 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op0?q0=                    ok=true  route=r746 vars=[bucket=minio object=admin/v4/op0] err=nil
GET     minio.example.org                  /minio/health/live                         ok=true  route=r422 vars=[] err=nil
HEAD    minio.example.org                  /minio/health/cluster                      ok=true  route=r420 vars=[] err=nil
GET     minio.example.org                  /minio/metrics/v3/cluster                  ok=true  route=r431 vars=[pathComps=/cluster] err=nil
GET     minio.example.org                  /minio/prometheus/metrics                  ok=true  route=r426 vars=[] err=nil
POST    minio.example.org                  /minio/storage/v60/readall                 ok=true  route=r015 vars=[] err=nil
POST    minio.example.org                  /minio/peer/v52/health                     ok=true  route=r028 vars=[] err=nil
GET     minio.example.org                  /minio/kms/v1/status                       ok=true  route=r443 vars=[] err=nil
GET     minio.example.org                  /minio/console/login-cli                   ok=true  route=r452 vars=[] err=nil
GET     minio.example.org                  /minio/mcp/anything                        ok=true  route=r453 vars=[] err=nil
POST    minio.example.org                  /?Action=AssumeRoleWithWebIdentity&Version=2011-06-15&Token=t ok=true  route=r437 vars=[Token=t] err=nil
GET     testbucket.s3.example.com          /minio/health/live                         ok=true  route=r422 vars=[] err=nil
GET     testbucket.s3.example.com          /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
### kubernetes (domains=[s3.example.com] kubernetes=true routes=722)
GET     minio.example.org                  /                                          ok=true  route=r719 vars=[] err=nil
GET     minio.example.org                  /testbucket                                ok=true  route=r716 vars=[bucket=testbucket] err=nil
GET     minio.example.org                  /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
PUT     minio.example.org                  /testbucket/obj                            ok=true  route=r642 vars=[bucket=testbucket object=obj] err=nil
HEAD    minio.example.org                  /testbucket/obj                            ok=true  route=r609 vars=[bucket=testbucket object=obj] err=nil
DELETE  minio.example.org                  /testbucket/obj                            ok=true  route=r643 vars=[bucket=testbucket object=obj] err=nil
GET     minio.example.org                  /testbucket?list-type=2                    ok=true  route=r670 vars=[bucket=testbucket] err=nil
POST    minio.example.org                  /testbucket?delete=                        ok=true  route=r695 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /                                          ok=true  route=r604 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
HEAD    testbucket.s3.example.com          /obj                                       ok=true  route=r497 vars=[bucket=testbucket object=obj] err=nil
DELETE  testbucket.s3.example.com          /obj                                       ok=true  route=r531 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /a/b/c.dat                                 ok=true  route=r522 vars=[bucket=testbucket object=a/b/c.dat] err=nil
GET     testbucket.s3.example.com          /?list-type=2                              ok=true  route=r558 vars=[bucket=testbucket] err=nil
GET     testbucket.s3.example.com          /obj?acl=                                  ok=true  route=r505 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com          /obj?partNumber=2&uploadId=xyz             ok=true  route=r500 vars=[bucket=testbucket object=obj partNumber=2 uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /obj?uploads=                              ok=true  route=r503 vars=[bucket=testbucket object=obj] err=nil
GET     testbucket.s3.example.com          /obj?uploadId=xyz                          ok=true  route=r501 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
POST    testbucket.s3.example.com          /?delete=                                  ok=true  route=r583 vars=[bucket=testbucket] err=nil
OPTIONS testbucket.s3.example.com          /obj                                       ok=true  route=r605 vars=[bucket=testbucket] err=nil
PATCH   testbucket.s3.example.com          /obj                                       ok=false route=<none> vars=[] err=method is not allowed
GET     testbucket.s3.example.com:9000     /obj                                       ok=true  route=r522 vars=[bucket=testbucket object=obj] err=nil
PUT     testbucket.s3.example.com:443      /obj                                       ok=true  route=r530 vars=[bucket=testbucket object=obj] err=nil
GET     my.dotted.bucket.s3.example.com    /obj                                       ok=true  route=r522 vars=[bucket=my.dotted.bucket object=obj] err=nil
GET     a.s3.example.com                   /obj                                       ok=true  route=r522 vars=[bucket=a object=obj] err=nil
GET     testbucket.s3.example.net          /obj                                       ok=true  route=r716 vars=[bucket=obj] err=nil
GET     testbucket.s3.example.org          /obj                                       ok=true  route=r716 vars=[bucket=obj] err=nil
GET     s3.example.com                     /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
GET     aistor.s3.example.com              /testbucket/obj                            ok=true  route=r634 vars=[bucket=testbucket object=obj] err=nil
GET     minio.s3.example.com               /obj                                       ok=true  route=r716 vars=[bucket=obj] err=nil
GET     minio.example.org                  /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /_delta-sharing/v1/shares                  ok=true  route=r467 vars=[] err=nil
GET     minio.example.org                  /_mem/v1/agents                            ok=true  route=r482 vars=[] err=nil
GET     testbucket.s3.example.com          /_iceberg/v1/config                        ok=true  route=r456 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v3/op5                        ok=true  route=r227 vars=[] err=nil
GET     minio.example.org                  /minio/admin/v4/op0?q0=                    ok=true  route=r634 vars=[bucket=minio object=admin/v4/op0] err=nil
GET     minio.example.org                  /minio/health/live                         ok=true  route=r422 vars=[] err=nil
HEAD    minio.example.org                  /minio/health/cluster                      ok=true  route=r420 vars=[] err=nil
GET     minio.example.org                  /minio/metrics/v3/cluster                  ok=true  route=r431 vars=[pathComps=/cluster] err=nil
GET     minio.example.org                  /minio/prometheus/metrics                  ok=true  route=r426 vars=[] err=nil
POST    minio.example.org                  /minio/storage/v60/readall                 ok=true  route=r015 vars=[] err=nil
POST    minio.example.org                  /minio/peer/v52/health                     ok=true  route=r028 vars=[] err=nil
GET     minio.example.org                  /minio/kms/v1/status                       ok=true  route=r443 vars=[] err=nil
GET     minio.example.org                  /minio/console/login-cli                   ok=true  route=r452 vars=[] err=nil
GET     minio.example.org                  /minio/mcp/anything                        ok=true  route=r453 vars=[] err=nil
POST    minio.example.org                  /?Action=AssumeRoleWithWebIdentity&Version=2011-06-15&Token=t ok=true  route=r437 vars=[Token=t] err=nil
GET     testbucket.s3.example.com          /minio/health/live                         ok=true  route=r422 vars=[] err=nil
GET     testbucket.s3.example.com          /minio/admin/v4/op5                        ok=true  route=r065 vars=[] err=nil
`
