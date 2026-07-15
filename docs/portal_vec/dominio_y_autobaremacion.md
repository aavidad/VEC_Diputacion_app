# Dominio del portal VEC y autobaremacion

## Estado del documento

| Campo | Valor |
| --- | --- |
| Naturaleza | Especificacion objetivo y memoria de decisiones de arquitectura. |
| Estado | Borrador sujeto a revision funcional, juridica, de seguridad y de sistemas. |
| Acredita implementacion | No. Los nombres de tipos, puertos, versiones y cortes expresan contratos requeridos o propuestos. |
| Autoriza exposicion | No. Ninguna interfaz HTTP, CLI, MCP, tarea interna ni integracion queda habilitada por este documento. |
| Flujo V1 de firma y baremacion | Rechazado como base productiva. Solo puede conservarse como antecedente de requisitos y no debe cablearse, desplegarse ni presentarse como funcionalidad terminada. |

Las referencias a idempotencia semantica V2, recibos materiales V2 y ejecucion
documental V4 designan el nivel minimo objetivo de los contratos. No prueban que
existan adaptadores productivos, migraciones, configuracion operativa ni una
composicion segura. Cada capacidad debera acreditarse mediante codigo revisado,
pruebas, adaptadores reales, configuracion cerrada y evidencias de despliegue.

## Alcance

Este documento define el modelo objetivo para un portal de bolsa de empleo con
Ventanilla Electronica del Candidato (VEC) y autobaremacion. La solucion debera
mantener una frontera hexagonal: nucleo de dominio neutral, puertos pequenos,
adaptadores habilitados expresamente y composicion fuera del nucleo.

Quedan dentro del alcance funcional la convocatoria, la solicitud, los meritos,
la documentacion, el autobaremo ciudadano, la revision tecnica, las
rectificaciones, alegaciones, listados y constitucion de bolsa. La identidad,
autorizacion, firma, registro, custodia, antivirus, notificaciones y formatos de
salida se integraran mediante puertos intercambiables sin debilitar las
invariantes de seguridad.

## Principios de dominio

- El nucleo no conoce HTTP, MCP, OPES, Codex, base de datos, proveedor de
  identidad, rutas locales ni credenciales.
- La politica es de denegacion por defecto y minimo privilegio. La ausencia de
  una concesion positiva y exacta equivale siempre a denegacion.
- Las reglas de baremo pertenecen a la convocatoria y son versionadas; un
  resultado siempre declara `convocatoria_id`, version de reglas, detalle por
  merito y motivo de desempate.
- La identidad distingue actor autenticado de persona candidata. Una misma
  persona puede actuar como candidata en una convocatoria y como empleado en
  otra operacion administrativa solo si su rol lo permite.
- Los datos personales maestros pertenecen al agregado compartido de persona.
  Candidato y empleado son proyecciones de contexto con referencias opacas; no
  duplican DNI, contacto o titulaciones salvo la instantanea minima que una
  obligacion legal exija conservar en un expediente.
- Todo cambio relevante debera confirmar atomicamente el estado, el evento de
  dominio y la entrada de auditoria con actor, instante, accion, resultado,
  referencias de autorizacion y recibos materiales. Las firmas o cadenas de
  integridad se aplicaran conforme a la politica publicada.
- La internacionalizacion (i18n) queda en adaptadores o capa compartida; el
  dominio emite codigos y estados canonicos, no textos de interfaz.
- Las listas de negocio, criterios, causas, calendarios, reglas, plantillas y
  transiciones ampliables se administraran mediante catalogos versionados. Las
  invariantes de seguridad, los limites absolutos y la denegacion por defecto no
  son configurables ni se rebajan desde la aplicacion.

## Entidades y agregados

| Agregado | Entidad/valor | Responsabilidad |
| --- | --- | --- |
| Identidad | `Usuario` | Sujeto autenticado por Clave, DNIe o Kerberos AD. Contiene identificador de sujeto, mecanismo, perfil activo y atributos verificables. |
| Identidad | `Empleado` | Usuario con rol `personal_interno`; puede revisar, admitir, excluir, requerir subsanacion y publicar listados segun permisos. |
| Identidad | `Candidato` | Proyeccion de la persona solicitante con referencia opaca al dato maestro y atributos estrictamente necesarios para la convocatoria. No decide permisos por si misma. |
| Procedimiento | `Convocatoria` | Oferta/version publicada con estado, calendario, bases, reglas de baremo, cupos y letra de sorteo. |
| Procedimiento | `Solicitud` | Inscripcion del candidato en una convocatoria. Une candidato, meritos, documentos, autobaremo, estado y trazabilidad. |
| Meritos | `MeritoRUM` | Merito importado o declarado desde Registro Unificado de Meritos (RUM), con tipo, periodo/unidades, origen y evidencias. |
| Meritos | `EvidenciaDocumento` | Evidencia ENI con CSV, SHA-256, referencias externas opacas, firmas, sello de tiempo y estado antivirus. |
| Baremo | `ConjuntoReglasBaremo` | Reglas versionadas por convocatoria: tipo de merito, seccion, unidad, puntos por unidad, topes y desempates. |
| Baremo | `ResultadoBaremo` | Resultado calculado: total, puntos por seccion, detalle por merito, topes aplicados y version de reglas. |
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
| `AdmitidaDefinitiva` | Puede entrar en la clasificacion de bolsa. | Sin salida ordinaria |
| `ExcluidaDefinitiva` | No participa en la clasificacion. | Sin salida ordinaria |

