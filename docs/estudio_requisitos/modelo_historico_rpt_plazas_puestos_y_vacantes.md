# Modelo historico de RPT, plazas, puestos, ocupaciones y vacantes

## Estado

| Campo | Valor |
| --- | --- |
| Fecha de estudio | 16 de julio de 2026 |
| Ambito | Diputacion Provincial de Granada y futuros organismos configurados |
| Decision | Separar clasificacion profesional, plaza, puesto, dotacion y ocupacion, con historia bitemporal e importacion gobernada. |
| Consulta requerida | Estado y disponibilidad en cualquier fecha o version; comparacion entre periodos con un boton. |
| Fuente maestra definitiva | Pendiente del diccionario de datos y conectores corporativos de RRHH. |
| Puerta productiva | **NO-GO** para el importador RPT actual hasta reconciliar codigos, dotaciones, plazas y puestos individuales. |

## Respuesta ejecutiva

`Plaza` y `puesto` no son sinonimos en la Diputacion. La plantilla ordena y
presupuesta plazas por escala, subescala, clase o categoria. La RPT describe
puestos de la organizacion, sus condiciones y los requisitos para ocuparlos.
Una persona obtiene o mantiene una plaza y puede ocupar, de forma definitiva o
temporal, un puesto concreto.

La aplicacion no utilizara un unico registro denominado `puesto` para
representarlo todo. Tampoco guardara un booleano `libre`. Para cada fecha
distinguira:

- si la plaza existe, esta dotada, ocupada, vacante, reservada o amortizada;
- si el puesto existe, tiene ocupante, esta reservado, temporalmente sin
  ocupante, bloqueado, suprimido o incluido en un proceso;
- si la unidad puede cubrirse ahora, de forma definitiva o temporal, y por que;
- que datos, actos y versiones sustentan el resultado;
- que informacion falta o esta en conflicto.

Se conservara cada RPT, plantilla y modificacion con fecha de efectos. Nada se
sustituye destructivamente. Una consulta podra responder, por ejemplo:

> En la version RPT vigente el 31 de diciembre de 2024, con la plantilla y los
> actos de personal eficaces ese dia, habia X plazas vacantes, Y puestos sin
> ocupante, Z unidades temporalmente cubribles y N estados no determinables.

La cifra solo se ofrecera si la cobertura de las fuentes es suficiente. Una
RPT publica por si sola no permite saber cuantos puestos estaban realmente
ocupados.

## Terminologia que debe conservar el portal

| Concepto | Significado funcional | Identidad |
| --- | --- | --- |
| `ClasificacionProfesional` | Regimen, grupo/subgrupo, escala, subescala, clase, especialidad o categoria. | Catalogos publicados, versionados y relacionables; no texto libre. |
| `PlazaPlantilla` | Unidad individual de plantilla, ligada a clasificacion y ordenacion presupuestaria. | ID tecnico estable y codigo de plaza de la fuente, con historia de aliases. |
| `PuestoTipoRPT` | Definicion o fila agrupada de la RPT con denominacion y condiciones comunes. | ID tecnico y codigo/clave de la fuente, versionados. |
| `PuestoRPT` | Unidad individual susceptible de ocupacion y provision. | ID tecnico estable y `N PUESTO` o codigo corporativo. |
| `Dotacion` | Numero de unidades de una fila o puesto tipo; no equivale por si solo a una lista de puestos individuales. | Cantidad en la fuente y, cuando exista, expansion reconciliada a unidades. |
| `VinculoPlazaPuesto` | Relacion efectiva entre una plaza y un puesto durante un intervalo. | Versionada; no se presume uno a uno para toda la historia. |
| `OcupacionPuesto` | Desempeno de un puesto por una relacion de servicio, con modalidad y fechas. | Privada, bitemporal y referenciada por actos administrativos. |
| `NumeroOrdenProceso` | Posicion de un puesto dentro del anexo o solicitud de una convocatoria. | Solo vale dentro de ese proceso y version; nunca sustituye al codigo de puesto. |

La interfaz usara las palabras oficiales de la Diputacion. Los nombres
tecnicos anteriores son una propuesta de dominio y se ajustaran cuando RRHH
entregue el diccionario de datos corporativo.

Un codigo de cinco cifras como `56432` es compatible con el patron observado
en codigos individuales de plaza, pero no se clasificara por longitud. Se
conserva como cadena opaca con organismo y sistema de origen y se consulta en
la fuente corporativa. Los ceros iniciales, cambios y reutilizaciones no pueden
perderse por convertirlo a numero.

