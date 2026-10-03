# Auditoría de la provisión inicial de fuentes ADMIN

Estado: candidata preparada sobre AD171 instalada en el frío post36/37.
Dirección ha ensayado AD174 y sus pruebas SQL en el clon PostgreSQL 18.4:
ambas terminaron con código 0. Quedan pendientes las dos revisiones
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

No se registra un `denegado` o `error` inventado en esa transacción de éxito.
La cobertura de fallos posteriores a rollback necesitaría un contrato propio
de identidad técnica; este append no lo fabrica.

El verificador offline debe reconocer la familia y su eslabón versionados,
rechazar campos cruzados y mantener los cálculos anteriores. Comprobar la
consistencia no autentica LOGIN, fuente, plan, aprobación o checkpoint.

## Preparación y comprobación

La preimagen es exclusivamente `auditoria_tipo_disjunto_v2` de AD171. La
huella SHA256 de `pg_get_constraintdef(oid, false)`, en UTF-8 y sin salto
final, es `f31b31dc0ec40bdd2d7a6930346210e919ab4aed225352235723525a27e7dc29`.
AD174 comprueba esa huella, conserva literalmente la condición y mantiene
su nombre. No supone columnas, preimágenes ni familias de AD172/173, que aún
no están integradas. Una sucesora necesita consumir la postimagen real.

El verificador y la CLI admiten el esquema
`vec.auditoria.verificacion.fuentes-iniciales.v1`. Comprueba consumo histórico
v1, intentos AD169, eventos AD171 y provisión AD174. Los esquemas anteriores
siguen rechazando AD174. Este formato no admite consumo AD172/173 ni declara
cubiertas familias futuras. Recalcula material y eslabón; no acredita el origen
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

Se han pasado las pruebas focales de `internal/vec/auditoria` y
`cmd/vec-auditoria-verificar`. Dirección ejecutó el SQL solo en el clon
autorizado. El productor no ha instalado SQL ni tocado servidores.
