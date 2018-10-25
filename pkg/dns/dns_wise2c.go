/*
Copyright 2016 The Wise2c information Technology Inc..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package dns

import (

	"strings"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/dns/pkg/dns/treecache"
	"k8s.io/dns/pkg/dns/util"

	"github.com/golang/glog"
	"github.com/miekg/dns"

)

const (
	// A subdomain added to the user specified domain for all services.
	wise2cServiceLabel = "io.wise2c.service"

	// A subdomain added to the user specified domain for all pods.
	wise2cStackLabel = "io.wise2c.stack"


)

func getWise2cLabel( svc *v1.Service) ([]string, bool) {
	wise2cSVC,ok := svc.Labels[wise2cServiceLabel]
	if !ok {
		return nil,ok
	}

	wise2cStack,ok := svc.Labels[wise2cStackLabel]
	if !ok {
		return nil,ok

	}
	return []string{wise2cStack,wise2cSVC}, ok
}

//wise2c fqdn pattern: domain.svc.ns.stack.servicename
func (kd *KubeDNS) wise2cfqdn(service *v1.Service, subpaths ...string) string {
	domainLabels := append(append(kd.domainPath, serviceSubdomain, service.Namespace), subpaths...)
	return dns.Fqdn(strings.Join(util.ReverseArray(domainLabels), "."))
}

func (kd *KubeDNS) wise2cRecordsForHeadlessService(e *v1.Endpoints, svc *v1.Service) error {
	//get wise2c labels:  wise2cLabels[0] - stack name; [1] - service name
	wise2cLabels,ok := getWise2cLabel(svc)
	if !ok {
         return nil
	}
	subCachePath := append(kd.domainPath, serviceSubdomain, svc.Namespace,wise2cLabels[0])

	subCache := treecache.NewTreeCache()
	//glog.V(4).Infof("Endpoints Annotations: %v", e.Annotations)
	//generatedRecords := map[string]*skymsg.Service{}
	for idx := range e.Subsets {
		for subIdx := range e.Subsets[idx].Addresses {
			address := &e.Subsets[idx].Addresses[subIdx]
			endpointIP := address.IP
			recordValue, endpointName := util.GetSkyMsg(endpointIP, 0)
			if hostLabel, exists := getHostname(address); exists {
				endpointName = hostLabel
			}
			subCache.SetEntry(endpointName, recordValue, kd.wise2cfqdn(svc, append(wise2cLabels, endpointName)...))
			//for wrise2c headless service, we don't genterate A records for named port
			/*
			for portIdx := range e.Subsets[idx].Ports {
				endpointPort := &e.Subsets[idx].Ports[portIdx]
				if endpointPort.Name != "" && endpointPort.Protocol != "" {
					srvValue := kd.generateSRVRecordValue(svc, int(endpointPort.Port), endpointName)
					glog.V(2).Infof("Added SRV record %+v", srvValue)

					l := []string{"_" + strings.ToLower(string(endpointPort.Protocol)), "_" + endpointPort.Name}
					subCache.SetEntry(endpointName, srvValue, kd.fqdn(svc, append(l, endpointName)...), l...)
				}
			} */

		}
	}
	//kd.cacheLock.Lock()
	//defer kd.cacheLock.Unlock()
	kd.cache.SetSubCache(wise2cLabels[1], subCache, subCachePath...)
	return nil
}

func (kd *KubeDNS) newWise2cPortalService(service *v1.Service) {
	//get wise2c labels:  wise2cLabels[0] - stack name; [1] - service name
	wise2cLabels,ok := getWise2cLabel(service)
	if !ok {
		return
	}

	subCache := treecache.NewTreeCache()
	recordValue, recordLabel := util.GetSkyMsg(service.Spec.ClusterIP, 0)
	subCache.SetEntry(recordLabel, recordValue, kd.fqdn(service, recordLabel))

	// Generate SRV Records
	for i := range service.Spec.Ports {
		port := &service.Spec.Ports[i]
		if port.Name != "" && port.Protocol != "" {
			srvValue := kd.generateSRVRecordValue(service, int(port.Port))

			l := []string{"_" + strings.ToLower(string(port.Protocol)), "_" + port.Name}
			glog.V(2).Infof("Added SRV record %+v", srvValue)

			subCache.SetEntry(recordLabel, srvValue, kd.fqdn(service, append(l, recordLabel)...), l...)
		}
	}
	subCachePath := append(kd.domainPath, serviceSubdomain, service.Namespace,wise2cLabels[0])
	//kd.cacheLock.Lock()
	//defer kd.cacheLock.Unlock()
	kd.cache.SetSubCache(wise2cLabels[1], subCache, subCachePath...)
}

func (kd *KubeDNS) newWise2cExternalNameService(service *v1.Service) {
	//get wise2c labels:  wise2cLabels[0] - stack name; [1] - service name
	wise2cLabels,ok := getWise2cLabel(service)
	if !ok {
		return
	}

	// Create a CNAME record for the service's ExternalName.
	recordValue, _ := util.GetSkyMsg(service.Spec.ExternalName, 0)
	cachePath := append(kd.domainPath, serviceSubdomain, service.Namespace, wise2cLabels[0])
	fqdn := kd.wise2cfqdn(service,wise2cLabels[0])
	glog.V(2).Infof("newExternalNameService: storing key %s with value %v as %s under %v",
		service.Name, recordValue, fqdn, cachePath)
	//kd.cacheLock.Lock()
	//defer kd.cacheLock.Unlock()
	// Store the service name directly as the leaf key
	kd.cache.SetEntry(wise2cLabels[1], recordValue, fqdn, cachePath...)
}

func (kd *KubeDNS) removeWise2cService(svc *v1.Service) (bool) {
	//get wise2c labels:  wise2cLabels[0] - stack name; [1] - service name
	wise2cLabels,ok := getWise2cLabel(svc)
	if !ok {
		return true
	}
	subCachePath := append(kd.domainPath, serviceSubdomain, svc.Namespace, wise2cLabels[1], wise2cLabels[0])
	success := kd.cache.DeletePath(subCachePath...)
	glog.V(2).Infof("removeService %v at path %v. Success: %v",
		svc.Name, subCachePath, success)
	return success;
}
