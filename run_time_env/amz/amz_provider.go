package amz

import (
	"errors"
	"fmt"
	"github.com/VolumezTech/volumez-cloud-provider/cloudprovider"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

type AdditionalInfo struct {
	InstanceType            string
	GroupName               string
	ImageID                 string
	Macs                    []*MacInfo
	AccountID               string
	BillingProducts         []string
	MarketplaceProductCodes []string
}

func (info *AdditionalInfo) ToArr() (arr []cloudprovider.AdditionalParam) {

	arr = []cloudprovider.AdditionalParam{
		{"InstanceType", info.InstanceType},
		{"GroupName", info.GroupName},
		{"ImageID", info.ImageID},
		{"MACs", fmt.Sprintf("%v", info.Macs)},
		{"AccountID", info.AccountID},
		{"BillingProducts", fmt.Sprintf("%v", info.BillingProducts)},
		{"MarketplaceProductCodes", fmt.Sprintf("%v", info.MarketplaceProductCodes)},
	}
	return
}

type MacInfo struct {
	Address          string
	VpcID            string
	SubnetID         string
	SecurityGroupIds string
}

func (info *MacInfo) String() string {
	return fmt.Sprintf(`{MAC: %v  VPC_ID: %v  SubnetID: %v  SecurityGroupIds:   %v}`, info.Address, info.VpcID, info.SubnetID, info.SecurityGroupIds)
}

type AmzServiceProvider struct {
	client *amz_client
}

var _ cloudprovider.ICloudProviderVirtualMachine = (*AmzServiceProvider)(nil)

func NewAmzServiceProvider() cloudprovider.ICloudProviderVirtualMachine {
	glog.Infoln("creating AMZ provider name")
	p := &AmzServiceProvider{}
	return p
}

func (provider *AmzServiceProvider) GetName() cloudprovider.CloudProviderType {
	glog.Infoln("got AMZ provider name")
	return cloudprovider.CloudProvider_Aws
}

func (provider *AmzServiceProvider) Init() (err error) {

	glog.Infoln("init AMZProvider")
	provider.client, err = NewClient()
	return
}

func (provider *AmzServiceProvider) GetMachineInfo() (info *cloudprovider.MachineInfo, err error) {

	if !provider.isValid() {
		return nil, errors.New("Not supported")
	}

	instanceDoc := provider.client.doc
	if err != nil {
		return
	}

	macs, _ := provider.GetMacsInfo()
	groupName, _ := provider.client.GetMetadata("placement/group-name")

	additionalInfo := &AdditionalInfo{
		InstanceType:            instanceDoc.InstanceType,
		GroupName:               groupName,
		ImageID:                 instanceDoc.ImageID,
		Macs:                    macs,
		AccountID:               instanceDoc.AccountID,
		BillingProducts:         instanceDoc.BillingProducts,
		MarketplaceProductCodes: instanceDoc.MarketplaceProductCodes,
	}

	dnsName, _ := provider.client.GetMetadata("public-hostname")

	info = &cloudprovider.MachineInfo{
		InstanceID:   instanceDoc.InstanceID,
		Zone:         instanceDoc.AvailabilityZone,
		Region:       instanceDoc.Region,
		Architecture: instanceDoc.Architecture,
		IPAddresses:  []string{instanceDoc.PrivateIP},
		PublicDNS:    dnsName,
		Cluster:      provider.getCluster(),
		Additional:   additionalInfo.ToArr(),
	}

	return
}

func (provider *AmzServiceProvider) GetVirtualMachineID() (instanceId string, err error) {

	return cloudprovider.GetVirtualMachineID(provider)
}

// http://169.254.169.254/latest/meta-data/network/interfaces/macs/12:85:18:5e:fc:eb/vpc-id
// http://169.254.169.254/latest/meta-data/network/interfaces/macs/12:85:18:5e:fc:eb/subnet-id
// curl http://169.254.169.254/latest/meta-data/network/interfaces/macs/12:85:18:5e:fc:eb/security-group-ids
func (provider *AmzServiceProvider) GetMacsInfo() (info []*MacInfo, err error) {
	// info = make([]MacInfo, 0, 1)
	macsStr, err := provider.client.GetMetadata("network/interfaces/macs")
	if err == nil {
		macs := strings.Split(macsStr, "/")
		for _, address := range macs {
			if address == "" {
				continue
			}
			vpcid, _ := provider.client.GetMetadata(fmt.Sprintf(`network/interfaces/macs/%v/vpc-id`, address))
			subnetID, _ := provider.client.GetMetadata(fmt.Sprintf(`network/interfaces/macs/%v/subnet-id`, address))
			securityGroupIDs, _ := provider.client.GetMetadata(fmt.Sprintf(`network/interfaces/macs/%v/security-group-ids`, address))
			info = append(info, &MacInfo{Address: address, VpcID: vpcid, SubnetID: subnetID, SecurityGroupIds: securityGroupIDs})
		}
	}
	return
}

func (provider *AmzServiceProvider) isValid() bool {
	return provider.client != nil
}

type Kubeconfig struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Users      []struct {
		Name string
		User struct {
			Exec struct {
				Version string   `yaml:"apiVersion"`
				Command string   `yaml:"command"`
				Args    []string `yaml:"args"`
			} `yaml:"exec"`
		} `yaml:"user"`
	} `yaml:"users"`
}

func (cfg *Kubeconfig) GetCluster() string {
	for _, user := range cfg.Users {
		if user.Name == "kubelet" {
			for i, arg := range user.User.Exec.Args {
				if arg == "-i" {
					if i+1 >= len(user.User.Exec.Args) {
						break
					}
					return user.User.Exec.Args[i+1]
				}
			}
		}
	}
	return ""
}

func (provider *AmzServiceProvider) getCluster() (cluster string) {
	absPath, _ := filepath.Abs("/var/lib/kubelet/kubeconfig")
	content, err := os.ReadFile(absPath)
	if err != nil {
		return
	}

	var c Kubeconfig
	yaml.Unmarshal(content, &c)
	cluster = c.GetCluster()
	return
}
