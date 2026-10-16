# Preferencias de Mi Bolsa interna (F1-12)

El aspirante que entra con su sesión corporativa ordena sus bolsas en
`GET/PUT /api/vec/bolsa/mi-bolsa/preferencias`, la misma ruta y el mismo
formulario que el aspirante externo. `GET` devuelve el orden y el último
recibo. `PUT` exige la versión esperada y una clave de idempotencia, y devuelve
un recibo; repetir la misma clave con el mismo orden devuelve el recibo y la
fecha originales. Sin el selector, la ruta no se monta y el servidor arranca
igual. Con el selector y alguna pieza ausente, solo esta ruta responde 503.

## SQL, en este orden y una sola vez

Lista: `deploy/principal/lista_sql_claude_preferencias_internas_20261016.txt`.
Va detrás de la cadena fija del núcleo #1147 → #1144 (AD276) → #1146 (AD278)
→ #1155 (AD283) y delante de AD288 (#1176) y AD273.

1. AD279 reconstruye el consumidor central sobre el cuerpo instalado tras AD283
   (huella `3c3e5f8a…ecda`, extraída de una copia con `pg_get_functiondef`) y
   añade solo la rama `preferencias_bolsas_interna`: sesión del LOGIN nominal
   `vec_interno_bolsa_preferencias_desarrollo`, miembro único del grupo
   `vec_bolsa_llamamientos_preferencias_interna`; audiencia
   `vec.bolsa.preferencias-bolsas.ordenar.interna.v1`; acción
   `bolsa.preferencias_bolsas.ordenar_propias` sobre `mi-bolsa:can_*`; vínculo
   `interna_corporativa`. Saca al ejecutor genérico de CT de ese perfil, añade
   la audiencia a la restricción de claves y sustituye la restricción de
   lectura v1 por la v2, que admite también el canal `interna_corporativa`.
   Crea el ayudante `registrar_y_consumir_preferencias_bolsas_interna_v3_atestada`,
   ejecutable solo por el propietario de Bolsa. Huellas posteriores: núcleo
   `a46491328a00b33d5d9276c9fa6b4eec6287ba015d865df14aed807fe2d58fb2`,
   audiencias `504fd62c…4b15`, lectura `87b134b2…aa9e`. AD288 parte de esta
   huella del núcleo.
2. AUT96 instala el cotejo de la lectura contra la asignación vigente del
   perfil propio.
3. B116 crea el grupo y las tres fachadas Bolsa `_interna_v1` (consulta,
   intento denegado y orden). Las claves internas se guardan con un prefijo
   que B104 no admite, con actor y perfil, para que cada superficie recupere
   solo sus propios recibos.
4. AD275 crea las dos fachadas de lectura auditada para la sesión interna.

Prueba focal, después de las cuatro:
`deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000279_preferencias_bolsas_internas.sql`.

## Activación en una copia

Nada de esto se concede por defecto ni lo crea el arranque, salvo la clave
de capacidad, que vec-server publica con su gobierno de desarrollo como las
demás audiencias internas cuando el selector está activo.

- El DBA crea el LOGIN `vec_interno_bolsa_preferencias_desarrollo` con
  `CONNECT` y membresía única en `vec_bolsa_llamamientos_preferencias_interna`
  (`INHERIT TRUE`, `SET FALSE`, `ADMIN FALSE`), sin credenciales compartidas
  con el portal externo ni con Usuarios.
- Fila de origen en `vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`:
  ese LOGIN, la audiencia interna, la acción de ordenar, proceso `vec-server`
  y canal `interna_corporativa`. Mismo molde que AD283.
- Concesión al perfil propio interno del aspirante: versión del rol
  `candidato_bolsa_portal_historial_propio_desarrollo` con
  `bolsa.preferencias_bolsas.consultar_propias` (los ocho campos de
  `CamposLecturaPreferenciasBolsas()`) y `bolsa.preferencias_bolsas.ordenar_propias`
  (sin campos), finalidad `gestion_preferencias_bolsas_propias` y ámbito
  `candidato_ref` único. Hoy no hay un acto de Administración que asigne este
  rol a un perfil interno, así que en la copia va como semilla declarada, sin
  acto ni recibo, igual que el rol externo (decisión de dirección del 16/10 a
  las 03:48). Responsable: equipo O. Se retira al integrar la tarea de
  permisos por datos de la fase 2 (F2-12).
- Material privado `identidad/bolsa-preferencias-interna.json` (`version: 1`,
  `superficie: "interna_corporativa"`, `dsn_bolsa` del LOGIN con TLS
  `verify-full`, `certificado_sha256`, `sujeto`, `cuenta_ref`, `perfil_ref`,
  `persona_ref`, `candidato_ref`). La cuenta y el perfil son distintos de los
  del aspirante externo.
- Selectores: `VEC_BOLSA_PREFERENCIAS_INTERNA_ENABLED=true` y
  `VEC_USUARIOS_PREFERENCIAS_ENABLED=true` con su material interno; el portal
  externo no debe estar combinado en el mismo proceso.

Comprobar en la copia: arranque con el selector apagado y encendido; `GET`
propio 200 con apunte de lectura; `PUT` 201 con recibo; repetición 200 con el
mismo recibo; tras reiniciar aplicación y PostgreSQL, la misma versión, orden,
recibo y fecha. Una candidatura ajena o una sesión externa se rechazan y dejan
su auditoría. `POST` responde 405.

## Estado (16/10)

Ensayado en una copia de HZ37 con main y las SQL de #1144, #1146 y #1155: las
cuatro migraciones se instalan una vez y la prueba focal pasa. El recorrido
HTTP con recibo y reinicio en copia está pendiente: necesita la identidad
interna del aspirante, Usuarios interno activo, la concesión y el LOGIN
anteriores. CT225 y CT227 de #1146 no entran sobre main en esa copia (la
consulta de detalle de RRHH de main tiene otra huella); no afecta al
consumidor central.
