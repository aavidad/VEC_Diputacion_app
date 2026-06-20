# Cumplimiento, seguridad y expediente electronico

## Alcance

Este documento aterriza requisitos implantables para el portal VEC de bolsa de
empleo. No sustituye informe juridico ni certificacion ENS; fija controles de
producto, roles, evidencias y pruebas de frontera que deben guiar el paso desde
el prototipo actual a un servicio publico operable.

Contexto de app revisado:

- Prototipo Go con frontera hexagonal: dominio, puertos, casos de uso,
  adaptadores HTTP/repositorios en memoria/auth fake y composicion en
  `cmd/bolsa-server`.
- Flujo ciudadano: alta de candidato, meritos, baremo y exportacion de
  expediente.
- Flujo administrativo: demo de convocatoria, solicitudes, listados provisional
  y definitivo.
- Ya existen modelos de dominio para auditoria encadenada y evidencias de
  documento electronico; falta integracion productiva con identidad,
  persistencia, firma, notificacion, archivo, auditoria externa y ENS.

Refs opacas preservadas: `portal-vec-cumplimiento-005`,
`worktree-bolsa-vec-portal-005` y `branch-bolsa-vec-portal-005`.

## Marco normativo minimo

| Norma | Impacto en el portal |
| --- | --- |
| RGPD, Reglamento (UE) 2016/679 | Licitud, minimizacion, transparencia, derechos de personas interesadas, seguridad del tratamiento, privacidad desde el diseno y por defecto, encargados y brechas. |
| LOPDGDD, Ley Organica 3/2018 | Adaptacion espanola del RGPD, tratamiento por administraciones publicas, delegado de proteccion de datos, derechos digitales y regimen sancionador. |
| ENS, Real Decreto 311/2022 | Politica de seguridad, categorizacion, analisis de riesgos, medidas organizativas/operacionales/de proteccion, auditoria y declaracion/certificacion de conformidad. |
| Ley 39/2015 | Derechos de las personas en el procedimiento, identificacion y firma, registros, terminos/plazos, notificaciones, validez de documentos, expediente administrativo y recursos. |
| Real Decreto 203/2021 | Actuacion y funcionamiento electronico del sector publico: sede, identificacion, firma, registro, copias, archivo, notificacion, expediente y relaciones electronicas. |
| ENI, Real Decreto 4/2010 | Interoperabilidad organizativa, semantica y tecnica; reutilizacion, conservacion, firma, certificados, documento electronico y expediente electronico. |
| NTI de documento, expediente, digitalizacion, copiado autentico, politica de firma, catalogo de estandares y reutilizacion | Metadatos obligatorios, formatos, firmas, indices, foliado/logica de expediente, intercambio y conservacion a largo plazo. |
| Accesibilidad: RD 1112/2018 y EN 301 549/WCAG aplicable | Portal perceptible, operable, comprensible y robusto; declaracion de accesibilidad y canal de reclamaciones. |

Fuentes oficiales de contraste:

- BOE Ley 39/2015: https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565
- BOE RD 203/2021: https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032
- BOE RD 311/2022 ENS: https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191
- BOE LO 3/2018 LOPDGDD: https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673
- BOE RD 4/2010 ENI: https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331

## Modelo de roles y autorizacion

