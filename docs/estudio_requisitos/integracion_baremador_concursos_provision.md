# Integracion del Baremador de puestos singularizados y otros procesos

## Estado

| Campo | Valor |
| --- | --- |
| Fecha de estudio | 16 de julio de 2026 |
| Fuente local | `Baremador` |
| Naturaleza de la fuente | Prototipo y material de estudio con datos personales; permanece excluido de Git, Docker y artefactos. |
| Caso de origen | Concurso general para la provision de puestos singularizados por personal funcionario de carrera. |
| Decision | Usarlo como primer caso real del motor comun de valoracion; reutilizar requisitos, reglas contrastadas y patrones de pantalla, sin copiar el prototipo como modulo productivo. |
| Modulo inicial | Portal interno de provision de puestos y movilidad, conectado con Personal, RPT, RUM, Documentos y Auditoria. |
| Ampliacion | El motor comun servira a otros procesos, pero cada modulo conservara sus bases, requisitos, fases, revision, desempates y resultados. |
| Acredita implementacion | No. Este documento define el porte seguro que debe realizarse por cortes. |
| Puerta productiva | **NO-GO** hasta contrastar bases, implementar reglas gobernadas, autorizacion, trazabilidad y pruebas. |

## Respuesta funcional

El Baremador debe incorporarse al portal. Su primer caso no es una bolsa ni una
OPE: resuelve un concurso de puestos singularizados para personal funcionario
de carrera de la propia Diputacion. La persona solicita puestos ofertados,
indica preferencias y es valorada contra los requisitos y el baremo de las
bases.

No se limitara para siempre a ese concurso. Se extraera la parte realmente
comun para que pueda valorar otros procesos reglados, sin forzarlos a compartir
un unico procedimiento ni convertir el caso singularizado en una plantilla
juridica universal.

Debe compartir con Bolsa:

- persona canonica y perfil de empleado;
- RUM, titulaciones, cursos y evidencias reutilizables;
- periodos de servicios de Personal obtenidos de oficio;
- catalogo profesional, puestos y RPT gobernados;
- aritmetica exacta y operadores seguros del motor de reglas;
- documentos, firma, registro, notificaciones y alegaciones;
- autorizacion, auditoria, idempotencia y trazabilidad;
- sistema de diseno, accesibilidad y ayuda.

El proceso de puestos singularizados debe conservar como propios:

- proceso y bases del concurso de puestos singularizados;
- puestos concretos ofertados y requisitos de cada puesto;
- solicitudes y orden de preferencia;
- admision/exclusion y sus causas;
- reglas sobre nivel, grado, permanencia y experiencia;
- baremo provisional/definitivo;
- algoritmo de adjudicacion y desempates;
- listados, alegaciones, resolucion y toma de posesion, si entra en el alcance.

El mismo hecho puede alimentar puestos singularizados, provision general,
promocion interna, Bolsa u OPE, pero cada valoracion pertenece a su proceso y
version de bases. No se comparte una puntuacion ya calculada.

## Estrategia de ampliacion

La ampliacion se realizara en tres capas:

1. Personal y RUM conservan hechos administrativos y evidencias reutilizables:
   servicios, jornada, puesto, nivel, grado, titulaciones, cursos y documentos.
2. Un nucleo comun ofrece aritmetica exacta, operadores tipados, explicacion,
   huellas, simulacion y reproduccion determinista.
3. Cada modulo proyecta esos hechos conforme a sus bases y gobierna su propio
   expediente, revision, firma, publicacion y recurso.

| Familia de proceso | Reutiliza | Mantiene como especifico |
| --- | --- | --- |
| Puestos singularizados | Personal, RPT, RUM, calculo exacto y preferencias | Nivel/grado, permanencia, requisitos del puesto, cadena de desempate y adjudicacion global. |
| Otras provisiones o movilidad interna | Personal, RPT, RUM y operadores comunes | Modalidad, participantes, meritos, preferencias, efectos y resolucion previstos en sus bases. |
| Bolsa de empleo temporal | Persona, RUM, servicios, documentos y calculo exacto | Categoria, admision, autobaremo, revision, orden de bolsa, llamamientos y vigencia. |
| OPE y procesos selectivos | Persona, RUM, servicios, documentos y calculo exacto | Requisitos de acceso, pruebas, fase de oposicion, fase de concurso, tribunal y resultado selectivo. |
| Promocion interna o estabilizacion | Hechos y operadores ya disponibles | Requisitos de participacion, fases, cupos, reglas y efectos propios de la convocatoria. |

