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
| A2 | **Consumidor de autorización del lote (AD190).** Fachada que consume la decisión del administrador para `administracion.perfiles.aplicar_lote_ordinario`, con la puerta de AUT45 (`acreditar_perfil_aplicacion_lote_ordinario_v1`) antes y después del consumo y la persona destinataria ligada al canon. Reconstruye el núcleo sobre la postimagen medida después de AD195 (Personal). | AUT45, AD195 instalada | Pendiente. |
| A3 | **Efecto del lote (CA35 + AUT44).** Una transacción SERIALIZABLE: consume la decisión (A2), compara por CAS todas las preimágenes, crea o revoca vínculos (CA35) y asignaciones, escribe sellos, historia, auditoría común, outbox y recibo. Replay con la misma referencia devuelve el recibo original; otra huella falla sin efecto. Nadie se asigna a sí mismo; un rol no ordinario anula el lote entero. | A1a, A2 | Pendiente. CA35 tiene borrador. |
| A4 | **Rol de Aplicación v6 en el entorno.** Sigue haciendo falta: AUT48 (#692) sólo hace que las lecturas acepten la versión vigente, pero la concesión `aplicar_lote_ordinario` existe únicamente en v6 y la puerta `acreditar_perfil_aplicacion_lote_ordinario_v1` está fijada a v6. Es el mantenimiento AUT45 v5→v6 con `vec-mantener-admin-fijo` (plan v2), después de v4→v5. Ya no hay que evitar v6: la lectura de usuarios funciona con ella. Sin código nuevo; se ensaya dentro de A9. | AUT48 (en main) | Pendiente (operación del runbook). |
| A5 | **Adaptador Go del lote y emisor de la decisión.** Rescatar `lote_ordinario.go`, `lote_ejecucion.go`, `lote_auditoria.go` y `lote_fuentes.go` del borrador, ajustados a A2/A3. Pruebas focales y de carrera. | A3 | Pendiente. |
| A6 | **Lecturas que necesita la pantalla.** Perfiles asignables vigentes (desde `rol_administrable_exacto_v1`) y preimagen del objetivo (versiones de persona, cuenta, procedencia y vínculo con su huella) para una persona. Sólo lectura, auditada como las demás lecturas de usuarios. | A1a, AUT48 | Pendiente. |
| A7 | **Composición en `vec-admin`.** Pool y LOGIN propios del lote, emisor y configuración privada; abre `POST /api/admin/perfiles/v1/lotes-ordinarios` sólo con autoridad real. Sin pool o sin permiso, 503 o 403 auditado. | A5, A6 | Pendiente. |
| A8 | **Pantalla.** En la ficha: «Asignar perfil» (perfil del catálogo, unidad si la exige, vigencia, motivo) y «Retirar» en cada perfil. Confirmación que muestra qué cambia, recibo al terminar y mensajes claros para conflicto, caducidad y denegación. Textos en `web/static/textos/{es,en}/`. Ayuda sólo tras el botón «?». | A7 | Pendiente. Requiere `usabilidad-vec`, `aspecto-vec`, `impeccable` y revisión de usabilidad independiente. |
| A9 | **Recorrido real.** Clon con el arranque 2+1, v6, perfiles asignables y `vec-admin`: asignar, retirar, reintento, conflicto de versión, intento denegado auditado, reinicio de `vec-admin` y PostgreSQL con el mismo recibo. Playwright con Chrome del sistema a 1440 y 390 px, en español e inglés. Actualizar el runbook. | A8 | Pendiente. |

Fuera de este primer corte, en cola: crear y versionar perfiles desde la
pantalla (contrato de gobierno de perfiles ya en `domain/administracion_gobierno_perfiles.go`,
sin SQL), alta y baja de administradores e Intervención con doble control
(propuesta y cierre de AUT24), selector de perfil activo para los perfiles
asignados y exportación de la auditoría con valor de prueba.

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
los planes hasta que se responda la pregunta 140 de `dudas.md`. Las exclusiones son
una lista de nombres; la clasificación positiva es la aprobación del plan. El
registro no da permisos a nadie: sólo hace que esos roles puedan asignarse
cuando existan A2-A7. `vec-admin` no cambia en este corte.
