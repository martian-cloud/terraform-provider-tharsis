#!/usr/bin/env bash
#
# Merges per-platform CycloneDX SBOMs into a single deduplicated SBOM.
# Aligned with the shared merge-sboms template for NIS2 compliance.
#
# Required env vars:
#   VERSION - git ref (e.g. refs/tags/v0.16.2)
#
# Optional env vars:
#   PRODUCT_CODE    - 3-char MSF product code (default: NOP)
#   PRODUCT_TENANCY - tenancy value (default: mt)
#   CPE             - override CPE (auto-generated if empty)

set -euo pipefail

PRODUCT_CODE="${PRODUCT_CODE:-NOP}"
PRODUCT_TENANCY="${PRODUCT_TENANCY:-mt}"
PRODUCT_VERSION="${VERSION#refs/tags/}"

# Install the CycloneDX CLI
curl -sSL https://github.com/CycloneDX/cyclonedx-cli/releases/download/v0.27.1/cyclonedx-linux-x64 -o /usr/local/bin/cyclonedx
chmod +x /usr/local/bin/cyclonedx

# Set args for the CycloneDX CLI
ARGS="--input-format autodetect --output-format autodetect --version $VERSION --hierarchical --name terraform-provider-tharsis"

echo -e "\e[1;32m$ find . -name \"gl-sbom-*.cdx.json\" -exec cyclonedx merge --output-file gl-sbom-all.cdx.json $ARGS --input-files \"{}\" +\e[0m"

# Run the CycloneDX CLI to merge the SBOMs.
find . -name "gl-sbom-*.cdx.json" -exec /usr/local/bin/cyclonedx merge --output-file gl-sbom-all.cdx.json $ARGS --input-files "{}" +

# Get Licenses from Individual SBOMs (deduplicated)
LICENSES=$(jq -c -n '[ inputs.metadata.component.licenses ] | unique | [ .[] | select(. != null) | .[] ] | if . == [] then [ {"license":{"name":"declared license of example","text":{"content":"Proprietary License","contentType":"text/plain"}}} ] else . end' gl-sbom-*.cdx.json)

# Remove individual SBOMs, keep only the merged file
find . -maxdepth 1 -name "gl-sbom-*.cdx.json" ! -name "gl-sbom-all.cdx.json" -delete

# Generate CPE if not provided
if [ -z "${CPE:-}" ]; then
  CPE="cpe:2.3:a:infor:${PRODUCT_CODE}:${PRODUCT_VERSION}:-:*:*:${PRODUCT_TENANCY}:*:*"
fi

# JQ metadata
JQ_VERSION=$(jq --version)

jq -c --arg manufacture_name "Infor" \
    --arg supplier_name "Infor" \
    --arg cpe "$CPE" \
    --arg product_code "$PRODUCT_CODE" \
    --arg product_version "$PRODUCT_VERSION" \
    --arg product_tenancy "$PRODUCT_TENANCY" \
    --argjson licenses "$LICENSES" \
    --argjson sbom_version "-1" \
    --arg jq_version "$JQ_VERSION" \
  '.version = if $sbom_version <= 0 then .version + 1 else $sbom_version end |
  .metadata.manufacture.name = $manufacture_name |
  .metadata.supplier.name = $supplier_name |
  .metadata.component.supplier.name = $supplier_name |
  .metadata.component |= if has("licenses") then . else .licenses = $licenses end |
  .metadata.component.purl = "pkg:generic/" + .metadata.component.name + "@" + .metadata.component.version |
  .metadata.component.cpe = $cpe |
  .metadata.component.properties = ([(.metadata.component.properties // [])[] | select(.name | IN("productCode","productVersion","productTenancy") | not)] + [
    {name: "productCode", value: $product_code},
    {name: "productVersion", value: $product_version},
    {name: "productTenancy", value: $product_tenancy}
  ]) |
  .components |= (group_by(.type + .name + .version) | map(.[0])) |
  .components |= map(if .components then .components |= (group_by(.type + .name + .version) | map(.[0])) else . end) |
  .dependencies |= (group_by(.ref) | map(.[0])) |
  .dependencies |= map(if .dependsOn then .dependsOn |= unique else . end) |
  .metadata |= if has("timestamp") then . else .timestamp = (now | strflocaltime("%Y-%m-%dT%H:%M:%SZ")) end |
  .metadata.tools += [{"vendor":"jq","name":"jq","version":$jq_version}]' gl-sbom-all.cdx.json > gl-sbom-all.cdx.json.tmp

mv gl-sbom-all.cdx.json.tmp gl-sbom-all.cdx.json
