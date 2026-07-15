# Seguridad y permisos de PostgreSQL

Estado: especificacion vinculante. El primer adaptador de autorizacion, sus
migraciones reversibles y pruebas PostgreSQL reales estan implementados, pero
su composicion productiva permanece cerrada hasta incorporar atestacion
criptografica y consumo atomico dentro del repositorio que aplica el efecto.

Fecha de corte: 15 de julio de 2026.

Implementacion y procedimiento reproducible:
`deploy/postgresql/autorizacion/README.md`. Este primer corte no es una base de
datos completa del portal ni autoriza exponer nuevas rutas.

La puerta criptografica documental V4 dispone ya de implementacion y pruebas,
pero permanece cerrada mientras termina su reauditoria independiente. Es una
frontera deliberadamente estrecha para
`vec.documentos.ejecucion.ejecutar_plan_v4`; no autoriza reutilizarla sin mas
en Bolsa ni convertirla en una funcion generica configurada por el llamador.
Su contrato se documenta en
`docs/portal_vec/atestacion_criptografica_decisiones.md`.

### Estado comprobado de la evolucion 30 a 31

El 15 de julio de 2026 se ejecuto desde el agente director, sobre PostgreSQL
18.4, el procedimiento reproducible:

```bash
VEC_POSTGRES_PRUEBA_SOLO_SQL=1 \
  deploy/postgresql/ejecucion_documental_v4/probar_integracion.sh
```

La prueba aplico y revirtio las migraciones, con `teardown` completo, y
demostro positivamente:

- conservacion de una decision historica con las 30 propiedades anteriores;
- rechazo de cualquier decision nueva que omita el vinculo reforzado;
- registro de las 31 propiedades actuales sin eliminar campos en fixtures;
- bloqueo y cotejo exacto de las 25 propiedades de sesion y `ContextoActor`;
- rechazo si cambia el actor entre atestacion y reserva;
- rechazo si se revoca la sesion entre reserva y consumo;
- ausencia de acceso directo a tablas para las cuentas de ejecucion.

La funcion V4 de revalidacion solo admite su accion documental exacta y solo
puede ejecutarla el propietario tecnico del esquema documental. No se ha
concedido al adaptador general ni a Bolsa. La fuente local de sesion y actor
necesita aun alimentacion autoritativa desde el IdP y el maestro de personas;
sin ella, o sin atestacion COSE y raiz historica verificables, el perfil
productivo debe fallar cerrado.

La cadena de compilacion parte de Go 1.25.8 como compatibilidad minima corregida
y fija Go 1.26.5 para entregas. El minimo exacto impide que
`GOTOOLCHAIN=local` permita compilar con una revision 1.25 anterior vulnerable.
Go 1.22 se ha retirado porque ya no recibe correcciones;
la politica oficial mantiene las dos versiones principales mas recientes:
<https://go.dev/doc/security/>. La revision 1.26.5 incorpora correcciones de
seguridad publicadas el 7 de julio de 2026:
<https://go.dev/doc/devel/release>.

Las imagenes base se referencian por version y resumen criptografico. Esa
inmutabilidad no significa congelarlas: el proceso de dependencias debe proponer
el nuevo resumen cuando haya una reconstruccion o correccion, ejecutar todas
las pruebas y registrar la aprobacion antes de sustituirlo.

## 1. Regla de autoridad

PostgreSQL aplica la misma politica que el nucleo: de menos a mas. Una cuenta,
tabla, fila, columna o funcion no se puede utilizar si no existe una concesion
positiva y exacta. La falta de contexto, una clave desconocida, una politica
ausente, un error o una conexion con el rol incorrecto deniegan.

La base de datos es una barrera adicional, no el punto que decide los permisos
funcionales. El nucleo sigue autorizando accion, recurso, ambito, finalidad,
campos y obligaciones antes del caso de uso. PostgreSQL limita el dano de un
fallo posterior y hace cumplir invariantes, unicidad, concurrencia y atomicidad.