RRHH podra crear nuevas convocatorias y combinaciones soportadas desde el
panel, seleccionando familias de reglas tipadas y catalogos versionados. Una
familia completamente nueva que introduzca semantica juridica no prevista se
incorporara como extension del modulo o nueva familia de regla; no se permitira
ejecutar codigo, SQL o formulas libres desde el navegador.

## Que contiene realmente el prototipo

La aplicacion local es un proyecto Go independiente con SQLite, API REST y web
estatica. Aunque su README usa el nombre generico de concurso de traslados, la
documentacion fuente identifica el caso como concurso de puestos
singularizados de personal funcionario de carrera. Modela:

- candidatos/empleados;
- puestos por numero de orden, centro, denominacion y nivel;
- solicitudes por puesto, preferencia, admision y causas de exclusion;
- periodos de servicio;
- titulaciones y cursos;
- calculo de experiencia, antiguedad, permanencia, grado, titulaciones y
  formacion;
- ranking separado para cada puesto.

La configuracion documentada actualmente contiene una ventana temporal fija,
tramos por diferencia entre nivel desempenado y nivel solicitado, coeficientes,
topes y un total de 100 puntos. Es una transcripcion util de un caso concreto,
no un baremo general ni una autoridad juridica.

## Elementos que se pueden reutilizar

### Como requisitos

- La comparacion entre atributos del empleado y del puesto solicitado.
- La experiencia clasificada por diferencia de nivel.
- La antiguedad por servicios computables.
- La permanencia diferenciada en puesto titular u otras situaciones.
- El grado personal respecto del nivel del puesto.
- Titulaciones y formacion con topes.
- Solicitudes multiples con orden de preferencia.
- Ranking explicable por puesto.
- Necesidad de revisar el desglose de una persona contra cada puesto.

### Como casos de conformidad

Las formulas de la hoja de calculo y del prototipo pueden transformarse en
vectores de prueba solo despues de:

1. localizar la version oficial de las bases que las origina;
2. transcribir literalmente regla, fecha de corte, unidad, redondeo y tope;
3. resolver contradicciones entre PDF, hoja y codigo;
4. anonimizar completamente entradas y resultados;
5. aprobar los resultados esperados con RRHH y el organo competente.

### Como patron de interfaz

Son utiles las vistas de empleados, puestos, detalle de meritos y ranking. La
version definitiva debe convertirlas en un espacio administrativo denso:

- primera pantalla con pendientes, plazos, incidencias y sincronizacion;
- tabla principal de puestos o solicitudes con filtros y cabecera fija;
- puntuacion, estado, preferencia y siguiente accion visibles sin abrir varias
  pantallas;
- panel lateral con resumen, requisitos, evidencias, desglose, comunicaciones,
  historial y auditoria;
- acciones juridicas separadas de la edicion ordinaria, con revision, firma y
  recibo;
- estado, icono y texto, sin depender solo del color;
- funcionamiento con teclado, zoom y anchuras de 1440, 1024 y 390 pixeles.

La apariencia seguira los tokens y temas centrales del portal. El modulo puede
usar un acento propio, pero no CSS o componentes administrativos aislados.

## Lo que no debe portarse literalmente

### Reglas y aritmetica

- Fechas de referencia y comienzo de ventana compiladas en Go.
- Coeficientes, topes, niveles y tipos de titulo fijos en mapas o `switch`.
- `float64` para anos, horas y puntos administrativos.
- Ano convencional de 360 dias aplicado sin referencia versionada a las bases.
- Inclusividad de fechas asumida por codigo.
- Redondeo unico a tres decimales sin declarar en que fase se aplica.
- Resumen manual que queda sustituido por el detalle solo por existir una fila,
  sin procedencia ni reconciliacion administrativa.

### Datos y seguridad

- DNI como clave tecnica, identificador de recurso y segmento de URL.
- Nombre y DNI en respuestas de ranking sin una proyeccion autorizada y
  minimizada.
- Edicion y borrado directo de periodos, cursos o titulos.
- SQLite y ficheros JSON con personas como persistencia del producto.
- Importacion automatica de datos locales al arrancar.
- API sin autenticacion reforzada, RBAC/ABAC, finalidad ni autorizacion por
  campos.
