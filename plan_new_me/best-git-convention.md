# Best Git Convention

How our team uses Git. It's short on purpose: read it once, then use the [cheat sheet](#8-cheat-sheet).

**Contents**

1. [The five rules](#1-the-five-rules)
2. [One-time setup](#2-one-time-setup)
3. [Branch names](#3-branch-names)
4. [Commit messages](#4-commit-messages)
5. [The feature loop, step by step](#5-the-feature-loop-step-by-step)
6. [Conflicts](#6-conflicts)
7. [What the graph looks like](#7-what-the-graph-looks-like)
8. [Cheat sheet](#8-cheat-sheet)
9. [Branch list](#9-branch-list)

---

## 1. The five rules

1. **Never commit on `main`.** `main` always builds and passes `make lint` and `make test`.
2. **One branch per feature.** One branch does one thing, such as the login screen, the Makefile or the `TAKE` command.
3. **Small commits with clear messages.** Each commit is one logical change that still builds.
4. **Merge with `--no-ff`.** This always creates a merge commit, so every feature shows up as its own bump in the graph. A fast-forward merge (`--ff`) puts the commits straight onto `main`, and the feature branch disappears from the graph.
5. **Delete the branch after merging.** Finished work lives in `main`.

**Why it matters for 42:** the subject requires that "each member should have commits" and "proper commit messages and branch management". The graph shows evaluators who built what.

---

## 2. One-time setup

Each person runs these once, on their own machine:

```sh
git config --global user.name  "<your 42 login>"
git config --global user.email "<your email>"
git config --global pull.ff only        # `git pull` never creates surprise merge commits
git config --global init.defaultBranch main
```

**Optional shortcut** for the graph:

```sh
git config --global alias.graph "log --oneline --graph --all --decorate"
```

Then `git graph` shows the tree.

**If we use GitHub:** in *Settings → General → Pull Requests*, leave only **"Allow merge commits"** turned on, and turn off squash and rebase merging. GitHub's merge button then does a `--no-ff` merge.

**`.gitignore`** (created on Day 1):

```
bin/
*.exe
.env
.DS_Store
.vscode/
.idea/
```

---

## 3. Branch names

Format: **`<type>/<short-name>`**, lowercase, words joined by `-`.

| Type | Use for | Example |
|---|---|---|
| `feat` | A new feature | `feat/gui-login` |
| `fix` | A bug fix | `fix/take-race` |
| `docs` | README or plan changes only | `docs/readme-testing` |
| `test` | Tests only | `test/combat` |
| `chore` | Build, config, tooling | `chore/makefile` |
| `refactor` | Code cleanup with no behavior change | `refactor/gui-render` |

Keep a branch short-lived, ideally under a day. A big feature becomes several branches.

---

## 4. Commit messages

Format: **`<type>(<scope>): <what it does>`**

- **type:** same list as branch types.
- **scope:** the area touched: `gui`, `server`, `game`, `world`, `cli`, `make`, `readme`.
- **summary:** imperative ("add", not "added"), lowercase, no full stop, about 50 characters at most.

**Good:**

```
feat(gui): add login screen
feat(gui): show NAME_IN_USE error under name field
fix(server): drop items when player disconnects
chore(make): add run-client-gui target
docs(readme): write testing section
```

**Bad:**

```
update
fix stuff
WIP
changes to app.js
```

If a commit needs explaining, add a blank line after the summary, then say **why** in a short paragraph.

**What one commit should be:** one change you could describe in one line, which still builds. Commit with `git add -p` to choose exactly which lines go in.

---

## 5. The feature loop, step by step

Here, building the GUI login screen is the example.

**1. Start from an up-to-date `main`:**

```sh
git switch main
git pull
```

**2. Create the feature branch:**

```sh
git switch -c feat/gui-login
```

**3. Work and commit in small steps:**

```sh
git add -p                                    # review each change before staging it
git commit -m "feat(gui): add login form markup"
git add -p
git commit -m "feat(gui): send CONNECT and show errors"
```

**4. Push the branch** (also works as a backup):

```sh
git push -u origin feat/gui-login             # first push; afterwards just `git push`
```

**5. Catch up with `main` before merging.** Do this only on your own branch, which nobody else is using:

```sh
git fetch
git rebase origin/main
make lint && make test                        # still builds on top of the latest main?
git push --force-with-lease                   # safe force: refuses if someone else pushed
```

Rebasing replays your commits on top of the latest `main`. Each feature then branches off the current tip, which keeps the graph tidy.

**6. Get a quick review.** Open a pull request (or show your teammate the diff). They read it and ask questions. This is also how everyone learns each other's code for the evaluation.

**7. Merge with `--no-ff`.** Use GitHub's merge button, or run:

```sh
git switch main
git pull
git merge --no-ff feat/gui-login -m "Merge branch 'feat/gui-login'"
git push
```

**8. Delete the branch:**

```sh
git branch -d feat/gui-login
git push origin --delete feat/gui-login
```

---

## 6. Conflicts

They happen during step 5 (`git rebase`) when two people changed the same lines.

```sh
git status                     # lists the conflicted files
# open each file, keep the right code, delete the <<<<<<< ======= >>>>>>> markers
git add <file>
git rebase --continue          # repeat until done
```

- Stuck? Run `git rebase --abort` to go back to how things were before, then ask the file's owner.
- Our file ownership rule (see `plan.md`) means conflicts should be rare. If you need a change in someone else's file, ask them.

---

## 7. What the graph looks like

Run `git log --oneline --graph --all`. With `--no-ff` merges it looks like this:

```
*   e4c1a2f Merge branch 'feat/gui-login'
|\
| * 9b3d7e1 feat(gui): send CONNECT and show errors
| * 51a8c0d feat(gui): add login form markup
|/
*   7f20b44 Merge branch 'chore/makefile'
|\
| * c3e9f10 chore(make): add all required targets
|/
* 1a0d5e3 chore: initial project skeleton
```

Every feature is one visible loop with its own commits inside. With a fast-forward merge, the same history would be one flat line, and you couldn't tell where a feature started or ended.

---

## 8. Cheat sheet

| I want to | Command |
|---|---|
| Start a feature | `git switch main && git pull && git switch -c feat/<name>` |
| See what changed | `git status` and `git diff` |
| Commit | `git add -p && git commit -m "type(scope): summary"` |
| Push my branch | `git push -u origin feat/<name>` (first time), then `git push` |
| Catch up with main | `git fetch && git rebase origin/main && git push --force-with-lease` |
| Merge | `git switch main && git pull && git merge --no-ff feat/<name> && git push` |
| Delete the branch | `git branch -d feat/<name> && git push origin --delete feat/<name>` |
| See the graph | `git log --oneline --graph --all` |
| Undo my last commit, keep the changes | `git reset --soft HEAD~1` (only if not pushed yet) |
| Throw away changes to one file | `git restore <file>` |

**Before every merge, check:**

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] The commit messages follow section 4
- [ ] A teammate looked at it
- [ ] I only touched files I own, or the owner agreed

**Never:**

- Commit directly on `main`
- Use `git push --force` on `main` or on a teammate's branch
- Commit `bin/`, secrets or `.env` files
- Commit for someone else. Each person commits their own work

---

## 9. Branch list

Each step in the role plans is one branch. Role C's, from [plan-role-c.md](plan-role-c.md#7-implementation-steps):

| Step | Branch | Example commits |
|---|---|---|
| 1 | `chore/makefile` | `chore(make): add all required targets` |
| 2 | `feat/gui-layout` | `feat(gui): add page skeleton with all panels` |
| 3 | `feat/gui-style` | `feat(gui): add palette and fonts`, `feat(gui): add pixel panel and button styles` |
| 4 | `feat/gui-connection` | `feat(gui): open websocket and show raw lines` |
| 5 | `feat/gui-login` | `feat(gui): add login form`, `feat(gui): add disconnected screen` |
| 6 | `feat/gui-pending-queue` | `feat(gui): match replies to commands in order`, `feat(gui): add raw command input` |
| 7 | `feat/gui-room-view` | `feat(gui): render LOOK reply`, `feat(gui): add id labels and image fallback` |
| 8 | `feat/gui-actions` | `feat(gui): add command bar`, `feat(gui): add take, drop and npc buttons` |
| 9 | `feat/gui-events-chat` | `feat(gui): handle room and global events`, `feat(gui): add chat tabs` |
| 10 | `feat/gui-group` | `feat(gui): add group panel` |
| 11 | `feat/gui-polling` | `feat(gui): poll LOOK every 5 seconds` |
| 12 | `feat/gui-map` | `feat(gui): draw map of our rooms` |
| 13 | `feat/gui-art` | `feat(gui): add room scenes`, `feat(gui): add npc and item sprites` |
| 14 | `docs/readme-*` | `docs(readme): write building and running` |

A and B make their own lists the same way, such as `feat/server-accept-loop`, `feat/game-combat` and `feat/world-loader`.
