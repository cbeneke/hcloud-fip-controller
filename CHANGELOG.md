* v0.4.0
  * Ips will now get redistributed onto multiple nodes (#32)
  * Hetzner floating ips can now be autodiscovered (#21)
  * **IMPORTANT** setting specific floating ips in config is now deprecated and will be removed soon. Please use floating ip label selector instead if only some ips should be used
  * Add tests for all important functions (#6)

* v0.3.5
  * Leader election lease renew duration is now configurable (#49)

* v0.3.4
  * Fix error message outPut for undefined options (#46)
  
* v0.3.3
  * Fix nilpointer deref on kubernetes v1.18.2 (#38)

* v0.3.2
  * Fix parameter validation

* v0.3.1
  * Include internal Hetzner IPs in server to node matching (#25)

* v0.3.0
  * Restructured project layout (no intended functional changes)

* v0.2.1
  * Added debug logs (disabled per default)

* v0.2.0
  * Added minimal logging
  * **IMPORTANT** config.json options switched from camelCase to snake_case

* v0.1.0
  * Initial preview release
