# Gobierno durable de Administración: CAT6 y AD152

Estos archivos son borradores de revisión. Ambos abortan antes de modificar la base. No están en una lista de instalación. Una copia aislada compiló en PostgreSQL 18.4 tras retirar únicamente esas guardas globales; ese ensayo de preparación no habilita el runtime ni acredita un recorrido. CAT6 usa la reserva `catalogos_configurables 000006`; AD152 usa `autorizacion_atestada_v3 000152`. AD151 pertenece a R5 y se conserva. Su número no constituye una dependencia de estos borradores.

El catálogo central tiene ID `administracion.modulos` y propietario funcional `vec.module.administracion`. Una configuración offline aprobada fija el registro compuesto, su referencia de versión y su SHA256, los IDs registrados, la lista positiva gobernable y el perfil ADMIN nominal. El formulario no elige el registro, el catálogo, el perfil ni la configuración. `vec.module.usuarios` y `vec.module.administracion` quedan excluidos. Cualquier ID fuera de la lista gobernable queda cerrado.

## Dos confirmaciones reales

Un actor A crea el borrador de la versión N+1 desde la publicación N. La confirmación conserva catálogo, historia, recibo y outbox, y consume su autorización V3 junto con la auditoría central en una transacción. La publicación N continúa operativa.

Un actor B, distinto del creador y del último editor, publica ese borrador. La segunda transacción coteja la cabeza N y el borrador exacto, consume otra autorización y conserva otro recibo. Solo entonces cambia el interruptor. No se simulan dos identidades ni se interpreta un borrador guardado como activación.

La retirada conserva una cabeza terminal `retirado`: no recupera una versión anterior. Las versiones y sus revisiones permanecen en `modulos_historia`. La consulta de cabeza usa el puntero durable, sin listar la historia ni aplicar el límite de 64 versiones.

El circuito runtime rechaza la creación inicial sin cabeza publicada positiva. La publicación inicial v1 requiere una provisión offline separada, aprobada y revisada, con autorización, historia, auditoría y recibo propios. Ese procedimiento todavía no está definido. Hasta que exista, el circuito permanece cerrado; no se insertan valores de activación de ejemplo al arrancar.

## Contratos de PostgreSQL

`confirmar_gobierno_modulos_v1` y `recuperar_gobierno_modulos_v1` reciben `p_material_exacto bytea` seguido de los diez argumentos estándar V3: capacidad, decisión, motivo, contexto, versión de persona, versión de perfil, payload, sobre, evidencia y raíz. Devuelven una fila con `catalogo_canonico bytea`, `recibo_canonico bytea`, `recibo_sha256 text` y `recuperada boolean`. La recuperación sin clave propia devuelve cero filas después de consumir autorización vigente.

El material de confirmación contiene estos campos:

- `clave_operacion`, `operacion` (`crear`, `actualizar`, `publicar`, `retirar`).
- `material_semantico_base64` y `huella_material_sha256` del servicio central.
- `configuracion_version`, `configuracion_huella_sha256`, `registro_version_ref`, `registro_sha256`.
- `cabeza_publicada_esperada`: objeto `{version, estado, huella_sha256}`.
- `borrador_esperado`: objeto `{version, revision, huella_sha256}` o `null` explícito al crear.
- `catalogo_canonico_base64`, `huella_comun`, `traza` y `evento` centrales.

El material reducido de recuperación contiene la clave, operación, bytes y huella semánticos, los cuatro campos de configuración y registro, `catalogo_id`, `version`, `revision` y `estado` del efecto original. La decisión vuelve a pedir la acción original y el recurso exacto. La recuperación no cambia la cabeza ni añade otra historia u otro outbox.

La clave se vincula a actor, catálogo y comando semántico. El material central excluye decisión, correlación y tiempos frescos; SQL comprueba el SHA256 de sus bytes originales y coteja los campos tipados con actor, catálogo, cabeza y contenido. El material de transporte completo queda ligado a V3 mediante este contexto exacto:

```json
{"ambitos":{},"atributos":{"estado":"ESTADO","material_sha256":"SHA256_DEL_MATERIAL_EXACTO","revision":"REVISION"}}
```

`recurso_ref` es `administracion.modulos:N`, `modulo_id` es `vec.module.administracion` y `tipo_recurso` es `catalogo_configurable`. La audiencia candidata es `vec_catalogos_configurables.gobierno_modulos.v1`; la finalidad candidata es `gobernar_modulos_administracion`. El perfil de consumo SQL `gobierno_modulos_administracion` es un selector técnico del núcleo V3, no una concesión funcional ni un perfil de usuario. El perfil funcional debe coincidir con el perfil ADMIN fijo de la configuración aprobada.

