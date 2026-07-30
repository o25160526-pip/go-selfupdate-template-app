#!/usr/bin/env sh
set -eu
fail=0
check_file(){ if [ -f "$1" ]; then printf 'PASS file %s\n' "$1"; else printf 'FAIL missing %s\n' "$1"; fail=1; fi; }
check_text(){ if grep -Eq "$2" "$1"; then printf 'PASS %s :: %s\n' "$1" "$2"; else printf 'FAIL %s :: %s\n' "$1" "$2"; fail=1; fi; }
for f in cmd/app/main.go internal/version/version.go internal/updater/updater.go internal/updater/source_github.go internal/updater/source_azure.go internal/updater/manifest.go internal/updater/cache.go internal/updater/metadata_cache.go internal/updater/apply.go internal/signing/minisign.go cmd/keygen/main.go internal/features/features.go internal/tray/tray_notray.go internal/tray/tray_tray.go internal/ui/menu.go configs/manifest.example.json .github/workflows/build.yml azure-pipelines.yml docs/ADDING_A_FEATURE.md; do check_file "$f"; done
check_text internal/updater/source.go 'type Source interface'
check_text internal/features/features.go 'type Feature interface'
check_text internal/updater/updater.go 'ExitUpToDate.*10'
check_text internal/updater/updater.go 'ExitSourcesUnavailable.*50'
check_text internal/updater/updater.go 'Candidates'
check_text internal/updater/metadata_cache.go '15 \\* time.Minute'
check_text internal/signing/minisign.go 'trusted comment:'
check_text cmd/binarysign/main.go 'signing.Sign'
check_text internal/buildinfo/buildinfo.go 'CurrentPublicKey'
check_text internal/buildinfo/buildinfo.go 'NextPublicKey'
check_text .github/workflows/build.yml 'cancel-in-progress: true'
check_text .github/workflows/build.yml 'smoke-update:'
check_text .github/workflows/build.yml 'Wait for current draft assets'
check_text scripts/e2e-local.sh 'rollback'
check_text scripts/build-matrix.sh 'windows linux darwin'
check_text scripts/build-matrix.sh 'amd64 arm64'
exit "$fail"
