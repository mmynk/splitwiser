# Splitwiser Design System

This document is the source of truth for Splitwiser's visual, motion, and voice identity. Every page composes the primitives defined here. If a page needs raw Tailwind for a button, card, input, or amount — that's a bug in the design system, not a license to write it inline.

## Principles

1. **Warm, not corporate.** Splitwiser is about sharing meals and trips with friends. The palette and type reflect that. No indigo, no Inter-everywhere, no `rounded-2xl shadow-lg`.
2. **Character, not flatness.** Modern web apps have over-pruned themselves into bank-grade greyness. We push back. Fonts have personality, motion has rhythm, copy has a voice, money has texture. *Character isn't chaos* — it's consistency with conviction.
3. **Fun, but principled.** Cute is allowed. Wit is allowed. Small celebratory moments (settling up, getting paid back) are encouraged. What's *not* allowed: cute that mocks the user's time, jokes that wear thin on the 50th read, or any "🎉" where a quiet "Settled." would do more work.
4. **Honest about money.** Amounts are the most important data on the screen. They get the most characterful treatment (variable serif, sign-aware color, tabular nums) and the most restraint everywhere else.
5. **Quiet motion.** Things rise 4px and fade in over 180ms. No bounces, no overshoots — except in deliberate celebration moments, where a single confident spring is permitted.
6. **Hairlines over shadows.** Resting cards are bordered, not shadowed. Shadow is reserved for true elevation (modal, dropdown).
7. **Compose, don't restyle.** If you reach for `class="bg-primary ..."` on a `<button>`, use `<Button>` instead.

---

## Color

We define every color twice — once for light, once for dark — and switch via a `data-theme="dark"` attribute on `<html>` (and via `prefers-color-scheme` by default). The token names are identical; only the values change.

### Light mode

| Token | Hex | Use |
|-------|-----|-----|
| `--color-primary` | `#B45C3A` | Primary actions, accents, focus ring |
| `--color-primary-hover` | `#9E4E30` | Hover state for primary |
| `--color-primary-foreground` | `#FFFFFF` | Text on primary |
| `--color-primary-soft` | `#F4E3D8` | Primary-tinted soft surface |
| `--color-surface` | `#FAF7F2` | Page background — warm off-white, slightly paper |
| `--color-surface-elevated` | `#FFFFFF` | Cards, modals, inputs |
| `--color-surface-sunken` | `#F2EDE5` | Subtle wells (tab groups, code) |
| `--color-text` | `#1A1714` | Primary text — warm ink, not pure black |
| `--color-text-muted` | `#6B635B` | Secondary text |
| `--color-text-subtle` | `#9A9087` | Tertiary text, placeholders |
| `--color-border` | `#E8E1D7` | Default hairline border |
| `--color-border-strong` | `#D4C9B8` | Emphasized border (focused input rest) |
| `--color-success` | `#0E7C7B` | Deep teal — money owed to you |
| `--color-success-soft` | `#E0F0EF` | Soft success surface |
| `--color-danger` | `#A93838` | Money you owe, destructive actions |
| `--color-danger-soft` | `#F6E2E0` | Soft danger surface |
| `--color-warning` | `#C58A22` | Amber, advisory only |

### Dark mode

Dark mode is not "light mode with inverted colors" — it's its own design. The clay primary becomes a touch lighter and warmer (a fireside glow, not a daytime brick). The surface is a deep warm brown, never pure black or cool slate.

| Token | Hex | Notes |
|-------|-----|-------|
| `--color-primary` | `#D4825E` | Lighter clay so it pops on dark surface |
| `--color-primary-hover` | `#E0916E` | |
| `--color-primary-foreground` | `#1A1714` | Dark text on light clay |
| `--color-primary-soft` | `#3A261B` | Primary-tinted dark surface |
| `--color-surface` | `#1A1612` | Warm dark brown, not slate |
| `--color-surface-elevated` | `#221D17` | One step lighter for cards |
| `--color-surface-sunken` | `#13100D` | One step darker for wells |
| `--color-text` | `#F2EDE5` | Warm off-white, not pure white |
| `--color-text-muted` | `#A99F92` | |
| `--color-text-subtle` | `#6E665C` | |
| `--color-border` | `#322A22` | |
| `--color-border-strong` | `#453B30` | |
| `--color-success` | `#3DB8B6` | Brightened teal for dark |
| `--color-success-soft` | `#1A2E2D` | |
| `--color-danger` | `#D66B6B` | Brightened red-clay for dark |
| `--color-danger-soft` | `#3A1F1F` | |
| `--color-warning` | `#E0A848` | |

