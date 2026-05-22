# Landing and Docs Redesign

Date: 2026-05-21

## Decision

Rebuild the public site around layout 1: an operational control-plane landing page for an open-source Go + Redis load balancer. The current terminal/hacker aesthetic will not be reused as a reference.

## Landing Page

The landing page will be custom Astro, focused on an operational control-plane style:

- dark technical background with a hard control-room grid
- live routing board, telemetry blocks, and Redis state panels
- green, cyan, amber, and rose status accents
- concise product positioning
- real API examples and request flow
- clear calls to documentation and GitHub

The page should explain the product in the first viewport: service registry, heartbeat-based load reporting, and HTTP 302 redirect routing.

## Documentation

Docs will move from hand-built Astro pages to Astro Starlight at `/docs`.

Recommended integrations:

- `@astrojs/starlight` for the documentation shell
- `@astrojs/mdx` for component-capable docs content
- `starlight-openapi` for API reference from an OpenAPI schema
- `starlight-links-validator` for broken-link checks
- `@astrojs/sitemap` for static sitemap generation

The existing Markdown docs content will be migrated into Starlight content collections.

## Scope

In scope:

- replace the landing page visual system
- add Starlight documentation
- migrate existing getting started, architecture, deployment, and API content
- add an OpenAPI document for current endpoints
- verify the Astro build

Out of scope:

- changing the Go service behavior
- adding runtime analytics
- changing deployment infrastructure beyond static site build needs
