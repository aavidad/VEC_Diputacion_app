# Arranque central de Administración V3

AUT36 publica el perfil fijo Sistemas con una sola concesión: consultar los
registros técnicos existentes del recolector común. AUT37 prepara un arranque
privado con dos personas distintas de Aplicación y un perfil Sistemas de una
de ellas. Los tres perfiles conservan referencias separadas.

Este corte contiene SQL, catálogo y pruebas preparadas. No acredita instalación,
aplicación del plan, recorrido ni producción. El CLI V3 de K2 prepara y coteja el
plan; su opción de aplicar sigue cerrada hasta componer el proveedor real.

## Publicación de Sistemas

El contrato está en
[`acciones_sistemas_v1.json`](../../../data/catalogos/administracion/acciones_sistemas_v1.json).
AUT36 conserva sus bytes y SHA256 y publica la versión de rol, control y categoría
desde esos datos. No modifica la versión Aplicación v4 ni el validador de roles.
Amplía los dominios de categoría y clase de control para admitir Sistemas.

La concesión es `administracion.registros_tecnicos.consultar`, módulo
`administracion`, recurso `registro_tecnico`, finalidad `operacion_tecnica` y
garantía `alto`. Permite trece campos técnicos. Excluye mensaje libre, identidad,
recurso funcional, rutas, SQL y errores libres. Admite las familias
`vec.incidencia_tecnica.v1` y `vec.resultado_tecnico.v1`, que ya produce el módulo.

El dataset es una referencia opaca cuya organización y proceso pertenecen a una
configuración privada. El cliente no aporta su directorio ni acredita su propia
organización. Los filtros y límites están declarados en el catálogo; 32 por
defecto y 100 como máximo son ejemplos de configuración versionada. El cursor
debe quedar ligado al dataset, fuente, filtros y ámbito. La auditoría en la
transacción es un requisito del núcleo, aunque `obligaciones` esté vacío.

La publicación del rol no abre el lector futuro. Este requiere sesión Sistemas,
decisión V3 exacta, lector propietario, auditoría y confirmación de la transacción.

## Plan y aprobación

La entrada privada es
`vec_autorizacion.registrar_bootstrap_central_admin_v3(plan_canonico,huella_aprobada)`.
Su canon sigue `PlanBootstrapAdministracionV3` y los tipos de K2 en
`9b1862ee37b4de4cecb1e25063e6348e66e8c61d`. El plan contiene ámbitos explícitos,
fuentes, reparto aprobado y el catálogo usado; la aprobación externa por SHA256
se coteja además con la configuración del LOGIN.

El mapa privado del kit aprobado sigue siendo un mapa. Las huellas nulas, fuentes
pendientes o referencias propuestas no lo convierten en un plan aplicable.

El DBA registra fuera de Git un LOGIN exclusivo, con una sola membresía heredable
y sin SET ROLE en `vec_admin_bootstrap_central_v3_ejecutor`. El rol carece de
inicio de sesión, propiedad de tablas y pertenencia a otros roles. Solo recibe
USAGE del esquema, CONNECT de la base y EXECUTE de preflight y ejecución, sin
grant option. CREATE, TEMP y cualquier permiso adicional se rechazan incluso
cuando pertenecen a esos mismos objetos. Ningún LOGIN se siembra en la migración.

`config_bootstrap_central_admin_v3` liga ese LOGIN a proceso, SHA del plan,
preimagen completa, referencia y SHA de aprobación, tres fuentes originales y
vigencia. La configuración es inmutable. Revocar la membresía o dejar caducar la
configuración cierra nuevas ejecuciones y recuperaciones. Un archivo aportado por
el CLI no crea esa autorización.

## Fuentes y efecto

Las fachadas nuevas de CA y de Identidad pertenecen a sus propietarios y solo
puede invocarlas AUT. Conservan CA20 e IS9. Comprueban persona y cuenta vigentes,
enlace acreditado, procedencia maestra, certificados coherentes y tres referencias
nuevas antes de escribir.

