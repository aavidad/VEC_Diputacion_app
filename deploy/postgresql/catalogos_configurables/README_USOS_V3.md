# Usos nominales de categorías RPT (000003 / AD3-126)

Estas migraciones preparan tres escrituras internas: reservar una categoría para
un uso, confirmar ese uso y cancelarlo. No conceden por sí mismas un permiso a
RRHH ni activan un consumidor. Se instalan después de H6 completo, AD3-132,
la autoridad común 000001 corregida, las lecturas 000002 y AD3-117. Solo los ejecutores técnicos
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
se aplica después de H6 completo y AD3-132, en este orden:
roles → 000001 corregida → 000002 → AD3-117 → 000003 → AD3-126.
El paquete H6 excluye las rutas RPT. Que 000002 y AD3-117 estén publicados en `main` no acredita su instalación: Dirección debe
comprobar la preimagen de la base y aplicar cada ruta una sola vez. CT148 queda
para después de AD3-126. No se reaplican migraciones con historia. Los archivos
DOWN son exclusivos de un clon vacío y no forman parte del despliegue.

## Preimagen y reconstrucción del núcleo

Las funciones nuevas de 000001, 000003 y AD3-126 fijan
`search_path=pg_catalog, pg_temp`. 000002 y AD3-117 conservan su definición
publicada. AD3-126 recibe el núcleo postH6/postAD117, incluida la consulta
de firmas de AD3-125, con
`search_path=pg_catalog` y `lock_timeout=2s`, valida su cuerpo completo y
reconstruye únicamente el perfil nominal de usos y la configuración de búsqueda.
Su DOWN exige la postimagen completa y devuelve la definición y configuración
postH6/postAD117 anteriores, incluida AD3-125. No retira una audiencia si
conserva claves ni revierte instalaciones con historia.

Las guardas se fijaron con una captura de PostgreSQL 18 tras las 62 SQL
de H3/H4/H6 (8 + 9 + 45), incluida AD3-125, y las cinco primeras SQL RPT.
La captura completa tiene SHA-256
`3421e7310dcd2505fff163d17d074ea431445763faa140aae6c5e74634a07a37`.
En ese clon se retiró TEMP de PUBLIC con autorización de laboratorio; no se
ejecutó AD3-132. Esa retirada no cambia el núcleo ni el CHECK. El CHECK se
captura con `pg_get_constraintdef(oid,true)`, igual que en las guardas;
la serialización predeterminada tiene otra huella.
Las postimágenes se calculan con las inserciones nominales exactas de AD3-126:

| Huella SHA-256 | Preimagen postH6/postAD117 | Postimagen AD3-126 |
| --- | --- | --- |
| Cuerpo `prosrc` | `8cda19bc0ab03f811386f7ec6b54a4662b68988590538d2857f48691c8c3c842` | `848799985debd0b736a3c281b0a3a3635e6182bd78789fb364121fe7f78d8a06` |
| `pg_get_functiondef` | `334d3a9d8397de1a37ca559ba12e0955510649da624b0ca3bdbab20a5729839f` | `3a06d11882d7b1ed5d3256f0547debf672975dce3c27060118bf169ce6bed692` |
| Definición del CHECK de audiencias | `cd92724aeb8e8baf8819be986ea23a215104619945944fc709352a5a0e5ec982` | `4fef385ffcee91a94046b1dda4368d3b85630df12d251384fd9d723352c8fdb8` |

Las guardas también comparan todos los metadatos de `pg_proc`, propietario,
ACL, lenguaje, esquema y dependencias de `pg_depend` y `pg_shdepend`.
El OID de la función se resuelve en la base concreta y debe conservarse dentro
de la transacción; no se importa el OID del clon de captura. La postimagen solo
permite cambios en `prosrc` y `proconfig`. Para el CHECK, DROP/ADD cambia el OID
propio y `conbin`; el resto de sus metadatos y dependencias debe conservarse.
Una definición parecida o unas marcas coincidentes no bastan para instalar.

Antes de integrar, el ensayo PostgreSQL debe cubrir UP/DOWN en el clon vacío,
las pruebas SQL de contrato y dos rechazos: un cambio del cuerpo del núcleo fuera
de las marcas nominales y una audiencia adicional en el CHECK. Ambos deben
producir `55000` sin dejar funciones, configuración, ACL ni audiencias parciales.
Las sondas SQL preparadas comprueban configuración y postimagen completas;
la prueba positiva de V3 exige decisiones firmadas por el circuito existente.
Este parche no acredita esos ensayos, una instalación ni un recorrido de usuario.
