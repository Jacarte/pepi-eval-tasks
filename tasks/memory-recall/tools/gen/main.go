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

var tiers = []string{"ephemeral", "short", "standard", "extended", "regulated", "permanent"}

// Storage templates: a profile includes a fragment, which includes a base; a base fixes the tier.
var baseNames = strings.Fields("bulk cache ledger logs media events snapshots scratch exports sessions backups metrics")

const fragmentCount = 30

func fragmentName(i int) string { return fmt.Sprintf("frag-%02d", i) }

// fragmentTier is the retention tier a fragment resolves to.
func fragmentTier(i int) string { return tiers[(i%len(baseNames))%len(tiers)] }

// The answer: the billing worker's staging environment resolves to the "regulated" tier in region
// eu-west-1 (set by a patch); of eu-west-1 > gdpr-core > emea > global only emea is on the way up
// and lists that tier before global does. Every wrong shortcut lands on a different value.
const answerDays = 217

// The billing worker's staging profile candidates, one tier each. Only "audit" is in effect.
var heronTier = map[string]string{
	"standard": "standard", "archive": "permanent", "audit": "regulated", "burst": "ephemeral", "legacy": "extended",
}

// Tier values pinned on the billing worker's resolution path (and the traps around it); every
// other region draws its tiers at random.
var pinnedRegions = map[string]map[string]int{
	"global":    {"ephemeral": 5, "short": 16, "standard": 31, "extended": 180, "regulated": 340, "permanent": 2555},
	"emea":      {"regulated": answerDays, "standard": 45},
	"gdpr-core": {"extended": 133, "permanent": 900},
	"eu-west-1": {"short": 12, "standard": 61},
	"eu-west-2": {"regulated": 74, "standard": 66},
	"us-east-1": {"regulated": 301},
}

type region struct{ name, parent string }

// Region tree: global > continent > residency group > cloud region. Parents are not derivable
// from the names.
var regions, leafRegions = func() ([]region, []string) {
	rs := []region{{"global", ""}, {"emea", "global"}, {"amer", "global"}, {"apac", "global"}}
	groups := []region{
		{"gdpr-core", "emea"}, {"gdpr-nordic", "emea"}, {"uk-ch", "emea"}, {"mena", "emea"},
		{"us-fed", "amer"}, {"us-commercial", "amer"}, {"latam", "amer"},
		{"apac-core", "apac"}, {"apac-south", "apac"},
	}
	leaves := []region{
		{"eu-west-1", "gdpr-core"}, {"eu-west-2", "uk-ch"}, {"eu-west-3", "gdpr-core"}, {"eu-central-1", "gdpr-core"},
		{"eu-north-1", "gdpr-nordic"}, {"eu-south-1", "gdpr-core"}, {"me-south-1", "mena"}, {"af-south-1", "mena"},
		{"us-east-1", "us-commercial"}, {"us-east-2", "us-commercial"}, {"us-west-1", "us-commercial"},
		{"us-west-2", "us-fed"}, {"ca-central-1", "us-commercial"}, {"sa-east-1", "latam"},
		{"ap-southeast-1", "apac-core"}, {"ap-southeast-2", "apac-core"}, {"ap-northeast-1", "apac-core"},
		{"ap-south-1", "apac-south"},
	}
	var names []string
	for _, l := range leaves {
		names = append(names, l.name)
	}
	return append(append(rs, groups...), leaves...), names
}()

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
			include, rg := includeFor(s, e, r), leafRegions[r.Intn(len(leafRegions))]
			if s.code == "heron" && e.dir == "stg" {
				rg = "eu-west-2"
			}
			files[ed+"/app.yaml"] = appYAML(s, e, r, include, rg)
			files[ed+"/alerts.yaml"] = alertsYAML(e, r)
			files[ed+"/resources.yaml"] = resourcesYAML(e, r)
			files[ed+"/scaling.yaml"] = scalingYAML(e, r)
			files[ed+"/flags.yaml"] = flagsYAML(e, r)
		}
		// Merge leftovers: the billing worker's staging one, plus a few elsewhere so it does not stand out.
		if s.code == "heron" {
			files[dir+"/envs/stg/app.yaml.orig"] = appYAML(s, envs[2], r, "legacy", "eu-west-1")
		} else if s.team != "payments" && orig.Intn(6) == 0 {
			e := envs[orig.Intn(len(envs))]
			files[dir+"/envs/"+e.dir+"/app.yaml.orig"] = appYAML(s, e, r, includeFor(s, e, r), leafRegions[r.Intn(len(leafRegions))])
		}
	}
	for i := 0; i < fragmentCount; i++ {
		r := rngFor(fragmentName(i))
		files["templates/fragments/"+fragmentName(i)+".yaml"] = fmt.Sprintf("# %s\ninclude: ../bases/%s.yaml\nclass: %s\n",
			fragmentName(i), baseNames[i%len(baseNames)], []string{"hot", "warm", "cold"}[r.Intn(3)])
	}
	for j, n := range baseNames {
		files["templates/bases/"+n+".yaml"] = fmt.Sprintf("# %s base\nretention_tier: %s\n", n, tiers[j%len(tiers)])
	}
	for _, rg := range regions {
		files["regions/"+rg.name+".yaml"] = regionYAML(rg)
	}
	addPatches(files)
	return files
}