| Rol | Identidad esperada | Permisos minimos | Evidencia obligatoria |
| --- | --- | --- | --- |
| Ciudadano/candidato | Clave, DNIe/certificado o mecanismo admitido por la sede | Crear solicitud propia, aportar meritos, consultar estado, recibir notificaciones, descargar justificantes y ejercer derechos RGPD. | Actor, mecanismo, fecha, IP/origen tecnico, solicitud, documento o tramite afectado. |
| Representante | Registro de apoderamientos o acreditacion aceptada | Actuar por cuenta de persona representada dentro del alcance del poder. | Vinculo representado-representante, vigencia y tramite autorizado. |
| Personal tramitador | Directorio corporativo/MFA | Revisar solicitudes, requerir subsanacion, validar meritos, preparar listados. | Unidad, rol, accion, motivo, expediente y payload hasheado. |
| Tribunal/comision | Directorio corporativo/MFA y nombramiento | Validar criterios, resolver alegaciones, aprobar listados y propuestas. | Acta/acuerdo, version de baremo, votos o aprobacion cuando aplique. |
| Administrador tecnico | Cuenta nominal privilegiada/MFA | Operar configuracion, despliegue, copias, claves y observabilidad sin alterar expediente material. | Ticket/cambio, comando o accion, ventana temporal y doble control si aplica. |
| Auditor/DPO/seguridad | Acceso segregado de lectura | Revisar trazabilidad, riesgos, DPIA, incidencias, accesos y cumplimiento. | Consulta justificada, filtros aplicados y exportacion controlada. |

Reglas:

- No usar cuentas compartidas para acciones con efecto juridico.
- Separar funciones: tramitacion, administracion tecnica, auditoria y
  resolucion no deben depender del mismo permiso amplio.
- Cada endpoint debe comprobar rol y titularidad en caso de datos personales.
- Los tokens o sesiones no son evidencia juridica por si solos; deben mapearse
  a actor, mecanismo de identificacion y contexto.
- El dominio debe permanecer neutral: identidad real, directorio, Clave, DNIe,
  firma, sello, TSA, registro y archivo son adaptadores opt-in mediante puertos.

## Proteccion de datos RGPD/LOPDGDD

Checklist implantable:

- Inventariar tratamientos: gestion de bolsa, baremacion, comunicaciones,
  alegaciones, archivo y auditoria.
- Definir base juridica por tratamiento: obligacion legal, mision de interes
  publico/poderes publicos o consentimiento solo cuando sea realmente libre.
- Publicar informacion por capas: responsable, DPO, fines, legitimacion,
  destinatarios, conservacion, derechos y vias de reclamacion.
- Minimizar datos: DNI/NIE, contacto, meritos, discapacidad u otros datos
  especialmente protegidos solo si la convocatoria lo exige y con medidas
  reforzadas.
- Aplicar privacidad por defecto: listados publicos con identificacion
  parcializada cuando proceda, acceso autenticado al expediente completo y
  descargas con caducidad o control de acceso.
- Mantener registro de actividades de tratamiento y contratos de encargado si
  hay proveedores de hosting, correo, firma, custodia, observabilidad o soporte.
- Ejecutar analisis de riesgos de privacidad y DPIA si hay tratamiento masivo,
  decisiones automatizadas con efectos significativos o categorias especiales.
- Habilitar derechos: acceso, rectificacion, oposicion, limitacion y supresion
  cuando proceda, con circuito trazable y plazo controlado.
- Definir retencion: expediente activo, fase de recursos, archivo y bloqueo.
  Borrado o anonimizacion solo tras cumplir normativa de archivo y control.
- Gestionar brechas: deteccion, clasificacion, contencion, notificacion a la
  autoridad y comunicacion a personas afectadas cuando aplique.

Requisitos tecnicos:

- Cifrado en transito con TLS moderno y cabeceras seguras.
- Cifrado o proteccion equivalente en reposo para documentos, backups y
  secretos.
- Seudonimizacion en logs: no registrar documentos completos, DNI completo,
  telefonos, correos o payloads de meritos salvo necesidad justificada.
- Trazabilidad de consultas administrativas a expedientes.
- Entornos separados; datos productivos no deben copiarse a desarrollo sin
  base juridica y medidas de anonimizado.

## ENS y seguridad operacional

Controles de cierre:

- Categorizar sistema por disponibilidad, autenticidad, integridad,
  confidencialidad y trazabilidad.
- Aprobar politica de seguridad, responsables, procedimiento de autorizacion y
  aceptacion de riesgos.
