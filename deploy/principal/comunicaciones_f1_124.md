# Consulta de comunicaciones de la ficha

Tarea F1-124. La ficha de Contratación temporal consulta
`GET /api/vec/contratacion-temporal/expedientes/comunicaciones` con la sesión
ligada al certificado de RRHH. Un expediente sin comunicaciones devuelve una
lista vacía; el vínculo con una emisión de Bolsa no es requisito de lectura.

## Activación

1. Crear una copia propia con la receta de `LEEME_copia_local.md`, apartados
   2 y 4, `--modulos`, el material original de RRHH y la lista
   `lista_sql_main_tras_hz37.txt`, que ya incluye CT248. Para una copia
   existente donde CT248 aún no esté instalada, la lista aislada es
   `lista_sql_codexy_comunicaciones_f1_124.txt`; no combinar ambas listas.
2. Instalar juntos CT248 y el binario de esta entrega. CT248 exige la
   definición instalada de CT231 y conserva propietario, permisos y
   configuración. Rechaza una segunda instalación. No requiere datos ni
   configuración nuevos y no se aplica sobre otra copia o servicio.
3. Abrir la ficha como RRHH. Consultar las comunicaciones sin vínculo y
   después de vincular una emisión mediante la operación existente de Bolsa.
   Ambas lecturas deben responder 200. El vínculo no fabrica una comunicación
   ni sustituye el llamamiento por la vía O6 del caso 4.0.
4. Consultar la ficha y sus comunicaciones al mismo tiempo; reiniciar la
   aplicación y PostgreSQL con la orden `reiniciar` de la fábrica y repetir
   las lecturas. Deben conservar sus datos y recibos originales.

Sin CT248, el binario sigue arrancando, pero no puede recuperar una carrera
que SQL haya traducido al error genérico anterior. CT248 sin el binario
tampoco basta: conserva el código original del fallo, pero el lector anterior
no lo reintenta.

## Qué se corrige

La composición incorpora el perfil estable de lectura de F1-114: cuadro,
detalle y descarga usan las tres concesiones que ya tenían, con la misma
persona y cuenta. Su asignación deja de cambiar en cada consulta. En la copia
de desarrollo se prepara con la vía existente de perfiles fijos; es material
no autoritativo y no acredita provisión por Administración para uso real.
Una revocación sigue denegada y el arranque no la reactiva. No añade
configuración ni SQL de permisos.

La consulta de SQL atrapaba los conflictos de serialización e interbloqueos
(`40001`, `40P01`) y los traducía a `P1405`, el mismo código de algunas
inconsistencias. CT248 conserva los dos primeros. Go reutiliza la política
común de reintentos con espera, límite de intentos y cancelación por contexto.
Cada intento termina su transacción antes de obtener otra autorización;
consumo y auditoría se confirman junto con la lectura. No se devuelven datos
del intento abortado.

Los bloqueos `55P03` siguen traduciéndose a `P1405`, sin reintento. Las
inconsistencias, denegaciones y fallos de conexión tampoco se reintentan. No se añade
ninguna concesión ni se transforma un fallo en una lista vacía.

## Validación

La regresión focal reproduce los dos códigos en configuración, consulta y
confirmación: falla con el lector anterior y comprueba recuperación con una
autorización nueva después del rollback. También comprueba que los errores
de permisos, datos, bloqueo y conexión no se repiten.

La reproducción HTTP combina consultas de ficha y comunicaciones. La copia
anterior registró `CT140: consulta transitoria` y devolvió 503 en las
comunicaciones. Los resultados de la candidata, el reinicio y las revisiones
independientes se adjuntan a la PR. Los registros privados del ensayo quedan
fuera de Git.
