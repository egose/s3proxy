#!/usr/bin/env bash

semver_identifier='(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
semver_pattern="^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-${semver_identifier}(\.${semver_identifier})*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$"

semver_valid() {
  local LC_ALL=C
  [[ $1 =~ $semver_pattern ]]
}

semver_sort() (
  export LC_ALL=C
  export S3PROXY_SEMVER_PATTERN="$semver_pattern"
  awk '
    function numeric_key(value,    size) {
      size = value
      gsub(/./, "1", size)
      return size "0" value "!"
    }
    function precedence_key(version,    core, parts, count, key, i) {
      sub(/\+.*/, "", version)
      core = version
      sub(/-.*/, "", core)
      split(core, parts, ".")
      key = numeric_key(parts[1]) numeric_key(parts[2]) numeric_key(parts[3])
      if (version == core) return key "2"
      sub(/^[^-]*-/, "", version)
      count = split(version, parts, ".")
      key = key "1"
      for (i = 1; i <= count; i++) {
        if (parts[i] ~ /^[0-9]+$/) key = key "0" numeric_key(parts[i])
        else key = key "1" parts[i] "!"
      }
      return key
    }
    $0 ~ ENVIRON["S3PROXY_SEMVER_PATTERN"] {
      print precedence_key($0) "\t" $0
    }
  ' | sort -u | cut -f2
)
