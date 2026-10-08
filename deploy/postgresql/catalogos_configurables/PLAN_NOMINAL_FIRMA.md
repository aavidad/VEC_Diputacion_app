# Plan nominal de firma CT: contrato CC7

CC7 conserva en CatalogoConfigurable las versiones del plan de competencia de
firma. La autoridad de Contratación temporal recibe la publicación exacta que
seleccionó; no escribe tablas de catálogos. No acredita un recorrido de firma.

Orden de instalación: AD193 → AD178 → AD177 → CC7 → CT176, con
`deploy/principal/lista_sql_claude_firmas_plan_20261005.txt` después de la
lista de AD178/AD177. La base de CC7 es CC1.

`confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)` recibe un material
UTF-8 exacto y un envelope creado por la fachada AD177. El material tiene
trece claves: `esquema`, `operacion`, `catalogo_id`, `version`,
`revision_esperada`, `huella_esperada`, `clave_operacion`,
`catalogo_canonico_base64`, `catalogo_sha256`, `traza_canonica_base64`,
`traza_sha256`, `evento_canonico_base64` y `evento_sha256`. El esquema es
`vec.catalogos.plan-firma.gobierno.v1`; la operación es crear, actualizar,
publicar o retirar. El catálogo conserva los bytes de `CatalogoConfigurable`
generados por el servicio común. `catalogo_id` se liga al recurso exacto
autorizado; `modulo_id` debe ser `contratacion_temporal`. Cada entrada usa el
esquema `ct.plan-competencia-firma.v2` y los dieciocho atributos que consume
`plannominal.DesdeCatalogo`.

El ID del catálogo y la clave de entrada cumplen a la vez la gramática de
CatalogoConfigurable y la del plan CT: `^[a-z][a-z0-9._-]{2,127}$`. Las
referencias de circuito, cargo y demás atributos conservan su propia gramática;
pueden contener dos puntos. La prueba estructural rechaza `a`, `b` y `ct:plan`
como ID y clave de entrada.

El envelope contiene `consumo` con los siete campos de AD170, más
`actor_ref`, `perfil_ref`, `accion`, `finalidad`, `proceso` y `canal`. CC7 no
acepta esos datos como prueba por sí solos: el comprobador privado de AD177
relee el consumo, la decisión y la auditoría comunes de la misma transacción.
Proceso y canal del envelope deben coincidir con los datos de esa auditoría
que devuelve el comprobador.
El recurso de gobierno liga `estado`, `revision` y SHA-256 del material exacto.
El decorador que añade esta huella al recurso operativo está pendiente del
responsable AD; el ServicioCatalogos actual no la añade. Hasta disponer de él,
la operación no puede producir una decisión admisible.

La revisión esperada y la huella anterior forman el CAS de borrador. Crear
abre la versión; actualizar incrementa su revisión. Publicar conserva el
contenido editado y exige un actor distinto del creador y último editor.
Retirar exige otro actor distinto del publicador y una aprobación propia. La
publicación guarda sus bytes y SHA-256 originales en una tabla inmutable; la
retirada cambia solo el control actual y añade historia. Una misma clave de
operación y material devuelve el recibo conservado. Otro material con esa
clave se rechaza. Historia, recibo, referencia de auditoría común y outbox se
escriben dentro de la misma transacción.

La respuesta de gobierno devuelve estado, revisión, huella y SHA de publicación,
además del actor, fecha confirmada, referencia de auditoría y recibo de outbox
originales. Esos cuatro datos proceden de `plan_firma_efecto` y
`plan_firma_outbox` también en un replay posterior a la retirada. La clave del
outbox es el recibo del efecto; no se genera otro identificador. Cada reintento
usa los mismos bytes y la misma clave, con una autorización V3 nueva. El
kit privado previsto deberá conservar ese fichero de material exacto para
permitir el reintento; todavía no está implementado en esta rama.

El canon Go incluye las fechas cero como `0001-01-01T00:00:00Z`, incluso en
campos `time.Time` con `omitempty`. CC7 conserva esos bytes. La entrada sin
fecha final tiene esa misma marca; al fijar el plan se comprueba que la
publicación ya ocurrió y que la entrada sigue vigente. Se rechazan atributos
numéricos donde el modelo exige `map[string]string` y claves JSON duplicadas
antes de convertir a JSONB.

`leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)` pertenece solo al
propietario CT. Revalida un consumo de firma de la misma transacción mediante
el wrapper AD177, bloquea el control con `FOR SHARE` hasta COMMIT y devuelve
`documento_exacto`, `publicacion_sha256`, `entrada` y `revision_control`.
Rechaza una versión retirada o una huella distinta. La entrada es una lectura
del documento original, sin volver a construirlo. Tras adquirir el bloqueo y
antes de devolverla vuelve a comprobar la vigencia de la decisión V3, además
de la fecha de publicación y la vigencia de la entrada. El propietario AD puede
confirmar gobierno; ningún rol LOGIN recibe EXECUTE de estas fachadas.

La prueba `pruebas_sql/plan_nominal_firma_000007.sql` comprueba objetos, RLS,
propietarios, ACL y funciones. Incluye los bytes de tres vectores generados por
`CatalogoConfigurable.ClonarCanonico` y `json.Marshal` en Go: borrador SHA-256
`ce029352e249d4260bcc2b5717de0fd9aebbff3f1595af5d7170930805f78e04`,
publicado SHA-256
`2a88331f403f3de34386a5b2e4930e22f7533282ae1f6e9f9faa85b3ac9f9dd6`
y retirado SHA-256
`817a7bb30120412671e8a1208ace781a1548f9c591a15a2c21dcf4f4254bc457`.
Sus copias exactas están en `pruebas_sql/testdata/`. No inserta concesiones
sintéticas.

El 5 de octubre de 2026 se ensayó en el clon de la principal posterior a AD193 (repetido sobre main con AD195 y AD196),
con AD178 y AD177: UP único con código 0 y prueba verde. Dos correcciones sobre
el borrador de Codex-E: el validador de selectores usaba la expresión regular
`{2,511}`, que PostgreSQL rechaza (máximo 255) y habría hecho fallar toda
validación de un plan; ahora la longitud se comprueba aparte. Y los manejadores
del parseo capturan sólo `data_exception`.

Sólo vale el plan vigente: `leer_plan_nominal_firma_v1` rechaza una versión si
hay otra posterior del mismo catálogo publicada, o cualquier otro catálogo de
plan publicado en el módulo; y publicar se rechaza mientras haya otro catálogo
de plan publicado. Así, al publicar la versión N+1, las firmas dejan de poder
fijarse a la N aunque todavía no se haya retirado. Si después se retira la N+1, la N no vuelve a valer: hay que publicar otra versión. Para publicar no basta con
ser distinto del creador y del último editor: nadie que haya creado o editado
esa versión puede publicarla. Una actualización no puede cambiar `creado_en`.
Las fechas `publicado_en`, `retirado_en` y `creado_en` del canon las aporta el
material (sólo se valida su orden); la hora real de cada operación queda en
`plan_firma_historia.registrada_en`.

`pruebas_sql/plan_nominal_firma_positivo_clon.sql` (sólo clon desechable,
ROLLBACK) recorre con consumos sintéticos sellados crear, reintento con la misma
clave (mismo recibo), actualizar, publicar con separación de funciones, segundo
catálogo rechazado, leer y retirar (después, leer se rechaza). Para ello sustituye
dentro del ROLLBACK la categoría Aplicación de AUT, que hoy no admite
`vec.catalogos.*`. Pendiente menor: el CAS fallido usa 40001, que un reintentador
genérico repetiría; la fecha `publicado_en` y las demás del canon admiten valores
imposibles que darían 22008 en vez de 22023.

`leer_plan_nominal_firma_v1` acredita que hay un consumo de firma de esta
transacción, pero no lo liga por sí misma al plan: esa liga la hacen CT176 y
AD177 con el contexto `plan_firma_sha256`. Sólo el propietario CT puede llamarla.
El gobierno sigue cerrado hasta que K extienda la categoría Aplicación a
`vec.catalogos.*` (y sólo para el plan); CC7 rechaza cualquier catálogo que no
use el esquema del plan. Faltan las pruebas transaccionales de concurrencia y
recuperación, el adaptador Go y el montaje.
