## 2.1.0 (May 26, 2025)

DEPRECATED:
- Deprecate unused cidr field from Peering Connection.

FEATURES:
- Add cidr_blocks support to Peering Connection.


## 2.0.0 (May 19, 2025)

This major version increase is primarily due to substantial internal refactoring and architectural improvements.

**Upgrade Guidance:**

* **Expected Straightforward Upgrade:** For most users, upgrading from version 1.x.x to 2.0.0 is expected to be straightforward with no immediate configuration changes required in your Terraform files.
* **Internal Changes:** While the external behavior and resource interfaces remain compatible, the underlying codebase has undergone significant enhancements. These changes aim to improve performance, maintainability, and prepare for future features.
* **Recommendation for Testing:** We **strongly recommend** that all users first test this new version in a non-production (development or staging) environment. This will help ensure that the provider behaves as expected with your specific configurations and infrastructure before deploying to production.


DEPRECATED:
- N/A

FEATURES:
- Now a peering connection can be requested and accepted in a single `terraform apply`.