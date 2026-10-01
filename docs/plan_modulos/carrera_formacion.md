# Carrera y Formación: continuación de Codex-H

Estado comprobado el 1 de octubre de 2026 sobre `origin/main@0a62a3ea6`.
La rama `main` de la raíz local es histórica; no utilizarla como base de producto.
Claude revisa, integra y despliega. Este plan no autoriza instalación ni uso de datos reales.

## Lo que existe

| Pieza en main | Ruta | Estado y límite |
| --- | --- | --- |
| Relaciones y servicios propios | `internal/modules/personal/domain/ficha_propia.go`; `web/static/portal-empleado/modulos/personal/cliente-http-ficha-propia.js` | Consulta compuesta de Personal. No aporta todavía grado ni subgrupo estructurado para Carrera. |
| Antecedentes y actos de Personal | `internal/modules/personal/domain/registro_empleado_b2_actos.go` | Registra relación, ocupación, servicio y situación. No registrar un grado usando otro tipo de acto. |
| Valoración del grado reconocido | `internal/modules/provision/domain/concursos_contrato.go`; `concursos_calculo.go` | Distingue nivel del puesto y grado personal. Puntuar un mérito no reconoce un grado. Propiedad de B; no duplicar. |
| Vista de méritos | `web/static/portal-empleado/modulos/meritos/vista.js` | Consulta por datos inyectados, sin fuente de Formación ni Registro Único de Méritos compuesto. |
| Méritos del candidato | `internal/candidate/domain/merit.go`; `application/service.go` | Ligados a convocatoria. `AddMerit` no sustituye al registro único ni a la acreditación de oficio. |
| Documentos comunes | `internal/vec/application/documentos.go`; `internal/vec/ports/documentos.go` | Reutilizar custodia y firma; un borrador o certificado de acceso no acredita firma documental. |

