# Preferencias de Mi Bolsa interna (F1-12)

El aspirante que entra con su sesión corporativa ordena sus bolsas en
`GET/PUT /api/vec/bolsa/mi-bolsa/preferencias`, la misma ruta y el mismo
formulario que el aspirante externo. `GET` devuelve el orden y el último
recibo. `PUT` exige la versión esperada y una clave de idempotencia, y devuelve
un recibo; repetir la misma clave con el mismo orden devuelve el recibo y la
fecha originales. Sin el selector, la ruta no se monta y el servidor arranca
igual. Con Usuarios operativo, si falta una dependencia propia de preferencias,
solo esta ruta responde 503.

## SQL, en este orden y una sola vez

Lista completa de 82 entradas: `deploy/principal/lista_sql_main_tras_hz37.txt`.
Conserva las 78 de main y añade las cuatro de esta tarea al final.
`deploy/principal/lista_sql_claude_preferencias_internas_20261016.txt` contiene
solo esas cuatro para una copia que ya tenga instaladas las 78 anteriores;
no se combina con la lista completa.
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
  `VEC_USUARIOS_PREFERENCIAS_ENABLED=true`. Esta ruta se sirve desde el proceso
  combinado: Usuarios exige sus dos configuraciones, interna y externa, con
  certificados, cuentas, perfiles y conexiones separados. El certificado de
  Antonio se incluye en Usuarios interno; para completar Usuarios externo se
  prepara otra identidad de la copia. No se selecciona el montaje externo de
  Mi Bolsa en ese proceso. La configuración completa de Contratación temporal
  publica la clave de capacidad que usa la escritura interna.
- Auditoría: configurar `VEC_AUDITORIA_SELLADO_DATABASE_URL` con el LOGIN
  exclusivo del grupo `vec_auditoria_encadenador`. El proceso debe mantener
  activo su sellador; si el latido supera el plazo admitido, la auditoría
  rechaza la operación. No se actualiza ese latido por SQL.
- Datos de la copia: certificado, sujeto, cuenta y persona deben corresponder
  al contexto corporativo emitido por la aplicación. La candidatura vinculada
  debe tener al menos una participación vigente para poder guardar un orden.
  Renovar el certificado exige revisar esas referencias. Los certificados y
  las asignaciones de la copia tienen una vigencia máxima de 90 días.

Comprobar en la copia: arranque con el selector apagado y encendido; `GET`
propio 200 con apunte de lectura; `PUT` 201 con recibo; repetición 200 con el
mismo recibo; tras reiniciar aplicación y PostgreSQL, la misma versión, orden,
recibo y fecha. No se admiten contextos de otra candidatura ni sesiones
externas. Cuando GET llega a evaluar la concesión, una denegación registra el
intento de lectura; si falla antes el cotejo del candidato, el registro
compartido de sesión y contexto no expresa la acción denegada. `POST` responde 405.

## Estado (18/10)

Ensayado en una copia de HZ37 con main y las SQL de #1144, #1146 y #1155: las
cuatro migraciones se instalan una vez y la prueba focal pasa. El ensayo final
incluye las 66 migraciones de main, siete de F1-16, cinco de F1-04 y estas
cuatro; CT225 y CT227 se instalan sin el conflicto de preimagen anterior.
El consumidor central conserva la huella `a4649132…fb2`. La guarda de B116
exige nombre y huella exactos de la restricción anterior: acepta la preimagen
esperada y rechaza una con una condición adicional, sin sustituirla.
La prueba de recibo SQL usa un doble transaccional del consumidor; por sí sola
no comprueba la firma real. La identidad corporativa de Antonio, las dos superficies de Usuarios, la
concesión y el LOGIN se prepararon como datos de la copia, con vigencias de
90 días como máximo. El recorrido HTTP se completó con el certificado
corporativo y el binario de `97adc3c5a`:

- Lectura 200 y guardado 201 con recibo; repetición 200 con la misma versión,
  orden, referencia y fecha.
- Tras detener y arrancar la aplicación y PostgreSQL, lectura 200 con el
  mismo estado y repetición 200 con el recibo original.
- Sin el selector, arranque con salud 200 y ruta de preferencias 404; al
  reactivarlo se conserva el estado anterior.
- Recuperación de una respuesta fallida: antes de corregir la comparación
  temporal, SQL había confirmado el guardado y HTTP había devuelto 503.
  Repetir aquella misma petición después de la corrección devolvió 200 y
  recuperó su recibo, sin crear otra versión.

La comparación de fechas usa ahora el reloj posterior al retorno del
repositorio. Las pruebas focales aceptan una fecha registrada durante la
escritura y rechazan una fecha posterior a ese retorno. Dos revisiones
independientes comprobaron el cambio sobre `97adc3c5a`; las cuatro SQL y
las huellas del núcleo permanecen intactas. Las 19 pruebas del guion de copia
pasan con la lista completa de 82 migraciones.
