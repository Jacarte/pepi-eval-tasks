// Command gen writes the fleet-config seed workspace.
//
// Usage: go run . -out <dir>
//
// Output is byte-for-byte deterministic: every random choice comes from a
// fixed per-service seed and files are written in sorted order.
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type service struct {
	code, name, team string
	role             string // primary or canary
	replacedBy       string // set when the deployment is decommissioned
}

type env struct{ dir, name string }

var envs = []env{
	{"dev", "development"}, {"qa", "qa"}, {"stg", "staging"}, {"stg-eu", "staging-eu"},
	{"perf", "perf"}, {"prod", "production"}, {"prod-eu", "production-eu"}, {"sandbox", "sandbox"},
}

// The billing deployments come first; the rest pair codenames with generic services.
var services = func() []service {
	s := []service{
		{"heron", "billing-worker", "payments", "primary", ""},
		{"heron-v1", "billing-worker", "payments", "primary", "heron"},
		{"heron-canary", "billing-worker", "payments", "canary", ""},
		{"osprey", "billing-legacy", "payments", "primary", ""},
		{"wren", "billing-api", "payments", "primary", ""},
		{"plover", "billing-scheduler", "payments", "primary", ""},
		{"tern", "invoice-worker", "payments", "primary", ""},
	}
	codes := strings.Fields(`kestrel falcon egret finch lark magpie robin swift thrush vireo warbler
		starling sparrow siskin rook raven petrel oriole nuthatch merlin linnet kite jay ibis hobby
		grebe gannet dunlin curlew crane bunting avocet shrike pipit puffin quail redstart sandpiper
		skua stint teal`)
	names := strings.Fields(`auth-api auth-worker catalog-api catalog-indexer search-api search-indexer
		cart-api checkout-api checkout-worker orders-api orders-worker inventory-api inventory-sync
		pricing-api pricing-worker notifications-api notifications-worker email-sender sms-sender
		reports-api reports-worker ledger-api ledger-worker payouts-api payouts-worker fraud-scorer
		risk-api users-api profiles-api media-api media-transcoder thumbnails-worker audit-api
		audit-worker export-worker import-worker webhooks-api webhooks-worker gateway admin-api`)
	teams := []string{"identity", "commerce", "fulfilment", "platform", "growth", "data"}
	for i, n := range names {
		s = append(s, service{codes[i], n, teams[i%len(teams)], "primary", ""})
	}
	return s
}()

var profiles = []string{"standard", "standard-eu", "low-cost", "short-lived", "archive", "audit", "compliance", "burst", "shared", "legacy"}

// The answer: services/heron/envs/stg/app.yaml includes heron's compliance profile. Every other
// retention value is drawn from retentionPool, which lacks 43.
const answerProfile, answerDays = "compliance", 43

var retentionPool = []int{7, 14, 21, 30, 45, 60, 90, 120, 180, 365}

func main() {
	out := flag.String("out", "", "output directory (replaced)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -out <dir>")
		os.Exit(2)
	}
	files := build()
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if err := os.RemoveAll(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	for _, p := range paths {
		full := filepath.Join(*out, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(full, []byte(files[p]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "gen:", err)
			os.Exit(1)
		}
	}
}

func rngFor(s string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(s))
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

func build() map[string]string {
	files := map[string]string{
		"go.mod":    "module fleet\n\ngo 1.22\n",
		"main.go":   "// Command fleet is a placeholder for deployment tooling; the configuration lives in services/.\npackage main\n\nfunc main() {}\n",
		"README.md": readme,
	}
	orig := rngFor("orig")
	for _, s := range services {
		r := rngFor(s.code)
		dir := "services/" + s.code
		files[dir+"/base.yaml"] = baseYAML(s, r)
		for _, p := range profiles {
			files[dir+"/storage/"+p+".yaml"] = profileYAML(s, p, r)
		}
		for _, e := range envs {
			ed := dir + "/envs/" + e.dir
			files[ed+"/app.yaml"] = appYAML(s, e, r, includeFor(s, e, r))
			files[ed+"/alerts.yaml"] = alertsYAML(e, r)
			files[ed+"/resources.yaml"] = resourcesYAML(e, r)
			files[ed+"/scaling.yaml"] = scalingYAML(e, r)
			files[ed+"/flags.yaml"] = flagsYAML(e, r)
		}
		// Merge leftovers: the billing worker's staging one, plus a few elsewhere so it does not stand out.
		if s.code == "heron" {
			files[dir+"/envs/stg/app.yaml.orig"] = appYAML(s, envs[2], r, "legacy")
		} else if s.team != "payments" && orig.Intn(6) == 0 {
			e := envs[orig.Intn(len(envs))]
			files[dir+"/envs/"+e.dir+"/app.yaml.orig"] = appYAML(s, e, r, includeFor(s, e, r))
		}
	}
	return files
}

const readme = `# fleet

Deployment configuration, one directory per deployment codename.

    services/<codename>/base.yaml           the deployment: service name, role, status, defaults
    services/<codename>/envs/<env>/*.yaml   per-environment settings
    services/<codename>/storage/*.yaml      storage profiles

Each service has exactly one primary, active deployment (role: primary, status: active in
base.yaml). Its configuration is the service's configuration; canary and decommissioned
deployments are not.

A key set in an environment file overrides the same key in base.yaml. "include: <path>"
(relative to the file) loads the profile it names in its place, as if its keys were written
there. Only files ending in .yaml are read.
`

// includeFor picks the storage profile an environment includes ("" for none). Among the
// billing-worker deployments only the staging environments include the answer profile.
func includeFor(s service, e env, r *rand.Rand) string {
	if s.name == "billing-worker" {
		if e.dir == "stg" {
			return answerProfile
		}
		p := profiles[r.Intn(len(profiles)-1)]
		if p == answerProfile {
			p = profiles[len(profiles)-1]
		}
		return p
	}
	if r.Intn(5) == 0 {
		return ""
	}
	return profiles[r.Intn(len(profiles))]
}

func baseYAML(s service, r *rand.Rand) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s deployment\n", s.code)
	fmt.Fprintf(&b, "service: %s\ncodename: %s\nrole: %s\n", s.name, s.code, s.role)
	if s.replacedBy != "" {
		fmt.Fprintf(&b, "status: decommissioned\nreplaced_by: %s\n", s.replacedBy)
	} else {
		b.WriteString("status: active\n")
	}
	fmt.Fprintf(&b, "team: %s\nowner: team-%s@example.com\n", s.team, s.team)
	fmt.Fprintf(&b, "runtime: go1.%d\nreplicas: %d\nlog_level: info\n", 20+r.Intn(3), 1+r.Intn(3))
	fmt.Fprintf(&b, "storage:\n  retention_days: %d\n  encryption: aes256\n", retentionPool[r.Intn(len(retentionPool))])
	fmt.Fprintf(&b, "http:\n  port: %d\n  timeout_ms: %d\n", 8080+r.Intn(4), 1000*(1+r.Intn(5)))
	return b.String()
}

