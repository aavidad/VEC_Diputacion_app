# Plan de Acción social: 4 de octubre de 2026

La persona consultará una ayuda aplicable, presentará su solicitud y recuperará el
justificante; la unidad competente comprobará requisitos y gestionará subsanaciones.
Propuesta, comisión, resolución, pago y reintegro conservarán actos y responsables
separados. Anticipos y pensiones tendrán circuitos posteriores propios. Este encargo
entrega documentación, sin código, SQL ni instalación.

Base inspeccionada: `origin/main@009472bd760e76cb2951433236711262f10648be`.
Requisitos principales: [ficha de Acción social](../estudio_requisitos/ficha_accion_social_2026-10-04.md),
AS1–AS12 y ASO-001–ASO-003 del [catálogo](../estudio_requisitos/catalogo_funcional_rrhh_y_hoja_ruta.md#8-acción-social-prl-igualdad-y-relaciones-laborales).
Personal B mantiene relaciones y servicios; K identidad/perfiles; L las autoridades
comunes. El plan no toma sus archivos ni sustituye el seguimiento general.

## Fuentes y decisiones aplicables

La ficha identifica el Reglamento provincial y modificaciones de 2015, 2018 y 2021.
El BOP de 09/06/2021 contiene el acto y un consolidado informativo; se conserva la
primacía de los actos auténticos. Los anuncios de septiembre de 2026 de CEMCI y
Turismo pertenecen a entidades distintas y no acreditan una convocatoria general
ni cuantías aplicables a otra entidad. Antes de activar una modalidad se cotejarán
bases completas, modificaciones, colectivo, plazo y acto competente.

El primer corte propuesto es educación superior, art. 5, en una entidad y ejercicio
cuya convocatoria vigente esté verificada. TREBEP 37/38 y Ley 39/2015 se estudian
según ámbito; las remisiones históricas a Ley 30/1992 no se trasladan al producto.
No se fijan importes desde ejemplos ni se considera aprobado un circuito interno.

Se conserva [dudas.md, 130](../../dudas.md), sobre modalidad/entidad inicial,
responsables y circuito. Desarrolla la 39 y se coordina con la 84 sobre conservación y la 26
sobre efecto económico, sin copiar reglas de Dietas. Las normas publicadas se
consultan en sus fuentes; no se solicita otra validación de su texto. Registro,
notificación, custodia y pago necesitan interfaces y evidencias separadas.

## Inventario en la base inspeccionada

La búsqueda empezó en `codebase-memory-mcp`, proyecto
`home-alberto-Trabajo-VEC_Diputacion_app-.worktrees-codexe-original-autorizacion-20261003`.
Se cotejaron los resultados con `git ls-tree` y lectura de archivos en esta base.
El índice anterior no es prueba suficiente de actualidad.

| Pieza | Rutas actuales | Reutilización y límite |
| --- | --- | --- |
| Personal B | `internal/modules/personal/ports/{ficha_propia,historia_relaciones_propia,relacion_empleado}.go`; `adapters/composicion/FICHA_PROPIA.md`; `adapters/postgres/ficha_propia.go` | Relación y consulta propia protegidas. B entregará entidad, colectivo y fechas/servicios exigidos por bases. No crear maestro paralelo de empleados/familiares ni leer su SQL. |
| Usuarios / K | `internal/modules/usuarios/application/correos.go`; `internal/vec/ports/contexto_actor.go`; `internal/vec/adapters/contextoactor/postgres/`; `internal/vec/adapters/administracionperfiles/postgres/` | Identidad/contacto y contexto comunes. No prueban requisitos de ayuda ni representación. K conserva perfiles fijos y emisor; falta consumidor y concesiones exactas de Acción social. |
| Auditoría L | `internal/vec/ports/auditoria_intento_nominal.go`; `auditoria_frontera_ruta_exacta.go`; `internal/vec/adapters/postgres/auditoria_intento_nominal.go` | Auditoría nominal de denegación/error con acuse y frontera técnica separada. Se reutiliza; faltan proceso/canal, motivos y rutas propias acordados con L. |
| Documentos / firma | `internal/vec/documentos/application/servicio.go`; `ports/{firma,custodia_firmado}.go`; `adapters/postgres/custodia_firmado.go`; `internal/vec/ports/documentos.go` | Custodia, referencias externas, lectura y verificación comunes. Las piezas específicas de CT no emiten resoluciones de ayuda. Registro de documento o preparación de notificación no acreditan asiento, firma, entrega o plazo. |
| Nóminas | `web/static/portal-empleado/modulos/nominas/vista.js`; `internal/modules/personal/domain/payroll.go`; [plan de Nóminas](https://github.com/aavidad/VEC_Diputacion_app/blob/5c2f146bd15761d225cda7297b4fc7b8f34034a2/docs/plan_modulos/nominas.md) | Vista inyectable sin conector y sumador preliminar de conceptos. Ninguna pieza recibe una ayuda aprobada ni acredita pago. No usar su ejemplo como pagador. |
| Dietas | `internal/modules/dietas/ports/{circuito_comision,efecto_autorizacion}.go`; `application/preparacionliquidacion/` | Antecedentes de contratos/efectos económicos por su dueño. Acción social mantiene sus requisitos, comisión, presupuesto y decisión; no reutiliza una política de comisión de servicio como norma de ayudas. |
| Igualdad y agregación | [ficha de Igualdad](../estudio_requisitos/ficha_igualdad_2026-10-04.md); `web/static/textos/{es,en}/datos-personales.json` | La ficha define indicadores mínimos; las claves existentes no acreditan receptor o estadística de ayudas. No enviar beneficiarios/familia nominales al diagnóstico general. |

No hay backend `internal/modules/accion_social/` ni pantalla propia de Acción social
en esta base. Son propuestas de ubicación, no contratos registrados. Las piezas
comunes integradas no acreditan instalación, fuente de ayudas conectada o recorrido.
Este plan no reserva ni reaplica migraciones existentes.

## Huecos y propietarios

| Capacidad | Resultado pendiente | Dueño y dependencia |
| --- | --- | --- |
| AS1–AS2 | Catálogo aplicable por entidad/ejercicio y vínculo histórico acreditado. | A conserva Acción social; RRHH aporta acto; B relación/servicios; K identidad. |
| AS3–AS5 | Solicitud mínima, registro/justificante, subsanación y reclamación. | A mantiene expediente; L/Documentos/Registro custodia y asiento; unidad competente tramita. |
| AS6–AS7 | Propuesta, comisión, resolución y control presupuestario distintos. | A conserva decisiones del módulo; comisión/órgano/contabilidad mantienen competencia y fuente. |
| AS8–AS9 | Efecto autorizado y conciliado, rectificación/reintegro enlazados. | A emite hecho mínimo; Nóminas/Tesorería confirma aceptación/pago por su contrato. |
| AS10–AS11 | Consulta/descarga independientes y compartimento reservado auditado. | A consume K/L; DPD/Archivo acuerdan datos y conservación. |
| AS12 | Anticipo y pensiones con normas, destinatarios y saldos propios. | A prepara ampliación; Nóminas/Tesorería/gestora mantienen descuento, pago y aportación. |

Perfiles propuestos de solicitante, tramitación, comisión, órgano resolutor, receptor
económico y auditoría se provisionan en K con huella/CAS, un perfil activo fijo y
acción/recurso/entidad/modalidad/ejercicio/finalidad/campos/vigencia exactos. Jefatura,
soporte, administración funcional y candidato no acceden a anexos reservados por cargo.
Representación exige poder acreditado vigente. Canales internos de gestión y
proyección pública de bases quedan separados del autoservicio propio; exposición
exterior del empleado necesita política expresa.

Auditoría común nominal para cada catálogo restringido, consulta, descarga,
solicitud, subsanación, decisión y efecto: actor acreditado, perfil, acción, recurso
opaco, finalidad, instante, resultado permitido/denegado/error, proceso/canal,
correlación y versión. Lectura y auditoría se confirman antes de entregar datos;
efecto, consumo de permiso, estado, historia y auditoría en la misma transacción.
Denegaciones/errores se registran tras cerrar el intento original mediante el
contrato común; sin identidad, frontera técnica sin actor inventado. Fallo del
registro impide revelar datos. Salud, familia, ingresos y documentos completos
quedan fuera de auditoría/logs y del receptor económico.

## Minitareas y salidas por PR

Dirección confirma SHA base, propietario y archivos antes de cada corte. Rutas
nuevas son previsión; montaje/manifiestos y contratos K/L/B esperan turno de sus
custodios. Cada puerto se entrega con consumidor útil, sin otra maqueta sintética.

| Corte y base | Salida usable por PR | Archivos propios previstos | Dependencias | Horas |
| --- | --- | --- | --- | ---: |
| A00 · inventario | Confirmar deuda y piezas vigentes. | Este plan durante su turno. | SHA/FIN vigentes. | 1–2 |
| A01 · fuente | Una modalidad, bases auténticas, campos mínimos, unidad/circuito y cobertura. | Contrato documentado del módulo. | Duda130; acto/convocatoria y RAT; RRHH/Sistemas. | 3–5 |
| A02 · B1 | Consumidor nominal y auditoría común que permite/deniega una operación exacta. | Nuevos `accion_social/ports/{autorizacion,auditoria}.go`; consumidor. | A01; ABI K/L; perfiles fijos positivos por huella/CAS. | 6–10 |
| A03 · B2/B3 | Correspondencia relación/entidad/colectivo/servicios sin copiar ficha integral. | Puerto y consumidor de Personal del módulo. | B produce vínculo/fechas; A02 antes de lectura. Acuerdo DTO independiente. | 3–5 |
| A04 · AS1 | Catálogo y detalle de bases con puerto, fuente admitida y consumidor focal. | Dominio/aplicación/puerto/adaptador propios. | A01; políticas versionadas; A02 si acceso restringido. | 4–6 |
| A05 · AS1/AS10 | Vista utilizable de bases/plazo, vacíos/error y estado de conexión ES/EN. | Entrada/cliente/vista propios y catálogos ES/EN. | A04; montaje por custodio; revisión de usabilidad. | 4–6 |
| A06 · AS3 | Preparar solicitud mínima con evidencia exigida y ámbito resuelto en servidor. | Caso de uso/formulario y consumidor Documentos propios. | A02/A03/A05; campos y custodia aprobados. | 5–8 |
| A07 · SQL necesario | Borrador reservado de expediente/versiones e idempotencia de presentación. | SQL/adaptador propios. | A01–A06; sin outbox para mero borrador. | 4–6 |
| A08 · ensayo | Ensayo clon principal, revisiones exactas y kit de instalación por dirección. | Pruebas y kit propios. | A07; preimagen, reserva y orden causal. | 4–8 |
| A09 · AS4 | Presentación y recuperación del asiento/justificante, diferenciando preparación local. | Puerto de registro, consumidor y recibo propios. | A06/A08; Registro admite interfaz; auditoría junto al efecto. | 5–8 |
| A10 · AS5 | Subsanación y reclamación propias con historia/fechas de política. | Aplicación/formulario/cliente propios. | A09; circuito/plazos aprobados y comunicación admitida. | 5–8 |
| A11 · AS10 | Consulta propia y descarga independiente de evidencia minimizada. | Casos de uso/consumidor documental propios. | A02/A09/A10; Documentos y política de anexos. | 4–6 |
| A12 · recorrido inicial | Solicitante/unidad→justificante/subsanación, negativos, reinicio y manual. | Pruebas focales y manual del módulo. | A05–A11; fuente montada y revisiones pertinentes. | 4–6 |
| A13 · AS6/AS7 | Propuesta e informe presupuestario, sin conceder por calcular importe. | Aplicación y vista propias; puerto de presupuesto con consumidor. | Ampliación encargada; fuente/topes/prelación/competencia aprobados. | 6–10 |
| A14 · AS6 | Comisión/acuerdo y resolución distintos con original, firma/registro cuando procedan. | Casos de uso y consumidores documentales propios. | A13; mandato/recusación/suplencia y órgano; L/Registro/firma. | 6–10 |
| A15 · AS8 | Entregar efecto económico mínimo y recuperar acuse; conciliar pago comunicado. | Puerto/consumidor económico y estado propio. | A14; Nóminas/Tesorería admite contrato; idempotencia y estado durable. | 6–10 |
| A16 · AS9 | Rectificación/reintegro enlazados y entrega idempotente del efecto posterior. | Casos de uso/historia y consumidor propios. | A14/A15; acto competente, sin reescribir concesión. | 5–8 |
| A17 · AS12 posterior | Inventario/contratos de anticipo y pensiones con consumidor preparatorio. | Contratos/plan de ampliación propios. | Reglamento/gestora/circuito y encargos específicos; no crear saldo ficticio. | 6–10 |
| A18 · entrega ampliada | Recorrido decisión→efecto→estado/rectificación; manual y revisión de minimización. | Pruebas focales/manual propios. | A13–A16 montados; A17 conserva su alcance preparatorio. | 4–6 |

A00–A12 cierran la modalidad inicial hasta subsanación y consulta. A13–A16/A18
amplían a concesión/efecto/reintegro solo con decisión y contratos admitidos. A17
prepara anticipos/pensiones; su gestión completa se estimará tras inventariar gestora,
aportaciones y circuitos, sin contar aquí sustitución de la contabilidad o Nóminas.

A07/A08 solo proceden si no existe una fuente admitida que mantenga ese expediente.
Recalcular al reutilizar su estado. Antes de cualquier migración: número/orden causal
en `RESERVAS_MIGRACIONES.md` fuera de Git, borrador, preimagen, ensayo en clon de
principal y dos revisiones independientes. MCP `postgres-clon-local` para lectura/
EXPLAIN; dirección instala. Nunca reaplicar historia, DOWN destructivo ni SQL en
cidonia por este carril. Nuevos efectos de la ampliación revalidan su persistencia.

## Dependencias y paralelismo

A00/A01 preceden a lectura/presentación nominales; A02 y DTO de A03 pueden avanzar
en paralelo, pero A03 no lee sin permiso. Dos equipos reparten catálogo/cliente
(A04/A05) y evidencia/persistencia (A06–A08) por archivos. A09 reúne ambas bases;
A10 y A11 pueden separarse cuando sus contratos estén cerrados; A12 integra.
Después, presupuesto/propuesta y acuerdo documental se preparan con un dueño por
contrato; el pago y reintegro esperan resolución y receptor económico. Una vista,
catálogo o archivo común tiene un solo escritor. K/L/B y Sistemas conservan sus
colas propias, fuera de este esfuerzo.

## Configuración y estimación

Modalidad, entidad/colectivo, servicios, ejercicio, documentación, cuantía/topes,
presupuesto, prelación/incompatibilidades, comisión/órgano, plazos, custodia y
conservación se configuran con fuente/acto, órgano, versión, huella y vigencia.
La solicitud conserva la versión aplicable. Textos/categorías usan datos por idioma.
La modalidad no solicita unidad familiar completa o informes clínicos por defecto.
RAT y habilitación específica para datos de salud/familia, información, destinatarios,
riesgos/EIPD cuando proceda, bloqueo/archivo/expurgo esperan DPD/Archivo antes de datos
reales. Historia de solo adición no permite conservación ilimitada.

A00–A12 suman **52–84 h**, **7–11 jornadas de un equipo** de ocho horas; con dos
equipos **5–9 jornadas**, condicionadas a fuentes/bases y al solape descrito.
A13–A18 añaden **33–54 h**: conjunto **85–138 h**, **11–18 jornadas de un equipo**,
o **9–15 jornadas con dos equipos**. Son días completos redondeados y contienen
integración/revisión; no prometen plazo de respuesta externo.

| Trabajo externo, fuera del total | Dedicación orientativa | Condición |
| --- | ---: | --- |
| RRHH / unidad de Acción social | 6–10 h | Resolver la 130 y verificar convocatoria/órganos, mínimos y recorrido. |
| Sistemas / Registro / fuente actual | 6–12 h | Interfaz, asiento, documentos, notificación y entorno de ensayo; no API presunta. |
| K/L, Documentos y Personal B | Estimación de sus dueños | Contexto/perfiles, auditoría, custodia/firma y vínculo/servicios; aquí solo consumidores. |
| DPD / Archivo | 4–8 h | RAT/evidencia mínima, acceso reservado, series y riesgos; aprobación sin plazo comprometido. |
| Comisión / órgano / Contabilidad / Nóminas / Tesorería | 6–12 h | Competencias, presupuesto y contrato/estado económico para la ampliación. |
| Gestora de pensiones y circuito de anticipos | Estimación posterior | Inventario A17; gestión completa fuera del total. |

## Comprobación y entrega

La PR documental verifica inventario por SHA, enlaces locales, sumas y
`git diff --check`; requiere revisión independiente documental y una CI de PR.
No ejecuta Go, SQL o navegador de producto ni acredita instalación.

Cada implementación usa `programar-backend-vec`, `persistir-autorizar-vec`,
`programar-interfaz-vec` y `probar-recorridos-vec` según delta. Pantallas con
`usabilidad-vec`, `aspecto-vec`, `disenar-sistema-visual-vec` e
`impeccable`/`VEC-PRIORIDAD.md` antes de programar y revisión de usabilidad
independiente. Textos con `humanizer`/`VEC-USO.md`; SQL con
`revisar-sql-vec`/`ensayar-sql`; `security-audit` focal y Semgrep local sobre cambios
sensibles, sin servicios externos. gopls para Go, índice primero para código.
`revisar-cambios-vec`, `documentar-entregar-vec` y `pr-vec` sobre candidata exacta;
dirección integra/despliega.

Aceptar exige presentación/reintento sin duplicados, titular ajeno, competencia/
canal incorrectos, revocación/caducidad, permiso separado de descarga, evidencia
mínima, subsanación con historia y fuente/auditoría caídas sin datos. PostgreSQL real
para efectos y recuperación tras reinicio; Chrome del sistema en PC/móvil con
teclado/foco e i18n. Pago solo se afirma con confirmación del receptor competente.
No se repiten campañas integradas sin delta/fallo. Las dudas no detienen contratos
independientes ni autorizan competencias, cuantías o resoluciones sintéticas.
