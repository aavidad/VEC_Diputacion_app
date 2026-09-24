# Cartografía histórica OSM: procedencia y Release

Este corte distribuye **un único ZIP de 1040 PNG**, ya declarado por los
manifiestos internos y productivo. El ZIP no está rastreado en la candidata
de rescate a `main`; la rama histórica de #25 conserva su objeto publicado.
El índice JSON sí
está rastreado en `web/cartografia/` y fija cada ruta y SHA-256. No acredita
cobertura completa de Granada ni la provincia con margen; 206 posiciones del
bbox de importación carecían de vector en el MBTiles histórico.

| Material | SHA-256 | Procedencia comprobable |
| --- | --- | --- |
| `granada-base-20260719-z8-z12.zip` | `0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165` | Exportación PNG del MBTiles histórico; 1040 teselas, zoom 8–12. |
| `granada-base-20260719-z8-z12.json` | `9d69233ad62c4371a2248bef2f91db8340823cb2a618272627693805ad021f81` | Índice rastreado con SHA-256 por tesela. |
| MBTiles histórico | `1ad538ef1f9331eca95137edc078d3a3b77336d54cabc838524172fba6547ebe` | Referencia del índice; el MBTiles no forma parte de esta Release. |

La versión del MBTiles declarada por el índice es
`20260719T115334Z-53aba0ad43c4`. El índice fija además la huella del estilo
`ff8c36d2a9b64f15c8b4cc0bd115ed2973e9469cedd95d2e18a3085290a6b233`
y la imagen TileServer
`sha256:fa01ab9f902f0a8ae80b3486476a8427d1625e677ccc53c326ddae55a2eb0e29`.
No se atribuye un proveedor o fecha de extracción PBF adicionales que el
índice no demuestra. El registro de origen PBF queda pendiente en el catálogo
de activos de Sistemas.

Los datos de OpenStreetMap están bajo ODbL. Toda vista o material que muestre
estas teselas debe conservar atribución visible «© OpenStreetMap contributors»
enlazada a <https://www.openstreetmap.org/copyright>. El estilo también usa
OpenMapTiles; conservar «© OpenMapTiles» enlazado a <https://openmaptiles.org/>.
Las referencias de licencia y presentación existentes están en
`deploy/osm-tiles-granada/README.md`. El aviso `ATTRIBUTION.txt` viaja como
asset separado junto con el ZIP y `procedencia.json`.

## Preparación y publicación

Los cuatro assets preparados se custodian fuera de Git en
`/home/alberto/.local/state/vec-direccion-20260921/osm-release/`:
ZIP, índice, `procedencia.json` y `ATTRIBUTION.txt`. El tag fijo es
`osm-granada-base-20260719-z8-z12-v1`, dirigido al commit de `origin/main`
`3796cf010dd93de07c1a83a443ff15a76b889ee1`; no modifica esa rama.
Antes de publicar, Dirección verifica esos hashes y el commit, habilita
**immutable releases** en el repositorio y comprueba la respuesta de GitHub.
Comandos exactos, desde este checkout:

```sh
gh api --method PUT repos/aavidad/VEC_Diputacion_app/immutable-releases
gh api repos/aavidad/VEC_Diputacion_app/immutable-releases --jq '.enabled'
gh release create osm-granada-base-20260719-z8-z12-v1 \
  /home/alberto/.local/state/vec-direccion-20260921/osm-release/granada-base-20260719-z8-z12.zip \
  /home/alberto/.local/state/vec-direccion-20260921/osm-release/granada-base-20260719-z8-z12.json \
  /home/alberto/.local/state/vec-direccion-20260921/osm-release/procedencia.json \
  /home/alberto/.local/state/vec-direccion-20260921/osm-release/ATTRIBUTION.txt \
  --repo aavidad/VEC_Diputacion_app \
  --target 3796cf010dd93de07c1a83a443ff15a76b889ee1 \
  --title 'Cartografía histórica Granada z8–z12 v1' \
  --notes-file /home/alberto/.local/state/vec-direccion-20260921/osm-release/RELEASE_NOTES.md \
  --latest=false
```

La Release se publicó el 24/09/2026 con `immutable releases` activado (`GET`
devuelve `enabled: true`), cuatro assets, tag dirigido al commit indicado y
descarga HTTPS del ZIP con SHA-256 coincidente. La URL pública es
<https://github.com/aavidad/VEC_Diputacion_app/releases/tag/osm-granada-base-20260719-z8-z12-v1>.

El checkout limpio no tiene
el ZIP. `scripts/aprovisionar_cartografia_osm.sh` usa el ZIP local si coincide
con la huella o descarga **ese único asset** por HTTPS durante preparación,
CI o empaquetado. Falla ante descarga ausente, cambio de huella, índice alterado
o copia local sospechosa. La puerta existente valida ZIP, miembros, PNG y
huellas individuales contra el índice. No descarga teselas sueltas; el proceso
servidor no contiene código de descarga. El ZIP entra en el contexto Docker y
solo en las superficies indicadas por sus manifiestos; la presentación no lo
recibe.
