# UX portal empleado tipo VEC

## Alcance

Documento de diseno funcional para un portal de empleado/candidato tipo VEC
aplicado a la bolsa de empleo. No abre alcance de implementacion ni cambia la
persistencia del prototipo actual.

Refs opacas preservadas: `worktree-bolsa-vec-portal-005` y
`branch-bolsa-vec-portal-005`.

## Principios

- Portal unico con navegacion por rol: candidato, empleado interno y
  administrador.
- Nucleo neutral: estados de expediente, meritos, autobaremo, alegaciones y
  notificaciones no dependen de HTTP, proveedor de identidad, base de datos ni
  rutas locales.
- Puertos pequenos: identidad, expediente, catalogo de convocatorias,
  documentos, mensajes, noticias y notificaciones.
- Adaptadores opt-in: Clave/certificado para candidato, Kerberos/AD para
  personal interno, almacenamiento documental y canal de notificacion.
- Composicion canonica: configuracion de endpoints, idioma, formatos de fecha,
  limites de fichero y canales activos en una unica superficie.
- i18n desde el inicio: textos, estados, errores y plantillas en catalogo
  traducible, con castellano como idioma base.

## Mapa de navegacion

### Candidato

| Area | Pantallas | Objetivo |
| --- | --- | --- |
| Inicio | Dashboard | Ver estado de solicitudes, avisos, plazos y acciones pendientes. |
| Convocatorias | Listado, detalle, inscripcion | Consultar bases, iniciar solicitud y revisar requisitos. |
| Mi expediente | Resumen, documentos, meritos, autobaremo | Mantener datos y evidencia aportada. |
| Alegaciones | Listado, nueva alegacion, detalle | Responder a listados provisionales o requerimientos. |
| Mensajes | Bandeja, detalle, archivo | Leer comunicaciones oficiales y avisos operativos. |
| Noticias | Listado, detalle | Consultar novedades publicas de la bolsa. |
| Perfil | Datos personales, contacto, preferencias | Revisar identidad, telefono, email, direccion y canal preferente. |
| Ayuda | FAQ, soporte, accesibilidad | Resolver dudas sin bloquear tramite. |

### Empleado interno

| Area | Pantallas | Objetivo |
| --- | --- | --- |
| Inicio | Dashboard operativo | Ver expedientes asignados, alertas y carga de trabajo. |
| Solicitudes | Busqueda, detalle, revision | Validar datos, documentos, meritos y estados. |
| Alegaciones | Cola, detalle, propuesta | Resolver alegaciones con trazabilidad. |
| Mensajes | Plantillas, envio, seguimiento | Comunicar requerimientos y resoluciones. |
| Informes | Listados, exportaciones | Preparar listados provisionales/definitivos y auditoria. |

### Administrador

| Area | Pantallas | Objetivo |
| --- | --- | --- |
| Configuracion | Convocatorias, baremos, plazos | Parametrizar reglas publicadas sin tocar codigo. |
| Usuarios | Roles, permisos, delegaciones | Controlar acceso de personal interno. |
| Noticias | Alta, edicion, publicacion | Gestionar comunicados visibles en portal. |
| Auditoria | Eventos, cambios, descargas | Revisar trazabilidad y evidencias. |

## Dashboard de candidato

Contenido prioritario:

- Tarjeta de solicitud activa: convocatoria, estado, fecha de ultimo cambio y
  siguiente accion.
- Avisos criticos: plazo abierto, subsanacion requerida, listado provisional,
  alegacion pendiente o resolucion disponible.
- Acciones primarias: continuar inscripcion, completar autobaremo, aportar
  documento, presentar alegacion o revisar mensaje.
- Resumen de puntuacion: total provisional, desglose por seccion y avisos de
  meritos no validados.
- Timeline: presentada, registrada, en revision, provisional, alegaciones,
  definitiva y cerrada.
- Acceso rapido a perfil/contacto para evitar notificaciones fallidas.

Estados vacios:

- Sin solicitud: mostrar convocatorias abiertas y acceso a noticias.
- Sin mensajes: indicar que no hay comunicaciones pendientes.
- Sin meritos: guiar a anadir merito desde el expediente.

## Mensajes y noticias

### Mensajes

- Bandeja con filtros: no leidos, oficiales, requerimientos, resoluciones,
  archivados.
- Cada mensaje muestra asunto, fecha, expediente asociado, nivel de urgencia y
  acuse si aplica.
- Detalle con cuerpo, adjuntos, accion vinculada y estado de lectura.
- Mensajes con requerimiento incluyen plazo, consecuencia de no responder y
  enlace directo al tramite.

### Noticias

- Listado publico con categoria, fecha y convocatoria relacionada.
- Detalle con contenido, adjuntos y enlace a bases o anuncios.
- Noticias administrativas solo editables por rol administrador.
- Separacion clara entre noticia informativa y mensaje oficial con efectos de
  tramite.

## Perfil y datos de contacto

Pantallas:

- Datos personales: identificador, nombre, apellidos y fecha de nacimiento como
  datos provenientes de identidad; si son solo lectura, se explica origen.
- Contacto: email, telefono movil, direccion postal y canal preferente.
- Preferencias: idioma, formato de aviso y accesibilidad visual si procede.
- Seguridad: sesiones activas, ultimo acceso y mecanismos de autenticacion
  disponibles.

Reglas UX:

- Validacion inline para email, telefono y campos obligatorios.
- Confirmacion antes de guardar cambios con impacto en notificaciones.
- Registro de cambios de contacto con fecha, usuario y origen.
- Mensaje claro si un dato solo puede corregirse en el proveedor de identidad.

