# fleet

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
