package commons

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/palantir/stacktrace"
	"strconv"
)

// TODO TODO TODO - do we ever need to handle different local host IPs?
const LOCAL_HOST_IP = "0.0.0.0"

type DockerManager struct {
	dockerCtx           context.Context
	dockerClient        *client.Client
	freeHostPortTracker *FreeHostPortTracker
}

func NewDockerManager(dockerCtx context.Context, dockerClient *client.Client, hostPortRangeStart int, hostPortRangeEnd int) (dockerManager *DockerManager, err error) {
	freeHostPortTracker, err := NewFreeHostPortTracker(hostPortRangeStart, hostPortRangeEnd)
	if err != nil {
		return nil, stacktrace.Propagate(err, "")
	}
	return &DockerManager{
		dockerCtx:           dockerCtx,
		dockerClient:        dockerClient,
		freeHostPortTracker: freeHostPortTracker,
	}, nil
}

func (manager DockerManager) CreateAndStartContainerForService(
	// TODO This arg is a hack that will go away as soon as Gecko removes the --public-ip command!
	serviceId int,
	serviceCfg JsonRpcServiceConfig,
	dependencyLivenessReqs map[JsonRpcServiceSocket]JsonRpcRequest) (containerIpAddr string, containerId string, err error) {

	// TODO this relies on serviceId being incremental, and is a total hack until --public-ips flag is gone from Gecko!
	containerConfigPtr, err := manager.getContainerCfgFromServiceCfg(serviceId, serviceCfg, dependencyLivenessReqs)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "Failed to configure container from service.")
	}
	containerHostConfigPtr, err := manager.getContainerHostConfig(serviceCfg)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "Failed to configure host to container mappings from service.")
	}
	// TODO probably use a UUID for the network name (and maybe include test name too)
	resp, err := manager.dockerClient.ContainerCreate(manager.dockerCtx, containerConfigPtr, containerHostConfigPtr, nil, "")
	if err != nil {
		return "", "", stacktrace.Propagate(err, "Could not create Docker container from image %v.", serviceCfg.GetDockerImage())
	}
	containerId = resp.ID
	if err := manager.dockerClient.ContainerStart(manager.dockerCtx, containerId, types.ContainerStartOptions{}); err != nil {
		return "", "", stacktrace.Propagate(err, "Could not start Docker container from image %v.", serviceCfg.GetDockerImage())
	}
	containerJson, err := manager.dockerClient.ContainerInspect(manager.dockerCtx, containerId)
	if err != nil {
		return "","", stacktrace.Propagate(err, "Inspect container failed, which is necessary to get the container's IP")
	}
	containerIpAddr = containerJson.NetworkSettings.IPAddress
	return containerIpAddr, containerId, nil
}

func (manager DockerManager) getFreePort() (freePort *nat.Port, err error) {
	freePortInt, err := manager.freeHostPortTracker.GetFreePort()
	if err != nil {
		return nil, stacktrace.Propagate(err, "")
	}
	port, err := nat.NewPort("tcp", strconv.Itoa(freePortInt))
	if err != nil {
		return nil, stacktrace.Propagate(err, "")
	}
	return &port, nil
}

func (manager DockerManager) getLocalHostIp() string {
	return LOCAL_HOST_IP
}

// Creates a Docker-Container-To-Host Port mapping, defining how a Container's JSON RPC and service-specific ports are
// mapped to the host ports
func (manager *DockerManager) getContainerHostConfig(serviceConfig JsonRpcServiceConfig) (hostConfig *container.HostConfig, err error) {
	freeRpcPort, err := manager.getFreePort()
	if err != nil {
		return nil, stacktrace.Propagate(err, "")
	}

	jsonRpcPortBinding := []nat.PortBinding{
		{
			HostIP: manager.getLocalHostIp(),
			HostPort: freeRpcPort.Port(),
		},
	}

	// TODO cycle through serviceConfig.GetOtherPorts to bind every one, not just default gecko staking port
	freeStakingPort, err := manager.getFreePort()
	if err != nil {
		return nil, stacktrace.Propagate(err, "")
	}
	stakingPortBinding := []nat.PortBinding{
		{
			HostIP: manager.getLocalHostIp(),
			HostPort: freeStakingPort.Port(),
		},
	}

	httpPort, err := nat.NewPort("tcp", strconv.Itoa(serviceConfig.GetJsonRpcPort()))
	// TODO cycle through serviceConfig.getOtherPorts to bind every one, not just gecko staking port
	stakingPort, err := nat.NewPort("tcp", strconv.Itoa(serviceConfig.GetOtherPorts()[0]))
	containerHostConfigPtr := &container.HostConfig{
		PortBindings: nat.PortMap{
			httpPort: jsonRpcPortBinding,
			stakingPort: stakingPortBinding,
		},
	}
	return containerHostConfigPtr, nil
}

// TODO should I actually be passing sorta-complex objects like JsonRpcServiceConfig by value???
// Creates a more generalized Docker Container configuration for Gecko, with a 5-parameter initialization command.
// Gecko HTTP and Staking ports inside the Container are the standard defaults.
func (manager *DockerManager) getContainerCfgFromServiceCfg(
			// TODO This arg is a hack that will go away as soon as Gecko removes the --public-ip command!
			ipAddrOffset int,
			serviceConfig JsonRpcServiceConfig,
			dependencyLivenessReqs map[JsonRpcServiceSocket]JsonRpcRequest) (config *container.Config, err error) {
	jsonRpcPort, err := nat.NewPort("tcp", strconv.Itoa(serviceConfig.GetJsonRpcPort()))
	if err != nil {
		return nil, stacktrace.Propagate(err, "Could not parse port int.")
	}

	portSet := nat.PortSet{
		jsonRpcPort: struct{}{},
	}
	for _, port := range serviceConfig.GetOtherPorts() {
		otherPort, err := nat.NewPort("tcp", strconv.Itoa(port))
		if err != nil {
			return nil, stacktrace.Propagate(err, "Could not parse port int.")
		}
		portSet[otherPort] = struct{}{}
	}

	startCmdArgs := serviceConfig.GetContainerStartCommand(ipAddrOffset, dependencyLivenessReqs)
	nodeConfigPtr := &container.Config{
		Image: serviceConfig.GetDockerImage(),
		// TODO allow modifying of protocol at some point
		ExposedPorts: portSet,
		Cmd: startCmdArgs,
		Tty: false,
	}
	return nodeConfigPtr, nil
}