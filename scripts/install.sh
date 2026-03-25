#!/bin/sh
set -e

echo "Installing syncopate..."

go install .

echo "Installed to $(go env GOPATH)/bin/syncopate"
echo ""
echo "Make sure $(go env GOPATH)/bin is in your PATH:"
echo "  export PATH=\"\$(go env GOPATH)/bin:\$PATH\""
