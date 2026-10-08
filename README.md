# lgcorzo/mux

![testing](https://github.com/lgcorzo/mux/actions/workflows/test.yml/badge.svg)
[![godoc](https://godoc.org/github.com/lgcorzo/mux?status.svg)](https://godoc.org/github.com/lgcorzo/mux)

> **Sovereign Infrastructure Note**: `lgcorzo/mux` is an actively maintained fork of Gorilla Mux, integrated directly into the **@lgcorzo Sovereign MinIO Ecosystem** and the **Dark Gravity AI Factory**. It serves as a core networking and request routing component for high-throughput S3 REST and STS API operations.

---

## Dark Gravity Factory & Sovereign Support Rationale

This repository is maintained under `@lgcorzo` as part of a sovereign, high-availability software ecosystem designed for autonomous AI infrastructure, strict regulatory compliance, and cloud-native object storage pipelines.

### Why Sovereign Maintenance Matters

1. **Full Supply-Chain Autonomy**: Eliminates dependency on upstream licensing shifts, unannounced deprecations, or sudden repository archived status, ensuring long-term operational predictability.
2. **Dark Gravity Factory Core Integration**: Serves as the critical high-performance HTTP router and multiplexer matching incoming S3 REST, STS, Admin, Health, and Internal Peer API routes across the storage stack.
3. **Compliance & Security Guarantees**: Maintained under zero-CVE SLAs with continuous automated scanning (CodeQL, GoSec, Govulncheck) to maintain compliance with EU AI Act, SOC 2 Type II, and ISO 25059 standards.
4. **Ecosystem Interoperability**: Engineered for native integration across all 38 repositories in `@lgcorzo` (including MinIO Server, MC, KES, Operator, DirectPV, Console, and SIMD-accelerated libraries).

---

## Sovereign MinIO Ecosystem Architecture

```
                                  +---------------------------------------+
                                  |     Dark Gravity AI Orchestrator      |
                                  +-------------------+-------------------+
                                                      |
                                                      v
                                  +---------------------------------------+
                                  |    lgcorzo/mux (Request Router)      |
                                  +-------------------+-------------------+
                                                      |
                         +----------------------------+----------------------------+
                         |                                                         |
                         v                                                         v
         +---------------+---------------+                         +---------------+---------------+
         |     lgcorzo/minio (Server)    |                         |      lgcorzo/kes (KMS)        |
         +---------------+---------------+                         +---------------+---------------+
                         |                                                         |
       +-----------------+-----------------+                     +-----------------+-----------------+
       |                 |                 |                     |                 |                 |
       v                 v                 v                     v                 v                 v
+--------------+  +--------------+  +--------------+      +--------------+  +--------------+  +--------------+
|  lgcorzo/mc  |  |  operator    |  |  directpv    |      | kms-go       |  |  pkg         |  | sha256-simd  |
+--------------+  +--------------+  +--------------+      +--------------+  +--------------+  +--------------+
```

---

## Sovereign Ecosystem Repositories (38 Repositories)

| Category | Repository | Description | Sovereign Role |
| :--- | :--- | :--- | :--- |
| **Core Storage** | `lgcorzo/minio` | High-Performance Object Storage Server | Central Storage Node |
| **Core Storage** | `lgcorzo/mc` | MinIO Client & Management CLI | Administrative Operations |
| **Networking & Routing** | `lgcorzo/mux` | High-Performance Request Router & Dispatcher | S3 REST / STS API Router |
| **Security & KMS** | `lgcorzo/kes` | Key Encryption Service | Cryptographic Key Management |
| **Security & KMS** | `lgcorzo/kms-go` | KMS Client Library | KMS Integration |
| **Orchestration** | `lgcorzo/operator` | MinIO Kubernetes Operator | K8s Lifecycle Automation |
| **Orchestration** | `lgcorzo/directpv` | Direct-Attached Storage CSI Driver | High-Speed Volume Provisioning |
| **Management UI** | `lgcorzo/console` | Web-Based Administration Portal | Storage Stack Dashboard |
| **Utilities & SDKs** | `lgcorzo/minio-go` | Go Client SDK for Amazon S3 | Application Data Access |
| **Utilities & SDKs** | `lgcorzo/madmin-go` | Go Client SDK for MinIO Admin | Management Automation |
| **Utilities & SDKs** | `lgcorzo/pkg` | Shared Utility Functions & Helpers | Core Helper Functions |
| **SIMD & Acceleration** | `lgcorzo/sha256-simd` | Hardware-Accelerated SHA-256 | High-Speed Hashing |
| **SIMD & Acceleration** | `lgcorzo/blake2b-simd` | Hardware-Accelerated BLAKE2b | Fast Cryptographic Hashing |
| **SIMD & Acceleration** | `lgcorzo/md5-simd` | Hardware-Accelerated MD5 | Parallel MD5 Computation |
| **SIMD & Acceleration** | `lgcorzo/simdjson-go` | SIMD-Accelerated JSON Parser | High-Throughput Metadata Parsing |
| **Ecosystem Core** | `lgcorzo/s3select` | S3 Select Query Engine | In-Memory Data Filtering |
| **Ecosystem Core** | `lgcorzo/dsimd` | Dynamic SIMD Dispatch | Runtime Hardware Optimization |
| **Ecosystem Core** | `lgcorzo/csv` | Fast CSV Reader & Writer | Tabular Data Processing |
| **Ecosystem Core** | `lgcorzo/parquet-go` | Parquet File Format Library | Analytics Format Integration |
| **Ecosystem Core** | `lgcorzo/zip` | Streaming ZIP Engine | Archive Manipulation |
| **Ecosystem Core** | `lgcorzo/certgen` | TLS Certificate Generator | Dev/Prod PKI Tooling |
| **Ecosystem Core** | `lgcorzo/sidekick` | High-Performance HTTP Load Balancer | Edge Traffic Distribution |
| **Ecosystem Core** | `lgcorzo/warp` | S3 Benchmarking Tool | Throughput & IOPS Testing |
| **Ecosystem Core** | `lgcorzo/aip` | Automated Installation Package | Zero-Touch Deployment |
| **Ecosystem Core** | `lgcorzo/mcp` | MinIO Control Protocol | Inter-Node Control Bus |
| **Ecosystem Core** | `lgcorzo/mvs` | MinIO Versioning System | Object Lifecycle & Delta Engine |
| **Ecosystem Core** | `lgcorzo/event-notification` | Event Streaming System | Asynchronous Event Pipelines |
| **Ecosystem Core** | `lgcorzo/bucket-replication` | Cross-Region Replication Engine | Multi-Cloud Synchronization |
| **Ecosystem Core** | `lgcorzo/ilm` | Information Lifecycle Management | Data Tiering & Expiration |
| **Ecosystem Core** | `lgcorzo/iam` | Identity and Access Management | Multi-Tenant Authorization |
| **Ecosystem Core** | `lgcorzo/observability` | Prometheus & Tracing Integration | Telemetry Engine |
| **Ecosystem Core** | `lgcorzo/audit-logging` | Structured Audit Log Dispatcher | Compliance Logging |
| **Ecosystem Core** | `lgcorzo/s3-api-tests` | Conformance Test Suite | S3 API Compatibility Verification |
| **Ecosystem Core** | `lgcorzo/benchmarks` | Micro-Benchmark Suite | Low-Level Performance Tuning |
| **Ecosystem Core** | `lgcorzo/helm-charts` | Sovereign Helm Deployment Charts | Cloud-Native Deployment |
| **Ecosystem Core** | `lgcorzo/terraform-provider` | Infrastructure as Code Provider | Automated Provisioning |
| **Ecosystem Core** | `lgcorzo/container-images` | Hardened Container Base Images | Zero-CVE Runtime Containers |
| **Ecosystem Core** | `lgcorzo/dark-gravity-sdk` | Dark Gravity Factory Agent SDK | AI Pipeline Integration |

---

Package `lgcorzo/mux` implements a request router and dispatcher for matching incoming requests to their respective handler.

The name mux stands for "HTTP request multiplexer". Like the standard `http.ServeMux`, `mux.Router` matches incoming requests against a list of registered routes and calls a handler for the route that matches the URL or other conditions. The main features are:

* It implements the `http.Handler` interface so it is compatible with the standard `http.ServeMux`.
* Requests can be matched based on URL host, path, path prefix, schemes, header and query values, HTTP methods or using custom matchers.
* URL hosts, paths and query values can have variables with an optional regular expression.
* Registered URLs can be built, or "reversed", which helps maintaining references to resources.
* Routes can be used as subrouters: nested routes are only tested if the parent route matches. This is useful to define groups of routes that share common conditions like a host, a path prefix or other repeated attributes. As a bonus, this optimizes request matching.

---

* [Install](#install)
* [Examples](#examples)
* [Matching Routes](#matching-routes)
* [Static Files](#static-files)
* [Serving Single Page Applications](#serving-single-page-applications)
* [Registered URLs](#registered-urls)
* [Walking Routes](#walking-routes)
* [Graceful Shutdown](#graceful-shutdown)
* [Middleware](#middleware)
* [Handling CORS Requests](#handling-cors-requests)
* [Testing Handlers](#testing-handlers)
* [Full Example](#full-example)

---

## Install

With a [correctly configured](https://golang.org/doc/install#testing) Go toolchain:

```sh
go get -u github.com/lgcorzo/mux
```

## Examples

Let's start registering a couple of URL paths and handlers:

```go
func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", HomeHandler)
    r.HandleFunc("/products", ProductsHandler)
    r.HandleFunc("/articles", ArticlesHandler)
    http.Handle("/", r)
}
```

Here we register three routes mapping URL paths to handlers. This is equivalent to how `http.HandleFunc()` works: if an incoming request URL matches one of the paths, the corresponding handler is called passing (`http.ResponseWriter`, `*http.Request`) as parameters.

Paths can have variables. They are defined using the format `{name}` or `{name:pattern}`. If a regular expression pattern is not defined, the matched variable will be anything until the next slash. For example:

```go
r := mux.NewRouter()
r.HandleFunc("/products/{key}", ProductHandler)
r.HandleFunc("/articles/{category}/", ArticlesCategoryHandler)
r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
```

The names are used to create a map of route variables which can be retrieved calling `mux.Vars()`:

```go
func ArticlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Category: %v\n", vars["category"])
}
```

And this is all you need to know about the basic usage. More advanced options are explained below.

### Matching Routes

Routes can also be restricted to a domain or subdomain. Just define a host pattern to be matched. They can also have variables:

```go
r := mux.NewRouter()
// Only matches if domain is "www.example.com".
r.Host("www.example.com")
// Matches a dynamic subdomain.
r.Host("{subdomain:[a-z]+}.example.com")
```

There are several other matchers that can be added. To match path prefixes:

```go
r.PathPrefix("/products/")
```

...or HTTP methods:

```go
r.Methods("GET", "POST")
```

...or URL schemes:

```go
r.Schemes("https")
```

...or header values:

```go
r.Headers("X-Requested-With", "XMLHttpRequest")
```

...or query values:

```go
r.Queries("key", "value")
```

...or to use a custom matcher function:

```go
r.MatcherFunc(func(r *http.Request, rm *RouteMatch) bool {
    return r.ProtoMajor == 0
})
```

...and finally, it is possible to combine several matchers in a single route:

```go
r.HandleFunc("/products", ProductsHandler).
  Host("www.example.com").
  Methods("GET").
  Schemes("http")
```

Routes are tested in the order they were added to the router. If two routes match, the first one wins:

```go
r := mux.NewRouter()
r.HandleFunc("/specific", specificHandler)
r.PathPrefix("/").Handler(catchAllHandler)
```

Setting the same matching conditions again and again can be boring, so we have a way to group several routes that share the same requirements. We call it "subrouting".

For example, let's say we have several URLs that should only match when the host is `www.example.com`. Create a route for that host and get a "subrouter" from it:

```go
r := mux.NewRouter()
s := r.Host("www.example.com").Subrouter()
```

Then register routes in the subrouter:

```go
s.HandleFunc("/products/", ProductsHandler)
s.HandleFunc("/products/{key}", ProductHandler)
s.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
```

The three URL paths we registered above will only be tested if the domain is `www.example.com`, because the subrouter is tested first. This is not only convenient, but also optimizes request matching. You can create subrouters combining any attribute matchers accepted by a route.

Subrouters can be used to create domain or path "namespaces": you define subrouters in a central place and then parts of the app can register its paths relatively to a given subrouter.

There's one more thing about subroutes. When a subrouter has a path prefix, the inner routes use it as base for their paths:

```go
r := mux.NewRouter()
s := r.PathPrefix("/products").Subrouter()
// "/products/"
s.HandleFunc("/", ProductsHandler)
// "/products/{key}/"
s.HandleFunc("/{key}/", ProductHandler)
// "/products/{key}/details"
s.HandleFunc("/{key}/details", ProductDetailsHandler)
```


### Static Files

Note that the path provided to `PathPrefix()` represents a "wildcard": calling
`PathPrefix("/static/").Handler(...)` means that the handler will be passed any
request that matches "/static/\*". This makes it easy to serve static files with mux:

```go
func main() {
    var dir string

    flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
    flag.Parse()
    r := mux.NewRouter()

    // This will serve files under http://localhost:8000/static/<filename>
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))

    srv := &http.Server{
        Handler:      r,
        Addr:         "127.0.0.1:8000",
        // Good practice: enforce timeouts for servers you create!
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    log.Fatal(srv.ListenAndServe())
}
```

### Serving Single Page Applications

Most of the time it makes sense to serve your SPA on a separate web server from your API,
but sometimes it's desirable to serve them both from one place. It's possible to write a simple
handler for serving your SPA and leverage mux's powerful routing for your API endpoints.

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lgcorzo/mux"
)

type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)

	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	spa := spaHandler{staticPath: "build", indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)

	srv := &http.Server{
		Handler:      router,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
```

### Registered URLs

Now let's see how to build registered URLs.

Routes can be named. All routes that define a name can have their URLs built, or "reversed". We define a name calling `Name()` on a route. For example:

```go
r := mux.NewRouter()
r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler).
  Name("article")
```

To build a URL, get the route and call the `URL()` method, passing a sequence of key/value pairs for the route variables. For the previous route, we would do:

```go
url, err := r.Get("article").URL("category", "technology", "id", "42")
```

...and the result will be a `url.URL` with the following path:

```
"/articles/technology/42"
```

This also works for host and query value variables:

```go
r := mux.NewRouter()
r.Host("{subdomain}.example.com").
  Path("/articles/{category}/{id:[0-9]+}").
  Queries("filter", "{filter}").
  HandlerFunc(ArticleHandler).
  Name("article")

url, err := r.Get("article").URL("subdomain", "news",
                                 "category", "technology",
                                 "id", "42",
                                 "filter", "gorilla")
```

All variables defined in the route are required, and their values must conform to the corresponding patterns. These requirements guarantee that a generated URL will always match a registered route -- the only exception is for explicitly defined "build-only" routes which never match.

Regex support also exists for matching Headers within a route. For example, we could do:

```go
r.HeadersRegexp("Content-Type", "application/(text|json)")
```

...and the route will match both requests with a Content-Type of `application/json` as well as `application/text`

There's also a way to build only the URL host or path for a route: use the methods `URLHost()` or `URLPath()` instead. For the previous route, we would do:

```go
// "http://news.example.com/"
host, err := r.Get("article").URLHost("subdomain", "news")

// "/articles/technology/42"
path, err := r.Get("article").URLPath("category", "technology", "id", "42")
```

And if you use subrouters, host and path defined separately can be built as well:

```go
r := mux.NewRouter()
s := r.Host("{subdomain}.example.com").Subrouter()
s.Path("/articles/{category}/{id:[0-9]+}").
  HandlerFunc(ArticleHandler).
  Name("article")

// "http://news.example.com/articles/technology/42"
url, err := r.Get("article").URL("subdomain", "news",
                                 "category", "technology",
                                 "id", "42")
```

To find all the required variables for a given route when calling `URL()`, the method `GetVarNames()` is available:
```go
r := mux.NewRouter()
r.Host("{domain}").
    Path("/{group}/{item_id}").
    Queries("some_data1", "{some_data1}").
    Queries("some_data2", "{some_data2}").
    Name("article")

// Will print [domain group item_id some_data1 some_data2] <nil>
fmt.Println(r.Get("article").GetVarNames())

```
### Walking Routes

The `Walk` function on `mux.Router` can be used to visit all of the routes that are registered on a router. For example,
the following prints all of the registered routes:

```go
package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/lgcorzo/mux"
)

func handler(w http.ResponseWriter, r *http.Request) {
	return
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.HandleFunc("/products", handler).Methods("POST")
	r.HandleFunc("/articles", handler).Methods("GET")
	r.HandleFunc("/articles/{id}", handler).Methods("GET", "PUT")
	r.HandleFunc("/authors", handler).Queries("surname", "{surname}")
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	http.Handle("/", r)
}
```

### Graceful Shutdown

Go 1.8 introduced the ability to gracefully shutdown a `*http.Server`. Here's how to do that alongside `mux`:

```go
package main

import (
    "context"
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "github.com/lgcorzo/mux"
)

func main() {
    var wait time.Duration
    flag.DurationVar(&wait, "graceful-timeout", time.Second * 15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
    flag.Parse()

    r := mux.NewRouter()

    srv := &http.Server{
        Addr:         "0.0.0.0:8080",
        WriteTimeout: time.Second * 15,
        ReadTimeout:  time.Second * 15,
        IdleTimeout:  time.Second * 60,
        Handler:      r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Println(err)
        }
    }()

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)

    <-c

    ctx, cancel := context.WithTimeout(context.Background(), wait)
    defer cancel()
    srv.Shutdown(ctx)
    log.Println("shutting down")
    os.Exit(0)
}
```

### Middleware

Mux supports the addition of middlewares to a `Router`, which are executed in the order they are added if a match is found, including its subrouters.

```go
type MiddlewareFunc func(http.Handler) http.Handler
```

A very basic middleware which logs the URI of the request being handled:

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Println(r.RequestURI)
        next.ServeHTTP(w, r)
    })
}
```

Middlewares can be added to a router using `Router.Use()`:

```go
r := mux.NewRouter()
r.HandleFunc("/", handler)
r.Use(loggingMiddleware)
```

### Handling CORS Requests

`CORSMethodMiddleware` intends to make it easier to strictly set the `Access-Control-Allow-Methods` response header.

```go
package main

import (
	"net/http"
	"github.com/lgcorzo/mux"
)

func main() {
    r := mux.NewRouter()

    r.HandleFunc("/foo", fooHandler).Methods(http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodOptions)
    r.Use(mux.CORSMethodMiddleware(r))
    
    http.ListenAndServe(":8080", r)
}

func fooHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    if r.Method == http.MethodOptions {
        return
    }

    w.Write([]byte("foo"))
}
```

## Full Example

Here's a complete, runnable example of a small `mux` based server:

```go
package main

import (
    "net/http"
    "log"
    "github.com/lgcorzo/mux"
)

func YourHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Mux!\n"))
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", YourHandler)

    log.Fatal(http.ListenAndServe(":8000", r))
}
```

## License

BSD licensed. See the [LICENSE](LICENSE) file for details.
