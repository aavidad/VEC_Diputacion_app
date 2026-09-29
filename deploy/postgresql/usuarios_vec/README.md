# Usuarios 5.08a: preferencias propias

## Orden causal

1. Núcleo de autorización V3 hasta AD3-105, incluyendo AD3-101, y ContextoActor certificado. AD3-102/103 y AD3-104/105 siguen su propio orden de instalación; su presencia en Git no acredita instalación en ninguna base.
2. `roles_up.sql`, ejecutado una sola vez por DBA. Los dos LOGIN técnicos se aprovisionan fuera de Git: cada uno es miembro directo exclusivo de `vec_usuarios_ejecutor_interno` o `vec_usuarios_ejecutor_externo`, con `INHERIT TRUE`, `SET FALSE` y `ADMIN FALSE`. Cada superficie usa pool y DSN propios. Nadie concede propiedad al runtime.
3. AD3-106 `consumidor_preferencias_propias.up.sql`, que añade cuatro audiencias y dos consumidores nominales después de crear los roles.
4. Usuarios 000001 `preferencias_base.up.sql`, después de AD3-106.
5. Usuarios 000002 `operaciones_preferencias.up.sql`.

Ningún `DOWN` se ejecuta sobre historia. Los archivos `down.sql` rechazan la operación. Para una corrección posterior se crea una migración nueva. No ejecutar estas migraciones en cidonia por existir en esta rama.

## Contrato del adaptador

Las funciones se ejecutan con el LOGIN técnico de su superficie, dentro de una transacción `SERIALIZABLE READ WRITE` para consulta, recuperación y guardado. `catalogo_vigente_preferencias_v1(p_superficie text)` devuelve el catálogo publicado solo si la superficie coincide con el rol técnico. Las funciones `consultar_preferencias_propias_v1` y `recuperar_preferencias_operacion_v1` reciben `p_material text` seguido de las diez piezas de `ExportacionMaterialConsumoAutorizacionAtestadaV3`. `guardar_preferencias_propias_v1` recibe además `p_valores jsonb` antes de las diez piezas. El material es el JSON literal de `ports.MaterialPreferencias` generado por el servidor, incluida su superficie acreditada; nunca un `persona_ref` ni una superficie libres del cliente. El servidor fija `RecursoAutorizable.Ambitos["persona_ref"]` con la persona canónica y `Atributos["material_sha256"]` con SHA256 de esos bytes; V3 firma la huella canónica del recurso `{"ambitos":{"persona_ref":"<persona>"},"atributos":{"material_sha256":"<hex>"}}`. SQL reconstruye ambas huellas y coteja superficie del material, vínculo V3 firmado, audiencia y rol de sesión.

AD3-106 usa `modulo_id=usuarios`, `tipo_recurso=preferencias_persona`, `recurso_ref=persona_ref`, finalidad `finalidad:usuarios:preferencias-propias:v1`, acción `vec.preferencias.consultar` o `vec.preferencias.actualizar`, y cuatro audiencias exactas `vec_usuarios.preferencias.{consultar|actualizar}.{interna_corporativa|externa_personal}.v1`. Campos `[catalogo,valores,version]` o `[valores,version]`, obligaciones vacías. Las cuatro combinaciones se ligan a la superficie del vínculo V3 firmado y al LOGIN técnico.

GET consume V3 y devuelve estado v0 y valores predeterminados sin crear fila cuando no hay preferencia. La recuperación consume y coteja V3 antes de mirar si existe la clave; devuelve `NULL` si no existe operación y, si existe, el recibo original sin mutar estado ni historia. Tras `NULL`, la aplicación obtiene otra exportación V3 fresca antes de PUT. PUT recalcula la huella semántica Go, valida valores, consume V3 antes de leer catálogo/clave/versión, y aplica CAS con estado, historia y recibo en la misma transacción. Avisos por correo son datos de opt-in: no existe outbox ni envío en este corte. Errores SQLSTATE: `P1409` conflicto, `22023` petición inválida, `42501` denegación, `55000` incompatibilidad de infraestructura; `40001` exige reintentar la transacción completa.

