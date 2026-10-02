# Gobierno durable de Administración: CAT6 y AD152

Estos archivos son borradores de revisión. Ambos abortan antes de modificar la base. No están en una lista de instalación, no se han ejecutado y no acreditan persistencia, permisos ni recorrido. CAT6 usa la reserva `catalogos_configurables 000006`; AD152 usa `autorizacion_atestada_v3 000152`. AD151 pertenece a R5 y se conserva.

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

El recibo guarda una sola vez los bytes UTF-8 de la representación SQL y su SHA256. Sus campos son `referencia`, `clave_idempotencia`, `huella_material_sha256`, `accion`, `catalogo_id`, `version`, `huella_sha256`, `estado`, `actor_ref`, `auditoria_ref`, `outbox_ref`, `confirmado_en`. `confirmado_en` conserva el instante original de la transición, en UTC con microsegundos. El adaptador verifica bytes y SHA; no exige que SQL y `json.Marshal` ordenen igual las claves. `outbox_ref` es el recibo que identifica la única fila de outbox. El catálogo conserva los bytes canónicos Go originales, sin reconstrucción desde JSONB.

Las consultas internas, concedidas solo al ejecutor técnico nominal, devuelven `(catalogo_canonico bytea, huella_sha256 text)`:

- `obtener_catalogo_modulos_v1(id text, version integer, max_bytes integer)`: instantánea actual exacta de una versión, prefiriendo retirada/publicación sobre borrador.
- `obtener_cabeza_modulos_v1(id text, max_bytes integer)`: cabeza operativa exacta, incluida la retirada.
- `listar_catalogos_modulos_v1(id text, max_versiones integer, max_bytes integer)`: una instantánea por versión, máximo 64 y 4 MiB; si excede presupuesto falla, no devuelve una historia parcial como completa.

Una lectura del puerto para composición no concede a una persona acceso HTTP. La frontera ADMIN debe autorizar sus consultas visibles por el circuito existente. Las lecturas técnicas no son un registro público de los actores.

## Configuración offline y privilegios

`provisionar_configuracion_modulos_v1(version_esperada, huella_esperada, canonico bytea, huella)` queda solo para el propietario NOLOGIN. La configuración conserva `catalogo_id`, `version`, `registro_version_ref`, `registro_sha256`, `perfil_admin_ref`, `aprobacion_ref`, `registrados` y `gobernados`. La provisión valida huella, CAS de configuración y pertenencia de gobernados al registro aprobado; rechaza cambios con borrador pendiente. La aprobación del registro y el alcance institucional del perfil se acreditan fuera de HTTP, antes de provisionar. El rol técnico no puede llamar a esa función. La misma transacción añade la provisión inmutable y su outbox con `session_user`, autoridad SQL offline, aprobación, huellas y versión anterior, registro aprobado y recibo. Esta procedencia no se presenta como decisión V3 ni como firma de la aprobación.

Los hechos tienen RLS forzada y políticas solo para el propietario. Historia, recibos, configuración publicada y outbox rechazan UPDATE, DELETE y TRUNCATE. Las cabezas permiten únicamente la actualización interna. CAT6 retira también ACL heredadas de privilegios por defecto, incluidas las de tipos de fila. No crea LOGIN ni entrega membresías de propietario. El rol `vec_catalogos_configurables_ejecutor_modulos` recibe únicamente las fachadas de confirmación, recuperación y consulta.

AD152 añade el consumidor nominal a la autoridad V3 existente. Reconstruye el núcleo solo si coinciden definición, fuente, ACL, metadatos, dependencias y marcas únicas de la postimagen AD151 real, y comprueba que el cambio se pueda invertir exactamente. No copia el gobierno especializado C3 ni lo extiende a ADMIN. `ADMIN_SUPERFICIE_APROBADA` permanece como marcador sin autoridad hasta fijar la superficie segregada real. La membresía de sesión se limita al ejecutor nominal, sin SET ni ADMIN ni otras membresías.

## Dependencias y orden de promoción

1. Inventariar las migraciones realmente instaladas y la postimagen de AD151, incluida su cadena anterior. No deducir instalación de números reservados o de Git.
2. Fijar el perfil institucional ADMIN de gobierno de módulos, su alcance, la superficie segregada y la configuración offline del registro compuesto.
3. Cerrar los marcadores `POST_AD151_*`, `POST_AD151_EXCLUSION_LITERAL_UNICO` y `ADMIN_SUPERFICIE_APROBADA` con evidencia obtenida en PostgreSQL 18 del clon. Ningún SHA, LOGIN o aprobación se inventa aquí.
4. Revisar CAT6 y AD152 final por dos revisores independientes. Si cambian los bytes, repetir la revisión afectada.
5. Preparar por Dirección la lista causal de ensayo: CAT6 antes de AD152, tras la postimagen AD151 y sus dependencias reales. CAT5/AD147 son referencias especializadas C3, todavía no instaladas en su corte; no se presupone su instalación ni se usan sus tablas.
6. Ensayar con `ensayar_sql_clon.sh` en el clon autorizado. El resultado debe ser `ENSAYO-OK`, con control de roles, ACL, tipos, transacciones y recuperación. Retirar las guardas globales solo en el candidato que haya cerrado esas dependencias; nunca para probar directamente en la principal.
7. Definir y revisar la provisión inicial offline v1 antes de habilitar el runtime. Mantener la denegación mientras falte.

No hay instrucciones DOWN sobre historia conservada. Una corrección posterior necesita una nueva migración. No se ha consultado ni modificado principal, cidonia, Emilio o datos reales.

## Pruebas previstas y límites de este corte

El ensayo debe cubrir borrador sin cambio operativo, publicación por otro actor, rechazo de creador/último editor, CAS de cabeza y borrador, dos publicaciones concurrentes, repetidos ON/OFF N→N+1, reintento con la misma clave y material, conflicto de material, recuperación con autorización revocada/ajena, conservación de bytes y fecha, ausencia de duplicados y recuperación después de reiniciar. También debe comprobar registro vacío/desconocido, core/ADMIN excluidos, configuración desfasada, perfil gestor sin concesión automática, ausencia de privilegios en tablas/tipos/funciones y fallos transaccionales después del consumo V3.

La cabeza debe seguir consultándose con más de 64 versiones. La lista debe rechazar el exceso y los catálogos grandes deben respetar el presupuesto antes de materializar resultados.

En este corte se ha hecho lectura y revisión estática del SQL y del contrato de dominio/aplicación. No hay PostgreSQL disponible como herramienta ejecutable en esta sesión; no se acredita sintaxis compilada de PL/pgSQL, ENSAYO-OK, ACL efectivas, concurrencia, instalación ni navegador. Semgrep no valida PostgreSQL/PLpgSQL; la comprobación SQL corresponde al ensayo y a las revisiones independientes. Las guardas y los marcadores de estos borradores impiden su instalación anticipada.

Fuentes aplicadas: ESPECIFICACIONES_AGENTES E03/E04/E06/E10/E12; `CatalogoConfigurable`, `ServicioCatalogos`, #416 y los puertos operativos centrales; contrato de módulos VEC; CAT5/AD147 de `65e91fcd5` solo como referencia de controles. Skills aplicadas: orquestar-vec, persistir-autorizar-vec, revisar-sql-vec, probar-recorridos-vec, security-audit focal y humanizer/VEC-USO. No se ha delegado ni se ha enviado código a servicios externos.