**Why clay, not blue/green:** every "vibe-coded" app picks indigo. Splitwise picked green. Clay is rare in productivity apps, reads warm without being twee, and pairs with a deep teal success color (the natural complement of orange) that gives the "you're owed" state a satisfying weight.

---

## Typography

| Use | Font | Weight | Size | Line height |
|-----|------|--------|------|-------------|
| Display | Fraunces (opsz max, SOFT 50, WONK 1) | 600 | 2.25rem | 1.1 |
| h1 | Fraunces | 600 | 1.75rem | 1.15 |
| h2 | Fraunces | 600 | 1.375rem | 1.2 |
| h3 | Inter Tight | 600 | 1.125rem | 1.3 |
| body | Inter Tight | 400 | 0.9375rem | 1.55 |
| small | Inter Tight | 400 | 0.8125rem | 1.45 |
| label | Inter Tight | 500 | 0.8125rem | 1.4 |
| **money** | **Fraunces** (tabular, SOFT 30) | **500** | inherits | inherits |

**Why this pairing:** Fraunces is a variable serif with optical-size, SOFT, and WONK axes. At display sizes with WONK on, it gets visibly *idiosyncratic* — flared terminals, a tilted `g`, a swung `R`. At body weights it tames itself. That single font family gives us editorial character without needing a second display face. Inter Tight is tighter than Inter, which makes dense data tables read better. The serif/sans split signals two things: amounts and headers are "the story," everything else is "the chrome."

**Fonts** are loaded from Google Fonts via `<link>` in `index.html`. We pull the full variable axes for Fraunces (`opsz`, `wght`, `SOFT`, `WONK`).

---

## Spacing

4px base unit. Tailwind's default spacing scale is fine (`p-1`, `p-2`, ...) — we use it.

- Page gutter: `px-5` mobile, `px-8` desktop
- Card padding: `p-5`
- Stack gap: `gap-3` tight, `gap-5` default, `gap-8` section

Density target: comfortable middle. Not Linear-dense, not Stripe-marketing-airy. A bill list row should be ~52px tall. Bills and balances are dense (~40px rows); marketing-y surfaces (login, empty states) are airy.

---

## Radii

| Token | Value | Use |
|-------|-------|-----|
| `--radius-input` | `6px` | Inputs, small buttons |
| `--radius-card` | `10px` | Cards, modals (slightly softer than 8px to land warmer) |
| `--radius-pill` | `9999px` | Avatars, badges, status chips |

Single, conservative radius story. `rounded-2xl` (16px) is the cliché — we don't use it.

---

## Borders & shadows

Default treatment is a 1px hairline border in `--color-border`. **Cards do not have shadows at rest.** This is the single biggest visual differentiator from generic Tailwind apps.

Shadow is reserved for true elevation:

| Token | Value | Use |
|-------|-------|-----|
| `--shadow-pop` | `0 1px 2px rgba(26,23,20,0.04), 0 4px 12px rgba(180,92,58,0.10)` | Dropdowns, popovers |
| `--shadow-modal` | `0 10px 40px rgba(26,23,20,0.18)` | Modals |

