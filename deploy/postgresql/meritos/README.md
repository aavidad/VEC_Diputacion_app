# Registro Único de Méritos: persistencia RUM03

La migración `000001_registro_hechos_v1` conserva declaraciones propias,
rectificaciones pendientes y rechazos por un revisor distinto del declarante.
No acredita hechos, consulta Documentos, concede puntos ni exige empleo a la
persona. Persona y Documentos conservan la autoridad sobre sus referencias.
Estos registros no acreditan firma de una decisión administrativa. La
reautenticación reciente y la firma exigidas por la fuente de acceso interno
deben acreditarse en el circuito admitido antes de atribuirles eficacia real.

## Dependencias y orden

La entrega se apila sobre S1, que instala AD141 y su lector de Bolsa. Ese
consumidor es un prerrequisito y no se reaplica aquí. La lista causal propia
es roles de Méritos → AD142 → Méritos `000001`, recogida en
`deploy/principal/lista_sql_codexa_rum03.txt`. AD142 conserva los archivos
reservados y revisados, sin modificar sus bytes. El ensayo independiente
`pruebas_sql/000142_meritos_acl_contratos.sql` comprueba su contrato y seis
negativos, incluidos los tres rechazos del exterior antes del núcleo interno.

Los cinco roles son `NOLOGIN`, sin privilegios elevados. Cada identidad de
conexión tiene exactamente dos concesiones directas: `vec_meritos_ejecutor` y
uno de `vec_meritos_externo` o `vec_meritos_interno`, con `INHERIT TRUE`,
`SET FALSE` y `ADMIN FALSE`. Las cuentas se provisionan por el circuito central;
no se crean ni reciben permisos como consecuencia de una petición.

## Contrato del adaptador

`vec_meritos.operar_hecho_v1` recibe once parámetros: bytes exactos del comando
canónico y los diez parámetros del material atestado V3 común. Devuelve `jsonb`.
Se llama dentro de una transacción `SERIALIZABLE`, de escritura, con UTC y
límites de tiempo. El adaptador coteja el resultado antes de confirmar el commit.
Un resultado incierto o commit fallido no permite devolver un recibo.

El comando usa `domain.ComandoHecho`, esquema
`vec.meritos.hecho.operacion.v1`. La huella SHA256 se calcula sobre los bytes
recibidos, nunca sobre una nueva representación textual de `jsonb`. El recurso
V3 liga persona, versión esperada y esa huella. El motivo conserva catálogo,
versión, huella y entrada; la evidencia conserva ID y versión opacos.

La función revalida y consume V3 antes de leer negocio. Después bloquea actor y
clave, referencia del hecho y origen, y compara la idempotencia antes del CAS.
La declaración crea versión 1; la rectificación conserva declarante, persona,
fuente, origen y tipo, pasa a pendiente y elimina cualquier revisión anterior.
El rechazo conserva el contenido anterior, cambia a rechazado y añade una
revisión derivada del actor autorizado, motivo y fecha del servidor.

Una operación confirmada devuelve:

```json
{"codigo":"confirmada","auditoria_ref":"auditoria:nueva","anterior":null,"recibo":{}}
```

`recibo` contiene los campos de `ports.Recibo`: referencia, acción, actor, clave,
huella del comando, versión esperada, registro, fecha, auditoría y evento.
`anterior` es el registro de la versión esperada de la operación original.
En recuperación se consume una concesión nueva, se añade auditoría de acceso
y se devuelven el mismo recibo, fecha y antecedente histórico. No se añaden otra
versión ni evento de negocio. `auditoria_ref` del sobre corresponde al acceso
nuevo; la del recibo sigue siendo la original.

Los rechazos de negocio devuelven `codigo` igual a `conflicto_version`,
`clave_reutilizada` o `denegada`, con una referencia de auditoría conservada,
y `anterior` y `recibo` nulos. El adaptador confirma esa auditoría y traduce el
código a su error nominal. Un error de autoridad o integridad aborta la
transacción: no devuelve este sobre ni un recibo.

## Historia y comprobación

La escritura incluye versión, operación, auditoría minimizada, evento y recibo
en la misma transacción que el consumo V3. Las tablas propias fuerzan RLS y
carecen de privilegios de negocio para los roles runtime. La única función
runtime concedida es `operar_hecho_v1`. La historia impide UPDATE, DELETE y
TRUNCATE. La unicidad persona/fuente/origen/tipo evita duplicar el mismo hecho.
La clave de idempotencia se restringe al actor.

`pruebas_sql/estructura_acl_v1.sql` comprueba validación, ACL, RLS y ausencia de
acreditación por referencia documental. No fabrica material V3 ni demuestra
por sí sola el circuito criptográfico. Los ensayos de declaración, rechazo,
rectificación, reintento, revocación, CAS y rollback usan el emisor y consumidor
V3 reales sobre el clon aislado autorizado.

