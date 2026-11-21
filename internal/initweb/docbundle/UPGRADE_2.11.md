# Upgrading to AhdCode v2.11

Türkçe

The project owner approved the removal of the public `Templates` builtin in
v2.11. This is an intentional compatibility break. Imports of `Templates`
and references to `TemplateSet`, `TemplateData`, and `TemplateError` must
be migrated before upgrading applications that used the v2.10 API.

Use `.ahd` files for pages, layouts, and components. Pass typed values to
functions that assemble `HTMLNode` values with `Web.UI`. Have the layout
return a response with `Web.page`, and register custom 404/500 handlers
through the existing Web or HTTP APIs. Pass dynamic strings as text nodes
or through text-taking helpers so they are escaped during rendering.
A newly generated Basic app shows the page, layout, and component structure.
See [Web application structure](WEB.md).

Develop and package the compiled application with its public assets. Future
PWA and packaging work will build on that structure.

The previous API is documented in the [v2.10.0 release](RELEASE_2.10.0.md).
