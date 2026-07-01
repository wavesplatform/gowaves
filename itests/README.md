### About

Integration Tests (itests) test Waves Scala and Go nodes for the correct identical behavior on all types of transactions

### Prerequisites

Docker must be installed prior to running Integration Tests
 * [How to install Docker on Mac](https://docs.docker.com/desktop/install/mac-install/) 
 * [How to install Docker on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)

Verify the correct installation
   ```sh
sudo docker --version
   ```
### Usage
   ```sh
   make itests
   ```

## Notes On Docker API Version

Starting with Docker Engine v29, the minimum supported API version was raised to 1.44 (see https://www.docker.com/blog/docker-engine-version-29).

The v3 version of the `dockertest` library used in our integration tests may initialize
the Docker client with an API version lower than the daemon’s minimum, which leads to
connection failures.

To avoid this issue, we explicitly set the `DOCKER_API_VERSION` environment variable
to `1.45` before creating the Docker pool.

We also set `DOCKER_MACHINE_NAME` environment variable to `local` to ensure the client uses 
the expected local Docker environment.

If needed, you can override this behavior by defining `DOCKER_API_VERSION` and
`DOCKER_MACHINE_NAME` in your environment before running the integration tests.