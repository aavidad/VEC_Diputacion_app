# CT-LITE-O5-01-WEB-A-R1-TEXTO-JSON — contrato web neutral

Fecha: 31 de agosto de 2026.

Estado del productor: candidato correctivo local pendiente de revisión
independiente del hash exacto. Este documento no concede `GO`, no cierra
`O5-01` y no acredita HTTP, formulario, montaje, composición, persistencia,
PostgreSQL, E2E ni producción.

## Punto de partida

- Worktree exclusivo:
  `.worktrees/ct-lite-o5-01-web-a-contrato-asignacion-20260831`.
- Rama:
  `trabajo/ct-lite-o5-01-web-a-contrato-asignacion-20260831`.
- `HEAD` y base exacta antes de editar:
  `bb3b840285c7ca819b68eb31cab9588b74f972a1`.
- Producto usado solo como contraste local:
  `8470288773f3c6986a30d3e2480f00152f227308`.
- Árbol inicial: limpio.
- Toolchain web: Node `v20.19.2`.
- `admin-data-web` no estaba instalada ni expuesta. No se instaló: se
  aplicaron directamente las restricciones administrativas suministradas.

No se usaron Git de red, otros worktrees, gates globales, Go, detector de
carreras, `go vet`, Docker, PostgreSQL, despliegue, producción ni
credenciales.

## Capability, invariante, write-set y siguiente corte

Capability única: sustituir las entradas objeto arbitrarias de los tres
validadores por texto JSON primitivo, canónico y limitado, conservando la
semántica y las salidas cerradas de asignación, reasignación y recibo.

Invariante: antes de cualquier inspección se exige `typeof entrada ===
"string"`. No se refleja, enumera, clona, serializa ni coerciona un objeto del
llamador. Un `Proxy` transparente u hostil, un `Proxy` revocado, un `String`
objeto o un objeto con getters se rechazan por tipo sin ejecutar trampas,
getters o conversiones. El JSON expresa solo intención funcional; nunca es
fuente de identidad, actor, perfil, organización, rol, permiso o decisión.
Las salidas son objetos ordinarios nuevos, congelados y compuestos únicamente
por primitivos validados.

Write-set exacto:

- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.js`;
- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs`;
- `docs/portal_vec/ct_lite_o5_01_web_a_contrato_asignacion_2026-08-31.md`.

Siguiente corte: un agente distinto debe revisar y reproducir el hash exacto
del único commit candidato. El productor no integra ni se autoaprueba. `WEB-B`
continúa separado y condicionado por el propietario `cliente-http.js`.

## Superficie pública exacta

El fichero productivo exporta únicamente:

```text
validarSolicitudAsignacion
validarSolicitudReasignacion
validarReciboAsignacion
```

Cada función recibe un `string` primitivo con un único valor JSON. La
asignación exige este orden exacto:

```text
expediente_ref
version_esperada
clave_idempotencia
unidad_ref
responsable_ref
```

La reasignación añade, también en este orden:

```text
motivo_reasignacion_clave
observaciones
```

El recibo exige el orden:

```text
esquema
operacion
expediente_ref
version_resultante
recibo_ref
confirmada_en
```

Las salidas conservan los mismos campos, valores y orden, pero se construyen
como objetos ordinarios nuevos y congelados. No se devuelve el objeto interno
analizado.

## Límite previo y canon único

El texto tiene un máximo absoluto de `8192` bytes UTF-8. El contador recorre
puntos de código, trata cada par sustituto válido como un solo punto de cuatro
bytes y corta al superar el límite. Esta puerta precede a `JSON.parse` y no usa
`TextEncoder`, `Blob`, DOM, Node ni `structuredClone`.

El módulo captura al importar los intrínsecos usados. Ejecuta `JSON.parse` sin
`reviver` y solo acepta como raíz el objeto ordinario fresco producido por ese
parser. Tras comprobar las propiedades propias esperadas y que cada valor sea
primitivo JSON, crea un registro interno con prototipo nulo, inserta sus claves
en el orden cerrado y compara el texto original con el `JSON.stringify` de ese
registro interno.

Nunca se aplica `JSON.stringify` a un objeto aportado por el llamador. La
comparación única rechaza, aunque el parser los aceptase:

