#!/usr/bin/env bash
# Публикация репозитория в GitHub.
#
# Использование (один из вариантов):
#   GH_TOKEN=ghp_xxx bash scripts/publish-to-github.sh
#   GH_TOKEN=ghp_xxx GH_USER=wester11 GH_REPO=void_wg bash scripts/publish-to-github.sh
#
# Перед запуском:
#   1. Создайте personal access token (classic, scope `repo`)
#      https://github.com/settings/tokens
#   2. Создайте пустой репозиторий: https://github.com/new  -> wester11/void_wg
#      (без README/license — иначе будет конфликт при первом push)
set -Eeuo pipefail

GH_USER="${GH_USER:-wester11}"
GH_REPO="${GH_REPO:-void_wg}"
GH_BRANCH="${GH_BRANCH:-main}"
GH_TOKEN="${GH_TOKEN:?Set GH_TOKEN env var (GitHub personal access token, scope: repo)}"

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

echo "Repo:   $GH_USER/$GH_REPO"
echo "Branch: $GH_BRANCH"
echo "Path:   $ROOT"

if [ ! -d .git ]; then
    git init -b "$GH_BRANCH"
fi

git config user.email "${GIT_EMAIL:-noreply@void-wg.local}"
git config user.name  "${GIT_NAME:-void-wg}"

git add -A
if git diff --cached --quiet; then
    echo "Nothing to commit."
else
    git commit -m "${COMMIT_MSG:-void-wg release}"
fi

git remote remove origin 2>/dev/null || true
git remote add origin "https://${GH_USER}:${GH_TOKEN}@github.com/${GH_USER}/${GH_REPO}.git"

git push -u origin "$GH_BRANCH"
echo
echo "Published: https://github.com/${GH_USER}/${GH_REPO}"
echo "One-click install command for users:"
echo
echo "  bash <(curl -Ls https://raw.githubusercontent.com/${GH_USER}/${GH_REPO}/${GH_BRANCH}/scripts/install.sh)"
