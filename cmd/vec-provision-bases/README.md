# Bases públicas de Provisión

`vec-provision-bases` muestra los datos de dos convocatorias publicadas: el concurso general 2026/PPT_01/000026 y la libre designación 2025/PPT_01/000474. La ficha del concurso identifica los nueve códigos de puesto del anexo publicado, los dos códigos retirados por rectificación, el plazo indicado en la página de RRHH y los seis máximos del baremo. Cada PDF lleva su SHA256 para comprobar la versión consultada.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-provision-bases
go run ./cmd/vec-provision-bases -expediente 2026/PPT_01/000026
```

Para cotejar un borrador de `domain.ProcesoProvision`, guarde solo el objeto `proceso`, sin solicitud ni datos de empleado. El comando usa el mismo contrato de entrada del preparador existente y devuelve diferencias con campos y valores esperados:

```sh
jq '.proceso' cmd/vec-simular-provision/testdata/proceso.sintetico.json > /tmp/provision-proceso.json
go run ./cmd/vec-provision-bases -expediente 2026/PPT_01/000026 -proceso /tmp/provision-proceso.json
```

El ejemplo sintético muestra diferencias reales frente a la publicación: convocatoria, referencia de bases, fecha de corte y vínculos de las reglas. El cotejo de máximos detecta una regla ausente, varias reglas de una familia o un valor distinto. Conserva como pendientes las fórmulas completas, excepciones, fuentes de méritos, correspondencia con RPT y desempates. El campo `estado` del contraste permanece `cotejo_parcial`, incluso si no aparecen diferencias en los campos cotejados.

Puede comprobar los bytes de un PDF descargado frente al sello del catálogo, sin enviarlo a ningún servicio:

```sh
go run ./cmd/vec-provision-bases -expediente 2026/PPT_01/000026 -tipo-fuente bop_bases_y_anexo_rectificados -verificar-fuente /ruta/al/bop.pdf
```

La salida `fuente.estado` indica `coincide` o `difiere`. La ruta del archivo no aparece en el resultado ni en el error. El catálogo contiene estos documentos públicos:

- [BOP del concurso, CVE BOP-GRA-2026-060006](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1774825281922-final-43091ebf.pdf?p=1776760572971), SHA256 `7cd3b0f34df21f0811c71242be4ca128105cc6c498ed10af1fa0f7ab28b9fc11`.
- [Rectificación 1306](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Documentos-Provision-de-Puestos/2026_PPT_01000026_-RECTIFICACION_ERRORES.pdf), SHA256 `f1f280d947cb7c97d647d62a3b487c2d12fa45cd939a7e6fb6f896b0b35a3aa0`.
- [Extracto BOE 8491](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Documentos-Provision-de-Puestos/2026_PPT_01000026-BOE.pdf), SHA256 `bd8feb124499108da02716cb71e9c047eb85a519377a73b8dee3144329beca72`.
- [BOP de libre designación, CVE BOP-GRA-2026-016005](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Documentos-Provision-de-Puestos/2025_PPT_01000474_BOP.pdf), SHA256 `dffadf491e320df735c2d7c31e1dad80c19ff90b8afd67cb081391767fe3ab39`.

La [página de RRHH del concurso](https://www.dipgra.es/servicios/areas/transparencia/mostrar/CONVOCATORIA-PARA-LA-PROVISION-DE-PUESTOS-DE-TRABAJO-POR-EL-PROCEDIMIENTO-DE-CONCURSO-GENERAL-Expte.-2026-PPT_01-000026/) indica el plazo inclusivo del 20 de abril al 4 de mayo de 2026. El motor usa fecha de corte exclusiva: el borrador se contrasta con el 5 de mayo de 2026. Esa página puede recibir nuevos anuncios; antes de preparar una versión institucional, RRHH debe confirmar el expediente, las publicaciones vigentes y la transcripción completa.

La libre designación conserva su circuito de requisitos, trayectoria, idoneidad y resolución motivada. No recibe las seis familias de puntos del concurso. Este comando consulta documentos y coteja borradores locales; no aprueba bases, comprueba vacantes, presenta solicitudes ni produce nombramientos.
