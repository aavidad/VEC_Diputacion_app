# Permisos para preparar una comisión de servicio

Tarea: F4-DIETAS-COPIA. Este es el corte previo a la etapa opcional de la
receta de copia. No activa Dietas ni acredita el recorrido desde la web.

Se reutiliza el gobierno de roles nuevos de AUT60/AUT61 y la excepción cerrada
de Méritos de AUT80. La propuesta y la aprobación siguen los actos existentes
de Administración, con dos personas distintas, catálogo, huellas y recibos.

Se admiten únicamente estos contratos, sin sumar perfiles:

| Rol | Acciones | Dimensiones |
| --- | --- | --- |
| `persona_dietas_comisiones` | Relación propia de Personal para Dietas; crear y consultar borrador propio | `empleado_ref`, `persona_ref` |
| `persona_dietas_rutas` | Consultar catálogo y calcular itinerario | `ambito_ref` |

Cada contrato exige sus acciones en orden, garantía alta y los campos y
finalidades exactos ya declarados por Dietas. No se admiten obligaciones ni
campos añadidos, selecciones repetidas, otra clase de control ni dimensiones
ampliadas. Las pruebas Go y SQL comprueban estas restricciones. Los roles
ordinarios de una acción y la excepción de Méritos conservan su camino.

## Instalación y activación

1. Aplicar una vez AUT107 al final de `lista_sql_main_tras_hz37.txt`, después de
   las migraciones de main. Requiere PostgreSQL 18 y las dos funciones de
   gobierno instaladas con las huellas de HZ37 comprobadas en la migración.
2. La migración conserva propietario, permisos y configuración de las funciones;
   no concede ejecución nueva ni cambia el núcleo de autorización.
3. El corte no necesita variables, cuentas ni datos nuevos al arrancar. Con
   Dietas desactivado, su comportamiento se conserva. Activar Dietas espera
   a la etapa posterior de la receta y su comprobación de arranque.
4. Para publicar un rol, la receta posterior debe aportar sus entradas al
   catálogo por la vía existente y tramitar propuesta y aprobación. Después
   se realiza la asignación gobernada correspondiente. AUT107 no hace esos actos.
5. La vigencia del material de desarrollo será de entre 60 y 90 días.

La migración no se reaplica ni tiene reversión con borrado. Ante un fallo de
preimagen, la transacción se revierte sin reconstruir funciones. Si hace falta
corregir una instalación confirmada, corresponde otra migración revisada.

## Comprobaciones y límites

La prueba SQL usa el canon y el validador reales y sustituye solo el lector
del catálogo en una transacción que termina con `ROLLBACK`. No demuestra una
publicación, una asignación ni un recibo de Administración. Queda registrada en
el ensayo de fase 1 para comprobarla antes y después del reinicio.

Pendientes de anotar sobre la cabeza final: pruebas Go, ensayo PostgreSQL 18,
revisión de seguridad y las dos revisiones independientes de dirección.

La asignación de Personal para Dietas sigue cerrada: no existe la vía de carga
del catálogo de validadores competentes ni su consumo transaccional. Según la
decisión de dirección del 21 de octubre a las 06:51, ese tramo se detiene. No se
abre la guarda ni se usa el preparador histórico. La consulta de esa asignación
requiere además un perfil de cuatro dimensiones; no cabe en el perfil de dos
dimensiones de la comisión propia. El recorrido web y la receta #1416 siguen
pendientes; no cambia el estado del tablero.
