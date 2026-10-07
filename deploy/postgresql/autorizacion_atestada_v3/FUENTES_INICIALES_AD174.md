# Auditoría de la provisión inicial de fuentes ADMIN

Estado: candidata preparada sobre AD171 instalada en el frío post36/37.
El ensayo anterior de la familia confirmada terminó con código 0 en el clon
PostgreSQL 18.4. La ampliación de intentos descrita aquí exige un ensayo nuevo;
ese resultado anterior no la acredita. Quedan pendientes dos revisiones
independientes del commit final y el ensayo causal junto a CA33, IS15 y AUT39.
No se ha instalado en la principal ni se declara LISTA.

AD174 registra el acto técnico `provisionar_fuentes_iniciales_admin_v1` en la
cadena común `auditoria_consumo_v3`. Su familia es
`provision_fuentes_iniciales_admin`. Conserva los registros, bytes y huellas
anteriores; no crea otro almacén de auditoría.

Este contrato se ha pactado con los productores de PlanFuentesInicialesV1,
CA33/IS15 y el orquestador AUT39. El plan de fuentes es distinto del plan de
bootstrap: este acto no crea roles, perfiles o asignaciones ni cambia
`control_continuidad_admin`. AUT38 pertenece al bootstrap sucesor.

## Fuente y autoridad

AUT39 coteja antes de escribir el LOGIN exclusivo, su configuración privada,
el entorno de desarrollo, la aprobación, la vigencia, el canon del plan y la
preimagen esperada. Los JSON recibidos no conceden autoridad. Los propietarios
CA33 e IS15 producen sus recibos reales; AUT39 construye el agregado mínimo de
referencias opacas, versión, procedencia declarada, recibos, instantes, plan,
operación y aprobación.

`fuente_ref` y `fuente_sha256` identifican y comprometen ese agregado real,
no el plan declarativo. No incluyen nombres, datos de búsqueda clara, material
HMAC, certificados o claves. No hay material cifrado nuevo en este contrato y
su huella no se presenta como huella de un cifrado.

El actor técnico es `operador_login = session_user`. No se inventa persona,
perfil activo, contexto, sesión ni decisión V3. La procedencia conserva
`alcance_fuente = sintetico_declarado`: no acredita una fuente institucional
ni habilita producción.

## ABI privado

```text
vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)
→ auditoria_ref, secuencia, huella_sha256, correlacion_ref, registrada_en
```

Solo tienen EXECUTE el propietario AD3 y el propietario de Autorización que
ejecuta AUT39. PUBLIC, Identidad y los roles runtime no reciben ese acceso.
La función recibe un hecho ya comprobado por el orquestador; no concede
permiso para provisionar fuentes ni consulta sus tablas.

El JSON contiene exactamente 19 cadenas, obligatorias y no nulas. Este orden
fija la preimagen independientemente del orden de claves del JSON:

```text
tipo_registro
evento_ref
operador_login
plan_ref
plan_sha256
preimagen_sha256
configuracion_sha256
aprobacion_ref
alcance_fuente
accion
recurso_ref
resultado
motivo_ref
proceso
canal
finalidad_ref
correlacion_ref
fuente_ref
fuente_sha256
```

La acción es `provisionar_fuentes_iniciales_admin_v1`, el canal es
`operacion_tecnica_privada`, la finalidad es
`provision_fuentes_iniciales_admin` y el resultado es `permitido`. El proceso
procede de la configuración positiva del LOGIN; no se supone uno por defecto.
AUT39 compara esos metadatos con su configuración antes del append.

## Material, transacción y recuperación

Se reutiliza `encuadrar_mac(text)` y SHA256 existentes. El material usa el
dominio `vec.auditoria.fuentes-iniciales.v1` seguido de los 19 campos. Cada
marco contiene longitud UTF-8 decimal, dos puntos, valor y salto de línea.
El eslabón usa `vec.auditoria.eslabon.fuentes-iniciales.v1`, secuencia,
huella anterior, referencia `aud_v3_f_<32 hex>`, huella del material y fecha
UTC con microsegundos.

El evento es `evento_<32 hex>`; la correlación es `correlacion_<32 hex>`.
Se reutiliza el cerrojo común de evento y el bloqueo de la cabeza existente.
Mismo evento y mismo material recuperan las coordenadas originales; una
divergencia se rechaza. AUT39 une fuentes, recibo y auditoría en una sola
transacción SERIALIZABLE de escritura. El consumidor entrega el recibo
únicamente después de COMMIT.

La familia confirmada no registra un `denegado` o `error` ficticio. La familia
de invocación siguiente conserva el resultado real del motor observado por
AUT39, también cuando su subtransacción de efecto se revierte.