## Evidencias oficiales encontradas

### RPT publicada

La [RPT de la Diputacion vigente publicada en Transparencia](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Recursos-Humanos-Relacion-de-Puestos-de-Trabajo/Relacion-de-Puestos-de-Trabajo.pdf)
esta fechada el 7 de mayo de 2026. Organiza filas por Delegacion y Centro y
publica, entre otros, `Denominacion/Datos de reserva`, `Dot.`, tipo de puesto,
adscripcion, forma de provision, grupo, escala, categoria, complemento de
destino, complemento especifico, condiciones y observaciones.

Muchas filas tienen una dotacion superior a uno. Por ello, una fila del PDF no
demuestra que exista un unico puesto individual ni ofrece por si sola su estado
de ocupacion.

### Plantilla y diferencia con la RPT

El [expediente oficial de Plantilla de Personal 2025](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/AREAS-Participacion-Publica-Normativa-Documentos-en-exposicion-publica-RRHH/Plantilla_Personal_2025.pdf)
declara expresamente que plantilla y RPT son instrumentos diferentes. Su anexo
resume `Denominacion de la Plaza`, numero de plazas, grupo y observaciones,
encuadradas por escala, subescala y clase. No describe las caracteristicas
esenciales de cada puesto ni enumera todos los codigos individuales de plaza.

La [Plantilla de Personal 2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/AREAS-Participacion-Publica-Normativa-Documentos-en-exposicion-publica-RRHH/Anuncio_exp._pub._plantilla_anual-2026.report.pdf)
aporta casos especialmente claros: identifica codigos individuales de plaza de
cinco cifras y, al ordenar cambios correlativos de la RPT, identifica puestos
individuales distintos con codigos de cuatro cifras. Tambien contempla plazas
que se transforman cuando queden vacantes y puestos no singularizados que se
crean o suprimen como efecto de cambios de plantilla. El modelo debe poder
representar todos esos actos y condiciones, no solo una foto anual.

### Uso simultaneo de codigo de plaza y puesto

La [Circular de solicitudes de personal temporal de 19 de febrero de 2026](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/2026_Cicular_Peticiones_de_personal.pdf)
pide, cuando existe plaza vacante, indicar por separado `codigo de plaza y
puesto (RPT)`. Es una evidencia directa de que el sistema debe conservar ambos
identificadores y su relacion.

### Puestos individuales en procesos de provision

Las [bases y Anexo I del concurso general publicado en marzo de 2025](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Documentos-Provision-de-Puestos/2025_PPT_01000046_B.O.P.pdf)
distinguen `ORDEN`, `CENTRO`, `DENOMINACION` y `N PUESTO`. El numero de orden
pertenece a la convocatoria; `N PUESTO` identifica la unidad ofertada. Las
resoluciones de adjudicacion publicadas incluyen ademas codigo de plaza, sin
que deba copiarse al portal ningun dato personal del adjudicatario.

Estas fuentes permiten fijar la separacion conceptual, pero no bastan para
reconstruir toda la relacion historica entre cada plaza, puesto, ocupante y
acto. Hace falta un conector corporativo y un diccionario aprobado por RRHH.

### El primer numero del PDF consolidado

El numero situado al comienzo de `Denominacion/Datos de reserva` no debe
denominarse codigo individual de puesto. Se ha contrastado un mismo puesto
singularizado entre la RPT consolidada y un anexo oficial de provision: el PDF
consolidado muestra un numero de tres cifras y la convocatoria un codigo de
puesto individual de cuatro cifras, en columnas que separan ademas orden y
codigo.

No se ha localizado una leyenda oficial que defina ese primer numero. Hasta
recibir el diccionario de RRHH se conservara literalmente como
`codigo_datos_reserva_fuente`, con documento, pagina y fila, sin inferir su
semantica.

### Areas funcionales y categorias

El [Acuerdo de areas funcionales publicado en febrero de 2026](https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1771200040625-final-3bd316f3.pdf)
define areas y categorias relacionadas y preve altas, bajas y cambios de
denominacion. Es una fuente oficial versionada que debe reconciliarse con la
plantilla, la RPT y el catalogo interno. Las 58 categorias recuperadas de OPES
son utiles para demostracion, pero no sustituyen esa autoridad ni deben
fusionarse por similitud textual.