El ámbito `organizacion_ref` se coteja contra la organización actual y la
procedencia maestra CA3. Su fuente es la referencia, versión y huella de esa
procedencia. La preimagen conserva también la versión organizativa actual.
`unidad_ref` se acredita mediante la fachada privada Personal31 sobre su
historia organizativa. AUT entrega la organización exacta, el ámbito original
y su vigencia; Personal devuelve nodo, revisión, fuente y generación. Su barrera
impide que una publicación o retirada invalide la fuente hasta COMMIT; un
snapshot cambiado propaga `40001`. CA no recibe acceso a Personal ni interpreta
una unidad por nombre de cargo o texto aportado. La tupla verificada queda
comprometida en la preimagen completa aprobada. Una unidad opcional puede omitirse
en la asignación; cuando se incluye, sus valores coinciden con el límite de
gobierno y su fuente real.

Los roles, controles, categorías y dimensiones se cotejan con AUT y su catálogo
publicado. El SQL no acepta concesiones del plan. La gobernanza existente se
compara con todos sus campos; un valor distinto produce conflicto. Solo se crea
configuración de rol cuando aún no existe y el plan aprobado la contiene.

La transacción exige `SERIALIZABLE`, escritura, UTC y ausencia de SET ROLE.
Valida las dos personas y los tres objetivos, registra el evento
`bootstrap_operador` en AD171, crea certificados mediante IS9, perfiles mediante
CA20 y asignaciones con sus sellos AUT24. Consume el bootstrap mediante un solo
CAS y conserva recibo y outbox. El emisor es técnico; no convierte el LOGIN en
una persona ni registra una decisión V3 ficticia.

Las tres asignaciones no pasan por la operación genérica AUT24 que impide dar
dos perfiles administrativos normales a una misma persona. Esa restricción sigue
vigente. La fuente de continuidad y doble control cuenta solo Aplicación mediante
categoría positiva, versión, SHA y control vigente. Los seis consumidores AUT24
se actualizan desde AUT37 conservando firmas, ACL y configuración; AUT24 no se
reaplica. La fuente histórica amplia permanece intacta para la guarda de persona
administrativa única. Una versión antigua sin categoría requiere adaptación
gobernada explícita antes de participar; no se reclasifica por nombre o prefijo. La fachada específica solo se usa en este arranque privado aprobado.

Una repetición exacta devuelve el mismo recibo cuando siguen acreditados el
operador y su aprobación. Coteja el acuse original mediante una fachada privada
de AD3; si falta el asiento común, deniega la recuperación. Otro plan no consume
de nuevo el bootstrap. `40001` se propaga para repetir la transacción completa;
el consumidor Go solo podrá confirmar después de COMMIT.

## Comprobaciones y límites

La lista causal propia contiene Personal31, AUT36 y AUT37, después de AUT33, CA20, IS9,
AUT24 y AD171 ya instaladas. No reaplicar esas dependencias ni usar DOWN sobre
historia conservada. Dirección ejecutará el ensayo sobre el clon autorizado y
cotejará la preimagen de datos, permisos y funciones.

Las pruebas preparadas comprueban la concesión Sistemas, canon Go/SQL, claves
adicionales, comodines, fechas con fracción y permisos del LOGIN, incluyendo contaminación CREATE del esquema, TEMP de la
base y GRANT OPTION de una función ya permitida. Una regresión posterior al
arranque real comprueba que el perfil Sistemas no aumenta la población Aplicación. La fixture de
ACL crea únicamente un LOGIN y configuración sintéticos dentro de ROLLBACK;
no fabrica personas ni modifica el control para simular un arranque permitido.

K2 cotejó de forma independiente el vector canónico con `CanonicoYHuella`; ambos
producen SHA256 `ca61b65da45901639332804d130884e765fbfe78af8ad23570152dd35bed8c54`.
El vector prueba serialización. Sus fuentes sintéticas no acreditan permiso.

Quedan pendientes las dos revisiones independientes del contenido final,
instalación causal, prueba con fuentes gobernadas, rechazos por cambio de estado,
replay y recuperación tras reinicio. No hay certificado, firma legal, envío ni
facultad de consultar expedientes por el mero arranque.
