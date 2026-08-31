# Ejecucion documental atestada V4 en PostgreSQL

Estado: implementacion tecnica de referencia y pruebas de integracion; **NO-GO
productivo** sin la separacion operacional, custodia de secretos y aprobacion
de Seguridad descritas en
[`atestacion_criptografica_decisiones.md`](../../../docs/portal_vec/atestacion_criptografica_decisiones.md).

Este directorio contiene el bootstrap de roles, las migraciones propias de la
ejecucion documental, las extensiones transaccionales del nucleo de
autorizacion y un runner efimero sobre PostgreSQL 18.4. La aplicacion no aplica
estas migraciones durante el arranque.

## Registro de autoridad de objeto esperado V1

La migracion aditiva
`migraciones/000002_registro_autoridad_objeto_esperado_v1` conserva la salida
especializada de `AutoridadObjetoEsperadoDocumentalV1.PrepararRegistro`. Recibe
los bytes de un documento JSON plano de diecinueve propiedades; limita el
tamano y calcula la huella de esos bytes antes de interpretar el `json`
original. Comprueba claves duplicadas y tipos antes de convertir a `jsonb`.
No es un codec publico ni permite reconstruir la capacidad Go.

La funcion coteja el TLV canonico completo del recibo material V2 y obtiene de
el, no de V4, la referencia durable, conector, efecto y pareja exacta de
objeto/version. La orden V4 se bloquea solo para confirmar la misma referencia
de efecto y conservar los compromisos P0-1 de plan autorizado y efecto
atestiguado. La caducidad de la capacidad V4 no se vuelve a evaluar: esa
capacidad ya fue consumida atomicamente por `000001` y no autoriza este nuevo
registro.

El cotejo TLV replica tambien la validez semantica canonica V2: maximos y
antipatrones de todos los alias, accion tecnica `escribir`, versiones y tamano
positivos, huellas no nulas, MIME, zona y estados cerrados, UTC representable a
microsegundos y retencion posterior coherente. La huella de manifiesto que
entrega la proyeccion debe coincidir exactamente con la huella de plan de la
orden V4; una discrepancia valida en forma se rechaza antes de persistir.

Cada pareja `efecto_ref`/`paso_ref` usa OCC de creacion `0 -> 1`. El primer
registro inserta en un unico `COMMIT` el estado terminal, un eslabon de la
auditoria global y un evento outbox; el mismo texto canonico devuelve
exactamente la respuesta del registro original y cualquier variante se
rechaza. Las tres tablas son privadas, append-only y fuerzan RLS.

Se persisten los bytes canonicos V2 —formados solo por referencias opacas y
hechos materiales minimizados— y todas sus huellas y ligaduras. El codigo HMAC
o COSE solo atraviesa la llamada para calcular longitud, huella y el sello de
atestacion definido por el puerto: no se guarda el codigo, un sobre completo
ni una clave. Tampoco se guarda la proyeccion JSON original.

`registrar_autoridad_objeto_esperado_v1` es `SECURITY DEFINER` porque la futura
identidad tecnica solo debe recibir `USAGE` del esquema y `EXECUTE`, nunca DML
sobre las tres tablas privadas. Tiene propietario y `search_path` fijos, y no
se concede a `PUBLIC`, al emisor de capacidad ni al ejecutor atestado. Por
tanto este corte SQL no puede ser invocado por un runtime con valores libres.
El siguiente corte debe aportar un adaptador Go y una identidad tecnica dedicada que
reciban la capacidad opaca, ejecuten `PrepararRegistro` con los verificadores
V2 durables y solo entonces llamen a esta operacion. Hasta ese corte, la
integracion funcional y la produccion permanecen en **NO-GO**.

El `down` se permite directamente cuando no existe historia. Con registros,
solo el opt-in de limpieza del runner efimero puede retirar una cola propia,
contigua y situada al final de la auditoria global; una historia intercalada o
una dependencia externa falla cerrada. Tras restaurar el eslabon anterior,
usa exclusivamente `RESTRICT`.

`roles_up.sql` exige superusuario antes de crear estado. `CREATEROLE`, incluso
combinado con la propiedad de la base, no es una autoridad de bootstrap valida.

