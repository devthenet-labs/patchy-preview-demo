# patchy-preview-demo

A tiny, benign Go app for patchy's isolated previews. No accounts, storage,
secrets or outbound requests. **The deliberately vulnerable `patchy-target`
must never be deployed as a preview.**

Routes on port 8080:

- `GET /`: welcome page with the build revision.
- `GET /healthz`: `{"status":"ok"}`.
- `GET /version`: JSON with `sha` and `built`.

## Develop

Use Go 1.26.6; there are no external Go dependencies.

```sh
go vet ./...
go test -race ./...
go run .
node --test .github/publish/guard.test.cjs
python3 -m unittest discover -s .github/publish -p 'test_*.py'
```

The root Dockerfile builds a static, non-root `scratch` runtime. For local use:

```sh
docker build -t patchy-preview-demo:local .
docker run --rm --read-only --cap-drop ALL --security-opt no-new-privileges \
  -p 127.0.0.1:8080:8080 patchy-preview-demo:local
```

`.patchy/Dockerfile` is a different image: patchy's agent base plus the pinned
Go toolchain, with no application source baked in. It can vet and test this
stdlib-only app offline in patchy's sandbox. `.patchy/agent.yaml` declares its
immutable `toolchain-v1` tag. Publication requires the separately approved ECR
and OIDC infrastructure; the tag does not exist yet during bootstrap.

## CI and previews

`test` vets/tests the exact PR head and builds an OCI runtime artifact without
cloud credentials or OIDC. `agent image` builds the toolchain only on main.
Trusted, separate publishers validate artifacts as data and copy them into
different immutable ECR repositories. Fork PRs cannot publish. Read
[SECURITY.md](SECURITY.md) for the complete trust boundary, rejection tests,
activation procedure and rollback.

Publishing starts **disabled**. Do not set `PREVIEW_PUBLISH_ENABLED=true` before
the owner's infrastructure check-in and approval. A skipped publisher is not a
successful publication. No Actions approval settings or release credentials
are changed by this bootstrap.

Future Project: `preview-demo`. The preview-controller renders all Kubernetes
resources; [deploy/example.yaml](deploy/example.yaml) only documents the runtime
contract and must not be applied. Preview ingress will be restricted to
`75.70.97.14/32`. No preview controller, Project, namespace, ALB or DNS entry is
enabled by this repository.
