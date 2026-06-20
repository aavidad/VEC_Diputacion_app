# Dominio portal VEC y autobaremacion

## Alcance

Este documento define el modelo implantable para un portal de bolsa de empleo
con Ventanilla Electronica del Candidato (VEC) y autobaremacion. Mantiene la
frontera hexagonal existente: nucleo de dominio neutral, puertos pequenos,
adaptadores opt-in y composicion canonica fuera del nucleo.

Refs opacas preservadas para Orquesta: `worktree-bolsa-vec-portal-005`,
`branch-bolsa-vec-portal-005` y `portal-vec-dominio-autobaremo-005`.

## Principios de dominio

- El nucleo no conoce HTTP, MCP, OPES, Codex, base de datos, proveedor de
  identidad, rutas locales ni credenciales.
- Las reglas de baremo pertenecen a la convocatoria y son versionadas; un
  resultado siempre declara `convocatoria_id`, version de reglas, detalle por
  merito y motivo de desempate.
- La identidad distingue actor autenticado de persona candidata. Una misma
  persona puede actuar como candidata en una convocatoria y como empleado en
  otra operacion administrativa solo si su rol lo permite.
- Todo cambio relevante emite evento de dominio y entrada de auditoria con
  actor, instante, accion, hash de payload y firma encadenada.
- La i18n queda en adaptadores o capa compartida; el dominio emite codigos y
  estados canonicos, no textos de interfaz.

## Entidades y agregados

| Agregado | Entidad/valor | Responsabilidad |
| --- | --- | --- |
| Identidad | `Usuario` | Sujeto autenticado por Clave, DNIe o Kerberos AD. Contiene `subject`, mecanismo, rol y atributos verificables. |
| Identidad | `Empleado` | Usuario con rol `personal_interno`; puede revisar, admitir, excluir, requerir subsanacion y publicar listados segun permisos. |
| Identidad | `Candidato` | Persona solicitante con ID, DNI/NIE, nombre, email y datos de contacto. No decide permisos por si misma. |
| Procedimiento | `Convocatoria` | Oferta/version publicada con estado, calendario, bases, reglas de baremo, cupos y letra de sorteo. |
| Procedimiento | `Solicitud` | Inscripcion del candidato en una convocatoria. Une candidato, meritos, documentos, autobaremo, estado y trazabilidad. |
| Meritos | `MeritoRUM` | Merito importado o declarado desde Registro Unificado de Meritos (RUM), con tipo, periodo/unidades, origen y evidencias. |
| Meritos | `DocumentoEvidence` | Evidencia ENI con CSV, SHA-256, refs externas opacas, firmas, sello de tiempo y estado antivirus. |
| Baremo | `BaremoRuleSet` | Reglas versionadas por convocatoria: tipo de merito, seccion, unidad, puntos por unidad, topes y desempates. |
| Baremo | `BaremoResult` | Resultado calculado: total, puntos por seccion, detalle por merito, caps aplicados y version de reglas. |
| Bolsa | `Listado` | Vista publicada provisional o definitiva con solicitudes, estados, puntuacion y posicion si procede. |
| Bolsa | `Bolsa` | Secuencia ordenada definitiva con estado operativo: provisional, alegaciones, definitiva, agotada o cerrada. |

## Estados canonicos

### Convocatoria

| Estado | Entrada | Salidas validas |
| --- | --- | --- |
| `Borrador` | Creacion interna con version y reglas validas. | `Inscripcion` |
| `Inscripcion` | Publicacion de bases y apertura VEC. | `Subsanacion`, `Alegaciones` |
| `Subsanacion` | Revision detecta falta documental o dato incoherente. | `Alegaciones`, `Definitiva` |
| `Alegaciones` | Listado provisional publicado. | `Definitiva` |
| `Definitiva` | Listado definitivo firmado. | `Cerrada` |
| `Cerrada` | Procedimiento terminado o archivado. | Sin salida ordinaria |

### Solicitud

