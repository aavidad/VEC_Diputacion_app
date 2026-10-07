# Preparar una aportación propuesta

El CLI de revisión de requisitos también prepara una aportación propuesta
ligada al material S4, su revisión y un requisito concreto. Los documentos se
identifican por referencias y versiones, junto al hecho esperado al que se
proponen vincular. La utilidad conserva las causas anteriores pendientes.

Desde la raíz del repositorio, primero obtenga el antecedente del material:

```sh
go run ./cmd/vec-selectivos-preparar-admision --salida antecedente \
  --idioma es < cmd/vec-selectivos-preparar-admision/testdata/material.json
```

El resultado contiene `esquema_material`, `preparacion_ref`, `revision` y
`huella_material_sha256`. Copie ese bloque como `propuesta.antecedente` y
conserve el mismo material dentro de `material_s4`. Indique la referencia y
versión del requisito seleccionado y las referencias propuestas de soporte.

El ejemplo sintético completo permite preparar la aportación:

```sh
go run ./cmd/vec-selectivos-preparar-admision --salida aportacion \
  --idioma es \
  < cmd/vec-selectivos-preparar-admision/testdata/aportacion-material.json
```

`--idioma en` traduce errores y ayuda. El contrato de éxito utiliza claves de
catálogo y conserva los datos aportados. El modo original `preparacion` sigue
siendo el valor predeterminado. Los tres modos usan el mismo lector cerrado de
JSON: un MiB de entrada, sin campos desconocidos, duplicados ni alias de claves.

## Qué se comprueba

La aplicación vuelve a preparar la estructura S4 mediante su caso de uso
existente. Deriva de ahí las bases, el contexto opcional S3 y la revisión
anterior del requisito. La propuesta no puede sustituir esas causas ni cambiar
el estado anterior. Referencia, revisión y huella del antecedente deben
coincidir con el material recibido; la referencia y versión del requisito deben
existir en esa revisión.

Cada soporte exige la referencia y versión de un hecho ya esperado por el
requisito. Se permite preparar documentos para un hecho esperado que todavía
no se haya aportado al paquete S4. Se rechazan soportes duplicados, versiones
distintas de las esperadas y documentos con el mismo identificador repetido,
incluso con versiones distintas. No se admiten documentos desligados del hecho
esperado en este corte.

Los documentos son `ReferenciaDocumento` de la autoridad común: `id` y
`version`. La utilidad no abre archivos, rutas o enlaces, comprueba el contenido
ni incorpora evidencias a RUM. Las listas de soportes o documentos vacías
conservan la preparación pendiente de completar y revisar.

## Identidad del material local

La huella usa la serialización local versionada
`seleccion.admision.material-local.v1`. Se calcula SHA256 sobre los bytes de
`encoding/json.Marshal` de un objeto con `esquema` primero y `material` después.
`material` es el DTO `ports.MaterialAdmisionPreparacion` de S4: mantiene el orden
de campos declarado, sus etiquetas y omisiones `omitempty`, el orden de las
listas y la distinción entre una lista `null` y una lista `[]`. Usa las reglas
de escape de esa serialización, incluido el escape de caracteres HTML.

El orden de claves y los espacios del JSON de entrada no alteran esa huella.
Cambiar los datos o el orden de una lista sí la altera. Modificar este contrato
de serialización exige otra versión. Esta huella identifica el material local;
no es una canonicalización universal, la huella del archivo original S3, una
firma, una comprobación de autenticidad ni una aprobación de las bases.

## Límites del resultado

El objeto preparado conserva `pendiente_revision_competente`. No requiere,
notifica, presenta o resuelve una subsanación, ni conserva historia durable.
No calcula fechas o condiciones legales, consume permisos o cambia RUM. Sus
indicadores de persistencia, presentación, requerimiento y resolución son
`false`.

El ejemplo recorrido conserva el antecedente en
`testdata/aportacion-antecedente.json` y el resultado en
`testdata/aportacion-resultado.json`. La salida de este último tiene SHA256
`e672e1a224c1853b8627a772aa787504196389eee8306047c200ca7798af578b`.

La autoridad competente debe cotejar la propuesta con las bases, decidir qué
puede subsanarse y bajo qué condiciones y revisar la aportación presentada por
el circuito admitido. La falta de soportes o su mera presencia no cambia una
admisión o exclusión.

Fuentes: `ESPECIFICACIONES_AGENTES.md` E02, E03, E07 y E08;
`docs/plan_modulos/selectivos.md` S4/SEL-006; el contrato S4 de preparación y el
consenso de arquitectura del 3 de octubre, conservado por dirección. Las pruebas
focales verifican el enlace exacto, las causas intactas, los soportes ausentes,
las versiones contradictorias y la entrada ambigua sin resultado. No acreditan
presentación registrada ni un recorrido institucional.