### Merito

| Estado | Descripcion | Salidas validas |
| --- | --- | --- |
| `Borrador` | Merito local aun no presentado. | `Presentado` |
| `Presentado` | Declarado por candidato o importado de RUM pendiente de validacion. | `Validado`, `Rechazado`, `Subsanacion` |
| `Subsanacion` | Requiere aclaracion documental. | `Presentado` |
| `Validado` | Computable segun la decision firmada vigente. | Puede ser sustituida por una rectificacion firmada, sin borrar la decision anterior. |
| `Rechazado` | No computable; conserva motivo y evidencia. | Puede ser sustituida por una rectificacion firmada, sin borrar la decision anterior. |

`Validado` y `Rechazado` son proyecciones de la ultima decision eficaz, no
campos editables. Una revision sobrevenida no cambia ni elimina aquella fila:
crea una nueva decision que referencia y sustituye exactamente a la anterior.

### Bolsa

| Estado | Descripcion | Salidas validas |
| --- | --- | --- |
| `SinConstituir` | No hay listado publicado. | `Provisional` |
| `Provisional` | Listado provisional visible. | `EnAlegaciones`, `Definitiva` |
| `EnAlegaciones` | Plazo de alegaciones abierto. | `Definitiva` |
| `Definitiva` | Orden de bolsa vigente. | `Agotada`, `Cerrada` |
| `Agotada` | No quedan candidatos disponibles. | `Cerrada` |
| `Cerrada` | Bolsa cerrada. | Sin salida ordinaria |

Estas tablas fijan categorias semanticas minimas para la especificacion, no
enumeraciones dispersas que deban quedar incrustadas en el codigo. Las fases y
transiciones ampliables proceden de una definicion versionada y publicada; no
pueden omitir las fronteras legales de registro, firma, alegacion, resolucion ni
auditoria.

## Eventos de dominio

| Evento | Emisor | Datos minimos |
| --- | --- | --- |
| `UsuarioAutenticado` | Puerto de identidad | `subject`, mecanismo, rol, instante. |
| `ConvocatoriaCreada` | Caso de uso administrativo | ID, version, calendario, huella de bases, reglas de baremo. |
| `ConvocatoriaPublicada` | Empleado autorizado | ID, version, periodo de inscripcion. |
| `SolicitudBorradorCreada` | Portal VEC | solicitud, candidato, convocatoria. |
| `MeritosRUMImportados` | Adaptador RUM | solicitud, candidato, lote RUM, meritos normalizados, evidencias. |
| `MeritoPresentado` | Portal VEC | merito, tipo, datos, documentos. |
| `SolicitudRegistrada` | Portal VEC | CSV/asiento, solicitud, candidato, huella de expediente. |
| `AutobaremoCalculado` | Nucleo baremo | solicitud, conjunto de reglas, total, secciones, detalle. |
| `SubsanacionRequerida` | Empleado | solicitud, motivo, plazo, meritos/documentos afectados. |
| `MeritoValidado` | Empleado | merito, criterio aplicado, auditoria. |
| `MeritoRechazado` | Empleado | merito, causa, norma, auditoria. |
| `DecisionBaremacionFirmada` | Tecnico autorizado | merito, documentos evaluados, puntos declarados/calculados/reconocidos, resultado, motivo, firma y huella. |
| `DecisionBaremacionRectificada` | Inspector/supervisor autorizado | decision sustituida, valor anterior y nuevo, motivo, pruebas, firma y efectos de recalculo. |
| `ActaBaremacionCerrada` | Organo de seleccion | conjunto de decisiones, totales, incidencias, votos, firmas, sello de tiempo y CSV. |
| `ListadoProvisionalPublicado` | Caso de uso procedimiento | convocatoria, elementos, huella, firma. |
| `AlegacionPresentada` | Portal VEC | solicitud, documentos, texto, CSV. |
| `ListadoDefinitivoPublicado` | Caso de uso procedimiento | convocatoria, clasificacion, desempates, huella, firma. |
| `BolsaConstituida` | Caso de uso bolsa | convocatoria, orden definitivo, estado inicial. |

