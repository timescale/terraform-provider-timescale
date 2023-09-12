## 1.0.0 (September 11, 2023)

DEPRECATED:
- resource/service: `storageGB`` is deprecated and ignored. With the new usage-based storage Timescale automatically allocates the disk space needed by your service and you only pay for the disk space you use.

FEATURES:
- resource/service: Create a service. `name`,`milliCPU`,`memoryGB`,`regionCode`,`replicaCount` and `vpcID` can be specified.
- resource/service: Update a service. `name`, `milliCPU`,`memoryGB`,`vpcID` and `enableHAReplica` are modifiable.
- resource/service: Delete a service.
- data-source/service: Import a service already created on the console.
- data-source/vpc: Import VPC already created on the console.
- data-source/products: Import list of products (allowed `milliCPU` and `memoryGB` values).