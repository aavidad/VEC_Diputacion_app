# F1-47: no incorporación y revisión por otra persona

La candidata conserva el formulario y la operación de no incorporación.
Corrige el idioma del control de archivo y permite que otra persona de RRHH
abra la propuesta desde un enlace, sin acceder al cuadro general de expedientes.

## Activación

CONFIG NUEVA: ninguna. Usa los selectores, catálogos y conexiones existentes.
SQL nueva: CT242, al final de `lista_sql_main_tras_hz37.txt`.

1. Instalar CT242 una sola vez, en el orden de la lista. Amplía la función
   `consultar_cese_cierre_expediente_v1` con versión, fase, estado y número
   visible actuales. Conserva propietario y permisos; coteja la preimagen
   instalada. No reaplicar ni hacer una reversión de SQL.
2. Publicar juntos el binario y los estáticos de esta candidata, reiniciar y
   recargar el portal. La versión web se calcula con el mecanismo de F0-31.
3. Conservar el circuito de Contratación: conexiones nominales, gobierno,
   catálogos y permisos de RRHH. Deben estar activos
   `VEC_CT_SEGUIMIENTO_CESE_ENABLED` y
   `VEC_CT_INCORPORACION_ACREDITADA_ENABLED`, los tres catálogos del seguimiento
   y `VEC_BOLSA_REGLAS_SOURCE_PATH` con sus consecuencias de no incorporación.
   También se necesita la conexión exclusiva del relevo de no incorporación.
4. La regla vigente `c22.no_incorporacion` determina los motivos y la segunda
   revisión. Los permisos los concede el circuito existente; el enlace no
   concede ninguno ni contiene credenciales.

Para la copia local, crear una instancia propia con `copia_local.py crear
--modulos --segundo-rrhh --rrhh-original <material-propio>` y la lista SQL
principal. Usar productores privados de roles y datos propios, puertos libres
propios y el catálogo de reglas de Bolsa antes del primer arranque. La segunda
persona ficticia se prepara mediante la semilla existente de F0-28; eso no
acredita un acto de Administración. Véase `LEEME_copia_local.md`.

## Recorrido

1. Con un expediente en nombramiento, aceptación vigente y sin incorporación,
   la primera persona abre «Seguimiento del nombramiento» en su ficha. Indica
   motivo, fecha de notificación, referencia y archivo de la resolución. El
   archivo permanece en su equipo; solo se registra su código de comprobación.
2. Pulsa «Proponer que no se incorpora» y conserva el recibo. La propuesta
   ofrece «Abrir revisión para otra persona de RRHH».
3. La segunda persona abre ese enlace con su propio certificado. La página
   usa `POST /api/vec/contratacion-temporal/seguimiento-cese` para consultar la
   propuesta y la versión actual. No consulta el cuadro ni el detalle general.
4. Comprueba motivo, fecha y resolución y confirma o rechaza. El cliente usa
   `POST /api/vec/contratacion-temporal/no-incorporaciones`. El servidor exige
   personas distintas y coteja de nuevo propuesta y versión antes del efecto.
5. Una confirmación muestra su recibo. El expediente vuelve a fiscalización y
   Bolsa recibe la consecuencia prevista por su regla. Reiniciar aplicación y
   PostgreSQL; reabrir el enlace con la segunda persona y cotejar el mismo
   recibo, fecha y estado, sin repetir el registro.

El segundo acceso queda limitado a la propuesta pendiente o al resultado que
esa persona confirmó, sin incorporación, cese ni otros antecedentes. Comprueba
sesión, certificado, permiso y expediente exactos y registra la decisión de
lectura antes de entregar los datos. No habilita cese, modificación ni GINPIX.

## Recuperación y límites

La página conserva el recibo si la decisión se registró y falla la consulta
posterior. Ofrece reintentar la lectura, sin repetir automáticamente el acto.
El recibo de una confirmación se recupera al reabrir; el contrato existente no
ofrece esa recuperación del rechazo tras cerrar la página. El rechazo conserva
su recibo mientras permanece abierta. No se amplía ese contrato en este corte.

El arranque, las dos personas, el recibo real, la consecuencia en Bolsa y el
reinicio de esta candidata siguen pendientes de comprobar. Las capturas
anteriores de `docs/operacion/capturas/f1-47/` muestran el formulario con datos
preparados y no acreditan ese recorrido.

Pruebas focales de interfaz: `seguimiento-no-incorporacion.test.mjs`,
`seguimiento-cese.test.mjs` y `revision-no-incorporacion.test.mjs`.
Go: selección de la hoja, registro de la lectura y proyección de seguimiento.
SQL: `pruebas_sql/ct242_lectura_revision_no_incorporacion.sql`, con un LOGIN
miembro del ejecutor y un expediente ficticio, en una copia desechable.
