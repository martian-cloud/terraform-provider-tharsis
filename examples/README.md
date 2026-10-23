# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or are testable even if some parts are not relevant for the documentation.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named resource page
* **resources/`full resource name`/import-by-string-id.tf** example file for the import section of the named resource page

Every importable resource should have an `import-by-string-id.tf` containing an
[`import` block](https://developer.hashicorp.com/terraform/language/import).
Without one, the generated page says nothing about import, which leaves users
unable to discover that the resource can be imported or to work out what
identifier to supply.

Resources are imported by ID or by TRN. Lead with the TRN where the resource has
a human-readable one, and mention the ID in a comment. Show a single `import`
block per file, because several blocks sharing one `to` address are invalid if
copied together.

## These examples only appear if the page template renders them

The `templates/` directory at the repository root replaces the generator's
built-in page template rather than adding to it. A template that omits a section
drops it from the generated page even when the corresponding example file exists
here, and because the generated output still matches the template, the
documentation check in CI passes without complaint.

So when adding a template for a new resource, carry over the whole structure
from an existing one, including the trailing block that renders
`import-by-string-id.tf`. Equally, adding an `import-by-string-id.tf` for a
resource whose template lacks that block will appear to do nothing.
