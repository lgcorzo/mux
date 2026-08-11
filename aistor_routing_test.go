package mux

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
)

// nameRoutesByIndex labels every route with its registration index so that a
// routing decision can be compared as a stable string.
func nameRoutesByIndex(r *Router) {
	i := 0
	r.Walk(func(route *Route, _ *Router, _ []*Route) error {
		route.name = fmt.Sprintf("r%03d", i)
		i++
		return nil
	})
}

var aistorCorpus = []struct{ method, target string }{
	{"GET", "/"},
	{"GET", "//"},
	{"GET", "/testbucket"},
	{"GET", "/testbucket/"},
	{"HEAD", "/testbucket"},
	{"PUT", "/testbucket"},
	{"DELETE", "/testbucket"},
	{"OPTIONS", "/testbucket"},
	{"GET", "/testbucket?list-type=2"},
	{"GET", "/testbucket?list-type=2&metadata=true"},
	{"GET", "/testbucket?versions="},
	{"GET", "/testbucket?uploads="},
	{"GET", "/testbucket?location="},
	{"GET", "/testbucket?policy="},
	{"GET", "/testbucket?acl="},
	{"PUT", "/testbucket?acl="},
	{"DELETE", "/testbucket?policy="},
	{"GET", "/testbucket?replication-metrics=2"},
	{"GET", "/testbucket?replication-metrics="},
	{"GET", "/testbucket?minio-inventory=&id=7"},
	{"GET", "/testbucket?minio-inventory=&id=7&generate="},
	{"POST", "/testbucket?delete="},
	{"GET", "/testbucket/obj"},
	{"HEAD", "/testbucket/obj"},
	{"PUT", "/testbucket/obj"},
	{"DELETE", "/testbucket/obj"},
	{"POST", "/testbucket/obj"},
	{"GET", "/testbucket/a/b/c/d.dat"},
	{"PUT", "/testbucket/a/b/c/d.dat"},
	{"GET", "/testbucket/obj?acl="},
	{"PUT", "/testbucket/obj?acl="},
	{"GET", "/testbucket/obj?tagging="},
	{"GET", "/testbucket/obj?attributes="},
	{"GET", "/testbucket/obj?uploadId=xyz"},
	{"PUT", "/testbucket/obj?partNumber=2&uploadId=xyz"},
	{"POST", "/testbucket/obj?uploads="},
	{"POST", "/testbucket/obj?uploadId=xyz"},
	{"DELETE", "/testbucket/obj?uploadId=xyz"},
	{"GET", "/testbucket/obj?annotation=&annotationName=n1"},
	{"GET", "/testbucket/obj?annotation="},
	{"PUT", "/testbucket/obj?annotation="},
	{"GET", "/testbucket/obj?torrent="},
	{"DELETE", "/testbucket/obj?acl="},
	{"GET", "/testbucket/obj?retention="},
	{"GET", "/testbucket/obj?legal-hold="},
	{"POST", "/testbucket/obj?select=&select-type=2"},
	{"GET", "/testbucket/obj?lambdaArn=arn:x"},
	{"POST", "/testbucket/obj?restore="},
	{"GET", "/_iceberg/x"},
	{"GET", "/_delta-sharing"},
	{"PATCH", "/testbucket/obj"},
	{"PATCH", "/testbucket"},
	{"GET", "/testbucket/a%2Fb/c"},
	{"PUT", "/testbucket/a%2Fb/c"},
	{"GET", "/testbucket/obj%20with%20spaces"},
	{"GET", "/bucket.with.dots/obj"},
	{"GET", "/testbucket/obj?uploadId="},
	{"PUT", "/testbucket/obj?partNumber=&uploadId="},
	{"GET", "/testbucket?events=x"},
	{"GET", "/?events=x"},
}

