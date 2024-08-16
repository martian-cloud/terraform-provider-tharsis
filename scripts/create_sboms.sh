#!/usr/bin/env bash

# JQ metadata
JQ_VERSION=$(jq --version)

# Our mapping file which contains the suppliers.
MAPPING_FILE="sbom-component-mapping.json"

# Set the args for CycloneDX.
ARGS="-assert-licenses -licenses -std -json -output-version 1.5"

OUTPUT_FILE="gl-sbom-go-go.$(echo $GO_OS:$GO_ARCH:$CGO_ENABLED | tr '/ ' '_').cdx.json"

echo -e "\e[1;32m$ cyclonedx-gomod app $ARGS -output $OUTPUT_FILE .\e[0m"

# Create SBOMs for each platform.
cyclonedx-gomod app $ARGS -output $OUTPUT_FILE .

jq -c --arg manufacture_name "Infor" \
   --arg supplier_name "Infor" \
   --argjson sbom_version "-1" \
   --arg jq_version "$JQ_VERSION" \
   'input as $mapping |
   .components[] |= if $mapping[.name] then . * $mapping[.name] else . end |
   .version = if $sbom_version <= 0 then .version + 1 else $sbom_version end |
   .metadata.manufacture.name = $manufacture_name |
   .metadata.supplier.name = $supplier_name |
   .metadata.component.supplier.name = $supplier_name |
   .metadata |= if has("timestamp") then . else .timestamp = (now | strflocaltime("%Y-%m-%dT%H:%M:%SZ")) end |
   .metadata.tools += [{"vendor":"jq","name":"jq","version":$jq_version}]' $OUTPUT_FILE $MAPPING_FILE > $OUTPUT_FILE.tmp

mv $OUTPUT_FILE.tmp $OUTPUT_FILE
