package on_prem

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VolumezTech/volumez-cloud-provider/cloudprovider"
)

const (
	DefaultConfigFilename = "/opt/vlzconnector/machine_info.json"
)

//
// Config file example
// {
//     "machine_info" :{
//         "instance_id" : "my_machine",
//         "zone": "z1",
//         "region": "r1",
//         "public_dns": "my_host.volumez.com"
//     }
// }

type MachineInfo struct {
	InstanceID string `json:"instance_id"`
	Zone       string `json:"zone"`
	Region     string `json:"region"`
	PublicDNS  string `json:"public_dns"`
}

type Config struct {
	Machine MachineInfo `json:"machine_info"`
}

type onPremConfigServiceProvider struct {
	filename string
	info     *MachineInfo
}

var _ cloudprovider.ICloudProviderVirtualMachine = (*onPremEnvServiceProvider)(nil)

func NewOnPremConfigServiceProvider(filename string) cloudprovider.ICloudProviderVirtualMachine {
	p := &onPremConfigServiceProvider{filename: filename}
	return p
}

func NewOnPremConfigServiceProviderDefault() cloudprovider.ICloudProviderVirtualMachine {
	return NewOnPremConfigServiceProvider(DefaultConfigFilename)
}

func (provider *onPremConfigServiceProvider) GetName() cloudprovider.CloudProviderType {
	return cloudprovider.CloudProvider_OnPremConfig
}

func (provider *onPremConfigServiceProvider) Init() (err error) {
	absPath, _ := filepath.Abs(provider.filename)
	content, err := os.ReadFile(absPath)

	if err != nil {
		return
	}

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return
	}
	info := config.Machine

	if info.Zone == "" {
		// Required
		err = errors.New(fmt.Sprintf(`%v: invalid config - zone parameter is required`, provider.filename))
		return
	}

	name, _ := os.Hostname() // ignore error
	if info.InstanceID == "" {
		info.InstanceID = name
	}

	provider.info = &info
	return
}

func (provider *onPremConfigServiceProvider) GetMachineInfo() (info *cloudprovider.MachineInfo, err error) {

	if provider.info == nil {
		return nil, errors.New("no valid config file found")
	}

	arch, _ := os.LookupEnv(architectureKey)

	info = &cloudprovider.MachineInfo{
		InstanceID:   provider.info.InstanceID,
		Zone:         provider.info.Zone,
		Region:       provider.info.Region,
		Architecture: arch,
		IPAddresses:  []string{},
		PublicDNS:    provider.info.PublicDNS,
		Additional:   nil,
	}
	return
}

func (provider *onPremConfigServiceProvider) GetVirtualMachineID() (instanceId string, err error) {

	return cloudprovider.GetVirtualMachineID(provider)
}