const readme = `# fleet

Deployment configuration, one directory per deployment codename.

    services/<codename>/base.yaml             the deployment: service name, role, status, defaults
    services/<codename>/envs/<env>/*.yaml     per-environment settings (app.yaml: environment, region, storage)
    services/<codename>/storage/*.yaml        storage profiles
    templates/fragments/*.yaml, templates/bases/*.yaml    shared storage templates
    regions/*.yaml                            retention tables per region
    patches/manifest.yaml, patches/*.yaml     patches to environment settings

Each service has exactly one primary, active deployment (role: primary, status: active in
base.yaml). Its configuration is the service's configuration; canary and decommissioned
deployments are not.

A key set in an environment's app.yaml overrides the same key in base.yaml. "include: <path>"
(relative to the file) loads the file it names in its place, as if its keys were written
there; includes nest. Only files ending in .yaml are read.

Retention. An environment's storage profile is the file named by "storage: include:" in its
app.yaml. Following its includes ends at a retention_tier. The retention_days of the environment
is the entry for that tier in the retention_days table of regions/<region>.yaml, where <region>
is the "region:" of the environment's app.yaml. A tier the region file does not list is taken
from the region named by its "parent:", repeatedly.

Patches. A patch (patches/<id>.yaml) sets keys of the environment directory named by its
"target:", overriding that environment's app.yaml; "set:" nests like app.yaml. A patch counts
when patches/manifest.yaml lists its id under "applied:" and not under "revoked:". A patch is
void when another patch that counts names it in "supersedes:" (that other patch may itself be
void). When several counting patches that are not void set the same key of a target, the one
listed last under "applied:" wins. A key set by a winning patch replaces the app.yaml value.
`

