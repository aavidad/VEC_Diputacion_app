# Plan de Relaciones sindicales — 4 de octubre de 2026

**Aparcado hasta cerrar Bolsa y CT (orden de Alberto, 07/10/2026).** Los objetivos vigentes están en [OBJETIVOS.md](OBJETIVOS.md).

VEC gestionará representación acreditada, mandatos, órganos, crédito horario,
cesiones, dispensas y consumo; después incorporará mesas, actas, acuerdos y la
proyección pública revisada. El descuento de cuota tendrá custodia y permisos
segregados. La primera entrega propuesta registra un uso sintético de crédito y
recupera saldo/recibo; no ejecuta descuentos ni publica datos personales. Este
encargo entrega documentación, sin código, SQL o instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos: [ficha de Relaciones sindicales integrada](../estudio_requisitos/ficha_relaciones_sindicales_2026-10-04.md),
SIN1–SIN10. Se mantiene la cola vigente y la propiedad de Personal B,
identidad/autorización K, núcleo/auditoría L y los servicios comunes.

## Fuentes y decisiones aplicables

La ficha recoge [LOLS](https://www.boe.es/eli/es/lo/1985/08/02/11/con),
[TREBEP, arts. 31–46](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719),
[Estatuto de los Trabajadores, arts. 61–81](https://www.boe.es/eli/es/rdlg/2015/10/23/2/con),
[RD 1844/1994](https://www.boe.es/buscar/act.php?id=BOE-A-1994-20236),
[RD 1846/1994](https://www.boe.es/buscar/act.php?id=BOE-A-1994-20237),
[acuerdo provincial de 2020](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/Acuerdo-regulacion-creditos-horarios-sindicales.pdf)
y [Ley andaluza 1/2014](https://www.boe.es/eli/es-an/l/2014/06/24/1/con).
El [portal de representantes](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/integrantes-organos-de-representacion/)
acredita información pública, sin saldos operativos ni interfaz técnica admitida.

Las cantidades y liberaciones de 2020 estaban ligadas a su mandato; no se heredan
como reglas actuales. La ficha no acredita un acuerdo postelectoral sustitutorio.
Representación unitaria, sindical y delegación de prevención conservan fundamentos
y facultades separados; tampoco ser representante acredita legitimación para toda
mesa. No se replica un órgano AGE como autoridad provincial.

Aplican [materias reservadas, §§11 y 12](../estudio_requisitos/materias_reservadas_economicas_y_relaciones_laborales.md),
[arquitectura](../portal_vec/arquitectura_tecnica.md),
[contrato de módulos](../portal_vec/contrato_modulos_vec.md),
[roles y ámbitos](../portal_vec/matriz_roles_y_ambitos.md) y
[cumplimiento](../portal_vec/cumplimiento_y_seguridad.md).
La duda 135 de [dudas.md](../../dudas.md) pide carga y gobierno del mandato
actual, circuito interno e integraciones autorizadas. No se modifica esa pregunta
ni se vuelven a solicitar las cifras históricas publicadas.

## Inventario en la base inspeccionada

La búsqueda empezó por `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Sus resultados se cotejaron mediante `git ls-tree` y lectura en el SHA indicado;
el índice por sí solo no representa la base actual.

| Pieza | Rutas actuales | Capacidad y límite |
| --- | --- | --- |
| Relaciones sindicales | `docs/estudio_requisitos/ficha_relaciones_sindicales_2026-10-04.md` | Ficha integrada; no hay paquete de representación/crédito sindical en `internal/modules/`, libro durable de crédito ni vista del módulo. |
| Personal | `internal/modules/personal/ports/{relacion_empleado,historia_relaciones_propia,lector_relacion_rpt}.go`; `application/consulta_relacion_empleado.go` | Vínculo propio para Dietas y contratos históricos/RPT. No habilitan consumo sindical ni consulta de otros representantes; B debe admitir el puerto mínimo de vínculo/jornada a fecha. |
| Contexto y perfiles | `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Autoridades comunes en código. Falta catálogo/proveedor nominal para mandato/órgano/mesa; afiliación y siglas no conceden acceso. |
| Cronos | `internal/modules/cronos/ports/{consulta_saldo,solicitud_permiso,resolucion_permiso}.go` | Saldo y permisos de Cronos con sus acciones y circuitos. No constituyen el libro de crédito sindical ni acreditan su importación; consumidor de ausencia mínima pendiente por contrato. |
| Documentos/firma | `internal/vec/documentos/application/servicio.go`; `ports/{contratos,firma}.go`; `adapters/validadorautofirma/cliente.go` | Custodia, lectura y verificación GrxFirma v2 comunes. Falta vincular acreditaciones/actas al mandato/mesa y separar reservado/publicable; código integrado no acredita firma o ratificación de acuerdos. |
| Usuarios | `internal/modules/usuarios/application/{correos,correo_avisos}.go`; `ports/correo_avisos.go` | Contacto bajo autoridad común; el aviso implementado está limitado a llamamiento de candidato. No permite circular datos sindicales ni obtener contactos generales de representantes. |
| Auditoría | `internal/vec/ports/{auditoria_intento_nominal,auditoria_frontera_ruta_exacta}.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Destinos comunes para intentos y frontera. Falta el consumo nominal del módulo para permitido/denegado/error, también consultas y descargas. |
| Registro/catálogos | `internal/vec/domain/types.go`; `internal/vec/application/service.go`; `internal/vec/ports/ports.go`; `web/static/portal-empleado/portal-catalogo-modulos.js`; `data/catalogos/` | Registro y navegación comunes; no instalan handlers, permisos, SQL ni reglas sindicales. Manifiestos/montaje permanecen con sus custodios. |

No hay módulo Nóminas en `internal/modules/` en esta base ni receptor de descuento
sindical acreditado. El plan define esa dependencia, sin escribir tablas externas.
Una fuente pública no acredita acceso nominal a censos o cuotas. Este inventario
distingue Git de instalación, publicación, configuración y recorrido verificado;
no da por disponible ningún recorrido sindical.

## Huecos y propietarios

| Requisitos | Resultado pendiente | Propietario y dependencia |
| --- | --- | --- |
| SIN1/SIN2 | Mandatos, órganos, unidad electoral, censo aplicable y acreditación por versión. | Relaciones sindicales conserva hechos; Personal B aporta vínculo; órgano competente valida acreditaciones y composición. |
| SIN3–SIN6 | Regla vigente, crédito, cesión/dispensa, consumo y corrección sin doble gasto. | Libro propio de Relaciones sindicales; regla aprobada por mandato/colectivo. Cronos conserva la ausencia mínima recibida. |
| SIN7 | Convocatoria, asistencia, acta y acuerdo con firma/ratificación/eficacia separadas. | Secretaría y órgano habilitado de cada mesa; Documentos/firma custodian y verifican. |
| SIN8 | Mandato específico de descuento, revocación y resultado conciliado. | Custodio segregado de mandatos y Nómina; nunca se utiliza el perfil de créditos para cuotas. |
| SIN9 | Proyección institucional revisada y retirada por versión. | Transparencia valida representación y dispensas publicables; excluye afiliación, consumos y cuotas. |
| SIN10 | Lecturas, cambios, descargas y comunicaciones auditadas en destino común. | Consumidor propio con K/L; sin acceso técnico general al contenido. |

Perfiles fijos por huella/CAS y uno activo por operación: representante, gestor
acreditado, gestor RRHH de representación, revisor/órgano habilitado, secretaría,
custodio de descuentos y auditor/DPD/transparencia. La concesión positiva enumera
entidad, órgano, mandato/mesa, recurso, acción, finalidad, vigencia y campos.
No se concede permiso al recibir la petición ni se deriva de la afiliación.
Consulta y descarga/exportación son acciones distintas. La jefatura solo ve la
ausencia autorizada y su intervalo; no destino sindical ni justificación reservada.

Auditoría común por actor nominal/perfil, recurso opaco, acción/finalidad, instante,
resultado `permitido/denegado/error`, correlación, proceso y canal, también por
consulta/descarga. Permitidos se confirman con la lectura/efecto; fallidos usan el
registrador común tras cierre del intento, sin identidad inventada en frontera.
Autorización consumida, versión, efecto, recibo, historia de solo adición y auditoría
comparten transacción. Correcciones añaden asiento enlazado. Entregas concretas
con consumidor incorporan outbox/acuse/reconciliación, sin duplicar el crédito.

Dominio/aplicación no dependen de SQL/HTTP/proveedor. Puertos y eventos mínimos,
versionados e idempotentes; sin acceso a tablas ajenas. Cronos recibe intervalo,
duración y autorización, Personal la situación que deba reflejar y Nómina solo
mandato vigente, destinatario e importe necesarios bajo acceso segregado. Bolsa
no recibe afiliación, cuota o actividad. Los contactos se resuelven en Usuarios
por consumo autorizado; no se crea otro directorio.

## Configuración y dudas

Mandato, unidades, órganos, censo aplicable, crédito, cesión, dispensa, preaviso,
urgencia, plantillas y destinatarios conservan fuente/artículo, publicación,
versión/huella, ámbito/colectivo, vigencia/efectos y órgano aprobador. Preparar y
aprobar reglas requiere permisos diferentes; ningún cambio de mandato hereda
automáticamente cantidades antiguas. Cada asiento conserva la regla aplicada.

La duda 135 concreta responsables/carga actual, comunicaciones/correcciones/actas,
integraciones Cronos/Nómina y custodios segregados. Las pruebas usan acreditaciones
sintéticas y regla provisional rotulada mientras se valida ese circuito. Su uso
real espera fuente y mandato vigentes. No se pide censo de afiliación para calcular
crédito ni motivo narrativo de la actividad. RAT, información, base por finalidad,
condición de categorías especiales y conservación se documentan antes de datos
reales. La conformidad para un descuento no autoriza circulación general de la cuota.

## Minitareas y salidas por PR

Dirección confirma SHA actualizado, dueño y rutas exclusivas. Los nombres nuevos
son previsiones. B1: autorización/auditoría; B2: vínculo; B3: organización/catálogos;
B4: libro durable; B5: documentos/firma; B6: entregas/proyección. Los propietarios
producen cambios comunes aparte, sin incluir su esfuerzo en esta estimación.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| S00 · inventario | Revalidar base, mandatos/fuentes y deuda concreta. | Este plan, en su turno. | Ficha y 135; no rehacer Cronos/Personal. | 1–2 |
| S01 · B3 | Catálogo de órganos/reglas con consumidor CLI y falta de vigencia explicada. | Nuevos `data/catalogos/relaciones-sindicales/`, lector propio. | Fuente y aprobación por mandato; sin constantes de 2020. | 4–6 |
| S02 · B1 | Consumidor nominal focal por mandato/recurso, con auditoría de los tres resultados. | Nuevos `relacionessindicales/ports/{autorizacion,auditoria}.go`, consumidor. | K/L y perfiles fijos; catálogo provisionado antes de petición, montaje por dueño. | 6–10 |
| S03 · B2/B3 | Consultar vínculo/jornada a fecha, organización/entidad y cobertura. | `ports/personal.go`, consumidor/pruebas propios. | S02; contrato mínimo admitido por B; no usar audiencia Dietas. | 3–5 |
| S04 · B4, SQL borrador | Candidata de mandato/libro de crédito y adaptador; ensayo preparado. | SQL reservado y `adapters/postgres/` propios. | S01–S03, reserva fuera de Git, preimagen y compartimento; sin activar. | 4–7 |
| S05 · B4, ensayo | Clon principal verifica consumo/replay/ACL, kit revisado para dirección. | Pruebas de adaptador/kit propios. | S04; dos revisiones exactas; MCP local solo lectura. | 4–6 |
| S06 · SIN1/3 | Cargar acreditación sintética, aplicar regla rotulada y consultar crédito inicial/recibo. | Dominio/caso de uso/HTTP propios de mandato/crédito. | S01–S05; sin datos reales ni descuentos. | 5–8 |
| S07 · SIN1/6 | Lista y detalle propios del mandato/crédito, saldo y límites visibles. | Cliente/vista y catálogos por idioma propios. | S06; registro/rutas/montaje por custodio; revisión usabilidad. | 4–6 |
| S08 · SIN4 | Comunicar/validar cesión o acumulación con recibo y rechazo de doble consumo. | Caso de uso/vista propios de cesiones. | S06; reglas y ámbitos autorizados, fechas dentro del mandato. | 4–7 |
| S09 · SIN5 | Tramitar dispensa/preaviso/urgencia y corregir con asiento nuevo enlazado. | Caso de uso/vista propios de dispensas. | S08; circuito de 135, regla vigente; no permisos por cargo. | 4–7 |
| S10 · SIN6 | Registrar comunicación de uso, saldo reproducible y corrección recuperables tras reinicio. | Consumo/saldo y pruebas propias. | S08/S09; concurrencia y reuniones administrativas según regla aplicable. | 4–7 |
| S11 · B6 | Entregar ausencia mínima a Cronos, obtener acuse y conciliar reintentos/correcciones. | `ports/ausencias.go`, entrega/reconciliación propias. | S10; receptor Cronos de su dueño y jefatura solo recibe intervalo autorizado. | 4–6 |
| S12 · SIN2/7 | Composición de mesa, convocatoria y asistentes acreditados con acceso por mesa. | Casos de uso/cliente propios de mesas. | S06/S08, mandato vigente y legitimación admitida; contacto por Usuarios si procede. | 4–7 |
| S13 · B5/SIN7 | Aportar/descargar acta original; distinguir firma, ratificación y publicación. | `ports/documentos.go`, consumidor documental/controles propios. | S12; Documentos/firma y órgano competente; permiso específico de descarga. | 4–6 |
| S14 · SIN8 | Alta/revocación de mandato de descuento en compartimento segregado y consulta por custodio. | Casos de uso/consumidor reservado de mandatos; SQL nuevo si necesario. | S02/S03/S08; habilitación específica, condición art.9 y custodia aprobadas; nuevo SQL ensayado. | 5–8 |
| S15 · SIN8/B6 | Enviar mandato mínimo a Nómina con recibo/resultado y corrección conciliados. | `ports/descuentos.go`, entrega/reconciliación propias. | S14; receptor Nómina y contrato/servicio autorizados; sin afiliar ni publicar. | 4–7 |
| S16 · SIN9 | Revisar/publicar o retirar proyección institucional separada del libro privado. | Proyección y adaptador Transparencia propios. | S06/S13; aprobación vigente, campos publicables y destinatario admitido. | 3–5 |
| S17 · conservación | Archivo/expurgo por series distintas con evidencia mínima y restauración ensayada. | Consumidores Archivo/Documentos y pruebas propios. | S11/S13/S15/S16; series y custodios aprobados; sin cuotas o actas en logs. | 4–6 |
| S18 · entrega | Chrome→permiso→PG→recibo y reinicio; manual distingue saldo, ausencia y descuento. | Pruebas focales/recorrido y manual propios. | Capacidades instaladas, revisiones sensibles/usabilidad. | 4–6 |

S06–S10 aportan el primer ejercicio de crédito. La ausencia en Cronos requiere
S11 y su receptor real; un envío pendiente no se presenta como aplicado. S14/S15
se abren solo si se autoriza gestión del descuento, conservando como pendiente ese
requisito si no se admite. Ninguna PR crea una ruta vacía o un segundo libro en
Cronos. SQL nuevo se reserva antes de escribir, permanece en borrador hasta ensayo
en clon principal y dos revisiones, y lo instala dirección; no se reaplica historia
ni se ejecuta DOWN o SQL en cidonia por A.

## Dependencias, paralelismo y estimación

Las 19 filas suman **75–122 horas técnicas**, con revisión/comprobaciones focales:
**10–16 jornadas de un equipo** de ocho horas, redondeadas al día completo.
Incluyen la gestión segregada de mandatos propuesta. Si se excluye SIN8, retirar
S14/S15 y ajustar las dependencias de S17 reduce el total a **66–107 h**, sin
presentar el descuento como cubierto. El trabajo de los dueños comunes va aparte.

Con dos equipos y todas las dependencias disponibles se estiman **8–13 jornadas**.
Camino orientativo: S00/S01 5–8 h; S02/S03 9–15 h; S04/S05 8–13 h;
S06–S10 17–29 h con S07 separado de cesión/dispensa/consumo; S11–S17 17–27 h:
un equipo conserva ausencia/mandato/descuento y otro mesas/acta/proyección, antes
del archivo; S18 4–6 h. Suma **60–98 h** de calendario técnico, redondeadas a
8–13 jornadas. Las vistas y catálogos se separan por archivo antes del solape;
un escritor conserva libro de crédito y cada contrato común. Fuentes, aprobación,
instalación y aceptación por Cronos/Nómina limitan el paralelismo. Sin solape se
utiliza la horquilla de un equipo.

| Trabajo externo, fuera del total técnico | Dedicación orientativa | Condición |
| --- | ---: | --- |
| RRHH/órganos y representación acreditada | 6–12 h efectivas | Mandato/carga y reglas actuales, circuito de 135, mesas y validación; espera sin plazo comprometido. |
| DPD/Archivo/Transparencia | 5–9 h efectivas | Categoría especial, custodia segregada, series, publicación y copias. |
| Sistemas/firma/servicios corporativos | 5–10 h efectivas | Contratos, entorno sintético y evidencias; no inferir API desde página pública. |
| Personal B, K/L, Cronos, Usuarios, Documentos y Nómina | Estimación por sus dueños | Lectores/receptores y composición común; sus desarrollos no están incluidos en S02/S03/S11/S15. |

Las horas son esfuerzo, no fechas de publicación o aprobación. Se revisan al
cerrar la carga del mandato y los contratos de sus consumidores.

## Comprobación y entrega

El cambio documental comprueba enlaces locales, inventario, sumas y
`git diff --check`; no necesita Go, SQL, contenedores, servicios o navegador.
Su PR no acredita instalación, recorrido ni autorización para datos reales.

Implementación con `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec`, `probar-recorridos-vec` según el corte. Antes de pantallas:
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec`,
`impeccable`/`VEC-PRIORIDAD.md`, revisión independiente de usabilidad. Textos con
`humanizer`/`VEC-USO.md`; SQL con `revisar-sql-vec`/`ensayar-sql` y revisiones
exactas; `security-audit` focal y Semgrep local sin subir código. Go usa gopls;
búsqueda primero por índice. Entregar con `revisar-cambios-vec`,
`documentar-entregar-vec`, `pr-vec`; dirección integra/despliega.

Validar dos consumos concurrentes, cesión fuera de mandato/entidad, regla caducada,
actor revocado, consulta/descarga ajena y fallo de fuente. Comprobar auditoría común
de permitidos/denegados/errores sin cuota, afiliación o actividad narrativa. La
jefatura y Cronos solo reciben ausencia mínima; Nómina solo el mandato necesario.
Cada efecto necesita PG real, replay/conflicto y reinicio con mismo recibo/saldo/
historia. Chrome del sistema en PC/móvil, teclado/foco e i18n desde catálogos;
descarga de acta reservada y proyección pública se prueban con permisos diferentes.