The pop shadow has a warm primary tint — shadows aren't grey, they're tinted toward the primary. In dark mode, shadows tint deeper rather than lighter (we don't invert them to glow).

---

## Motion

One easing, one duration, applied everywhere by default:

```ts
export const ease = [0.32, 0.72, 0, 1];  // confident ease-out
export const dur = 180;                   // ms — default
export const durFast = 120;               // ms — route transitions, small flips
export const durCelebrate = 420;          // ms — settle-up, paid-back moments
```

Enter: fade + 4px rise. Exit: fade + 4px fall. Lists: `animate:flip` at `durFast`. **No springs, no bounces, no overshoots** — *except* in deliberate celebration moments (settling up a debt, accepting a friend request), where a single confident spring `[0.34, 1.56, 0.64, 1]` over `durCelebrate` is permitted. One reward per real action; never decorative.

---

## Voice

Plainspoken, warm, lightly witty. Never sarcastic, never cloying. Sentence case (not Title Case). Exclamation points are rationed — saved for genuine moments. No emoji in copy (we get our texture from type and motion, not from 🎉). Slightly *opinionated* copy is encouraged — Splitwiser has a perspective on splitting bills (everyone should pay their share; settling up should feel good).

### Tone calibration

| Tone | Use |
|------|-----|
| Plain | Errors, confirmations, destructive actions |
| Warm | Empty states, onboarding, first-time hints |
| Witty | Celebrations, easter eggs, the occasional empty state |

### Sample microcopy (locked references)

- Empty bills list: **"No bills yet. The first one's always the awkward one — just add it."**
- Empty balances when settled: **"All square. Nice."**
- Empty balances when never had any: **"Nothing to settle. Yet."**
- Empty friends: **"You haven't added anyone. Search above for someone you split with."**
- Network error toast: **"Couldn't reach the server. Check your connection and try again."**
- Validation error (negative amount): **"Amounts have to be positive. Subtract instead?"**
- 404 page: **"This page got lost. Probably owes you for lunch."** (followed by a single link home)
- Settled celebration: **"Settled."** (one word, subtle checkmark animation; the *motion* carries the joy, not the copy)
- Just got paid back: **"Paid back. Spend it well."**
- Friend request sent: **"Request sent. Ball's in their court."**
- Delete bill confirmation: **"Delete this bill? Can't be undone."** (plain, no hedging)

### Anti-patterns in voice

- "Oops! Something went wrong 😅" — patronizing
- "You're all caught up! Way to go! 🎉" — exhausting
- "Hmm, we couldn't find that…" — passive
- "Are you sure you really want to delete this? This action cannot be undone." — pleading
- Any sentence over ~14 words in UI copy

---

## Icons

Lucide Svelte, **stroke width 1.75** (default 2 is too heavy against a serif). Sizes:

- 16px — in dense UI (buttons, tabs, table rows)
- 18px — standalone affordances
- 20px — section headers
- Never larger inline

Color: inherits `currentColor` always — no colored icons except status (success/danger/warning).

---

## Numeric (money) treatment

This is the most important data on the screen. Rules:

1. **Always tabular numerals** (`font-variant-numeric: tabular-nums`).
2. **Font is Fraunces** at weight 500 (slightly heavier than body to anchor the eye).
3. **Sign-aware color:**
   - Positive (owed to you): `--color-success` (teal)
   - Negative (you owe): `--color-danger` (red-clay)
   - Zero: `--color-text-muted`
4. **Format:** `$1,234.56`. No abbreviations under $10,000 — the precision matters. Above $10,000, `$12.3k` is acceptable in tight UI.
5. **Sign rendering:** show the literal `+` for positive owed-to-you amounts in summary contexts. Removes ambiguity.
6. **The `<Amount>` primitive** is the only correct way to render money. It owns rules 1–5.

In Session 6, money that *changes* (after a settle-up, after adding a bill) animates by digit (a slot-machine-style roll) — but only on real change events, never on initial render.

---

## Primitives

Source: `frontend/src/lib/components/ui/`.

| Component | Purpose |
|-----------|---------|
| `Button.svelte` | Variants: `primary`, `secondary`, `ghost`, `danger`. Sizes: `sm`, `md`, `lg`. Loading state. |
| `IconButton.svelte` | Square icon-only button. Same variants and sizes. |
| `Input.svelte` | Text input with label, error, prefix/suffix slots. |
| `Card.svelte` | Hairline-bordered surface. `elevated` prop for true elevation. |
| `Amount.svelte` | Money formatter. Props: `value`, `signed` (color by sign), `showPlus`, `currency`. |

**Rule:** if a page needs to style a button, card, input, or amount and the primitive doesn't support it — extend the primitive, don't fall back to raw Tailwind.

---

## Reference: `#/_styles`

`StyleGuide.svelte` at `#/_styles` renders every primitive in every state, in both light and dark. Use it as the visual regression reference. If something looks off in a real page, compare to the style guide first.

---

## Anti-patterns (explicit avoidlist)

- `bg-blue-*`, `bg-indigo-*`, `bg-violet-*` anywhere
- `rounded-2xl` (use `rounded-card` = 10px)
- `shadow-lg` on resting cards
- Inter as the only typeface
- Generic top-right toast (we use bottom; styled with hairline border, not shadow)
- Emoji in microcopy
- Title Case headings
- Spinner for skeleton states (use skeleton blocks)
- `alert()` — use the toast store
- Apologetic or pleading copy ("Oops!", "Are you sure you really…")
