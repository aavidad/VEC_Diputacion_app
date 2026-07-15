# Matriz inicial de perfiles, roles y ambitos

Estado: linea base funcional y de seguridad. Las asignaciones definitivas
requieren validacion de RRHH, Secretaria, Seguridad, DPD y responsables de cada
modulo.

Fecha de corte: 15 de julio de 2026.

## Regla general

La politica siempre crece de menos a mas. Si no existe una concesion publicada
que autorice expresamente modulo, accion, tipo y referencia/contexto del
recurso, conjunto exacto de dimensiones de ambito, finalidad, campos y
obligaciones, la operacion se deniega. El menu, un nombre de rol enviado por el
navegador o la pertenencia generica a Active Directory no conceden acceso.

La regla operativa es una lista positiva cerrada: lo no declarado no existe
como capacidad. Una lista vacia, un dato ausente, un codigo desconocido, una
decision ambigua, una politica caducada, un repositorio no disponible o un
error de evaluacion producen el mismo resultado seguro: denegacion. No se
aplican permisos heredados, aproximaciones por nombre, normalizaciones
permisivas ni valores predeterminados que amplien acceso.

Ante varias reglas aplicables prevalece siempre el resultado mas restrictivo.
Las politicas ABAC solo pueden reducir una concesion RBAC previa, nunca crearla;
las restricciones de campos se intersectan, una prohibicion expresa prevalece
y los permisos de perfiles distintos no se suman. Para ampliar autoridad hay
que publicar una nueva concesion exacta y asignarla mediante el procedimiento
gobernado correspondiente.

Omitir un filtro tampoco concede amplitud. Una referencia, lista de tipos o
dimension de alcance vacia, repetida, no canonica o con comodines se deniega;
no se interpreta como «consultar todo». Las consultas globales se modelan como
operaciones distintas, con permiso, finalidad, limites y traza propios.

Cada caso de uso comprobara en el servidor:

1. superficie y sesion correctas;
2. identidad efectiva y, si existe, persona representada;
3. un unico perfil activo;
4. version publicada del rol y asignacion vigente;
5. modulo, accion, tipo, referencia y huella del contexto del recurso exactos;
6. igualdad cerrada de dimensiones de ambito, coincidencia positiva de cada
   valor, relacion con el recurso y finalidad;
7. garantia y antiguedad de la autenticacion;
8. campos concretos que el caso de uso puede leer o modificar, si proceden;
9. restricciones, incompatibilidades y obligaciones implementadas.

Si una decision contiene campos que el caso de uso no sabe aplicar, se
deniega. Si contiene una obligacion —por ejemplo doble control, firma o
registro reforzado— que el caso de uso no implementa y acredita, se deniega.
Ignorar una restriccion nunca es una opcion de compatibilidad.

No existe ningun comodin positivo, tampoco `global=["*"]`. Las dimensiones de
la asignacion y del recurso deben ser exactamente las mismas y cada valor se
enumera: una dimension o valor nuevo, adicional o ausente deniega aunque las
restantes coincidan. Una operacion transversal se modela como otro caso de uso
sobre una referencia de conjunto gobernada y versionada; ampliar ese conjunto
exige publicar una nueva version y sus concesiones.

La evaluacion usa una instantanea coherente de asignacion, version exacta de
rol, control de vigencia global de esa version y catalogo completo de
politicas. La concesion no es valida hasta que el registro de solo adicion
complete un CAS sobre asignacion, control de rol y revision/huella del catalogo.
Una revocacion o publicacion concurrente que gane la carrera deja la operacion
denegada y no registra la concesion anterior.

La concesion vence en la primera frontera entre su TTL tecnico, el fin de la
asignacion y el proximo inicio o fin conocido de una politica publicada. Para
producir un efecto administrativo, el repositorio vuelve a validar decision,
uso y configuracion actual dentro de la misma transaccion que escribe el
efecto. Un TTL corto limita exposicion, pero no sustituye este control contra
revocaciones concurrentes.

Los roles, concesiones y asignaciones son datos gobernados, versionados,
fechados y revocables. Una accion nueva no entra en roles antiguos mediante un
comodin. Los estados e invariantes tecnicos de seguridad si permanecen cerrados
en codigo.

La persistencia repite la misma regla como defensa adicional: identidades
tecnicas separadas por modulo y superficie, privilegios SQL exactos y seguridad
por filas sin politica permisiva predeterminada. No sustituye la decision del
nucleo ni convierte un rol de base de datos en rol funcional. Se detalla en
`docs/portal_vec/seguridad_persistencia_postgresql.md`.

Las acciones con efectos distintos nunca comparten una concesion generica. En
particular, adoptar una primera decision de baremacion, rectificarla, revocar
una aceptacion y rehabilitar un rechazo son cuatro capacidades independientes.
Poder baremar no convierte al tecnico en inspector ni le permite deshacer una
decision eficaz. Cada una exige su perfil, ambito, garantia, motivacion y firma
positivamente declarados; la ausencia de cualquiera deniega.

## Superficies

