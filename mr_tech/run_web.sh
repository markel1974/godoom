#!/bin/bash
unset GOROOT
# cp /Users/tinmr305/Development/go/1.24.4/go1.24.4/lib/wasm/wasm_exec.js html/

echo "=> removing old WASM..."
rm html/godoom.wasm
echo "=> Done..."

echo "=> copying index.html..."
cp renderers/open_gl/web/html/index.html html/index.html || { echo "failed to copy index.html"; exit 1; }
echo "=> Done..."

echo "=> compiling WASM..."
GOOS=js GOARCH=wasm go build -o html/godoom.wasm . || { echo "failed to build"; exit 1; }
echo "=> Done..."

echo "=> starting server on port 8080..."
cd html && python3 -m http.server 8080 -b 0.0.0.0
