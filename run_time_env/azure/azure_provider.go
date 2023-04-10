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

func (info *AdditionalInfo) ToArr() (arr []cloudprovider.AdditionalParam) {

	arr = []cloudprovider.AdditionalParam{
		{"InstanceType", info.InstanceType},
		{"GroupName", info.GroupName},
		{"ImageID", info.ImageID},
		{"OS Type", info.OsType},
		{"AccountID", info.AccountID},
		{"VmScaleSetName", info.VmScaleSetName},
	}
	return
}

type AzureServiceProvider struct {
	info *cloudprovider.MachineInfo
}

var _ cloudprovider.ICloudProviderVirtualMachine = (*AzureServiceProvider)(nil)

func NewAzureServiceProvider() cloudprovider.ICloudProviderVirtualMachine {
	p := &AzureServiceProvider{}
	return p
}

func (provider *AzureServiceProvider) GetName() cloudprovider.CloudProviderType {
	return cloudprovider.CloudProvider_Azure
}

func (provider *AzureServiceProvider) Init() (err error) {

	var jsonData []byte
	azureMetaData, err := getMetadata(formatURL("attested/document"))
	_ = azureMetaData
	if err != nil {
		// Try local config
		var readErr error
		jsonData, readErr = os.ReadFile("azure_instance.json")
		if readErr != nil {
			err = errors.New(fmt.Sprintf(`failed to retrieve azure metadata (%v), failed to read local config (%v)`, err, readErr))
			return
		}
	} else {
		var s string
		s, err = getMetadata(formatURL("instance"))
		if err != nil {
			return
		}
		jsonData = []byte(s)
	}

	var data AzureMetaData
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return
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
	instanceID := data.Compute.VMID
	if instanceID == "" {
		instanceID = data.Compute.VMID
	}
	info := &cloudprovider.MachineInfo{
		InstanceID:   instanceID,
		Zone:         fmt.Sprintf(`%v-%v`, data.Compute.Location, data.Compute.Zone), //zone seems to be just number in azure creating concatenation of region+zone to get virtual zone
		Region:       data.Compute.Location,
		Architecture: "",
		IPAddresses:  data.Network.GetPrivateIPs(),
		PublicDNS:    data.Network.GetPublicDNS(),
		Additional:   additionalInfo.ToArr(),
	}
	provider.info = info
	return
}

func (provider *AzureServiceProvider) GetMachineInfo() (info *cloudprovider.MachineInfo, err error) {

	if provider.info == nil {
		err = errors.New("Failed to retrieve")
		return
	}
	info = provider.info
	return
}

func (provider *AzureServiceProvider) GetVirtualMachineID() (instanceId string, err error) {

	return cloudprovider.GetVirtualMachineID(provider)
}

func formatURL(relativePath string) string {
	// http://169.254.169.254/metadata/instance/compute?api-version=2021-02-01
	// http://169.254.169.254/metadata/attested/document?api-version=2020-09-01
	return fmt.Sprintf(`http://169.254.169.254/metadata/%v?api-version=2021-02-01`, relativePath)
}

func getMetadata(url string) (result string, err error) {

	request, err := http.NewRequest(http.MethodGet, url, nil)
	request.Header.Add("Metadata", "True")
	// request.Header.Add("content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		err = errors.New(fmt.Sprintf(`%v`, response))
		return
	}

	body, err := io.ReadAll(response.Body)
	result = string(body)
	return

	// ssh := shell.MakeLocalBashExecuter("HOST_NAME")
	// executor := shell.BindExecOptions(ssh, shell.CmdExecOptions{Timeout: 1 * time.Second, Priority: false})
	// safeExecutor := shell.MakeExtendedShell(executor)
	// result, err = safeExecutor.Runf("curl -H Metadata:true --noproxy %v %v", "*", url).Raw()
	// return
}

type AzureMetaData struct {
	Compute AzureMetaDataCompute `json:"compute"`
	Network AzureMetaDataNetwork `json:"network"`
}

// The format of json
// https://learn.microsoft.com/en-us/azure/virtual-machines/linux/instance-metadata-service?tabs=linux
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
	if len(provider.Interfaces) > 0 {
		i := provider.Interfaces[0]
		return i.IPv4.GetPublicDNS()
	}
	return ""
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
	if len(provider.Addresses) > 0 {
		return provider.Addresses[0].PublicIpAddress
	}
	return ""
}

type AzureMetaDataAddressPair struct {
	PrivateIpAddress string `json:"privateIpAddress"`
	PublicIpAddress  string `json:"publicIpAddress"`
}
type AzureMetaDataSubnet struct {
	Address string `json:"address"`
	Prefix  string `json:"prefix"`
}