- Ausencia de firma, auditoria append-only, documento de bases, huellas y
  recibos.
- Datos personales del directorio `Baremador`; no se copian a codigo, tests,
  imagenes, commits ni entornos de desarrollo.

### Procedimiento

- Ranking empatado alfabeticamente por nombre. El desempate debe proceder de
  las bases y ser reproducible.
- Ranking aislado por puesto como si equivaliera a adjudicacion final. Un
  concurso con varias preferencias necesita un algoritmo global que asigne
  puestos sin adjudicar dos incompatibles a una persona y respete el orden y
  los desempates publicados.
- Estados libres `admitted`/`excluded` y causas como texto o numeros sin
  catalogo versionado.
- Excluir o puntuar definitivamente por una operacion automatica sin revision,
  motivacion, audiencia/alegacion y firma cuando procedan.
- Borrado de datos que deban permanecer en el expediente historico.

## Arquitectura objetivo

### Modulo inicial de provision y movilidad

El nombre funcional propuesto es `Provision de puestos y movilidad interna`.
Su primer tipo de proceso sera `puestos singularizados`. La clave tecnica
definitiva se decidira junto con el catalogo de modulos, sin reutilizar el
nombre generico `Baremador` como si definiera un unico procedimiento.

Agregados previstos:

| Agregado | Responsabilidad |
| --- | --- |
| `ProcesoProvision` | Bases, puestos ofertados, calendario, reglas, organo, estado y versiones publicadas. |
| `PuestoOfertado` | Referencia exacta a RPT, requisitos, nivel, centro, vacante y condiciones publicadas. |
| `SolicitudProvision` | Empleado, puestos solicitados, preferencias, requisitos, presentacion y expediente. |
| `ValoracionProvision` | Entrada exacta, reglas, desglose, decisiones de revision y puntuacion reproducible. |
| `ListadoProvision` | Admitidos/excluidos, baremo provisional/definitivo, version y firma. |
| `AdjudicacionProvision` | Asignacion global, preferencias, desempates, renuncias y resultado firmado. |

El modulo referencia persona, empleado, RPT, meritos y documentos; no los
duplica. Una rectificacion de Personal o RUM crea otra instantanea y otro
calculo sin reescribir lo ya publicado. Personal no almacena puntos ni marcas
como `computable_para_bolsa` o `computable_para_singularizados`: conserva el
hecho neutral y cada proceso aplica su version exacta de reglas.

### Motor comun, reglas separadas

La existencia de puestos singularizados, Bolsa y futuros procesos justifica
extraer un nucleo compartido muy pequeno para:

- puntos enteros y fracciones exactas;
- unidades, conversiones y redondeos enumerados;
- topes, sumas, minimos, maximos y tablas de decision;
- entradas, desgloses y huellas canonicas;
- simulacion y reproduccion deterministas.

Ese nucleo no conoce OPE, puesto, grado, DNI, convocatoria, HTTP ni base de
datos. Cada modulo define sus familias de regla, valida que se relacionan con
sus bases y gobierna su propio ciclo de publicacion. No se crea un interprete
arbitrario ni una tabla universal de puntos. La extensibilidad procede de
puertos, familias de reglas registrables y configuracion gobernada, no de
condicionales compilados para cada convocatoria.

### Datos obtenidos de oficio

El proceso inicial debe obtener, mediante puertos y con la fecha de corte
exacta:

- identidad y condicion de empleado;
- nombramientos, contratos, ceses y servicios;
- grado personal consolidado y sus efectos;
- puesto titular, ocupaciones provisionales y permanencia;
- grupo/subgrupo, categoria, nivel y centro;
- titulaciones y formacion presentes en RUM o Personal;
- RPT y puestos ofertados publicados.

La persona ve la instantanea y puede pedir rectificacion, pero no edita el
historial de Personal desde el baremador. Una consulta fallida produce
incidencia, no cero puntos.

## Flujo objetivo

1. RRHH crea el proceso y vincula las bases oficiales.
2. Importa o selecciona puestos desde una version exacta de RPT.
3. Configura requisitos y baremo mediante reglas gobernadas.
4. Simula y aprueba casos limite, desempates y adjudicaciones.
5. Revision tecnica, juridica y de proteccion de datos; negociacion cuando
   proceda; firma y publicacion inmutable.
