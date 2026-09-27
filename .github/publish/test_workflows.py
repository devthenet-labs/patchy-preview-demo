# Copyright 2026 DevTheNet Labs.
# SPDX-License-Identifier: MIT
"""Pin the build/publish boundary as well as the data validators."""
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class WorkflowBoundaryTest(unittest.TestCase):
    def test_builds_have_no_oidc_or_publisher_credentials(self):
        for name in ["ci.yml", "agent-image.yml"]:
            text = (ROOT / "workflows" / name).read_text()
            for forbidden in ["id-token:", "secrets.", "role-to-assume:", "docker login", "--push"]:
                self.assertNotIn(forbidden, text)
            self.assertIn('test -z "${ACTIONS_ID_TOKEN_REQUEST_URL:-}"', text)
            self.assertIn("persist-credentials: false", text)

    def test_publishers_are_separate_trusted_workflows(self):
        for kind in ["runtime", "agent"]:
            text = (ROOT / "workflows" / f"publish-{kind}.yml").read_text()
            self.assertIn("ref: ${{ github.workflow_sha }}", text)
            self.assertIn("github.ref == 'refs/heads/main'", text)
            self.assertIn(f"role/devthenet-labs-preview-demo-{kind}-push", text)
            self.assertIn(f"IMAGE_KIND: {kind}", text)
            self.assertNotIn("inputs.kind", text)
            self.assertNotIn("secrets: inherit", text)
            for forbidden in ["docker run", "docker build", "head.sha", "npm install", "go test"]:
                self.assertNotIn(forbidden, text)

    def test_publishing_disabled_without_explicit_activation(self):
        text = (ROOT / "workflows" / "publish-images.yml").read_text()
        self.assertEqual(text.count("vars.PREVIEW_PUBLISH_ENABLED == 'true'"), 2)


if __name__ == "__main__":
    unittest.main()