## Retirada de roles

`roles_down.sql` solo se ejecuta despues de retirar las migraciones y todas las
cuentas `LOGIN`. Es una operacion de superusuario para una ventana de
mantenimiento del cluster. El orden de exclusiones es obligatorio:

1. bloqueo advisory del modulo;
2. `pg_authid` en `ACCESS EXCLUSIVE`;
3. `pg_auth_members` en `ACCESS EXCLUSIVE`;
4. prevalidacion completa;
5. retirada explicita de ACL, membresia estructural y roles.

La prevalidacion fija los cuatro OID, todos sus atributos —incluidos contraseña,
caducidad y ajustes globales o por base— y la unica membresia admitida:
propietario a migrador con `ADMIN FALSE`, `INHERIT FALSE`, `SET TRUE` y el
superusuario bootstrap del cluster como otorgante. Tambien inventaria `roleid`,
`member` y `grantor`; cualquier relacion adicional aborta antes de mutar.
PostgreSQL atribuye al bootstrap (OID interno 10) las membresias concedidas por
cualquier superusuario, aunque la operacion la ejecute otra cuenta DBA.

La guarda DDL se identifica por OID, no solo por nombre. El esquema y la funcion
deben pertenecer al mismo DBA que ejecuta la retirada, conservar sus ACL
exactas, y el disparador debe mantener evento, estado, conjunto exacto de
etiquetas y funcion exactos. Tambien se fijan lenguaje, `SECURITY DEFINER`,
`search_path`, volatilidad, paralelismo, aridad y el inventario completo de
objetos y usos dentro de la guarda.

Antes del primer `DROP` o `REVOKE` se inventarian igualmente las cinco
concesiones exactas de base, `USAGE` sobre `public`, el unico `EXECUTE` de HMAC,
las ACL de las demas clases de objeto y los privilegios por defecto. Se admite
el inventario vacio de un bootstrap que aun no alcanzo la primera migracion o
las dos filas globales exactas creadas por ella; ningun estado intermedio. Una
concesion adicional provoca un fallo sin retirar autoridad ni borrar
dependencias ajenas.

No se usa `CASCADE`. `DROP ROLE` y la transaccion completa fallan cerrados ante
cualquier dependencia no prevista.

## Prueba real

```bash
deploy/postgresql/ejecucion_documental_v4/probar_integracion.sh
```

Ademas de migraciones, ACL, RLS y pruebas Go, el runner comprueba:

- rechazo sin estado de un propietario de base con `CREATEROLE` pero no
  superusuario;
- retirada y reinstalacion limpias de un bootstrap interrumpido antes de la
  primera migracion;
- aplicacion, matriz adversarial y replay concurrente del registro de
  autoridad de objeto con sesiones separadas;
- OCC de creacion, terminalidad, ligaduras TLV V2, compromisos P0-1,
  minimizacion de atestacion y atomicidad estado/auditoria/outbox;
- rechazo efectivo del ejecutor V4, `DOWN` protegido con historia, limpieza
  explicita, reinstalacion y retirada final sin historia;
- cuentas `LOGIN` distintas para fuente, registro de autorizacion, emision y
  ejecucion, con comprobacion de ACL negativa sobre el nuevo registro;
- desaparicion exacta de las tablas nuevas y de los cuatro roles V4 al cierre.

`probar_integracion.sh` ejecuta el helper cerrado
`probar_baseline_000001.sh`. Entre ambos conservan la matriz del runner padre
`c499700`: relaciones futuras y `RESTRICT`, superficie minima de `pgcrypto`,
membresias y otorgante, atributos/ACL/DDL/default privileges, carrera real
`GRANT`/`roles_down` observada desde una conexion previa y ciclo con DBA
alternativo. Los dos scripts exigen PostgreSQL `18.4` y cada fichero permanece
por debajo del limite de 800 lineas.

El runner es la unica evidencia dinamica prevista para este corte. No se ha
ejecutado durante la produccion del commit: la integracion permanece bloqueada
hasta reproducirlo en PostgreSQL 18.4 desechable y obtener revision
independiente.