| Superficie | Perfiles admitidos | Frontera |
| --- | --- | --- |
| Exterior anonima | Visitante | Solo informacion expresamente publicable |
| Exterior autenticada | Titular/aspirante y representante acreditado | Solo recursos propios o validamente representados |
| Interna de empleado | Empleado | Autoservicio y funciones corporativas propias desde Mulhacen/VPN |
| Interna de gestion | RRHH y organos de seleccion | Solo trabajo, procedimientos y unidades asignados |
| Administracion privilegiada | Cuentas administrativas nominativas | Segmento de gestion, elevacion temporal y controles reforzados |
| Servicio a servicio | Identidades tecnicas | Audiencia y operaciones minimas de un conector; nunca sesion humana |

Un empleado que participe en una bolsa entra como ciudadano por la superficie
exterior y como empleado por la interna. Sus sesiones y perfiles no se mezclan
ni suman privilegios.

## Perfiles exteriores

| Perfil | Puede | No puede |
| --- | --- | --- |
| Visitante | Consultar convocatorias, bases, plazos, ayuda, preguntas publicas y documentos legalmente publicados | Consultar expedientes, identidades o documentos protegidos |
| Titular o aspirante | Gestionar sus datos aportados, solicitudes, meritos, documentos, autobaremo, subsanaciones, alegaciones, avisos y certificados propios | Ver datos de otra persona o alterar una decision administrativa |
| Representante | Actuar para la persona, tramite y periodo cubiertos por representacion verificada | Ampliar la representacion por declaracion propia o reutilizarla fuera de alcance |

## Perfiles internos de RRHH y seleccion

| Perfil base | Capacidad | Limites obligatorios |
| --- | --- | --- |
| Empleado | Autoservicio de datos, nominas, dietas, permisos, fichajes, certificados y comunicaciones propias | Titularidad; una excepcion exige otro perfil activo |
| Administrativo de RRHH | Completar, requerir, subsanar, clasificar y tramitar expedientes | Convocatoria, unidad, fase y tareas asignadas |
| Tecnico de seleccion o baremador | Evaluar requisitos, documentos y meritos; preparar decisiones y lotes de firma | Criterios y expedientes asignados; abstenciones e incompatibilidades |
| Revisor o inspector | Revisar decisiones eficaces y firmar rectificaciones motivadas | No borra decisiones anteriores; toda sustitucion conserva la cadena |
| Responsable de RRHH | Supervisar cargas y autorizar las operaciones funcionales configuradas | No administra claves ni elude firma, quorum o doble control |
| Gestor de convocatoria | Preparar calendario, tramites, plantillas, reglas y catalogos funcionales | Publicacion separada de preparacion cuando lo exija la politica |
| Gestor de comunicaciones | Preparar plantillas, segmentos y envios autorizados | Sin acceso indiscriminado al contenido de expedientes o destinatarios exportables |

## Organos de seleccion

| Perfil base | Capacidad | Limites obligatorios |
| --- | --- | --- |
| Vocal | Consultar lo necesario, deliberar, votar y firmar | Solo procedimientos con nombramiento vigente |
| Presidencia | Dirigir, comprobar quorum, aprobar contenido y firmar acuerdos | No sustituye a Secretaria ni altera una version ya firmada |
| Secretaria | Preparar/custodiar actas, certificar acuerdos y coordinar firmas | No inventa decisiones ni modifica bytes durante el circuito |
| Suplente | Ejercer temporalmente la funcion sustituida | Sin acceso mientras la sustitucion no este activada y documentada |

Composicion, asistencia, ausencia, abstencion, sustitucion, quorum y orden de
firma se congelan por acta. Un grupo generico no equivale a nombramiento para
todas las convocatorias.

## Control, privacidad y soporte

| Perfil base | Capacidad | Limites obligatorios |
| --- | --- | --- |
| Auditor interno | Consultar trazas, decisiones y muestras autorizadas | Solo lectura, finalidad, periodo y muestra acotados |
| Auditor de seguridad | Examinar eventos tecnicos, accesos y evidencias de control | Datos funcionales minimizados o seudonimizados salvo necesidad aprobada |
| DPD o equipo autorizado | Supervisar tratamientos, derechos e incidencias | Acceso excepcional por caso/finalidad, no acceso permanente indiscriminado |
| Soporte | Diagnosticar salud y metadatos tecnicos minimizados | Sin suplantar, ver documentos ni cambiar resultados |
| Control interno o intervencion | Fiscalizar actuaciones de su competencia | Modulo, expediente, periodo y finalidad fiscalizable |

## Administracion privilegiada

| Perfil base | Capacidad | No concede por si mismo |
| --- | --- | --- |
| Administrador de identidades y acceso | Versionar roles, asignaciones y revocaciones mediante circuito aprobado | Leer expedientes |
| Administrador de seguridad | Politicas tecnicas, identidad y respuesta | Firmar actos de RRHH o leer documentos masivamente |
| Operador de plataforma | Desplegar, observar, copiar y recuperar servicios | Acceso funcional a datos en claro |
| Administrador de base de datos | Operacion tecnica controlada | Autorizacion funcional o modificacion directa de negocio |
| Custodio de claves | HSM/KMS, rotacion y recuperacion con doble control | Usar claves para otra finalidad o consultar expedientes |
| Configurador funcional | Catalogos, plantillas, reglas y flujos de su dominio | Administrar infraestructura/identidades o aprobar su propio cambio sensible |

