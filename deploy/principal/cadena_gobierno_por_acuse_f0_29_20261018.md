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
   La lista incluye las 81 migraciones de main y AD295, una sola vez.
2. AD295 exige la definición instalada de AD235, su propietario y permisos.
   Amplía la misma función con una referencia opcional y retira la firma
   anterior. Solo el grupo lector actual puede ejecutarla; no cambia el núcleo.
3. Compilar la consola de esta candidata. Para comprobar el acto original,
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

La candidata `ed5987b82` instaló una vez las 82 SQL en 76,02 segundos.
Pasaron los casos de AD295 y la prueba de permisos de AD235, ambos con
ROLLBACK. Esta última comprueba los 13 campos vigentes de la instantánea,
incluido el contador incorporado por AD266. La primera copia sin preparar los
orígenes nominales se detuvo en AD278; se descartó y no cuenta como ensayo.

En la copia propia que conservaba el acto confirmado se instaló AD295 una
sola vez. La consola nueva, con el perfil de desarrollo existente y `-acuse`,
devolvió `cadena_verificada` para el mismo plan, material y acuse que antes
obtenía cobertura vacía. No se repitió preparar ni aplicar. El arranque con
módulos y el reinicio de aplicación, PostgreSQL y buzón de la copia nueva
pasaron las cuatro consultas: salud, portal, contratación y Bolsas. La
aplicación mantiene su configuración de arranque; este corte no añade ninguna.

Dos revisiones independientes de la fuente son favorables. GoSec cargó los
dos paquetes completos: 80 avisos heredados y ninguno en las líneas añadidas;
Semgrep local y las focales Go fueron conformes. La función de verificación
conserva el límite de 100 registros y no trunca tramos para declararlos válidos.
La integración y la comprobación de la candidata de dirección siguen pendientes.

Vuelta atrás: recuperar copia y binario anteriores. No ejecutar DOWN ni
reinstalar una migración con historia. El cambio no añade configuración de
arranque a la aplicación.
