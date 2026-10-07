# Exportación de servicios propios: preparación SQL

Personal32 y AD175 están preparadas en esta rama, sobre la ficha Personal22.
AD175 ya mide el núcleo y el CHECK de audiencias posteriores a AD211 en un
clon PG18. Su verificación estática es correcta; faltan las revisiones y el
ensayo de ambas migraciones, por lo que todavía no se declaran instaladas.
La lista causal está en `deploy/principal/lista_sql_codexb_exportacion_servicios_20261003.txt`.
Si una migración reescribe antes el núcleo o el CHECK, AD175 se detiene y debe
medirse de nuevo sobre la fuente que vaya a instalarse.

Personal32 añade `vigente_en date` y `conocido_en timestamptz(6)` al recibo existente.
Los recibos antiguos quedan con ambas columnas NULL. El único cambio en la consulta
Personal22 es guardar el corte que ya validaba, en su INSERT original. Se verifica
el cuerpo anterior, la inversión exacta del parche, metadatos, ACL y dependencias.
El hash del cuerpo procede del literal de Personal22 en Git; Dirección debe
cotejarlo con la definición real del clon antes de instalar Personal32.
No se modifica Personal22 instalada.

La función nueva `vec_personal.exportar_servicios_propios_empleado_v1` acepta
material y los diez argumentos V3 de la consulta existente. Revalida la persona,
el empleado canónico actual, una concesión de exportación nueva y el recibo/corte
exactos. Rechaza igual un recibo antiguo sin corte, un recibo ajeno o un corte
alterado. Lee sólo historia propia de Personal, con el mismo corte y proyección
de cinco campos que la consulta original. No crea una tabla de exportaciones.

Acción: `personal.registro_empleado.ficha_propia.servicios.exportar`.
Audiencia: `vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1`.
Finalidad: `exportar_servicios_propios`. Tipo y perfil técnico V3:
`exportacion_servicios_propios`. El grupo runtime sigue siendo el ejecutor propio
de Personal, con un único LOGIN nominal sin SET/ADMIN ni privilegios elevados.
La concesión de consulta no permite exportar. No se siembran cuentas, perfiles,
claves ni configuración de origen; la ausencia de provisión positiva fija/CAS y
del origen común de AD172 debe cerrar la operación.

El material incluye recibo, corte, idioma, referencia/versión/huella del catálogo
servidor y el actor actual original. La respuesta cerrada es `{corte,servicios,evidencia}`.
La evidencia conserva el recibo fuente y usa decisión, consumo y auditoría nuevos
para la generación. El adaptador Go debe serializar y validar el CSV antes de
COMMIT. Un fallo revierte consumo y auditoría de éxito; su intento se registra
por la autoridad común después del rollback. Esto no acredita entrega del fichero
ni convierte el listado en certificado oficial.

La prueba `pruebas_sql/exportacion_servicios_propios_000032_preservacion.sql` está
preparada, sin ejecutar: compara todos los recibos anteriores, sus cortes NULL,
la consulta completa y sus metadatos, y comprueba que el recibo sigue inmutable.
Aplica Personal32 una sola vez en un clon desechable con AD175 ya cerrada.
La prueba adicional `pruebas_sql/exportacion_servicios_propios_000032_acl.sql`
comprueba el consumidor exclusivo del propietario, la función nominal, ausencia
de permisos directos del runtime sobre el corte y las protecciones del recibo.
Faltan fixtures V3 reales sintéticos para probar exportación positiva, permiso
sólo de consulta, retirada de permiso, corte alterado, recibo ajeno y NULL,
falla de serialización antes de COMMIT y conservación de la cadena común.
Dirección conserva el ensayo PostgreSQL, las revisiones y el recorrido integrado.
