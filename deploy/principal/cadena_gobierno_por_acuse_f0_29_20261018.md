# Comprobación del justificante de administración

F0-29 corrige la lectura de la cadena de gobierno ADMIN: un registro de otro
tipo posterior al acto dejaba vacío el tramo exportado. La consulta ligada al
acuse recupera el tramo sellado que lo contiene, incluso cuando después existe
otro tramo de gobierno. La consulta sin referencia conserva el comportamiento
anterior. No se repite el acto ni se cambian sus huellas.

## Activación

1. Congelar la candidata y conservar el binario y la copia anteriores. Crear
   una copia propia con PostgreSQL 18 y datos ficticios mediante
   `deploy/principal/copia_local.py crear --modulos`, con sus preparadores
   propios de roles y datos nominales, pasando explícitamente
   `--lista-sql deploy/principal/lista_sql_main_tras_hz37.txt`.
   La lista completa incluye main tras HZ37 y las SQL propias, con AD295
   en su posición exacta y una sola vez.
2. AD295 exige la definición instalada de AD235, su propietario y permisos.
   Amplía la misma función con una referencia opcional y retira la firma
   anterior. Solo el grupo lector actual puede ejecutarla; no cambia el núcleo.
3. Instalar primero la SQL y solo después la consola nueva: contra una base
   sin AD295 la consola nueva falla con un error genérico. Compilar la
   consola de esta candidata. Para comprobar el acto original,
   usar `vec-gobierno-usuarios-admin -fase verificar -config <configuración>
   -textos <catálogo> -timeout 1m -acuse <acuse original>`. Mantener el mismo
   directorio privado con su plan, material y acuse. La consola coteja esos
   archivos y el asiento del documento antes de emitir su resultado.
4. El acto debe estar sellado. Si aún no lo está, la consulta se rechaza sin
   atribuirle cobertura. Esperar el sellado de ese mismo acto y repetir solo
   la verificación. No ejecutar otra vez preparar ni aplicar.

## Pruebas y límites

El focal `ad295_cadena_gobierno_por_acuse.sql` reproduce los dos casos de B y R,
un acuse histórico seguido por otro gobierno, el modo sin referencia, una
referencia inexistente o mal formada y un asiento pendiente de sellado.
Usa los escritores de auditoría instalados, sin cambiar permisos ni guardas;
sus filas terminan en ROLLBACK y no acreditan un acto de administración.
`ad235_lector_gobierno_admin.sql` conserva los controles de lectura y denegación
con la firma vigente. Las pruebas Go comprueban la vinculación al acuse.

Ensayo del 19 de octubre, con main unido: una copia nueva instaló las 97 SQL
de la lista, con AD295 al final, y la aplicación respondió en sus consultas
públicas. La función quedó con la huella final que exige la migración y la
focal de AD295 terminó sin errores. Una revisión SQL independiente repitió la
lista completa en su propio PostgreSQL 18.4 y dio el visto bueno: reaplicar
AD295 se detiene en su guarda sin efectos.

En una copia anterior que conservaba un acto confirmado, la consola nueva con
`-acuse` devolvió `cadena_verificada` para el mismo plan, material y acuse que
antes daban cobertura vacía, sin repetir preparar ni aplicar. Administración
arrancó y abrió `/admin/usuarios/` con el certificado de ese acto. Siguen
pendientes la consulta de personas y el recorrido de administración completo.

Vuelta atrás: recuperar copia y binario anteriores. No ejecutar DOWN ni
reinstalar una migración con historia. El cambio no añade configuración de
arranque a la aplicación.
