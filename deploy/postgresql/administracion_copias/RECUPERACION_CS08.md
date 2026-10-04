# Preservación del WIP de propuestas y órdenes CS08

Estado: **NO-GO para integración, ensayo o instalación**. Este corte conserva
seis archivos cedidos por el operador. No confirma su funcionamiento ni completa
el circuito ADMIN. No se han ejecutado Go, SQL, UP ni DOWN, ni se han creado roles
o permisos. No se abre PR.

La base es `c329842850a33ba034ba729e94f49cdbfcb92bfe`, de la rama
`trabajo/codexa-cs08-ordenes-retoma-20261004`. Los seis archivos proceden del WIP
sin versionar del worktree `codexk-cs08-orden-v3-20261001`. Se han copiado sin
modificarlos y se han comparado sus SHA256. El original permanece intacto y sin
versionar. Las reservas Administración Copias `000001` y AD143 siguen siendo
exclusivas de K; este corte no reserva otro número ni reutiliza una reserva ajena.

## Archivos conservados

Las rutas SQL siguientes son relativas a `deploy/postgresql/administracion_copias/`.
Las rutas Go son relativas a `internal/modules/administracion/adapters/controlrestauracionpg/`.

| Archivo | SHA256 idéntico en origen y copia |
| --- | --- |
| `migraciones/000001_propuestas_ordenes_v3.up.sql` | `65f9ba25aa2411e00b4797359d11410000e513fb95abf9819fcf12a221739e7f` |
| `migraciones/000001_propuestas_ordenes_v3.down.sql` | `e0c12ec67810e54150785e4d69981b17382d8c8998cc8d0bc7351f42f58bf8a3` |
| `pruebas_sql/000001_plan_cerrado.sql` | `201b7e4d9aa0f39208df3db94be016fa39b5754730895308caff22329d507edd` |
| `pruebas_sql/000001_contratos_acl.sql` | `44108e3213be8b700847894a4b40d8fea3f2bed99e557aba9bc16a553be485a3` |
| `registro.go` | `a8739ee2182854c34f33a99dc1c7ac4f1fb61b8898787d88f2ce5c532e12551a` |
| `registro_test.go` | `9142bd089bfdd02f37783ff3d6ba5dbc39cb67f248e54efaec946cf4e252469d` |

## Lectura estática y bloqueos

| Fuente o contrato | Estado observado en este corte | Condición que impide continuar |
| --- | --- | --- |
| Precondición `$pre$` del UP | Exige tres consumidores AD143 con diez argumentos: cuatro `bytea`, dos `numeric` y cuatro `bytea`. Sus definiciones no están versionadas en la base indicada. No se ha consultado una base instalada. | No hay ABI común final acreditada para `consumir_propuesta_restauracion_copias_v3_atestada`, `consumir_revision_restauracion_copias_v3_atestada` y `consumir_orden_copias_v3_atestada`. La guarda prevista es `copias_consumidor_ad143_incompatible`; no es un fallo de ensayo observado. |
| Roles técnicos | El UP exige propietarios y ejecutor existentes, con `NOLOGIN`, sin superusuario ni `BYPASSRLS`. No se provisionan en esta entrega. | La configuración y las ACL reales no están comprobadas. Guarda prevista: `copias_rol_tecnico_incompatible`. |
| Esquema de destino | El UP sólo admite un esquema nuevo; el estado instalado no se ha observado. | Un esquema existente activa `copias_esquema_ya_instalado`. Conservar estos archivos no autoriza recrearlo ni ejecutar DOWN. |
| Materializador y fuente de K | El adaptador recibe `MaterializadorV3` de la autoridad común y coteja la decisión canónica. No se ha compuesto la fuente operativa final. | El material debe corresponder al actor, sesión, perfil, plan y aprobación exactos, con versiones y política actuales. No se sustituye por JSON de la petición, una fábrica de pruebas o una decisión histórica. |
| Auditoría común de L | Las escrituras SQL esperan `auditoria_ref` del consumidor central. La tabla local `vec_administracion_copias.auditoria` conserva enlaces y huellas del progreso. | Esa tabla no acredita ni sustituye la auditoría V3 común segregada. Falta contrastar el contrato final de consumo, auditoría y outbox con L. No se introduce otra autoridad de auditoría en este corte. |
| COMMIT y recuperación | Propuesta, revisión, orden, CAS y referencias de consumo se plantean dentro de una transacción. Los adaptadores invocan fachadas SQL y la publicación relee la orden. | Hace falta demostrar el COMMIT confirmado con consumo único, auditoría V3 y outbox, y la recuperación posterior sin otro efecto. Una fila, un candidato o un recibo local no bastan. |
| Pruebas conservadas | Hay comprobaciones de estructura, ACL y plan, y pruebas Go de rechazo y respuesta cerrada. No se ejecutan aquí. | No prueban el positivo V3 real, revocación, dos Personas, concurrencia o recuperación de una instalación. No se emite `ENSAYO-OK`. |

El adaptador Go usa los tipos nominales comunes de solicitud, decisión y
confirmación V3; no fabrica permisos ni consulta tablas de otros módulos. Sus
consultas de propuesta y revisión tienen once y doce argumentos, respectivamente,
coincidentes con las fachadas conservadas. Esto acredita sólo el cotejo estático
de firmas. La función SQL central sigue siendo responsable de revalidar la
autoridad antes de cualquier escritura o replay.

El SQL conserva `search_path` fijo, límites de tiempo, bloqueo transaccional,
RLS forzada, revocación de acceso público y guardas de historia. El DOWN rechaza
tablas con historia y se conserva exclusivamente como parte del borrador;
no se ha ejecutado. Estos controles escritos no acreditan ACL efectivas,
compatibilidad instalada ni seguridad de la versión final.

## Comprobaciones permitidas y siguiente condición

En este corte se comprueban igualdad de archivos, formato Go sin modificarlo,
diff y análisis Semgrep local. No se cambian las preimágenes SQL para adaptarlas
a una base desconocida, ni se ejecuta un clon con sustitutos de la autoridad.

`gofmt -l` no señaló archivos y el diff pasó su comprobación. Semgrep local
recorrió los dos archivos Go con 42 reglas y los cuatro SQL con tres reglas
textuales, sin hallazgos y con métricas desactivadas. El análisis textual SQL
no comprueba sintaxis PL/pgSQL, transacciones ni ACL instaladas.

Para levantar el NO-GO, dirección debe recibir las fuentes y contratos finales
de K y L, cotejar la ABI y las precondiciones del destino autorizado y definir
el ensayo aislado. La versión final requiere ensayo PostgreSQL y dos revisiones
independientes, incluida SQL. La provisión nominal, la política y el anclaje
actuales pertenecen a sus autoridades comunes. No los decide esta preservación.
