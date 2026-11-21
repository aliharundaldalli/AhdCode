# Upgrading to AhdCode v2.11

Türkçe

The public `Templates` builtin was removed in AhdCode v2.11.0. Before
upgrading applications that used the v2.10 `Templates` API, migrate
`Templates` imports and references to `TemplateSet`, `TemplateData`, and
`TemplateError`.

Use `.ahd` files for pages, layouts, and components. Pass typed values to
functions that assemble `HTMLNode` values with `Web.UI`. Have the layout
return a response with `Web.page`, and register custom 404/500 handlers
through the existing Web or HTTP APIs. Pass dynamic strings as text nodes
or through text-taking helpers so they are escaped during rendering.
A newly generated Basic app shows the page, layout, and component structure.
See [Web application structure](WEB.md).

Package the compiled application together with its public assets and optional
PWA files using `ahdcode package app.ahd --web` (and `--pwa` when PWA is
configured).

The previous API is documented in the [v2.10.0 release](RELEASE_2.10.0.md).