## Hallazgo sobre el importador actual

El script `scripts/import_rpt_pdf.py` extrae 842 filas y comprueba una suma de
1.714 dotaciones. Genera un elemento `RPTPosition` por fila, trata el numero al
inicio de `Denominacion/Datos de reserva` como `official_code` y, si se repite,
inventa una clave combinando codigo, centro y secuencia.

Ese resultado es util como lectura provisional del PDF, pero no es un
inventario de 1.714 puestos individuales:

- una fila agrupada y una dotacion no son una unidad individual;
- la clave desduplicada no es un codigo oficial de plaza ni de puesto;
- `replace: true` borra la foto anterior en vez de crear historia;
- `RPTPosition` mezcla puesto tipo, puesto individual, categoria y dotacion;
- `state` y `coverage` son textos libres;
- el PDF no contiene ocupaciones, reservas, actos ni vacantes individuales;
- una edicion o borrado CRUD puede reescribir el pasado;
- los conteos no se reconcilian con plantilla, puestos individuales y fuente
  corporativa.

El analisis de la salida actual encuentra ademas columnas desplazadas, valores
compuestos imposibles y cientos de grupos o categorias vacios. La version de
salida incorpora la fecha de generacion y por ello los mismos bytes de entrada
no producen necesariamente el mismo artefacto. El fichero heredado
`config/rpt_positions_import.json` conserva tambien una ruta local del equipo;
debe regenerarse o retirarse antes de una publicacion del repositorio.

Hasta corregirlo, su salida es `preparacion_no_autoritativa`. No puede alimentar
un proceso de provision, una baremacion por puesto ni un informe de vacantes.

## Historia bitemporal

Cada hecho relevante conserva dos ejes temporales:

1. **Tiempo de validez administrativa:** desde y hasta cuando produjo efectos
   la RPT, plaza, puesto, ocupacion o acto.
2. **Tiempo de conocimiento del sistema:** desde y hasta cuando el portal
   conocio esa version, incluida una rectificacion recibida mas tarde.

Esto permite dos preguntas diferentes:

- `vigente_en=2024-12-31`: reconstruir los hechos eficaces ese dia con la
  informacion consolidada actual;
- `vigente_en=2024-12-31&conocido_en=2025-01-15`: reproducir lo que el sistema
  podia afirmar el 15 de enero de 2025.

Las fechas de aprobacion, publicacion y efectos se conservan por separado. No
se deduce que una modificacion entra en vigor el dia del PDF. Si la fuente no
expresa fecha de efectos se abre una incidencia y no se adivina.

Una version historica nunca se edita. La rectificacion crea una sucesora que
referencia version y huella sustituidas y explica el motivo. Los codigos de la
fuente son atributos historicos, no claves primarias; si un sistema los cambia
o reutiliza, el ID tecnico y la genealogia evitan mezclar unidades.

## Agregados y relaciones propuestos

### `VersionRPT`

- administracion y organismo;
- numero/version oficial y predecesora;
- aprobacion, publicacion y periodo de efectos;
- documentos, CSV, firmas y huellas;
- fuente, importacion, recibo y esquema;
- estado: preparacion, reconciliada, aprobada, publicada, sustituida o
  retirada;
- conteos declarados y reconciliados;
- incidencias y aprobaciones maker-checker.

### `PuestoTipoRPT`

Conserva en cada version la fila publicada: centro, denominacion, dotacion,
tipo, adscripcion, provision, grupo, escala, categoria, nivel, complementos,
jornada/condiciones, titulacion, formacion, requisitos y observaciones
catalogadas. Una nueva RPT crea otra version efectiva, aunque visualmente no
haya cambios.

### `PuestoRPT`

Representa una unidad individual. Se relaciona con el puesto tipo exacto y
conserva codigo corporativo, centro, estado estructural y genealogia. Una
division, fusion, traslado de centro, reclasificacion o supresion se expresa
como una transicion versionada; no como cambio silencioso de columnas.

### `VersionPlantilla` y `PlazaPlantilla`

La version de plantilla conserva documento, ejercicio, modificaciones y
fechas. Cada plaza individual tiene clasificacion, codigo corporativo,
dotacion/presupuesto, estado y sucesion. El resumen publico por categorias es
una proyeccion; no reemplaza el inventario corporativo.

### `VinculoPlazaPuesto`

