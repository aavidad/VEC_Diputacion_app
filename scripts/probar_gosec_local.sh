#!/usr/bin/env bash
set -euo pipefail

raiz_gosec=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
temporal_gosec=$(mktemp -d "${TMPDIR:-/tmp}/vec-gosec-control.XXXXXX")
trap 'rm -rf -- "$temporal_gosec"' EXIT
cd "$temporal_gosec"
printf 'module vec-control-gosec\n\ngo 1.25.12\n' > go.mod

python3 - "$raiz_gosec/scripts/gosec_local.py" "$temporal_gosec" <<'PY'
import importlib.util
import os
from pathlib import Path
import sys
spec = importlib.util.spec_from_file_location('gosec', sys.argv[1])
modulo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(modulo)
go = Path(modulo.seleccionar_go())
os.environ['PATH'] = str(go.parent) + os.pathsep + os.environ['PATH']
os.environ['GOMODCACHE'] = str(Path(sys.argv[2]) / 'cache-vacia')
if not Path(modulo.seleccionar_go()).samefile(go):
    raise SystemExit('Go del PATH exacto no funciona con caché de módulos vacía')
PY

printf 'package control\nfunc Valor() int { return 1 }\n' > control.go
python3 "$raiz_gosec/scripts/gosec_local.py" . > limpio.json

mkdir vacio
if python3 "$raiz_gosec/scripts/gosec_local.py" . ./vacio/... > parcial.json 2> parcial.log; then
  printf 'GoSec aceptó omitir un patrón sin paquetes.\n' >&2
  exit 1
else
  estado_gosec=$?
  [[ "$estado_gosec" -eq 2 ]] || exit 1
fi

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
printf 'GoSec: herramienta, cobertura, fuente correcta, TLS inseguro y errores de carga comprobados.\n'
