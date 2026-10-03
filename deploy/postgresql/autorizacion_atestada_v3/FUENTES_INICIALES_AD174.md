# Auditoría de la provisión inicial de fuentes ADMIN

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

## Preparación

AD174 necesita la preimagen causal de AD171. Mantendrá literalmente la
condición tipada vigente y su nombre para permitir las ampliaciones de
AD172/173, sin editar esas migraciones ni el consumidor de negocio.
La migración, los vectores, las negativas y el verificador se preparan en
esta rama. No se declara LISTA antes de dos revisiones exactas y ensayo en
el clon autorizado. No se instala SQL ni se ejecuta DOWN sobre historia.