## Flujo de inscripcion y autobaremo

1. Candidato accede con identidad valida.
2. Portal muestra convocatorias abiertas compatibles.
3. Candidato abre detalle: bases, requisitos, plazos, baremo y documentacion.
4. Inicia solicitud y confirma datos personales/contacto.
5. Aporta meritos por seccion: experiencia, formacion y otros.
6. Por cada merito, adjunta evidencia, indica datos requeridos y acepta
   declaracion responsable.
7. Sistema calcula autobaremo provisional con desglose y topes.
8. Candidato revisa alertas: falta documento, formato invalido, merito fuera de
   plazo o puntuacion limitada por maximo.
9. Candidato presenta solicitud.
10. Portal emite justificante, numero de registro y resumen descargable.

Puntos de control:

- Guardado borrador en cada paso.
- Barra de progreso por seccion.
- Resumen final antes de presentar.
- Bloqueo explicito de edicion tras presentacion, salvo subsanacion o plazo
  permitido.

## Flujo de alegaciones

1. Administracion publica listado provisional o requerimiento.
2. Portal notifica al candidato y muestra plazo exacto.
3. Dashboard activa accion "Presentar alegacion".
4. Candidato selecciona expediente y motivo: puntuacion, merito excluido,
   error de datos, documento no valorado u otro.
5. Candidato redacta alegacion, relaciona meritos afectados y adjunta evidencia.
6. Sistema valida formato, tamano, campos obligatorios y plazo.
7. Candidato revisa resumen y presenta.
8. Portal emite justificante y deja la alegacion en estado registrada.
9. Empleado interno revisa, propone resolucion y deja trazabilidad.
10. Candidato recibe resolucion y ve efecto en puntuacion/listado.

Estados:

- Borrador, registrada, en revision, requiere subsanacion, estimada,
  estimada parcialmente, desestimada y archivada.

Errores recuperables:

- Plazo cerrado: explicar fecha de cierre y via de consulta.
- Documento rechazado: indicar formato/tamano permitido.
- Sesion caducada: conservar borrador local/servidor si es posible.
- Conflicto de version: mostrar ultima version y pedir revalidacion.

## Estados de expediente

| Estado | Lectura para candidato | Accion principal |
| --- | --- | --- |
| Borrador | Solicitud no presentada. | Continuar o descartar. |
| Presentada | Solicitud registrada. | Descargar justificante. |
| En revision | Administracion revisa documentacion. | Consultar mensajes. |
| Subsanacion requerida | Falta informacion o documento. | Aportar subsanacion. |
| Provisional publicado | Resultado no definitivo. | Revisar y alegar. |
| En alegaciones | Alegacion registrada o en revision. | Consultar estado. |
| Definitivo publicado | Resultado final disponible. | Descargar resolucion. |
| Cerrada | Expediente finalizado. | Consultar historico. |

## Accesibilidad y responsive

- WCAG 2.2 AA como referencia minima.
- Navegacion completa por teclado y foco visible.
- Contraste suficiente en estados, alertas y botones.
- Iconos siempre acompanados de texto o alternativa accesible.
- Formularios con labels persistentes, ayudas y errores asociados al campo.
- Tablas con cabecera, ordenacion accesible y alternativa de tarjetas en movil.
- No depender solo de color para diferenciar estados.
- Objetivos tactiles minimos de 44 x 44 px en movil.
- Layout responsive:
  - movil: navegacion inferior o menu compacto, acciones prioritarias arriba;
  - tablet: dos columnas para resumen y detalle;
  - escritorio: sidebar estable, panel principal y panel contextual.

## Errores y confirmaciones

- Error de autenticacion: explicar mecanismo requerido y permitir reintento.
- Error de permisos: indicar que el usuario no tiene rol para la accion.
- Error de validacion: marcar campo, causa y reparacion.
- Error tecnico: mostrar codigo de incidencia, hora y accion segura.
- Confirmaciones obligatorias: presentacion de solicitud, envio de alegacion,
  cambio de contacto y descarte de borrador.
- Operaciones irreversibles deben requerir doble confirmacion y resumen de
  impacto.

## Fronteras para implementacion futura

Nucleo:

- Entidades: expediente, convocatoria, merito, documento, mensaje, noticia,
  alegacion, perfil de contacto y evento de auditoria.
- Reglas: transiciones de estado, validacion de plazo, calculo de autobaremo,
  elegibilidad de alegacion y acuse de mensaje.

Puertos:

- `IdentityProvider`, `ExpedientRepository`, `DocumentStore`,
  `NotificationChannel`, `NewsRepository`, `AuditLog` y `Clock`.

Adaptadores:

- HTTP/API para portal.
- Proveedor de identidad institucional.
- Almacen documental.
- Canal email/SMS/notificacion interna.
- Repositorio duradero cuando el alcance productivo lo pida.

Composicion:

- Configuracion unica para idioma por defecto, limites de adjuntos, canales de
  notificacion, URLs publicas, plazos visibles y mecanismos de autenticacion.

## Criterios de aceptacion UX

- Candidato entiende en menos de una pantalla que debe hacer hoy.
- Toda accion con plazo muestra fecha/hora y consecuencia.
- Autobaremo explica puntuacion total, desglose y topes aplicados.
- Alegacion puede completarse desde listado provisional sin navegar por menus
  profundos.
- Mensaje oficial y noticia informativa no se confunden.
- Perfil permite corregir contacto antes de presentar o alegar.
- Administrador puede mapear convocatorias, baremos y noticias sin mezclar
  funciones de revision diaria.
- Empleado interno puede resolver colas sin ver configuracion global salvo que
  tenga rol.
