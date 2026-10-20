# Estado de Dietas para el tablero, 21 de octubre de 2026

Tarea: TABLERO-CRONOS-DIETAS. Base examinada: `origin/main`, commit
`cda070a31`. Comprobaciones ejecutadas el 20 de octubre, para la actualización
del 21. No se ha cambiado código, SQL ni configuración versionada.

Dietas tiene solicitud propia, cálculo provisional, circuito y correcciones
conectados a la aplicación. Las pruebas Go de sus 16 paquetes pasan. No está
acreditado aquí el uso completo desde el navegador: ninguna fila nueva se
propone como «Hecha» por el mero resultado de esas pruebas.

## Qué significa cada estado

Criterio fijado por dirección en el canal el 20/10, a las 22:10:

- **Fusionada:** código en main, composición raíz, ruta, orden SQL versionado,
  pantalla y pruebas Go positivas. Se añade «sin recorrer en la web».
- **Hecha:** además se ha recorrido y comprobado en la web. Esta entrega no
  propone ninguna capacidad nueva con ese estado.
- **Construida sin montar:** existen piezas y pruebas, pero falta conectarlas
  a la aplicación. Se conserva como pendiente en el tablero.
- **A medias:** parte del alcance está conectada; faltan efectos, dependencias
  o comprobaciones para dar la capacidad completa por terminada.
- **No existe:** no se ha localizado una implementación del efecto pedido en main.

«Sin comprobar en la web» es una limitación adicional. Una capacidad montada
con esa limitación no se llama «sin montar».

## Cortes del plan DIE-01 a DIE-07

Las rutas se expresan respecto a la raíz del repositorio; los números señalan
la primera línea relevante de cada evidencia.

