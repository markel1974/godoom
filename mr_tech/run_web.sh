#!/bin/bash
unset GOROOT
# cp /Users/tinmr305/Development/go/1.24.4/go1.24.4/lib/wasm/wasm_exec.js html/
echo "=> Compilazione dell'engine in WebAssembly..."
# Metti qui il path corretto al tuo main se non è la root
GOOS=js GOARCH=wasm go build -o html/godoom.wasm .

echo "=> Avvio del server locale sulla porta 8080..."
echo "=> Vai su http://localhost:8080 per vedere l'engine in azione!"
cd html && python3 -m http.server 8080 -b 0.0.0.0
