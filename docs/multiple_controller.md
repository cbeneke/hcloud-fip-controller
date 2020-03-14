## Multiple controller

The controller can be installed multiple times, e.g. to handle
IP-addresses which must only be routed through specific nodes. Take care
to redefine the `LEASE_NAME` variable per deployment.

When the controller was installed via helm, the `LEASE_NAME` is
generated from the deployment name and does not need manual handling.
