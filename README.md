# gh-tag

The missing tag command.

`gh tag` replaces the manual `git tag v1.0.0 && git push origin v1.0.0` ritual with a single interactive command that discovers existing remote tags, asks how to bump the version, and handles everything in one shot.

Semver is *the* best way to version your packages and repos, and this tool emphasizes and favors this pattern. https://semver.org/

Never fear a major release! 

## Installation

```
gh extension install karldreher/gh-tag
```

## Usage

### Create and push the next tag

```
gh tag
```

Fetches the latest semver tag from the remote, shows it, and asks whether to bump major, minor, or patch. Then confirms before tagging and pushing.

### Non-interactive bump type

```
gh tag --major
gh tag --minor
gh tag --patch
```

Skips the bump-type prompt. The flags are mutually exclusive.

### Skip confirmation

```
gh tag --confirm
gh tag --patch --confirm
```

`--confirm` bypasses the final `y/N` prompt. Combined with a bump flag, the command is fully non-interactive.

### View or change the tag prefix

```
gh tag prefix           # show current prefix (default: v)
gh tag prefix --edit    # interactively set a new prefix
```

The prefix is stored in `~/.gh-tag/config.json` and applies to all repos.

## How it works

1. Runs `git ls-remote --tags origin` to list all remote tags.
2. Finds the highest semver tag matching the configured prefix (numeric comparison, so `v1.0.10 > v1.0.9`).
3. If no tags exist, asks for the starting version type (major → `v1.0.0`, minor → `v0.1.0`, patch → `v0.0.1`).
4. If tags exist, asks how to bump: `M` for major, `m` for minor, `p` for patch. Full words (`major`, `minor`, `patch`) are also accepted.
5. Shows the planned commands and asks for confirmation (unless `--confirm`).
6. Runs `git tag <newtag>` then `git push origin <newtag>`.

Pre-release tags (`v1.0.0-beta`), annotated tag dereferences (`v1.0.0^{}`), and any non-semver refs are silently ignored during version discovery.

## Configuration

Config is stored at `~/.gh-tag/config.json`:

```json
{
  "prefix": "v"
}
```

Set it interactively with `gh tag prefix --edit`. The default prefix is `v`, producing tags like `v1.2.3`. Custom prefixes (e.g. `release-`) produce tags like `release-1.2.3`.

## Examples

```
$ gh tag
🏷️  gh tag — The missing command.

📡 Fetching remote tags...
✅ Latest tag: v1.2.3

📦 Bump type? [M]ajor / [m]inor / [p]atch: p

✨ New tag: v1.2.4

⚡ Ready to tag and push:
   git tag v1.2.4
   git push origin v1.2.4

🚀 Proceed? [y/N]: y

✅ Done! Tag v1.2.4 created and pushed.
```

```
$ gh tag --patch --confirm
🏷️  gh tag — The missing command.

📡 Fetching remote tags...
✅ Latest tag: v1.2.4

✨ New tag: v1.2.5

⚡ Ready to tag and push:
   git tag v1.2.5
   git push origin v1.2.5

✅ Done! Tag v1.2.5 created and pushed.
```
