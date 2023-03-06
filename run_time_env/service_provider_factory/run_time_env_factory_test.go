package service_provider_factory_test

import (
	"fmt"
	"testing"

	"github.com/VolumezTech/volumez-cloud-provider/run_time_env/service_provider_factory"
)

func TestRunTimeEnv(t *testing.T) {

	providers := service_provider_factory.GetSupportedServiceProviders()
	for i := range providers {
		provider := providers[i]
		if provider != nil {
			fmt.Printf("===== Trying Provider=%v... ", provider.GetName())
			err := provider.Init()
			msg := "OK\n"
			if err != nil {
				msg = fmt.Sprintf("FAILED\nerr=%v\n-----------------------\n", err)
			}
			fmt.Print(msg)
			if err == nil {
				break
			}
		}
	}

	provider := service_provider_factory.GetServiceProvider()

	if provider == nil {
		panic("Unsupported env")
	}
	info, err := provider.GetMachineInfo()
	if err != nil {
		fmt.Print(err)
		panic("Failed to retrieve machine info")
	}
	if info == nil {
		panic("Failed to retrieve machine info")
	}
	fmt.Printf("%v", info)

}
