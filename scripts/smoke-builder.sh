#!/usr/bin/env bash
set -euo pipefail
BASE="${AGENT_BASE_URL:-http://127.0.0.1:3001/api/agent/v1}"
JSON='Content-Type: application/json'
project=$(curl -fsS -X POST "$BASE/builders" -H "$JSON" -d '{"name":"Builder Smoke","kind":"website","entry":"index.html","files":{"index.html":"<main>builder smoke</main>","app.js":"console.log(\"smoke\")"}}')
id=$(jq -r '.id' <<<"$project")
test -n "$id" -a "$id" != null
visual=$(curl -fsS -X POST "$BASE/builders/$id/visual" -H "$JSON" -d '{"components":[{"id":"hero","type":"hero","props":{"text":"Ollama Full"},"width":640,"height":120}]}')
jq -e '.components[0].id == "hero"' <<<"$visual" >/dev/null
preview=$(curl -fsS -X POST "$BASE/builders/$id/preview" -H "$JSON" -d '{}')
jq -e '.project.status == "preview" and (.artifact.sha256 | length) > 0' <<<"$preview" >/dev/null
archive=$(curl -fsS -X POST "$BASE/builders/$id/export" -H "$JSON" -d '{}')
jq -e '(.archive_path | length) > 0' <<<"$archive" >/dev/null
published=$(curl -fsS -X POST "$BASE/builders/$id/publish" -H "$JSON" -d '{}')
jq -e '.project.status == "published" and (.published_path | length) > 0' <<<"$published" >/dev/null
status=$(curl -sS -o /tmp/builder-deploy-approval.json -w '%{http_code}' -X POST "$BASE/builders/$id/deploy/generic" -H "$JSON" -d '{"target":"https://deploy.example.com/project","approved":true}')
test "$status" = 400
printf 'BUILDER_SMOKE=PASS project=%s\n' "$id"
