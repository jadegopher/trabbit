# Architecture

Read [Architecture](CONTRIBUTING.md#architecture) and [Code guidelines](CONTRIBUTING.md#code-guidelines) before changing code. Follow the ownership, message routing, rendering and testing rules there. Update those sections when the architecture changes.

# UI ownership

- `internal/tui/components` contains abstract, reusable UI building blocks: inputs, selectors, form bodies, panels and dialogs. It exists to share control behavior, rendering and geometry across screens, not to collect every UI type.
- Keep screen-specific compositions, workflow messages and domain actions in `internal/tui/scenes/<scene>/widgets`. Use the name `widgets`, not `comps`. A publish form belongs to the publish scene even when it uses shared controls.
- Scenes own request lifetime and route messages to their widgets. Widgets compose shared components and own their draft, focus and scroll state. Shared components must not depend on scenes or application workflows.
- Reuse components through their public behavior. Do not expose or copy child implementation state to make a screen composition work.

# Layout and numeric values

- Put all numeric values that control behavior or presentation in descriptive named constants in the owning package. This applies throughout implementation code, including history capacities, payload limits, retry counts, timing, dimensions, padding and offsets. Do not leave numeric policy values inline.
- Define each shared value once and reuse it everywhere. For example, both editors must use the same named undo-history limit for the length check and stack trimming; never repeat a literal such as `100`. Include units in names where applicable and explain the purpose when it is not clear.
- Derive content sizes and coordinates from shared geometry and owning components. Subtract borders, padding and reserved scrollbar space once; use the same bounds for sizing, rendering and mouse input.
- Do not replace magic numbers with arbitrary names. Name the design constraint or invariant, and calculate derived values instead of duplicating arithmetic.

# Rendering

- Use the shared `core.Theme` for all UI colors and styles, including reusable components and notifications. Keep palette values as named constants in `internal/tui/core/theme.go`; never hardcode colors in renderers, scenes or widgets.
- Reuse existing theme colors before adding new ones. Define semantic styles and color choices in the theme so visual changes have one owner.
- Reuse shared rendering helpers wherever possible instead of assembling custom styled output in scenes or widgets. Extend the owning helper when shared behavior is missing.
- Define keyboard hints with `core.NewShortcut` and render them through `Theme.RenderHelp`. Keep displayed hints and input matching tied to the same bindings; do not hardcode key labels, separators or hint styles.

- All screens, including editors and full-screen forms, must use the shared hint style. Show useful action shortcuts; omit standard copy, paste and selection instructions.

# Keyboard shortcuts

- Use the `core.Key*` constants from `internal/tui/core/buttons.go` for key bindings and input matching. Add missing keys there; do not repeat raw key strings in scenes or widgets.
- Reuse shared shortcut helpers such as `core.Cancel()`, `core.Submit()` and `core.Esc()` when the action and bindings match. Add shared behavior to the owning helper instead of redefining it per screen.
- Use `core.NewShortcut` with key constants for screen-specific actions. Match input against the same bindings used by help.
- Give Ctrl action shortcuts optional Command aliases (`super`, with `meta` accepted where appropriate) in the shared binding helper. Preserve standard editor meanings and resolve conflicts explicitly; do not globally rewrite Command input to Ctrl.
- Keep Ctrl+C reserved for quitting, including when text is selected. Engine must show a top-line confirmation banner on the first press and quit only on a second consecutive press; other input cancels confirmation. Copy uses Command+C or Ctrl+Shift+C when the terminal forwards them distinctly; do not intercept Ctrl+C for clipboard actions. Legacy terminals can send Ctrl+Shift+C as Ctrl+C, so never promise that combination is safe without checking terminal support.
- Command and modified Enter shortcuts depend on what the terminal forwards. Keep a usable terminal fallback, and do not claim an alias works when the terminal consumes it. Use Bubble Tea's modifier order (`shift+super+z`, `shift+meta+z`) and test real `tea.KeyPressMsg` modifier combinations.
- Keep editor shortcut references in the shared editor component, derived from its active bindings. Show Command aliases once with readable labels and keep non-editing actions out of editor help.
- Keep payload editor help short: show everyday editing actions and omit advanced transformations and navigation. Put less discoverable actions first and show the reference in a help modal rather than filling the screen footer. Use a control shortcut for help so punctuation remains editable.
- Do not bind or advertise Command+A as editor select-all; use Ctrl+A.

# Scrollbars

- Every scrollbar must use the same generic shared renderer, including help windows and other modals. Border, clipping, reserved column, padding and colors belong to that renderer; never build a screen-specific scrollbar or wrap it in padding that changes its placement. Use `core.ScrollbarColumns` for the reserved width.
- Use the shared screen scrollbar approach: `Theme.RenderInsetScrollContainer` for bordered panels, and `Theme.RenderScrollbarViewport` for already clipped editor/form viewports. Keep the scrollbar in the reserved column immediately inside the right border, with consistent theme colors.
- Do not implement custom scrollbar drawing or placement in individual screens or widgets. Extend the shared renderer when behavior is missing; preserve cursor coordinates and keep scroll state in the owning model.

# Debouncing

- Use `core.Debouncer` from [internal/tui/core/debounce.go](internal/tui/core/debounce.go); reuse it instead of implementing timer generations in scenes or widgets.
- Keep one debouncer per independent action in its owning model. Initialize with `core.NewDebouncer()` or use the zero value.
- In `Update`, return the command from `Trigger(delay, callback)`. Capture inputs before creating the callback; return a typed message containing the supplied generation without reading or changing model state.
- Route that message to its owner and check `IsValid(generation)` in `Update` before acting. Keep selection checks and request cancellation in the owning model.
- Call `Cancel()` when pending work becomes irrelevant, such as when hiding a view, clearing selection or starting the request. This invalidates messages; existing timer commands still finish. Invalidate a consumed timer so duplicate delivery cannot repeat the action.

# Charts

- Use `components.ChartPanel` from [internal/tui/components/chart_panel.go](internal/tui/components/chart_panel.go) for chart views. Keep sizing, clipping, padding and scrollbar rules there; do not duplicate them in scenes.
- Pass the panel's inner size: subtract the outer border once in the parent layout. Use `ContentWidth()` for legends and accompanying text, and `NewChart()` / `DrawChart()` for plots. Never size a plot to the terminal or full panel width.
- Rebuild charts after `SetSize()` reports a change. Use `DrawChart()` to guard against missing plot space when axis labels fill a narrow or hidden panel.
- Render the whole chart view through `ChartPanel.Render()`. Content and scrollbar must stay inside uninterrupted borders, with the reserved side gaps intact. Width measurements must use terminal cells and preserve ANSI styles, not byte or rune counts.
- Keep data, focus and scroll state in the owning widget. Clamp scrolling with `ClampOffset()` in `Update` after content or size changes; `View` must not change scroll state.
- Test populated charts through scrolling, refreshes, narrow/hidden/restored sizes and long styled or wide-character text. Assert exact panel dimensions, intact borders, an interior scrollbar, and no chart cells in neighboring panels. String tests do not prove live terminal rendering; state when that has not been checked.

# Documentation

- Keep architecture documentation in CONTRIBUTING.md focused on high-level architecture: layer boundaries, state ownership, message flow and dependency direction.
- Update the architecture documentation only when those abstractions change. Keep feature mechanics, timing values, configuration fields and implementation examples in code or focused documentation.
- Keep documentation and change summaries concise; include only information needed to understand the design or outcome.

# Versioning

- Use Git tags as the source of truth for release versions. GoReleaser embeds the tag-derived version; do not hardcode a release number in source or configuration.
- CLI output and UI labels must use `version.Current`, resolved in `internal/version/version.go`. Without an injected version, use Go module or VCS build metadata, falling back to `dev` when unavailable.
- Use semantic tags with a `v` prefix, choosing patch for fixes, minor for features, and major for breaking changes. Do not create a tag or publish a release unless requested.

# Testing

- Every test and subtest must call `t.Parallel()`.

# Code explanations

- Explain non-obvious code with short, concrete comments. Give complex functions a compact overview of their purpose and approach.
- Comment on constraints, coordinate systems, state ownership, or reasons for unusual steps where they matter. Do not narrate obvious statements.
- Use plain language. Avoid filler, promotional phrasing, and generic AI-style commentary.
- Keep related code readable with normal spacing; do not compress logic just to reduce line count.
- Leave a blank line between function declarations.
- Inside functions, separate distinct logical steps with a blank line. Keep closely related statements together.
