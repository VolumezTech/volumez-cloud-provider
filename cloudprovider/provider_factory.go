package cloudprovider

import (
	"errors"
	"os"

	ec2metadata "github.com/VolumezTech/volumez-cloud-provider/util"
)

type CloudProvider struct {
	Name CloudProviderType
}

var errUnimplemented = errors.New("provider is not implemented")

var _ ICloudProviderVirtualMachine = (*CloudProvider)(nil)

func GetCloudProvider(name string) (p *CloudProvider, err error) {
	t, err := ConvertToCloudProviderType(name)
	if err == nil {
		p = &CloudProvider{Name: t}
	}
	return
}

func GetCurrentCloudProvider(useHostNameAsVMID bool) (*CloudProvider, error) {

	//  Detect what provider we are on - implement later
	name := CloudProvider_Aws
	if useHostNameAsVMID {
		name = CloudProvider_OnPremConfig
	}

	return GetCloudProvider(string(name))
}

func (cp *CloudProvider) GetName() CloudProviderType {
	return cp.Name
}

func (cp *CloudProvider) Init() (err error) {
	return
}

func (cp *CloudProvider) GetVirtualMachineID() (id string, err error) {
	info, err := cp.GetMachineInfo()
	if err == nil {
		id = info.InstanceID
	}
	return
}

var cloudProviderGetters = map[CloudProviderType]func() (info *MachineInfo, err error){
	CloudProvider_Aws:          getAWSVM,
	CloudProvider_Azure:        getAzureVM,
	CloudProvider_OnPremConfig: getLocalVM,
	CloudProvider_OnPremEnv:    getLocalVM,
}

func (cp *CloudProvider) GetMachineInfo() (info *MachineInfo, err error) {

	cb, ok := cloudProviderGetters[cp.Name]
	if ok {
		info, err = cb()
	} else {
		err = errUnimplemented
	}
	return
}

func getAWSVM() (info *MachineInfo, err error) {
	ec2MetaData := ec2metadata.New(21600)
	var ec2ID string
	ec2ID, err = ec2MetaData.GetEC2InstanceIDWithRetry(3)
	if err != nil {
		return nil, err
	}
	info = &MachineInfo{InstanceID: ec2ID}
	return
}

func getAzureVM() (info *MachineInfo, err error) {
	err = errUnimplemented
	return
}

func getLocalVM() (info *MachineInfo, err error) {

	hn, err := os.Hostname()

	if err != nil {
		return nil, err
	}
	info = &MachineInfo{InstanceID: hn}

	return
}
