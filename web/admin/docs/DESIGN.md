# Obelisk admin design system

## Layers

1. **Tokens** — `app/globals.css` (Kinetic / broadcast palette, typography utilities, `.field`, shadows)
2. **Primitives** — [Radix UI](https://www.radix-ui.com/primitives/docs/overview/introduction) via the [`radix-ui`](https://www.npmjs.com/package/radix-ui) package (tree-shakeable; subpaths like `radix-ui/dialog`)
3. **Components** — `components/ui/*` (styled wrappers; export via `components/ui/index.ts`)

We do **not** use Radix Themes. Radix supplies unstyled behavior; all color, radius, and type come from our tokens.

Unstyled namespaces are also available from `lib/radix-primitives.ts` when you need full control.

## Radix primitives (wrapped)

| Primitive | Wrapper module |
|-----------|----------------|
| Accessible Icon | `accessible-icon.tsx` |
| Accordion | `accordion.tsx` |
| Alert Dialog | `alert-dialog.tsx` |
| Aspect Ratio | `aspect-ratio.tsx` |
| Avatar | `avatar.tsx` |
| Checkbox | `checkbox.tsx` |
| Collapsible | `collapsible.tsx` |
| Context Menu | `context-menu.tsx` |
| Dialog | `dialog.tsx` |
| Direction | `direction.tsx` |
| Dropdown Menu | `dropdown-menu.tsx` |
| Form | `form.tsx` |
| Hover Card | `hover-card.tsx` |
| Label | `label.tsx` |
| Menubar | `menubar.tsx` |
| Navigation Menu | `navigation-menu.tsx` |
| One-Time Password Field (unstable) | `one-time-password-field.tsx` |
| Password Toggle Field (unstable) | `password-toggle-field.tsx` |
| Popover | `popover.tsx` |
| Portal | `portal.tsx` |
| Progress | `progress.tsx` |
| Radio Group | `radio-group.tsx` |
| Scroll Area | `scroll-area.tsx` |
| Select | `select.tsx` (Radix listbox; use `native-select.tsx` for simple `<select>`) |
| Separator | `separator.tsx` |
| Slider | `slider.tsx` |
| Slot | `button.tsx` (`asChild`) |
| Switch | `switch.tsx` |
| Tabs | `tabs.tsx` |
| Toast | `toast.tsx` |
| Toggle | `toggle.tsx` |
| Toggle Group | `toggle-group.tsx` |
| Toolbar | `toolbar.tsx` |
| Tooltip | `tooltip.tsx` |
| Visually Hidden | `visually-hidden.tsx` |

Shared menu/popover classes: `components/ui/radix-styles.ts`.

## Adding a component

1. Pick a [Radix primitive](https://www.radix-ui.com/primitives/docs/overview/introduction).
2. Create or extend `components/ui/<name>.tsx` — compose primitive parts + `cn()` + token classes only (no one-off hex).
3. Export from `components/ui/index.ts`.
4. Use in `app/` and `components/layout/`.

## Typography & color

See utilities in `globals.css`: `text-display`, `text-page-title`, `text-body`, `text-eyebrow`, semantic colors (`bg-broadcast`, `text-link`, …).

## Data & API

- **Redux Toolkit + RTK Query** — `lib/store/obelisk-api.ts` (`createApi`), `StoreProvider` in `app/layout.tsx`. Hooks exported from `lib/api.ts`; domain types in `lib/types.ts`.
- Cache tags: `Status`, `Connections`, `BackupTarget`. Status/connections poll every 3s via `statusPollOptions`.
- **Filtering** — `lib/filters/engine.ts` (`applyFilters`), surface state in `filters` slice (`lib/store/filters-slice.ts`). Services facets: `lib/filters/services-config.ts`; UI: `components/services/services-filters.tsx`. Header search writes Redux `query` for the services surface.
- **Services** — `@tanstack/react-table` + `@tanstack/react-virtual`; row helpers in `lib/service-rows.ts`.

## Interaction

- Every click target gets `cursor: pointer` (global rule in `globals.css`: buttons, `role=button|tab|menuitem`, links, summary, selects, checkboxes/radios); disabled controls get `not-allowed`. Don't set it per component — use real `<button>`/`<a>` or a `role`.
- Hover: neutral surfaces tint (`hover:bg-surface`); red/primary darken to `broadcast-hover`. Focus: 2px `ring`.
## Scrollbars

Global (in `globals.css`): 10px slim rounded thumb with transparent padding, transparent track, theme tokens `--scrollbar-thumb` / `--scrollbar-thumb-hover` (turns brand red while dragging). `scrollbar-gutter: stable` is set on `html` and every element, so the gutter is always reserved and content never shifts when a scrollbar appears or disappears. Firefox falls back to `scrollbar-width: thin` + `scrollbar-color`. `.scrollbar-none` opts out (chip rows, tab lists).
