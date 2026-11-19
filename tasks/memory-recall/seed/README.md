# fleet

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
effective app.yaml (app.yaml with the winning patches applied, see Patches). Following its
includes ends at a retention_tier. The retention_days of the environment is the entry for that
tier in the retention_days table of regions/<region>.yaml, where <region> is the "region:" of
its effective app.yaml. A tier the region file does not list is taken from the region named by
its "parent:", repeatedly.

Patches. A patch (patches/<id>.yaml) sets keys of the environment directory named by its
"target:"; "set:" nests like app.yaml, and paths in it are relative to the target's app.yaml.
A patch counts when patches/manifest.yaml lists its id under "applied:" and not under
"revoked:". A patch is void when another patch that counts names it in "supersedes:" (that
other patch may itself be void). Patches that do not count or are void have no effect. When
several counting patches that are not void set the same key of a target, the one listed last
under "applied:" wins. A key set by a winning patch replaces the app.yaml value.
