/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"flag"
	"os"
	"time"
	"syscall"
	"sync"
        "os/exec"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
        "fmt"
	"strconv"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

const (
	provisionerName = "mochacoffee/rozofs"
)

func getNextVid() (string, error) {
	cmd := "rozo volume list | grep 'VOLUME ' | awk '{print $3}' | sed -e 's/://' | tail -n1 | tr -d '\n' "
        out,err := exec.Command("bash","-c",cmd).Output()
	value := string(out)
        if err != nil {
                fmt.Sprintf("Failed to get last Rozofs Vid")
		return "",err
        }
	if value == "" {
		value = "0"
	}
	i,er := strconv.Atoi(value)
	if er != nil {
		fmt.Sprintf("Error when converting volume id")
		return "",er
	}
	i += 1
	output := strconv.Itoa(i)
	return string(output),nil

}
func getNextExport() (string, error) {
	cmd := "rozo export get | grep 'EXPORT ' | awk '{print $3}' | sed -e 's/://' | tail -n1 | tr -d '\n' "
        out,err := exec.Command("bash","-c",cmd).Output()
	value := string(out)
        if err != nil {
                fmt.Sprintf("Failed to get last Rozofs Export")
		return "",err
        }
	if value == "" {
		value = "0"
	}
	i,er := strconv.Atoi(value)
	if er != nil {
		fmt.Sprintf("Error when converting export id")
		return "",er
	}
	i += 1
	output := strconv.Itoa(i)
	return string(output),nil
}

func createNewExport(vid string) error{
	cmd := exec.Command("rozo","export","create",vid)
	err := cmd.Run()
	return err
}

func createNewMount(exportid string) error{
	cmd := exec.Command("rozo","mount","create","-i",exportid)
	err := cmd.Run()
	return err
}

func createNewVolume(vid string,p *rozoProvisioner) error{
        request := "rozo volume expand %s"
	cmd := exec.Command("bash","-c",fmt.Sprintf(request,p.clusternodes))
	err := cmd.Run()
	return err
}

type rozoProvisioner struct {
	driver string
	identity string
	exportnode string
	clusternodes string
	mux sync.Mutex
}

func NewRozoProvisioner() controller.Provisioner {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		klog.Fatal("env variable NODE_NAME must be set so that this provisioner can identify itself")
	}
	return &rozoProvisioner{
		driver: provisionerName,
		identity: nodeName,
		exportnode: os.Getenv("ROZO_EXPORT_HOSTNAME"),
		clusternodes: os.Getenv("CLUSTER_NODES"),
	}
}

var _ controller.Provisioner = &rozoProvisioner{}

// Provision creates a storage asset and returns a PV object representing it.
func (p *rozoProvisioner) Provision(options controller.ProvisionOptions) (*v1.PersistentVolume, error) {


	//Avoid volume claim overlaping by using mutex
	p.mux.Lock()
	exportid,err := getNextExport()
	vid, err := getNextVid()

	if err != nil {
	        p.mux.Unlock()
		return nil,err
	}

	err = createNewVolume(vid,p)
	if err != nil {
	        p.mux.Unlock()
		fmt.Println("Error when creating new volume")
		return nil,err
	}

	err = createNewExport(vid)
	i := 0
	for err != nil {
		if i > 10 {
	                p.mux.Unlock()
			fmt.Println("Error when creating new export")
			return nil,err
		}
		i += 1
		time.Sleep(10 * time.Second)
		err = createNewExport(vid)
	}
	createNewMount(exportid)

	p.mux.Unlock()

	defer time.Sleep(5 * time.Second)

        pv := &v1.PersistentVolume{
                ObjectMeta: metav1.ObjectMeta{
                        Name: options.PVName,
                },
                Spec: v1.PersistentVolumeSpec{
                        PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
                        AccessModes:                   options.PVC.Spec.AccessModes,
                        Capacity: v1.ResourceList{
                                v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
                        },
                        PersistentVolumeSource: v1.PersistentVolumeSource{
                                FlexVolume: &v1.FlexPersistentVolumeSource{
                                        Driver: provisionerName,
                                        Options: map[string]string{
                                                "node": p.exportnode,
                                                "export_id": exportid,
				                "vid": vid,
                                        },
                                },
                        },
                },
        }

	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *rozoProvisioner) Delete(volume *v1.PersistentVolume) error {
	p.mux.Lock()
	cmd := exec.Command("rozo","mount","remove","-i",volume.Spec.FlexVolume.Options["export_id"])
	err := cmd.Run()
	if err != nil {
		p.mux.Unlock()
		return err
	}
        cmd = exec.Command("rozo","export","remove","-f",volume.Spec.FlexVolume.Options["export_id"])
	err = cmd.Run()
	if err != nil {
		p.mux.Unlock()
		return err
	}
        cmd = exec.Command("rozo","volume","remove","-v",volume.Spec.FlexVolume.Options["vid"])
	err = cmd.Run()
	if err != nil {
		p.mux.Unlock()
		return err
	}
	p.mux.Unlock()
	defer time.Sleep(5 * time.Second)

	return nil
}

func main() {
	syscall.Umask(0)

	flag.Parse()
	flag.Set("logtostderr", "true")

	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("Error getting server version: %v", err)
	}

	// Create the provisioner: it implements the Provisioner interface expected by
	// the controller
	rozoProvisioner := NewRozoProvisioner()

	pc := controller.NewProvisionController(clientset, provisionerName, rozoProvisioner, serverVersion.GitVersion)
	pc.Run(wait.NeverStop)
}
