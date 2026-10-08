# Administración de perfiles: asignar y retirar perfiles desde la pantalla

Plan de continuación del 5 de octubre de 2026, sobre `main@be755f354`.
Primer corte pedido: que la persona administradora, desde la ficha de un
usuario en `vec-admin`, pueda **asignar** un perfil ya existente a una persona
y cuenta ya existentes y **retirarlo** después. Cada cambio debe quedar
autorizado a nombre de quien lo hace, con su decisión de autorización ligada,
y confirmarse en la misma transacción que el estado, la auditoría común y el
aviso de salida (outbox). Debe admitir reintentos sin duplicar efectos,
comprobar la versión de lo que se cambia y dejar constancia también de los
intentos denegados.

Este documento describe el orden de trabajo. No acredita instalación en
cidonia ni un recorrido real.

## Qué hay ya y qué se reutiliza

| Pieza | Dónde | Uso en este corte |
| --- | --- | --- |
| Lista y ficha de usuarios (sólo lectura) | `cmd/vec-admin`, AUT43/AD185, `web/static/admin/usuarios/` | Punto de partida de la pantalla. Otro ayudante está cambiando la lectura para aceptar la versión vigente del rol de Aplicación (AUT48); no se tocan esos ficheros hasta que entre su PR. |
| Contrato de «lote ordinario» de cambios de perfil | `internal/vec/domain/administracion_perfiles_lote.go`, `application/administracion_perfiles_lote.go`, `ports/administracion_perfiles_lote.go` | Una asignación o una retirada es un lote de un solo cambio. No se crea otra vía singular. |
| Ruta `POST …/lotes-ordinarios` | `internal/vec/adapters/httpapi/administracionperfiles/handler_lotes.go` | Ya valida y deniega con auditoría; falta la autoridad real detrás. |
| Permiso del administrador para el lote | AUT45: rol de Aplicación v6 con `administracion.perfiles.aplicar_lote_ordinario` | Se publica con la CLI existente `vec-mantener-admin-fijo` (paso 4 del runbook, plan v2). Requiere AUT48 para que la lista de usuarios siga funcionando con v6. |
| Alta y baja del vínculo persona-perfil | CA20 (`crear_perfil_vinculo_admin_v1`, `revocar_perfil_vinculo_admin_v1`) y el borrador CA35 de K | CA35 añade la fecha única del lote. |
| Actos singulares AUT24 (`aplicar_acto_ordinario_admin_v1`) | AUT24 | **No sirven tal cual**: exigen las acciones `administracion.perfiles.otorgar/revocar`, que ninguna versión del rol concede, y escriben auditoría propia en lugar de la común. Se reutilizan sus tablas de catálogo, sellos y preimagen. |
| Borrador del lote de K | `origin/trabajo/codexk-admin-lote-ordinario-20261004@25cc6f444` | Adaptador Go, contrato v3 con fechas y fuentes, DTO HTTP y CA35. Faltan AUT44 y AD190 en SQL. Se rescata por piezas, no se fusiona la rama. |

**Hueco de base detectado.** El catálogo `rol_administrable_exacto_v1` sólo
contiene los roles de administración y Sistemas. No hay ningún perfil
ordinario que la pantalla pueda ofrecer, así que ninguna asignación podría
salir bien. Por eso la primera minitarea es el registro de perfiles
asignables.

## Grafo de minitareas

Cada minitarea es una PR propia, en este orden salvo donde se indica.

