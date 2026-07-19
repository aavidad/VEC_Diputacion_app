# Cartografía OpenStreetMap on-premise para VEC

Este paquete prepara la cartografía base de Granada sin consultar servicios
cartográficos públicos durante el funcionamiento. Expone teselas PNG únicamente
en loopback y deja al proxy HTTPS del portal la ruta same-origin:

```text
deploy/osrm-granada/data/granada-buffer.osm.pbf (sólo lectura)
        │
        ├─ importación aislada: tilemaker → versión MBTiles inmutable
        │
        └─ estado/activo ─→ TileServer GL ─→ 127.0.0.1:8091
                                             │
                              proxy HTTPS ───┘
                                             │
                              /tiles/osm/{z}/{x}/{y}.png
```

El navegador no habla con `tile.openstreetmap.org`, CDN, geocodificadores ni
servicios de terceros. Las geometrías de viaje y los datos del expediente no
entran en las teselas: Leaflet los dibuja como una capa separada y sólo después
de comprobar los permisos de Dietas.

## Decisión técnica

Se usa una cadena open source pequeña y reemplazable:

- [tilemaker 3.1.0](https://github.com/systemed/tilemaker/releases/tag/v3.1.0)
  convierte el PBF ya existente al esquema OpenMapTiles dentro de un trabajo sin
  red. No requiere mantener PostgreSQL/PostGIS sólo para la cartografía base.
- [TileServer GL 5.6.0](https://github.com/maptiler/tileserver-gl/releases/tag/v5.6.0)
  renderiza las teselas vectoriales como PNG mediante MapLibre GL Native. Su
  interfaz queda detrás de Nginx; no se publica su visor ni su API de datos.
- [Leaflet 1.9.4](https://leafletjs.com/download.html), última versión estable a
  2026-07-19, se debe servir vendorizada desde el propio artefacto web. La rama
  2.0 sigue siendo prerelease y no se adopta en esta fase.

La alternativa clásica `osm2pgsql + PostGIS + Mapnik + renderd + mod_tile` es
válida y está descrita por [Switch2OSM](https://switch2osm.org/serving-tiles/),
pero supone más servicios, memoria y operación continua. Se reserva para el caso
en que Sistemas necesite actualizaciones diferenciales muy frecuentes o el
estilo OSM Carto exacto. El puerto y la URL pública no cambian al sustituir el
adaptador.

El estilo incluido es deliberadamente autónomo: representa carreteras, caminos,
ferrocarril, agua, edificios, suelo y límites sin fuentes, iconos ni peticiones
remotas. Antes de producción, Diseño y Sistemas pueden aprobar un estilo más
completo y un paquete local de tipografías; ese cambio no altera el contrato de
teselas. La ausencia actual de topónimos debe constar como limitación de la
primera fase, no ocultarse como una prestación terminada.

## Imágenes fijadas y falla cerrada

El 2026-07-19 se comprobaron en los registros upstream estos índices OCI
multi-arquitectura:

| Componente | Referencia exacta | Evidencia upstream |
|---|---|---|
| tilemaker | `ghcr.io/systemed/tilemaker:master@sha256:d32505d7827907089c2dd07517524276946d8930b4b82e93cf5c25ec989bbe41` | etiqueta interna `Version=3.1.0`, revisión `e16203e4e2fb38a11580621fc0503ef463ab849f` |
| TileServer GL | `maptiler/tileserver-gl:v5.6.0@sha256:3a9ccdb24820b6814c8119bcc8a4376c39867cb0ffe69d62919ef898b90c2427` | release upstream 5.6.0, base Ubuntu 24.04, usuario `node` |

Esta comprobación confirma la resolución del manifiesto, no sustituye el
análisis de vulnerabilidades ni una prueba sobre la plataforma de Sistemas. El
Compose usa `pull_policy: never`: si las imágenes exactas no están previamente
en el host, no descarga otra ni arranca con `latest`. Sistemas debe:

1. replicar por digest ambas imágenes en el registro interno;
2. ejecutar análisis de vulnerabilidades, SBOM y política de firma/procedencia;
3. aprobar la arquitectura (`linux/amd64` o `linux/arm64`);
4. precargar la referencia exacta en el nodo;
5. ejecutar `./verificar_imagenes_fijadas.sh` antes de importar o activar.

Si el registro interno reescribe el manifiesto, cambia el digest o exige otro
nombre, se modifica `compose.yaml` mediante cambio revisado; nunca se relaja la
fijación en una variable de entorno.

## Requisitos iniciales

- Linux con Docker Engine y Docker Compose v2.
- `bash`, `jq`, `sqlite3`, `sha256sum`, `flock` y `rg` en el host operativo.
- PBF regular en
  `deploy/osrm-granada/data/granada-buffer.osm.pbf`. Se monta en sólo lectura y
  no se duplica.
- SSD con espacio para dos generaciones MBTiles durante una actualización.
- Límites iniciales: 1 CPU/1 GiB para servir y 2 CPU/3 GiB para importar. Son
  cotas, no garantías: Sistemas debe medir tiempo, memoria, latencia y tamaño.

Copiar `.env.example` a `.env` sólo si se necesitan valores distintos. El
fragmento Nginx incluido asume el puerto inicial `8091`; si Sistemas lo cambia,
debe cambiar ambos puntos de forma coordinada.

## Validación ligera

Esta comprobación no genera teselas ni descarga imágenes:

```bash
cd deploy/osm-tiles-granada
./tests/probar_contrato.sh
```

Valida sintaxis de shell y JSON, resolución completa del perfil de importación,
digests, aislamiento de red, sólo lectura, capacidades, loopback, ausencia de
CDN y que ningún PBF/MBTiles se haya copiado a este paquete.

## Importación separada

La importación es una operación de administración, nunca una fase de arranque
del servicio:

```bash
cd deploy/osm-tiles-granada
./verificar_imagenes_fijadas.sh
./importar_version.sh
```

El script:

1. calcula la huella del PBF compartido;
2. crea una versión `AAAAMMDDThhmmssZ-12hex` que nunca sobrescribe otra;
3. ejecuta tilemaker sin red, sin capacidades y con almacenamiento temporal en
   `estado/trabajo/`;
4. valida SQLite, metadatos, zoom 0-14 y una tesela semilla de Granada;
5. publica `granada.mbtiles` en sólo lectura junto a `manifiesto.json`.

No se ha ejecutado esa importación durante la preparación del repositorio porque
es una carga pesada y debe realizarse en la máquina dimensionada por Sistemas.
Un fallo elimina únicamente la versión parcial y conserva todas las versiones
ya publicadas.

## Activación, salud y reversión

La importación imprime el identificador resultante. Se activa expresamente:

```bash
./activar_version.sh 20260719T120000Z-0123456789ab
```

La activación vuelve a verificar huella y MBTiles, intercambia el enlace
`estado/activo` de forma atómica, recrea sólo `tiles-osm` y espera una tesela PNG
real de Granada. Si el contenedor no queda saludable en dos minutos, restaura el
enlace anterior y vuelve a arrancarlo. No hay conmutación a Internet ni mapa
vacío silencioso.

Para volver conscientemente a una versión conservada se usa el mismo comando:

```bash
./activar_version.sh VERSION_ANTERIOR
```

La primera fase puede tener una interrupción breve durante la recreación. Una
segunda fase podrá añadir dos réplicas azul/verde y cambio del upstream cuando
existan recursos, sin cambiar la URL del navegador.

Comprobaciones operativas:

```bash
docker compose ps
docker compose logs --since=10m tiles-osm
curl --fail --head http://127.0.0.1:8091/styles/osm-granada/256/8/125/99.png
```

El endpoint loopback es diagnóstico local. La prueba funcional debe hacerse por
el dominio HTTPS interno y la ruta same-origin una vez instalado el fragmento
`nginx-osm-same-origin.conf`:

```text
/tiles/osm/8/125/99.png
```

Nginx sólo admite `GET` y `HEAD`, elimina cookies, autorización y cabeceras de
identidad antes de llegar al renderer. Cualquier ruta que no cumpla el patrón
`z/x/y.png` devuelve 404. El proxy de portal debe aplicar previamente su control
de red y sesión; este fragmento no sustituye Kerberos ni la autorización del
portal interno.

## Sin egreso y tratamiento de datos

- El importador usa `network_mode: none`.
- El renderer vive en una red Docker `internal: true` y sólo publica loopback.
- El renderer tiene raíz de sólo lectura, todas las capacidades eliminadas,
  `no-new-privileges`, límites de procesos, CPU y memoria y CORS desactivado.
- El mapa base contiene datos cartográficos OSM, no DNI, nombres de personal,
  expedientes, posiciones GPS ni recorridos de empleados.
- La geometría del viaje se obtiene del OSRM interno y se superpone en el
  navegador. No se debe serializar en URL de tesela, cabeceras ni logs.
- Las peticiones de teselas no se registran en el fragmento Nginx para evitar
  volumen y metadatos sin valor probatorio. La acción administrativa sobre una
  dieta se audita en VEC, no en el mapa base.

La política de firewall debe negar salida de los contenedores y negar acceso al
puerto 8091 desde interfaces distintas de loopback. Sistemas debe verificarlo
desde otra máquina antes del piloto.

## Leaflet vendorizado

No se debe insertar una etiqueta `<script>` o `<link>` de CDN. El artefacto
oficial `leaflet.zip` 1.9.4 tiene SHA-256:

```text
aaec1d5c3239a613a53e996087629aca1483cb2f0438b11b8a335c6cede4c16b
```

Durante el proceso controlado de dependencias se extraen solamente:

```text
leaflet.css
leaflet.js
images/layers.png
images/layers-2x.png
images/marker-icon.png
images/marker-icon-2x.png
images/marker-shadow.png
```

También se incorpora `LICENSE` de la etiqueta `v1.9.4` (BSD-2-Clause). Antes de
integrarlos en el artefacto web:

```bash
./verificar_leaflet_vendorizado.sh RUTA/leaflet-1.9.4
```

El verificador contiene las huellas individuales upstream. La política CSP debe
mantener `script-src 'self'` y `style-src 'self'`; Leaflet carga las teselas de
`/tiles/osm/{z}/{x}/{y}.png` con `maxZoom: 14`, sin `crossOrigin` ni plugins de
terceros. La capa se crea con atribución visible y accesible:

```text
© OpenStreetMap contributors · © OpenMapTiles
```

`OpenStreetMap` debe enlazar a
<https://www.openstreetmap.org/copyright> y `OpenMapTiles` a
<https://openmaptiles.org/>. La atribución no se oculta con CSS y se conserva en
capturas, impresiones y documentos que incorporen un mapa.

## Licencias y fuentes primarias

- Datos OpenStreetMap: ODbL; [copyright y licencia](https://www.openstreetmap.org/copyright)
  y [guía de atribución OSMF](https://osmfoundation.org/wiki/Licence/Attribution_Guidelines).
- Esquema OpenMapTiles: BSD-3-Clause para código y CC-BY-4.0 para diseño;
  [licencia upstream](https://github.com/openmaptiles/openmaptiles/blob/master/LICENSE.md).
- TileServer GL: licencia BSD-2-Clause del repositorio upstream.
- tilemaker: FTWPL y licencias de terceros enumeradas por upstream.
- Leaflet 1.9.4: BSD-2-Clause.

La procedencia y fecha del PBF deben registrarse fuera del código en el catálogo
de activos de Sistemas. `manifiesto.json` conserva su SHA-256 para que cada
generación sea reproducible y trazable.

## Decisiones pendientes de Sistemas antes de producción

- registro interno, firma/SBOM y criterio de vulnerabilidades de las imágenes;
- CPU, RAM, disco, retención de versiones y copia de seguridad de `estado/`;
- dominio HTTPS, autenticación previa y reglas de firewall Mulhacén;
- frecuencia de actualización del PBF y responsable de aprobar cada versión;
- estilo definitivo, topónimos y paquete de fuentes enteramente local;
- métricas, alertas de salud y objetivo de recuperación;
- prueba de carga con concurrencia real y aceptación de accesibilidad visual.

Hasta cerrar esas decisiones, este paquete es un contrato ejecutable de primera
fase y falla cerrado; no declara el servicio listo para producción.