Relaciona plaza y puesto en un intervalo con fuente, acto y huella. Admite que
el vinculo cambie en la historia y que haya unidades pendientes de conciliar.
No obliga a una cardinalidad universal hasta validar el modelo real de RRHH.

### `OcupacionPuesto`

Relaciona de forma privada una relacion de servicio con plaza y puesto,
modalidad, titularidad, fecha de inicio/fin, reserva y actos de alta/cese. La
proyeccion de vacantes no necesita revelar la identidad de la persona.

### `TitularidadPlaza` y `Reserva`

La titularidad de la plaza no se confunde con la ocupacion efectiva de un
puesto. Ambas relaciones pueden coexistir con una reserva mientras la persona
ocupa provisionalmente otro destino. Titularidad, reserva, ocupacion y cese
tienen intervalos y fuentes separados; las proyecciones solo reciben el efecto
minimo necesario.

## Vacante, no ocupado y cubrible

No existira un campo manual `libre=true`. Se calcula una
`InstantaneaDisponibilidadDotaciones` a partir de versiones y eventos exactos.

### Estado de plaza

- vigente y ocupada;
- vigente y vacante;
- cubierta interina o temporalmente;
- reservada;
- sin dotacion presupuestaria o bloqueada;
- incluida en proceso de cobertura;
- amortizada/suprimida;
- indeterminada por datos incompletos o contradictorios.

### Estado de puesto

- ocupado definitivamente;
- ocupado provisional o temporalmente;
- sin ocupante;
- reservado a una persona o relacion;
- temporalmente desatendido con sustitucion posible;
- ofertado/adjudicado y pendiente de toma de posesion;
- bloqueado por acto, litigio, reorganizacion o incidencia;
- suprimido;
- indeterminado.

### Disponibilidad de cobertura

- cubrible definitivamente;
- cubrible temporalmente;
- no cubrible por reserva, bloqueo, falta de dotacion o proceso en curso;
- requiere decision de RRHH;
- no determinable.

Jubilacion, cese, fallecimiento, renuncia, traslado, excedencia, comision de
servicios, permiso, incapacidad, suspension, reserva, creacion, amortizacion y
otros motivos proceden de un catalogo versionado. El nombre del motivo no
decide por si solo el resultado. Por ejemplo, una excedencia puede conservar
reserva o no segun su modalidad y acto; el conector aporta el efecto juridico
minimo atestado sin exponer una causa sensible.

La jubilacion produce un cese efectivo, pero la plaza solo se mostrara vacante
tras comprobar que sigue vigente, dotada y no esta cubierta, reservada,
amortizada o afectada por otra transicion.

## Consulta y boton de comparacion

La pantalla interna `Plantilla y RPT historica` permitira seleccionar:

- organismo y ambito organizativo;
- fecha exacta o version RPT;
- version de plantilla compatible;
- fecha de conocimiento, por defecto `ahora`;
- plazas, puestos, puestos tipo o dotaciones;
- vacante, sin ocupante, cubrible definitiva/temporalmente, reservada,
  bloqueada o indeterminada;
- regimen, categoria, grupo, centro, tipo, forma de provision y motivo;
- comparacion contra otra fecha o version.

El boton `Calcular disponibilidad` devuelve:

- totales y denominadores;
- desglose por estado, categoria, centro y motivo minimizado;
- diferencias: altas, bajas, ocupaciones, vacantes, reservas, cambios y
  unidades no reconciliadas;
- versiones RPT y plantilla, fecha de corte y reglas usadas;
- cobertura de cada fuente y antiguedad de sincronizacion;
- incidencias que impiden afirmar una cifra;
- huella de la consulta y enlace al detalle autorizado.

`Comparar 2024 con ahora` no usa automaticamente el 1 de enero ni la fecha del
fichero. La persona elige la fecha de 2024 o una version y la interfaz muestra
la fecha de efectos tomada. Para comparaciones repetidas RRHH puede publicar
un hito catalogado, por ejemplo `cierre_plantilla_2024`.

El resultado puede materializarse en PDF firmado y en CSV/JSON, con el mismo
manifiesto, versiones, filtros, conteos y huella. Las partes autorizadas pueden
descargarlo conforme a la politica documental; el detalle individual no se
publica por defecto.

## Encaje en la arquitectura y puertas de garantia

