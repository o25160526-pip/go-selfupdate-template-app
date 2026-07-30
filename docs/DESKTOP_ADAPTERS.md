# Desktop adapters

The core updater and CLI build with `CGO_ENABLED=0` for all six targets. The default `!tray` file returns a clear unsupported error. Building with `-tags tray` enables a dependency-free interactive adapter used for compile and lifecycle testing.

For a production desktop distribution, replace only `internal/tray/tray_tray.go` with an adapter around `fyne.io/systray`. Keep the `tray` build tag. Linux builders then need `libayatana-appindicator3-dev`, CGO, and a native build runner for each target.

The same boundary applies to `internal/ui`: its `Choose` contract can be rendered through Bubble Tea without changing the updater, features, or command surface. Pin the chosen Bubble Tea major version and let Renovate/Dependabot propose upgrades.

macOS release binaries must be signed and notarized on a macOS runner. Windows installation under `Program Files` may require an elevated helper. Those deployment credentials are intentionally not embedded in the template.
