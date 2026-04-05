# gh-tag

The missing tag command.

`gh tag` replaces the manual `git tag v1.0.0 && git push origin v1.0.0` ritual with a single interactive command that discovers existing remote tags, asks how to bump the version, and handles everything in one shot.

Semver is *the* best way to version your packages and repos, and this tool emphasizes and favors this pattern. https://semver.org/

## Installation

```
gh extension install karldreher/gh-tag
```

## Usage

```
gh tag              # bump and push the next semver tag
gh tag list         # list all semver tags, newest first
gh tag view         # show the latest tag and its commit SHA
gh tag view <tag>   # show a specific tag and its commit SHA
gh tag prefix       # view or change the tag prefix (default: v)
```

`gh tag view --web` opens the tag's GitHub releases page in the browser. When no tag argument is given, it requires exactly one semver tag in the repository — if multiple exist, specify one explicitly (e.g. `gh tag view v1.2.3 --web`).

Run any command with `--help` for all available flags.

## Example

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
  "prefix": "v",
  "overwrite_confirmed": false
}
```

Set it interactively with `gh tag prefix --edit`. The default prefix is `v`, producing tags like `v1.2.3`. Custom prefixes (e.g. `release-`) produce tags like `release-1.2.3`.
