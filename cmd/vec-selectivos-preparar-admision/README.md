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

El código de salida es `2` cuando no se puede escribir el diagnóstico de error.

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

## Borrador de la lista provisional

Con `--salida lista-provisional` el CLI reparte las solicitudes en admitidas y
excluidas a partir de la decisión que propone RRHH para cada revisión:

```sh
go run ./cmd/vec-selectivos-preparar-admision --idioma es \
  --salida lista-provisional \
  < cmd/vec-selectivos-preparar-admision/testdata/lista-material.json
```

El material lleva las revisiones de requisitos (`revisiones_s4`, el mismo
contrato de la salida `preparacion`) y una decisión por revisión. Cada decisión
nombra su revisión por el antecedente que devuelve `--salida antecedente`; el
CLI lo recalcula y rechaza la lista si falta una decisión, sobra otra, cambia
una huella o las revisiones vienen de bases distintas. Cada revisión cuenta como
una solicitud por su `preparacion_ref`; mientras no exista el registro de
solicitudes presentadas, el CLI no comprueba que estén todas las de la
convocatoria ni que dos referencias no correspondan a la misma solicitud.

Una exclusión necesita al menos un motivo y una admisión ninguno. Los motivos
y el plazo de subsanación salen del catálogo configurable
`data/catalogos/seleccion/admision_ejemplo.json` (se cambia con
`--catalogo-admision-dir` y `--catalogo-admision`). El material elige la
referencia y versión exactas; si no existen, no hay salida. Una exclusión es
subsanable si lo son todos sus motivos. El catálogo actual es de ejemplo y lo
marca como pendiente hasta que RRHH responda la pregunta 139 de `dudas.md`.

El resultado es un borrador `borrador_pendiente_aprobacion`, con `aprobada`,
`publicada` y `persistida` en `false`. Quedan para pasos posteriores:

- la aprobación por el perfil competente (pregunta 110);
- el nombre, apellidos y documento enmascarado de cada persona, y el orden por
  apellidos, que se obtienen al publicar por la autoridad de identidad;
  mientras tanto la lista se ordena por referencia de revisión;
- el último día para subsanar (`vencimiento_subsanacion`), que se calcula con
  el puerto de Calendarios desde el día siguiente a la publicación;
- la publicación oficial.

`testdata/lista-resultado.json` conserva la salida de ese comando. Su SHA256 es
`7505719610fac014f11d36b14ffe975b35f1740c463d247c6bf0983925f7ba1b`. La
entrada de esta salida admite hasta 16 MiB y 2.000 solicitudes.

## Borrador de la lista definitiva

La definitiva parte de una provisional concreta. Primero se obtiene su huella:

```sh
go run ./cmd/vec-selectivos-preparar-admision --salida antecedente-lista \
  < cmd/vec-selectivos-preparar-admision/testdata/lista-material.json
```

Después, `--salida lista-definitiva` recibe el material completo de esa
provisional, la huella anterior (`antecedente_provisional`) y una resolución
por cada excluida:

```sh
go run ./cmd/vec-selectivos-preparar-admision --idioma es \
  --salida lista-definitiva \
  < cmd/vec-selectivos-preparar-admision/testdata/lista-definitiva-material.json
```

El CLI recompone la provisional con el mismo catálogo y rechaza el material si
la huella no coincide. Cada excluida necesita una resolución, y solo una:

- `estimada`, con `via` `subsanacion` o `reclamacion`: pasa a admitida y
  `origen` guarda la vía. Por subsanación solo se estima una exclusión
  subsanable; una reclamación puede corregir cualquier motivo, incluido un
  error al preparar la provisional.
- `desestimada`, con su vía y `motivos_persistentes`: sigue excluida con los
  motivos que quedan sin resolver, que deben estar entre los de la provisional.
  No se añaden motivos nuevos.
- `no_presentada`, sin vía: sigue excluida con los mismos motivos.

Quien estaba admitido en la provisional sigue admitido. Excluirlo exigiría
darle audiencia, y eso no forma parte de este borrador. Tampoco se pueden
incorporar aquí las solicitudes omitidas en la provisional (ni admitidas ni
excluidas): hay que preparar antes otra revisión de la provisional que las
incluya (`--salida revision-provisional`). La salida queda en
`borrador_pendiente_aprobacion`, sin datos personales, con `aprobada`,
`publicada` y `persistida` en `false`. Quedan pendientes comprobar en el
registro que cada escrito llegó en plazo y que la provisional de la que parte
es la última revisión publicada: sin un registro de revisiones, el CLI no
puede saber si existe otra posterior.

`testdata/lista-definitiva-resultado.json` conserva la salida de ese comando.
Su SHA256 es
`1ee46997480bd8e9a3e8bf97e4b32e84daf9dd01e88b4684e49e20c9b46613bb`.

## Incorporar solicitudes omitidas a la provisional

Si una solicitud no aparece en la provisional (ni admitida ni excluida), se
prepara una nueva revisión de esa provisional con `--salida
revision-provisional`. El material lleva tres partes:

- `material`: el material completo de la nueva revisión, con la misma
  `lista_ref`, la `revision` siguiente y todas las solicitudes, las de antes y
  las nuevas;
- `anterior`: la provisional anterior tal como la devolvió el CLI;
- `antecedente_anterior`: la huella de esa anterior, la que da
  `--salida antecedente-lista` o el campo `anterior` de la revisión previa.

```sh
go run ./cmd/vec-selectivos-preparar-admision --idioma es \
  --salida revision-provisional \
  < cmd/vec-selectivos-preparar-admision/testdata/revision-material.json
```

El CLI rechaza la anterior si su huella no es la declarada: un archivo
recortado o con una decisión cambiada no pasa aunque sea coherente por dentro.
Después prepara la nueva provisional y comprueba que conserva la anterior:
mismas bases, catálogo y plazo, y cada solicitud anterior con la misma
decisión y los mismos motivos (el orden de los motivos da igual). Debe añadir al menos una solicitud. Corregir una
decisión ya tomada es otro acto y aquí se rechaza.

La salida (`vec.seleccion.lista-admision-revision.v1`) contiene la nueva
provisional completa (`lista`), la huella de la anterior (`anterior`, la misma
que da `--salida antecedente-lista`) y las solicitudes incorporadas. La lista
va sin el enlace dentro, de modo que su huella sigue saliendo de su propio
material y la definitiva puede partir de ella como de cualquier provisional.
Para las incorporadas, el plazo de subsanación empieza al día siguiente de
publicar esta revisión.

`testdata/revision-resultado.json` conserva la salida de ese comando. Su
SHA256 es `c60769202d050c6389766dc08717624d86ac50ca114ebf46dbc412d28269d4bf`.

## Comprobación focal

```sh
go test -p 8 -timeout 120s \
  ./internal/modules/seleccion/domain \
  ./internal/modules/seleccion/application \
  ./internal/modules/seleccion/adapters/catalogoadmision \
  ./cmd/vec-selectivos-preparar-admision
go vet -p 8 ./internal/modules/seleccion/domain \
  ./internal/modules/seleccion/application \
  ./internal/modules/seleccion/adapters/catalogoadmision \
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