| Estado | Descripcion | Salidas validas |
| --- | --- | --- |
| `Borrador` | Candidato prepara inscripcion. | `Inscrita` |
| `Inscrita` | Solicitud registrada y sellada. | `AdmitidaProvisional`, `SubsanacionRequerida`, `ExcluidaProvisional` |
| `SubsanacionRequerida` | Empleado pide correccion o documento. | `Subsanada`, `ExcluidaProvisional` |
| `Subsanada` | Candidato aporta respuesta dentro de plazo. | `AdmitidaProvisional`, `ExcluidaProvisional` |
| `AdmitidaProvisional` | Incluida en listado provisional. | `AlegacionPresentada`, `AdmitidaDefinitiva` |
| `ExcluidaProvisional` | Excluida en listado provisional. | `AlegacionPresentada`, `ExcluidaDefinitiva` |
| `AlegacionPresentada` | Reclamacion registrada contra provisional. | `AdmitidaDefinitiva`, `ExcluidaDefinitiva` |
| `AdmitidaDefinitiva` | Puede entrar en ranking de bolsa. | Sin salida ordinaria |
| `ExcluidaDefinitiva` | No participa en ranking. | Sin salida ordinaria |

### Merito

| Estado | Descripcion | Salidas validas |
| --- | --- | --- |
| `Borrador` | Merito local aun no presentado. | `Presentado` |
| `Presentado` | Declarado por candidato o importado de RUM pendiente de validacion. | `Validado`, `Rechazado`, `Subsanacion` |
| `Subsanacion` | Requiere aclaracion documental. | `Presentado` |
| `Validado` | Computable para baremo segun reglas vigentes. | Sin salida ordinaria |
| `Rechazado` | No computable; conserva motivo y evidencia. | Sin salida ordinaria |

### Bolsa

| Estado | Descripcion | Salidas validas |
| --- | --- | --- |
| `SinConstituir` | No hay listado publicado. | `Provisional` |
| `Provisional` | Listado provisional visible. | `EnAlegaciones`, `Definitiva` |
| `EnAlegaciones` | Plazo de alegaciones abierto. | `Definitiva` |
| `Definitiva` | Orden de bolsa vigente. | `Agotada`, `Cerrada` |
| `Agotada` | No quedan candidatos disponibles. | `Cerrada` |
| `Cerrada` | Bolsa cerrada. | Sin salida ordinaria |

## Eventos de dominio

| Evento | Emisor | Datos minimos |
| --- | --- | --- |
| `UsuarioAutenticado` | Puerto de identidad | `subject`, mecanismo, rol, instante. |
| `ConvocatoriaCreada` | Caso de uso administrativo | ID, version, calendario, hash de bases, reglas de baremo. |
| `ConvocatoriaPublicada` | Empleado autorizado | ID, version, periodo de inscripcion. |
| `SolicitudBorradorCreada` | Portal VEC | solicitud, candidato, convocatoria. |
| `MeritosRUMImportados` | Adaptador RUM | solicitud, candidato, lote RUM, meritos normalizados, evidencias. |
| `MeritoPresentado` | Portal VEC | merito, tipo, datos, documentos. |
| `SolicitudRegistrada` | Portal VEC | CSV/asiento, solicitud, candidato, hash de expediente. |
| `AutobaremoCalculado` | Nucleo baremo | solicitud, rule set, total, secciones, detalle. |
| `SubsanacionRequerida` | Empleado | solicitud, motivo, plazo, meritos/documentos afectados. |
| `MeritoValidado` | Empleado | merito, criterio aplicado, auditoria. |
| `MeritoRechazado` | Empleado | merito, causa, norma, auditoria. |
| `ListadoProvisionalPublicado` | Caso de uso procedimiento | convocatoria, items, hash, firma. |
| `AlegacionPresentada` | Portal VEC | solicitud, documentos, texto, CSV. |
| `ListadoDefinitivoPublicado` | Caso de uso procedimiento | convocatoria, ranking, desempates, hash, firma. |
| `BolsaConstituida` | Caso de uso bolsa | convocatoria, orden definitivo, estado inicial. |

