const LEDGER_BRANCH = 'cla-signatures';
const LEDGER_PATH = 'signatures/version1/cla.json';
const DOCUMENT_PATH = 'CLA.md';
const COMMENT_MARKER = '<!-- cla-check -->';
const GITHUB_ACTIONS_BOT = 'github-actions[bot]';
const GITHUB_ACTIONS_BOT_ID = 41898282;

module.exports = async ({ github, context, core }) => {
  if (context.eventName === 'pull_request_target') {
    await checkPullRequest(github, context, core, context.payload.pull_request.number);
    return;
  }
  if (context.eventName === 'issue_comment') {
    await handleIssueComment(github, context, core);
    return;
  }
  core.info(`No CLA action for event ${context.eventName}.`);
};

async function checkPullRequest(github, context, core, pullNumber) {
  const { identified, unidentifiable } = await collectAuthors(github, context, pullNumber);
  const ledger = await readLedger(github, context);
  const pending = identified.filter(
    (author) => !ledger.signedContributors.some((signature) => signature.id === author.id),
  );
  const satisfied = pending.length === 0 && unidentifiable.length === 0;

  await upsertComment(
    github,
    context,
    pullNumber,
    commentBody(context, pending, unidentifiable),
    !satisfied,
  );

  if (satisfied) {
    core.info('Every commit author in this pull request has signed the CLA.');
    return;
  }

  const missing = [];
  if (pending.length > 0) {
    missing.push(`not signed by ${pending.map((author) => `@${author.login}`).join(', ')}`);
  }
  if (unidentifiable.length > 0) {
    missing.push(
      `no GitHub account linked to ${unidentifiable.map(describeAuthor).join(', ')}`,
    );
  }
  core.setFailed(`CLA check failed: ${missing.join('; ')}.`);
}

async function handleIssueComment(github, context, core) {
  if (!context.payload.issue.pull_request) {
    return;
  }

  const comment = context.payload.comment;
  // The bot's own comment quotes the phrase, so a bot can never sign.
  if (comment.user.type === 'Bot' || comment.user.login === GITHUB_ACTIONS_BOT) {
    return;
  }

  const body = comment.body.trim().toLowerCase();
  const phrase = process.env.CLA_SIGNATURE_PHRASE.trim().toLowerCase();
  const recheck = process.env.CLA_RECHECK_KEYWORD.trim().toLowerCase();
  if (body !== phrase && body !== recheck) {
    return;
  }

  const { data: pullRequest } = await github.rest.pulls.get({
    ...context.repo,
    pull_number: context.payload.issue.number,
  });

  if (body === phrase && !(await sign(github, context, core, comment, pullRequest.number))) {
    return;
  }

  await checkPullRequest(github, context, core, pullRequest.number);
  await rerunHeadCheck(github, context, core, pullRequest.head.sha);
}

// Records the commenter's signature, unless they are not one of the pull request's pending commit authors.
async function sign(github, context, core, comment, pullNumber) {
  const { identified } = await collectAuthors(github, context, pullNumber);
  const ledger = await readLedger(github, context);
  const authored = identified.some((author) => author.id === comment.user.id);
  const alreadySigned = ledger.signedContributors.some((signature) => signature.id === comment.user.id);

  if (!authored || alreadySigned) {
    core.info(`@${comment.user.login} has no pending commit in this pull request; the comment does not sign.`);
    return false;
  }

  await appendToLedger(github, context, {
    login: comment.user.login,
    id: comment.user.id,
    pullRequestNo: pullNumber,
    commentId: comment.id,
    signedAt: new Date().toISOString(),
    documentSha: await documentSha(github, context),
  });
  core.info(`Recorded the CLA signature of @${comment.user.login}.`);
  return true;
}

// Splits every commit author in the pull request into accounts GitHub resolved and raw git authors it could not, dropping the allowlist.
async function collectAuthors(github, context, pullNumber) {
  const commits = await github.paginate(github.rest.pulls.listCommits, {
    ...context.repo,
    pull_number: pullNumber,
    per_page: 100,
  });
  const allowlist = process.env.CLA_ALLOWLIST.split(',')
    .map((login) => login.trim().toLowerCase())
    .filter(Boolean);

  const identified = new Map();
  const unidentifiable = new Map();

  for (const commit of commits) {
    if (!commit.author) {
      const author = { name: commit.commit.author.name, email: commit.commit.author.email };
      unidentifiable.set(describeAuthor(author).toLowerCase(), author);
      continue;
    }
    if (commit.author.id === GITHUB_ACTIONS_BOT_ID || allowlist.includes(commit.author.login.toLowerCase())) {
      continue;
    }
    identified.set(commit.author.id, { id: commit.author.id, login: commit.author.login });
  }

  return { identified: [...identified.values()], unidentifiable: [...unidentifiable.values()] };
}

function describeAuthor(author) {
  return `${author.name} <${author.email}>`;
}