// includeFor picks the storage profile an environment includes. The billing worker's staging
// environment includes "standard"; the patches change that.
func includeFor(s service, e env, r *rand.Rand) string {
	if s.code == "heron" && e.dir == "stg" {
		return "standard"
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
	fmt.Fprintf(&b, "http:\n  port: %d\n  timeout_ms: %d\n", 8080+r.Intn(4), 1000*(1+r.Intn(5)))
	return b.String()
}

func profileYAML(s service, p string, r *rand.Rand) string {
	frag := r.Intn(fragmentCount)
	if want, ok := heronTier[p]; ok && s.code == "heron" {
		for fragmentTier(frag) != want {
			frag = (frag + 1) % fragmentCount
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s storage profile\n", p)
	fmt.Fprintf(&b, "include: ../../../templates/fragments/%s.yaml\n", fragmentName(frag))
	fmt.Fprintf(&b, "versioning: %t\nencryption: %s\n", r.Intn(2) == 0, []string{"aes256", "kms"}[r.Intn(2)])
	return b.String()
}

func regionYAML(rg region) string {
	r := rngFor("region/" + rg.name)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s retention\n", rg.name)
	if rg.parent != "" {
		fmt.Fprintf(&b, "parent: %s\n", rg.parent)
	}
	b.WriteString("retention_days:\n")
	chance := 4 // deeper regions list fewer tiers
	switch rg.parent {
	case "global":
		chance = 2
	case "emea", "amer", "apac":
		chance = 3
	}
	pinned := pinnedRegions[rg.name]
	list := make([]string, 0, len(tiers))
	for _, t := range tiers {
		if _, ok := pinned[t]; ok || (pinned == nil && r.Intn(chance) == 0) {
			list = append(list, t)
		}
	}
	if len(list) == 0 {
		list = append(list, tiers[r.Intn(len(tiers))])
	}
	for _, t := range list {
		d, ok := pinned[t]
		if !ok {
			d = days(r)
		}
		fmt.Fprintf(&b, "  %s: %d\n", t, d)
	}
	return b.String()
}

func days(r *rand.Rand) int { return 5 + r.Intn(395) }

func envHeader(e env, what string) string {
	return fmt.Sprintf("# %s %s\nenvironment: %s\n", e.name, what, e.name)
}

func appYAML(s service, e env, r *rand.Rand, include, region string) string {
	var b strings.Builder
	b.WriteString(envHeader(e, "settings; keys missing here come from ../../base.yaml"))
	fmt.Fprintf(&b, "region: %s\nreplicas: %d\nlog_level: %s\n", region, 1+r.Intn(6), []string{"debug", "info", "warn"}[r.Intn(3)])
	if strings.HasSuffix(s.name, "-worker") || strings.HasSuffix(s.name, "-sender") {
		fmt.Fprintf(&b, "queue:\n  name: %s-%s\n  batch_size: %d\n  visibility_timeout_s: %d\n",
			s.code, e.name, 50*(1+r.Intn(8)), 30*(1+r.Intn(4)))
	}
	fmt.Fprintf(&b, "storage:\n  include: ../../storage/%s.yaml\n", include)
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
		days(r), days(r))
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

type patch struct {
	id, target, supersedes, key, value string
	state                              string // applied, revoked or draft (in neither list)
}

// Patches on the billing worker. On heron's staging environment the one in effect is P-0388
// (P-0371 is void, P-0402 is revoked, P-0395 is a draft) and P-0360 moves it to eu-west-1.
var heronPatches = []patch{
	{"P-0360", "heron/stg", "", "region", "eu-west-1", "applied"},
	{"P-0366", "heron/stg", "", "region", "us-east-1", "revoked"},
	{"P-0371", "heron/stg", "", "storage.include", "../../storage/archive.yaml", "applied"},
	{"P-0388", "heron/stg", "P-0371", "storage.include", "../../storage/audit.yaml", "applied"},
	{"P-0395", "heron/stg", "", "storage.include", "../../storage/legacy.yaml", "draft"},
	{"P-0402", "heron/stg", "P-0388", "storage.include", "../../storage/burst.yaml", "revoked"},
	{"P-0372", "heron-v1/stg", "", "storage.include", "../../storage/compliance.yaml", "applied"},
	{"P-0373", "heron-canary/stg", "", "storage.include", "../../storage/shared.yaml", "applied"},
	{"P-0376", "heron/stg-eu", "", "storage.include", "../../storage/low-cost.yaml", "applied"},
	{"P-0377", "heron/prod", "", "region", "ap-south-1", "applied"},
	{"P-0380", "heron-canary/stg", "", "storage.include", "../../storage/burst.yaml", "applied"},
	{"P-0381", "heron-canary/stg", "", "storage.include", "../../storage/legacy.yaml", "applied"},
}

const genericPatches = 90

func addPatches(files map[string]string) {
	r := rngFor("patches")
	patches := append([]patch(nil), heronPatches...)
	taken := map[string]bool{}
	for _, p := range heronPatches {
		taken[p.id] = true
	}
	var others []service
	for _, s := range services {
		if s.name != "billing-worker" {
			others = append(others, s)
		}
	}
	for len(patches) < len(heronPatches)+genericPatches {
		id := fmt.Sprintf("P-%04d", 1+r.Intn(450))
		if taken[id] {
			continue
		}
		taken[id] = true
		s, e := others[r.Intn(len(others))], envs[r.Intn(len(envs))]
		p := patch{id: id, target: s.code + "/" + e.dir, state: []string{"applied", "applied", "applied", "revoked", "draft"}[r.Intn(5)]}
		switch r.Intn(3) {
		case 0:
			p.key, p.value = "storage.include", "../../storage/"+profiles[r.Intn(len(profiles))]+".yaml"
		case 1:
			p.key, p.value = "region", leafRegions[r.Intn(len(leafRegions))]
		default:
			p.key, p.value = "replicas", fmt.Sprint(1+r.Intn(6))
		}
		patches = append(patches, p)
	}
	sort.Slice(patches, func(i, j int) bool { return patches[i].id < patches[j].id })
	var applied, revoked []string
	for _, p := range patches {
		files["patches/"+p.id+".yaml"] = patchYAML(p)
		switch p.state {
		case "applied":
			applied = append(applied, p.id)
		case "revoked":
			revoked = append(revoked, p.id)
		}
	}
	// P-0388 was a hotfix, applied before the older P-0371.
	for i, id := range applied {
		if id == "P-0388" {
			applied = append([]string{id}, append(applied[:i:i], applied[i+1:]...)...)
			break
		}
	}
	var b strings.Builder
	b.WriteString("# applied lists patches in the order they were applied.\napplied:\n")
	for _, id := range applied {
		fmt.Fprintf(&b, "  - %s\n", id)
	}
	b.WriteString("revoked:\n")
	for _, id := range revoked {
		fmt.Fprintf(&b, "  - %s\n", id)
	}
	files["patches/manifest.yaml"] = b.String()
}

func patchYAML(p patch) string {
	var b strings.Builder
	fmt.Fprintf(&b, "id: %s\ntarget: services/%s\n", p.id, strings.Replace(p.target, "/", "/envs/", 1))
	if p.supersedes != "" {
		fmt.Fprintf(&b, "supersedes: %s\n", p.supersedes)
	}
	b.WriteString("set:\n")
	if p.key == "storage.include" {
		fmt.Fprintf(&b, "  storage:\n    include: %s\n", p.value)
	} else {
		fmt.Fprintf(&b, "  %s: %s\n", p.key, p.value)
	}
	return b.String()
}
