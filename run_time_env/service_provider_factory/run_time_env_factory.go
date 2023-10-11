package service_provider_factory

import (
	"github.com/VolumezTech/volumez-cloud-provider/cloudprovider"
	"github.com/VolumezTech/volumez-cloud-provider/run_time_env/amz"
	"github.com/VolumezTech/volumez-cloud-provider/run_time_env/azure"
	"github.com/VolumezTech/volumez-cloud-provider/run_time_env/on_prem"
	"github.com/golang/glog"
)

var s_Provider cloudprovider.ICloudProviderVirtualMachine

func GetSupportedServiceProviders() []cloudprovider.ICloudProviderVirtualMachine {
	constructors := []cloudprovider.ServiceProviderConstructor{
		on_prem.NewOnPremEnvServiceProvider,
		newConfigProvider,
		amz.NewAmzServiceProvider,
		azure.NewAzureServiceProvider,
	}
	list := make([]cloudprovider.ICloudProviderVirtualMachine, 0, len(constructors))
	for i := range constructors {
		provider := constructors[i]()
		list = append(list, provider)
	}
	return list
}

// Returns the first provider that was successfully initialized
func GetServiceProvider() cloudprovider.ICloudProviderVirtualMachine {

	glog.Infoln("inside GetServiceProvider")
	if s_Provider == nil {
		providers := GetSupportedServiceProviders()
		for i := range providers {
			provider := providers[i]
			if provider != nil {
				err := provider.Init()
				if err == nil {
					s_Provider = provider
					glog.Infof("caught provider - %v", s_Provider)
					break
				}
			}
		}
	}
	return s_Provider
}

func newConfigProvider() cloudprovider.ICloudProviderVirtualMachine {
	return on_prem.NewOnPremConfigServiceProvider("/opt/vlzconnector/vlzconnector.json")
}
