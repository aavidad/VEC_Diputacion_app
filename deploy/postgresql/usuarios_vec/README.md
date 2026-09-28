# Usuarios 5.08a: preferencias propias

## Orden causal

1. Núcleo de autorización V3 hasta AD3-105, incluyendo AD3-101, y ContextoActor certificado. AD3-102/103 y AD3-104/105 siguen su propio orden de instalación; su presencia en Git no acredita instalación en ninguna base.
2. `roles_up.sql`, ejecutado una sola vez por DBA. El LOGIN técnico se aprovisiona fuera de Git como miembro directo exclusivo de `vec_usuarios_ejecutor`, con `INHERIT TRUE`, `SET FALSE` y `ADMIN FALSE`. Nadie concede propiedad al runtime.
3. AD3-106 `consumidor_preferencias_propias.up.sql`, que añade audiencias y dos consumidores nominales después de crear los roles.
4. Usuarios 000001 `preferencias_base.up.sql`, después de AD3-106.
5. Usuarios 000002 `operaciones_preferencias.up.sql`.

Ningún `DOWN` se ejecuta sobre historia. Los archivos `down.sql` rechazan la operación. Para una corrección posterior se crea una migración nueva. No ejecutar estas migraciones en cidonia por existir en esta rama.

## Contrato del adaptador

Las funciones se ejecutan con un LOGIN técnico miembro exclusivo de `vec_usuarios_ejecutor`, dentro de una transacción `SERIALIZABLE READ WRITE` para consulta, recuperación y guardado. `catalogo_vigente_preferencias_v1()` devuelve el catálogo publicado. Las funciones `consultar_preferencias_propias_v1` y `recuperar_preferencias_operacion_v1` reciben `p_material text` seguido de las diez piezas de `ExportacionMaterialConsumoAutorizacionAtestadaV3`. `guardar_preferencias_propias_v1` recibe además `p_valores jsonb` antes de las diez piezas. El material es el JSON literal de `ports.MaterialPreferencias` generado por el servidor; nunca un `persona_ref` libre del cliente. El servidor fija `RecursoAutorizable.Ambitos["persona_ref"]` con la persona canónica y `Atributos["material_sha256"]` con SHA256 de esos bytes; V3 firma la huella canónica del recurso `{"ambitos":{"persona_ref":"<persona>"},"atributos":{"material_sha256":"<hex>"}}`. SQL reconstruye ambas huellas y las coteja con la decisión y capacidad V3.

AD3-106 usa `modulo_id=usuarios`, `tipo_recurso=preferencias_persona`, `recurso_ref=persona_ref`, finalidad `finalidad:usuarios:preferencias-propias:v1`, acción `vec.preferencias.consultar` o `vec.preferencias.actualizar`, audiencias respectivas `vec_usuarios.preferencias.consultar.v1` y `vec_usuarios.preferencias.actualizar.v1`, campos `[catalogo,valores,version]` o `[valores,version]`, obligaciones vacías.

GET consume V3 y devuelve estado v0 y valores predeterminados sin crear fila cuando no hay preferencia. La recuperación consume y coteja V3 antes de mirar si existe la clave; devuelve `NULL` si no existe operación y, si existe, el recibo original sin mutar estado ni historia. Tras `NULL`, la aplicación obtiene otra exportación V3 fresca antes de PUT. PUT recalcula la huella semántica Go, valida valores, consume V3 antes de leer catálogo/clave/versión, y aplica CAS con estado, historia y recibo en la misma transacción. Avisos por correo son datos de opt-in: no existe outbox ni envío en este corte. Errores SQLSTATE: `P1409` conflicto, `22023` petición inválida, `42501` denegación, `55000` incompatibilidad de infraestructura; `40001` exige reintentar la transacción completa.

El estado, historia y recibos usan RLS por persona y operación. Un marcador privado por `xid8`, PID, LOGIN, persona y modo se crea únicamente tras consumo V3 nuevo; una transacción solo admite un contexto activo. Cada función lo retira antes de devolver, y el rollback lo revierte. El ejecutor no tiene SELECT/DML ni EXECUTE sobre marcador o helpers. Catálogo y publicaciones tienen políticas separadas para migración gobernada.

La prueba `pruebas_sql/preferencias_pg18.sh` levanta PostgreSQL 18.4 efímero sin red y borra su volumen temporal. Verifica DDL, ACL y RLS reales, vector de Go, GET ausente, CAS concurrente, replay, conflicto, savepoint, historia y denegación con V3 falsa para clave existente y ausente. Usa un doble estructural del núcleo V3 y no acredita COSE real, cadena causal completa ni arranque del binario. La puerta completa corresponde a la integración de la vertical.