No se trasladan a la base nombres de rol recibidos del navegador, grupos de
Active Directory ni valores de cabeceras. Solo se usa contexto opaco que ya ha
sido autenticado y autorizado por componentes confiables.

## 2. Separacion de identidades tecnicas

No existira una cuenta universal de aplicacion. Como minimo se separan:

| Identidad | Uso permitido | Prohibiciones principales |
| --- | --- | --- |
| Propietario de esquema, sin inicio de sesion | Poseer objetos y politicas | No ejecuta la aplicacion ni migraciones ordinarias |
| Migrador | Aplicar una version de esquema aprobada | No atiende peticiones; credencial retirada al terminar |
| Portal exterior de Bolsa | Operaciones exteriores expresamente concedidas | Sin esquemas internos, Cronos, administracion ni auditoria completa |
| Gestion interna de Bolsa/RRHH | Casos de uso internos permitidos | Sin propiedad de objetos, claves, migraciones ni acceso a otros modulos |
| Trabajador de Bolsa | Trabajos concretos de bandeja de salida | Sin consultas interactivas generales |
| Exportador de auditoria | Leer el flujo o vistas minimas de auditoria | Sin modificar negocio ni borrar evidencias |
| Operacion y monitorizacion | Metricas y estado aprobados | Sin lectura general de datos de negocio |
| Copias | Funciones estrictamente necesarias para copia | Sin uso interactivo por la aplicacion |

Cada modulo desplegable usa credenciales propias. Las superficies exterior e
interna usan pools y roles de conexion distintos, aunque en la primera fase
compartan instancia. Cronos mantiene base, credenciales, red y copias exclusivas
dentro de Mulhacen; ninguna cuenta del portal exterior puede conectarse a ella.

Las cuentas de ejecucion no son `SUPERUSER`, no poseen objetos y carecen de
`CREATEDB`, `CREATEROLE`, `REPLICATION` y `BYPASSRLS`. Tampoco reciben los roles
predefinidos `pg_read_all_data` o `pg_write_all_data`: su alcance es demasiado
amplio y sus capacidades pueden evolucionar entre versiones. PostgreSQL
advierte expresamente que los roles predefinidos dan acceso privilegiado y que
sus permisos pueden cambiar:
<https://www.postgresql.org/docs/current/predefined-roles.html>.

## 3. Privilegios cerrados

La preparacion de cada base debe:

1. revocar a `PUBLIC` conexion no necesaria, creacion en esquemas, ejecucion de
   funciones y cualquier privilegio heredado que amplie acceso;
2. fijar privilegios predeterminados del propietario para que los objetos
   futuros tampoco nazcan accesibles;
3. conceder `USAGE` de cada esquema y operaciones de tablas, secuencias,
   vistas o funciones mediante listas positivas por identidad tecnica;
4. impedir a las cuentas de aplicacion crear, alterar, truncar o eliminar
   objetos;
5. conceder columnas concretas o vistas de proyeccion cuando un lector no
   necesite la fila completa;
6. verificar los privilegios efectivos tras cada migracion y fallar el
   despliegue si aparece una concesion no inventariada.

Revocar a un rol concreto no basta si conserva el permiso por `PUBLIC` o por
otra pertenencia. La comprobacion debe calcular el privilegio efectivo. Esta
semantica esta documentada por PostgreSQL en
<https://www.postgresql.org/docs/current/sql-revoke.html>.

Ningun adaptador usa SQL dinamico construido con datos de entrada. Valores,
identificadores funcionales y contexto se transmiten como parametros. Los
nombres de tabla, columna, ordenacion o funcion solo proceden de listas
positivas cerradas dentro del adaptador.

## 4. Seguridad por filas

Las tablas que contengan expedientes, solicitudes, documentos, recibos,
baremaciones, decisiones, notificaciones o datos personales habilitan seguridad
por filas (`ROW LEVEL SECURITY`) y la fuerzan también para su propietario
cuando este deba ejecutar comprobaciones. El rol de ejecucion nunca posee la
tabla.

