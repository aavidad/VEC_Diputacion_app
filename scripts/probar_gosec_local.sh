#!/usr/bin/env bash
set -euo pipefail

raiz_gosec=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
temporal_gosec=$(mktemp -d "${TMPDIR:-/tmp}/vec-gosec-control.XXXXXX")
trap 'rm -rf -- "$temporal_gosec"' EXIT
cd "$temporal_gosec"
printf 'module vec-control-gosec\n\ngo 1.25.12\n' > go.mod

printf 'package control\nfunc Valor() int { return 1 }\n' > control.go
python3 "$raiz_gosec/scripts/gosec_local.py" . > limpio.json

cat > control.go <<'GO'
// Code generated for the security control. DO NOT EDIT.
package control
import "crypto/tls"
func Configuracion() *tls.Config {
    return &tls.Config{InsecureSkipVerify: true}
}
GO
if python3 "$raiz_gosec/scripts/gosec_local.py" . > inseguro.json; then
  printf 'GoSec aceptó TLS sin verificar.\n' >&2
  exit 1
else
  estado_gosec=$?
  [[ "$estado_gosec" -eq 1 ]] || exit "$estado_gosec"
fi
python3 - <<'PY'
import json
from pathlib import Path
datos = json.loads(Path('inseguro.json').read_text())
if not any('G402' in d['message'] for p in datos.values() for d in p['gosec']):
    raise SystemExit('Falta el diagnóstico G402 del control inseguro')
PY

printf 'package control\nfunc Valor() int { return desconocido }\n' > control.go
if python3 "$raiz_gosec/scripts/gosec_local.py" . > tipos.json 2> tipos.log; then
  printf 'GoSec aceptó una fuente que no puede analizar.\n' >&2
  exit 1
else
  estado_gosec=$?
  [[ "$estado_gosec" -eq 2 ]] || exit 1
fi
if python3 "$raiz_gosec/scripts/gosec_local.py" control.go > fichero.json 2> fichero.log; then
  printf 'GoSec aceptó un archivo suelto.\n' >&2
  exit 1
else
  estado_gosec=$?
  [[ "$estado_gosec" -eq 2 ]] || exit 1
fi
printf 'GoSec: fuente correcta, TLS inseguro y errores de carga comprobados.\n'
