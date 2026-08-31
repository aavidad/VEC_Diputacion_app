# CT-LITE-O5-01-WEB-A-CONTRATO-ASIGNACION — contrato web neutral

Fecha: 31 de agosto de 2026.

Estado del productor: candidato local pendiente de revisión independiente del
hash exacto. Este documento no concede `GO`, no cierra `O5-01` y no acredita
HTTP, formulario, montaje, composición, persistencia, PostgreSQL, E2E ni
producción.

## Punto de partida

- Worktree exclusivo:
  `.worktrees/ct-lite-o5-01-web-a-contrato-asignacion-20260831`.
- Rama:
  `trabajo/ct-lite-o5-01-web-a-contrato-asignacion-20260831`.
- Base y `HEAD` exactos antes de editar:
  `ba9d2463f0f8672702d3de6986f0882faf7b78cd`.
- Árbol inicial: limpio.
- Toolchain web: Node `v20.19.2`.
- `skill_ref`: `admin-data-web`. La skill no estaba instalada ni expuesta en
  la sesión; se aplicaron directamente los criterios administrativos
  suministrados para este corte: neutralidad del contrato, trazabilidad sin
  certeza jurídica inventada y preparación para una interfaz accesible futura.

No se usaron Git de red, otros worktrees, gates globales, Go, detector de
carreras, `go vet`, Docker, PostgreSQL, despliegue, producción ni credenciales.

## Capability, invariante, write-set y siguiente corte

Capability única: validar y copiar de forma cerrada la intención JavaScript de
asignar o reasignar un expediente y el recibo HTTP público mínimo. El contrato
no ejecuta transporte, no presenta interfaz y no monta el módulo.

Invariante: el navegador expresa solo intención funcional. Ningún campo de
autenticación, sesión, actor, identidad, perfil, organización, rol, permiso o
decisión puede entrar, derivarse, persistirse o devolverse por estos
validadores. Las salidas son objetos nuevos y congelados compuestos solo por
primitivos ya validados.

Write-set exacto:

- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.js`;
- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs`;
- `docs/portal_vec/ct_lite_o5_01_web_a_contrato_asignacion_2026-08-31.md`.

Siguiente corte: revisión independiente del hash candidato. `WEB-B` continúa
condicionado a descomponer el propietario `cliente-http.js`, que en la base
ocupa 792 líneas, sin duplicarlo ni ampliarlo dentro de este corte.

## Superficie pública y contrato cerrado

El fichero productivo exporta únicamente:

```text
validarSolicitudAsignacion
validarSolicitudReasignacion
validarReciboAsignacion
```

La asignación admite exactamente:

```text
expediente_ref
version_esperada
clave_idempotencia
unidad_ref
responsable_ref
```

La reasignación añade obligatoriamente:

```text
motivo_reasignacion_clave
observaciones
```

Las referencias opacas usan la gramática ASCII compartida y ocupan entre 3 y
160 caracteres. La versión esperada permite el incremento seguro en clientes
IEEE-754. La idempotencia exige UUIDv4 canónica minúscula distinta del
centinela nulo. El motivo solo conserva una clave de catálogo; su existencia,
vigencia y significado siguen perteneciendo a la autoridad gobernada.

Las observaciones son obligatorias, NFC, sin espacios periféricos, con un
máximo de 1.000 puntos de código Unicode y sin controles salvo salto de línea
y tabulador internos. El validador compara la forma original: no recorta ni
normaliza una entrada inválida.

El recibo admite exactamente:

```text
esquema
operacion
expediente_ref
version_resultante
recibo_ref
confirmada_en
```

El esquema es
`vec.contratacion-temporal.recibo-asignacion.v1`; la operación solo puede ser
`asignar` o `reasignar`; y el instante exige `Z`, año civil de `0001` a
`9999`, fecha y hora reales y hasta seis cifras decimales en forma canónica.
No se publican organización, unidad, responsable, notificación, auditoría,
evento, decisión ni HMAC.

