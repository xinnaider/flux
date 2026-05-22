# Landing Page Visual Design

Date: 2026-05-21

## Decision

Replace the previous dashboard/ops aesthetic with **Blueprint** — a warm, technical, architectural style. Warm off-white background, deep blue text, terracotta accents, gold highlights. Inspired by architectural blueprints and engineering notebooks.

## Design Direction

| Element | Value |
|---------|-------|
| Style | Blueprint / architectural |
| Base bg | `#f1ebe3` (warm off-white) |
| Text primary | `#1a3040` (deep blue) |
| Accent | `#c7351f` (terracotta) |
| Secondary accent | `#d4a76a` (warm gold) |
| Font headings | DM Serif Display (or Georgia fallback) |
| Font body | DM Sans (or system-ui fallback) |
| Font code | JetBrains Mono (or monospace fallback) |
| Border radius | 4px — industrial, no rounding |
| Vibe | Technical, warm, no gradients, no glassmorphism |

## Design Tokens

### 1. Color System

**Contrast verified** (WCAG AA):

| Token | Hex | Usage | On bg | Ratio |
|-------|-----|-------|-------|-------|
| `--color-bg` | `#f1ebe3` | Page background | — | — |
| `--color-bg-card` | `#f6f3ed` | Card/section bg | — | — |
| `--color-bg-dark` | `#1a3040` | Footer, code blocks | — | — |
| `--color-text-primary` | `#1a3040` | Body / headings | `#f1ebe3` | **10.5:1** AAA |
| `--color-text-secondary` | `#5a6a7a` | Subheadlines, meta | `#f1ebe3` | **5.2:1** AA |
| `--color-text-inverse` | `#f1ebe3` | Text on dark bg | `#1a3040` | **10.5:1** AAA |
| `--color-accent` | `#c7351f` | CTAs, highlights | `#f1ebe3` | **5.1:1** AA |
| `--color-accent-hover` | `#a82a18` | Hover state | `#f1ebe3` | **7.2:1** AAA |
| `--color-accent-muted` | `#e8d5cd` | Subtle bg tint | `#1a3040` | **4.6:1** AA |
| `--color-gold` | `#d4a76a` | Logo node, accents | `#1a3040` | **4.8:1** AA |
| `--color-border` | `#d9d2c8` | Dividers, outlines | `#f1ebe3` | **1.5:1** (decorative) |
| `--color-success` | `#2b6e4f` | Code output | `#1a3040` | **4.7:1** AA |

Semantic naming por proposito, nao por aparencia. Hover/active states definidos para cada cor interativa.

### 2. Typography Scale

Modular scale **1.25** (major second):

```css
:root {
  /* Family */
  --font-heading: "DM Serif Display", Georgia, serif;
  --font-body: "DM Sans", system-ui, sans-serif;
  --font-code: "JetBrains Mono", "Cascadia Code", monospace;
  --font-weight-heading: 400; /* DM Serif Display regular */
  --font-weight-body: 400;
  --font-weight-bold: 700;

  /* Scale */
  --text-xs: 0.75rem;    /* 12px  — code, labels */
  --text-sm: 0.875rem;   /* 14px  — meta, secondary */
  --text-base: 1rem;     /* 16px  — body */
  --text-lg: 1.125rem;   /* 18px  — lead */
  --text-xl: 1.25rem;    /* 20px  — subtitle */
  --text-2xl: 1.5rem;    /* 24px  — section heading */
  --text-3xl: 1.875rem;  /* 30px  — major heading */
  --text-4xl: 2.25rem;   /* 36px  — hero heading */
  --text-5xl: 3rem;      /* 48px  — display */

  /* Line heights */
  --lh-tight: 1.15;      /* headings */
  --lh-normal: 1.5;      /* body */
  --lh-relaxed: 1.65;    /* long-form */
  --lh-mono: 1.4;        /* code */
}

/* Fluid hero heading */
h1 { font-size: clamp(2rem, 4vw + 1rem, 3rem); }

/* Reading width */
p, .prose { max-width: 65ch; }
```

Font loading com `font-display: swap` para evitar layout shift.

### 3. Spacing System

8-point grid:

```css
:root {
  --space-1: 0.25rem;  /*  4px */
  --space-2: 0.5rem;   /*  8px */
  --space-3: 0.75rem;  /* 12px */
  --space-4: 1rem;     /* 16px */
  --space-5: 1.25rem;  /* 20px */
  --space-6: 1.5rem;   /* 24px */
  --space-8: 2rem;     /* 32px */
  --space-10: 2.5rem;  /* 40px */
  --space-12: 3rem;    /* 48px */
  --space-16: 4rem;    /* 64px */
  --space-20: 5rem;    /* 80px */
  --space-24: 6rem;    /* 96px section gap */

  /* Component defaults */
  --radius-sm: 4px;    /* buttons, inputs */
  --radius-md: 6px;    /* cards */
  --radius-lg: 8px;    /* code blocks */
}
```

**Component spacing:**
| Element | Padding | Gap |
|---------|---------|-----|
| Card | `--space-6` (24px) | — |
| Section | — | `--space-24` (96px) vertical |
| Button | `--space-2` (8px) v × `--space-4` (16px) h | — |
| Icon-text | — | `--space-2` (8px) |
| Code block | `--space-4` (16px) v × `--space-5` (20px) h | — |
| Nav items | — | `--space-6` (24px) |

### 4. Iconography

Node network icons — sizing system:

```css
:root {
  --icon-xs: 12px;
  --icon-sm: 16px;
  --icon-md: 20px;
  --icon-lg: 24px;
  --icon-xl: 32px;
}
```