- Mantener analisis de riesgos vivo por convocatoria y por cambio relevante.
- Implantar MFA para personal interno y administradores.
- Aplicar minimo privilegio, revision periodica de permisos y baja automatica
  por cambio de puesto.
- Centralizar logs de seguridad, auditoria de negocio y eventos de sistema con
  integridad y retencion definidas.
- Tener copias probadas, restauracion ensayada y plan de continuidad.
- Gestionar vulnerabilidades: SCA, SAST razonable, parcheo, hardening de
  imagenes y dependencias, pentest si el riesgo lo exige.
- Separar configuracion canonica en composicion; no duplicar variables ni
  secretos en dominio, tests o adaptadores.
- Documentar declaracion/certificacion ENS y evidencias de auditoria segun
  categoria.

Medidas concretas para esta app:

- Sustituir autenticador fake por puerto de autenticacion real manteniendo
  `ports.Authenticator`.
- Sustituir repositorios en memoria por adaptadores persistentes con cifrado,
  migraciones, backup y auditoria de acceso.
- Convertir `domain.AuditEntry` en evidencia persistida, encadenada y exportable
  con sello o referencia de firma cuando proceda.
- Exigir correlacion de request, actor, expediente, solicitud y convocatoria en
  logs; excluir payload sensible.
- Proteger endpoints de expediente y listados con politicas distintas: consulta
  privada, publicacion oficial, vista anonima/parcial y exportacion firmada.

## Documento y expediente electronico

Requisitos verificables:

- Cada documento aportado o generado debe tener identificador estable, CSV o
  localizador verificable, huella SHA-256, metadatos ENI, formato admitido,
  origen, fecha, actor y estado antivirus.
- Documentos no limpios por antivirus quedan en cuarentena y no son exportables.
- Cada firma debe registrar firmante, mecanismo, referencia de firma y fecha.
- El expediente electronico debe agrupar solicitud, meritos, requerimientos,
  subsanaciones, informes, listados, acuerdos, notificaciones y recursos.
- El indice del expediente debe ser reproducible, firmado o sellado cuando
  aplique y enlazar documentos por identificador/huella.
- Las copias autenticas y digitalizaciones deben conservar metadatos de origen,
  organo, responsable, fecha, formato y relacion con el documento origen.
- Toda version de baremo aplicada debe quedar congelada por convocatoria; el
  recalculo posterior no puede alterar resultado historico sin nueva evidencia.
- Listados provisional/definitivo deben guardar version, fecha de publicacion,
  firmante/sello, criterio de desempate y trazabilidad de cambios.
- Notificaciones deben registrar puesta a disposicion, acceso/rechazo,
  vencimiento de plazo y contenido comunicado.
- Exportaciones deben ser paquetes reproducibles: expediente, indice, metadatos,
  firmas/sellos, huellas y auditoria minima.

Mapeo con dominio existente:

- `DocumentEvidence` cubre CSV, SHA-256, refs externas, metadatos ENI,
  antivirus, presentador, fecha y firmas.
- `ElectronicFile` agrupa candidato, procedimiento, creador, fecha y documentos.
- `AuditEntry` cubre secuencia, actor, accion, fecha, hash de payload, firma
  previa y firma calculada.
- Pendiente productivo: puertos/adaptadores para almacenamiento, archivo, firma,
  sellado de tiempo, registro, notificacion y verificacion CSV.

## ENI/NTI e interoperabilidad

Checklist:

- Usar metadatos ENI en documentos y expedientes desde la entrada, no como
  enriquecimiento tardio.
- Normalizar catalogos: organo, procedimiento, tipo documental, estado,
  idioma, formato y version de esquema.
- Admitir formatos abiertos o estandarizados para conservacion; rechazar o
  convertir formatos no permitidos segun politica.
- Mantener interoperabilidad semantica de meritos y estados: codigos estables,
  descripciones i18n y versionado por convocatoria.