## Flujo completo de autobaremacion desde RUM

1. Identidad autentica al usuario por Clave o DNIe. El puerto devuelve
   `AuthPrincipal` con rol `ciudadano`; el adaptador traduce errores a i18n.
2. Candidato elige convocatoria abierta. Caso de uso carga `Convocatoria` y
   `BaremoRuleSet` versionado.
3. Portal crea `Solicitud` en `Borrador` y consulta RUM mediante puerto
   `RUMMeritProvider`.
4. Adaptador RUM devuelve meritos externos normalizados sin filtrar por UI:
   tipo canonico, fechas/unidades, origen, CSV o referencia verificable.
5. Nucleo convierte cada entrada en `MeritoRUM`/`Merito` `Presentado`; si falta
   evidencia o antivirus no esta limpio, queda no exportable y marcado para
   revision/subsanacion.
6. Candidato confirma o completa meritos manuales. Cada documento se guarda
   como `DocumentEvidence` con refs opacas de almacenamiento, secreto y sello
   de tiempo.
7. Al registrar, la solicitud pasa a `Inscrita`, se emite CSV/asiento, se
   encadena auditoria y se calcula `AutobaremoCalculado` con reglas de la
   convocatoria.
8. `CalcularAutobaremo` ordena meritos de forma determinista, valida tipo,
   estado y datos, aplica puntos por unidad y topes por seccion, y conserva
   detalle crudo/aplicado por merito.
9. Empleado revisa evidencias. Puede validar, rechazar o pedir subsanacion; el
   recalculo se ejecuta con la misma version de reglas salvo nueva version
   formal de convocatoria.
10. Publicacion provisional clasifica solicitudes admitidas/excluidas. Solo
    admitidas definitivas entran en ranking.
11. Alegaciones y subsanaciones generan nuevos eventos y auditoria. El listado
    definitivo recalcula ranking con total de puntos y reglas de desempate:
    mayor experiencia, mayor formacion, letra de sorteo y fallback estable.
12. Bolsa queda constituida desde `ListadoDefinitivoPublicado`; exportacion de
    expediente solo incluye documentos limpios, firmados y manifestados.

## Puertos propuestos

| Puerto | Direccion | Metodos minimos |
| --- | --- | --- |
| `IdentityProvider` | Entrada/adaptador | `Authenticate(credentials) AuthPrincipal` |
| `ConvocatoriaRepository` | Salida | `Save`, `GetByID`, `ListOpen` |
| `SolicitudRepository` | Salida | `Save`, `GetByID`, `ListByConvocatoria`, `ListByCandidate` |
| `MeritRepository` | Salida | `Save`, `ListBySolicitud`, `FindByRUMRef` |
| `RUMMeritProvider` | Salida opt-in | `ListMerits(subject, convocatoriaID) []ExternalMerit` |
| `EvidenceStore` | Salida opt-in | `StoreDocument`, `GetMetadata`, `MarkAVStatus` |
| `AuditTrail` | Salida | `Append(entry)`, `VerifyChain(scope)` |
| `NotificationPort` | Salida opt-in | `NotifySubsanacion`, `NotifyListadoPublicado` |

## Invariantes

- Una solicitud pertenece a una unica convocatoria y a un unico candidato.
- Una convocatoria publicada no muta reglas en sitio; cualquier cambio crea
  nueva version trazable.
- Un merito sin regla aplicable no puntua y debe quedar como incidencia de
  configuracion, no como cero silencioso.
- Solo meritos `Validado` deben computar en baremo administrativo definitivo;
  el autobaremo ciudadano puede marcar `Presentado` como provisional.
- Solo solicitudes `AdmitidaDefinitiva` entran en ranking de bolsa.
- Todo documento exportable requiere CSV, digest SHA-256, metadatos ENI,
  almacenamiento opaco y antivirus `CLEAN`.