PostgreSQL aplica denegacion predeterminada cuando la seguridad por filas esta
habilitada y no existe politica. Los superusuarios, las cuentas con
`BYPASSRLS` y normalmente el propietario pueden eludirla, motivo por el que
esas capacidades quedan fuera de la aplicacion:
<https://www.postgresql.org/docs/current/ddl-rowsecurity.html>.

Reglas para las politicas:

- se crean por operacion (`SELECT`, `INSERT`, `UPDATE` o `DELETE`) y por rol
  tecnico; no existe una politica general `USING (true)`;
- la fila debe coincidir con modulo, superficie, sujeto o ambito opacos que
  exige el caso de uso;
- `USING` y `WITH CHECK` se definen expresamente; poder leer una fila no implica
  poder crear otra ni cambiar su propietario;
- las politicas sensibles se combinan de forma restrictiva; no se depende de
  una suma permisiva accidental entre politicas;
- contexto ausente, vacio, no canonico o no correspondiente devuelve falso o
  provoca error y cancela la transaccion;
- tablas de auditoria y hechos probatorios son de solo adicion para las cuentas
  de negocio; no reciben `UPDATE`, `DELETE` ni `TRUNCATE`.

RLS no sustituye la separacion fisica de Cronos, el filtrado de columnas, el
cifrado ni la autorizacion del nucleo.

## 5. Contexto por transaccion y pools

Toda operacion comienza una transaccion y establece, mediante parametros y con
alcance local a esa transaccion, el contexto minimo ya verificado: superficie,
identidad tecnica, principal opaco, perfil activo, sujeto o representado,
ambito, finalidad, decision de autorizacion, sesion y correlacion. No se
establece contexto persistente de sesion, porque una conexion del pool sera
reutilizada por otras peticiones.

El adaptador comprueba que:

- la decision y la sesion siguen vigentes con la hora de PostgreSQL antes de
  confirmar el efecto;
- modulo, accion, tipo/referencia/huella de contexto, finalidad, campos,
  obligaciones y dimensiones de ambito coinciden exactamente con la operacion;
- asignacion actual, control de vigencia de la version exacta de rol y
  revision/huella del catalogo completo siguen coincidiendo con la instantanea;
- la revalidacion o consumo de la decision y el efecto de negocio suceden en
  la misma transaccion; no existe una ventana entre comprobar y confirmar;
- el contexto solo puede fijarse una vez y no puede ampliarse durante la
  transaccion;
- la identidad tecnica de la conexion es la esperada para el modulo y la
  superficie;
- toda transaccion termina con `COMMIT` o `ROLLBACK`, incluso si se cancela el
  contexto de Go;
- los limites de sentencia, bloqueo y sesion ociosa en transaccion son finitos;
- `search_path` es cerrado y las referencias a objetos de seguridad se
  cualifican por esquema.

El pool comprueba conectividad al arrancar, usa TLS autenticado según el perfil
de red, tiene limites de conexiones y vida medidos, y no incluye contrasenas en
lineas de comandos, trazas ni errores. Las credenciales proceden del gestor de
secretos y se pueden rotar por identidad sin recompilar.

## 6. Transacciones, concurrencia y bandeja de salida

### Precision temporal portable

La representacion canonica del dominio sera UTC con precision de microsegundo,
que PostgreSQL puede conservar en `timestamptz`. Una fecha con resto inferior
al microsegundo se rechaza en los contratos que forman huellas o se canoniza en
el unico borde de creacion antes de firmarla; nunca se redondea silenciosamente
al leer de la base. Las caducidades se comparan con hora durable y, ante una
conversion inevitable, se acortan en vez de ampliarse. Los campos usados para
consultar no se emplean para reconstruir una evidencia: se conserva tambien la
representacion canonica exacta cuya huella se registro.

Esta regla evita que una ida y vuelta Go/PostgreSQL cambie una huella, haga
fallar el control optimista o alargue una concesion por nanosegundos.

