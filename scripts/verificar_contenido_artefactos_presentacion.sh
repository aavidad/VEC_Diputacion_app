#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
ruta_checkout="$(pwd -P)"

imagen_produccion="${VEC_IMAGEN_PRODUCCION_PRUEBA:-vec-diputacion-granada:prueba-contenido-produccion}"
imagen_presentacion="${VEC_IMAGEN_PRESENTACION_PRUEBA:-vec-diputacion-granada:prueba-contenido-presentacion}"
imagen_cartografia="${VEC_IMAGEN_CARTOGRAFIA_PRESENTACION_PRUEBA:-vec-diputacion-granada:prueba-contenido-cartografia-presentacion}"
contenedor_produccion=""
contenedor_presentacion=""
contenedor_cartografia=""
inventario_produccion="$(mktemp)"
inventario_presentacion="$(mktemp)"
inventario_cartografia="$(mktemp)"
contenido_produccion="$(mktemp -d)"
contenido_presentacion="$(mktemp -d)"
contenido_cartografia="$(mktemp -d)"

limpiar() {
  if [ -n "$contenedor_produccion" ]; then docker rm -f "$contenedor_produccion" >/dev/null 2>&1 || true; fi
  if [ -n "$contenedor_presentacion" ]; then docker rm -f "$contenedor_presentacion" >/dev/null 2>&1 || true; fi
  if [ -n "$contenedor_cartografia" ]; then docker rm -f "$contenedor_cartografia" >/dev/null 2>&1 || true; fi
  rm -f "$inventario_produccion" "$inventario_presentacion" "$inventario_cartografia"
  rm -rf "$contenido_produccion" "$contenido_presentacion" "$contenido_cartografia"
}
trap limpiar EXIT INT TERM

scripts/aprovisionar_cartografia_osm.sh
docker build --target runtime -t "$imagen_produccion" .
docker build --target runtime-presentacion -t "$imagen_presentacion" .
docker build --target runtime-cartografia-presentacion -t "$imagen_cartografia" .

contenedor_produccion="$(docker create "$imagen_produccion")"
contenedor_presentacion="$(docker create "$imagen_presentacion")"
contenedor_cartografia="$(docker create "$imagen_cartografia")"
docker export "$contenedor_produccion" | tar -tf - >"$inventario_produccion"
docker export "$contenedor_presentacion" | tar -tf - >"$inventario_presentacion"
docker export "$contenedor_cartografia" | tar -tf - >"$inventario_cartografia"
docker export "$contenedor_produccion" | tar -xf - -C "$contenido_produccion"
docker export "$contenedor_presentacion" | tar -xf - -C "$contenido_presentacion"
docker export "$contenedor_cartografia" | tar -xf - -C "$contenido_cartografia"

if grep -Ei 'app/web/.*presentacion|app/web/.*demo|(^|/)data/demo/|\.demo\.json$|usr/local/bin/vec-presentacion$|app/web/static/(index\.html|app\.js|catalogo-categorias\.(js|css))$|app/web/static/modulos/' "$inventario_produccion"; then
  echo "ERROR: el artefacto de produccion contiene material exclusivo de presentacion" >&2
  exit 1
fi

scripts/verificar_web_produccion.sh "$contenido_produccion/app/web"

# La copia historica se sirve solo desde los artefactos internos o productivos.
# La presentacion dispone de un mediador cartografico propio y no recibe el ZIP.
if grep -Eq '^app/web/cartografia(/|$)' "$inventario_presentacion" ||
   [ -e "$contenido_presentacion/app/web/cartografia" ]; then
  echo "ERROR: el artefacto de presentacion contiene la cartografia historica productiva" >&2
  exit 1
fi

if grep -Eq '^app/web/static/presentacion(/|$)' "$inventario_presentacion" ||
   [ -e "$contenido_presentacion/app/web/static/presentacion" ]; then
  echo "ERROR: el artefacto de presentacion conserva el lanzador retirado" >&2
  exit 1
fi

for directorio_cerrado in area-personal portal-empleado
do
  if grep -Eq "^app/web/static/${directorio_cerrado}(/|$)" "$inventario_presentacion" ||
     [ -e "$contenido_presentacion/app/web/static/$directorio_cerrado" ]; then
    echo "ERROR: el artefacto de presentacion conserva $directorio_cerrado sin API" >&2
    exit 1
  fi
done

for raiz in "$contenido_produccion" "$contenido_presentacion" "$contenido_cartografia"
do
  if [ -e "$raiz/app/config" ]; then
    echo "ERROR: el artefacto contiene /app/config, que no es material de ejecucion" >&2
    exit 1
  fi
  if find "$raiz" -type f \( -iname '*.go' -o -iname '*_test.go' -o -iname '*.test.js' -o -iname '*.test.mjs' -o -iname '*test-helper*' \) -print -quit | grep -q .; then
    echo "ERROR: el artefacto contiene fuentes Go o fuentes de prueba" >&2
    exit 1
  fi
  # -r no sigue los enlaces simbolicos del sistema base; se inspecciona cada
  # fichero regular una sola vez y se evitan ciclos /bin -> /usr/bin.
  if coincidencias="$(grep -rIlF "$ruta_checkout" "$raiz" 2>/dev/null)" && [ -n "$coincidencias" ]; then
    printf '%s\n' "$coincidencias" >&2
    echo "ERROR: el artefacto contiene una ruta local del entorno de trabajo" >&2
    exit 1
  fi
done

for ruta in \
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

if ! grep -Fxq 'usr/local/bin/vec-cartografia-presentacion' "$inventario_cartografia"; then
  echo "ERROR: falta el binario del mediador en el artefacto cartografico" >&2
  exit 1
fi

for ruta_prohibida in \
  usr/local/bin/vec-server \
  usr/local/bin/vec-presentacion \
  app/web \
  app/data
do
  if grep -Eq "^${ruta_prohibida}(/|$)" "$inventario_cartografia"; then
    echo "ERROR: el artefacto cartografico contiene $ruta_prohibida" >&2
    exit 1
  fi
done

echo "Contenido de los artefactos de produccion, presentacion y mediador cartografico verificado."
