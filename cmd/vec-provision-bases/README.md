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

El ejemplo sintético muestra diferencias frente a la publicación en los campos que se pueden cotejar: referencia de convocatoria, referencia de bases, fecha de corte, máximo total, y máximo y referencia de base de cada una de las seis familias. Una familia con cero o varias reglas queda señalada para revisión. El resultado lleva `cobertura_datos: parcial` y `oferta_estado: no_cotejada` aunque `diferencias` esté vacío. `PuestoOfertado` solo contiene referencias opacas de Provisión y RPT; no contiene el código del anexo BOP. Por eso el comando no puede comprobar que los puestos del borrador estén entre los nueve publicados ni que excluya los dos retirados. El pendiente `codigos_del_anexo_no_cotejados` lo indica de forma expresa. También quedan por cotejar las fórmulas completas, excepciones, fuentes de méritos, RPT y desempates.

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

## Ensayo del baremo del concurso general de 2026

El modo `concursos` del CLI existente calcula con una configuración ligada al CVE BOP-GRA-2026-060006 y una entrada sintética. Las reglas y la entrada están en `testdata/`; sus huellas de archivo fijan los bytes usados:

```sh
GOCACHE=$HOME/.cache/go-build go run -p 6 ./cmd/vec-baremador \
  --modo concursos \
  --reglas cmd/vec-provision-bases/testdata/reglas_concurso_2026_ensayo.json \
  --reglas-sha256 6c8b3352d3ded5538866afa46ac3515971003fdb55d3ddc9257ac6e880d42839 \
  --entrada cmd/vec-provision-bases/testdata/entrada_concurso_2026_sintetica.json \
  --entrada-sha256 244b3a64ff14a28ea147d71d4442dd0f32415380a5e7b2209314c1e7d58528fc
```

La versión `BOP-GRA-2026-060006:ensayo-v1` identifica esta transcripción técnica. El ejemplo calcula experiencia A (2,05 puntos), grado B (15), antigüedad C (1,02), permanencia D (2,5) y formación F (0,48). El resultado conserva huella de reglas, entrada y desglose. Titulaciones E figura como `pendiente_regla` y `total` es `null`; la suma visible de las otras familias no es la puntuación total del concurso.

La experiencia A suma meses civiles completos de periodos del mismo nivel antes de aplicar el umbral de fracción superior a seis meses. Su ventana empieza el 5 de mayo de 2016 para el corte exclusivo del 5 de mayo de 2026. Antigüedad y permanencia conservan su ventana general: la limitación de diez años de experiencia no se les aplica. Si una fecha deja un resto de días sin meses reconocidos, el cálculo devuelve `meses_no_acreditados`; no convierte esos días con un divisor supuesto.

En permanencia D, la configuración declara el factor `1/2` para tiempo provisional y detrae primero de ese tiempo los meses descartados. El reparto de puntos cuando coexisten meses definitivos y provisionales requiere confirmación de RRHH. En ese caso, la familia queda `pendiente_politica`, el total permanece `null` y las demás familias siguen calculadas. La política y los valores de esta configuración son un ensayo; no acreditan una decisión administrativa.

Las bases publicadas [6E.4 y 6E.6.d](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1774825281922-final-43091ebf.pdf?p=1776760572971) contienen una discrepancia sobre los másteres: una cláusula los excluye y otra asigna dos puntos al Máster Universitario Oficial. RRHH debe aclarar el criterio antes de incorporar la regla E. Tampoco se han cotejado fuentes de Personal/RUM/RPT, requisitos de los puestos, admisión ni adjudicación. El ensayo no registra solicitudes ni produce una propuesta o un nombramiento.
