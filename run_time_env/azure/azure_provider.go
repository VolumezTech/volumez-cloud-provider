package azure

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/VolumezTech/volumez-cloud-provider/cloudprovider"
)

type AdditionalInfo struct {
	InstanceType   string // Compute.InstanceType  (vmSize)
	GroupName      string // Compute.ResourceGroupName (resourceGroupName)
	ImageID        string // Compute.Sku
	OsType         string // Compute.OsType (osType)
	VmID           string // Compute.VMID
	AccountID      string // Compute.SubscriptionId (subscriptionId)
	VmScaleSetName string
}

func (info *AdditionalInfo) ToArr() []cloudprovider.AdditionalParam {
	return []cloudprovider.AdditionalParam{
		{"VmID", info.VmID},
		{"InstanceType", info.InstanceType},
		{"GroupName", info.GroupName},
		{"ImageID", info.ImageID},
		{"OS Type", info.OsType},
		{"AccountID", info.AccountID},
		{"VmScaleSetName", info.VmScaleSetName},
	}
}

type AzureServiceProvider struct {
	info *cloudprovider.MachineInfo
}

var _ cloudprovider.ICloudProviderVirtualMachine = (*AzureServiceProvider)(nil)

func NewAzureServiceProvider() cloudprovider.ICloudProviderVirtualMachine {
	return &AzureServiceProvider{}
}

func (provider *AzureServiceProvider) GetName() cloudprovider.CloudProviderType {
	return cloudprovider.CloudProvider_Azure
}

func (provider *AzureServiceProvider) Init() error {
	var jsonData []byte
	if _, err := getMetadata(formatURL("attested/document")); err != nil {
		// Try local config
		var readErr error
		jsonData, readErr = os.ReadFile("azure_instance.json")
		if readErr != nil {
			return errors.New(fmt.Sprintf(`failed to retrieve azure metadata (%v), failed to read local config (%v)`, err, readErr))
		}
	}

	s, err := getMetadata(formatURL("instance"))
	if err != nil {
		return err
	}
	jsonData = []byte(s)

	var data AzureMetaData
	if err = json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	additionalInfo := &AdditionalInfo{
		InstanceType:   data.Compute.InstanceType,
		GroupName:      data.Compute.ResourceGroupName,
		ImageID:        data.Compute.Sku,
		OsType:         data.Compute.OsType,
		VmID:           data.Compute.VMID,
		AccountID:      data.Compute.SubscriptionId,
		VmScaleSetName: data.Compute.VmScaleSetName,
	}
	instanceID := data.Compute.OSProfile.ComputerName
	if instanceID == "" {
		instanceID = data.Compute.Name
	}
	instanceID = fmt.Sprintf(`%v-%v`, additionalInfo.GroupName, instanceID)

	zone := data.Compute.Location
	if data.Compute.Zone != "" {
		zone = fmt.Sprintf(`%v-%v`, data.Compute.Location, data.Compute.Zone) //zone seems to be just number in azure creating concatenation of region+zone to get virtual zone
	}
	provider.info = &cloudprovider.MachineInfo{
		InstanceID:   instanceID,
		Zone:         zone,
		Region:       data.Compute.Location,
		Architecture: "",
		IPAddresses:  data.Network.GetPrivateIPs(),
		PublicDNS:    data.Network.GetPublicDNS(),
		Additional:   additionalInfo.ToArr(),
	}
	return nil
}

func (provider *AzureServiceProvider) GetMachineInfo() (*cloudprovider.MachineInfo, error) {
	if provider.info == nil {
		return nil, errors.New("Failed to retrieve")
	}
	return provider.info, nil
}

func (provider *AzureServiceProvider) GetVirtualMachineID() (instanceId string, err error) {
	return cloudprovider.GetVirtualMachineID(provider)
}

func formatURL(relativePath string) string {
	return fmt.Sprintf(`http://169.254.169.254/metadata/%v?api-version=2021-02-01`, relativePath)
}

func getMetadata(url string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	request.Header.Add("Metadata", "True")
	// request.Header.Add("content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf(`%v`, response))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil

	// ssh := shell.MakeLocalBashExecuter("HOST_NAME")
	// executor := shell.BindExecOptions(ssh, shell.CmdExecOptions{Timeout: 1 * time.Second, Priority: false})
	// safeExecutor := shell.MakeExtendedShell(executor)
	// result, err = safeExecutor.Runf("curl -H Metadata:true --noproxy %v %v", "*", url).Raw()
	// return
}

// The format of json
// https://learn.microsoft.com/en-us/azure/virtual-machines/linux/instance-metadata-service?tabs=linux
// Can be retrieved with
//
//	curl -H "Metadata: True" http://169.254.169.254/metadata/instance?api-version=2021-02-01  | json_pp | less
type AzureMetaData struct {
	Compute AzureMetaDataCompute `json:"compute"`
	Network AzureMetaDataNetwork `json:"network"`
}

type AzureMetaDataCompute struct {
	VMID              string         `json:"vmId"`
	Zone              string         `json:"zone"`
	Location          string         `json:"location"` //REGION
	Name              string         `json:"name"`
	OSProfile         AzureOsProfile `json:"osProfile"`
	Sku               string         `json:"sku"`
	ResourceGroupName string         `json:"resourceGroupName"`
	InstanceType      string         `json:"vmSize"`
	OsType            string         `json:"osType"`
	VmScaleSetName    string         `json:"vmScaleSetName"`
	SubscriptionId    string         `json:"subscriptionId"`
}

type AzureOsProfile struct {
	ComputerName string `json:"computerName"`
}

type AzureMetaDataNetwork struct {
	Interfaces []AzureMetaDataInterface `json:"interface"`
}

func (provider *AzureMetaDataNetwork) GetPublicDNS() string {
	if len(provider.Interfaces) == 0 {
		return ""
	}
	return provider.Interfaces[0].IPv4.GetPublicDNS()
}

func (provider *AzureMetaDataNetwork) GetPrivateIPs() []string {
	ips := make([]string, 0)
	for i := range provider.Interfaces {
		for _, pair := range provider.Interfaces[i].IPv4.Addresses {
			ips = append(ips, pair.PrivateIpAddress)
		}
	}
	return ips
}

type AzureMetaDataInterface struct {
	IPv4       AzureMetaDataAddressFamily `json:"ipv4"`
	IPv6       AzureMetaDataAddressFamily `json:"ipv6"`
	MacAddress string                     `json:"macAddress"`
}

type AzureMetaDataAddressFamily struct {
	Addresses []AzureMetaDataAddressPair `json:"ipAddress"`
	Subnet    []AzureMetaDataSubnet      `json:"subnet"`
}

func (provider *AzureMetaDataAddressFamily) GetPublicDNS() string {
	if len(provider.Addresses) == 0 {
		return ""
	}
	return provider.Addresses[0].PublicIpAddress
}

type AzureMetaDataAddressPair struct {
	PrivateIpAddress string `json:"privateIpAddress"`
	PublicIpAddress  string `json:"publicIpAddress"`
}
type AzureMetaDataSubnet struct {
	Address string `json:"address"`
	Prefix  string `json:"prefix"`
}
