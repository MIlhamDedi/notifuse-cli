# notifuse-cli

Agent-safe CLI for Notifuse workspaces. `notifuse` selects a named workspace profile, resolves the profile API key from a secret backend, injects `workspace_id`, and writes JSON responses to stdout.

The CLI is intentionally conservative: production broadcast scheduling and broad sends are blocked. Agents should use draft creation, compile/preview, and allowlisted test sends.

## Install

From source:

```sh
git clone git@github.com:milhamdedi/notifuse-cli.git
cd notifuse-cli
go build -o notifuse .
./notifuse --version
```

From a tagged release:

```sh
curl -fsSL https://raw.githubusercontent.com/milhamdedi/notifuse-cli/main/scripts/install.sh | bash
```

## Configuration

Config lives at:

```text
~/.config/notifuse-cli/config.yaml
```

Override it with `NOTIFUSE_CONFIG` or `--config`.

Example:

```yaml
default_profile: courtpro

profiles:
  courtpro:
    endpoint: https://notifuse.cakrawala.ai
    workspace_id: courtpro
    api_key_ref: keychain:notifuse-cli/courtpro
    max_recipients: 100
    allowed_test_recipients:
      - ilham@alif.ventures
      - radja@alif.ventures
```

Supported `api_key_ref` values:

- `keychain:notifuse-cli/courtpro` on macOS
- `env:NOTIFUSE_API_KEY_COURTPRO`
- `file:/run/secrets/notifuse_api_key`

Add a profile:

```sh
notifuse profiles add courtpro \
  --endpoint https://notifuse.cakrawala.ai \
  --workspace-id courtpro \
  --api-key-ref keychain:notifuse-cli/courtpro \
  --allowed-test-recipient ilham@alif.ventures \
  --allowed-test-recipient radja@alif.ventures \
  --max-recipients 100 \
  --default

notifuse auth login courtpro
```

## Commands

```sh
notifuse profiles list --pretty
notifuse openapi list --filter contacts --pretty

notifuse --profile courtpro contacts list --query limit=20 --pretty
notifuse --profile courtpro contacts count --pretty
notifuse --profile courtpro contacts upsert --body-file contact.json --dry-run --pretty

notifuse --profile courtpro templates list --pretty
notifuse --profile courtpro templates compile --body-file compile.json --dry-run --pretty
notifuse --profile courtpro templates validate-file --file examples/templates/video_analysis_ready.json --pretty
notifuse --profile courtpro templates compile-from-file --file examples/templates/video_analysis_ready.json --dry-run --pretty
notifuse --profile courtpro templates create-from-file --file examples/templates/video_analysis_ready.json --dry-run --pretty
notifuse --profile courtpro templates update-from-file --file examples/templates/video_analysis_ready.json --dry-run --pretty

notifuse --profile courtpro broadcasts create-draft --body-file broadcast.json --dry-run --pretty
notifuse --profile courtpro broadcasts test-send --body-file test-send.json --dry-run --pretty
notifuse --profile courtpro transactional test-send --body-file examples/transactional/video_analysis_ready_test_send.json --dry-run --pretty
```

`broadcasts test-send` requires the recipient email in the body to be present in the profile `allowed_test_recipients` list. `broadcasts schedule`, `broadcasts resume`, and production `broadcasts send` are blocked by design.

`transactional test-send` also requires `notification.contact.email` to be present in `allowed_test_recipients`. Production `transactional send` is blocked by design.

## Rich visual email templates

Notifuse's API accepts rich visual-email templates through `email.visual_editor_tree`, with rendered HTML stored in `email.compiled_preview`. This CLI can validate, create, update, and compile those JSON files, so templates can be reviewed in Git and managed by agents.

The visual tree itself is easiest to author in the Notifuse web dashboard, then export or copy into a JSON file. Hand-authoring minimal MJML trees is possible, but the dashboard remains the better editor for complex visual layouts.

## Raw allowlisted API calls

For new API coverage without exposing unrestricted raw access:

```sh
notifuse --profile courtpro api get /api/templates.list --pretty
notifuse --profile courtpro api post /api/templates.compile --body-file compile.json --dry-run --pretty
```

Only allowlisted endpoints are accepted. The CLI injects and validates `workspace_id` for every request.