- Documentar politica de firma y certificados: quien firma, que se firma,
  formato, validacion, sello de tiempo y conservacion.
- Preparar intercambio: paquete de expediente con indice, documentos,
  metadatos, firmas y huellas.
- Preservar dimension temporal: lectura futura del expediente aunque cambien
  reglas de baremo, formatos o proveedores.

## Accesibilidad, i18n y transparencia

Requisitos:

- UI compatible con teclado, foco visible, contraste suficiente, etiquetas de
  formulario, errores asociados al campo y mensajes no dependientes solo del
  color.
- PDFs/documentos generados deben ser accesibles cuando se publiquen o entreguen
  a personas interesadas.
- Textos legales y de estado deben estar en catalogo i18n; no hardcodear
  mensajes nuevos en adaptadores salvo fallback local acotado.
- Fechas, plazos, estados y importes/puntos deben presentarse con locale
  explicito.
- Publicar criterios de baremo, reglas de desempate, calendario, medios de
  subsanacion y recursos.
- Listados publicos deben equilibrar transparencia y minimizacion: identificar
  suficiente para defensa de derechos sin exponer datos completos innecesarios.

## Auditoria y evidencias

Eventos minimos a registrar:

- Creacion/modificacion de candidato, solicitud y merito.
- Carga, sustitucion, validacion, rechazo o cuarentena de documento.
- Calculo de baremo y version de reglas usada.
- Publicacion de listados y correcciones.
- Acceso administrativo a expediente.
- Notificacion, acceso, rechazo o vencimiento.
- Ejercicio de derechos RGPD y respuesta.
- Cambios de permisos, configuracion y claves.

Cada evento debe contener:

- `actor_id`, rol, mecanismo de autenticacion y unidad si aplica.
- `action`, `occurred_at`, convocatoria, solicitud, expediente y correlacion
  tecnica.
- Hash del payload o referencia a evidencia; no payload sensible completo en
  logs operativos.
- Firma/sello o encadenado de integridad para eventos con efecto juridico.
- Resultado: aceptado, rechazado, error funcional, error tecnico o revertido.

## Checklist de aceptacion productiva

| Area | Criterio verificable |
| --- | --- |
| Datos | Registro de actividades, informacion por capas, DPO, retencion y circuito de derechos documentados. |
| Identidad | Clave/DNIe/certificado para ciudadania, directorio/MFA para personal, representantes soportados y trazados. |
| Autorizacion | Matriz RBAC/ABAC por endpoint, titularidad de expediente y segregacion de funciones. |
| ENS | Categorizacion, analisis de riesgos, politica, medidas implantadas, auditoria y evidencias de conformidad. |
| Expediente | Documento ENI, expediente ENI, indice, firmas/sellos, CSV, huellas y exportacion reproducible. |
| Interoperabilidad | Catalogos, formatos, metadatos, versionado y paquete de intercambio validados. |
| Auditoria | Cadena integra, consulta controlada, retencion, exportacion y revision periodica. |
| Accesibilidad | Auditoria WCAG/EN 301 549, declaracion de accesibilidad y correccion de incidencias. |
| Seguridad | TLS, secretos, backups, restauracion, logs, vulnerabilidades, hardening y respuesta a incidentes. |
| Pruebas | Tests de dominio/puertos, fake de adaptadores, pruebas de autorizacion, auditoria, expediente y exportacion. |

## Backlog derivado limitado

Estas tareas quedan fuera de este contrato documental y requieren write-set
propio:

- Puerto de almacenamiento documental y adaptador persistente.
- Puerto de firma/sello/TSA y verificacion CSV.
- Puerto de notificaciones electronicas.
- Adaptador de identidad real para ciudadania, representantes y personal.
- Persistencia productiva de expedientes, auditoria y convocatorias.
- Tests de frontera para exportacion ENI y autorizacion por rol/titularidad.
- Declaracion de accesibilidad y plantilla de informacion RGPD por capas.