El estado, historia y recibos usan RLS por persona, superficie y operación. Un marcador privado por `xid8`, PID, LOGIN, persona, superficie y modo se crea únicamente tras consumo V3 nuevo; una transacción solo admite un contexto activo. Cada función lo retira antes de devolver, y el rollback lo revierte. Los ejecutores no tienen SELECT/DML ni EXECUTE sobre marcador o helpers. Catálogo y publicaciones tienen políticas separadas para lectura técnica de las superficies y publicación por migrador/DBA acreditado.

La prueba `pruebas_sql/preferencias_pg18.sh` levanta PostgreSQL 18.4 efímero sin red y borra su volumen temporal. Verifica DDL, ACL y RLS reales, vectores Go de ambas superficies, GET/PUT en cada una, cruce de rol/material/audiencia/vínculo, replay entre superficies de la misma persona, CAS concurrente, savepoint, historia y denegación con V3 falsa para clave existente y ausente. Usa un doble estructural del núcleo V3 y no acredita COSE real, cadena causal completa ni arranque del binario. La puerta completa corresponde a la integración de la vertical.

# Usuarios 5.08b: «Mis correos»

## Orden causal

Tras las SQL de 5.08a: AD3-107 `consumidor_correos_usuarios.up.sql` (seis acciones `vec.correos.{consultar,anadir,reenviar,verificar,activar,retirar}` y doce audiencias `vec_usuarios.correos.<accion>.{interna_corporativa|externa_personal}.v1`), después Usuarios 000004 `correos_propios.up.sql` y Usuarios 000005 `frontera_correos.up.sql`. No hay roles nuevos. Ningún `DOWN` sobre historia.

## Contrato

- La dirección se guarda cifrada (AES-GCM, AAD persona + correo_ref + versión) y con una huella HMAC de igualdad por persona para impedir duplicados; historia, recibos, envíos y auditoría solo llevan referencias opacas.
- El código de verificación (8 dígitos, CSPRNG) nunca llega a PostgreSQL: se guarda su HMAC ligado a persona, correo, desafío y vencimiento. Cada desafío admite 5 intentos, caduca (por defecto a la hora) y se usa una vez. Límites: 5 direcciones vivas, 5 códigos por persona y hora, 3 por dirección y hora.
- La verificación usa dos llamadas en la misma transacción: `preparar_verificacion_correo_v1` consume V3 y bloquea el desafío; Go compara el código en tiempo constante; `cerrar_verificacion_correo_v1` anota el intento o el éxito. Sin dirección activa, la primera verificada pasa a activa.
- `aplicar_correos_propios_v1` crea la salida de correo ya reservada en la misma transacción que el efecto; `confirmar_envio_correo_v1` anota una sola vez la respuesta del relay usando la reserva como llave (en la base solo queda su SHA-256). «aceptado» solo acredita aceptación del relay.
- Rotar la clave HMAC de igualdad bloquea nuevas altas (55000) de quien ya tenga direcciones hasta que una migración posterior reindexe sus huellas; la rotación de cifrado, huella semántica y código admite claves retenidas.
- La dirección activa no se retira (P1413): antes se elige otra. Un cambio de activa avisa a la anterior.
- SQLSTATE: `P1409` conflicto, `P1410` código caducado o agotado, `P1411` dirección ya registrada, `P1412` máximo de direcciones, `P1413` dirección en uso, `P1429` límite de códigos, `22023` petición inválida, `42501` denegación.

La prueba `pruebas_sql/correos_pg18.sh` levanta PostgreSQL 18.4 efímero con la V3 sintética de forma y comprueba ACL, RLS, vector Go/SQL, alta, replay, conflicto, intentos, activación, retirada, límites, confirmación de envío y la inmutabilidad de la historia.
