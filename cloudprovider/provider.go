package cloudprovider

import (
	"fmt"
	"strings"
)

type AdditionalParam struct {
	Key   string
	Value string
}

func (param *AdditionalParam) ToText() string {
	return fmt.Sprintf(`%-27v%v`, param.Key, param.Value)
}

type MachineInfo struct {
	InstanceID   string
	Zone         string
	Region       string
	Architecture string
	IPAddresses  []string
	PublicDNS    string
	Cluster      string
	Additional   []AdditionalParam
}

func (info *MachineInfo) ToText() string {
	arr := []string{
		fmt.Sprintf(`InstanceID:                %v`, info.InstanceID),
		fmt.Sprintf(`Zone:                      %v`, info.Zone),
		fmt.Sprintf(`Region:                    %v`, info.Region),
		fmt.Sprintf(`Architecture:              %v`, info.Architecture),
		fmt.Sprintf(`IPAddresses:               %v`, info.IPAddresses),
		fmt.Sprintf(`Public DNS:                %v`, info.PublicDNS),
	}
	if info.Cluster != "" {
		arr = append(arr, fmt.Sprintf(`Cluster:                   %v`, info.Cluster))
	}
	if info.Additional != nil {
		arr = append(arr, "==== Additional ====")
		for _, p := range info.Additional {
			arr = append(arr, p.ToText())
		}
	}
	return strings.Join(arr, "\n")
}

type ICloudProviderVirtualMachine interface {
	GetName() CloudProviderType
	Init() error
	GetVirtualMachineID() (string, error)
	GetMachineInfo() (info *MachineInfo, err error)
}

type ServiceProviderConstructor func() ICloudProviderVirtualMachine

func GetVirtualMachineID(provider ICloudProviderVirtualMachine) (instanceId string, err error) {
	info, err := provider.GetMachineInfo()
	if err == nil {
		instanceId = info.InstanceID
	}
	return
}

type CloudProviderType string

const (
	CloudProvider_Aws          CloudProviderType = "AWS"
	CloudProvider_Azure        CloudProviderType = "Azure"
	CloudProvider_OnPremConfig CloudProviderType = "OnPrem/Config"
	CloudProvider_OnPremEnv    CloudProviderType = "OnPrem/ENV"
)

var supportedCloudProviders = []CloudProviderType{
	CloudProvider_Aws,
	CloudProvider_Azure,
	CloudProvider_OnPremConfig,
	CloudProvider_OnPremEnv,
}

func ConvertToCloudProviderType(s string) (t CloudProviderType, err error) {

	stringToCloudProviderType := map[string]CloudProviderType{}
	for _, p := range supportedCloudProviders {
		stringToCloudProviderType[string(p)] = p
	}

	t, ok := stringToCloudProviderType[s]
	if !ok {
		err = fmt.Errorf("Unsupported cloud provider type %v", s)
	}
	return
}