## Endurecimiento del límite JavaScript

Los tres validadores aceptan únicamente objetos cuyo prototipo sea exactamente
`Object.prototype`. La inspección usa descriptores propios y no lee el valor a
través de la propiedad aportada. Se rechazan antes de copiar:

- nulos, arrays, instancias y prototipos alternativos;
- símbolos, propiedades extra o ausentes;
- accesores, incluso si son enumerables, sin ejecutarlos; y
- propiedades no enumerables.

Toda entrada inválida produce un `TypeError` opaco que no incorpora el dato
rechazado. El módulo no usa DOM, red, reloj, aleatoriedad, cookies ni
almacenamiento y su importación no modifica `globalThis`.

## Criterio administrativo, privacidad, i18n y accesibilidad

El recibo acredita solo la forma mínima de una respuesta HTTP. Su sintaxis no
afirma competencia, autorización, notificación, auditoría, firma, firmeza,
efecto jurídico ni cumplimiento. Esas garantías permanecen en las autoridades
y transacciones del servidor y requieren sus revisiones competentes.

Las referencias son opacas y no se incorporan nombres, DNI, correo, teléfono,
documentos ni otros datos personales directos. No hay texto visible, i18n,
CSS, formulario o foco en este corte. Una interfaz futura deberá traducir los
estados y errores mediante la autoridad i18n central y superar sus puertas de
teclado, foco, lector, contraste, zoom y revisión visual; este contrato no las
da por cumplidas.

## Pruebas reales

Resultados ejecutados antes de preparar la evidencia final:

```text
node --test web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs
14/14 verdes

node --test web/static/portal-empleado/modulos/contratacion-temporal/*.test.mjs
102/102 verdes
```

La matriz focal cubre los tres nominales y sus copias congeladas; cada campo
ausente; campos extra y autoridad inyectada; prototipos no ordinarios;
símbolos; accesores no ejecutados; no enumerables; alfabetos y límites de
referencias, versiones, UUID y motivo; observaciones vacías, periféricas, NFD,
controles, 1.000/1.001 puntos incluidos astrales y newline/tab; esquema y
operación; instantes imposibles, zonas, años y precisión; mutación posterior;
importación sin efectos; y ausencia semántica de red, DOM, almacenamiento y
autoridad en el productivo.

La primera ejecución focal descubrió que dos vectores negativos de prueba eran
sintácticamente válidos. Se corrigieron solo esos datos de prueba y la
repetición final fue `14/14`; no se relajó el productivo.

Las puertas restantes terminaron también en verde:

```text
git diff --cached --check
sin salida

write-set preparado
solo los tres archivos nuevos declarados; modo 100644

líneas
contrato-asignacion.js: 174
contrato-asignacion.test.mjs: 393
este documento: 197

Gitleaks --staged
cero hallazgos sobre el cambio preparado
```

Antes de ejecutar Gitleaks se verificó que
`/tmp/vec-gitleaks-20260831` tenía la huella SHA-256 exacta
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`.
El productivo queda claramente por debajo del objetivo de 500 líneas y los
tres ficheros permanecen por debajo del tope duro de 800.

## Límites y revisión requerida

Este corte no modifica `cliente-http.js`, sus pruebas previas, i18n, rutas,
composición ni ningún contrato Go. No ejecuta HTTP, no genera claves, no crea
formularios, no monta vistas, no registra rutas y no añade persistencia,
notificación, auditoría o efectos.

No se ejecutaron Go, carrera, `go vet`, Docker ni PostgreSQL porque el
write-set contiene solo JavaScript puro, su prueba `node:test` y esta
evidencia. Tampoco se ejecutaron gates globales por prohibición expresa del
corte.

Revisión independiente: **PENDIENTE** sobre el hash exacto del único commit
candidato. El productor no integra ni se autoaprueba.