El verificador offline debe reconocer la familia y su eslabón versionados,
rechazar campos cruzados y mantener los cálculos anteriores. Comprobar la
consistencia no autentica LOGIN, fuente, plan, aprobación o checkpoint.

## Preparación y comprobación

AD174 admite únicamente dos parejas de preimagen, observadas con
`pg_get_constraintdef(oid, false)` en UTF-8 sin salto final:

| Cadena | CHECK | SHA256 previo |
| --- | --- | --- |
| H9/AD171 | auditoria_tipo_disjunto_v2 | f31b31dc0ec40bdd2d7a6930346210e919ab4aed225352235723525a27e7dc29 |
| POST173 | auditoria_tipo_disjunto_v4 | 4ea0f7f797122ddb81c59c4a601c87c213f10206d619d3c031f947f8efd2a1ea |

Conserva literalmente la condición y su nombre. En POST173, ambas familias
propias añaden `version_consumo IS NULL`: no pueden atribuirse una versión
nominal de consumo. No acepta otra variante por nombre ni predice AD177/178.
La postimagen POST173→174 medida es CHECKv4,
`3a2b7514294cd022102440e37b784916e48c340e63ce7195defdea118bad34ba`.
La variante H9 mantiene su postimagen CHECKv2 histórica.

El verificador y la CLI admiten el esquema
`vec.auditoria.verificacion.fuentes-iniciales.v1`. Comprueba consumo histórico
v1, intentos AD169, eventos AD171 y provisión AD174. Los esquemas anteriores
siguen rechazando AD174. La dependencia Go L #557@9e09b8a331afe45090db34fdf552cf9aed9e8ef7
se integra sin cambiar sus campos. La unión admite consumos reales v1/v2/v3,
conserva el aviso de históricos sin fecha ligada y no declara cubiertas
familias futuras. Recalcula material y eslabón; no acredita el origen
del operador, la aprobación, la fuente ni el checkpoint.

Hay dos vectores de bytes, uno con LOGIN UTF-8, en
`pruebas_sql/ad174_vectores_cadena.json` y en la entrada sintética de la CLI
`cmd/vec-auditoria-verificar/testdata/fuentes_iniciales_ad174.json`. Los
valores esperados se calcularon fuera del código Go; la prueba SQL coteja
los mismos valores con `encuadrar_mac` y SHA256 de PostgreSQL.

La prueba `pruebas_sql/ad174_auditoria_fuentes_iniciales.sql` hace ROLLBACK.
Comprueba ACL privada, positivo, replay sin duplicado, LOGIN ajeno, campos
cruzados, falta de fuente, alcance institucional y resultado denegado; coteja
que los rechazos no avancen la cabeza ni cambien la historia. No hace DOWN.

La CLI consume un checkpoint separado y límites explícitos. Ejemplo con la
entrada sintética y un checkpoint que contenga su `manifiesto`:

```sh
go run -p 8 ./cmd/vec-auditoria-verificar \
  -checkpoint checkpoint.json -max-bytes 65536 -max-registros 2 \
  < cmd/vec-auditoria-verificar/testdata/fuentes_iniciales_ad174.json
```

Comprobaciones locales ejecutadas, todas con código 0:

- `GOCACHE=$HOME/.cache/go-build go test -p 8 ./internal/vec/auditoria ./cmd/vec-auditoria-verificar`.
- Los mismos paquetes con `go test -race -p 8` y `go vet -p 8`.
- `gosec -quiet -fmt text` solo sobre esos dos paquetes: sin hallazgos.
- Semgrep con `--metrics=off` y reglas locales de
  `pruebas_sql/ad174_semgrep_local.yml`: cinco archivos productivos cambiados,
  tres reglas, sin hallazgos.
- `git diff --check`.

La revisión focal del productor recorrió la frontera LOGIN/AUT39, ABI de
campos cerrados, ACL, rollback, exclusividad de familias y minimización del
informe offline. No encontró un defecto pendiente; no sustituye las dos
revisiones independientes de esta zona sensible. El SQL no lee tablas CA/IS.
AUT39 debe suministrar el agregado real y unirlo al efecto antes del COMMIT.

Dirección ejecutó el SQL solo en el clon autorizado. El productor no ha
instalado SQL ni tocado servidores.

## Invocaciones gestionadas por el motor

