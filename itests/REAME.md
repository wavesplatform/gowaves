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