6. El empleado consulta puestos, requisitos y datos obtenidos de oficio.
7. Ordena preferencias, revisa el autobaremo y firma/presenta la solicitud.
8. RRHH revisa requisitos y meritos con decisiones motivadas.
9. Se publica listado provisional, se tramitan alegaciones y se recalcula.
10. Se publica baremo definitivo y se ejecuta una adjudicacion global
    determinista sobre la version cerrada.
11. La resolucion y sus efectos quedan firmados, registrados y auditados.

## Pantallas objetivo

### Portal del empleado

- puestos disponibles y filtros por centro, nivel, area y requisitos;
- comparador de puestos con el perfil propio;
- orden de preferencias accesible y validado;
- meritos reutilizados y datos obtenidos de oficio;
- simulacion/autobaremo por puesto con desglose;
- avisos de dato ausente, conflicto o documento no vigente;
- solicitud, firma, recibo, estado, provisional, alegaciones y definitivo.

### RRHH y organo de valoracion

- cola de procesos, puestos, solicitudes, incidencias y vencimientos;
- tabla de candidatos por puesto y vista global de preferencias;
- filtros por estado, causa, revisor, diferencia de puntuacion y conflicto;
- panel de detalle con datos, evidencia, formula y auditoria;
- aceptacion/rechazo parcial motivado, subsanacion y rectificacion;
- comparador entre version declarada, provisional y definitiva;
- simulador de adjudicacion y prueba de desempates antes de firmar;
- publicacion, retirada/rectificacion y exportacion documental gobernadas.

## Pruebas obligatorias

- Puntuacion para diferencias de nivel positivas, cero, negativas y limites.
- Ventana temporal, fecha de corte, dias extremos y ano bisiesto.
- Jornada completa, media, tercio, cambios y reducciones protegidas segun
  reglas del proceso.
- Periodos solapados y pertenencia a mas de una familia.
- Antiguedad frente a experiencia y permanencia sin doble computo indebido.
- Grado superior, igual, inferior, ausente y rectificado.
- Titulacion requisito que no puntua y titulo reutilizado valido.
- Formacion duplicada, caducada, no relacionada y topada.
- Empates cubiertos por toda la cadena publicada; nunca por nombre implicito.
- Una persona solicita varios puestos con preferencias distintas.
- Dos o mas personas compiten por varios puestos; adjudicacion estable y sin
  duplicidad incompatible.
- Renuncia, exclusion, alegacion y rectificacion con recalculo causal.
- Mismos bytes de entrada y reglas producen mismo desglose y huella.
- Indisponibilidad de Personal/RPT bloquea el dato afectado sin inventar cero.
- Proyecciones no revelan DNI ni informacion de otra persona sin permiso.

## Plan de porte

1. Inventariar y custodiar las fuentes locales sin publicar datos.
2. Obtener y validar con RRHH las bases exactas del concurso de puestos
   singularizados representado.
3. Convertir sus formulas en casos anonimos y detectar diferencias entre hoja,
   codigo y bases.
4. Cerrar primero el nucleo compartido de aritmetica y reglas que tambien usa
   Bolsa.
5. Crear el dominio independiente de Provision y sus pruebas, comenzando por el
   tipo de proceso de puestos singularizados.
6. Integrar Personal, RPT y RUM por referencias e instantaneas.
7. Crear casos de uso, puertos y adaptadores, empezando por memoria para tests y
   PostgreSQL como primer adaptador durable.
8. Construir la web real sobre la API comun, sin copiar la pagina del prototipo.
9. Ejecutar un piloto paralelo con datos anonimizados y reconciliar todos los
   resultados antes de usar expedientes reales.
10. Incorporar despues otras familias de procesos por configuracion o
    extension, usando el mismo nucleo y casos de conformidad propios; nunca por
    copia del baremo de puestos singularizados.

## Conclusion

El Baremador se incorpora como segundo consumidor real del futuro motor de
reglas y como origen de requisitos del modulo de Provision. Su valor esta en el
conocimiento del procedimiento y en los ejemplos, no en reutilizar el binario,
la base SQLite, los JSON personales ni las constantes actuales. Este enfoque
permite automatizar las plazas ofertadas sin contaminar Bolsa, duplicar datos o
crear otro sistema de puntuacion aislado.