// dumpRouting renders the router's decision for every corpus entry.
func dumpRouting(t *testing.T) string {
	t.Helper()
	router := newAIStorRouter()
	nameRoutesByIndex(router)

	var b strings.Builder
	for _, c := range aistorCorpus {
		req, err := http.NewRequest(c.method, c.target, nil)
		if err != nil {
			t.Fatalf("%s %s: %v", c.method, c.target, err)
		}
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
		fmt.Fprintf(&b, "%-7s %-46s ok=%-5v route=%s vars=[%s] err=%s\n",
			c.method, c.target, ok, name, strings.Join(vars, " "), errStr)
	}
	return b.String()
}

const aistorRoutingGolden = `
GET     /                                              ok=true  route=r114 vars=[] err=nil
GET     //                                             ok=false route=<none> vars=[] err=no matching route was found
GET     /testbucket                                    ok=true  route=r111 vars=[bucket=testbucket] err=nil
GET     /testbucket/                                   ok=true  route=r111 vars=[bucket=testbucket] err=nil
HEAD    /testbucket                                    ok=true  route=r088 vars=[bucket=testbucket] err=nil
PUT     /testbucket                                    ok=true  route=r087 vars=[bucket=testbucket] err=nil
DELETE  /testbucket                                    ok=true  route=r096 vars=[bucket=testbucket] err=nil
OPTIONS /testbucket                                    ok=true  route=r112 vars=[bucket=testbucket] err=nil
GET     /testbucket?list-type=2                        ok=true  route=r065 vars=[bucket=testbucket] err=nil
GET     /testbucket?list-type=2&metadata=true          ok=true  route=r064 vars=[bucket=testbucket] err=nil
GET     /testbucket?versions=                          ok=true  route=r067 vars=[bucket=testbucket] err=nil
GET     /testbucket?uploads=                           ok=true  route=r063 vars=[bucket=testbucket] err=nil
GET     /testbucket?location=                          ok=true  route=r040 vars=[bucket=testbucket] err=nil
GET     /testbucket?policy=                            ok=true  route=r041 vars=[bucket=testbucket] err=nil
GET     /testbucket?acl=                               ok=true  route=r054 vars=[bucket=testbucket] err=nil
PUT     /testbucket?acl=                               ok=true  route=r060 vars=[bucket=testbucket] err=nil
DELETE  /testbucket?policy=                            ok=true  route=r091 vars=[bucket=testbucket] err=nil
GET     /testbucket?replication-metrics=2              ok=true  route=r097 vars=[bucket=testbucket] err=nil
GET     /testbucket?replication-metrics=               ok=true  route=r098 vars=[bucket=testbucket] err=nil
GET     /testbucket?minio-inventory=&id=7              ok=true  route=r083 vars=[bucket=testbucket id=7] err=nil
GET     /testbucket?minio-inventory=&id=7&generate=    ok=true  route=r081 vars=[bucket=testbucket id=7] err=nil
POST    /testbucket?delete=                            ok=true  route=r090 vars=[bucket=testbucket] err=nil
GET     /testbucket/obj                                ok=true  route=r029 vars=[bucket=testbucket object=obj] err=nil
HEAD    /testbucket/obj                                ok=true  route=r004 vars=[bucket=testbucket object=obj] err=nil
PUT     /testbucket/obj                                ok=true  route=r037 vars=[bucket=testbucket object=obj] err=nil
DELETE  /testbucket/obj                                ok=true  route=r038 vars=[bucket=testbucket object=obj] err=nil
POST    /testbucket/obj                                ok=false route=<none> vars=[] err=method is not allowed
GET     /testbucket/a/b/c/d.dat                        ok=true  route=r029 vars=[bucket=testbucket object=a/b/c/d.dat] err=nil
PUT     /testbucket/a/b/c/d.dat                        ok=true  route=r037 vars=[bucket=testbucket object=a/b/c/d.dat] err=nil
GET     /testbucket/obj?acl=                           ok=true  route=r012 vars=[bucket=testbucket object=obj] err=nil
PUT     /testbucket/obj?acl=                           ok=true  route=r016 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?tagging=                       ok=true  route=r013 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?attributes=                    ok=true  route=r005 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?uploadId=xyz                   ok=true  route=r008 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
PUT     /testbucket/obj?partNumber=2&uploadId=xyz      ok=true  route=r007 vars=[bucket=testbucket object=obj partNumber=2 uploadId=xyz] err=nil
POST    /testbucket/obj?uploads=                       ok=true  route=r010 vars=[bucket=testbucket object=obj] err=nil
POST    /testbucket/obj?uploadId=xyz                   ok=true  route=r009 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
DELETE  /testbucket/obj?uploadId=xyz                   ok=true  route=r011 vars=[bucket=testbucket object=obj uploadId=xyz] err=nil
GET     /testbucket/obj?annotation=&annotationName=n1  ok=true  route=r020 vars=[annotationName=n1 bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?annotation=                    ok=true  route=r022 vars=[bucket=testbucket object=obj] err=nil
PUT     /testbucket/obj?annotation=                    ok=true  route=r023 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?torrent=                       ok=true  route=r002 vars=[bucket=testbucket object=obj] err=nil
DELETE  /testbucket/obj?acl=                           ok=true  route=r003 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?retention=                     ok=true  route=r014 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?legal-hold=                    ok=true  route=r015 vars=[bucket=testbucket object=obj] err=nil
POST    /testbucket/obj?select=&select-type=2          ok=true  route=r025 vars=[bucket=testbucket object=obj] err=nil
GET     /testbucket/obj?lambdaArn=arn:x                ok=true  route=r027 vars=[bucket=testbucket lambdaArn=arn:x object=obj] err=nil
POST    /testbucket/obj?restore=                       ok=true  route=r039 vars=[bucket=testbucket object=obj] err=nil
GET     /_iceberg/x                                    ok=false route=<none> vars=[] err=method is not allowed
GET     /_delta-sharing                                ok=false route=<none> vars=[] err=method is not allowed
PATCH   /testbucket/obj                                ok=false route=<none> vars=[] err=method is not allowed
PATCH   /testbucket                                    ok=false route=<none> vars=[] err=method is not allowed
GET     /testbucket/a%2Fb/c                            ok=true  route=r029 vars=[bucket=testbucket object=a%2Fb/c] err=nil
PUT     /testbucket/a%2Fb/c                            ok=true  route=r037 vars=[bucket=testbucket object=a%2Fb/c] err=nil
GET     /testbucket/obj%20with%20spaces                ok=true  route=r029 vars=[bucket=testbucket object=obj%20with%20spaces] err=nil
GET     /bucket.with.dots/obj                          ok=true  route=r029 vars=[bucket=bucket.with.dots object=obj] err=nil
GET     /testbucket/obj?uploadId=                      ok=true  route=r008 vars=[bucket=testbucket object=obj uploadId=] err=nil
PUT     /testbucket/obj?partNumber=&uploadId=          ok=true  route=r007 vars=[bucket=testbucket object=obj partNumber= uploadId=] err=nil
GET     /testbucket?events=x                           ok=true  route=r052 vars=[bucket=testbucket events=x] err=nil
GET     /?events=x                                     ok=true  route=r113 vars=[events=x] err=nil
`

// TestAIStorRouting pins the route chosen, the variables extracted and the match
// error for a corpus of S3 requests against the AIStor route table. The golden
// output was generated before the path matching fast paths were added, so a diff
// here means routing behaviour changed, not just its cost.
//
// Routes are identified by registration index, which makes the comparison exact
// but couples it to registerAIStorAPIRouter: editing the reproduction shifts
// every index, so the golden has to be regenerated wholesale and a diff only
// carries meaning while the reproduction is untouched.
func TestAIStorRouting(t *testing.T) {
	compareGolden(t, "s3 routing", dumpRouting(t), strings.TrimPrefix(aistorRoutingGolden, "\n"))
}