| Id | Minitarea | Depende de | Estado |
| --- | --- | --- | --- |
| A1a | **Perfiles asignables (SQL).** AUT49 + AD196: un operador técnico, con LOGIN propio y una aprobación ligada a la huella del plan, registra versiones de rol ordinarias ya publicadas como asignables. Auditoría común en la misma transacción; replay con el mismo recibo; intentos denegados auditados. Excluye administración, Sistemas, Intervención, roles sensibles y fijos. | — | #699, en main; ensayada en clon, dos revisiones. |
| A1b | **Verificador de la cadena para los tipos nuevos de AD196** (#704). Esquema nuevo en `internal/vec/auditoria` y `cmd/vec-auditoria-verificar` que acepte todos los tipos anteriores más `perfiles_asignables_admin` e `intento_perfiles_asignables_admin`. | A1a | #704, en main. |
| A2 | **Consumidor de autorización del lote (AD190).** Perfil de mutación `lote_perfiles_admin` en el núcleo, audiencia `vec_autorizacion.administracion_perfiles.lote_ordinario.v1`, recurso persona destinataria distinta del actor, campos `[]`, obligación `auditar`, superficie ADMIN con cuenta privilegiada. Puerta del lote de AUT45 (generalizada por AUT51 a la versión vigente del rol con la concesión exacta) antes y después del consumo y revalidación de la decisión viva. Sólo el propietario de Autorización ejecuta la fachada y el núcleo exige que la sesión sea un LOGIN exclusivo del grupo nuevo `vec_admin_perfiles_lote_ejecutor`. | AUT45, AD195, AD178/AD177, AUT51 | En PR (#720), medida tras AD178/AD177 y AUT51; vector estructural y negativo en clon. El consumo positivo necesita el emisor real y se acredita en A9. |
| A3 | **Efecto del lote (CA35 + AUT44).** Una transacción SERIALIZABLE: recalcula el recurso (persona, organización, unidad y huella de la solicitud) y lo exige igual en la decisión, la capacidad y el acuse del consumo AD190; compara TODAS las preimágenes con el estado anterior al lote y después aplica: registra la procedencia del acto, crea o revoca vínculos (CA35) y asignaciones con su historia, y escribe registro, outbox y recibo. Replay con el mismo recibo; otra huella, 23505. | A1a, A2 | En PR (#721); vector de 18 casos en clon sobre main con AD178/AD177 y AUT51, arranque 2+1 y Rol7 (con el consumo AD190 sustituido sólo dentro de la prueba). |
| A4 | **Rol de Aplicación con la concesión del lote.** Mantenimiento a la versión vigente que tenga `administracion.perfiles.aplicar_lote_ordinario` (hoy v6 con AUT45; AUT51 la generaliza a la versión vigente con la concesión exacta y prepara v7). Tras el cambio de versión, cada administrador vuelve a elegir perfil para que su asignación actual sea la nueva. Operación de dirección en cidonia, sin código aquí. | AUT48 | Pendiente (operación). |
| A5a | **Contrato v3 del lote (sin SQL).** Rescate del borrador de K (`25cc6f444`): organización fijada por configuración, inicio inmediato o programado, unidad obligatoria, recibo con huella de fuentes e inicios efectivos; DTO HTTP estricto y `NuevoHandlerLoteOrdinario`. Ninguna autoridad nueva: el efecto sigue cerrado (503). | — | En esta PR. |
| A5b | **Adaptador Go del lote y emisor de la decisión.** Rescate de `lote_ordinario.go`, `lote_ejecucion.go`, `lote_auditoria.go`, `lote_fuentes.go` y `lote_material_fuentes.go` del borrador de K, que ya llaman a `aplicar_lote_ordinario_admin_v1` y `acreditar_login_lote_ordinario_admin_v1` con la firma de AUT44, y emisor V3 nuevo `internal/app/administracion/emisor_lote.go` (concesión exacta del lote en la versión del rol asignada, sesión ADMIN privilegiada, correlación del acceso actual y huella de la solicitud). El adaptador acepta la versión vigente del rol de Aplicación, no emite decisión sin fuentes de ámbito y clasifica los SQLSTATE de AUT44 (42501 denegado; 40001, 40P01, 55P03, 55000, P0002 y 23505 conflicto, 409 auditado como error; 22023, 22P02, 22007 y 23514 inválido) y deja que las bajas no exijan que el perfil siga ofreciéndose. Sin composición. | A3 | #722; segunda revisión GO. |
| A6 | **Lecturas que necesita la pantalla.** (a) Perfiles asignables vigentes desde `rol_administrable_exacto_v1`; (b) preparación del cambio: cuenta, versiones de persona y cuenta, procedencia y huella de la preimagen para una persona, un perfil y una unidad, con referencias nuevas de perfil y vínculo. La ficha actual (AUT43) no da esos datos a propósito y el Go de `listar_roles_admin_v1`/`consultar_persona_admin_v1` no tiene SQL. Decisión: ambas lecturas consumen una decisión de autorización por el mismo consumidor que el lote (AD190), con la acción `administracion.perfiles.aplicar_lote_ordinario` y un efecto «preparar» distinto del «aplicar»; así no hace falta una v7 del rol. Por eso A6 también espera a AD195. | A1a, A2 | Pendiente. |
| A7 | **Composición en `vec-admin`.** En cuatro PR apiladas. A7a (#729): preparación en Go (dominio, puerto, adaptador AUT50, emisor con atributo `preparacion_sha256`; los fallos se auditan con la acción `administracion.perfiles.preparar_lote_ordinario`). A7b (#730): servicio del lote y HTTP: `GET /personas/{per}/preparacion-lote?unidad_ref=` y `POST /lotes-ordinarios`; el resto de escrituras, 404 auditado; sin la concesión del lote, 403 en GET y en POST; destino de auditoría propio `preparar_lote_ordinario`. A7c (#731): overlay privado `VEC_ADMIN_LOTE_CONFIG_FILE` con pool y LOGIN propios, cadena V3 de una capacidad, emisor, fuentes de ámbito por unidad y servicio. A7d: gobierno genérico de claves de capacidad ADMIN (AD198, conjunto 1 = usuarios + lote) con la renovación diaria de `vec-gobierno-usuarios-admin`. | A5, A6 | GO de revisión independiente en A7a–A7c y GO SQL en A7d (tras corregir su P1 de auditoría). Pendiente: P2 sin resolver, abajo. |
| A8 | **Pantalla.** En la ficha: «Asignar perfil» (perfil del catálogo, unidad si la exige, vigencia, motivo) y «Retirar» en cada perfil. Confirmación que muestra qué cambia, recibo al terminar y mensajes claros para conflicto, caducidad y denegación. Textos en `web/static/textos/{es,en}/`. Ayuda sólo tras el botón «?». | A7 | Candidato #733 publicado en su rama (`4c054733807b7b873d72e0aa55052cd37e2c55cf`), con dos revisiones independientes GO; pendiente de fusión. Node 30/30. Chrome del sistema con fixture sintético ES/EN a 1440/390 px comprobó los formularios, el bloqueo durante un resultado incierto y el reintento del mismo cuerpo y `operacion_ref`, sin desbordamiento. A9 mantiene pendientes el consumo nominal, el recorrido durable y la recuperación tras reinicio. |
| A9 | **Recorrido real.** Además de lo de la composición, el consumo positivo del lote exige una fila en `vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1` para el LOGIN del lote (audiencia `vec_autorizacion.administracion_perfiles.lote_ordinario.v1`, acción `administracion.perfiles.aplicar_lote_ordinario`, superficie `administracion_privilegiada`) y que el pool fije `statement_timeout` ≤ 15 s e `idle_in_transaction_session_timeout` ≤ 20 s en la conexión. Clon con el arranque 2+1, v6, perfiles asignables y `vec-admin`: asignar, retirar, reintento, conflicto de versión, intento denegado auditado, reinicio de `vec-admin` y PostgreSQL con el mismo recibo. Playwright con Chrome del sistema a 1440 y 390 px, en español e inglés. Actualizar el runbook. | A8 | Pendiente. |

Fuera de este primer corte, en cola: crear y versionar perfiles desde la
pantalla (contrato de gobierno de perfiles ya en `domain/administracion_gobierno_perfiles.go`,
sin SQL), alta y baja de administradores e Intervención con doble control
(propuesta y cierre de AUT24), selector de perfil activo para los perfiles
asignados y exportación de la auditoría con valor de prueba.

## Decisiones de A3

- **Procedencia del acto.** CA20 exige una procedencia nueva para revocar y sólo las
  fuentes iniciales registraban procedencias. Cada lote confirmado registra la suya
  (`prc_` derivada de la operación, huella = la de la solicitud) en `procedencias` y,
  aparte, en `procedencia_acto_admin_v1`; CA35 sólo acepta para el alta y la baja de
  vínculos del lote procedencias de esa tabla. Que CA32/CA33 las excluyan queda en
  `pendientes_v2.md`. La preimagen del objetivo sigue llevando la procedencia actual
  de la persona, para el CAS.
- **Persona administrable.** La destinataria tiene que estar en el conjunto que ve el
  administrador (alguna asignación actual con la misma organización y unidad, y
  metadatos de persona administrable), como en la lista de usuarios (AUT43). El
  administrador tiene que tener esa organización y esa unidad en su asignación.
- **Ámbitos de la asignación nueva.** Organización y unidad cotejadas con sus fuentes
  propietarias (AUT37/Personal), las mismas que la lista de usuarios.
- **Cuenta.** El vínculo se crea con la cuenta de la preimagen, que tiene que ser una
  cuenta ordinaria activa de la persona: AUT44 rechaza la privilegiada.
- **Configuración del perfil.** Para dar un perfil, su registro en AUT49 tiene que tener
  la audiencia del lote y como ámbito fijo exactamente la organización del lote; la
  unidad la añade cada asignación. Para retirarlo basta que la versión esté registrada
  como ordinaria, aunque ya no se ofrezca.
- **Anular un alta programada** antes de que empiece no es posible todavía
  (`pendientes_v2.md`).

## Deuda anotada de A7 (P2 de revisión)

- La prueba del canon de la preparación no usa un vector fijo (canon y huella
  esperados) generado desde una solicitud válida. La equivalencia con AUT50
  está comprobada por lectura y por el ensayo; falta fijarla en una prueba.
- Un dato raro del catálogo de perfiles tumba la preparación entera, siempre en
  cerrado: una duración propuesta de más de diez años, un nombre de rol con `.`
  o `:` (que AUT44 sí admite) o una fecha de baja que no se pueda leer. Además,
  la preparación no coteja la unidad pedida con las del overlay del lote: si no
  está, el error llega al aplicar.
- AD198 no tiene clave ajena de su registro de operaciones a la auditoría común.
- Tras cada renovación diaria hay que reconfigurar y reiniciar `vec-admin`, y no
  dejar LOGIN en el grupo operador de AD188 (la vía antigua no renueva la clave
  del lote).

## A1a: registro de perfiles asignables

Ficheros: `deploy/postgresql/autorizacion_atestada_v3/migraciones/000196_registro_perfiles_asignables_admin.up.sql`,
`deploy/postgresql/autorizacion/migraciones/000049_perfiles_asignables_admin.up.sql`,
su vector `deploy/postgresql/autorizacion/pruebas_sql/perfiles_asignables_000049.sql`
y el procedimiento `deploy/postgresql/autorizacion/PERFILES_ASIGNABLES_AUT49.md`.

Por qué un acto técnico y no la pantalla: el rol del administrador no concede
hoy registrar perfiles, y crear esa concesión exige otra versión del rol. El
registro es una decisión de gobierno (qué perfiles se pueden repartir), no un
reparto. Lo aprueba Alberto sobre la huella exacta del plan.

Ensayo del 5 de octubre de 2026 en un clon desechable (PostgreSQL 18.4,
`--memory 2g`) restaurado del frío `frio-runtime-IS16-CA36-20261005` de K más
AD193 y AUT47, equivalente a la principal H12:

- AD196 y AUT49 se instalan una vez, sin errores.
- El vector termina con 24 casos en verde: registro, replay, huella distinta,
  administración, Sistemas, Intervención (también el lector de firmas para
  fiscalización), aspirantes y usuarios externos, control divergente, ya
  registrado, plan caducado, fechas nulas, claves repetidas, duplicado, LOGIN
  sin configuración o con permisos de más, ACL, inmutabilidad, aprobación en la
  auditoría y cadena enlazada.
- Registro real por TLS 1.3 con un LOGIN técnico: tres perfiles. Tras
  reiniciar PostgreSQL, el replay devuelve el mismo recibo y sólo añade su
  intento. Un intento con huella falsa queda auditado como denegado después
  del COMMIT.
- Dos revisiones independientes (SQL y seguridad). Sus hallazgos están
  corregidos: exclusiones por identificador de rol y no sólo por versión,
  Intervención y externos ampliados, plan canónico, fechas no nulas, ventana
  de un día como máximo, aprobación dentro de la cadena, límites de espera y
  código propio para la comparación de versiones.

Límites: el verificador Go de los tipos nuevos ya está en main (#704).
`organizacion_preparacion` y `entrega-peticion-rrhh-fijo` quedan fuera de
los planes hasta que se responda la pregunta 141 de `dudas.md`. Las exclusiones son
una lista de nombres; la clasificación positiva es la aprobación del plan. El
registro no da permisos a nadie: sólo hace que esos roles puedan asignarse
cuando existan A2-A7. `vec-admin` no cambia en este corte.


## Preferencias sin espera inicial con idioma elegido — 8 de octubre de 2026

En Área personal, una URL con idioma elegido carga primero el idioma y Mi
Bolsa; las preferencias se consultan después. Sin idioma en la URL se conserva
la lectura previa, necesaria para mostrar desde el principio el idioma guardado.
Las preferencias sólo cambian la presentación; las capacidades y los datos
operativos siguen procediendo de sus consultas autorizadas.

La lectura inicial se comparte con «Recargar» y las peticiones de preferencias,
imagen y correos se serializan. Las demás lecturas de Bolsa conservan su
transporte y límite propios. Una respuesta tardía actualiza sólo el panel de
preferencias y conserva los datos, la navegación elegida, el foco y los
borradores de los paneles hermanos. Tras un fallo se puede volver a cargar.

La revisión independiente encontró y corrigió el límite de 64 KiB heredado por
el historial, la petición inicial duplicada y el foco perdido al reconstruir
los paneles. Chrome comprobó ES a 1440 px y EN a 390 px con una API local de
prueba: Mi Bolsa apareció en 126 ms aunque preferencias tardara 1 segundo.
También se comprobaron 401/403, fallo y reintento 503, idioma guardado sin URL,
una respuesta válida de historial superior a 64 KiB y los cambios de foco.
Esto acredita el arranque del navegador con esa fixture; no mide PostgreSQL,
la autenticación nominal ni un GET de preferencias inferior a 50 ms.

La reducción de ese GET sigue pendiente de la sesión en memoria ligada al
certificado y el apunte común transaccional. El recorrido actual encadena ocho
llamadas SQL nominales y al menos seis transacciones. S conserva la captura de
sesión y W el consumidor común; este corte no crea otra autoridad ni SQL nuevo.
