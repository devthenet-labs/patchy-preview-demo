// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT
'use strict';

const REPO = 'patchy-preview-demo';
const OWNER = 'devthenet-labs';
// GitHub's immutable repository ID, read back from the API at creation.
const REPO_ID = 1391236194;
const OWNER_ID = 332591015;
const SHA = /^[0-9a-f]{40}$/;

function check(ok, message) {
  if (!ok) throw new Error(message);
}

function validate(repo, run, workflow, pr, artifacts, kind) {
  check(REPO_ID !== 0 && repo.id === REPO_ID && repo.owner.id === OWNER_ID &&
    repo.full_name === `${OWNER}/${REPO}` && !repo.fork && repo.default_branch === 'main', 'repository identity');
  check(run.repository.id === REPO_ID && run.head_repository.id === REPO_ID &&
    !run.head_repository.fork, 'forks must never publish');
  check(run.status === 'completed' && run.conclusion === 'success' && SHA.test(run.head_sha), 'successful exact head required');
  check(Number.isSafeInteger(run.run_attempt) && run.run_attempt > 0, 'run attempt');
  check(['runtime', 'agent'].includes(kind), 'image kind');
  const path = kind === 'runtime' ? '.github/workflows/ci.yml' : '.github/workflows/agent-image.yml';
  check(workflow.path === path && workflow.state === 'active' && workflow.id === run.workflow_id && run.path === path, 'workflow identity');
  if (run.event === 'pull_request') {
    check(kind === 'runtime' && pr && run.pull_requests.length === 1 &&
      run.pull_requests[0].number === pr.number, 'PR association');
    check(pr.state === 'open' && pr.base.ref === 'main' && pr.base.repo.id === REPO_ID &&
      pr.head.repo && pr.head.repo.id === REPO_ID && !pr.head.repo.fork &&
      pr.head.sha === run.head_sha, 'open same-repository PR at current head required');
  } else {
    check(['push', 'workflow_dispatch'].includes(run.event) && run.head_branch === 'main', 'main only');
  }
  const name = `${kind}-${run.head_sha}-${run.run_attempt}`;
  const matches = artifacts.filter(a => a.name === name && !a.expired);
  check(matches.length === 1, 'exactly one matching artifact required');
  const artifact = matches[0];
  check(Number.isSafeInteger(artifact.id) && artifact.id > 0 && artifact.size_in_bytes > 0 &&
    artifact.size_in_bytes <= (kind === 'runtime' ? 128 : 2048) * 1024 * 1024, 'artifact bounds');
  return {artifact_id: String(artifact.id), sha: run.head_sha, kind};
}

async function guard({github, runID, kind}) {
  check(/^[1-9][0-9]{0,18}$/.test(String(runID)), 'run ID');
  const params = {owner: OWNER, repo: REPO};
  const {data: repo} = await github.rest.repos.get(params);
  const {data: run} = await github.rest.actions.getWorkflowRun({...params, run_id: runID});
  const {data: workflow} = await github.rest.actions.getWorkflow({...params, workflow_id: run.workflow_id});
  let pr;
  if (run.event === 'pull_request' && run.pull_requests.length === 1) {
    ({data: pr} = await github.rest.pulls.get({...params, pull_number: run.pull_requests[0].number}));
  }
  // Per-run artifacts only; never search globally or trust metadata from the artifact.
  const artifacts = await github.paginate(github.rest.actions.listWorkflowRunArtifacts,
    {...params, run_id: runID, per_page: 100});
  return validate(repo, run, workflow, pr, artifacts, kind);
}

module.exports = {guard, validate, REPO_ID, OWNER_ID};
