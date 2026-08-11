// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mux

import (
	"net/http"
	"testing"
)

// The benchmarks in this file reproduce the shape of the AIStor S3 API route
// table (cmd/api-router.go registerAPIRouter) so that routing costs can be
// measured against a realistic, deeply nested route set rather than the
// two-route micro benchmarks in bench_test.go.

type rejectedAPI struct {
	methods []string
	queries []string
	path    string
}

var benchRejectedObjAPIs = []rejectedAPI{
	{methods: []string{http.MethodPut, http.MethodDelete, http.MethodGet}, queries: []string{"torrent", ""}, path: "/{object:.+}"},
	{methods: []string{http.MethodDelete}, queries: []string{"acl", ""}, path: "/{object:.+}"},
}

var benchRejectedBucketAPIs = []rejectedAPI{
	{methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete}, queries: []string{"inventory", ""}},
	{methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete}, queries: []string{"metrics", ""}},
	{methods: []string{http.MethodPut}, queries: []string{"website", ""}},
	{methods: []string{http.MethodPut, http.MethodDelete}, queries: []string{"logging", ""}},
	{methods: []string{http.MethodPut, http.MethodDelete}, queries: []string{"accelerate", ""}},
	{methods: []string{http.MethodPut, http.MethodDelete}, queries: []string{"requestPayment", ""}},
	{methods: []string{http.MethodDelete, http.MethodPut, http.MethodHead}, queries: []string{"acl", ""}},
	{methods: []string{http.MethodDelete, http.MethodPut, http.MethodGet}, queries: []string{"publicAccessBlock", ""}},
	{methods: []string{http.MethodDelete, http.MethodPut, http.MethodGet}, queries: []string{"ownershipControls", ""}},
	{methods: []string{http.MethodDelete, http.MethodPut, http.MethodGet}, queries: []string{"intelligent-tiering", ""}},
	{methods: []string{http.MethodDelete, http.MethodPut, http.MethodGet}, queries: []string{"analytics", ""}},
}

func nopHandler(http.ResponseWriter, *http.Request) {}

// discardResponseWriter is a zero-allocation http.ResponseWriter so that
// ServeHTTP benchmarks measure routing rather than response recording.
type discardResponseWriter struct{ header http.Header }

