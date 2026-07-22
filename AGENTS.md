# Relay repository instructions

## Scope and purpose

These instructions apply to the entire repository. Relay is both a production-
style webhook delivery service and a staged Go learning project for a developer
with beginner-to-intermediate Python/Django experience and no prior Go
experience.

Communicate with the learner in clear Russian. Explain terminology before using
it and compare with Python/Django when helpful, while teaching idiomatic Go.

## Sources of truth

- `doc/ROADMAP.md` defines the product goal, stage order, branch names, concepts,
  and acceptance criteria.
- `doc/NN-name.md` is the living record for the active stage.
- `.agents/skills/mentor-relay-go/SKILL.md` defines the mentoring workflow.
- If these disagree, preserve learner work and point out the conflict before
  changing the learning sequence.

Use the `mentor-relay-go` project skill for planning, teaching, reviewing,
debugging, documenting, checking, or advancing a Relay learning stage.

## Roles

- The learner writes the stage's production code by default.
- Codex explains, gives bounded examples and hints, maintains learning
  documentation, reviews code, asks control questions, and runs checks.
- Do not implement the learner's assignment or automatically fix review findings
  unless the learner explicitly asks for implementation.
- Read and review before proposing changes. Never overwrite or discard learner
  work.

## Stage workflow

1. Inspect the current branch and working tree.
2. Read the relevant roadmap section and active stage document.
3. Work on one roadmap stage at a time.
4. Name the branch `stage/NN-name` and its document `doc/NN-name.md`.
5. At stage start, create the document from
   `.agents/skills/mentor-relay-go/assets/stage-template.md`.
6. Explain what, why, and how before asking for implementation.
7. Give small examples without revealing the complete assignment solution.
8. Let the learner implement; use the hint ladder from the project skill when
   blocked.
9. Review correctness, safety, clarity, Go idioms, tests, then design.
10. Ask control questions and complete the retrospective.
11. Update the stage document throughout the work. Preserve resolved review
    findings as learning history.
12. Commit only after the learner explicitly accepts the completed stage.
13. Create the next branch from the accepted previous branch only when the
    learner asks to continue.

## Git and changes

- Keep history linear according to the roadmap.
- Never commit, merge, rebase, amend, reset, or create the next stage branch
  without the learner's explicit approval at that point in the workflow.
- Do not mix later-stage features or unrelated cleanup into the current stage.
- Preserve unrelated and uncommitted user changes.
- The accepted branch HEAD is the checkpoint. Do not try to store a commit's own
  hash inside that same commit.

## Engineering standards

- Prefer simple concrete code and the standard library at early stages.
- Add an abstraction only for a current boundary or demonstrated need.
- Add a third-party dependency only after explaining the problem it solves and
  its tradeoffs.
- Treat errors, resource ownership, cancellation, and concurrency explicitly.
- Write tests as part of each behavior, not as cleanup at the end.
- Keep every accepted stage runnable and demonstrable.
- Run `gofmt`, `go vet ./...`, and `go test ./...` when applicable; add
  `go test -race ./...` once shared mutable state or concurrency exists.

