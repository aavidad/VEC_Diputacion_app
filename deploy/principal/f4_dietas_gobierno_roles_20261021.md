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

Las dos revisiones independientes de dirección dieron visto bueno sobre
`3b7db89ba`, sin hallazgos graves. La migración conserva exactamente esta
huella SHA-256:

`8348e9a0f586d19b42c04afd28a6086048c7ae04ade3cae59d644c975a0e6314`

Los cambios posteriores se limitan a esta nota, la prueba y su registro. El
SQL y el código Go revisados conservan sus bytes. La orden de dirección del
21 de octubre a las 10:56 pide mantenerlos mientras se completan las pruebas.

- `go test` y `go test -race`, con `-p 4` y filtro
  `GobiernoRol|RolNuevo|MeritosCartera`, en dominio y el manejador de
  Administración: positivos. `go vet` de ambos paquetes: positivo.
- Quince pruebas Python del conductor SQL: positivas. El registro conserva
  AUT107 detrás de AUT106 e incluye también la regresión AUT80: 33 casos.
- Ensayo local propio sobre la unión `5f4cea731`: esquema HZ37 sin datos,
  662 tablas comprobadas, preparación versionada y **148 migraciones**.
  AUT80 y AUT107 originales pasaron antes y después del reinicio, sin
  reaplicar SQL. Se retiró la copia al terminar.
- La prueba AUT107 ampliada rechaza diez variantes para cada rol de Dietas,
  incluidos un perfil existente, una entrada de Administración y una concesión
  de Bolsa. Conserva los positivos de Méritos con dos y tres concesiones y
  de un rol genérico de una concesión. Usa los validadores y el canon reales;
  solo sustituye el lector del catálogo en una transacción con `ROLLBACK`.
- CI de la unión `741190978`: ensayo PostgreSQL positivo con 148 migraciones
  y los 33 casos antes y después del reinicio. Incluye las nuevas variantes
  y las dos concurrencias del recorrido de fase 1. La migración conserva el
  ensayo local de sus mismos bytes; esta corrida completa de CI añade los
  casos pedidos sin ocupar el turno reservado para la copia del recorrido.
- Guarda de `search_path`: conforme. Semgrep local: tres reglas Go, cuatro
  archivos, sin hallazgos. gosec: 106 archivos, sin errores de carga; sus 25
  avisos corresponden a archivos anteriores ajenos al cambio. Ninguno afecta
  a los cuatro archivos de producción modificados.

El ensayo local usa las utilidades de
`deploy/postgresql/fase1/ensayar.py` para restaurar y retirar el contenedor.
Ejecuta solo AUT80 y AUT107; la batería completa corresponde a la CI. Estas
pruebas no acreditan publicación, asignación ni recibo de Administración.
El arranque de Dietas y la comisión desde su pantalla siguen pendientes.

La comprobación final de huella del cuerpo instalado y el reconocimiento de
una repetición idéntica quedan como mejora para una migración futura, según
la revisión de dirección. AUT107 no se modifica para añadirlas.

La asignación de Personal para Dietas sigue cerrada: no existe la vía de carga
del catálogo de validadores competentes ni su consumo transaccional. Según la
decisión de dirección del 21 de octubre a las 06:51, ese tramo se detiene. No se
abre la guarda ni se usa el preparador histórico. La consulta de esa asignación
requiere además un perfil de cuatro dimensiones; no cabe en el perfil de dos
dimensiones de la comisión propia. El recorrido web y la receta #1416 siguen
pendientes; no cambia el estado del tablero.
