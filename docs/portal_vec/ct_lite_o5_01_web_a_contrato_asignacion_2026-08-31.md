# CT-LITE-O5-01-WEB-A-R2 — cierre de RegExp e instante cero

Fecha: 31 de agosto de 2026.

Estado del productor: candidato correctivo local pendiente de revisión
independiente del hash exacto. `R1` recibe `NO-GO`, con dos hallazgos `P1`
reproducidos en el preflight de este corte. Este documento no concede `GO`, no
cierra `O5-01` y no acredita HTTP, formulario, montaje, composición,
persistencia, PostgreSQL, E2E ni producción.

## Punto de partida

- Worktree exclusivo:
  `.worktrees/ct-o5-web-a-r2-20260901`.
- Rama:
  `trabajo/ct-o5-web-a-r2-20260901`.
- `HEAD` y base exacta antes de editar:
  `58800913b32e22b0f77eb8d62900d95c452e98fa`.
- Primer padre de la base:
  `c647628bcbfc278075e5d704a90033dbe1152c59`.
- Árbol inicial: limpio.
- Toolchain web: Node `v20.19.2`.
- Único `AGENTS.md` aplicable: el de la raíz, leído completo junto con las
  autoridades obligatorias y las autoridades O5 enlazadas. También se
  contrastaron completos `domain/tipos.go` y
  `adapters/httpinterno/asignacion_contrato.go`.

No se usaron Git de red, otros worktrees, gates pesados, Go, detector de
carreras, `go vet`, Docker, PostgreSQL, despliegue, producción ni credenciales.

## Preflight local anterior a la edición

El preflight comprobó la rama, la base exacta, el árbol limpio, los modos
`100644` y el write-set previsto. La prueba focal heredada terminó `14/14`
verde y la totalidad de `web/**/*.test.mjs`, todavía ligera, terminó
`434/434` verde. La importación aislada expuso solo los tres validadores
declarados.

Ese verde no cubría los dos `P1`. Un ensayo adversarial separado, ejecutado
después de importar el módulo, alteró simultáneamente
`RegExp.prototype.exec` y `RegExp.prototype.test`: el error de `exec` escapó
desde la validación de una referencia inválida. El mismo ensayo confirmó que
`0001-01-01T00:00:00Z` era aceptado. Quedó así reproducido el `NO-GO` real de
`R1` antes de modificar ningún fichero.

`gitleaks` no está en `PATH`, pero el ejecutable local exacto autorizado
`/tmp/vec-gitleaks-20260831` sí estaba disponible. Su SHA-256 comprobada antes
de editar es
`c100de843d374f76143b03487de20fe341fb20cae8a71b6fdff896aec561391d`;
no se descargó ni instaló nada.

## Capability, invariante, write-set y siguiente corte

Capability única: validación web pura, fail-closed y sin autoridad del recibo
de asignación de contrato. `R2` conserva la entrada como texto JSON primitivo,
canónico y limitado y cierra exclusivamente la dependencia dinámica de
`RegExp.prototype.exec` y la aceptación del instante cero de Go.

Invariante: ninguna entrada imposible o manipulada se convierte en recibo
válido; hay cero DOM, red, almacenamiento web, cookies o autoridad. Antes de
cualquier inspección se exige `typeof entrada === "string"`. No se refleja,
enumera, clona, serializa ni coerciona un objeto del llamador. Un `Proxy`
transparente u hostil, un `Proxy` revocado, un `String` objeto o un objeto con
getters se rechazan por tipo sin ejecutar trampas, getters o conversiones. El
JSON expresa solo intención funcional; nunca es fuente de identidad, actor,
perfil, organización, rol, permiso o decisión. Las salidas son objetos
ordinarios nuevos, congelados y compuestos únicamente por primitivos
validados.

Write-set exacto:

- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.js`;
- `web/static/portal-empleado/modulos/contratacion-temporal/contrato-asignacion.test.mjs`;
- `docs/portal_vec/ct_lite_o5_01_web_a_contrato_asignacion_2026-08-31.md`.

Siguiente corte: un agente distinto debe revisar y reproducir el hash exacto
del único commit `R2`. El productor no integra ni se autoaprueba. Un `GO`
independiente del hash exacto es condición previa a cualquier integración;
`WEB-B` continúa separado y condicionado por el propietario `cliente-http.js`.

## NO-GO de R1 y corrección R2

El candidato `R1` `52b57279534031248759ac2f8f28cc2012b66190`, incorporado en
la genealogía de la base mediante `c647628bcbfc278075e5d704a90033dbe1152c59`,
queda registrado como `NO-GO` para este alcance por dos `P1`:

1. Había capturado `RegExp.prototype.test`, pero el `test` original resolvía
   dinámicamente `patron.exec`. Una alteración posterior a la importación
   podía aceptar referencias y UUID inválidas o filtrar el error atacante.
2. Admitía `0001-01-01T00:00:00Z`, representación textual exacta de
   `time.Time{}` que `domain.InstanteUTCCanonico` y el adaptador HTTP rechazan.

`R2` elimina toda dependencia de `test`: captura el `exec` original al
importar y lo invoca directamente con el `Reflect.apply` original capturado,
exigiendo resultado distinto de `null`. La validación temporal rechaza solo
el instante cero exacto; `0001-01-01T00:00:00.000001Z` permanece válido y
acredita que el resto del año 1 canónico no se cierra.

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
hasta seis decimales sin ceros finales. Se rechaza exactamente
`0001-01-01T00:00:00Z`; una fracción canónica no nula en el año `0001` sigue
siendo válida.

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
15/15 verdes

rg --files web -g '*.test.mjs' -0 | xargs -0 node --test
435/435 verdes
```

La matriz focal cubre los tres exports; nominales, copia ordinaria, congelado
y valores primitivos; semántica previa; `Proxy` transparente y hostil con cero
trampas, revocado, `Proxy` de `String` y getters con cero accesos; campos
ausentes, extra, duplicados y de autoridad; espacios, BOM, orden, `\u0061`,
`\/` y `1.0`; JSON malformado, nulo, arrays, primitivas, múltiples valores y
anidados; `__proto__`, `constructor` y `prototype`; 1.000/1.001 astrales;
límite temprano de 8.192 bytes; `exec` y `test` alterados simultáneamente tras
la importación, restaurados en `finally`, con referencias y UUID inválidas
rechazadas y cero invocaciones del código atacante; rechazo del instante cero
y aceptación del año `0001` con fracción no nula; intrínsecos capturados;
`TypeError` opaco; e importación sin efectos.

La inspección del productivo comprueba además ausencia de DOM, red,
almacenamiento web, autoridad de navegador, dependencias Node,
`TextEncoder`, `Blob` y `structuredClone`. La búsqueda focal no devolvió
coincidencias y la importación aislada confirmó los tres exports exactos sin
añadir propiedades globales. `git diff --check` terminó limpio.

La auditoría final exige además write-set exacto de tres ficheros, modos
`100644`, menos de 800 líneas por fichero y Gitleaks local con el SHA-256
autorizado ya comprobado. El escaneo del diff preparado con ese ejecutable
terminó sin filtraciones.

## Límites

Este corte no modifica `cliente-http.js`, formularios, i18n, rutas,
composición ni contratos Go. No ejecuta HTTP, genera claves, monta vistas,
registra rutas ni añade persistencia, notificación, auditoría o efectos.

No se ejecutan Go, carrera, `go vet`, Docker, PostgreSQL ni gates globales por
estar fuera del write-set y por prohibición expresa de la tarea.

Revisión independiente: **PENDIENTE** sobre el hash exacto del único commit
candidato.
