const { test } = require('node:test');
const assert = require('node:assert');

process.env.CLA_ALLOWLIST = 'alefcastelo,dependabot[bot]';
process.env.CLA_SIGNATURE_PHRASE = 'I have read the CLA Document and I hereby sign the CLA';
process.env.CLA_RECHECK_KEYWORD = 'recheck';
process.env.GITHUB_WORKFLOW_REF = 'contracttesting/broker/.github/workflows/cla.yml@refs/heads/main';

const cla = require('./cla.js');

const repo = { owner: 'contracttesting', repo: 'broker' };

function notFound() {
  const error = new Error('Not Found');
  error.status = 404;
  return error;
}

// Records every API call the module makes against a mutable repository state.
function makeWorld(options) {
  const world = {
    commits: options.commits,
    ledger: options.ledger,
    ledgerSha: options.ledger ? 'ledgersha' : undefined,
    comments: options.comments ?? [],
    calls: [],
    commitPages: [],
    conflictsLeft: options.conflictsLeft ?? 0,
    nextCommentId: 900,
  };

  const github = {
    // Follows pages until one comes back shorter than per_page, like Octokit's paginate.
    paginate: async (route, parameters) => {
      const items = [];
      for (let page = 1; ; page += 1) {
        const { data } = await route({ ...parameters, page });
        items.push(...data);
        if (data.length < parameters.per_page) {
          return items;
        }
      }
    },
    rest: {
      pulls: {
        listCommits: async (p) => {
          world.calls.push(['listCommits', p.per_page]);
          world.commitPages.push(p.page);
          return { data: world.commits.slice((p.page - 1) * p.per_page, p.page * p.per_page) };
        },
        get: async (p) => ({
          data: { number: p.pull_number, head: { sha: 'headsha' } },
        }),
      },
      repos: {
        getContent: async (p) => {
          if (p.path === 'CLA.md') return { data: { sha: 'cladocsha' } };
          if (world.ledger === null) throw notFound();
          return {
            data: {
              sha: world.ledgerSha,
              content: Buffer.from(JSON.stringify(world.ledger)).toString('base64'),
            },
          };
        },
        createOrUpdateFileContents: async (p) => {
          if (world.conflictsLeft > 0) {
            world.conflictsLeft -= 1;
            world.ledger = { signedContributors: [{ login: 'racer', id: 7 }] };
            world.ledgerSha = 'newsha';
            const error = new Error('conflict');
            error.status = 409;
            throw error;
          }
          world.calls.push(['writeLedger', p.sha, p.branch, p.path]);
          world.written = Buffer.from(p.content, 'base64').toString('utf8');
          world.ledger = JSON.parse(world.written);
          world.ledgerSha = 'aftersha';
          return { data: {} };
        },
      },
      issues: {
        listComments: async () => ({ data: world.comments }),
        createComment: async (p) => {
          world.calls.push(['createComment']);
          world.comments.push({ id: world.nextCommentId++, body: p.body, user: { login: 'github-actions[bot]' } });
          return { data: {} };
        },
        updateComment: async (p) => {
          world.calls.push(['updateComment', p.comment_id]);
          world.comments.find((c) => c.id === p.comment_id).body = p.body;
          return { data: {} };
        },
      },
      actions: {
        listWorkflowRunsForRepo: async (p) => {
          world.calls.push(['listRuns', p.event, p.head_sha]);
          return {
            data: {
              workflow_runs: [
                { id: 1, path: '.github/workflows/ci.yml', conclusion: 'failure' },
                { id: 42, path: '.github/workflows/cla.yml', conclusion: 'failure' },
              ],
            },
          };
        },
        reRunWorkflowFailedJobs: async (p) => world.calls.push(['reRunFailedJobs', p.run_id]),
        reRunWorkflow: async (p) => world.calls.push(['reRunWorkflow', p.run_id]),
      },
    },
  };

  const core = {
    info: (m) => world.calls.push(['info', m]),
    setFailed: (m) => world.calls.push(['setFailed', m]),
  };

  return { world, github, core };
}