- Todo listado publicado requiere hash, firma, actor y auditoria encadenada.

## Gaps contra app actual

| Area | Existe hoy | Gap implantable |
| --- | --- | --- |
| Identidad | Autenticador fake con Clave/Kerberos AD y roles basicos. | Integrar Clave/DNIe/Kerberos reales, sesiones, permisos por operacion y delegacion. |
| Convocatoria | `Convocatoria` con ID, version, estado y rule set en memoria. | Calendario, bases firmadas, plazas/cupos, organo gestor y publicacion versionada. |
| Solicitud | Registro demo con estado y autobaremo. | Borrador VEC, asiento registral, subsanacion, alegaciones y expediente por solicitud. |
| RUM | No hay puerto RUM. | Crear puerto `RUMMeritProvider`, normalizador y deduplicacion por ref externa. |
| Meritos | Tipos basicos, estados y datos de meses/horas/puntos. | Periodos, incompatibilidades, solapes, origen RUM/manual y motivos de validacion/rechazo. |
| Baremo | Reglas, topes, detalle, desempates y ranking basico. | Versionado legal por convocatoria, solo computo definitivo con meritos validados y explicabilidad i18n. |
| Evidencias | Modelo ENI/documento/auditoria en dominio. | Conectar almacenamiento, antivirus, firma, sello de tiempo y registro sin meter adaptadores en nucleo. |
| Listados | Provisional/definitivo demo en memoria. | Publicacion firmada, alegaciones, historico, notificaciones y exportacion oficial. |
| Persistencia | Repositorios en memoria. | Repositorios duraderos transaccionales y migraciones fuera del nucleo. |
| i18n | Catalogo compartido con fallback espanol. | Codigos de error/evento completos para VEC y empleado; textos en adaptador. |

## Vertical slices propuestos

| Slice | Objetivo | Frontera hexagonal | Prueba focal |
| --- | --- | --- | --- |
| VS1 Convocatoria versionada | Crear/publicar convocatoria con reglas y calendario. | Nucleo `Convocatoria`; puerto repo; adaptador memoria/SQL opt-in. | Crear version, rechazar reglas invalidas, no mutar version publicada. |
| VS2 Borrador VEC | Crear solicitud borrador y confirmar inscripcion con CSV. | Caso de uso solicitud; puerto registro/evidencia fake. | Borrador -> inscrita, auditoria y CSV requeridos. |
| VS3 Importacion RUM | Importar meritos externos y deduplicar por ref RUM. | Puerto `RUMMeritProvider`; normalizador en caso de uso. | Mismo ref RUM no duplica; tipo desconocido queda incidencia. |
| VS4 Autobaremo explicable | Calcular autobaremo provisional con detalle i18n-ready. | Nucleo baremo puro; adaptador HTTP solo mapea vistas. | Topes por seccion, regla ausente y detalle por merito. |
| VS5 Revision empleado | Validar/rechazar meritos y pedir subsanacion. | Caso de uso administrativo; puerto auditoria. | Transiciones validas, motivo obligatorio y recalculo. |
| VS6 Alegaciones | Registrar alegacion contra provisional y resolverla. | Solicitud/listado en nucleo; evidencia opt-in. | Provisional -> alegacion -> definitiva, documentos limpios. |
| VS7 Bolsa definitiva | Publicar listado definitivo y constituir bolsa ordenada. | Ranking en nucleo; repositorio de bolsa. | Solo admitidos definitivos, desempates trazados, hash/firma. |
| VS8 Exportacion expediente | Exportar manifiesto oficial del expediente. | `ElectronicFile` puro; adaptador almacenamiento/firma. | Documento en cuarentena bloquea exportacion. |

## Cierre implantable

El siguiente paso de codigo no debe crear una app nueva. Debe ampliar el corte
existente por slices, empezando por puertos y casos de uso pequenos. La primera
entrega recomendable es VS3 o VS4, porque conecta el objetivo VEC/RUM con el
nucleo de baremo actual sin introducir persistencia productiva prematura.
