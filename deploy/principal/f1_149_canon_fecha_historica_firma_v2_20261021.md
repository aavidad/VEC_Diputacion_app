# F1-149 — Fecha histórica de la función anterior de firma

La función anterior de registro de firma conserva la comparación de fechas como
texto que CT259 corrigió en la versión actual. CT260 aplicará el mismo cotejo
como instante a `registrar_firma_verificada_v2`. No concede ejecución a nadie:
la versión anterior conserva sus restricciones actuales.

## Activación preparada

Usar una copia local nueva de la principal y la lista SQL canónica de esta rama.
CT260 debe quedar después de CT259 y del resto de las migraciones de main.
La preimagen instalada de la función anterior tiene SHA256
`49cf7bf45eb803baf87abee9556fe1417c61de4cf6daa8b0865f208e25ee6bd0`.
El cuerpo preparado reproduce exactamente la huella de `pg_get_functiondef`
que se leyó en la copia propia antes de retirarla. Se ha cotejado desde CT209
normalizando solo las comillas de `search_path` y el terminador externo, como
los representa PostgreSQL; no es una nueva extracción de una base viva.
La migración comprobará propietario, permisos y configuración antes de cambiar
la función y rechazará cualquier divergencia. El ensayo independiente debe
confirmar esos metadatos.

No necesita configuración, actos ni datos nuevos. El arranque conserva la
configuración existente. Dirección organiza el ensayo y dos revisiones
independientes sobre la cabeza final; la prueba de comparación aislada no
sustituye a una firma completa.

## Reconocimiento de la migración instalada

CT260 solo se aplica sobre su preimagen exacta, igual que CT259 y el resto de
migraciones CT: si se reaplica, falla cerrada sin tocar nada. Una primera versión
reconocía también la postimagen con metacomandos de psql (`\gset`, `\if`), pero
la receta de copia local solo admite `\set ON_ERROR_STOP on` y `\restrict` en las
migraciones versionadas, así que se retiró ese reconocimiento.

La prueba histórica de CT219 se ajustará a las postimágenes vigentes de ambas
funciones. Conserva el resto de los cotejos de permisos y del plan de firma.

Postimagen prevista de v2: SHA256
`e25c5f2396341320420d8dced263eaff3d7c50114db1b784751368a242fdbac0`.
Prueba registrada: `deploy/postgresql/contratacion_temporal/pruebas_sql/ct260_canon_fecha_historica_firma_v2.sql`,
con los siete casos equivalentes, diferentes, nulos y ausentes de CT259.
La prueba histórica CT219 conserva su dependencia de un plan publicado; no se
añade al registro de la base vacía ni se afirma ejecutada por esa puerta.