const prContext = {
  eventName: 'pull_request_target',
  repo,
  payload: { pull_request: { number: 5 }, repository: { default_branch: 'main' } },
};

function commentContext(body, user, isPr = true) {
  return {
    eventName: 'issue_comment',
    repo,
    payload: {
      issue: { number: 5, pull_request: isPr ? {} : undefined },
      comment: { id: 555, body, user },
      repository: { default_branch: 'main' },
    },
  };
}

const identified = (login, id) => ({ sha: 'c1', author: { login, id }, commit: { author: { name: login, email: `${login}@x` } } });
const unlinked = (name, email) => ({ sha: 'c2', author: null, commit: { author: { name, email } } });

test('allowlisted author + missing ledger passes and creates no comment', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('alefcastelo', 1)], ledger: null });
  await cla({ github, context: prContext, core });
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'setFailed'), []);
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'createComment'), []);
  assert.deepStrictEqual(world.calls[0], ['listCommits', 100]);
});

test('github-actions[bot] author by id is dropped', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('github-actions[bot]', 41898282)], ledger: null });
  await cla({ github, context: prContext, core });
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'setFailed'), []);
});

test('pending + unidentifiable fails with one comment naming both', async () => {
  const { world, github, core } = makeWorld({
    commits: [identified('octo', 2), unlinked('Ghost', 'ghost@nowhere'), unlinked('Ghost', 'ghost@nowhere')],
    ledger: { signedContributors: [] },
  });
  await cla({ github, context: prContext, core });
  const failed = world.calls.find((c) => c[0] === 'setFailed')[1];
  assert.match(failed, /@octo/);
  assert.match(failed, /Ghost <ghost@nowhere>/);
  assert.strictEqual(world.comments.length, 1);
  assert.match(world.comments[0].body, /<!-- cla-check -->/);
  assert.match(world.comments[0].body, /### Commit authors with no linked GitHub account/);
  assert.strictEqual((world.comments[0].body.match(/Ghost <ghost@nowhere>/g) || []).length, 1);

  world.calls.length = 0;
  await cla({ github, context: prContext, core });
  await cla({ github, context: prContext, core });
  assert.strictEqual(world.comments.length, 1, 'three runs must leave exactly one comment');
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'createComment'), []);
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'updateComment'), [], 'identical body is not rewritten');
});

test('authors past the first page are gated', async () => {
  const commits = Array.from({ length: 250 }, () => identified('octo', 2));
  commits[249] = identified('lastpage', 42);
  const { world, github, core } = makeWorld({
    commits,
    ledger: { signedContributors: [{ login: 'octo', id: 2 }] },
  });
  await cla({ github, context: prContext, core });
  const failed = world.calls.filter((c) => c[0] === 'setFailed');
  assert.strictEqual(failed.length, 1, 'the author found only on the last page must fail the check');
  assert.match(failed[0][1], /@lastpage/);
  assert.strictEqual(world.comments.length, 1);
  assert.match(world.comments[0].body, /@lastpage/);
  assert.deepStrictEqual(world.commitPages, [1, 2, 3]);
});

test('signed author passes and the existing comment flips to the all-signed state', async () => {
  const { world, github, core } = makeWorld({
    commits: [identified('octo', 2)],
    ledger: { signedContributors: [{ login: 'octo', id: 2 }] },
    comments: [{ id: 900, body: '<!-- cla-check -->\nplease sign', user: { login: 'github-actions[bot]' } }],
  });
  await cla({ github, context: prContext, core });
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'setFailed'), []);
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'updateComment'), [['updateComment', 900]]);
  assert.match(world.comments[0].body, /Every commit author in this pull request has signed the CLA/);
});