No existira un `superadministrador` universal. El acceso de emergencia sera
temporal, justificado, aprobado, alertado y revisado.

## Ambitos combinables

Una concesion se restringe mediante dimensiones obtenidas de fuentes fiables:

- organismo, modulo, unidad, servicio o centro;
- convocatoria, bolsa, oferta y fase;
- criterio, lote, tarea o expediente asignado;
- territorio cuando proceda;
- relacion: titular, representante, subordinado, tramitador o miembro nombrado;
- finalidad y periodo temporal;
- clasificacion y campos que se pueden mostrar o exportar.

Varias dimensiones se combinan restrictivamente. Una necesidad transversal
usa una accion distinta y una referencia exacta a un conjunto gobernado,
versionado, excepcional y temporal. Nunca se expresa omitiendo dimensiones ni
mediante `*`; la ausencia de un ambito necesario deniega.

## Perfiles propios de modulos

Cada modulo puede publicar perfiles funcionales sin modificar el nucleo:

- Cronos: empleado, responsable jerarquico, gestor de presencia, gestor de
  permisos y auditor. El responsable solo alcanza subordinados y periodos de
  su unidad vigentes.
- Dietas: solicitante, responsable, tramitador, fiscalizador y pagador,
  manteniendo las separaciones del procedimiento.
- Nominas: titular, gestor y auditor; operar el conector no concede acceso
  funcional.
- Bolsa: aspirante, administrativo, baremador, revisor, organo de seleccion y
  responsable de convocatoria.

Son perfiles de referencia, no constantes inamovibles. Sus acciones se
publican al activar el modulo y no amplian roles existentes automaticamente.

## Segregaciones minimas

- Quien prepara una regla o politica sensible no la publica por si solo cuando
  exista doble control.
- Quien presenta no barema ni revisa su solicitud usando otro perfil.
- Conflicto, abstencion o incompatibilidad excluyen el expediente aunque el rol
  general lo permita.
- Una decision eficaz solo cambia mediante otra decision firmada y de solo
  adicion.
- Administrar plataforma, base, claves o identidad no concede acceso funcional.
- Exportaciones masivas, publicaciones, roles, desbloqueos de retencion,
  devoluciones y emergencias pueden exigir dos personas distintas.
- Soporte no inicia sesion como el usuario. Una futura vista asistida sera
  minimizada, autorizada, visible y auditada.

## Ciclo de vida

Cada version de rol conserva concesiones exactas, garantia, campos,
obligaciones, autor, motivo, fecha y huella. Su retirada global usa un control
separado y versionado que nombra exactamente la version, actor, fecha, acto y
motivo tipificado; una version posterior retirada no revoca otra por
inferencia. Cada asignacion conserva persona, perfil, version de rol, ambitos,
vigencia, emisor y revocacion.

Active Directory y otros sistemas pueden proponer altas, cambios y bajas, pero
no convierten grupos ciegamente en autorizaciones. Antes de publicar un cambio
se simulan los accesos ganados y perdidos. La baja urgente revoca asignaciones
y sesiones dentro del objetivo temporal aprobado por Seguridad.

## Pruebas de aceptacion

- Una URL interna no es alcanzable desde la entrada exterior.
- Un aspirante solo obtiene recursos propios o representados.
- Dos perfiles de una persona no suman permisos en una sesion.
- Un tecnico no asignado recibe denegacion aunque posea el rol base.
- Una accion nueva queda denegada para todos hasta publicar concesiones.
- Un administrador de plataforma no puede leer una baremacion por administrar
  el despliegue.
- Un baremador con permiso de decision inicial no puede rectificar, revocar ni
  rehabilitar una decision; esas acciones requieren concesiones inspectoras
  separadas y, cuando la politica lo ordene, una segunda persona.
- Una cabecera de rol enviada por el cliente no altera la decision.
- Una asignacion revocada, version retirada o politica no disponible falla
  cerrada.
- Una concesion de Bolsa con la misma accion y tipo no cruza a Cronos ni a otro
  modulo.
- Una dimension o valor de ambito adicional, ausente o nuevo deniega; no hay
  excepcion por alcance global ni comodines positivos.
- Si cambia una politica, la asignacion o el control de vigencia del rol entre
  lectura y registro, el CAS no inserta la concesion.
- Si no existe una instantanea fiable, se deniega sin fabricar evidencia de
  evaluacion.
- Una decision con un campo futuro o una obligacion no implementada no ejecuta
  parcialmente la operacion: se deniega y se audita.
- Cada decision permite reconstruir identidad, perfil, accion, recurso,
  finalidad, versiones e instante sin copiar datos personales innecesarios.
