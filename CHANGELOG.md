## 1.0.0 (September 11, 2023)

BREAKING CHANGES:
- resource/service: Does not support `storageGB` anymore

FEATURES:
- resource/service: Create a service. `name`,`milliCPU`,`memoryGB`,`regionCode`,`replicaCount` and `vpcID` can be specified.
- resource/service: Update a service. `name`, `milliCPU`,`memoryGB`,`vpcID` and `enableHAReplica` are modifiable.
- resource/service: Delete a service.
- data-source/service: Import a service already created on the console.
- data-source/vpc: Import VPC already created on the console.
- data-source/products: Import list of products (allowed `milliCPU` and `memoryGB` values).