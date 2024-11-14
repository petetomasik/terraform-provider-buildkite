---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "buildkite_organization_rule Data Source - terraform-provider-buildkite"
subcategory: ""
description: |-
  Use this data source to retrieve an organization rule by its ID.
  More information on organization rules can be found in the documentation https://buildkite.com/docs/pipelines/rules.
---

# buildkite_organization_rule (Data Source)

~> Rules is a feature that is currently in development and enabled on an opt-in basis for early access. To have this enabled for your organization for utilizing this data source, please reach out to Buildkite's [Support Team](https://buildkite.com/support).

Use this data source to retrieve an organization rule by its ID.

More information on organization rules can be found in the [documentation](https://buildkite.com/docs/pipelines/rules).

## Example Usage

```terraform
# Read an organization rule by its id
data "buildkite_organization_rule" "data_artifacts_read_dev_test" {
    id = buildkite_organization_rule.artifacts_read_dev_test.id
}

# Read an organization rule by its uuid
data "buildkite_organization_rule" "data_artifacts_read_test_dev" {
    uuid = buildkite_organization_rule.artifacts_read_test_dev.uuid
}
```

## Schema

### Optional

- `id` (String) The GraphQL ID of the organization rule.
- `uuid` (String) The UUID of the organization rule.

### Read-Only

- `action` (String) The action defined between source and target resources.
- `description` (String) The description of the organization rule.
- `effect` (String) Whether this organization rule allows or denys the action to take place between source and target resources.
- `source_type` (String) The source resource type that this organization rule allows or denies to invoke its defined action.
- `source_uuid` (String) The UUID of the resource that this organization rule allows or denies invocating its defined action.
- `target_type` (String) The target resource type that this organization rule allows or denies the source to respective action.
- `target_uuid` (String) The UUID of the target resourcee that this organization rule allows or denies invocation its respective action.
- `type` (String) The type of organization rule.
- `value` (String) The JSON document that this organization rule implements.