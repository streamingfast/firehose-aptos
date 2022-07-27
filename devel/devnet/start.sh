#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

clean=
sfnear="$ROOT/../fireaptos"

main() {
  pushd "$ROOT" &> /dev/null

  while getopts "hc" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  set -e

  if [[ $clean == "true" ]]; then
    rm -rf firehose-data &> /dev/null || true
  fi

  # Temporary just to ensure I have the right folder structure while testing stuff around,
  # should be removed once properly handled in firehose-aptos core code directly.
  mkdir -p firehose-data/extractor/data

  exec $sfnear -c $(basename $ROOT).yaml start "$@"
}

usage_error() {
  message="$1"
  exit_code="$2"

  echo "ERROR: $message"
  echo ""
  usage
  exit ${exit_code:-1}
}

usage() {
  echo "usage: start.sh [-c]"
  echo ""
  echo "Start $(basename $ROOT) environment which goals is to sync with Aptos Devnet"
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
}

main "$@"