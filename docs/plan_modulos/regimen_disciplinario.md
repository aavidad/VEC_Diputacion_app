# Plan de Régimen disciplinario — 4 de octubre de 2026

VEC conservará actuaciones previas, iniciación, designaciones, instrucción,
defensa, propuesta, resolución, recurso y ejecución en un expediente reservado.
La primera entrega propuesta prepara un expediente sintético y recupera su recibo
interno. La calificación, sanción y efectos legales esperan criterio jurídico y
competencias acreditadas. Una incidencia de Cronos o una denuncia no determina
responsabilidad. Este encargo entrega documentación, sin código, SQL o instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos: [ficha de Régimen disciplinario integrada](../estudio_requisitos/ficha_regimen_disciplinario_2026-10-04.md),
DIS1–DIS10. Se conserva la prioridad vigente; Personal B, identidad/autorización K,
núcleo/auditoría L y los demás módulos mantienen sus archivos y autoridades.

## Fuentes y decisiones aplicables

La ficha cita [TREBEP, arts. 93–98](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719),
[Estatuto de los Trabajadores](https://www.boe.es/eli/es/rdlg/2015/10/23/2/con),
[Ley andaluza 5/2023](https://www.boe.es/eli/es-an/l/2023/06/07/5/con),
[Convenio provincial de 2006](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/CONVENIO-2006.pdf)
y [Ley 39/2015](https://www.boe.es/eli/es/l/2015/10/01/39/con)/
[Ley 40/2015](https://www.boe.es/eli/es/l/2015/10/01/40/con).
El criterio por vínculo, entidad y procedimiento se solicita en la duda 134 de
[dudas.md](../../dudas.md); el plan no resuelve la aplicación interna ni modifica
esa pregunta.

La duración de doce meses del art. 174 de la Ley 5/2023 y los seis meses del
art. 74.2 del convenio no se convierten en opciones intercambiables del formulario.
El catálogo conserva la regla aprobada y su fundamento por régimen. Prescripción
de falta, prescripción de sanción y duración/caducidad del procedimiento son
cómputos distintos. Las alertas requieren revisión humana y hechos acreditados.
Instrucción y sanción tienen órganos distintos; resolución, ejecutividad y firmeza
conservan estados y evidencia separados según la ficha.

Se aplican [materias reservadas, §§8.1, 10 y 12](../estudio_requisitos/materias_reservadas_economicas_y_relaciones_laborales.md),
[arquitectura](../portal_vec/arquitectura_tecnica.md),
[contrato de módulos](../portal_vec/contrato_modulos_vec.md),
[roles y ámbitos](../portal_vec/matriz_roles_y_ambitos.md) y
[cumplimiento](../portal_vec/cumplimiento_y_seguridad.md).
El detalle jurídico procede de la ficha; no se crea una escala de faltas, sanciones
o plazos con interpretaciones nuevas.

## Inventario en la base inspeccionada

La búsqueda empezó por `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Las rutas y límites se cotejaron con `git ls-tree` y lectura en el SHA indicado,
porque ese índice corresponde a otra instantánea.

| Pieza | Rutas actuales | Capacidad y límite |
| --- | --- | --- |
| Disciplina | `docs/estudio_requisitos/ficha_regimen_disciplinario_2026-10-04.md` | Ficha integrada. No hay módulo disciplinario en `internal/modules/`, superficie específica ni persistencia de este expediente. |
| Personal | `internal/modules/personal/ports/{relacion_empleado,historia_relaciones_propia,lector_relacion_rpt}.go`; `application/consulta_relacion_empleado.go` | Consulta propia de relación para Dietas y contratos históricos/RPT. Sus acciones/audiencias no permiten investigar a otra persona. Falta el puerto a fecha y de inscripción/cancelación mínimo aprobado por B; la ficha ordinaria excluye causas disciplinarias. |
| Contexto y perfiles | `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Autoridades comunes disponibles en código. Faltan perfiles/designaciones y proveedor nominal de esta materia; no hay rol ADMIN material prestado. |
| Documentos/firma | `internal/vec/documentos/application/servicio.go`; `ports/{contratos,firma}.go`; `adapters/validadorautofirma/cliente.go` | Custodia, lecturas y verificación GrxFirma v2 comunes. Falta compartimento y relación expediente/actuación/documento; su existencia no acredita firma o notificación disciplinaria. |
| Cronos | `internal/modules/cronos/ports/{marcajes,justificacion,resolucion_permiso}.go` | Hechos de tiempo y circuitos propios. No califica faltas ni inicia expedientes. Falta un suministro minimizado a requerimiento autorizado, acordado con su dueño. |
| Usuarios | `internal/modules/usuarios/application/correos.go`; `application/correo_avisos.go`; `ports/correo_avisos.go` | Contactos canónicos; el consumo de avisos actual está limitado a llamamientos. No concede acceso a domicilios o un servicio de notificación para disciplina. |
| Auditoría | `internal/vec/ports/{auditoria_intento_nominal,auditoria_frontera_ruta_exacta}.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Intentos denegados/error y frontera común; permitido unido a la lectura/efecto. Falta consumidor/composición disciplinarios. Un registro local de actuaciones no sustituye esta auditoría. |
| Registro/montaje | `internal/vec/domain/types.go`; `internal/vec/application/service.go`; `internal/vec/ports/ports.go`; `web/static/portal-empleado/portal-catalogo-modulos.js`; `data/catalogos/` | Registro de manifiestos y catálogos reutilizables. No hay instalación automática ni montaje de Disciplina; cambios compartidos corresponden a sus custodios. |

No existe módulo Nóminas en `internal/modules/` en esta base, ni un receptor
disciplinario acreditado en Personal. El conector mínimo debe encargarse al dueño
de esa autoridad; no se sustituye por escribir su tabla. Tampoco una ficha del
Canal interno crea un acceso a denuncias. Código integrado no acredita SQL
instalado, publicación, proveedor operativo o recorrido de este módulo.

## Huecos y propietarios

| Requisitos | Resultado pendiente | Propietario y dependencia |
| --- | --- | --- |
| DIS1–DIS3 | Régimen/vínculo a fecha, decisión de iniciación y designaciones; abstención/recusación y sustitución. | B conserva hechos laborales; Jurídico valida régimen; Disciplina conserva procedimiento y permisos por designación. |
| DIS4/DIS5 | Prueba, cargos, defensa y medidas provisionales motivadas, con acceso por actuación. | Instructor/secretaría actúan según designación; Documentos custodia; órgano competente adopta medidas. |
| DIS6/DIS7 | Propuesta/decisión, firma, notificación, recurso/eficacia y cómputos distintos. | Órganos habilitados y servicios oficiales; ninguna inferencia desde incidencia o alerta. |
| DIS8 | Entregar inscripción o efecto mínimo, obtener aceptación del consumidor y conciliar. | Personal y Nómina mantienen sus efectos; receptor por puerto, sin cargos o prueba completos. |
| DIS9 | Cancelación, archivo y eliminación conforme a serie/acto. | Personal conserva la inscripción legal; Archivo/DPD y Documentos conservan/eliminan lo autorizado. |
| DIS10 | Operación, consulta y descarga nominal con todos sus resultados auditados. | Consumidor propio de auditoría K/L; el acceso general de RRHH no abre el expediente. |

Perfiles fijos provisionados por huella/CAS, un perfil activo, denegación por defecto
y autorización positiva por entidad, expediente, actuación, acción, finalidad,
campos y vigencia. Instructor,
secretario, resolutor, ejecutor y defensa tienen funciones separadas; un nombramiento
no suma roles. La persona expedientada/representante consulta solo las actuaciones
accesibles. El cliente no aporta identidad libre ni publica permisos en su petición.
La gestión interna reforzada y la defensa exterior, si se admite en Sede, mantienen
fronteras distintas. Soporte técnico no tiene acceso material general.

La auditoría común registra actor, perfil, recurso opaco, acción/finalidad, instante,
resultado `permitido/denegado/error`, correlación, proceso y canal para cada lectura,
descarga y operación. Los permitidos acompañan la lectura/efecto; intentos fallidos
usan la autoridad común después del cierre y sin fabricar actor en la frontera.
Efecto, consumo V3, versión, historia de solo adición, recibo y auditoría comparten
transacción; entregas con consumidor añaden outbox/reconciliación idempotentes.
Logs y eventos excluyen cargos, testimonios, diagnósticos y contenidos completos.

Disciplina consume puertos neutrales y referencias opacas; dominio/aplicación no
dependen de SQL/HTTP/proveedor ni acceden a tablas ajenas. Personal recibe la
inscripción autorizada; Nómina el efecto económico habilitado y su corrección.
Canal interno mantiene identidad protegida y custodia; Igualdad/PRL solo derivan
la actuación admitida por sus responsables. Bolsa solo podría recibir una
restricción mínima por contrato específico y decisión habilitada, nunca el expediente.

## Configuración y dudas

Régimen, tipificación, sanción, procedimiento, hitos, órganos, suplencias, cómputos,
plantillas y conservación se gobiernan con fuente/artículo, publicación, versión/
huella, colectivo, entidad, vigencia/efectos y órgano aprobador. Se fija la versión
al expediente; una aplicación posterior favorable conserva su acto y fundamento.
RRHH mantiene lo aprobado, sin crear faltas ni alterar máximos legales.

La duda 134 pide criterio por vínculo/organismo, texto consolidado aplicado,
órganos y separación, plantillas/custodia/conservación e interfaz para Personal/
Nómina. Sin respuesta pueden prepararse contratos y pruebas reservadas sintéticas;
sanción y cómputos operativos quedan pendientes. RAT, bases por finalidad, riesgos,
EIPD cuando proceda y serie deben estar aprobados antes de datos reales. Categorías
especiales y datos penales requieren habilitación propia; no toda disciplina es
un dato penal. Cancelar un antecedente impide reutilizarlo como vigente.

## Minitareas y salidas por PR

Dirección confirma SHA base actual y archivos exclusivos por corte. Rutas nuevas
son previsiones. B1: identidad/autorización/auditoría; B2: vínculo; B3: organización/
catálogo; B4: expediente; B5: Documentos/firma; B6: intercambio. Los cambios comunes
pertenecen a B/K/L y demás custodios, en PR separadas, fuera de estas horas.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| D00 · inventario | Revalidar capacidades, fuentes y deuda en SHA vigente. | Este plan, durante su turno. | Ficha y 134; no rehacer Personal/auditoría. | 1–2 |
| D01 · B3 | Catálogo por régimen y actuación con criterio validado; CLI informa regla ausente/conflicto. | Nuevos `data/catalogos/regimen-disciplinario/`, lector propio. | Jurídico/134; sin escala predeterminada ni máximos elegibles por usuario. | 4–6 |
| D02 · B1 | Consumidor nominal focal permite/deniega expediente asignado y audita los tres resultados. | Nuevos `regimendisciplinario/ports/{autorizacion,auditoria}.go`, consumidor propio. | K/L, perfiles/designaciones gobernados; montaje por custodio. | 6–10 |
| D03 · B2/B3 | Lectura de vínculo/situación a fecha por finalidad disciplinaria, con cobertura y falta de fuente explicadas. | `ports/personal.go`, consumidor/pruebas propios. | D02; B admite puerto mínimo; entidad y organización gobernadas. | 4–6 |
| D04 · B4, SQL borrador | Candidata de expediente/designaciones/actuaciones y adaptador con ensayo preparado. | SQL nuevo reservado y `adapters/postgres/` propios. | D01–D03; reserva fuera de Git, compartimento y preimagen; sin activar. | 5–8 |
| D05 · B4, ensayo | Clon principal con ACL/aislamiento/replay y kit revisado para dirección. | Pruebas de adaptador/kit propios. | D04, dos revisiones exactas; MCP local solo lectura; sin DOWN sobre historia. | 4–7 |
| D06 · DIS2/3 | Preparar expediente y designaciones sintéticas, recuperar mismo recibo y corregir por nuevo acto. | Dominio/caso de uso y HTTP/PG propios. | D02–D05; prueba de recusación/sustitución; sin incoación oficial atribuida. | 6–10 |
| D07 · DIS4/10 | Lista reservada y detalle por expediente/actuación; defensa solo ve lo accesible. | Cliente/vista propios, catálogos por idioma. | D06; montaje/rutas por custodio; revisión usabilidad y frontera. | 4–6 |
| D08 · B5 | Aportar prueba y descargar original autorizado ligado a actuación; negar terceros. | `ports/documentos.go`, consumidor y controles propios. | D02/D06; custodia segregada Documentos, límites de adjuntos. | 4–7 |
| D09 · DIS4 | Cargos, admisión/denegación motivada de prueba, audiencia y alegación versionadas. | Casos de uso/controles de instrucción y defensa propios. | D07/D08; designación y circuito de 134. Sin valoración automática. | 5–8 |
| D10 · DIS5 | Registrar medida provisional y revisión con fundamento/límite propios. | Caso de uso/vista propios de medidas. | D01/D09; órgano habilitado; efecto económico espera consumidor y acto. | 4–6 |
| D11 · DIS7 | Consulta de tres cómputos y alertas revisables desde hitos comprobados. | Motor de cómputo propio y catálogo versionado. | D01/D09; criterio jurídico aplicado, suspensión/interrupción con evidencia. | 4–7 |
| D12 · DIS6, propuesta | Preparar propuesta y documento ligados a prueba/defensa cerradas. | Caso de uso y consumidor documental propios. | D09; plantilla vigente; no convertir propuesta en sanción. | 4–7 |
| D13 · DIS6, acto | Resolver por actor distinto, vincular firma y asiento/notificación acreditados. | Consumidores de decisión/firma/Registro/notificación propios. | D12; órganos y servicios oficiales disponibles; cada evidencia separada. | 5–8 |
| D14 · DIS6, eficacia | Registrar recurso y cambios de ejecutividad/firmeza con acto y versión. | Caso de uso/historia propios de eficacia. | D13; circuito jurídico aprobado; no inferir firmeza de fecha sola. | 4–6 |
| D15 · DIS8 | Entrega mínima a Personal/Nómina con recibo, reintento y corrección conciliados. | `ports/efectos.go`, entrega/reconciliación propias. | D13/D14; acto habilitante y receptores de sus dueños; sin datos probatorios. | 4–7 |
| D16 · DIS9 | Cancelación/archivo/expurgo y prueba de restauración sin reintroducir datos suprimidos. | Consumidor Archivo/Documentos/Personal y pruebas propios. | Serie/acto aprobados; D15; historia mínima compatible con eliminación. | 4–6 |
| D17 · entrega | Chrome→permiso→PG→recibo y reinicio; manual explica límites y quién decide. | Pruebas focales/recorrido y manual propios. | Capacidades instaladas, revisiones sensibles y usabilidad. | 4–6 |

D06–D09 son el primer recorrido reservado sintético. D10/D11 se desarrollan por
archivos distintos cuando exista criterio jurídico. D12–D15 esperan órganos y
servicios acreditados; ninguna dependencia se reemplaza por memoria, aviso o tabla
ajena. Cada ampliación de persistencia reserva su número nuevo, se mantiene en
borrador hasta ensayo en clon principal y recibe dos revisiones antes de instalar.
Dirección instala y conserva historia; A no ejecuta SQL ni servicios en cidonia.

## Dependencias, paralelismo y estimación

Las 18 filas suman **76–123 horas técnicas**, con revisión y comprobaciones focales:
**10–16 jornadas de un equipo** de ocho horas, redondeadas al día completo. Incluyen
SQL propio y consumidores del módulo; excluyen producción común de B/K/L y servicios
externos. Las horas se reestiman al fijar 134 y los contratos de ejecución.

Con dos equipos se estiman **9–13 jornadas**, con órganos/fuentes disponibles.
Camino orientativo: D00/D01 5–8 h; D02/D03 10–16 h; D04/D05 9–15 h;
D06–D09 16–25 h con UI/documentos en paralelo; D10–D16 21–34 h: propuesta,
acto, eficacia, entrega y archivo en secuencia, mientras el otro equipo prepara
medidas/cómputos; D17 4–6 h. Suma **65–104 h**, redondeadas a 9–13 jornadas
de calendario técnico. Un equipo conserva expediente/instrucción
y otro documentos/cómputos/entregas; cada contrato, catálogo y vista tiene un solo
escritor. Iniciación, decisión y entrega limitan el solape. Sin esas condiciones
se utiliza la horquilla de un equipo.

| Trabajo externo, fuera del total técnico | Dedicación orientativa | Condición |
| --- | ---: | --- |
| Jurídico/RRHH/órganos competentes | 8–16 h efectivas | Criterio de 134 por vínculo, designaciones, plantillas, cómputos y circuito; espera sin plazo comprometido. |
| DPD/Archivo | 5–9 h efectivas | RAT, accesos de defensa/terceros, serie, conservación, cancelación y copias. |
| Sistemas/firma/Registro/notificación | 6–12 h efectivas | Entorno sintético y servicios oficiales con contratos/evidencias. |
| Personal B, K/L, Cronos, Documentos y Nómina | Estimación por sus propietarios | Puertos mínimos, consumidor económico y composición; no están incluidos en D02/D03/D15. |

La duración de estas decisiones externas no es una promesa de publicación.
Contratos y pruebas independientes avanzan mientras se completa su habilitación.

## Comprobación y entrega

La comprobación documental cubre enlaces locales, rutas, sumas y `git diff --check`.
No requiere Go, SQL, contenedores, servicios o navegador. La PR no declara el
módulo instalado, recorrido ni aprobado para datos reales.

Cada implementación aplicará `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec` y `probar-recorridos-vec` según el cambio; antes de pantallas,
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e
`impeccable`/`VEC-PRIORIDAD.md`, con revisión independiente de usabilidad. Textos
con `humanizer`/`VEC-USO.md`; SQL con `revisar-sql-vec`/`ensayar-sql`, reserva y
ensayo; seguridad focal `security-audit` y Semgrep local sin enviar código fuera.
Go usa gopls; búsqueda primero por índice. `revisar-cambios-vec`,
`documentar-entregar-vec` y `pr-vec` cierran la entrega; dirección integra/despliega.

Validar revocación concurrente, recusación, actor que intenta instruir y resolver,
persona/entidad ajenas, defensa con documento protegido de terceros, descarga sin
permiso y fuente caída. Cada lectura/descarga/efecto debe tener traza común de
permitido/denegado/error. Efectos requieren PG real, primera escritura, replay,
conflicto y recuperación tras reinicio con misma historia/recibo. Comprobar
ES/EN desde catálogos, teclado/foco y Chrome del sistema PC/móvil. La ejecución
de un efecto externo solo se afirma con aceptación de su propietario.