## Flujo objetivo de autobaremacion desde RUM

Este flujo describe comportamiento requerido, no comportamiento acreditado de
la aplicacion actual:

1. El proveedor de identidad autentica al actor mediante uno de los mecanismos
   admitidos. El servidor obtiene una sesion revocable y un principal con
   identidad, perfil activo, garantia de autenticacion y atributos verificables;
   nunca acepta esos datos desde un objeto de transferencia proporcionado por
   el cliente.
2. Antes de consultar datos no publicos, el servidor solicita al punto de
   decision de politica (PDP) una
   preautorizacion de lectura ligada al actor, sesion, finalidad, accion y
   ambito opaco. Si no puede obtenerla o verificarla, no consulta convocatoria
   restringida, solicitud, RUM, meritos, documentos ni indices que revelen su
   existencia.
3. Con esa capacidad de lectura vigente, el candidato selecciona una
   convocatoria abierta y el caso de uso recupera su version publicada, el
   calendario y el conjunto de reglas exacto.
4. Crear o recuperar el borrador exige una autorizacion exacta sobre la
   solicitud propuesta y una identidad de operacion idempotente. La intencion se
   reserva de forma durable antes de confirmar estado, auditoria y bandeja
   transaccional en una unica transaccion.
5. La consulta a RUM requiere una decision propia y de alcance minimo. El
   adaptador devuelve meritos externos normalizados: tipo canonico,
   fechas/unidades, origen y referencia verificable. El nucleo no confia en
   filtros de interfaz ni en autorizaciones heredadas de pasos anteriores.
6. El nucleo convierte cada entrada admitida en un merito `Presentado`. Un tipo
   desconocido, una evidencia ausente o un analisis antivirus no limpio produce
   una incidencia explicita y bloquea su uso o exportacion; nunca se interpreta
   como cero silencioso.
7. Para documentos manuales se autoriza y reserva primero una sesion de carga
   directa a cuarentena. El almacenamiento devuelve una referencia opaca y un
   recibo material verificable; analisis, promocion, retencion y lectura son
   acciones independientes y requieren decisiones diferentes.
8. Registrar la solicitud es una accion legal separada del borrador. Antes de
   registro, firma o custodia se persiste el plan durable; despues se ejecutan
   los efectos idempotentes y se confirman solicitud, asiento/CSV, auditoria,
   bandeja transaccional y recibos sin ocultar resultados indeterminados.
9. El calculador de autobaremo ordena entradas de forma determinista, valida
   tipo, estado y datos, usa aritmetica decimal fija, aplica incompatibilidades y
   topes, y conserva el detalle declarado y aplicado por merito con la version
   congelada de reglas.
10. Antes de mostrar a un empleado un merito o documento, el PDP autoriza la
    lectura y los campos proyectables. Adoptar, firmar, rectificar, revocar o
    rehabilitar una decision son acciones distintas. Cada efecto usa una
    autorizacion exacta y una intencion durable previa.
11. La decision eficaz conserva puntos declarados, calculados y reconocidos,
    regla, motivacion, versiones de evidencias, firma validada, sello temporal
    cuando proceda y recibos materiales. El recalculo usa la misma version de
    reglas, salvo rectificacion formal de la convocatoria.
12. Publicacion provisional, alegaciones, subsanaciones, resolucion y listado
    definitivo generan estados y eventos trazables. Solo solicitudes admitidas
    definitivamente entran en la clasificacion; los desempates proceden de reglas
    publicadas, no de una lista fija en codigo.
13. La bolsa se constituye desde un listado definitivo firmado. La exportacion
    del expediente incluye solo representaciones admitidas, limpias, verificadas
    y declaradas en un manifiesto; cada formato de salida se resuelve mediante
    un perfil publicado y un conector autorizado.

## Barreras obligatorias de seguridad y durabilidad

### Autorizacion antes de lectura y de efecto

La autorizacion se evaluara en el servidor. Como minimo existirán tres barreras
independientes:

1. **Preautorizacion de descubrimiento o lectura.** Debe concederse antes de
   consultar cualquier dato, indice o metadato no publico, antes de recuperar
   bytes y antes de renderizar datos personales. Se limita a un ambito opaco y a
   los campos estrictamente necesarios.
