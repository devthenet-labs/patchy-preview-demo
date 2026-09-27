// Copyright 2026 DevTheNet Labs.
// SPDX-License-Identifier: MIT
'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const {validate, REPO_ID, OWNER_ID} = require('./guard.cjs');
const sha = 'a'.repeat(40);

function fixture(kind = 'runtime', event = 'pull_request') {
  const repo = {id: REPO_ID, owner: {id: OWNER_ID}, full_name: 'devthenet-labs/patchy-preview-demo', fork: false, default_branch: 'main'};
  const path = kind === 'runtime' ? '.github/workflows/ci.yml' : '.github/workflows/agent-image.yml';
  return [repo, {repository: repo, head_repository: repo, status: 'completed', conclusion: 'success',
    head_sha: sha, workflow_id: 5, path, event, head_branch: 'main', run_attempt: 1, pull_requests: [{number: 3}]},
  {id: 5, path, state: 'active'}, {number: 3, state: 'open', base: {ref: 'main', repo}, head: {repo, sha}},
  [{id: 7, name: `${kind}-${sha}-1`, expired: false, size_in_bytes: 123}], kind];
}
test('same-repo current PR, main runtime, and main agent accepted', () => {
  for (const args of [fixture(), fixture('runtime', 'push'), fixture('agent', 'push')]) {
    assert.equal(validate(...args).sha, sha);
  }
});
for (const [name, mutate] of Object.entries({
  fork: a => {a[1].head_repository = {...a[0], id: 888, fork: true};},
  reused_name: a => {a[0].id++;},
  wrong_owner: a => {a[0].owner.id++;},
  failed_run: a => {a[1].conclusion = 'failure';},
  in_progress: a => {a[1].status = 'in_progress';},
  wrong_workflow: a => {a[2].path = '.github/workflows/evil.yml';},
  stale_head: a => {a[3].head.sha = 'b'.repeat(40);},
  closed_pr: a => {a[3].state = 'closed';},
  fork_pr: a => {a[3].head.repo = {...a[0], fork: true, id: 888};},
  missing_pr: a => {a[1].pull_requests = [];},
  ambiguous_pr: a => {a[1].pull_requests.push({number: 9});},
  wrong_base: a => {a[3].base.ref = 'other';},
  unsafe_sha: a => {a[1].head_sha = '$(id)';},
  old_attempt: a => {a[1].run_attempt = 2;},
  missing_artifact: a => {a[4] = [];},
  duplicate_artifact: a => {a[4].push({...a[4][0]});},
  expired_artifact: a => {a[4][0].expired = true;},
  huge_artifact: a => {a[4][0].size_in_bytes = 2 ** 32;},
  target_event: a => {a[1].event = 'pull_request_target';},
  wrong_main: a => {a[1].event = 'push'; a[1].head_branch = 'other';},
  agent_pr: a => {a[5] = 'agent'; a[2].path = a[1].path = '.github/workflows/agent-image.yml';},
})) {
  test(`reject ${name}`, () => {const args = fixture(); mutate(args); assert.throws(() => validate(...args));});
}
