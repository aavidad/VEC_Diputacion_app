# Leaflet 1.9.4 vendorizado

Esta carpeta contiene la distribución oficial de Leaflet 1.9.4 necesaria para
el mapa interactivo del Portal del Empleado. Los activos se sirven desde el
mismo origen que el portal: el navegador no consulta CDN, gestores de paquetes
ni servicios externos para cargar la biblioteca.

## Procedencia verificable

- Proyecto: Leaflet.
- Versión: `1.9.4`.
- Licencia: BSD de dos cláusulas; copia literal en `LICENSE`.
- Distribución oficial: <https://github.com/Leaflet/Leaflet/releases/download/v1.9.4/leaflet.zip>.
- Código fuente de la licencia: <https://raw.githubusercontent.com/Leaflet/Leaflet/v1.9.4/LICENSE>.
- Fecha de incorporación: 2026-07-19.
- SHA-256 de `leaflet.zip`: `aaec1d5c3239a613a53e996087629aca1483cb2f0438b11b8a335c6cede4c16b`.
- SHA-256 de la licencia original: `53e8dc25862014e4324741ca18fbe3611e11d42ef69f59f86ea8c5389647d4cb`.

`leaflet.css`, `leaflet.js` y las cinco imágenes se copiaron sin modificar de
la carpeta `dist/` del ZIP oficial. `SHA256SUMS` permite verificar cada activo
que forma parte del artefacto web. No se incluyen fuentes ni mapas de fuentes
porque el portal consume la distribución UMD ya construida.

## Actualización controlada

Una actualización debe tratarse como un cambio de dependencia: descargar una
versión oficial en un directorio temporal, comprobar el archivo y la licencia,
crear una carpeta versionada nueva, actualizar las huellas y ejecutar
`leaflet-local.test.mjs`. No se debe sobrescribir esta versión ni apuntar el
portal a una CDN.