test('exact phrase from a pending author records six fields and re-runs the head run', async () => {
  const { world, github, core } = makeWorld({
    commits: [identified('octo', 2)],
    ledger: { signedContributors: [] },
  });
  const context = commentContext('  I HAVE read the CLA Document and I hereby sign the CLA \n', { login: 'octo', id: 2, type: 'User' });
  await cla({ github, context, core });

  const write = world.calls.find((c) => c[0] === 'writeLedger');
  assert.deepStrictEqual(write, ['writeLedger', 'ledgersha', 'cla-signatures', 'signatures/version1/cla.json']);
  const entry = JSON.parse(world.written).signedContributors[0];
  assert.deepStrictEqual(Object.keys(entry).sort(), ['commentId', 'documentSha', 'id', 'login', 'pullRequestNo', 'signedAt'].sort());
  assert.deepStrictEqual(entry.login, 'octo');
  assert.strictEqual(entry.id, 2);
  assert.strictEqual(entry.pullRequestNo, 5);
  assert.strictEqual(entry.commentId, 555);
  assert.strictEqual(entry.documentSha, 'cladocsha');
  assert.ok(!Number.isNaN(Date.parse(entry.signedAt)));
  assert.match(world.written, /^\{\n  "signedContributors": \[\n    \{\n      "login"/, 'indent must be 2 everywhere');
  assert.ok(world.written.endsWith('\n'));
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'setFailed'), [], 'the PR passes right after signing');
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'listRuns'), [['listRuns', 'pull_request_target', 'headsha']]);
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'reRunFailedJobs'), [['reRunFailedJobs', 42]]);
});

test('409 on the ledger write is retried once against the re-read sha', async () => {
  const { world, github, core } = makeWorld({
    commits: [identified('octo', 2)],
    ledger: { signedContributors: [] },
    conflictsLeft: 1,
  });
  const context = commentContext(process.env.CLA_SIGNATURE_PHRASE, { login: 'octo', id: 2, type: 'User' });
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls.find((c) => c[0] === 'writeLedger'), ['writeLedger', 'newsha', 'cla-signatures', 'signatures/version1/cla.json']);
  const signed = JSON.parse(world.written).signedContributors;
  assert.deepStrictEqual(signed.map((s) => s.login), ['racer', 'octo'], 'the concurrent signature is not lost');
});

test('phrase plus extra text does not sign', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('octo', 2)], ledger: { signedContributors: [] } });
  const context = commentContext(`Sure! ${process.env.CLA_SIGNATURE_PHRASE}`, { login: 'octo', id: 2, type: 'User' });
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls, []);
});

test('the bot quoting the phrase does not sign', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('octo', 2)], ledger: { signedContributors: [] } });
  const context = commentContext(process.env.CLA_SIGNATURE_PHRASE, { login: 'github-actions[bot]', id: 41898282, type: 'Bot' });
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls, []);
});

test('a non-author signing writes nothing and does not re-run', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('octo', 2)], ledger: { signedContributors: [] } });
  const context = commentContext(process.env.CLA_SIGNATURE_PHRASE, { login: 'stranger', id: 99, type: 'User' });
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'writeLedger'), []);
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'listRuns'), []);
});

test('recheck re-evaluates and re-runs without writing the ledger', async () => {
  const { world, github, core } = makeWorld({ commits: [identified('octo', 2)], ledger: { signedContributors: [] } });
  const context = commentContext(' Recheck ', { login: 'anyone', id: 77, type: 'User' });
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'writeLedger'), []);
  assert.strictEqual(world.comments.length, 1);
  assert.ok(world.calls.some((c) => c[0] === 'setFailed'));
  assert.deepStrictEqual(world.calls.filter((c) => c[0] === 'reRunFailedJobs'), [['reRunFailedJobs', 42]]);
});

test('a comment on a plain issue is ignored', async () => {
  const { world, github, core } = makeWorld({ commits: [], ledger: null });
  const context = commentContext(process.env.CLA_SIGNATURE_PHRASE, { login: 'octo', id: 2, type: 'User' }, false);
  await cla({ github, context, core });
  assert.deepStrictEqual(world.calls, []);
});