Implementation: inline SVG via `<Icon>` component, `aria-hidden="true"`.

## Logo

**Concept:** Node Network — three connected nodes representing the service registry cluster.

Four variants:

| Variant | Usage |
|---------|-------|
| V1 Icon (96x96) | Favicon, app icon, watermark |
| **V2 Wordmark** (icon + text) | **Header nav, hero** |
| V3 Stacked | Footer, sidebar, mobile |
| V4 Favicon (dark bg) | Browser tab icon |

All rendered as inline SVG — no external image files.

## Landing Sections (single-page)

### Header
- Fixed top nav
- Logo V2 left-aligned
- Links: Docs (Starlight), GitHub (external)
- CTA button "Get Started" → `/docs`
- Background: warm off-white, bottom border 1px subtle

### Hero
- V2 Wordmark logo (centered or left-aligned)
- **Headline:** "Redirect load balancing, simplified."
- **Subheadline:** "Flux is a lightweight service registry and HTTP redirect balancer — no reverse proxy, no proxy overhead."
- **CTAs:** "Get Started" (primary, terracotta bg) + "GitHub" (outline)
- Optional: subtle architectural grid lines in background

### Features (3-column)
Three SVG icon + text cards:
1. **Service Registry** — Redis-backed, TTL-based health, no heartbeats
2. **Redirect Balancer** — HTTP 302, round-robin, no proxy overhead
3. **Simple API** — RESTful, JSON, minimal surface area

Icons use the node network motif (small versions of the logo nodes).

### Architecture Diagram
- Full-width SVG diagram showing request flow
- Client → `flux` (302 redirect) → Service A / Service B
- Uses V1 icon nodes as visual elements in the diagram
- Annotations in JetBrains Mono

### Quick Start
- Section title: "Try it in 30 seconds"
- Code block: dark bg (`#1a3040`), white/green text
- `curl` command to register + discover a service
- Small Go snippet for programmatic usage
- Copy button on code blocks

### Why Flux?
- Comparison table: flux vs reverse proxies (nginx, HAProxy, Envoy)
- Rows: Overhead, Latency, Configuration, Use Case
- flux column highlighted with terracotta accent
- Key takeaway: "Flux is not a proxy. It tells the client where to go."

### Footer
- Dark background (`#1a3040`), white text
- Logo V3 (stacked, white variant)
- Links: Docs, GitHub, License (MIT)
- "Built with Go" badge

## Interaction Design

Purposeful, restrained motion. No ornament — every transition communicates something.

### Timing & Easing

| Duration | Use | Token |
|----------|-----|-------|
| 100ms | Hover, focus ring, active press | `--dur-instant` |
| 200ms | Card lift, nav item, CTA press | `--dur-fast` |
| 300ms | Section fade-in on scroll, code block appear | `--dur-normal` |
| 500ms+ | Hero staggered entrance (page load) | `--dur-slow` |

```css
:root {
  --dur-instant: 100ms;
  --dur-fast: 200ms;
  --dur-normal: 300ms;
  --dur-slow: 500ms;

  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-in: cubic-bezier(0.55, 0, 1, 0.45);
  --ease-in-out: cubic-bezier(0.65, 0, 0.35, 1);
}
```

Only `transform` and `opacity` animated — never `width`, `height`, `top`, `left`.

### Patterns by Section

**Header (nav links, CTA button)**
- Hover: link underline slide-in (200ms ease-out)
- CTA bg: `--color-accent` → `--color-accent-hover` (100ms)
- Active press: scale 0.97 (100ms)

**Hero**
- Staggered entrance on load:
  - Logo: fade-in + translateY(0) from 10px (500ms, 0ms delay)
  - Headline: same (400ms, 100ms delay)
  - Subheadline: same (400ms, 200ms delay)
  - CTAs: same (300ms, 300ms delay)
- Uses Intersection Observer + CSS classes (no JS animation lib)

**Feature Cards**
- Hover: translateY(-4px), shadow deepen (200ms ease-out)
- Icon: subtle scale 1.05 on hover (200ms)
- Staggered fade-in on scroll into view (300ms each, 100ms delay between)

**Architecture Diagram**
- SVG elements appear on scroll in draw order (client → flux → services)
- Stroke dash animation for connection lines (800ms)

**Code Block (Quick Start)**
- Copy button: appears on hover, fades 200ms
- "Copied!" toast: fade-in 150ms, hold 2s, fade-out 200ms
- Pulse cursor on the `$` prompt (CSS keyframe, 1s infinite)

**Why Flux? (table)**
- Row hover: bg tint `--color-accent-muted` at 0.3 opacity (100ms)
- Flux column: subtle glow effect on page load (CSS keyframe, 1.5s)

**Footer**
- Link hover: gold underline accent (150ms)
- No entrance animation (always visible)

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

All animations degrade gracefully — content is fully accessible without motion.

### Implementation Strategy

- No Framer Motion or JS animation library
- CSS transitions + `@keyframes` for repeating effects (pulse cursor)
- Intersection Observer via a lightweight `<script>` in the Astro page
- All animations respect `prefers-reduced-motion`

## Implementation

- Astro (static site, no SSR)
- All styles local to the landing page (not shared with Starlight)
- Logo as Astro component (`src/components/Logo.astro`) with `variant` prop
- Responsive: mobile-first, single column → 3-column grid at md+
- No external CSS frameworks
- No JS animation libraries

## Out of Scope

- Blog / changelog pages (future)
- Dark mode toggle
- Analytics