No hay módulos propios `carrera` ni `formacion` en esa base publicada.
Granada sí dispone de [plataforma y plan institucional de Formación](https://www.dipgra.es/servicios/empleo-y-formacion/formacion-para-el-empleo-publico/plan-agrupado-de-formacion/).
Orden de dirección 16:15: **VEC consulta y deriva** mientras se aclara qué funciones conserva esa plataforma.

## Candidatas de esta sesión

| Rama propia | Fuente congelada | Resultado |
| --- | --- | --- |
| `trabajo/codexh-formacion-plan-20261001` | `17b29f11d` | CLI Go revisa un plan sintético, fechas, necesidades, ediciones, plazas y presupuesto; visor ES/EN con filtro, detalle, pendientes y descarga del JSON original. |
| `trabajo/codexh-carrera-preparacion-20261001` | `ccd814ba71d64b41c4d4ba707113bc9b8c0c0c8d` | CLI Go revisa integridad de grado, progresión y promoción; conserva antecedentes, fuentes, periodos y referencias. Visor ES/EN y descarga. Todos los casos siguen pendientes. |

Ambas fuentes están en ramas locales, todavía sin PR ni CI; tienen pruebas focales
y revisión independiente, con calidad global en comprobación. Antes de cerrar se
anotarán sus PR y SHA remotos exactos.

Son herramientas locales de preparación, no inscripciones, solicitudes registradas,
selección, reconocimiento, firma ni incorporación a méritos. No añaden SQL o permisos.
Los ejemplos son retirables: están en JSON de entrada y en la proyección de cada visor;
no se instalan en bases ni se publican en manifiestos productivos.
Arranque común: `python3 scripts/servir_preparacion_rrhh.py --modulo formacion`
o `--modulo carrera`. Solo sirve recursos cerrados en loopback, sin credenciales.
Antes de retomar, comprobar las PR y su integración; nunca volver a producir estas piezas.

## Huecos frente al catálogo

| Capacidad | Trabajo pendiente |
| --- | --- |
| CAR-001 Grado funcionarial | Fuente de relación, nivel, grado reconocido y periodos; política provincial versionada; expediente autorizado, revisión, resolución e inscripción por Personal. La preparación actual no concede derechos. |
| CAR-002 Progresión laboral | Convenio consolidado y modificaciones, colectivo y categoría; convocatoria, curso/prueba, revisión y resolución. No convertir antigüedad en progresión automática. |
| CAR-003 Promoción interna | Requisitos de las bases y antecedentes de Personal; referencia al proceso de A, seguimiento de pruebas y acto/toma de posesión por B. No crear otro motor selectivo. |
| FOR-001 Plan de formación | Consulta del plan y ediciones de la fuente institucional, con versión y procedencia. Necesidades, presupuesto y publicación se gestionarán donde RRHH confirme su autoridad. |
| FOR-002 Solicitud y selección | Derivación canónica o adaptador corporativo; solicitud/estado/recibo originales, autorización competente, criterios, listas y sustituciones. No implementar otra inscripción o selección por defecto. |
| FOR-003 Ejecución y certificado | Fuente de asistencia, aprovechamiento, evaluación, coste y certificado firmado. Entrega idempotente de hechos acreditados a Méritos, sin atribuir puntos ni grado. |

## Orden de minitareas

Ahora solo se terminan las dos candidatas casi hechas y este plan. Las filas siguientes
requieren reanudación expresa de dirección; no se empieza más producto en esta sesión.

Cada fila se entrega con un consumidor o una comprobación concreta. Si no cabe,
dividir por el contrato indicado; avisar a dirección tras 30 minutos sin avance.

| Orden | Minitarea y propietario | Salida y condición para continuar | Archivos previstos | Dependencias |
| --- | --- | --- | --- | --- |
| H01 | H: verificar PR, SHA de main y limpieza de esta entrega | Inventario actualizado en este fichero; no repetir candidatas integradas. | Este plan y `CANAL_CLAUDE_CODEX.md` | Cierre y PR de las candidatas |
| H02 | H con RRHH: identificar fuente institucional de Formación | Decisión sobre catálogo, inscripción, selección y certificados; URL/identificadores/contrato confirmados. Sin respuesta, solo derivación informativa. | Este plan; `dudas.md` solo en turno H | RRHH; autoridad corporativa; turno H |
| H03 | H: puerto de consulta de catálogo corporativo | Plan/acción/edición, fecha, plazas, estado, fuente y versión; datos mínimos. Una fuente caída no genera ejemplos como sustituto. | `internal/modules/formacion/ports/catalogo.go` | H02 y contrato confirmado; no inferir API de una URL |
| H04 | H: adaptador de esa fuente y consumidor interno de solo lectura | Consulta o enlace canónico acordado, estados vacío/error/denegado, ES/EN. Sin autoridad de lectura propia, no devolver solicitudes de personas. | `formacion/adapters/corporativo/*`; `modulos/formacion/cliente-http.js`, `vista.js` y catálogos | H02–H03; lectura autorizada si hay datos propios; turno de montaje |
| H05 | B: contrato autorizado de antecedentes para Carrera | Persona/empleado/referencia de relación, régimen, grupo/subgrupo, ocupaciones, nivel, grado y servicios con procedencia, versión y vigencia. Consulta por puerto, sin tablas cruzadas. | Puerto de Carrera `ports/antecedentes.go`; B decide sus archivos propios | Acuerdo B; contrato sin leer datos personales |
| H06 | H: política CAR-001 en catálogo de datos | Vías, periodos computables, límites y evidencias según fuente provincial aprobada. Configuración ausente o sin aprobación devuelve pendiente. | `data/catalogos/carrera/*`; `carrera/domain/politica.go` | RRHH valida fuente/versión; solo pruebas sintéticas hasta H08 |
| H07 | H: expediente de preparación CAR-001 con esas instantáneas | Faltantes, contradicciones, periodos y revisión explicada. Nivel del puesto separado del grado; no sumar periodos solapados. | `carrera/application/preparar.go`; `modulos/carrera/vista.js` y catálogos | H05–H06; H08 y lector autorizado B antes de consumir datos personales |
| H08 | H y autoridad central: perfil fijo y acciones Carrera | Consulta/preparación/decisión/descarga exactas, ámbito y finalidad. Provisión por huella y CAS; nunca publicación de permisos por petición. | `carrera/ports/autorizacion.go`; composición nominal; gobierno central según dueño | Autoridad central; lector nominal B; SQL reservado si corresponde |
| H09 | H: candidata de persistencia y SQL borrador | SQL solo en borrador reservado, con posición en `ORDEN_SQL_NUCLEO.md`; historia aditiva, versión y operación idempotente. Instalación solo por dirección. | `carrera/adapters/postgres/*`; SQL nuevo reservado; `ORDEN_SQL_NUCLEO.md` por custodio | H05–H08; ensayo/clon y revisión sensible; sin instalación implícita |
| H10 | H: revisión/resolución CAR-001; B: inscripción del acto | Documento y firma reales por sus puertos. Personal confirma inscripción con recibo; Carrera conserva la referencia y reconcilia fallos sin duplicar el acto. | `carrera/application/resolver.go`, puertos documentos/registro; B decide inscripción | H09 instalado por dirección; H08 montado; recorrido durable; firma real |
| H11 | H y A: cotejo CAR-003 frente a bases versionadas | A conserva convocatoria, admisión, pruebas y decisión selectiva. Carrera muestra cumple/no cumple/pendiente con motivo y fuente; no sustituye la admisión. | `carrera/ports/proceso_selectivo.go`; `application/cotejar_promocion.go` | A confirma contrato y autoridad; H05 y H08/lector B antes de datos personales |
| H12 | H y B: seguimiento de promoción | Estado desde A y toma de posesión desde B. Un resultado de prueba o una propuesta no crea una relación de servicio. | `carrera/application/seguimiento_promocion.go`; vista y catálogos propios | H11; H08; fuentes autorizadas A/B y actos reales de B |
| H13 | RRHH y H: política CAR-002 independiente | Convenio vigente consolidado, cadena de modificaciones, colectivo/categoría, convocatoria y curso/prueba admitidos. Sin fuente aprobada, preparación pendiente. | Catálogo Carrera de progresión; pregunta en `dudas.md` en turno | RRHH/convenio consolidado; H08 antes de datos personales |
| H14 | H: expediente CAR-002 sobre contratos H05/H08/H09 | Mismo patrón de integridad, historia y recibo; política propia, sin reutilizar requisitos funcionariales. Resolución e inscripción separadas. | `carrera/domain/progresion.go`; caso de uso/vista propios; SQL borrador si nuevo dato | H05, H08; H09 instalado y H13 aprobado antes de efecto real |
| H15 | H: consulta propia/derivación FOR-002 con la autoridad confirmada | Recuperar estado y justificante original; distinguir solicitud, autorización, selección y notificación. No confirmar éxito ante resultado incierto. | `formacion/ports/solicitudes.go`; adaptador corporativo y vista propia | H02–H04; autoridad nominal/contexto empleado B; permisos/privacidad revisados |
| H16 | H: hechos FOR-003 desde Formación y Documentos | Asistencia, superación y certificado son estados diferentes; validar procedencia y firma/custodia según contrato, sin emitir certificados ficticios. | `formacion/ports/ejecucion.go`, `certificados.go`; adaptadores fuente/documentos | H02; lectura nominal propia y de documentos; competencia/aprobación confirmadas |
| H17 | Autoridad de Méritos con H: entrega y reconciliación | Curso, horas, resultado, evidencia y vigencia se incorporan una vez cuando proceda. Conservar recibo y reintento; no copiar puntos de una convocatoria. | `formacion/ports/meritos.go`; entrega/reconciliación; Méritos decide consumidor | H16; autoridad Méritos confirmada; permisos; SQL/eventos durables autorizados |

H03/H04 se entregan juntos con consumidor. Una URL pública permite derivar,
pero no demuestra que exista una API. H05 acuerda el contrato; H07 usa únicamente
sintéticos hasta H08 y el lector autorizado de B.

H02–H04 pueden avanzar a la vez que H05–H07 en archivos exclusivos.
H11 se acuerda con A mientras RRHH valida H06; no invadir su proceso.
H13 no bloquea CAR-001/CAR-003 ni la derivación de Formación.

## Decisiones y dependencias

En el turno H, numerar en `dudas.md` las preguntas ya enviadas al canal: funciones
que conserva Formación corporativa, convenio vigente para laborales y vías de
reconocimiento del grado. Añadir responsables, fuente/versiones y fecha de aprobación
a los catálogos; ninguna cifra de ensayo se convierte en norma oficial.

B conserva Personal, sus actos y provisión; A conserva procesos selectivos; el baremador conserva su autoridad y responsable
confirmado, sin reasignarlo por inferencia;
M conserva Organización/RPT. Documentos conserva custodia y firma. La autoridad de
Méritos debe confirmarse: el modelo del candidato no permite declararla ya resuelta.
Las operaciones entre módulos usan referencias opacas, eventos y recibos reconciliables.

Manifiestos, padres, importadores y dudas siguen la cola B → A → E → G → F → M → H → I → J,
con LIBERO y SHA. No iniciar una migración sin reserva y orden causal, ni usar un
borrador SQL como capacidad instalada. No escribir en cidonia ni en datos reales.

## Pruebas y primera acción de la próxima sesión

Primero comprobar `gh pr list`, los FIN de H y los hashes publicados de las dos candidatas.
Después ejecutar **H02**: cerrar con RRHH el contrato de Formación y registrar la decisión.
Si todavía no responde, acordar **H05** con B y empezar el contrato de antecedentes
de CAR-001; no crear un catálogo corporativo paralelo ni otra demo.

Pruebas focales de fechas, fuentes, solapes, versiones y pendientes; Node sobre la
interfaz afectada; Chrome del sistema a 1440/390 y zoom 200 %, ES/EN y teclado.
Gosec solo en paquetes cambiados y Semgrep local sin métricas. Revisión independiente;
dos en SQL, permisos, identidad, firma o datos personales. Calidad global una vez
por PR cuando corresponda; no repetir puertas verdes. Registrar impedimentos de entorno sin omitir controles obligatorios.

## Consenso Astra

- Se compararon Granada, Alicante, EAPC, INAP y GVA. Granada ya tiene plataforma
  formativa: ausencia de módulo VEC no autoriza sustituirla. Dirección confirmó consulta y derivación.
- Se acordó separar grado, progresión laboral y promoción. No existe una regla universal
  de examen o ascenso por antigüedad; solo políticas provinciales aprobadas y versionadas.
- Las herramientas inmediatas se limitaron a preparación sintética exportable. El canal
  Go → JSON → visor local evita tocar el servidor de ensayos que pertenece a A.
- Se conservaron procesos de A, actos/registro de B y custodia/firma comunes. Un curso
  acreditado no equivale a puntos, y una resolución no equivale a inscripción confirmada.
- Ronda final: Astra dio GO al contenido `af90952f…` tras exigir archivos y dependencias
  por minitarea, autorización antes de antecedentes personales, SQL borrador e instalación
  separados, SHA/estado de candidatas y parada explícitos. La tabla se corrigió a cinco
  columnas sin cambiar su contenido. Falta el GO de dirección sobre este documento exacto;
  no abrir la PR del plan antes de recibirlo.
