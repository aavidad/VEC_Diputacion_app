# Unidad propietaria para bootstrap ADMIN — Personal31

La fachada privada `vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)`
coteja una unidad de `org_nodo_historia` dentro de la transacción de AUT. Sólo
`vec_autorizacion_propietario` recibe EXECUTE; la aplicación y ContextoActor no
reciben acceso. No publica unidades, permisos ni otra auditoría.

El primer argumento es la organización exacta que ContextoActor ha acreditado.
El segundo conserva el ámbito original del plan: `dimension: unidad_ref`, un
solo valor de unidad y `fuente: {referencia,version,huella_sha256}`. `version`
corresponde a `org_nodo_historia.revision`, igual que la traza de Personal10;
referencia y SHA son los valores originales de esa fila. El tercero es la
vigencia solicitada, finita y futura, que no puede superar la de la unidad.

La respuesta contiene `esquema: vec.personal.unidad-bootstrap-admin.v1`,
`organizacion_ref`, `unidad_ref`, `nodo_ref`, `revision`, `catalogo`, `fuente`,
`acto_ref`, `vigente_desde`, `vigente_hasta`, `conocido_desde`, `valida_hasta` y
`generacion`. `catalogo` conserva referencia, versión, revisión y clave de
entrada; `fuente` conserva la terna original con la revisión de la fila.
AUT incluye esa respuesta en su preimagen aprobada. El canon del plan no cambia.

La consulta elige primero la última revisión de cada nodo candidato y después
coteja organismo, unidad, vigencia, conocimiento y retirada. No recupera una
revisión antigua para evitar una retirada o un cambio de ámbito. Una unidad
ambigua, desconocida, retirada o con versión/SHA divergentes se deniega.

Un singleton técnico propio cambia de generación mediante un trigger BEFORE
INSERT FOR EACH STATEMENT sobre la historia. Incluye la retirada por revisión
nueva. El helper bloquea ese singleton FOR SHARE antes de consultar historia,
sin bloquear adicionalmente toda la tabla. Un publicador posterior espera al
COMMIT de AUT; una generación cambiada desde la instantánea SERIALIZABLE produce
`40001`, que se propaga para reintentar la transacción completa. La barrera no
se sustituye por un bloqueo sobre una fila histórica inmutable.

Personal10 y sus protecciones deben estar instalados. La lista causal incluye
únicamente Personal31; AUT37 la consume después. No se modifica Personal11,
Personal30 ni la historia conservada. La generación inicial no acredita una
unidad: la instalación no siembra fuentes, nodos ni datos personales.

El productor no ha ejecutado PostgreSQL ni Go. Quedan dos revisiones sensibles
sobre el hash final y el ensayo coordinado por Dirección. Los casos positivos
requieren una unidad sintética gobernada que exista en la fuente; si falta,
se informa la omisión y no se inserta una unidad fingida para superar la prueba.

## Pruebas preparadas

`pruebas_sql/unidad_bootstrap_admin_000031_negativas.sql` comprueba ACL del helper
y del singleton, trigger habilitado BEFORE INSERT STATEMENT, ausencia de unidad,
comodines, versión cero y singleton ausente, todo en ROLLBACK.

`pruebas_sql/unidad_bootstrap_admin_000031_concurrencia.sql` se ejecuta en dos
terminales del mismo clon desechable con `lado=lector` o `lado=publicador`.
El lector recibe `organizacion`, `ambito` JSON y `hasta` desde el fixture
sintético gobernado. No se imprime el tuple. El publicador usa un INSERT de cero
filas que ejecuta el trigger real sin añadir unidades ni retirar historia;
esta prueba acredita la barrera técnica y no una publicación de negocio.

Se realizan estos órdenes, cada uno en una transacción nueva:

1. `caso=pre_guard, fin=commit`: iniciar lector y fijar instantánea; confirmar el
   publicador; continuar lector. La llamada debe propagar `40001` antes de dar
   un resultado y el lector hace ROLLBACK.
2. `caso=lector_primero, fin=rollback`: el lector coteja el tuple auténtico y
   retiene el guard; iniciar publicador. Debe esperar el bloqueo del guard
   hasta que el lector confirme. Después el publicador revierte su generación.
3. `caso=publicador_pendiente`: iniciar lector hasta su primer prompt, iniciar
   publicador y retener su INSERT, continuar lector para que espere el guard.
   Si el publicador hace `fin=commit`, el lector debe obtener `40001` y revertir.
   Repetir con `fin=rollback`: el lector continúa con la fuente anterior y no
   hay cambio de generación comprometido.

Dirección debe observar la espera real y `pg_blocking_pids` del publicador o del
lector según el caso, antes de liberar el otro terminal. Los prompts no prueban
por sí solos que exista un bloqueo. Los límites son 20 segundos para el bloqueo
y 30 para cada sentencia; no dejar terminales pendientes ni otros escritores
activos. En el caso de confirmar generación cambia únicamente el contador del
clon desechable, por lo que no se usa la principal conservada.

La prueba favorable se omite si no hay unidad gobernada. También se preparan los
casos de fuente divergente y retirada mediante
`pruebas_sql/unidad_bootstrap_admin_000031_fixture_gobernado.sql`; no se
recupera una versión vieja para hacerlos pasar.
