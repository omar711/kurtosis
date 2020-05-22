* [ ] TestSuiteRunner needs to throw an error if a user double-registers a test with the same name
* [ ] Upgrade code to pull images from Docker Hub, not just locally
* [ ] Implement FreeHostPortTracker
    1. [ ] Create thread-safe object that tracks free ports with:
        * [ ] Method to dole out free ports
        * [ ] Method to release ports
    2. [ ] Create in the TestSuiteRunner's RunTests method
    3. [ ] Pass to JsonRpcServiceNetworkConfig (probably during CreateAndRun)
* [ ] Make JsonRpcServiceNetworkConfig's CreateAndRun method create a Docker network with a UUID
* [ ] Create a multi-node Ava network config provider
    * [ ] Modify GeckoServiceConfig to allow for waiting for boot nodes to start up
        * [ ] Implement inter-node dependencies for JsonRpcServiceConfig
            1. [ ] Implement a fluent Builder for JsonRpcServiceConfig
                1. [ ] Switch current constructor to Builder
                2. [ ] Implement a method to add nodes
            2. [ ] Modify the Builder's AddService method to also allow depending on existing service
            3. [ ] Modify CreateAndRun command to start nodes in the correct order, passing the LivenessRequests from dependencies to the dependents
                1. [ ] TODO TODO TODO
        * [ ] Make GeckoServiceConfig modify its start command based on if it had boot nodes passed in or not
            * [ ] Inside the GetStartCommand method, wrap the actual start command with an image-specific busy loop that checks to make sure at least one boot node is up (NOTE: we should really just have them change Gecko to keep retrying boot nodes until it gets one!!!)
* [ ] Use nat.Port objects to represent ports
    * [ ] Switch JsonRpcServiceNetwork's port fields
    * [ ] Switch JsonRpcServiceConfig's port field
* [ ] Ascertain whether we should be passing structs around by value, or by reference
* [ ] Implement graceful cleanup of Docker containers in TestSuiteRunner that's impossible to skip (even if an exception is thrown)