| Capacidad | Estado y alcance demostrado | Evidencia de main y prueba | Qué falta |
| --- | --- | --- | --- |
| DIE-01. Comprobar las fuentes de las tarifas antes de calcular | Fusionada; sin recorrer en la web | `internal/modules/dietas/adapters/postgres/tarifas_provisionales.go:40` lee las dos referencias; `internal/modules/dietas/application/preparar_comision_test.go:86` rechaza fuente ausente o incongruente. `internal/app/bootstrap/dietas_comisiones.go:711` crea el preparador y `:719` lo inyecta. PR [#838](https://github.com/aavidad/VEC_Diputacion_app/pull/838), integrada por `aab120dc2` | Confirmar cálculo nominal y su recuperación desde pantalla con el SQL y configuración de la copia acreditados. El corte no publica tarifas aprobadas |
| DIE-02. Envío con justificantes custodiados y antivirus | A medias | Borrador y envío montados en `internal/app/bootstrap/dietas_borradores.go:98` y `:112`; ruta de envío admitida en `internal/app/bootstrap/dietas_comisiones.go:185`. `internal/modules/dietas/application/custodia_justificantes_comision.go:79` registra referencias, sin bytes. Prueba `TestJustificanteComisionLlegaAlPuertoComunSinContenido`, en `application/custodia_justificantes_comision_test.go:58` | Subida efectiva a Documentos, inspección, custodia de bytes y envío con recibo; negativos de inspección y custodia. No confundir huella con fichero guardado |
| DIE-03. Bandeja y decisión por perfil, etapa y unidad; devolución y reenvío | A medias; circuito montado, sin comprobar en la web | `internal/app/bootstrap/dietas_borradores.go:119` compone servicio e identidad del circuito y `:135` enruta peticiones. `internal/app/bootstrap/dietas_asignacion_apertura.go:33` usa perfiles fijos. Pruebas `TestCircuitoHTTPDerivaUnidadYRechazaSuplantacion` y `TestDocumentoCircuitoProyectaDevolucionAnteriorSinActor`, en `adapters/httpinterno/circuito_comision_test.go:130` y `:189` | Asignaciones y competencias consumibles, bandejas y decisiones reales en la copia; devolución, edición y reenvío con historia. El alta/corrección completa de asignación sigue cerrada por catálogo vacío (`dietas_asignacion_apertura.go:29`) |
| DIE-04. Liquidación durable, rectificación enlazada y PDF exacto | A medias; propuesta construida sin montaje económico | `internal/modules/dietas/application/preparacionliquidacion/preparacion.go:1` declara propuesta local sin efectos; `cmd/vec-dietas/main.go:43` ofrece preparación por terminal. `deploy/postgresql/dietas_borradores/borradores/000012_liquidacion_economica.md:1` es diseño, no migración instalable. Paquete `application/preparacionliquidacion` pasa | Acto económico persistido, instantánea, recibo, rectificación, ruta y pantalla; PDF del acto exacto. La decisión genérica de la etapa «liquidación» no acredita ese efecto |
| DIE-05. Fiscalización e informes nominales con exportación separada | A medias; etapa montada e informes locales construidos sin montaje nominal | `internal/app/bootstrap/dietas_asignacion_apertura.go:39` incluye fiscalización; `internal/modules/dietas/application/informeperiodo/informe.go:1` limita el informe a paquetes ficticios, sin consulta de expedientes ni autorización. `cmd/vec-dietas/main.go:21` y `:24` ofrecen CSV/PDF. Paquetes `application/informeperiodo` y `adapters/informeperiodo` pasan | Fiscalización del acto económico de DIE-04; lectura nominal paginada, política de campos y autorización/auditoría de exportación; consumidor en el portal |
| DIE-06. Salida económica con acuse, rechazo y conciliación | No existe el efecto integrado en main | El plan `docs/plan_modulos/dietas.md`, tabla de cortes, exige contrato del receptor; el montaje de `internal/app/bootstrap/dietas_borradores.go:119` termina en el circuito. No se ha localizado envío al receptor, acuse ni conciliación en los consumidores de Dietas | Contrato receptor, lote durable, permisos, acuse, reintento y conciliación. Fiscalizado no significa pagado |
| DIE-07. Recorrido completo con recibo, negativos y reinicio | A medias | Los 16 paquetes Go pasan; las piezas anteriores conservan los límites descritos. El plan `docs/plan_modulos/dietas.md` exige navegador, PostgreSQL, reintento y recuperación | Completar DIE-02 a DIE-06 y registrar un recorrido sobre una única copia, con reinicio, recibos y revisiones. La copia local no permite completar el circuito; véase la comprobación de navegador |

## Capacidades que ya conviene inventariar por separado

Estos componentes permiten dividir las tareas grandes del tablero. La columna
«Montaje» describe el código; no acredita instalación en una base concreta.

| Capacidad | Montaje y evidencia | Estado para el tablero |
| --- | --- | --- |
| Alta, consulta, edición, borrado lógico y envío de comisión propia, con recibos | `internal/modules/dietas/adapters/httpinterno/borradores.go:21`; raíz en `internal/app/bootstrap/composicion_desarrollo.go:413` y `:423`. `TestCrearBorradorUsaFuncionDietasNominalYConfirmaAntesDeExponer` y `TestCrearBorradorNoExponeReciboSiCommitEsIncierto`, en `adapters/postgres/borrador_comision_test.go:156` y `:174`, pasan con dobles de base | Fusionada; sin recorrer en la web |
| Cálculo orientativo de manutención y kilometraje con fuentes y versión | `internal/app/bootstrap/dietas_comisiones.go:711`; prueba `TestPrepararComisionConservaOrdenYCalculaImporteServidor`, en `application/preparar_comision_test.go:122` | Fusionada; sin recorrer en la web; no liquidable |
| Itinerario y mapa con cálculo de ruta interno | `internal/app/bootstrap/bootstrap.go:311` y `:337`; `web/static/portal-empleado/portal-composicion-empleado.js:158` compone calculador y visor. Paquetes `adapters/osrm` y `application` pasan | Fusionada; sin recorrer en la web. Activación de motor y cartografía sin acreditar |
| Gastos con justificante por referencia y huella | `web/static/portal-empleado/modulos/dietas/INTEGRACION.md`, apartado «Ruta y mapa»; SQL `000009_otros_gastos_justificados.up.sql`, incluido en `deploy/principal/04_dietas_migraciones.sh:93`; aplicación de custodia en `application/custodia_justificantes_comision.go:79` | A medias; referencia y huella construidas, subida/custodia real pendientes |
| Bandejas, revisión, autorización y transiciones de liquidación/fiscalización | `internal/app/bootstrap/dietas_borradores.go:119`; cliente inyectado en `web/static/portal-empleado/portal-composicion-empleado.js:162`; competencias en `dietas_asignacion_apertura.go:33` | Fusionada como circuito; sin recorrer en la web. No sustituye liquidación económica; falta acreditar asignaciones consumibles |
| Devolución, motivo, corrección y reenvío | SQL `000010` y `000011` enumeradas juntas en `deploy/principal/04_dietas_migraciones.sh:94` y `:95`; prueba HTTP de devolución en `adapters/httpinterno/circuito_comision_test.go:189` | Fusionada; sin recorrer en la web. Instalación de la copia sin contrastar |
| Consulta y solicitud de rectificación propia de asignación | `web/static/portal-empleado/portal-modulos-coordinador.js:219` importa cliente; `portal-composicion-empleado.js:164` lo crea y `:184` lo inyecta. `internal/app/bootstrap/dietas_borradores.go:161` añade rutas si existe auditoría de rectificación; `dietas_comisiones.go:599` comprueba disponibilidad | A medias; montaje propio condicionado. La resolución administrativa completa no está inyectada |
| Propuesta local de liquidación, comparación y recuperación | `cmd/vec-dietas/main.go:27` y `:43`; paquetes `application/preparacionliquidacion` y `adapters/informeliquidacion` probados | Construida sin montar en API/portal; Pendiente |
| Informes por persona, unidad, periodo y situación, con CSV/PDF locales | `application/informeperiodo/informe.go:1`; `cmd/vec-dietas/main.go:21`; pruebas de los paquetes de informe positivas | Construida sin montar sobre expedientes nominales; Pendiente |
| Publicación administrativa de tarifas y liquidación económica posterior | Fuera de main: PR abiertas [#1136](https://github.com/aavidad/VEC_Diputacion_app/pull/1136) y [#1139](https://github.com/aavidad/VEC_Diputacion_app/pull/1139), ambas pausadas y en borrador | Pendiente; conservar F4-01 como Aparcada. La evidencia de esas ramas no se atribuye a main |

## Rutas, SQL y acceso desde el portal

La composición de desarrollo crea Dietas en
`internal/app/bootstrap/composicion_desarrollo.go:404` y `:413`, incorpora sus
rutas en `:423`, y protege el manejador en `:604`. El preparador usa tarifas
propias y la relación/asignación que Personal acredita por sus puertos. Este
montaje condicionado no demuestra que todos sus selectores estén activados.

La pantalla existe y recibe clientes en
`web/static/portal-empleado/portal-composicion-empleado.js:144`. Dietas no tiene
entrada en menú ni en Inicio; su URL directa se conserva, según
`portal-modulos-coordinador.js:85`. Oculta en el menú y sin montar son situaciones
distintas.

El SQL ejecutable de borradores llega a `000011`. El orden histórico de base e
incremental está en `deploy/principal/04_dietas_migraciones.sh:54` y `:83`.
`deploy/principal/lista_sql_main_tras_hz37.txt:1` enumera solo las migraciones
posteriores al volcado de referencia del 14/10: no contiene entradas de
`dietas_borradores`. El listado de Dietas en main es el que ensambla
`04_dietas_migraciones.sh`: base en `:54` e incremental en `:83`, con
000009–000011 en `:93`, `:94` y `:95`. Su orden y sus fuentes están cubiertos
por `deploy/principal/ensamblado_dietas_test.go:12` y `:61`. Es evidencia del
orden versionado en main; no se presenta como una lista nueva ni como una
instalación. La ausencia de entradas en la lista posterior a HZ37 tampoco
prueba que falten las anteriores en el volcado. En la copia restaurada se
comprobó que existen `vec_dietas.comision_revision` y
`vec_dietas.cola_circuito_comision`; esa consulta no acredita por sí sola
las firmas, cuerpos y permisos de todas las migraciones.

## Comprobación en copia local, 20/10 a las 22:18

Dirección corrigió a las 22:10 la decisión anterior y pidió comprobar Dietas
en una copia local sin `--modulos`. Esa opción prepara Bolsa y Contratación,
no Dietas. Se usó la receta de main sobre la cabeza documental `3b5035e0c`,
con el mismo código de aplicación que `cda070a31`.

La primera creación sin `--modulos`, con
`--lista-sql deploy/principal/lista_sql_main_tras_hz37.txt`, restauró el
volcado HZ37 y sus permisos, pero la instalación posterior terminó con
`AD278: origen CT nominal incompatible`. No llegó a generar material ni a
compilar la aplicación. No se alteró la migración ni se omitió su guarda.
Se borró la instancia incompleta y se restauró de nuevo en el mismo destino,
sin la lista general, para comprobar la base disponible. No se atribuye a
esta segunda restauración ninguna migración posterior a HZ37.

La orden utilizada fue `deploy/principal/copia_local.py crear --equipo d
--instancia estudio-dietas-local-20261021`, con el origen HZ37 autorizado.
Terminó con salida 0 y anunció la aplicación disponible en 68,84 segundos.
La copia genera sus certificados y arranca con Dietas apagado. El inventario
confirmó que no existe ninguna de las once cuentas técnicas de la receta
actual de Dietas y que falta `identidad/dietas-comisiones.json` en el material
recién generado.

Con el entorno del proceso de esa copia y solo
`VEC_DIETAS_BORRADORES_ENABLED=true`, `vec-server comprobar-dietas` terminó
con salida 1: «faltan conexiones PostgreSQL separadas para borradores de
dietas». Es una comprobación previa en solo lectura; no se reinició la
aplicación con configuración incompleta ni se afirma Dietas encendido.

Faltan cuatro conexiones de entorno y siete conexiones del manifiesto, con
identidades distintas, y las once cuentas técnicas. Los motivos y las
asignaciones que deben consumir esas conexiones quedan sin acreditar. La composición exige también las cinco variables `VEC_OSRM_*` y
material de autorización del gobierno común de Contratación
(`internal/app/bootstrap/dietas_comisiones.go:469` y
`internal/app/bootstrap/dietas_comprobacion_arranque.go:59`).
`deploy/principal/03_entorno.md:3` prohíbe repetir el preparador histórico;
la receta actual de `f4b_acceso_dietas/README.md:12` presupone ocho cuentas
anteriores y la asignación retirada. Las cuentas faltan en esta copia; la
asignación requerida no se ha acreditado. Activar el selector o reinstalar las migraciones de tablas no produce ese material.
No se improvisó otro productor, conforme a la orden de dirección.

Chrome del sistema, manejado con Playwright y el certificado propio de la
copia, abrió `/portal-empleado/?lang=es#dietas` a 1440 × 1000. La sesión
respondió 200 y se pintó «Mis dietas y comisiones de servicio». El resultado
fue el siguiente:

| Paso pedido | Resultado observado |
| --- | --- |
| Solicitud | «Nueva comisión de servicio» deshabilitada; el aviso indica que no se han podido consultar las relaciones de Personal |
| Consulta de borradores | «Consultar borradores registrados» hace la petición real, que responde 503 |
| Envío | Sin alta disponible; no se envió ninguna comisión ni se obtuvo recibo |
| Bandeja | Competencias del circuito responde 503; no se acreditó ninguna bandeja operativa |
| Devolución y reenvío | No ejecutables sin comisión y competencias; no se atribuye ningún acto ni historia |

Las peticiones a `/api/vec/personal/relaciones-dietas`,
`/api/vec/dietas/route-catalog`, `/api/vec/dietas/comisiones?limit=6` y
`/api/vec/dietas/comisiones/circuito/competencias` devolvieron 503.
La consola registró esos fallos, otros 503 de preferencias y un error de
certificado al cargar un script; no se presenta como recorrido sin errores.

La captura `dietas-local-no-disponible.png` y la instantánea
`dietas-local-final.yml` se conservan en la evidencia local del equipo,
fuera de Git. SHA-256 de la captura:
`ac50072a295faad9aeade3bb72abd3a4ecf7b04baf6a1f1436d88a22518e680d`.
Se conservan también el inventario y la salida de la comprobación previa,
sin secretos. La captura acredita la pantalla de indisponibilidad, no la
funcionalidad de solicitud o circuito.

Tras la comprobación se cerró la sesión propia de Chrome y se ejecutaron
`copia_local.py parar` y `copia_local.py borrar` para esa instancia, ambas
con salida 0. Se liberó el turno para C en el canal. No se usaron copias ni
servicios de otros equipos ni se accedió a cidonia. F4-08 a F4-11 conservan
«Fusionada, sin recorrer en la web»; F4-12 a F4-14 siguen «Pendiente».

## Documentación e historial que requieren cautela

`web/static/portal-empleado/modulos/dietas/INTEGRACION.md` conserva dos notas
anteriores al montaje actual: dice que la rectificación propia no se inyecta
y que la competencia solo responde sin fuente. El cliente propio sí se
inyecta y la raíz usa la fuente de perfiles fijos indicada arriba. Eso no abre
la resolución administrativa de rectificaciones ni garantiza asignaciones
vigentes. El estudio aplica el código de main.

DIE-01 ya figura integrado en el plan desde el 7/10. Las PR fusionadas que
explican otros componentes son #36 y #40 (composición/circuito), #49
(gastos y devolución), #258 (solicitud), #259 (bandejas), #424 (competencia),
#427 (rectificación), #683 y #693 (informes CSV/PDF), #838 (fuentes), #846
(retirada de páginas de muestra) y #870 (catálogo compartido). Se contrastaron
con `gh pr list --state merged --search dietas --limit 100` y el historial de
`origin/main`. La consulta literal `--search 'DIE-'` no devolvió PR; #838 se
titula «DIE01». El inventario de ramas encontró las continuaciones de
activación y liquidación; sus efectos pendientes se conservan en F4-01.

## Filas propuestas a C, sin editar el tablero

| ID propuesto | Tarea acotada | Estado propuesto | Evidencia para la fila |
| --- | --- | --- | --- |
| F4-08 | DIE-01: fuentes normativas antes del cálculo | Fusionada; sin recorrer en la web | #838; `dietas_comisiones.go:711`; SQL de base en `04_dietas_migraciones.sh:54`; prueba de fuentes ausentes/incongruentes |
| F4-09 | Solicitud propia y recuperación de recibos | Fusionada; sin recorrer en la web | `dietas_borradores.go:98`; `portal-composicion-empleado.js:184`; SQL de base e incremental en `04_dietas_migraciones.sh:54` y `:83` (documento en `:88`); prueba de confirmación antes de exponer |
| F4-10 | Ruta y cálculo orientativo versionado | Fusionada; sin recorrer en la web | `bootstrap.go:311`; `dietas_comisiones.go:711`; SQL de base en `04_dietas_migraciones.sh:54`; paquetes OSRM/aplicación positivos |
| F4-11 | Bandejas, devolución y reenvío | Fusionada como parte de DIE-03; sin recorrer en la web | `dietas_borradores.go:119`; cliente en `portal-composicion-empleado.js:162`; SQL incremental en `04_dietas_migraciones.sh:83`, `:94` y `:95`; prueba HTTP de devolución |
| F4-12 | Consulta y solicitud de rectificación propia | Pendiente; montaje condicionado | `dietas_borradores.go:161`; `portal-composicion-empleado.js:164` |
| F4-13 | Preparación, comparación y recuperación local de liquidación | Pendiente; construida sin montar en portal | `cmd/vec-dietas/main.go:27`; `application/preparacionliquidacion/preparacion.go:1` |
| F4-14 | Informes locales por periodo y salidas CSV/PDF | Pendiente; construida sin montar sobre datos nominales | `cmd/vec-dietas/main.go:21`; `application/informeperiodo/informe.go:1` |

C decide con dirección si incorpora estas subdivisiones o amplía las notas de
las filas existentes para evitar contar dos veces una misma capacidad. F4-08 a F4-11 tienen código montado, cliente y orden SQL versionado; las
pruebas Go del módulo pasan. F4-12 sigue condicionada a la acreditación de
rectificaciones y F4-13/F4-14 solo tienen consumidor local. No se propone
pasar F4-02 a F4-07 a «Hecha». El cero de tareas terminadas no debe
interpretarse como ausencia de código: existe trabajo integrado y probado,
con los límites de montaje y recorrido de este estudio.

## Comprobaciones y entrega

`go test -p 6 ./internal/modules/dietas/...` terminó con salida 0: los 16
paquetes pasaron, sin resultados de caché. Incluye pruebas con dobles de SQL,
HTTP, Documentos y motor de rutas; no equivale a un ensayo PostgreSQL ni a
peticiones nominales de navegador. No se ejecutaron pruebas SQL de Dietas ni actos sobre expedientes; la
restauración, el intento de lista general y la comprobación previa en solo
lectura se describen arriba. La puerta general `GOFLAGS=-p=6 GOMAXPROCS=6
scripts/verificar_calidad.sh` terminó después con salida 0 sobre el mismo
código de main, sin cambios de comportamiento: incluye pruebas Go normales
y con detección de carreras, análisis estático, 4.169 pruebas Node positivas
y las demás guardas. No se repitió la puerta completa.

El índice de código y `gopls` no estaban disponibles en esta sesión. Se
reutilizaron el plan, el inventario de ficheros, la lectura de los consumidores,
el historial y las PR existentes. Una revisión independiente dio GO
documental sobre `e8455b659`; se ajustaron después referencias de línea y se
completó la orden del intento de copia. La revisión confirmó también
`22805490d`, con la clasificación fijada por dirección y el orden SQL de
main. Esos dictámenes no acreditan el recorrido web.

Configuración nueva: ninguna. SQL a instalar: ninguno. La copia temporal
se retiró; solo cambia este documento. La integración en el tablero, su recuento y la
PR conjunta corresponden a C y dirección.
