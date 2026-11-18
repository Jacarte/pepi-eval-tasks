# fleet

Deployment configuration for every service, one directory per service codename.

    services/<codename>/base.yaml           defaults of the service
    services/<codename>/envs/<env>/*.yaml   per-environment settings

A key set in an environment file overrides the same key in base.yaml.