- espacios periféricos, BOM o cualquier formato con whitespace;
- claves ausentes, extra, reordenadas o duplicadas;
- raíces múltiples, primitivas, nulas o arrays;
- objetos o colecciones anidados;
- escapes alternativos como `\u0061` o `\/`;
- números equivalentes no canónicos como `1.0`; y
- `__proto__`, `constructor`, `prototype` y campos de autoridad.

Los máximos alcanzables después de aplicar la semántica son inferiores al
límite absoluto:

| Forma | Máximo canónico UTF-8 |
| --- | ---: |
| Asignación | `634` bytes |
| Reasignación | `4764` bytes |
| Recibo | `524` bytes |

La regresión de reasignación acredita exactamente `4764` bytes con 1.000
caracteres astrales. El carácter astral 1.001 se rechaza por semántica; una
entrada mayor de `8192` bytes se rechaza antes del parser.

## Semántica conservada

Las referencias opacas conservan su gramática ASCII de 3 a 160 caracteres.
La versión esperada es un entero seguro entre 1 y
`Number.MAX_SAFE_INTEGER - 1`; la versión resultante, entre 2 y
`Number.MAX_SAFE_INTEGER`. La idempotencia exige UUIDv4 canónica minúscula no
nula. El motivo conserva una clave de catálogo de 2 a 80 caracteres.

Las observaciones siguen siendo obligatorias, NFC, sin espacios periféricos,
con un máximo de 1.000 puntos de código y sin controles salvo salto de línea y
tabulador internos. No se recortan ni normalizan datos inválidos.

El recibo conserva el esquema
`vec.contratacion-temporal.recibo-asignacion.v1`, las operaciones `asignar` y
`reasignar`, y un instante civil UTC canónico con `Z`, años `0001` a `9999` y
hasta seis decimales sin ceros finales.

Toda entrada inválida produce el mismo `TypeError` opaco y nunca incorpora el
dato rechazado.

## Seguridad, privacidad, i18n y accesibilidad

El texto JSON solo transporta referencias opacas y datos funcionales mínimos.
No contiene nombres, DNI, correo, teléfono, documentos ni datos personales
directos. Tampoco puede aportar autenticación, sesión, actor, identidad,
perfil, organización, rol, permiso o decisión.

Un recibo validado acredita únicamente la forma mínima de la respuesta. No
afirma competencia, autorización, notificación, auditoría, firma, firmeza,
efecto jurídico ni cumplimiento; esas garantías pertenecen al servidor y a
sus autoridades durables.

No hay texto visible, CSS, formulario, foco ni cambio i18n en este corte. Una
interfaz futura deberá usar la autoridad i18n y superar por separado teclado,
foco, lector, contraste, zoom y revisión visual.

## Pruebas reproducidas

```text
node --test web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs
14/14 verdes

node --test web/static/portal-empleado/modulos/contratacion-temporal/*.test.mjs
102/102 verdes
```

La matriz focal cubre los tres exports; nominales, copia ordinaria, congelado
y valores primitivos; semántica previa; `Proxy` transparente y hostil con cero
trampas, revocado, `Proxy` de `String` y getters con cero accesos; campos
ausentes, extra, duplicados y de autoridad; espacios, BOM, orden, `\u0061`,
`\/` y `1.0`; JSON malformado, nulo, arrays, primitivas, múltiples valores y
anidados; `__proto__`, `constructor` y `prototype`; 1.000/1.001 astrales;
límite temprano de 8.192 bytes; intrínsecos capturados; `TypeError` opaco; e
importación sin efectos.

La inspección del productivo comprueba además ausencia de DOM, red,
almacenamiento web, autoridad de navegador, dependencias Node,
`TextEncoder`, `Blob` y `structuredClone`.

Las puertas finales del candidato exigen además `git diff --check`, write-set
exacto de tres ficheros, modos `100644`, menos de 800 líneas por fichero, tres
exports exactos, Gitleaks local con SHA-256 autorizado y contraste
`merge-tree` de solo lectura.

## Límites

Este corte no modifica `cliente-http.js`, formularios, i18n, rutas,
composición ni contratos Go. No ejecuta HTTP, genera claves, monta vistas,
registra rutas ni añade persistencia, notificación, auditoría o efectos.

No se ejecutan Go, carrera, `go vet`, Docker, PostgreSQL ni gates globales por
estar fuera del write-set y por prohibición expresa de la tarea.

Revisión independiente: **PENDIENTE** sobre el hash exacto del único commit
candidato.
