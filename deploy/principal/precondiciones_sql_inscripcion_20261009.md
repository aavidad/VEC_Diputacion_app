# Precondiciones SQL de inscripción en Bolsa

La lista `lista_sql_inscripcion_solicitud_20261009.txt` contiene sólo las
migraciones nuevas de presentación y revisión. Antes de aplicarla hay que
comprobar en la base de destino qué piezas anteriores están instaladas.
Instalar únicamente las ausentes, una vez y en orden causal. No ejecutar DOWN
ni repetir UP sobre tablas con historia.

Convocatorias necesita sus roles de `bolsa_convocatorias/roles_up.sql`, la
revalidación de autorización
`bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql`
y BC1–BC8. BC8 exige los consumidores y roles de preparación de bases de
AD153, por lo que el tramo pendiente es BC1–BC7, AD153 y BC8. BC9 se instala
después. AD228 parte de AD227 y AD230 exactas; AD229 depende de AD228. CC11
requiere CC1 instalado y BC9 consume CC11. CA38 y las autoridades nominales
vigentes son precondiciones de la incorporación y del recorrido real.

El clon frío P1 de ensayo, SHA256 `323fb79f789120a4e1241f50b8365b6323500c94c0a6aeb516ebc67494b59356`,
no tiene BC1–BC8, la revalidación de Convocatorias ni AD153. CC1 está instalado
sin publicaciones. AD153 disponible en el repositorio exige una preimagen V3
anterior a la del clon postAD230: **no se puede aplicar allí tal como está**.
El dueño de Autorización debe aportar y revisar el tramo causal compatible
antes del ensayo completo. Esta comprobación no dice qué falta en otra base;
cada entorno necesita su inventario propio.

Las categorías, las políticas de presentación externa y de empleado, el ámbito
de gestión de cada convocatoria y el conjunto RRHH se publican en CC1 mediante
su procedimiento administrado con dos aprobaciones y control de preimagen. La
versión de la convocatoria fija el ID, la versión y la huella de ese catálogo.
Una versión ya publicada no se modifica para añadir una entrada: se publica
la nueva versión o rectificación gobernada que corresponda. Sin esas fuentes
y permisos nominales, el módulo debe permanecer cerrado.