El adaptador confirma como una sola unidad:

```text
agregado + version OCC + hecho inmutable + auditoria + bandeja de salida
```

Si una version, huella, permiso, caducidad, reserva, unicidad o numero de filas
no coincide exactamente, se revierte todo. Una actualizacion que afecte cero o
mas de una fila nunca se considera exito. Los conflictos serializables se
traducen a errores cerrados y solo se reintentan cuando el comando completo es
idempotente.

La escritura de un efecto administrativo consume y vuelve a validar la
decision de autorizacion dentro de esa misma transaccion. Se bloquean y cotejan
su uso unico, asignacion, control de vigencia del rol y revision/huella del
catalogo antes del `COMMIT`; una revocacion o publicacion concurrente hace
`ROLLBACK`. Validar al principio del caso de uso o confiar solo en el TTL deja
una ventana TOCTOU y no es apto para produccion.

Las llamadas a S3, firma, antivirus, correo, Telegram, pasarela de pago u otro
sistema no se ejecutan dentro de una transaccion SQL abierta. Primero se
registra una intencion durable; un trabajador la reclama con arrendamiento
breve, usa una clave remota idempotente, registra el resultado y permite
consulta o reconciliacion tras una respuesta ambigua.

Los trabajadores pueden usar bloqueos de fila y `SKIP LOCKED` para repartir
una cola, pero no para omitir permanentemente trabajos. PostgreSQL explica que
los bloqueos de fila duran hasta el final de la transaccion:
<https://www.postgresql.org/docs/current/explicit-locking.html>.

## 7. Migraciones y operacion

- Las migraciones son versionadas, revisadas, con huella y ejecutadas por una
  tarea separada con la identidad de migrador.
- La aplicacion no migra al arrancar y no conoce la credencial de migracion.
- Se sigue expansion y contraccion: primero se añade compatibilidad, despues se
  cambia el trafico y solo en otra version se retira lo antiguo.
- Una migracion que cree un objeto crea tambien propietario, privilegios,
  politica por filas, indices, restricciones y pruebas correspondientes.
- El esquema no guarda binarios documentales ni secretos; conserva referencias
  opacas, huellas, versiones, estados y evidencia necesaria.
- Copias, WAL y restauracion a un punto se prueban. Restaurar datos sin claves,
  objetos o auditoria relacionados no se considera recuperacion completa.

## 8. Auditoria y privacidad

La auditoria funcional firmada y encadenada pertenece al modelo de negocio y se
escribe en la misma transaccion. El registro de PostgreSQL y, si se aprueba,
`pgaudit`, son controles operativos adicionales; no sustituyen esa evidencia.

Las sentencias, errores, estadisticas y nombres de conexion no contienen DNI,
documentos, motivos medicos, coordenadas, tokens, URLs firmadas ni secretos. El
acceso del DBA y del operador queda separado del uso funcional y se exporta a
un destino de auditoria con credenciales distintas.

## 9. Pruebas de aceptacion

- Una cuenta exterior no puede enumerar esquemas internos ni conectarse a
  Cronos.
- Sin contexto transaccional no se ve ni modifica ninguna fila protegida.
- Cambiar sujeto, perfil, ambito, finalidad o decision por un valor parecido,
  vacio o desconocido no amplía acceso.
- Un `UPDATE` no puede cambiar el propietario logico de una fila.
- El rol de aplicacion no puede desactivar RLS, alterar tablas, truncar
  auditoria ni concederse permisos.
- Una migracion que olvida politica o privilegios cerrados falla en CI.
- Dos confirmaciones concurrentes producen un solo efecto, una sola auditoria
  y un solo evento; la otra recibe conflicto cerrado.
- Caducidad o revocacion entre reserva y confirmacion impiden el `COMMIT`.
- Un error externo o una caida no pierde ni duplica la intencion durable.
- Los errores entregados a capas superiores son codigos cerrados y no contienen
  SQL, DSN, nombres internos ni datos personales.