2. **Autorizacion exacta de operacion.** Una vez conocido el recurso de forma
   autorizada, el PDP evalua actor, perfil, sesion, representacion, finalidad,
   accion, clasificacion, convocatoria, versiones, huellas, plan y obligaciones.
   La concesion resultante solo vale para esa preimagen canonica y ese efecto.
3. **Revalidacion autoritativa.** Se comprueban sesion, asignacion, rol,
   catalogo, recurso, version y concesion inmediatamente antes de iniciar el
   efecto remoto y se comprueban de nuevo dentro de la transaccion que confirma
   su recibo y el nuevo estado. Si la infraestructura no permite cerrar una de
   esas carreras, la operacion queda pendiente de reconciliacion y no se declara
   completada.

Una decision de lectura no autoriza firma, escritura, custodia, retencion,
publicacion ni descarga. Tampoco se reutiliza una decision de un actor, finalidad
o efecto para otro. El resultado idempotente de una ejecucion anterior solo se
devuelve despues de autorizar de nuevo su consulta y proyeccion.

Los datos declarados publicos se serviran desde una proyeccion separada, con
lista positiva de campos, politica publicada y pruebas de ausencia de datos
personales. Que una ruta sea accesible sin autenticacion no convierte en publico
su repositorio de origen.

### Idempotencia semantica V2

Toda accion con efecto juridico, persistente o remoto debera usar una identidad
semantica V2 calculada en el servidor. Una clave libre aportada por el cliente no
es por si sola una identidad de operacion.

La preimagen canonica incluira, segun la accion, el tipo de operacion, actor y
representado seudonimizados, sesion, finalidad, convocatoria y reglas, agregado
y revision esperada, referencias y versiones de documentos, huellas de bytes,
plan inmutable, efectos previstos y correlacion. No contendra DNI, nombre,
correo, secreto, token ni ruta local en claro.

La instantanea del principal usada para derivar o comparar esa identidad sera
completa, canonica y obtenida de fuentes autoritativas: sujeto seudonimizado,
metodo y garantia de autenticacion, perfil activo, roles y asignaciones,
representacion, ambito organizativo, sesion y epocas de clave aplicables. El
llamante no puede omitir atributos, seleccionar solo algunos roles ni aportar
una fotografia parcial que produzca otra identidad valida.

Durante una rotacion de claves, el servidor derivara y comprobara la matriz
completa de indices admitidos a partir de instantaneas autoritativas y completas
de los anillos de claves aplicables. La matriz sera el producto cartesiano
completo de todas las dimensiones de clave publicadas por el contrato; no se
aceptan subconjuntos elegidos por el llamante, combinaciones parciales ni una
unica clave que omita candidatos historicos vigentes. La falta, duplicidad o
ambiguedad de un candidato cierra la operacion.

El repositorio aplicara estas reglas:

- misma identidad semantica y misma preimagen: recuperar el resultado exacto o
  continuar el mismo plan durable, previa autorizacion de la consulta;
- misma clave cliente con preimagen diferente: conflicto, sin ejecutar efectos;
- varios resultados para los indices candidatos: incidente de integridad y
  denegacion cerrada;
- resultado incierto: estado `Indeterminado`, sin repetir automaticamente un
  efecto irreversible;
- ninguna firma, sello, escritura, promocion, retencion, publicacion o asiento
  antes de persistir la intencion, los pasos y sus claves idempotentes.

### Recibos materiales V2

Cada conector que lea o produzca material sensible devolvera un recibo nominal
V2 propio de su accion. No se admiten como prueba productiva una cadena libre,
una referencia generica ni una huella sin contexto. Se requieren tipos
incompatibles, como minimo, para recuperacion, renderizado, firma, validacion de
firma, sello de tiempo, escritura en custodia, retencion, analisis antivirus,
promocion y publicacion.

El recibo ligara de forma canonica:

- esquema y version del recibo, operacion, paso y referencia de efecto;
- conector y version/capacidades atestadas;
- objeto, version, zona, clasificacion y perfil publicado;
- huella criptografica y tamaño de los bytes de entrada y salida;
- identidad semantica de la operacion, manifiesto y decision de autorizacion;
- instante verificable, resultado y referencia de evidencia del conector;
- firma, MAC o atestacion comprobable mediante una raiz de confianza aprobada.

Los recibos y capacidades sensibles seran opacos fuera de su puerto. Sus
representaciones para registros, JSON, texto y errores estaran redactadas y no
revelaran bytes, secretos, tokens, identificadores personales ni material
criptografico. El servidor verificara el recibo contra la solicitud exacta y no
aceptara la autoafirmacion no atestada del mismo conector como unica prueba.

## Decision firmada, rectificacion y revision inspectora

La unidad de trabajo es el merito individual. Cada decision es una instantanea
inmutable que incluye el resultado de cada documento o evidencia asociado. No
se permite un simple `UPDATE aceptado = false`.

