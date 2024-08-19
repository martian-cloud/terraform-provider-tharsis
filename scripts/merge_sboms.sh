#!/usr/bin/env bash

# Install the CLI
curl -sSL https://github.com/CycloneDX/cyclonedx-cli/releases/download/v0.25.0/cyclonedx-linux-x64 -o /usr/local/bin/cyclonedx

chmod +x /usr/local/bin/cyclonedx

# Set args for the CycloneDX cli
ARGS="--input-format autodetect --output-format autodetect --version $VERSION --hierarchical --name terraform-provider-tharsis"

echo -e "\e[1;32m$ find . -name "gl-sbom-*.cdx.json" -exec cyclonedx merge --output-file gl-sbom-all.cdx.json $ARGS --input-files "{}" +\e[0m"

# Run the CycloneDX CLI to merge the SBOM.
find . -name "gl-sbom-*.cdx.json" -exec /usr/local/bin/cyclonedx merge --output-file gl-sbom-all.cdx.json $ARGS --input-files "{}" +

# Get Licenses from Individual SBOMs (deduplicated), if not present in the merged SBOM then add them
LICENSES=`jq -c -n '[ inputs.metadata.component.licenses ] | unique | [ .[] | select(. != null) | .[] ] | if . == [] then [ {"license":{"name":"declared license of 'example'","text":{"content":"Proprietary License","contentType":"text/plain"}}} ] else . end' gl-sbom-*.cdx.json`

# JQ metadata
JQ_VERSION=$(jq --version)

jq -c --arg manufacture_name "Infor" \
    --arg supplier_name "Infor" \
    --argjson licenses "$LICENSES" \
    --argjson sbom_version "-1" \
    --arg jq_version "$JQ_VERSION" \
    '.version = if $sbom_version <= 0 then .version + 1 else $sbom_version end |
    .metadata.manufacture.name = $manufacture_name |
    .metadata.supplier.name = $supplier_name |
    .metadata.component.supplier.name = $supplier_name |
    .metadata.component |= if has("licenses") then . else .licenses = $licenses end |
    .metadata.component.purl = "pkg:generic/" + .metadata.component.name + "@" + .metadata.component.version |
    .metadata |= if has("timestamp") then . else .timestamp = (now | strflocaltime("%Y-%m-%dT%H:%M:%SZ")) end |
    .metadata.tools += [{"vendor":"jq","name":"jq","version":$jq_version}]' gl-sbom-all.cdx.json > gl-sbom-all.cdx.json.tmp

# Create the merged SBOM file.
mv gl-sbom-all.cdx.json.tmp gl-sbom-all.cdx.json