Este dominio pertenece al modulo `Personal/Organizacion`. No se implementa en
el adaptador HTTP, en el Baremador ni como tablas compartidas por todos los
modulos. El dominio define identidades, intervalos, versiones, relaciones y
consultas; la aplicacion orquesta importacion, reconciliacion y publicacion; los
adaptadores conectan PDF, BOP, sistema corporativo y PostgreSQL.

Bolsa, Provision, Nominas, Cronos y el archivo documental consumen
proyecciones o eventos mediante puertos. Ninguno lee directamente las tablas de
Personal ni modifica su historia. Un cambio de base de datos o de sistema
maestro requiere otro conector, no una reescritura del nucleo.

Antes de habilitar datos reales se exigen:

- prueba de arquitectura hexagonal y dependencias permitidas;
- clasificacion de la informacion y analisis de riesgos ENS;
- metadatos, expediente, documento e indice conforme a ENI cuando proceda;
- registro de actividades, minimizacion, plazos, derechos y EIPD conforme a
  RGPD y LOPDGDD cuando el DPD la determine;
- autorizacion de denegacion por defecto, RBAC/ABAC y separacion interno/externo;
- cifrado, custodia de claves, copias, restauracion, continuidad y auditoria;
- pruebas unitarias, de propiedades, carrera, integracion, recuperacion y
  seguridad;
- validacion funcional de RRHH y aprobacion de Sistemas, Secretaria,
  Intervencion, Asesoria Juridica y DPD dentro de sus competencias.

El panel, CLI, API y MCP solo llaman casos de uso autorizados. Una IA no decide
si una plaza esta juridicamente vacante o cubrible ni accede al historial
nominal sin una capacidad positiva y una finalidad acreditada.

## Fuentes y conectores

| Puerto | Finalidad |
| --- | --- |
| `FuenteRPTPublicada` | Importar PDF, hoja o datos abiertos oficiales como evidencia de la version publicada. |
| `FuentePlantillaPublicada` | Importar plantilla anual y modificaciones. |
| `FuenteMaestraOrganizacionRRHH` | Obtener puestos y plazas individuales, codigos y relaciones desde el sistema corporativo. |
| `FuenteOcupacionesPersonal` | Obtener titularidades, ocupaciones, reservas y fechas efectivas. |
| `FuenteActosPersonal` | Obtener altas, ceses y efectos necesarios sin difundir documentos o causas sensibles. |
| `FuenteProcesosProvision` | Identificar unidades ofertadas, reservadas, adjudicadas o pendientes de toma de posesion. |
| `RepositorioHistoriaRPTPlantilla` | Conservar versiones, genealogia, intervalos, huellas e incidencias. |
| `ConsultaDisponibilidadDotaciones` | Producir la instantanea historica explicable sin conocer el adaptador concreto. |

PostgreSQL sera el primer adaptador durable. Oracle, PeopleNet, hoja
corporativa, API o fichero son adaptadores de entrada; no cambian el dominio.
Active Directory puede ayudar a resolver identidad y unidad, pero no se
considera fuente maestra de plazas, puestos o actos sin validacion expresa.

## Flujo de importacion gobernada

1. Descargar o recibir la fuente en zona de preparacion y conservar bytes,
   firma, CSV, huella, fecha y procedencia.
2. Detectar esquema y convertir a un contrato de preparacion sin inventar
   identificadores.
3. Validar diccionario, tipos, catalogos, fechas, duplicados y limites.
4. Reconciliar filas, dotaciones, plazas, puestos individuales, centros y
   conteos entre fuentes.
5. Calcular diferencias contra la version anterior, incluidas divisiones,
   fusiones, altas, amortizaciones y cambios de condicion.
6. Mostrar incidencias; ninguna ausencia se convierte en cero ni en baja.
7. Revision funcional por RRHH y, para cambios sensibles, doble control.
8. Aprobar, firmar y publicar atomica e inmutablemente la nueva version y su
   manifiesto.
9. Recalcular proyecciones afectadas y emitir eventos; no reescribir resultados
   de procesos ya cerrados.

El conector automatico puede preparar una version y proponer enlaces, pero no
resolver en silencio una colision de codigos o una diferencia juridica.

## Privacidad y seguridad

- La RPT y la plantilla publicables se separan de ocupaciones y actos de
  personal, que permanecen en la superficie interna.
- Los cuadros de vacantes no muestran nombre, DNI, causa medica, familiar o
  disciplinaria ni referencia que permita inferirlos.