| Dato probatorio | Contenido minimo |
| --- | --- |
| Identidad | Convocatoria, solicitud, merito, version de reglas y decision. |
| Comparacion | Puntos declarados por el aspirante, calculados por el motor y reconocidos por el organo. |
| Evidencias | Referencias opacas y versiones exactas de cada documento; resultado individual y motivo. |
| Autoridad | Actor, perfil activo, rol, ambito, decision de autorizacion, finalidad y correlacion. |
| Motivacion | Codigo configurable, texto razonado, norma/base aplicada y pruebas consideradas. |
| Integridad | Huella del estado anterior y posterior, firma, validacion de firma, sello de tiempo, auditoria y bandeja transaccional atomicas. |

Una rectificacion puede revocar un documento antes aceptado o admitir uno antes
desestimado. Debe:

1. referenciar la decision anterior exacta y su huella;
2. conservar una fotografia completa del antes y del despues;
3. prohibir cambios sin efecto y escrituras concurrentes sobre una revision ya
   sustituida;
4. exigir motivo, evidencia, autorizacion reforzada y firma del inspector o
   supervisor competente;
5. generar una nueva version del baremo y, si afecta a un listado publicado,
   iniciar el flujo gobernado de rectificacion, notificacion y eventual
   republicacion;
6. mantener accesibles todas las versiones para auditoria, alegacion y recurso.

La decision inicial, la rectificacion, la revocacion de una aceptacion y la
rehabilitacion de un rechazo son acciones de autorizacion diferentes. Un
permiso general de baremacion no cubre las tres actuaciones inspectoras. Estas
exigen concesion positiva propia, garantia reforzada, alcance sobre el merito,
motivacion y firma; cuando la politica aplicable ordene doble control, quien
adopta y quien confirma seran personas distintas y ambas quedaran en la
evidencia.

La firma individual de cada decision es un requisito reforzado de este
producto. No sustituye el cierre colegiado: el lote debe producir tambien un
acta PDF y una representacion estructurada, aprobadas y firmadas conforme a la
configuracion del organo de seleccion. La doble revision por merito no es una
obligacion general identificada; sera una politica configurable y podra
exigirse para rechazos, rectificaciones, excepciones manuales, gran impacto en
puntos, empates/cortes, conflictos de interes o muestras aleatorias.

La subsanacion corrige defectos de acreditacion. La admision de meritos nuevos
fuera de plazo debe estar cerrada por defecto y solo habilitarse si las bases
versionadas de la convocatoria lo permiten expresamente.

## Catalogos y configuracion gobernada

Los cambios ordinarios de negocio no exigiran recompilar el nucleo. Se
gestionaran desde el portal interno mediante catalogos y definiciones
versionados, al menos:

- tipos y categorias de merito, titulaciones y equivalencias;
- criterios, unidades, topes, incompatibilidades, solapes y desempates;
- tipos documentales, formatos de salida y perfiles de renderizado;
- causas de admision, rechazo, subsanacion, rectificacion y revocacion;
- calendarios, ventanas de aportacion, fases y plazos;
- plantillas, canales de notificacion y textos parametrizables;
- estados y transiciones ampliables que respeten las invariantes del dominio;
- perfiles funcionales, asignaciones y politicas de doble control, sin conceder
  permisos por el mero nombre del rol.

Cada publicacion de catalogo tendra identificador, version, revision, vigencia,
fuente normativa, huella, autor, aprobacion y estado (`Borrador`, `Publicado` o
`Retirado`). Una version publicada no se modifica: se crea otra y se conserva la
anterior para reproducir expedientes historicos. Las operaciones en curso fijan
la version aplicable y no siguen automaticamente la ultima disponible.

La configuracion se validara antes de publicarse. No se admitiran expresiones o
codigo ejecutable arbitrario, formatos no homologados, transiciones que salten firma o
auditoria, ni catalogos que reduzcan garantias tecnicas. Las acciones de
seguridad, separacion de redes, limites absolutos, algoritmos admitidos y
obligaciones de autorizacion se mantienen en politica gobernada por Sistemas y
Seguridad, no como listas editables por un administrador funcional.

### Contraste con procedimientos publicos

La decision se apoya en patrones publicados por administraciones, sin copiar
sus pantallas ni convertir sus manuales en norma del producto:

- La [VEC del SAS](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-visualizo-mi-baremo-provisional)
  separa puntuacion aportada y validada, muestra estado y causa por merito, y
  permite comprobar los cambios realizados por la comision.