func profileYAML(s service, p string, r *rand.Rand) string {
	days := retentionPool[r.Intn(len(retentionPool))]
	if s.code == "heron" && p == answerProfile {
		days = answerDays
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s storage profile\n", p)
	fmt.Fprintf(&b, "class: %s\nretention_days: %d\n", []string{"hot", "warm", "cold"}[r.Intn(3)], days)
	fmt.Fprintf(&b, "versioning: %t\nencryption: %s\n", r.Intn(2) == 0, []string{"aes256", "kms"}[r.Intn(2)])
	return b.String()
}

func envHeader(e env, what string) string {
	return fmt.Sprintf("# %s %s\nenvironment: %s\n", e.name, what, e.name)
}

func appYAML(s service, e env, r *rand.Rand, include string) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "settings; keys missing here come from ../../base.yaml"))
	fmt.Fprintf(&b, "replicas: %d\nlog_level: %s\n", 1+r.Intn(6), []string{"debug", "info", "warn"}[r.Intn(3)])
	if strings.HasSuffix(s.name, "-worker") || strings.HasSuffix(s.name, "-sender") {
		fmt.Fprintf(&b, "queue:\n  name: %s-%s\n  batch_size: %d\n  visibility_timeout_s: %d\n",
			s.code, e.name, 50*(1+r.Intn(8)), 30*(1+r.Intn(4)))
	}
	if include != "" {
		fmt.Fprintf(&b, "storage:\n  include: ../../storage/%s.yaml\n", include)
	}
	fmt.Fprintf(&b, "http:\n  timeout_ms: %d\n", 500*(1+r.Intn(8)))
	b.WriteString("upstreams:\n")
	for _, u := range upstreams(s, r) {
		fmt.Fprintf(&b, "  - %s\n", u)
	}
	return b.String()
}

func upstreams(s service, r *rand.Rand) []string {
	var u []string
	if s.name == "billing-api" || s.name == "billing-scheduler" || s.name == "invoice-worker" {
		u = append(u, "billing-worker")
	}
	for n := 1 + r.Intn(2); n > 0; n-- {
		o := services[r.Intn(len(services))]
		if o.name != s.name && o.replacedBy == "" && o.role == "primary" {
			u = append(u, o.name)
		}
	}
	return u
}

func alertsYAML(e env, r *rand.Rand) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "alerts"))
	fmt.Fprintf(&b, "log_retention_days: %d\nmetrics_retention_days: %d\n",
		retentionPool[r.Intn(len(retentionPool))], retentionPool[r.Intn(len(retentionPool))])
	b.WriteString("alerts:\n")
	fmt.Fprintf(&b, "  - name: error-rate\n    threshold: 0.0%d\n    for: 5m\n    severity: page\n", 1+r.Intn(9))
	fmt.Fprintf(&b, "  - name: latency-p99\n    threshold_ms: %d\n    for: 10m\n    severity: ticket\n", 100*(2+r.Intn(20)))
	return b.String()
}

func resourcesYAML(e env, r *rand.Rand) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "resources"))
	req := 100 * (1 + r.Intn(5))
	fmt.Fprintf(&b, "cpu_request: %dm\ncpu_limit: %dm\n", req, req*(2+r.Intn(3)))
	mem := 128 * (1 + r.Intn(8))
	fmt.Fprintf(&b, "mem_request_mb: %d\nmem_limit_mb: %d\n", mem, mem*2)
	return b.String()
}

func scalingYAML(e env, r *rand.Rand) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "autoscaling"))
	lo := 1 + r.Intn(3)
	fmt.Fprintf(&b, "min_replicas: %d\nmax_replicas: %d\ntarget_cpu_percent: %d\n", lo, lo+1+r.Intn(10), 50+5*r.Intn(7))
	return b.String()
}

var flagNames = []string{"new_retry_policy", "async_writes", "strict_validation", "batch_compaction", "shadow_reads", "fast_path"}

func flagsYAML(e env, r *rand.Rand) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "feature flags"))
	b.WriteString("flags:\n")
	for _, f := range flagNames {
		if r.Intn(2) == 0 {
			fmt.Fprintf(&b, "  %s: %t\n", f, r.Intn(2) == 0)
		}
	}
	return b.String()
}