- Los motivos tienen etiqueta interna y proyeccion minimizada; el documento
  fuente mantiene su clasificacion y permisos.
- Toda consulta, detalle, exportacion y descarga se autoriza por finalidad y se
  audita.
- El portal externo solo publica puestos, plazas, ofertas o agregados que una
  politica y un acto hayan marcado expresamente como publicos.
- Una IA puede consultar agregados publicos o una proyeccion interna autorizada,
  pero no obtener el historial personal que sustenta una vacante.

## Pruebas de aceptacion

1. La misma categoria puede corresponder a muchas plazas y puestos sin
   colisionar identidades.
2. Una fila con `Dot.=19` no se importa como un unico puesto ni se expande a 19
   codigos inventados.
3. Numero de orden, codigo de puesto, codigo de plaza e ID tecnico nunca se
   intercambian.
4. Una modificacion conserva consultable la version anterior y el documento
   que la aprobo.
5. Consultar por fecha administrativa y por fecha de conocimiento reproduce
   resultados diferentes cuando existe una rectificacion tardia.
6. Cambio de categoria, centro, nivel, complemento, jornada o requisitos crea
   una nueva version efectiva del puesto.
7. Plaza amortizada y puesto suprimido no aparecen como vacantes cubribles.
8. Puesto sin ocupante pero reservado no aparece como libre para provision
   definitiva.
9. Excedencia con y sin reserva produce estados distintos sin exponer su causa.
10. Jubilacion seguida de interinidad no cuenta como puesto sin ocupante, pero
    conserva la situacion de plaza que corresponda.
11. Adjudicacion pendiente de toma de posesion se distingue de vacante no
    comprometida.
12. Fuente de ocupaciones caida produce estado no determinable, no una lista
    mas grande de vacantes.
13. La suma por estados mas indeterminadas coincide con el universo exacto de
    la consulta.
14. Comparar dos versiones explica cada alta, baja y cambio y conserva la misma
    huella al repetir entradas identicas.
15. PDF, CSV y JSON de una consulta contienen los mismos filtros, versiones,
    conteos y manifiesto.
16. Una persona sin permiso no puede enumerar unidades ni inferir causas por
    errores, tiempos o descargas.

## Datos que debe aportar RRHH/Sistemas

Antes de implementar el conector productivo se necesita:

1. diccionario de cada columna y abreviatura de la RPT;
2. significado y unicidad de todos los codigos publicados e internos;
3. ejemplos anonimizados de codigo de plaza, puesto, dotacion y ocupacion;
4. sistema maestro y mecanismo de lectura para plazas, puestos y personas;
5. historia disponible desde al menos 2024 y politica para cargar periodos
   anteriores;
6. catalogo de actos y efectos sobre reserva, vacante y cobertura;
7. reglas de relacion entre plantilla, RPT, presupuesto y nomina;
8. organismos incluidos y posibles secuencias de codigos independientes;
9. responsables de reconciliacion, aprobacion y firma;
10. definicion institucional de los indicadores que se publicaran.

Sin esos datos podemos construir y probar el nucleo temporal, el importador de
preparacion y la interfaz con datos anonimos, pero no certificar el numero real
de plazas o puestos libres de 2024 ni de hoy.

## Normativa y fuentes

- [Real Decreto Legislativo 5/2015, Estatuto Basico del Empleado Publico](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719),
  en especial ordenacion de puestos, oferta de empleo y provision.
- [Ley 7/1985, Reguladora de las Bases del Regimen Local](https://www.boe.es/buscar/act.php?id=BOE-A-1985-5392),
  articulo 90 sobre plantilla y relaciones de puestos.
- [Real Decreto Legislativo 781/1986](https://www.boe.es/buscar/act.php?id=BOE-A-1986-9865),
  articulos 126 y 127 sobre plantilla local y publicacion.
- [Relacion de Puestos de Trabajo de la Diputacion de Granada](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/relacion-de-Puestos-de-trabajo/).
- [Procesos de seleccion y provision de puestos de la Diputacion](https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/procesos-de-seleccion-de-personal/).

La definicion final de vacante, reserva y posibilidad de cobertura debe ser
validada por RRHH, Intervencion, Secretaria y Asesoria Juridica para cada
indicador. Este documento fija el modelo tecnico y las salvaguardas; no crea
por si solo un efecto sobre una plaza o puesto.
