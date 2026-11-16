# Obelisk admin design system

## Layers

1. **Tokens** — `app/globals.css` (Pulse palette, typography utilities, `.field`, shadows)
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

## Visual language (Pulse)

- **Canvas:** cool grey `--background` (#ebebed) with fixed ambient gradients on `body` (mint top-right, peach bottom-right, lime bottom-left). Don't paint `bg-background` on page wrappers — it hides the gradients.
- **Surfaces:** white `Card` (`rounded-2xl` = 28px, `shadow-rb-2`), inset `StatTile` (`bg-muted`, `rounded-xl`), and the forest-green `FeatureCard` (`bg-feature`, white text, `bg-white/10` rows) for the one hero control card per page.
- **Accent:** lime `bg-broadcast` is a *fill only* — text on it is `text-broadcast-foreground`; never use lime as text on light surfaces. Pale lime `bg-broadcast-tint` marks the active row/highlight. Errors use `text-destructive` / `bg-destructive-tint`.
- **Active state:** solid black pill (`bg-ink text-ink-foreground`) for nav and tabs; lime chip for toggle-group/toggle "on"; black-thumb switch on lime track.
- **Shape:** everything interactive is a pill (`rounded-full`, 40px tall); popovers/dialogs 28px/20px.
- **Type:** Outfit; display/stat numbers are weight 300 with tight tracking; body 400. Hero headings are two-tone (ink, then `text-muted-foreground`).

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