El recibo guarda una sola vez los bytes UTF-8 de la representación SQL y su SHA256. Sus campos son `referencia`, `clave_idempotencia`, `huella_material_sha256`, `accion`, `catalogo_id`, `version`, `huella_sha256`, `estado`, `actor_ref`, `auditoria_ref`, `outbox_ref`, `confirmado_en`. `confirmado_en` conserva el instante original de la transición, en UTC con microsegundos. Antes de cualquier cast, CAT6 exige RFC3339 con sufijo `Z` y un máximo de seis decimales en `creado_en`, `ultima_modificacion_en`, `publicado_en` y `retirado_en`. Una fracción submicro se rechaza sobre el texto original, sin permitir que PostgreSQL la redondee. El adaptador verifica bytes y SHA; no exige que SQL y `json.Marshal` ordenen igual las claves. `outbox_ref` es el recibo que identifica la única fila de outbox. El catálogo conserva los bytes canónicos Go originales, sin reconstrucción desde JSONB.

Las consultas internas, concedidas solo al ejecutor técnico nominal, devuelven `(catalogo_canonico bytea, huella_sha256 text)`:

- `obtener_catalogo_modulos_v1(id text, version integer, max_bytes integer)`: instantánea actual exacta de una versión, prefiriendo retirada/publicación sobre borrador.
- `obtener_cabeza_modulos_v1(id text, max_bytes integer)`: cabeza operativa exacta, incluida la retirada.
- `listar_catalogos_modulos_v1(id text, max_versiones integer, max_bytes integer)`: una instantánea por versión, máximo 64 y 4 MiB; si excede presupuesto falla, no devuelve una historia parcial como completa.

Una lectura del puerto para composición no concede a una persona acceso HTTP. La frontera ADMIN debe autorizar sus consultas visibles por el circuito existente. Las lecturas técnicas no son un registro público de los actores.

## Configuración offline y privilegios

`provisionar_configuracion_modulos_v1(version_esperada, huella_esperada, canonico bytea, huella)` queda solo para el propietario NOLOGIN. La configuración conserva `catalogo_id`, `version`, `registro_version_ref`, `registro_sha256`, `perfil_admin_ref`, `aprobacion_ref`, `registrados` y `gobernados`. La provisión valida huella, CAS de configuración y pertenencia de gobernados al registro aprobado; rechaza cambios con borrador pendiente. La aprobación del registro y el alcance institucional del perfil se acreditan fuera de HTTP, antes de provisionar. El rol técnico no puede llamar a esa función. La misma transacción añade la provisión inmutable y su outbox con `session_user`, autoridad SQL offline, aprobación, huellas y versión anterior, registro aprobado y recibo. Esta procedencia no se presenta como decisión V3 ni como firma de la aprobación.

Los hechos tienen RLS forzada y políticas solo para el propietario. Historia, recibos, configuración publicada y outbox rechazan UPDATE, DELETE y TRUNCATE. Las cabezas permiten únicamente la actualización interna. CAT6 retira también ACL heredadas de privilegios por defecto, incluidas las de tipos de fila. No crea LOGIN ni entrega membresías de propietario. El rol `vec_catalogos_configurables_ejecutor_modulos` recibe únicamente las fachadas de confirmación, recuperación y consulta.

AD152 añade el consumidor nominal a la autoridad V3 existente. Reconstruye el núcleo solo si coinciden definición, fuente, ACL, metadatos, dependencias y marcas únicas de la preimagen real del núcleo común, y comprueba que el cambio se pueda invertir exactamente. No copia el gobierno especializado C3 ni lo extiende a ADMIN. `ADMIN_SUPERFICIE_APROBADA` permanece como marcador sin autoridad hasta fijar la superficie segregada real. La membresía de sesión se limita al ejecutor nominal, sin SET ni ADMIN ni otras membresías.

## Dependencias y orden de promoción

1. Inventariar los objetos realmente presentes: el núcleo `consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)`, la tabla `clave_capacidad_version` y su constraint de audiencias, el propietario V3, el propietario CAT y `rechazar_cambio_inmutable()`. AD152 requiere además los dos objetos CAT6: el ejecutor nominal y `confirmar_gobierno_modulos_v1(...)`. No referencia funciones ni tablas propias de AD149, AD150 o AD151; no se espera a esos números. No deducir instalación de Git o de una reserva.
2. Fijar el perfil institucional ADMIN de gobierno de módulos, su alcance, la superficie segregada y la configuración offline del registro compuesto.
3. AD152 fija la preimagen obtenida del clon PostgreSQL 18.4 sobre `origin/main@c836e6b2` después de AD142: definición `202b1580f00e1618e0fb311dcf0911992e56d9eb5f17bc642d58c73768f360f1`, fuente `4a98b94be7e198a6c1e364949e60f35f39b2e5bf931bc361b1adb52a12a94905`, constraint de audiencias con `pg_get_constraintdef(oid,true)` `d5c8048786b283485016af29fba41ff68b93076ba4f37f2badfa6bb7d5532fd9`. Se cotejaron los archivos del agente de ensayo y las tres marcas de sustitución aparecieron una sola vez. Cerrar `ADMIN_SUPERFICIE_APROBADA` requiere acreditar la superficie ADMIN real; sigue pendiente. No se inventan LOGIN ni aprobaciones; estas huellas identifican objetos del clon, no instalación de CAT6/AD152 en la principal.
4. Revisar CAT6 y AD152 final por dos revisores independientes. Si cambian los bytes, repetir la revisión afectada.
5. Preparar por Dirección la lista causal de ensayo: CAT6 antes de AD152, sobre la preimagen real del núcleo y sus dependencias por objeto. CAT6 requiere el propietario y `rechazar_cambio_inmutable()` del esquema central; su función de confirmación queda sin acceso efectivo al consumo hasta que AD152 cree la fachada V3. La identidad/sesión ADMIN runtime depende de los objetos nominales V2 aportados por AUT24/CA23/IS12, cuando el proveedor de composición los use; no se considera sustituida por esta migración SQL. CAT5/AD147 son referencias especializadas C3, todavía no instaladas en su corte; no se presupone su instalación ni se usan sus tablas.
6. Ensayar con `ensayar_sql_clon.sh` en el clon autorizado. El resultado debe ser `ENSAYO-OK`, con control de roles, ACL, tipos, transacciones y recuperación. Retirar las guardas globales solo en el candidato que haya cerrado esas dependencias; nunca para probar directamente en la principal.
7. Definir y revisar la provisión inicial offline v1 antes de habilitar el runtime. Mantener la denegación mientras falte.

