#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

imagen_produccion="${VEC_IMAGEN_PRODUCCION_PRUEBA:-vec-diputacion-granada:prueba-contenido-produccion}"
imagen_presentacion="${VEC_IMAGEN_PRESENTACION_PRUEBA:-vec-diputacion-granada:prueba-contenido-presentacion}"
contenedor_produccion=""
contenedor_presentacion=""
inventario_produccion="$(mktemp)"
inventario_presentacion="$(mktemp)"

limpiar() {
  if [ -n "$contenedor_produccion" ]; then docker rm -f "$contenedor_produccion" >/dev/null 2>&1 || true; fi
  if [ -n "$contenedor_presentacion" ]; then docker rm -f "$contenedor_presentacion" >/dev/null 2>&1 || true; fi
  rm -f "$inventario_produccion" "$inventario_presentacion"
}
trap limpiar EXIT INT TERM

docker build --target runtime -t "$imagen_produccion" .
docker build --target runtime-presentacion -t "$imagen_presentacion" .

contenedor_produccion="$(docker create "$imagen_produccion")"
contenedor_presentacion="$(docker create "$imagen_presentacion")"
docker export "$contenedor_produccion" | tar -tf - >"$inventario_produccion"
docker export "$contenedor_presentacion" | tar -tf - >"$inventario_presentacion"

if grep -Ei 'app/web/.*presentacion|(^|/)data/demo/|\.demo\.json$|usr/local/bin/vec-presentacion$' "$inventario_produccion"; then
  echo "ERROR: el artefacto de produccion contiene material exclusivo de presentacion" >&2
  exit 1
fi

for ruta in \
  app/web/static/presentacion/index.html \
  app/web/static/area-personal/adaptador-presentacion.js \
  app/web/static/portal-empleado/datos-presentacion.js \
  app/web/static/portal-empleado/portal-presentacion-adaptador.js \
  app/web/static/bolsa/documentos/bases-demo.css \
  app/web/static/bolsa/documentos/bases-auxiliar-demo.html \
  app/web/static/bolsa/documentos/bases-auxiliar-demo.pdf \
  app/web/static/bolsa/documentos/bases-gestion-demo.html \
  app/web/static/bolsa/documentos/bases-gestion-demo.pdf \
  app/web/static/bolsa/documentos/bases-operario-demo.html \
  app/web/static/bolsa/documentos/bases-operario-demo.pdf \
  app/data/demo/convocatorias_publicas.demo.json \
  usr/local/bin/vec-presentacion
do
  if ! grep -Fxq "$ruta" "$inventario_presentacion"; then
    echo "ERROR: falta $ruta en el artefacto de presentacion" >&2
    exit 1
  fi
done

if grep -Fxq 'usr/local/bin/vec-server' "$inventario_presentacion"; then
  echo "ERROR: el artefacto de presentacion contiene el binario productivo" >&2
  exit 1
fi

echo "Contenido de los artefactos de produccion y presentacion verificado."