La familia separada `intento_fuentes_iniciales_admin` registra cada invocación
que AUT39 llega a gestionar: éxito, replay, denegación o error. Su actor es
`session_user`; no contiene fuente confirmada, aprobación, preimagen de efecto,
persona, perfil ni decisión V3. Los campos de confirmación quedan nulos y la
nueva columna `fuentes_solicitud_sha256` solo pertenece a esta familia.

El ABI privado es `registrar_intento_fuentes_iniciales_admin_v1(jsonb)`.
Retorna las mismas cinco coordenadas del ABI confirmado. Recibe exactamente
estas 12 cadenas en este orden:

```text
tipo_registro
evento_ref
operador_login
solicitud_sha256
accion
recurso_ref
resultado
motivo_ref
proceso
canal
finalidad_ref
correlacion_ref
```

`solicitud_sha256` compromete los bytes reales de la entrada, sin almacenarlos.
`recurso_ref` es `solicitud_fuentes:<32 hex>`: una referencia opaca de la
invocación, no una aprobación o plan ficticio. El proceso exacto `postgresql`
identifica al motor observador; no pretende acreditar la CLI de origen.
Acción, canal y finalidad conservan los valores cerrados del ABI confirmado.

Los motivos forman un catálogo técnico cerrado de datos SQL, cotejado con
`motivos_intento_fuentes.json` del verificador:

| Resultado | Motivo |
| --- | --- |
| permitido | fuentes_registradas |
| permitido | fuentes_replay |
| denegado | fuentes_denegadas |
| error | fuentes_error |

El material usa `vec.auditoria.intento-fuentes-iniciales.v1` seguido de los
12 campos; el eslabón usa `vec.auditoria.eslabon.intento-fuentes-iniciales.v1`
con las coordenadas anteriores y referencia `aud_v3_fi_<32 hex>`. Comparten
la cabeza y el cerrojo de eventos de la auditoría común. Un evento repetido
con idéntico material recupera sus coordenadas; cualquier diferencia se rechaza.
Cada invocación real nueva necesita un evento propio, incluido un replay del
plan de provisión, que conserva un hecho distinto del replay del append.

AUT39 mantiene el efecto dentro de una subtransacción y registra después el
resultado real de la invocación. En el éxito, efecto, confirmación e intento se
confirman juntos; en un rechazo gestionado, el efecto se revierte y se confirma
únicamente el intento. Si falla el append, se aborta toda la transacción. Solo
se devuelve el sobre de resultado después del COMMIT. Cancelaciones,
desconexiones anteriores y fallos de conexión no quedan acreditados por este
contrato. No se guardan SQLERRM, entradas completas, secretos ni material HMAC.

La proyección CLI es `intento_fuentes_iniciales`, con las 11 cadenas del
material sin `tipo_registro`, seis coordenadas del eslabón y `modulo_id`.
Rechaza campos de fuente, aprobación, persona o perfil añadidos y las mezclas
con otras familias. El informe distingue `material_intentos_fuentes_recalculado`
del material confirmado; ninguno autentica el LOGIN o la solicitud de origen.

Los cuatro vectores independientes están en
`pruebas_sql/ad174_intentos_vectores_cadena.json` y en
`cmd/vec-auditoria-verificar/testdata/intentos_fuentes_ad174.json`. La prueba
`pruebas_sql/ad174_intento_fuentes.sql` coteja material y eslabón con PostgreSQL,
los cuatro resultados y replay, ACL, campos cruzados, LOGIN ajeno, motivo libre,
resultado incompatible, proceso falso, inmutabilidad e historia. Todo hace
ROLLBACK en el clon y no ejecuta DOWN. Estas pruebas SQL nuevas están pendientes
del ensayo de Dirección; las pruebas focales Go de la ampliación han pasado.

## Ensayo de la unión POST173

SQL174 variante SHA256
`6dc657372fe3ec9a7ca7d8d3f9be6cf68936c242bee315ea63e2e67dca56ebdc`:
UP y dos pruebas SQL terminaron con código 0 en el único clon. Las negativas
incluyen un INSERT de familia técnica con `version_consumo=3`, que se rechaza
sin efectos. Las 6.240 filas, cabeza e historia anteriores permanecen idénticas.

El vector independiente `union_consumos_fuentes_ad173_ad174.json` enlaza
consumos v1/v2/v3 y ambas familias de fuentes. El dominio y la CLI cotejan
la cadena y rechazan cruces entre consumos y registros técnicos. Se mantienen
los esquemas propios y no se inventan versiones v4/v5 de la proyección.

Las capturas y logs quedan fuera de Git en el estado privado de K
`vec-codexk-union-post173-20261004`. El núcleo POST173 no cambia. No se ha
instalado en la principal; las revisiones independientes del SHA final siguen
pendientes.