No hay instrucciones DOWN sobre historia conservada. Una corrección posterior necesita una nueva migración. No se ha consultado ni modificado principal, cidonia, Emilio o datos reales.

## Pruebas previstas y límites de este corte

El ensayo debe cubrir borrador sin cambio operativo, publicación por otro actor, rechazo de creador/último editor, CAS de cabeza y borrador, dos publicaciones concurrentes, repetidos ON/OFF N→N+1, reintento con la misma clave y material, conflicto de material, recuperación con autorización revocada/ajena, conservación de bytes y fecha, ausencia de duplicados y recuperación después de reiniciar. La prueba negativa de fechas debe pasar por las cuatro propiedades originales con siete o más decimales y confirmar rechazo antes del cast; también debe comprobar UTC `Z`, entre cero y seis decimales válidos y rechazo de offsets alternativos. También debe comprobar registro vacío/desconocido, core/ADMIN excluidos, configuración desfasada, perfil gestor sin concesión automática, ausencia de privilegios en tablas/tipos/funciones y fallos transaccionales después del consumo V3.

La cabeza debe seguir consultándose con más de 64 versiones. La lista debe rechazar el exceso y los catálogos grandes deben respetar el presupuesto antes de materializar resultados.

El agente de ensayo ejecutó copias de CAT6 y AD152 de `d6e7c9bfca9e4edda50d878632cb56a22c7a1637` en su clon aislado PostgreSQL 18.4 sobre el núcleo post142 real. Retiró únicamente los dos bloques globales `DO pendiente` en scratch y conservó sus diffs. `psql` terminó con código 0 para ambos archivos. Los bytes originales ensayados tienen SHA256 CAT6 `5d6aefc0301a087e581e9e2d15bd64961c96b35849884fa1a2785b9572b2b9a1` y AD152 `e806a404feda151d53128ab90903945987a0628903a83615a9e6790931fc150a`. La definición del núcleo pasó de la preimagen documentada a `1d81e87d56da6f307b0f057dde483d5e5742c3ce0f118fce2304eabc7c86d452` en ese clon.

El ensayo verificó las ocho tablas nuevas vacías, RLS habilitada y forzada, ausencia de DML directo para el ejecutor y ausencia de EXECUTE nuevo para PUBLIC. El propietario CAT pudo acceder a la fachada AD152; el ejecutor no pudo invocar directamente el núcleo V3. Se conservaron los 71 expedientes y las 6199 filas de auditoría V3 anteriores. Esto acredita compilación y controles de preparación en la copia ensayada, sin confirmar una operación ADMIN.

El parche temporal superó 32 casos léxicos locales, ocho valores por cada una de las cuatro propiedades. La misma expresión ejecutada en PostgreSQL aceptó seis decimales y rechazó nueve decimales y offset. No se recorrió positivamente cada operación V3 ni se acreditaron concurrencia, persistencia de recibos con reinicio, bootstrap inicial, interfaz o navegador. El acta y los diffs del agente de ensayo se conservan fuera de Git; Dirección dispone de sus referencias privadas.

No se declara `ENSAYO-OK` de instalación. La fuente Git continúa cerrada con las guardas globales y el marcador de superficie ADMIN. Semgrep no valida PostgreSQL/PLpgSQL; la aceptación final corresponde al ensayo causal completo y a las dos revisiones independientes.

Fuentes aplicadas: ESPECIFICACIONES_AGENTES E03/E04/E06/E10/E12; `CatalogoConfigurable`, `ServicioCatalogos`, #416 y los puertos operativos centrales; contrato de módulos VEC; CAT5/AD147 de `65e91fcd5` solo como referencia de controles. Skills aplicadas: orquestar-vec, persistir-autorizar-vec, revisar-sql-vec, probar-recorridos-vec, security-audit focal y humanizer/VEC-USO. No se ha delegado ni se ha enviado código a servicios externos.
