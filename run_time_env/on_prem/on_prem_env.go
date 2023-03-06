package on_prem

import (
	"errors"
	"fmt"
	"os"

	"github.com/VolumezTech/volumez-cloud-provider/cloudprovider"
)

const (
	connectorZoneKey   = "CONNECTOR_ZONE"
	connectorRegionKey = "CONNECTOR_REGION"
	instanceIdKey      = "INSTANCE_ID"
	architectureKey    = "HOSTTYPE"
)

type onPremEnvServiceProvider struct {
	settings map[string]string
}

var _ cloudprovider.ICloudProviderVirtualMachine = (*onPremEnvServiceProvider)(nil)

func NewOnPremEnvServiceProvider() cloudprovider.ICloudProviderVirtualMachine {
	p := &onPremEnvServiceProvider{}
	return p
}

func (provider *onPremEnvServiceProvider) GetName() cloudprovider.CloudProviderType {
	return cloudprovider.CloudProvider_OnPremEnv
}

func (provider *onPremEnvServiceProvider) Init() (err error) {
	keys := []string{
		connectorZoneKey,
		connectorRegionKey,
		instanceIdKey,
		architectureKey,
	}

	values := map[string]string{}
	for _, k := range keys {
		v, exists := os.LookupEnv(k)
		if !exists {
			v = ""
		}
		values[k] = v
	}

	if values[connectorZoneKey] == "" || values[connectorRegionKey] == "" {
		err = errors.New(fmt.Sprintf(`%v or %v is not set`, connectorZoneKey, connectorRegionKey))
		return
	}

	provider.settings = values
	return
}

func (provider *onPremEnvServiceProvider) GetMachineInfo() (info *cloudprovider.MachineInfo, err error) {

	name, _ := os.Hostname() // ignore error

	instanceID, ok := provider.settings[instanceIdKey]
	if !ok || instanceID == "" {
		instanceID = name
	}

	info = &cloudprovider.MachineInfo{
		InstanceID:   instanceID,
		Zone:         provider.settings[connectorZoneKey],
		Region:       provider.settings[connectorRegionKey],
		Architecture: provider.settings[architectureKey],
		IPAddresses:  []string{},
		PublicDNS:    name,
		Additional:   nil,
	}
	return
}

func (provider *onPremEnvServiceProvider) GetVirtualMachineID() (instanceId string, err error) {

	return cloudprovider.GetVirtualMachineID(provider)
}
