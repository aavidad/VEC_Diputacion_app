# Consulta de puestos de Personal desde el proceso combinado

Esta candidata conecta el GET existente de Organización histórica con
`vec-server` combinado. El consumidor es la selección desde la bolsa constituida:
Contratación consulta el puesto por la interfaz de Personal y conserva su propia
unidad gestora. Este corte no acredita todavía que el ofrecimiento esté completo.
Todo el ensayo se hace en una copia local con datos ficticios.

## Qué reutiliza

Reutiliza la aplicación, el repositorio y el manejador HTTP de Organización
histórica, el proveedor de autorización nominal, el registro de intentos y el
consumo transaccional de Personal. La fábrica de `vec-interno` conserva sus
comprobaciones. No hay migración nueva ni acceso de Contratación a las tablas de
Personal. Deben estar instalados Personal10, AD51, la auditoría de intentos y la
revalidación corporativa que ya usa la sesión registrada.

## Preparar la configuración privada

1. Conservar la configuración y el binario anterior de la copia para la vuelta
   atrás. El directorio de material queda fuera de Git, con modo0700; sus
   ficheros, con modo0600 y sin enlaces.
2. Seleccionar un perfil de RRHH que tenga la acción
   `personal.organizacion_historica.consultar`, recurso
   `personal/organizacion_historica`, finalidad
   `consultar_organizacion_historica` y garantía suficiente. La concesión enumera los seis campos que consume
   Personal: `dotaciones`, `plazas`, `puestos_individuales`, `puestos_tipo`,
   `unidades` y `vinculos`. La asignación usa
   exactamente `organismo_ref` y `unidad_clave`, con los valores permitidos para
   la copia. El perfil y la cuenta deben corresponder a la misma persona del
   certificado. Se conservan los perfiles y los ámbitos de Contratación y Bolsa.
3. Preparar cuatro referencias publicadas de motivos: consulta, denegación,
   entrada inválida y falta de disponibilidad. Cada referencia conserva catálogo,
   versión, huella y clave. El montaje no publica motivos ni concede permisos.
4. Publicar la capacidad OH mediante el selector existente
   `VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED=true` y el publicador de la misma
   generación de idempotencia. Conservar coordenadas, versión, revisión, ventana y
   huellas. La clave privada del material debe coincidir con la derivación y con
   esa publicación; una huella por sí sola no acredita autorización.
5. Preparar `organizacion_historica_v3.json` y su archivo de capacidad con el
   formato privado existente: versión1, catálogo de motivos, motivo de consulta,
   capacidad y contextos admitidos por cuenta. Cada contexto selecciona perfil,
   versión, organismo y unidad; se comprobarán de nuevo en cada petición.
6. Preparar `organizacion_historica_combinado.json`, versión1, con
   `dsn_personal`, `dsn_fuente_autorizacion`, `dsn_registro_autorizacion`,
   `dsn_motivos`, `motivo_denegado`, `motivo_entrada_invalida` y
   `motivo_no_disponible`. Los LOGIN son propios de la copia, con TLS verificado,
   membresía única y sin permisos elevados ni escritura directa. Los grupos son
   `vec_personal_ejecutor`, `vec_autorizacion_fuente`,
   `vec_autorizacion_registro` y `vec_autorizacion_motivos_evaluador`,
   respectivamente. La auditoría usa su configuración nominal existente y los
   llamantes técnicos admitidos; no se sustituye por un registro en memoria.
   La CA de las conexiones es la de PostgreSQL, verificada contra el nombre
   del servidor; puede diferir de la CA de la web. Reservar dos conexiones
   adicionales para el pool de auditoría OH, además de las usadas por los
   consumidores ya montados; en el ensayo el límite nominal pasó de cuatro a seis.

Los datos ficticios de perfil, catálogo y motivos del ensayo se preparan solo en
la copia propia, con procedencia de semilla. No se presentan como actos o recibos
administrativos. Las sesiones, decisiones, consumos y recibos de la consulta los
producen los servicios reales. No se incluyen contraseñas, claves, certificados
ni identificadores de personas en este guion.

## Arranque y comprobación

Este montaje se limita al perfil positivo de desarrollo. En producción se usa
la cápsula del proceso `vec-interno`. La sesión de esta consulta se construye
con dependencias explícitas y conserva separado el perfil de Contratación.
Configurar bajo la doble llave existente de desarrollo:

```sh
VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED=true
VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR="$MATERIAL_OH"
```

El material se instala con la aplicación detenida y se arranca con vigilancia.
Si falla, volver al binario y configuración anteriores; no corregir ficheros que
el proceso ya leyó mientras sigue en marcha.

Comprobar una vez sobre la cabeza final:

- Sin selector ni material: la app arranca; sesión y Bolsa responden y solo el
  GET de Organización histórica devuelve503.
- Con configuración inválida: mismo aislamiento, con registro técnico acotado
  y sin datos privados en el error.
- Con configuración completa: GET200 desde la sesión certificada de RRHH,
  puesto de la petición en el organismo y unidad permitidos, recibo y auditoría
  SQL. El contexto se resuelve una sola vez por petición.
- Perfil o ámbito ajenos, vínculo corporativo retirado y capacidad de otra
  audiencia: lectura denegada y registrada, sin datos de otra unidad.
- Tras reiniciar aplicación y PostgreSQL: consultar el mismo puesto y corte,
  conservar la historia y los recibos anteriores y obtener la nueva lectura
  auditada sin duplicar actos de negocio.
- El proceso externo y el portal público conservan cerrada esta ruta interna.

## Resultado del ensayo local

Código comprobado: `22a5ae82e`. Sin selector y con directorio de material
inválido, salud, sesión y Bolsa respondieron200 y solo la consulta histórica
respondió503. Con el material completo, la consulta certificada devolvió200
con quince hechos de la unidad autorizada, incluido el puesto de la petición.
Se cotejaron en PostgreSQL el recibo, la decisión, la auditoría, la huella del
consumo y la cardinalidad. Otra unidad devolvió403 y dejó su apunte de frontera.
Tras detener la aplicación, reiniciar PostgreSQL y arrancar de nuevo, el mismo
corte devolvió la misma página con otro recibo; ambos siguen guardados. Chrome
del sistema obtuvo también200, puesto, recibo y auditoría. Esta comprobación de
Chrome cubre la lectura autenticada; el recorrido completo de Contratación
pertenece a la entrega de criterios.

La preparación detectó y corrigió tres diferencias de los datos de la copia:
las fechas de motivos deben llevar seis decimales, la concesión debe enumerar
los seis campos y la CA de PostgreSQL debe ser la de esa conexión. El código
corrigió además la versión de capacidad: el proveedor usa el mismo material al
que el gobierno acaba de asignar las coordenadas publicadas de Personal.

Las revisiones independientes finales y la comprobación focal de esa corrección
dieron su visto bueno. Pasaron las pruebas focales, incluido el cierre con una
consulta activa bajo el detector de carreras, y las comprobaciones locales de
Semgrep y gosec sobre lo cambiado. La CI de la cabeza final sigue en curso.

La selección completa sigue dependiendo de la necesidad RPT guardada por la
petición y de distinguir la unidad gestora de CT de la unidad del puesto.
