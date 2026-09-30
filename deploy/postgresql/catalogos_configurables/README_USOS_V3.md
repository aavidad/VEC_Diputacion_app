# Usos nominales de categorías RPT (000003 / AD3-126)

Estas migraciones preparan tres escrituras internas: reservar una categoría para
un uso, confirmar ese uso y cancelarlo. No conceden por sí mismas un permiso a
RRHH ni activan un consumidor. Se instalan después de la autoridad común
000001 corregida, las lecturas 000002 y AD3-117. Solo los ejecutores técnicos
existentes de Contratación temporal, Bolsa y Personal reciben `EXECUTE` sobre
las tres fachadas AD3. No reciben acceso a las tablas ni a las funciones core.

Las fachadas son
`vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada`,
`confirmar_uso_categoria_rpt_v3_atestada` y
`cancelar_uso_categoria_rpt_v3_atestada`. Comparten los once argumentos de
AD3-117: `material jsonb`, capacidad, decisión, motivo y contexto canónicos,
versiones de persona y perfil, payload, sobre, evidencia y raíz. Cada llamada
requiere una decisión V3 nueva. La acción es, respectivamente,
`vec.catalogos.categorias.reservar_uso`, `confirmar_uso` o `cancelar_uso`;
audiencia `vec_catalogos_configurables.usos_categorias.v1`, recurso `uso_ref`,
tipo `uso_categoria`, finalidad `vincular_categoria_a_operacion`, campos
`["recibo","uso"]` y obligaciones vacías. Los ámbitos exactos son
`catalogo_id`, `modulo_id` y `consumidor`. La huella del material se calcula
con los bytes de `p_material::text` de PostgreSQL antes de solicitar la
decisión; la fachada vuelve a calcularla.

La reserva acepta un objeto JSONB con exactamente estas ocho claves:
`catalogo_id`, `modulo_id`, `consumidor`, `uso_ref`, `categoria_id`, `version`
(número entero positivo hasta 2147483647), `huella_sha256` y
`reserva_recibo_ref`. Las operaciones terminales añaden exactamente
`terminal_recibo_ref`, `evidencia_ref` y `evidencia_sha256`. El estado terminal
lo determina la función invocada; no procede del material. La evidencia es
una referencia y una huella opacas de la operación del módulo propietario.
El `uso_ref` público admite de 3 a 160 bytes ASCII visibles (0x21–0x7e),
excepto `*`, sin recortar ni normalizar. Es el límite del recurso V3 nativo;
las funciones core antiguas conservan su contrato privado. Los recibos y la
referencia de evidencia mantienen su límite UTF-8 de 3 a 160 bytes.
Ese módulo debe comprobar el efecto, o la ausencia de efecto para cancelar,
antes de pedir la decisión V3. Este SQL no inspecciona sus tablas ni interpreta
la evidencia como prueba legal.

El LOGIN debe pertenecer a una sola familia técnica: Contratación temporal
produce `contratacion_temporal`, Bolsa `bolsa` y Personal `personal`. La
fachada compara esa familia con el material. Obtiene el actor de la decisión
atestada y la referencia versionada de motivo del motivo V3 consumido; no los
acepta como campos libres. Consume la decisión, registra la auditoría central,
aplica el core y relee el uso con su publicación histórica en la misma
transacción. Comprueba el documento canónico original, su SHA-256, catálogo,
versión, categoría y módulo. Si falla cualquier comprobación, se revierte
también el consumo V3.

La respuesta contiene los siete campos del recibo V3, `recibo_ref` y `uso`.
`uso` tiene `consumidor`, `uso_ref`, `categoria_id`, `catalogo_id`, `version`,
`huella_sha256`, `estado`, `revision`, `reserva_recibo_ref`,
`terminal_recibo_ref`, `reservado_en` y `terminal_en`. Las fechas se expresan
en UTC con microsegundos. No se devuelve la publicación: la decisión solo
permite recibo y uso. Un reintento de reserva con la misma identidad y recibo
puede devolver el uso ya confirmado o cancelado. Confirmar y cancelar son
terminales de revisión 2; ninguno reabre la identidad de uso.

La tabla común `evidencia_terminal` liga de forma inmutable consumidor, uso,
reserva, terminal, estado, categoría, publicación y evidencia opaca. Un
terminal anterior sin ese vínculo no se completa retrospectivamente. La
cancelación no demuestra por sí sola que el módulo propietario carezca de
efecto: el consumidor debe presentar una evidencia comprobada antes de emitir
su capacidad. Faltan la provisión positiva de perfiles y concesiones con
huella/CAS, el acto prospectivo de Contratación temporal y el plan durable de
Personal. Estas migraciones no acreditan instalación ni operación utilizable.

La lista `deploy/principal/lista_sql_trabajo_codexd_rpt_escritura_v3_20260930.txt`
describe el orden causal RPT para un clon nuevo. La lista H6 de 26 entradas ya
contiene roles, 000001, 000002 y AD3-117: al componer el clon se apartan esas
cuatro rutas de H6, se instalan una sola vez desde este árbol y se deja CT148
para después de AD3-126. Hay que comprobar la unicidad de todas las rutas.
No se reaplican 000001, 000002 o AD3-117 en una base donde ya tengan historia.
Los archivos DOWN son solo para un clon vacío y rechazan historia; no forman
parte del procedimiento de despliegue.
