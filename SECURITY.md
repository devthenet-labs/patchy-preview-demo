# Build and publication boundary

This is a benign, stateless preview app, not the deliberately vulnerable
`patchy-target`. It has no accounts, secrets, database, filesystem writes or
outbound requests. It serves a welcome page, `/healthz` and `/version` on 8080.

## Two different images, two different roles

| Image | Source | Destination | Trusted reusable publisher |
| --- | --- | --- | --- |
| Runtime | PR head, or main | `patchy/previews/patchy-preview-demo:sha-<40hex>` | `publish-runtime.yml` |
| Agent toolchain | main only | `patchy/app-envs/patchy-preview-demo:toolchain-v1` | `publish-agent.yml` |

Build jobs have only `contents: read`, no OIDC permission, no cloud/model
credentials and no persisted checkout credential. They pass no token, secret,
host Docker socket or SSH mount into the Docker build. Fork PRs may test/build
but must never publish. Artifact upload uses GitHub Actions' artifact service;
the artifact is still untrusted data, not authority.

Publication runs on `workflow_run`, from the default branch's trusted workflow
snapshot. It never checks out, builds or runs PR source, and never unpacks image
layers. Before obtaining AWS credentials it re-reads GitHub's repository, run,
workflow, PR and artifact metadata and validates a bounded OCI archive. The PR
must still be open, in this exact repository, targeting main, at the successful
run's exact head. Missing/ambiguous metadata fails closed. Artifact names do not
grant authority: the exact artifact ID is selected from that specific run.

The validator accepts one linux/amd64 image, checks all blob hashes and descriptor
sizes, rejects links/traversal/duplicates/foreign URLs/unreferenced blobs, and
copies only opaque files to a fresh directory outside the trusted checkout.
Only a trusted, distribution-packaged `skopeo copy --preserve-digests` sees those
blobs after credential acquisition. There is no Docker execution in publishers.
No cache is shared from build jobs into publishers.

AWS must bind each role to this repository and owner **numeric IDs**, the exact
immutable `main` subject, audience `sts.amazonaws.com`, and its **different**
`job_workflow_ref`. The runtime workflow must not be able to assume the agent
role. This intentionally refines the original preview design's `:pull_request`
subject: the privileged job is a trusted main-branch publisher, not a PR job.
Neither role gets Kubernetes, IAM, Secrets Manager or any other ECR repository.

Tags are immutable. Retrying an identical digest is a no-op; a different digest
at an existing tag is a hard failure. Bump `toolchain-v1` in both the declaration
and trusted copy script when deliberately updating the agent recipe. Changes to
`.github/` and `.patchy/` require human review; patchy's intent changeset policy
already refuses both paths. These protections assume main's maintainers remain
trusted; no repository workflow can protect itself from a malicious main admin.

## Bootstrap and rollback

Publishing is disabled unless the repository variable `PREVIEW_PUBLISH_ENABLED`
equals `true`. Leave it unset until the owner approves the separate Terraform
apply and activation. Do not change Actions approval settings or release
credentials. Initial CI can validate images without any AWS resources.

After approval, create the two immutable ECR repositories and narrowly scoped
roles, enable the variable, then rerun main's successful `test` and `agent image`
workflows. Verify image digests, the agent-image sandbox preflight and an actual
same-repo PR publication before enabling Project `preview-demo`. A publish check
being skipped during bootstrap is **not** proof of AWS access or image upload.

Rollback: disable the variable, revoke the two new role trust policies, then
review a Terraform removal plan. Retain already-published images unless the
owner explicitly approves deletion; repositories have no force-delete setting.
This repository does not apply infrastructure. No preview namespace, ALB, DNS,
Project or controller is created by these workflows.