El DOWN exige superusuario, base desechable y todas las tablas sin historia.
Bloquea las tablas antes de comprobarlas. Una declaración o una auditoría de
rechazo basta para impedirlo. No se ejecuta sobre la historia conservada.
`roles_down.sql` requiere que el SQL propio y el consumidor AD hayan sido
revertidos y que no queden cuentas runtime provisionadas. Ningún DOWN usa
CASCADE ni borra dependencias ajenas.

El ensayo del 1 de octubre de 2026 en PostgreSQL 18.4 pasó la secuencia
roles UP → AD141 → AD142 → Méritos UP → estructura/ACL → Méritos DOWN sin
historia → Méritos UP, dentro de ROLLBACK. Devolvió
`MERITOS-ESTRUCTURA-ACL-OK` y `MERITOS-ROUNDTRIP-OK`, con código 0.
Se ejecutó por entrada estándar en el clon sintético aislado asignado, sin
red ni puertos, con 2 CPU, 2 GiB, 128 procesos y entorno limitado. Esto
acredita el esquema reversible y los controles estructurales; el recorrido
positivo con material V3 real queda pendiente de la comprobación conjunta.

El nuevo ensayo del 1 de octubre restauró una copia fría propia de la base
sintética, anterior a las candidatas. Conservó sus 82 migraciones y 71
expedientes. Aplicó una sola vez roles de Méritos → AD141 → AD142 corregida
(`88a790dcb87d95c4db5d80cd02d7401c1668751d`) → Méritos `000001`.
La instalación y la comprobación de estructura/ACL terminaron con código 0.
No se modificó la fuente original ni se reaplicaron sus migraciones.

El driver `internal/modules/meritos/adapters/postgres/ensayo` completó el
recorrido interno con PostgreSQL 18.4 y los adaptadores comunes reales:

| Operación | Resultado |
| --- | --- |
| Declaración propia y replay | `confirmada`, mismo recibo |
| Rectificación propia | `confirmada`, versión 2 pendiente |
| Rechazo por otro actor y replay | `confirmada`, versión 3 rechazada, mismo recibo |
| Rectificación con versión antigua | `conflicto_version`, sin recibo |
| Clave de declaración con contenido distinto | `clave_reutilizada`, sin recibo |

Cada operación revalidó la sesión y reconstruyó el contexto y la decisión
registrada. Firmó COSE con Ed25519, verificó la firma con el servicio común
y emitió y consumió una capacidad HMAC V3 nueva. El fixture OWNER inicial
usó ausencia y CAS exactos, perfiles fijos y claves sintéticas privadas;
comprobó que seguían presentes las 85.787 filas preexistentes. El driver no
publicó permisos ni alteró el gobierno durante las operaciones.

Después de parar y volver a crear el contenedor PostgreSQL sobre el mismo
PGDATA, un proceso nuevo repitió las siete operaciones con concesiones nuevas.
Conservó todos los códigos, recibos, fechas y antecedentes. Las huellas de
identidad del hecho, tres versiones, tres operaciones y tres eventos de outbox
permanecieron idénticas. Solo crecieron los accesos de recuperación (2 → 7) y
las auditorías (7 → 14), con referencias nuevas para cada acceso.

Los contenedores tenían red desactivada, raíz de solo lectura, 2 CPU, 2 GiB,
128 procesos y tiempo limitado. Las claves, sesiones, entradas y datos del
fixture quedan fuera de Git. Las pruebas focales Go, race y vet también
terminaron con código 0. La documentación del driver recoge el procedimiento
de reproducción y la distinción entre recibo original y auditoría del acceso.

El cierre causal se repitió en una copia fría posterior a S1, con AD141 y
Bolsa7 ya instaladas. Allí se aplicaron una sola vez roles de Méritos → AD142 →
Méritos1, sin reaplicar los dos antecedentes. El nuevo fixture OWNER conservó
los registros anteriores y avanzó configuración, raíz y checkpoint por CAS.
Las siete operaciones y su recuperación tras reiniciar PostgreSQL y el driver
volvieron a pasar con recibos, fechas e historia de negocio idénticos.

También se comprobó la lectura S1 después del reinicio conjunto, con sesiones
nuevas por su API y la configuración de confianza vigente. Conservó la clave
HMAC S1 original y sus concesiones: devolvió `obtenida`, y denegó al actor y a
la versión sin concesión. La fuente y los once accesos anteriores se
conservaron; la lectura autorizada añadió el acceso número doce. La definición
de la fachada AD141 mantuvo su huella y los 71 expedientes previos siguieron
presentes. No se publicaron nuevos roles, concesiones ni claves para S1.

Esto acredita persistencia y recuperación del circuito interno sintético.
La principal, el canal externo, la composición HTTP, el navegador, la firma
administrativa y la acreditación de hechos permanecen fuera de este ensayo.
Dirección reúne las dos revisiones sensibles del contenido exacto antes de
integrar; estos archivos no acreditan publicación ni producción.