func (d discardResponseWriter) Header() http.Header         { return d.header }
func (d discardResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (d discardResponseWriter) WriteHeader(int)             {}

func newTestResponseWriter() discardResponseWriter {
	return discardResponseWriter{header: make(http.Header)}
}

// registerAIStorAPIRouter mirrors registerAPIRouter from AIStor, in
// registration order, with every handler replaced by a no-op.
func registerAIStorAPIRouter(router *Router) {
	apiRouter := router.PathPrefix("/").Subrouter()

	bucketRouter := apiRouter.PathPrefix("/{bucket:[^_][^/]*}").Subrouter()
	registerMockS3Routes(bucketRouter)
	// Root operations.
	apiRouter.Methods(http.MethodGet).Path("/").HandlerFunc(nopHandler).Queries("events", "{events:.*}")
	apiRouter.Methods(http.MethodGet).Path("/").HandlerFunc(nopHandler)
	apiRouter.Methods(http.MethodGet).Path("//").HandlerFunc(nopHandler)
	apiRouter.Methods(http.MethodOptions).HandlerFunc(nopHandler)
}

func newAIStorRouter() *Router {
	router := NewRouter().SkipClean(true).UseEncodedPath()
	registerAIStorAPIRouter(router)
	return router
}

var aistorBenchCases = []struct {
	name    string
	method  string
	target  string
	headers map[string]string
}{
	{name: "HeadObject", method: http.MethodHead, target: "/testbucket/path/to/my-object.dat"},
	{name: "GetObject", method: http.MethodGet, target: "/testbucket/path/to/my-object.dat"},
	{name: "PutObject", method: http.MethodPut, target: "/testbucket/path/to/my-object.dat"},
	{name: "DeleteObject", method: http.MethodDelete, target: "/testbucket/path/to/my-object.dat"},
	{name: "PutObjectPart", method: http.MethodPut, target: "/testbucket/path/to/my-object.dat?partNumber=3&uploadId=abc123"},
	{name: "ListObjectsV2", method: http.MethodGet, target: "/testbucket?list-type=2&prefix=path%2Fto&max-keys=1000"},
	{name: "ListBuckets", method: http.MethodGet, target: "/"},
	{name: "NotFound", method: http.MethodPatch, target: "/testbucket/path/to/my-object.dat"},
}

func BenchmarkAIStorRouter(b *testing.B) {
	router := newAIStorRouter()
	for _, tc := range aistorBenchCases {
		req, err := http.NewRequest(tc.method, tc.target, nil)
		if err != nil {
			b.Fatal(err)
		}
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		var match RouteMatch
		matched := router.Match(req, &match)
		if tc.name == "NotFound" {
			if matched || match.MatchErr != ErrMethodMismatch {
				b.Fatalf("%s: expected a method mismatch, got matched=%v err=%v", tc.name, matched, match.MatchErr)
			}
		} else if !matched || match.MatchErr != nil {
			b.Fatalf("%s: no route matched (err %v)", tc.name, match.MatchErr)
		}
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

func BenchmarkAIStorRouterServeHTTP(b *testing.B) {
	router := newAIStorRouter()
	for _, tc := range aistorBenchCases {
		req, err := http.NewRequest(tc.method, tc.target, nil)
		if err != nil {
			b.Fatal(err)
		}
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		b.Run(tc.name, func(b *testing.B) {
			w := newTestResponseWriter()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				router.ServeHTTP(w, req)
			}
		})
	}
}

// registerMockS3Routes registers the S3 API route set, in AIStor's registration
// order, on one bucket subrouter. AIStor runs the equivalent loop body once per
// bucket DNS style subrouter and once for the path style subrouter.
func registerMockS3Routes(bucketRouter *Router) {

	for _, r := range benchRejectedObjAPIs {
		t := bucketRouter.Methods(r.methods...).HandlerFunc(nopHandler).Queries(r.queries...)
		t.Path(r.path)
	}

	// Object operations.
	bucketRouter.Methods(http.MethodHead).Path("/{object:.+}").HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("attributes", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").
		HeadersRegexp("X-Amz-Copy-Source", ".*?(\\/|%2F).*?").
		HandlerFunc(nopHandler).Queries("partNumber", "{partNumber:.*}", "uploadId", "{uploadId:.*}")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).
		Queries("partNumber", "{partNumber:.*}", "uploadId", "{uploadId:.*}")
	bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("uploadId", "{uploadId:.*}")
	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("uploadId", "{uploadId:.*}")
	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("uploads", "")
	bucketRouter.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("uploadId", "{uploadId:.*}")

	for _, q := range []string{"acl", "tagging", "retention", "legal-hold"} {
		bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler).Queries(q, "")
	}
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("acl", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("tagging", "")
	bucketRouter.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("tagging", "")

	for _, m := range []string{http.MethodPut, http.MethodGet, http.MethodDelete} {
		bucketRouter.Methods(m).Path("/{object:.+}").HandlerFunc(nopHandler).
			Queries("annotation", "", "annotationName", "{annotationName:.+}")
	}
	bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("annotation", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("annotation", "")
	bucketRouter.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("annotation", "")

	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).
		Queries("select", "").Queries("select-type", "2")
	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("lock", "")
	bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("lambdaArn", "{lambdaArn:.*}")
	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("lambdaArn", "{lambdaArn:.*}")

	// GetObject.
	bucketRouter.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(nopHandler)

	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").
		HeadersRegexp("X-Amz-Rename-Source", ".+").HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").
		HeadersRegexp("X-Amz-Copy-Source", ".*?(\\/|%2F).*?").HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("retention", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("legal-hold", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("encryption", "")
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").
		HeadersRegexp("X-Amz-Meta-Snowball-Auto-Extract", "true").HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").
		HeadersRegexp("X-Amz-Write-Offset-Bytes", "").HandlerFunc(nopHandler)

	// PutObject.
	bucketRouter.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(nopHandler)
	// DeleteObject.
	bucketRouter.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodPost).Path("/{object:.+}").HandlerFunc(nopHandler).Queries("restore", "")

	// Bucket operations.
	for _, q := range []string{
		"location", "policy", "cors", "lifecycle", "encryption", "session", "qos", "qos-metrics",
		"object-lock", "replication", "versioning", "notification",
	} {
		bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries(q, "")
	}
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("events", "{events:.*}")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("replication-reset-status", "")
	for _, q := range []string{"acl", "website", "accelerate", "requestPayment", "logging", "tagging"} {
		bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries(q, "")
	}
	bucketRouter.Methods(http.MethodPut).HandlerFunc(nopHandler).Queries("acl", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("website", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("tagging", "")

	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("uploads", "")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("list-type", "2", "metadata", "true")
	// ListObjectsV2.
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("list-type", "2")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("versions", "", "metadata", "true")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("versions", "")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("policyStatus", "")

	for _, q := range []string{
		"lifecycle", "replication", "encryption", "policy", "qos", "cors", "object-lock",
		"tagging", "versioning", "notification", "replication-reset", "replication-reset-cancel",
	} {
		bucketRouter.Methods(http.MethodPut).HandlerFunc(nopHandler).Queries(q, "")
	}

	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("id", "{id:.*}").Queries("generate", "")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("id", "{id:.*}").Queries("status", "")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("id", "{id:.*}")
	bucketRouter.Methods(http.MethodPut).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("id", "{id:.*}")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("id", "{id:.*}")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).
		Queries("minio-inventory", "").Queries("continuation-token", "{continuationToken:.*}")

	// PutBucket / HeadBucket.
	bucketRouter.Methods(http.MethodPut).HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodHead).HandlerFunc(nopHandler)

	bucketRouter.Methods(http.MethodPost).
		MatcherFunc(func(r *http.Request, _ *RouteMatch) bool {
			return r.Header.Get("Content-Type") == "multipart/form-data"
		}).HandlerFunc(nopHandler)

	bucketRouter.Methods(http.MethodPost).HandlerFunc(nopHandler).Queries("delete", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("policy", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("cors", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("replication", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("lifecycle", "")
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler).Queries("encryption", "")
	// DeleteBucket.
	bucketRouter.Methods(http.MethodDelete).HandlerFunc(nopHandler)

	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("replication-metrics", "2")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("replication-metrics", "")
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler).Queries("replication-check", "")

	for _, r := range benchRejectedBucketAPIs {
		bucketRouter.Methods(r.methods...).HandlerFunc(nopHandler).Queries(r.queries...)
	}

	// ListObjectsV1.
	bucketRouter.Methods(http.MethodGet).HandlerFunc(nopHandler)
	bucketRouter.Methods(http.MethodOptions).HandlerFunc(nopHandler)
}
