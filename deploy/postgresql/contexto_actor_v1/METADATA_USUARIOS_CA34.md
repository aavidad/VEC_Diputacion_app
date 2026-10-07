# Metadatos opacos de usuarios — CA34

CA34 conserva Persona y perfiles en CA. AUT43 recibe únicamente referencias y
versiones por dos funciones propietarias; no consulta tablas CA ni recibe
cuentas, procedencias, etiquetas o nombres. Reutiliza la separación de autoridad
del fragmento de E `dd56066` y elimina su búsqueda/página de personas y cuentas.

- `metadatos_persona_administrable_v1(persona text)` devuelve `{persona_ref,version}` o NULL si no existe la fila actual o se ha comprobado que Persona no está vigente.
- `metadatos_perfil_administrable_v1(persona text,perfil text)` devuelve `{perfil_ref,version,estado,vigente_desde,vigente_hasta}` o NULL si no pertenece a esa Persona.

Sólo los propietarios CA y AUT tienen EXECUTE. Los perfiles caducados o
revocados siguen consultables como metadatos de una Persona vigente: esta proyección no concede acceso.
Persona histórica queda fuera del primer conjunto; un fallo de proveedor se propaga y nunca se convierte en ausencia. Las versiones superiores
al entero exacto JSON rechazadas no se redondean.

El consumidor AUT43 limita el conjunto por asignaciones y ámbito, consume V3
y registra la lectura en la auditoría común antes de pedir estos metadatos,
dentro de la misma transacción. La versión de nombre usa el puerto propio CA32.
Estos helpers no se exponen al LOGIN ni a HTTP.

Estado: preparado; ensayo PostgreSQL y dos revisiones independientes pendientes.
No se ha instalado en la principal ni ejecutado DOWN.
