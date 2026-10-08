# Plan de Incompatibilidades — 4 de octubre de 2026

**Aparcado hasta cerrar Bolsa y CT (orden de Alberto, 07/10/2026).** Los objetivos vigentes están en [OBJETIVOS.md](OBJETIVOS.md).

VEC tramitará declaraciones y solicitudes de compatibilidad, informes, decisión,
condiciones y revisiones. La persona declara los hechos y el órgano competente
decide. La primera entrega propuesta conserva una solicitud sintética y su recibo
interno; la concesión, el asiento oficial, la firma, la notificación y la publicación
requieren sus circuitos acreditados. Este encargo entrega documentación, sin código
ni SQL, y mantiene la cola vigente.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos: [ficha de Incompatibilidades integrada](../estudio_requisitos/ficha_incompatibilidades_2026-10-04.md),
INC1–INC10. Personal B, identidad/autorización K y núcleo/auditoría L conservan
sus archivos y responsabilidades. No se crean maestros ni autoridades paralelas.

## Fuentes y decisiones aplicables

La ficha recoge [Ley 53/1984](https://www.boe.es/eli/es/l/1984/12/26/53/con),
[RD 598/1985](https://www.boe.es/eli/es/rd/1985/04/30/598/con),
[Ley 19/2013](https://www.boe.es/eli/es/l/2013/12/09/19/con) y la
[relación provincial actualizada a 30/01/2026](https://www.dipgra.es/export/sites/diputaciongranada/servicios/areas/transparencia/.galleries/AREAS-Transparencia-Documentos/Autorizaciones-de-compatibilidad.pdf).
La relación pública acredita publicación de acuerdos; no describe la solicitud
ni autoriza un conector. No se copian sus datos personales al plan.

Se conserva la distinción entre segunda actividad pública, actividad privada y
supuesto exceptuado. Los arts. 9 y 14 de la Ley 53/1984 determinan competencias
locales y condiciones diferentes. El art. 1 del RD 598/1985 excluye la actividad
principal local: sus órganos y plazos no se cargan como procedimiento provincial.
La regla del art. 16 conserva el fundamento temporal señalado por la ficha.
No se introduce una renovación anual universal ni se decide por una casilla.

Aplican también [materias reservadas, §§9 y 12](../estudio_requisitos/materias_reservadas_economicas_y_relaciones_laborales.md),
[arquitectura](../portal_vec/arquitectura_tecnica.md),
[contrato de módulos](../portal_vec/contrato_modulos_vec.md),
[roles y ámbitos](../portal_vec/matriz_roles_y_ambitos.md) y
[cumplimiento](../portal_vec/cumplimiento_y_seguridad.md).
La configuración interna pendiente está en la duda 133 de [dudas.md](../../dudas.md).
Este plan no la modifica ni vuelve a preguntar por normas ya publicadas.

## Inventario en la base inspeccionada

Primero se consultó `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Es una instantánea anterior; las rutas y capacidades siguientes se cotejaron con
`git ls-tree` y lectura en el SHA indicado. El índice no acredita actualidad.

| Pieza | Rutas actuales | Capacidad y límite |
| --- | --- | --- |
| Incompatibilidades | `docs/estudio_requisitos/ficha_incompatibilidades_2026-10-04.md` | Requisitos integrados. No hay paquete del módulo en `internal/modules/`, vista propia ni persistencia de incompatibilidades en esta base. |
| Personal | `internal/modules/personal/ports/{relacion_empleado,historia_relaciones_propia,lector_relacion_rpt}.go`; `application/consulta_relacion_empleado.go` | Relación propia para Dietas y contratos históricos/RPT con autorización específica. No son un lector general para Incompatibilidades: el caso de uso de Dietas fija acción y audiencia. B debe admitir el puerto mínimo a fecha y su cobertura. |
| Persona y perfiles | `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Contexto canónico y administración de perfiles existentes. Faltan catálogo y consumidor nominal propios de esta materia; no se presta el actor de ADMIN. |
| Usuarios | `internal/modules/usuarios/application/{correos,correo_avisos}.go`; `ports/correo_avisos.go` | Contactos bajo su autoridad. `ConCorreoActivoAvisos` se limita a llamamientos de candidatos; no habilita notificaciones de compatibilidad. Se acuerda otro consumo mínimo con su propietario si hace falta. |
| Documentos y firma | `internal/vec/documentos/application/servicio.go`; `ports/{contratos,firma}.go`; `adapters/validadorautofirma/cliente.go` | Custodia/lecturas y verificación GrxFirma v2 existentes. Faltan tipos, relación expediente-documento, circuito y competencia de Incompatibilidades. Verificar una firma no resuelve ni acredita notificación. |
| Cronos | `internal/modules/cronos/ports/{solicitud_permiso,resolucion_permiso}.go` | Solicitudes y decisiones propias de Cronos. No aportan un permiso de compatibilidad ni un lector intercambiable de jornada; acordar el hecho autorizado por puerto. |
| Auditoría común | `internal/vec/ports/{auditoria_intento_nominal,auditoria_frontera_ruta_exacta}.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Registro común de intentos nominales denegados/error y frontera sin identidad. El permitido debe unirse a la lectura/efecto propio. Falta componer ese consumo para el nuevo módulo. |
| Registro y catálogos | `internal/vec/domain/types.go`; `internal/vec/application/service.go`; `internal/vec/ports/ports.go`; `web/static/portal-empleado/portal-catalogo-modulos.js`; `data/catalogos/` | Manifiestos, registro y catálogos comunes. No instalan automáticamente módulos, handlers, reglas o SQL; montaje y manifiestos se solicitan a sus custodios. |

Integrado en Git, instalado en PostgreSQL, publicado y comprobado en navegador
son estados distintos. Este inventario no afirma instalación ni un recorrido de
ninguna de esas piezas para Incompatibilidades.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Propietario y dependencia |
| --- | --- | --- |
| INC1–INC4 | Solicitud reservada, actividad y evidencias; informes/subsanaciones con procedencia y versión. | Incompatibilidades conserva expediente; B aporta vínculo/puesto a fecha; Documentos custodia originales. |
| INC5 | Propuesta, decisión competente, firma y notificación con evidencias separadas. | Tramitación/Jurídico/órgano habilitado según 133; servicios comunes conservan firma, registro y notificación. |
| INC6 | Revisar cambios de relación, puesto, horario o actividad y registrar acto posterior. | Incompatibilidades coordina; Personal/Cronos emiten hechos mínimos. Una alerta no revoca. |
| INC7/INC9 | Consulta, descarga y cada operación con autorización y auditoría nominal. | Consumidor del módulo con K/L y Documentos; sin acceso material por ser administrador. |
| INC8 | Proyección publicable revisada, publicación/retirada y versión pública separadas. | Transparencia valida los campos; el expediente reservado no se expone. |
| INC10 | Archivo, conservación y eliminación autorizada con traza mínima. | Archivo/DPD determinan serie; Documentos aplica la custodia, copias y expurgo admitidos. |

Un perfil activo fijo y provisionado por huella/CAS identifica solicitante,
tramitador, revisor jurídico, resolutor o revisor de transparencia según su
concesión. Cada operación exige entidad, expediente, actuación, acción, finalidad,
campos y vigencia exactos. Representación acreditada y suplencias se revalidan;
el cliente no elige identidad ni recibe permisos al enviar una petición.

Cada consulta y descarga tiene autorización propia y auditoría común de actor,
perfil, acción, recurso opaco, finalidad, instante, resultado
`permitido/denegado/error`, correlación, proceso y canal. Los permitidos acompañan
la lectura; los denegados/errores usan el registrador común después del cierre del
intento. La frontera sin identidad no fabrica una persona. Efectos, consumo de
autorización, versión, recibo, historia y auditoría se confirman en la misma
transacción; outbox solo para una entrega concreta con consumidor.

Dominio y aplicación son neutrales respecto a HTTP, SQL y proveedor. Personal,
Cronos, Documentos, Usuarios y Transparencia se consumen por puertos mínimos,
referencias opacas y contratos versionados; no se leen ni escriben sus tablas.
Bolsa no recibe actividades privadas. La jefatura recibe únicamente la condición
que deba aplicar. El expediente y sus anexos quedan fuera de la ficha general.

## Configuración y dudas

Los supuestos, documentos, informes, órganos/suplencias, condiciones y reglas
se gobiernan con fuente/artículo, publicación, versión/huella, entidad/colectivo,
vigencia/efectos y órgano aprobador. La versión aplicada permanece en el expediente;
RRHH mantiene lo aprobado, sin editar libremente obligaciones legales.

La duda 133 solicita unidad tramitadora, circuito de informes/revisión/firma,
plantillas, repositorio, integración mínima con Personal y aprobación de
transparencia. Contratos, inventario y prueba sintética pueden avanzar mientras
se responde. La decisión y la publicación reales esperan esas competencias.
RAT, bases por finalidad, riesgos y conservación se documentan antes de datos
reales. No se copian DNI, familia o banco; importes y empleadores solo cuando
los necesita el supuesto, con descarga restringida.

## Minitareas y salidas por PR

Son cortes futuros. Dirección confirma SHA base actualizado y archivos exclusivos
antes de asignar. Las rutas nuevas son previsiones, no interfaces aprobadas.
B1 es autorización/auditoría; B2 vínculo; B3 organización y catálogos; B4 expediente
durable; B5 documentos/firma; B6 intercambio y proyección. Los custodios producen
sus deltas comunes en PR separadas; no se toma su código ni se suma su esfuerzo aquí.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| I00 · inventario | Revalidar la deuda en la base vigente y el circuito de 133. | Este plan, en su turno documental. | Seguimiento y propietarios; sin rehacer piezas integradas. | 1–2 |
| I01 · B3 | Catálogo de supuestos y documentos validado por versión; consumidor CLI informa regla ausente/conflicto. | Nuevos `data/catalogos/incompatibilidades/`; lector propio. | Ficha y circuito/competencias admitidos; provisional rotulado solo en prueba. | 3–5 |
| I02 · B1 | Consulta nominal focal permite/deniega recurso exacto y deja los tres resultados en auditoría común. | Nuevos `incompatibilidades/ports/{autorizacion,auditoria}.go`, consumidor propio. | ABI/proveedor/emisor K y auditoría L; perfiles publicados fuera de petición; montaje por custodio. | 6–10 |
| I03 · B2/B3 | Consultar vínculo/puesto/jornada a fecha con cobertura y procedencia; rechazar ausencia/ambigüedad. | `incompatibilidades/ports/personal.go`, consumidor y pruebas propios. | I02; B admite contrato, organización aporta entidad; no reutilizar acción Dietas. | 4–6 |
| I04 · B4, SQL borrador | Candidata de persistencia de solicitud/actuaciones y adaptador; ensayo preparado y mínimo efecto probatorio. | SQL nuevo reservado y `adapters/postgres/` propios. | I01–I03; reserva previa fuera de Git y preimagen. Sin activar en principal. | 4–7 |
| I05 · B4, ensayo | Ensayo clon principal, dos revisiones exactas y kit de instalación por dirección. | Pruebas de adaptador y kit propios. | I04; consumo V3, ACL, concurrencia e idempotencia; MCP local solo lectura. | 4–6 |
| I06 · INC2/4 | Crear/recuperar solicitud sintética con misma clave y recibo; actividad y documentos pendientes visibles. | `domain/solicitud.go`, `application/solicitud.go`, HTTP/PG propios. | I01–I05; sin concesión, asiento oficial ni envío. | 6–10 |
| I07 · INC7 | Lista propia y detalle, carga/error/denegación y recuperación del recibo por web. | Vista/cliente propios y catálogos por idioma. | I06; registro/rutas/montaje por custodio; revisión usabilidad. | 3–5 |
| I08 · B5 | Aportar evidencia y descargar original vinculado al expediente con permiso/auditoría propios. | `ports/documentos.go`, consumidor documental, UI/textos propios. | I02/I06; contrato y custodia Documentos; límites y tipos aprobados. | 4–6 |
| I09 · INC4 | Subsanar y registrar informes/revisión jurídica, conservando aportaciones y versiones. | Casos de uso de actuaciones e interfaz propios. | I06/I08; asignación y circuito de informes de 133. | 5–8 |
| I10 · INC5, propuesta | Preparar propuesta motivada con condiciones desde expediente revisado. | Caso de uso/documento de propuesta y textos propios. | I09 y plantillas/catálogo vigentes; no declarar resolución. | 3–5 |
| I11 · INC5, decisión | Registrar acto competente y evidencia de firma original, con separación preparador/resolutor. | `ports/resolucion.go`, consumidor firma y decisión propios. | I10; órgano/delegación comprobados y B5 común configurado. | 4–7 |
| I12 · INC5, comunicación | Enlazar asiento y notificación acreditados; reintentar sin repetir envío. | Consumidor de Registro/notificación y reconciliación propios. | I11; adaptadores oficiales, contacto autorizado Usuarios, recibo del destinatario. | 4–6 |
| I13 · INC6 | Cambio de puesto/actividad abre revisión; acto posterior conserva condiciones/historia. | `ports/cambios.go`, caso de uso y receptor propios. | I11; hechos mínimos B/Cronos y regla aplicable. No revocación automática. | 4–7 |
| I14 · INC8 | Preparar/revisar/publicar o retirar una proyección mínima, recuperable por versión. | Proyección propia y adaptador Transparencia por contrato. | I11; aprobación funcional/DPD, destinatario admitido; excluye anexos privados. | 3–5 |
| I15 · INC10 | Archivo/expurgo con evidencia mínima y pruebas de copias/restauración. | Consumidor Archivo/Documentos y pruebas propios. | Serie y política aprobadas; I08/I11; no borrar historia conservada indiscriminadamente. | 4–6 |
| I16 · entrega | Recorrido Chrome, negativos y reinicio; manual distingue recepción interna y efectos legales. | Pruebas focales y manual del módulo. | Capacidades instaladas anteriores; revisiones sensibles/usabilidad. | 4–6 |

I06/I07 entregan una solicitud utilizable en el ejercicio sintético. I08 e I09
amplían ese recorrido; I10–I12 esperan el circuito formal. Ninguna PR crea una
ruta vacía ni otro preparador sin consumidor. Cada efecto nuevo que amplíe el
SQL vuelve a reserva, ensayo y revisión antes de instalación.

## Dependencias, paralelismo y estimación

Las 17 filas suman **66–107 horas técnicas**, incluidas revisión y comprobaciones
focales: **9–14 jornadas de un equipo** de ocho horas, redondeadas al día completo.
La suma incorpora una provisión de SQL propio; si dirección decide consumir
exclusivamente un trámite institucional existente, se recorta el alcance y se
recalcula, sin presentar esa derivación como gestión del expediente.

Con dos equipos se estiman **7–12 jornadas** una vez disponibles las dependencias:
I00/I01 4–7 h; I02/I03 10–16 h; I04/I05 8–13 h; I06–I09 14–23 h con interfaz
y documentos repartidos por archivo; I10–I15 15–25 h: propuesta y decisión en
secuencia, luego un equipo conserva comunicación/seguimiento y otro publicación/
archivo; I16 4–6 h. Ese camino orientativo suma 55–90 h, redondeadas a 7–12
jornadas de calendario técnico. Sin solape se vuelve a 9–14.
Un equipo conserva solicitud/instrucción y otro documentos/proyección/seguimiento;
el caso de uso de decisión y los contratos compartidos tienen un solo escritor.

| Trabajo externo, fuera del total técnico | Dedicación orientativa | Condición |
| --- | ---: | --- |
| RRHH/Jurídico/órgano competente | 5–9 h efectivas | Resolver 133, validar reglas/competencias, plantillas y circuito; espera sin fecha comprometida. |
| DPD/Transparencia/Archivo | 4–8 h efectivas | Proyección, bases por finalidad, serie, conservación y eliminación. |
| Sistemas/Registro/firma/notificación | 5–10 h efectivas | Contratos, entorno y evidencia de servicios oficiales; disponibilidad no presumida. |
| Personal B, K/L, Usuarios/Cronos/Documentos | Estimación por sus dueños | Puertos y composición comunes; sus desarrollos no están incluidos en I02/I03/I08. |

Las horas miden esfuerzo, sin prometer fechas de publicación o respuesta externa.
La instalación solo corresponde a dirección y no se ejecuta SQL en cidonia desde A.

## Comprobación y entrega

El plan requiere comprobar enlaces locales, inventario, sumas y `git diff --check`.
La PR documental no instala ni aprueba una capacidad. No se necesitan Go,
contenedores, servicios o navegador para este cambio.

En implementación se aplicarán `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec` y `probar-recorridos-vec` según el corte. Antes de pantallas:
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec`,
`impeccable`/`VEC-PRIORIDAD.md`; revisión independiente de usabilidad. Textos con
`humanizer`/`VEC-USO.md`. SQL con `revisar-sql-vec`/`ensayar-sql`, reserva y ensayo
en clon principal, dos revisiones exactas en zonas sensibles; `security-audit`
focal y Semgrep local sin subir código. Go usa gopls; búsqueda primero por índice.
`revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec` cierran la entrega;
solo dirección integra y despliega.

Aceptar cada escritura con primer efecto, replay, conflicto, versión, lectura y
reinicio con mismo recibo/historia. Cubrir persona/entidad ajenas, representación
caducada, perfil revocado durante el efecto, descarga sin permiso y dependencia
caída. Verificar auditoría común de los tres resultados, ausencia de contenido
privado en eventos/logs, i18n por catálogos, teclado/foco y Chrome del sistema en
PC/móvil. Firma, registro y notificación solo se afirman con su evidencia real.
