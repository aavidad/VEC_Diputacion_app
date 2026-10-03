# Preparar la revisión de requisitos

Este CLI prepara una lista de revisión motivada por requisito desde material
sintético local. La salida identifica las bases propuestas, la revisión de
preparación, los soportes por referencia y versión y las carencias que requieren
atención. Todos los requisitos quedan pendientes de revisión competente.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-selectivos-preparar-admision \
  --idioma es \
  < cmd/vec-selectivos-preparar-admision/testdata/material.json
```

Use `--idioma en` para los errores y la ayuda en inglés. `--catalogos-dir`
selecciona la raíz del catálogo común. `--help` muestra la ayuda traducida. El
resultado de éxito es el mismo contrato JSON en ambos idiomas; sus causas y
acciones son claves de catálogo. Los títulos propuestos son datos del material
aportado y se conservan sin traducción automática.

Una entrada inválida devuelve código distinto de cero, un error en la salida
de errores y ningún resultado en la salida estándar. Se rechazan campos
desconocidos, claves duplicadas o con mayúsculas, JSON adicional, UTF-8 inválido,
versiones de hechos distintas de las esperadas y afirmaciones de presentación.
La entrada tiene un límite de un MiB y 32 niveles; el catálogo, de 64 KiB.

## Material y resultado

El ejemplo `testdata/material.json` propone dos requisitos: uno estructurado
con un hecho declarado y otro en texto libre. El resultado conserva ambos
pendientes y propone preparar su aportación o revisión. Estas propuestas se
identifican dentro de `preparacion_ref` y `revision`, junto al requisito y sus
soportes exactos. Ninguna crea un requerimiento de subsanación.

`testdata/resultado.json` conserva la salida real de ese comando para los
consumidores del contrato. Su SHA256 es
`b812a75e1a2a9c7249878b6c3abfa565636628218946d2f41acb56183ed7a882`.

`bases` conserva referencia, versión y huella **aportadas**. `requisitos`
contiene referencias, versiones, títulos e hitos **propuestos**. Este corte
no coteja ese material con la versión autorizada de las bases; siempre devuelve
`universo_requisitos: propuesto_no_cotejado`. Una lista vacía conserva los
pendientes de definir y cotejar el universo, sin producir admisión.

`hechos` reutiliza `meritos/ports.HechosPreparados`, el contrato sintético RUM.
No recibe datos de Persona ni nombres de aspirantes. La aplicación minimiza
la salida a referencia y versión del hecho, fuente y versión, estado aportado,
número de evidencias y vigencia. No abre documentos ni interpreta puntos,
equivalencias o reglas de acceso. Un estado acreditado, rechazado o unas fechas
fuera del hito permanecen pendientes de aplicación a las bases.

`solicitud_contexto` es opcional. Solo identifica la propuesta de contexto S3
por sus metadatos y la huella del archivo original. S3
(`web/static/bolsa/preparacion/recuperacion/material.js`, PR #403) recupera un
resumen local `sin_presentar`; no ofrece referencia de solicitud registrada,
recibo, requisito versionado ni hito. Las referencias nuevas de esta utilidad
no se extraen de S3. La versión y huella propuestas deben concordar entre los
dos bloques locales; esta comprobación no confirma su autenticidad.

Las acciones indican qué preparar para revisión. La autoridad competente debe
determinar todavía qué es subsanable y bajo qué condiciones, motivar las
decisiones y aprobar las listas. Este corte no calcula plazos, admite, excluye,
publica, persiste, firma, autentica ni consulta permisos. `persistido` y
`admision_oficial` permanecen en `false`.

## Comprobación focal

```sh
go test -p 8 -timeout 120s \
  ./internal/modules/seleccion/application \
  ./cmd/vec-selectivos-preparar-admision
go vet -p 8 ./internal/modules/seleccion/application \
  ./cmd/vec-selectivos-preparar-admision
```

Las pruebas comprueban referencias y versiones, preparación sin solicitud,
lista vacía, rechazo y vigencia sin exclusión, entrada ambigua sin resultado y
catálogos reales ES/EN. No acreditan lectura nominal de RUM, hechos de Personal,
presentación registrada, instalación SQL ni un recorrido institucional.

Fuentes: `ESPECIFICACIONES_AGENTES.md` E02, E03, E07 y E08;
`docs/plan_modulos/selectivos.md` S3/S4 y SEL-006;
`docs/estudio_requisitos/diseno_s1_b15_requisitos_2026-09-23.md`, evaluación
personal y separación de admisión; `internal/modules/meritos/ports/hechos.go`,
contrato de preparación sintética. S2 se conserva en su preparación de bases
existente; este corte no crea otra autoridad de bases.