- El [autobaremo de VEC](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales/como-cumplimento-una-solicitud-de-autobaremo)
  vincula cada merito con su identificador, calculo y apartado concreto.
- [BAPE de SACYL](https://www.saludcastillayleon.es/profesionales/es/procesos_selectivos/nuevo-procedimiento-bolsas-empleo)
  distingue el autobaremo sin validar de la puntuacion revisada por el organo
  gestor y permite consultar meritos validados/rechazados y sus motivos.
- La [guia del INAP para organos de seleccion](https://www.inap.es/sites/portal/files/public/2026-02/Doc.%202%20GUIA%20INTERNA%20ABREVIADA%20PARA%20ORGANOS%20DE%20SELECCION.pdf)
  respalda el cierre colegiado mediante calculo desglosado, acta aprobada y
  firma de secretaria con visto bueno de presidencia.
- Las [bases publicadas en el BOP de Granada](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1753052521004-final-76280d53-2.pdf)
  aportan un referente local: cotejo entre autobaremo y documentos, actas,
  desglose provisional, alegaciones y resultado definitivo.

Estas fuentes sostienen el modelo de revision individual y acta verificable.
La firma individual de cada decision, el doble control selectivo y la cadena de
solo adicion son requisitos reforzados de esta solucion, no obligaciones
generales atribuidas a todas las administraciones.

## Puertos propuestos

| Puerto | Direccion | Metodos minimos |
| --- | --- | --- |
| `ProveedorIdentidad` | Entrada/adaptador | Autenticar y devolver un principal verificable sin exponer credenciales. |
| `RegistroSesiones` | Salida | Consultar vigencia, garantia, revocacion y vinculo exacto entre sesion y principal. |
| `PuntoDecisionPolitica` | Salida | Exigir y revalidar una decision para actor, accion, recurso, finalidad, campos y obligaciones exactos. |
| `RepositorioConvocatorias` | Salida | Guardar, obtener por ID/version y listar proyecciones publicas abiertas. |
| `RepositorioSolicitudes` | Salida | Reservar operacion y confirmar agregado, auditoria y bandeja transaccional en una transaccion. |
| `RepositorioMeritos` | Salida | Guardar versiones y consultar por solicitud o referencia externa autorizada. |
| `RepositorioDecisionesBaremacion` | Salida | Reservar, confirmar firmada, obtener vigente y listar historia inmutable. |
| `DerivadorIdempotenciaSemanticaV2` | Salida | Derivar y verificar la matriz completa de indices desde instantaneas autoritativas de claves. |
| `RepositorioPlanesDurables` | Salida | Reservar plan, adquirir arrendamiento y testigo monotono de exclusion, confirmar pasos y marcar resultado indeterminado. |
| `ProveedorMeritosRUM` | Salida habilitable | Listar meritos del sujeto y convocatoria dentro de una autorizacion exacta. |
| `AlmacenEvidencias` | Salida habilitable | Preparar carga, custodiar, leer, retener y promover mediante acciones separadas y recibos V2. |
| `VerificadorRecibosMaterialesV2` | Salida | Verificar esquema, atestacion y vinculo exacto entre solicitud, bytes, objeto y resultado. |
| `FirmadorDecisiones` | Salida | Preparar artefacto canonico, firmar/validar y devolver referencias, nunca claves privadas. |
| `GeneradorActas` | Salida | Generar PDF y formato estructurado desde una plantilla publicada exacta. |
| `RegistroAuditoria` | Salida | Añadir entrada y verificar la cadena por ambito sin registrar datos personales innecesarios. |
| `PublicadorEventos` | Salida | Publicar desde la bandeja transaccional confirmada; nunca sustituye la transaccion autoritativa. |
| `PuertoNotificaciones` | Salida habilitable | Notificar subsanacion o publicacion con minimizacion y preferencia de canal. |

## Invariantes

- Ninguna lectura de datos no publicos, recuperacion de bytes o renderizado con
  datos personales comienza antes de una preautorizacion PDP vigente.
- Ningun efecto irreversible comienza antes de una autorizacion exacta y de una
  intencion durable con identidad semantica V2, pasos y claves idempotentes.
- Cada efecto remoto se acredita con un recibo material V2 verificado contra la
  solicitud exacta; una referencia o huella aislada no acredita el resultado.
- Una solicitud pertenece a una unica convocatoria y a un unico candidato.
- Una convocatoria publicada no muta reglas en sitio; cualquier cambio crea
  nueva version trazable.
- Un merito sin regla aplicable no puntua y debe quedar como incidencia de
  configuracion, no como cero silencioso.
- Los puntos usan aritmetica decimal fija, no `float64`; el mismo conjunto de
  entradas y la misma version de reglas producen siempre los mismos bytes.
- Solo meritos `Validado` deben computar en baremo administrativo definitivo;
  el autobaremo ciudadano puede marcar `Presentado` como provisional.
- Una decision tecnica sin firma validada nunca es eficaz ni altera el total.
- Una rectificacion solo puede sustituir la decision vigente exacta; la
  anterior permanece inmutable y nunca se elimina.
- Revocar una aceptacion, rehabilitar un rechazo y confirmar un doble control
  son acciones distintas, firmadas y autorizadas de manera independiente.
- La evaluacion de un documento y la valoracion del merito son datos distintos:
  retirar la aptitud de una evidencia fuerza el recalculo, pero no borra el
  documento ni su historial.
- Solo solicitudes `AdmitidaDefinitiva` entran en la clasificacion de bolsa.
- Todo documento exportable requiere digest SHA-256, metadatos ENI aplicables,
  almacenamiento opaco, antivirus limpio, manifiesto y CSV o evidencia de
  verificacion cuando corresponda a su origen.
- Todo listado publicado requiere huella, firma, actor y auditoria encadenada.
- Criterios, causas, formatos, calendarios y transiciones ampliables proceden de
  una version publicada de catalogo; no se fijan mediante listas dispersas en
  el codigo.
- Las proyecciones publicas contienen solo campos expresamente publicados. Los
  datos internos o personales permanecen fuera aunque compartan agregado de
  origen.

## Estado de evidencia y exclusion del prototipo V1

Este documento no mantiene una tabla de «existe hoy» porque el arbol de trabajo
no constituye evidencia suficiente de una capacidad. Un archivo, una interfaz,
un adaptador de memoria o una prueba unitaria pueden demostrar una idea local,
pero no acreditan integracion, persistencia durable, autorizacion completa,
seguridad operativa ni despliegue.

El prototipo V1 de firma y baremacion queda rechazado como camino productivo por
estas razones de arquitectura:

- permite representar resultados externos mediante referencias o huellas
  genericos en lugar de recibos materiales nominales y atestados;
- no garantiza en toda la vertical una preautorizacion antes de cada lectura de
  datos sensibles;
- puede ejecutar firma, custodia o retencion antes de haber persistido una
  intencion durable y reconciliable;
- delega parte de la autorizacion exacta al ejecutor o a dobles de prueba;
- un adaptador de memoria no acredita recuperacion tras reinicio, exclusion
  monotona entre trabajadores,
  revocacion autoritativa ni atomicidad con PostgreSQL u Oracle;
- una prueba de camino feliz no acredita fallos ambiguos, rotacion de claves,
  concurrencia, repeticion, caida ni compromiso de un conector.

Por tanto, V1 no debe exponerse, cablearse en una composicion canonica,
migrarse como contrato de persistencia ni describirse como funcionalidad
terminada. Su codigo, si se conserva temporalmente, sirve solo para extraer
casos de prueba y requisitos que deberan expresarse de nuevo mediante contratos
V2/V4. La sustitucion no se acredita hasta superar las condiciones de
aceptacion siguientes.

| Area | Condicion minima para acreditar implementacion |
| --- | --- |
| Identidad y sesion | Integracion real con los proveedores aprobados, sesion durable y revocable, garantia autenticada y pruebas de suplantacion, caducidad y revocacion. |
| Autorizacion | PDP con preautorizacion anterior a lectura/renderizado, decision exacta por efecto y revalidacion autoritativa en la frontera de confirmacion. |
| Idempotencia | Identidad semantica V2, matriz completa durante rotacion, conflicto por preimagen diferente y resultado indeterminado sin repeticion ciega. |
| Firma y custodia | Plan durable anterior al primer efecto, recibos materiales V2 verificados, claves en KMS/HSM, reconciliador y pruebas de caida entre todos los pasos. |
| Persistencia | Migraciones revisadas, control de concurrencia optimista, testigos monotonos de exclusion, agregado, auditoria y bandeja transaccional atomicas, recuperacion tras reinicio y comportamiento equivalente en el conector admitido. |
| Convocatorias y baremo | Catalogos y reglas versionados, decimal fijo, periodos/solapes/incompatibilidades y reproduccion byte a byte de un calculo historico. |
| Revision y rectificacion | Decisiones inmutables firmadas, sustitucion por referencia y huella exactas, separacion de acciones, conflicto de interes y doble control configurable. |
| Documentos | Carga directa a cuarentena, antivirus, promocion, retencion, lectura autorizada, ENI, formatos gobernados y ausencia de rutas o datos personales en registros. |
| Exposicion | Analisis de amenazas, pruebas de aislamiento interno/externo, pruebas de autorizacion negativa y aprobacion expresa de Seguridad, Sistemas y responsable funcional. |

## Cortes verticales propuestos

| Corte | Objetivo | Frontera hexagonal | Prueba focal |
| --- | --- | --- | --- |
| VS0 Contratos de seguridad | Fijar preautorizacion, PDP exacto, idempotencia semantica V2, recibos materiales V2 y planes durables. | Valores opacos y puertos sin adaptador productivo implicito. | Matriz completa en rotacion, redaccion de secretos, interfaces con punteros nulos tipados, concesion ajena/caducada y denegacion por defecto. |
| VS1 Convocatoria versionada | Crear y publicar convocatoria con reglas, catalogos y calendario. | Nucleo `Convocatoria`; repositorio intercambiable; composicion de prueba cerrada. | Crear version, rechazar reglas invalidas, no mutar version publicada y reproducir una version historica. |
| VS2 Borrador VEC | Crear o recuperar un borrador mediante identidad semantica V2. | Caso de uso de solicitud; repositorio durable de operacion, auditoria y bandeja transaccional. | Reintento equivalente, conflicto por preimagen distinta, revocacion de sesion y ausencia de lectura previa al PDP. |
| VS3 Importacion RUM | Importar meritos externos y deduplicar por referencia RUM. | `ProveedorMeritosRUM`; normalizador en aplicacion. | Misma referencia no duplica, tipo desconocido queda incidencia y ningun dato se consulta sin autorizacion. |
| VS4 Autobaremo explicable | Calcular autobaremo provisional con detalle preparado para i18n. | Nucleo de baremo puro; adaptadores solo proyectan vistas autorizadas. | Decimal fijo, topes, solapes, regla ausente, version congelada y mismos bytes para mismas entradas. |
| VS5 Carga documental | Subir directamente a cuarentena, analizar y promover. | Puertos de almacen, antivirus, planes durables y recibos V2. | Caida entre pasos, recibo adulterado, cuarentena cerrada, tamaño/huella y no repeticion de efecto incierto. |
| VS6 Revision tecnica | Validar o rechazar documento a documento y pedir subsanacion. | Decision inmutable; PDP, firma, repositorio transaccional y auditoria. | Lectura autorizada, firma obligatoria, motivo, puntos declarados/calculados/reconocidos y recalculo reproducible. |
| VS6B Rectificacion inspectora | Revocar una aceptacion o rehabilitar un rechazo sin borrar historia. | Nueva decision de solo adicion que sustituye referencia y huella exactas. | Cambio en ambos sentidos, cambio sin efecto prohibido, conflicto de concurrencia optimista, firma, doble control y efecto sobre listados. |
| VS7 Registro y alegaciones | Registrar solicitud, alegacion y resolucion con efectos legales separados. | Puertos de registro, firma, notificacion y expediente. | Intencion durable anterior al efecto, CSV/asiento, plazo, recibos y recuperacion tras reinicio. |
| VS8 Bolsa definitiva | Publicar listado definitivo y constituir bolsa ordenada. | Clasificacion en nucleo; publicacion mediante bandeja transaccional y conector firmado. | Solo admitidos definitivos, desempates de catalogo, huella/firma y proyeccion publica minimizada. |
| VS9 Exportacion de expediente | Exportar manifiesto oficial en un formato gobernado. | `ExpedienteElectronico` puro; perfiles publicados y conectores de almacenamiento/firma. | Documento en cuarentena bloquea exportacion, formato no homologado deniega y manifiesto reproduce todos los recibos. |

Ningun corte se considera implantado por aprobar solo pruebas con memoria. Cada
corte debe entregar por separado dominio y puertos, aplicacion, adaptador
durable, migraciones, pruebas negativas/de fallo, documentacion operativa y
revision de seguridad. La interfaz externa permanece cerrada hasta completar
esa cadena para el caso de uso concreto.

## Siguiente paso recomendado

El primer paso de codigo es cerrar y revisar VS0 sin reutilizar el flujo V1.
Despues puede implementarse un corte pequeño de convocatoria o borrador sobre
PostgreSQL, manteniendo Oracle como adaptador futuro, con control optimista,
testigos monotonos de exclusion, auditoria y bandeja transaccional atomicas. No
se añadiran rutas sensibles hasta que la preautorizacion de lectura, la
revalidacion transaccional, la sesion real y la
recuperacion tras caida esten demostradas para ese corte.

Firma, sello de tiempo, custodia, antivirus, registro y notificacion se
sustituyen uno a uno por conectores reales. Cada sustitucion necesita pruebas de
contrato, fallo y reconciliacion; la equivalencia de interfaz no permite rebajar
las obligaciones de seguridad del puerto.
