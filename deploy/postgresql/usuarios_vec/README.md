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

# Usuarios 5.08c: «Mi imagen»

## Orden causal

Tras las SQL de 5.08a y 5.08b: AD3-108 `consumidor_imagen_usuarios.up.sql` (acciones `vec.imagen.consultar` y `vec.imagen.actualizar`, cuatro audiencias `vec_usuarios.imagen.<accion>.{interna_corporativa|externa_personal}.v1`), después Documentos 000007 `imagen_personal.up.sql`, Usuarios 000006 `imagen_propia.up.sql` y Usuarios 000007 `frontera_imagen.up.sql` (lista `deploy/principal/lista_sql_trabajo_usuarios_508c_20260929.txt`). No hay roles ni cuentas nuevos. Ningún `DOWN` sobre historia.

## Contrato

- Usuarios guarda solo la elección (`iniciales`, `icono` o `foto`, con paleta e icono de un vocabulario cerrado) y, en modo foto, la referencia opaca `docimg_…` y la huella SHA-256 que devuelve Documentos. Nunca los bytes.
- La aplicación recodifica la foto antes de pedir la autorización: solo admite JPEG, PNG y WebP reconocidos por su contenido, comprueba tamaño (1,4 MB tras la reducción que hace la interfaz; cabe en el límite común de 2 MB por petición), ancho y alto (8000 px) y píxeles (16 millones) antes de decodificar, recorta el centro y guarda un JPEG de 256 px sin EXIF, GPS ni ningún otro metadato. La V3 queda ligada a la huella de ese JPEG.
- `guardar_imagen_propia_v1(material, foto, …V3)` consume la V3 y, en la misma transacción, retira en Documentos la foto anterior (sus bytes se borran), custodia la nueva, aplica el CAS y escribe estado, historia y recibo. Modo foto sin foto nueva conserva la vigente y solo cambia la paleta.
- `consultar_imagen_propia_v1` devuelve el estado y, en modo foto, los bytes que Documentos entrega a su titular. Nadie más puede leer la foto: no hay acción de lectura ajena.
- El catálogo (`catalogo_imagen`, publicado por secuencia) puede ofrecer menos paletas o iconos, nunca otros.
- SQLSTATE: `P1409` conflicto (versión, catálogo, clave reutilizada o modo foto sin foto), `22023` petición inválida (foto que no casa con su huella o no es JPEG), `42501` denegación; `40001` obliga a repetir la transacción completa.

La prueba `pruebas_sql/imagen_pg18.sh` levanta PostgreSQL 18.4 efímero con la V3 sintética de forma y comprueba ACL y RLS de Usuarios y Documentos, el vector Go/SQL, elección, repetición, conflicto, subida, conservación, sustitución y retirada de la foto (bytes borrados en Documentos), la lectura desde la otra superficie y la inmutabilidad de las historias.

# B59: el aviso de llamamiento al correo activo de «Mis correos»

## Orden causal

Tras las SQL de 5.08a, 5.08b y 5.08c: AD3-109 `consumidor_correo_avisos_llamamiento.up.sql`, ContextoActor 000010 `persona_candidato_avisos.up.sql`, Bolsa 000059 `fuente_correo_llamamiento.up.sql` y Usuarios 000008 `correo_avisos_llamamiento.up.sql` (lista `deploy/principal/lista_sql_trabajo_avisos_mis_correos_20260929.txt`). Sin roles ni LOGIN nuevos. Ningún `DOWN` sobre historia.

## Contrato

- Cuando RRHH emite un llamamiento, Bolsa pregunta a Usuarios, por cada persona candidata, si tiene un correo activo. Usuarios responde con una sola lectura, `correo_activo_avisos_llamamiento_v1`, que sólo ejecuta el LOGIN ejecutor interno.
- La lectura consume una V3 fresca antes de mirar ninguna fila. El permiso es el mismo que el de emitir el llamamiento (acción `llamamiento.emitir.v1` sobre la bolsa constituida, finalidad `gestion_llamamientos_bolsa`), pero con audiencia propia (`vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1`) y perfil de consumo propio en el núcleo (AD3-109): la capacidad de emisión no abre esta lectura ni al revés. La huella del recurso fija el material exacto (bolsa, unidad, ámbito, llamamiento y referencia de candidato) y queda auditada con el consumo. Que la candidata sea de ese llamamiento lo comprueba Bolsa antes de preguntar (`candidato_participacion_avisos_v1`); Usuarios no puede comprobarlo sin leer tablas de Bolsa.
- La persona se obtiene de la referencia de candidato con la fachada de ContextoActor `persona_candidato_avisos_v1`, que sólo devuelve persona si hay exactamente un vínculo de candidato activo y vigente. Usuarios no lee tablas de identidad ni de Bolsa, y Bolsa no lee tablas de Usuarios.
- Sólo se entrega el sobre cifrado del correo ACTIVO y VERIFICADO que la persona añadió y confirmó desde el área personal externa: a una persona candidata sólo se le escribe al correo que dio en la superficie externa. Nada de la lista ni de otras direcciones. La dirección se descifra en Go y sólo existe durante el envío.
- Si no hay tal correo, o Usuarios no responde (presupuesto de 10 s por emisión y 2 s por consulta), el aviso sale al correo del alta en la bolsa, como antes, y el llamamiento no se bloquea. Bolsa guarda la fuente y el motivo (000059).
- SQLSTATE: `42501` denegación, `22023` material inválido; cualquier otro fallo se trata como «no disponible».

La prueba `pruebas_sql/correo_avisos_clon.sh <contenedor>` se ejecuta sobre un clon de la principal ya migrado, dentro de una transacción que termina en `ROLLBACK`, con un doble de la fachada AD3-109: comprueba material, huella, elección del correo, superficies, RLS, ACL y el registro de la fuente en Bolsa. No acredita COSE ni el núcleo V3.