async function readLedger(github, context) {
  try {
    const { data } = await github.rest.repos.getContent({
      ...context.repo,
      path: LEDGER_PATH,
      ref: LEDGER_BRANCH,
    });
    const ledger = JSON.parse(Buffer.from(data.content, 'base64').toString('utf8'));
    return { sha: data.sha, signedContributors: ledger.signedContributors ?? [] };
  } catch (error) {
    if (error.status !== 404) {
      throw error;
    }
    return { sha: undefined, signedContributors: [] };
  }
}

async function appendToLedger(github, context, entry) {
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const ledger = await readLedger(github, context);
    const signedContributors = [...ledger.signedContributors, entry];
    const parameters = {
      ...context.repo,
      path: LEDGER_PATH,
      branch: LEDGER_BRANCH,
      message: `Record the CLA signature of @${entry.login} (#${entry.pullRequestNo})`,
      content: Buffer.from(`${JSON.stringify({ signedContributors }, null, 2)}\n`, 'utf8').toString('base64'),
    };
    if (ledger.sha) {
      parameters.sha = ledger.sha;
    }

    try {
      await github.rest.repos.createOrUpdateFileContents(parameters);
      return;
    } catch (error) {
      if (error.status !== 409 || attempt === 1) {
        throw error;
      }
    }
  }
}

// Blob SHA of the CLA text in force, so a signature names the exact document it agreed to.
async function documentSha(github, context) {
  const { data } = await github.rest.repos.getContent({
    ...context.repo,
    path: DOCUMENT_PATH,
    ref: context.payload.repository.default_branch,
  });
  return data.sha;
}

async function upsertComment(github, context, pullNumber, body, createIfAbsent) {
  const comments = await github.paginate(github.rest.issues.listComments, {
    ...context.repo,
    issue_number: pullNumber,
    per_page: 100,
  });
  const existing = comments.find((comment) => comment.body?.includes(COMMENT_MARKER));

  if (!existing) {
    if (createIfAbsent) {
      await github.rest.issues.createComment({ ...context.repo, issue_number: pullNumber, body });
    }
    return;
  }
  if (existing.body !== body) {
    await github.rest.issues.updateComment({ ...context.repo, comment_id: existing.id, body });
  }
}

function commentBody(context, pending, unidentifiable) {
  const { owner, repo } = context.repo;
  const document = `https://github.com/${owner}/${repo}/blob/${context.payload.repository.default_branch}/${DOCUMENT_PATH}`;
  const lines = [COMMENT_MARKER, '## Contributor License Agreement', ''];

  if (pending.length === 0 && unidentifiable.length === 0) {
    lines.push('Every commit author in this pull request has signed the CLA. Thank you.');
    return lines.join('\n');
  }

  if (pending.length > 0) {
    lines.push(
      `${pending.map((author) => `@${author.login}`).join(', ')} — please read the [Contributor License Agreement](${document}) and sign it by posting a new comment whose entire text is:`,
      '',
      '```',
      process.env.CLA_SIGNATURE_PHRASE,
      '```',
      '',
      'The comment must contain nothing else.',
      '',
    );
  }

  if (unidentifiable.length > 0) {
    lines.push(
      '### Commit authors with no linked GitHub account',
      '',
      'These authors cannot be identified, so their signature cannot be recorded:',
      '',
      ...unidentifiable.map((author) => `- \`${describeAuthor(author)}\``),
      '',
      'Add the address to your GitHub account under Settings → Emails, then re-author the commits with `git commit --amend --reset-author` (or `git rebase --exec`) and force-push.',
      '',
    );
  }

  lines.push(`Comment \`${process.env.CLA_RECHECK_KEYWORD}\` to run this check again.`);
  return lines.join('\n');
}

// Only the run bound to the pull request head produces a check on the commit branch protection reads; an issue_comment run lands on the default branch.
async function rerunHeadCheck(github, context, core, headSha) {
  const { data } = await github.rest.actions.listWorkflowRunsForRepo({
    ...context.repo,
    event: 'pull_request_target',
    head_sha: headSha,
    per_page: 100,
  });
  const run = data.workflow_runs.find((candidate) => candidate.path === currentWorkflowPath());

  if (!run) {
    core.info(`No pull_request_target run found for ${headSha}; nothing to re-run.`);
    return;
  }
  if (run.conclusion === 'failure') {
    await github.rest.actions.reRunWorkflowFailedJobs({ ...context.repo, run_id: run.id });
    return;
  }
  await github.rest.actions.reRunWorkflow({ ...context.repo, run_id: run.id });
}

// GITHUB_WORKFLOW_REF is `owner/repo/.github/workflows/file.yml@ref`.
function currentWorkflowPath() {
  const reference = process.env.GITHUB_WORKFLOW_REF;
  return reference.slice(0, reference.lastIndexOf('@')).split('/').slice(2).join('/');
}
