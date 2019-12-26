# Rozofs Provisioner

This is a functional implementation of Rozofs distributed filesystem on Kubernetes.  
This project features a daemonset to bring up Rozofs daemons on all kubernetes nodes,
and a deployment to bring up a provisioner for dynamic Rozofs volume provisioning.  
This project includes code from [kvaps/docker-rozofs]  and [kubernetes-sigs/sig-storage-lib-external-provisioner] 

## About Rozofs
Wiki: http://rozofs.github.io/rozofs/master/AboutRozoFS.html  
Github: https://github.com/kvaps/docker-rozofs

## Requirements
  - Each node runs systemd
  - rpcbind has to be installed on each node
## Deployment:

  - Define the export node and the cluster nodes on which Rozofs will create volumes, by creating a kubernetes secret.  The fileystem requires a layout with least 4 nodes in the cluster in order to deploy volumes. More info: http://rozofs.github.io/rozofs/master/AboutRozoFS.html#layouts
```
$ export EXPORTNODE="<export_node>"
$ export CLUSTER="<node1> <node2> <node3> <node4> ..."
$ kubectl create secret generic rozofs-secret --from-literal=clusternodes="${CLUSTER}" --from-literal=exportnode="${EXPORTNODE}"
```
  - Deploy the daemonset and the provisioner:
```
$ kubectl apply -f https://github.com/MochaCaffe/rozofs-provisioner/raw/master/daemonset.yaml
$ kubectl apply -f https://github.com/MochaCaffe/rozofs-provisioner/raw/master/provisioner/deployment.yaml
```
A storage class called "rozofs" is automatically created with the provisioner.

## Create a volume claim
The provisioner watches for Persistent Volume Claims that request a new PersistentVolume, and automatically provision a new Rozofs volume to be binded with the claim.  
A Persistent Volume defines mount parameters for the FlexVolume driver.  
  - Create a PVC
```
$ kubectl apply -f https://github.com/MochaCaffe/rozofs-provisioner/raw/master/provisioner/claim.yaml
```

  - Deploy a sample pod

```
$ kubectl apply -f https://github.com/MochaCaffe/rozofs-provisioner/raw/master/provisioner/test-pod.yaml
```
## Credits
[dpertin/docker-rozofs]  
[kvaps/docker-rozofs]  
[rozofs/rozofs]

   [rozofs/rozofs]: <https://github.com/rozofs/rozofs>
   [dpertin/docker-rozofs]: <https://github.com/dpertin/docker-rozofs>
   [kvaps/docker-rozofs]: <https://github.com/kvaps/docker-rozofs>
   [kubernetes-sigs/sig-storage-lib-external-provisioner]: <https://github.com/kubernetes-sigs/sig-storage-lib-external-provisioner>
