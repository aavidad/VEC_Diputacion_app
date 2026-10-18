# Registro de la comprobación del ofrecimiento

Tarea F1-09, segundo corte de servidor. Añade el adaptador HTTP y su montaje
opcional; depende del primer corte SQL de la PR #1190. No termina F1-09:
la proyección del contexto y del estado del acto se entrega en las PR #1201
y #1203. También faltan la pantalla F1-57 y la comprobación de CT antes de
reservar (F1-10).

CONFIG NUEVA: ninguna. B134 amplía la bitácora existente con la acción y
clase nominales de la comprobación; conserva las demás acciones y su historia.

## Activación

1. Dirección instala la lista completa de main posterior a HZ37 más las SQL
   propias en el orden único, con B134 después del primer corte F1-09.
   No instalar listas parciales
   ni repetir SQL. La bitácora distingue el intento de registrar hechos de
   un cambio de situación, también cuando el POST se rechaza antes de V3.
2. AD291, junto a F0-18/AD273, admite la acción, audiencia y origen nominal
   de consumo. En la lista única quedó después de AD288, B137 y AD293;
   la proyección debe quedar comprobada antes del recorrido completo.
3. Dos administradores publican la concesión
   `bolsa.comprobacion_ofrecimiento.registrar`, recurso `participacion_bolsa`,
   finalidad `gestionar_contratacion_temporal`, garantía alta, ámbitos
   `unidad_ref` y `ambito_ref`, campos vacíos y obligación `auditar`.
   La fuente de esa acción pertenece a #1190. Consultar motivos o cambiar
   situaciones no concede esta escritura.
4. Usar la configuración existente de Bolsa y su conexión nominal. Al preparar
   el material, la composición intenta publicar únicamente la audiencia
   `vec.bolsa.comprobacion_ofrecimiento.v1`, con clave y dominio propios.
   Si falla, conserva el resto del montaje y este POST devuelve
   `comprobacion_ofrecimiento_no_habilitada`. No se publican permisos al arrancar.
5. Reiniciar con vigilancia y comprobar salud, portal y operaciones anteriores
   tanto antes del alta central como con la activación completa.

## Contrato para la pantalla

POST `/api/vec/bolsa/bolsas/{bolsa}/candidatos/{participacion}/comprobaciones-ofrecimiento`,
con JSON, `Accept: application/json` e `Idempotency-Key`. El cuerpo lleva
`contexto`, `hechos` y `version_esperada`. Los tipos están en
`internal/modules/bolsa/ports/comprobacion_ofrecimiento.go`.

La pantalla debe recibir el contexto desde la proyección autorizada de las
listas existentes. #1201 aporta el contexto y #1203 el estado del acto; ambas
deben estar instaladas antes del recorrido completo. No se pide a RRHH que
escriba referencias, versiones o huellas ni se toma el filtro de navegación por autoridad.
Cada hecho lleva valor, referencia opaca de documento y huella. El servidor
añade actor y fecha de verificación; el cliente nunca aporta el resultado.

El adaptador normaliza el cuerpo con PostgreSQL sin consultar tablas de
negocio. Solicita V3 con ambos ámbitos y el atributo `material_sha256`.
B133 coteja la huella de ese contexto antes de guardar. Un registro nuevo
devuelve 201; la misma intención, 200 y su recibo original. Un contexto,
versión o material divergente da 409; sin permiso, 403. El recibo solo
entrega referencias, versiones, resultado, motivo y fecha. Si se pierde la
respuesta al COMMIT, se reconcilia una vez la misma intención y se coteja su
recibo. Si tampoco se confirma, devuelve `resultado_no_confirmado`; la pantalla
mantiene la misma operación para recuperar su resultado. Una fecha imposible
se trata como dato inválido, sin atribuirla a falta de habilitación.

Las pruebas focales de Go cubren la frontera HTTP, la audiencia, el material,
la denegación antes de escribir y el recibo. No acreditan una escritura real
en PostgreSQL. Quedan pendientes el ensayo con autorización real,
idempotencia, concurrencia y recuperación tras reiniciar con toda la activación.

Vuelta atrás: recuperar el binario anterior y su configuración conservada.
No ejecutar DOWN ni borrar comprobaciones o recibos. La falta del material
opcional mantiene cerrado este registro y conserva las otras operaciones.

Ensayo local del 18/10: candidata conjunta `169dca4ae`, con main `9f2a072c7`,
SQL `82a345058` y API `6528fed78`. Restauración HZ37 y sus permisos,
73 SQL completas de main y las cuatro propias instaladas una vez. B134 pasó
su guarda de cuerpo instalado, conservó los permisos y registró una denegación
nominal; rechazó las parejas de acción y clase incompatibles. B133 pasó cálculo,
canon compartido con Go y denegaciones. La aplicación compiló sin red y salud
y portal devolvieron 200 tras arrancar y reiniciar aplicación y PostgreSQL.
Chrome del sistema cargó el portal base. Las consultas de módulos y preferencias
siguen cerradas con 503 sin configuración: el ensayo no acredita una selección
ni un registro autorizado. El registro autorizado, su repetición y la concurrencia reales siguen
pendientes; AD291 se ensayó después en la candidata conjunta, sin publicar
todavía el permiso.


Ensayo interno posterior: candidata `350da6e75`, con las 78 migraciones de
main y las cuatro propias, instaladas una vez. Con la configuración interna,
salud, portal, cuadro de contratación y Bolsas devolvieron 200. Sin ella,
el mismo binario devolvió 200 en salud y portal, y 503 en Bolsas. Aplicación,
PostgreSQL y buzón local reiniciaron correctamente.

La petición real con `Idempotency-Key` detectó que el enrutador exterior no
reconocía este POST. `04ad4c8f8` corrigió el despacho por la ruta exacta;
la prueba focal y la revisión independiente resultaron favorables. Después,
el POST devolvió 503 con
`error.codigo=comprobacion_ofrecimiento_no_habilitada`, dejó un intento
nominal en la bitácora y ningún acto nuevo. Este resultado acredita el cierre
antes de habilitar la operación. No acredita una escritura autorizada.

La candidata `2f296ca09` incorpora el guion de copia de main `86c0bf1a0`.
Su código Go coincide con el ensayo anterior; las 82 fuentes SQL conservan
sus huellas, sin reaplicación. Compilación sin red, cotejo del binario y arranque
vigilado conformes, con 200 en las cuatro consultas anteriores. La prueba de
lista pasó sus 19 casos. Se guardaron el binario y el estado anteriores para
volver atrás. La lista de candidatos necesita recargar el fichero con el
material de la copia y una concesión de importación publicada; el kit ADMIN
común prepara esos actos. El registro positivo, la recuperación del recibo
y la concurrencia siguen pendientes de la activación gobernada completa.

La candidata conjunta `601686060` instaló 98 SQL en dos copias nuevas,
interna con RRHH original y externa con material nuevo. Las 97 SQL previas
son idénticas a `1ed8b0e42`, CT238 se añadió en su posición y Go no cambió.
Las focales AD291, AD292 y CT238 pasaron con ROLLBACK; ambas copias
reiniciaron con la comprobación pública del guion. AD291 ya está en esas
copias, pero no se publicó aún el permiso F1-09 ni se hizo un POST autorizado.
Siguen pendientes el recibo real, su recuperación y la concurrencia.
