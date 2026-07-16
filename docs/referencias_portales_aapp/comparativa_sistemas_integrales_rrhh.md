# Comparativa de sistemas integrales de Recursos Humanos públicos

Fecha de corte: **16 de julio de 2026**.

Estado: estudio de referencia funcional. Las capacidades descritas proceden de
fuentes oficiales públicas, pero no acreditan por sí solas que cada función siga
desplegada en todos los organismos ni convierten una práctica externa en norma
aplicable a la Diputación de Granada.

Este documento amplía la [comparativa de portales de selección, bolsas y
autoservicio](comparativa_y_composicion_recomendada.md). Su finalidad es decidir
qué patrones conviene adoptar para el portal integral de RRHH y cuáles deben
mejorarse o descartarse.

## 1. Resultado ejecutivo

No se copiará una solución completa. La composición de referencia será:

- **VEPA y FIDES** para expediente de méritos, obtención de oficio,
  reutilización, subsanación y autobaremación;
- **SIGP, Funciona y Registro Central de Personal** para expediente canónico y
  procedimientos especializados federados;
- **TRAMA y GERHONTE** para jornada, incidencias, turnos y conexión con nómina;
- **MAGMA** para la relación empleado-responsable, delegaciones y bandejas;
- **SILME** para RPT, plantilla, presupuesto, trazabilidad, migración y salida
  del proveedor;
- las necesidades, sistemas y normas de la **Diputación de Granada** como
  autoridad funcional real del producto.

La solución propia mejorará esos referentes mediante reglas versionadas y no
embebidas en código, eventos inmutables de rectificación, mínimo privilegio por
finalidad, trazabilidad resistente a manipulación, expediente bitemporal,
conectores sustituibles y una experiencia coherente sin mezclar zonas de
confianza.

## 2. Método y límites de la comparación

Se han usado manuales, normas, fichas de servicio, resoluciones y pliegos
publicados por sus organismos. La leyenda de la matriz significa:

- `Sí`: capacidad acreditada en una fuente pública oficial;
- `Parcial`: distribuida, limitada o atendida por otro sistema;
- `N/V`: no verificable en la documentación pública consultada; no significa
  necesariamente que la capacidad no exista.

| Sistema | Expediente | RPT y plantilla | Nómina | Jornada | Carrera, provisión y formación | Selección y bolsas | Autoservicio y responsable | PRL, igualdad y analítica | Interoperabilidad |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Junta: SIRHUS + VEPA | Sí | Sí | Sí | N/V | Sí | Sí | aspirante: sí; responsable: N/V | N/V | Sí |
| SAS: GERHONTE + e_atención + VEC | Sí | Parcial | Sí | Sí | Sí | Sí | empleado: sí; responsable: N/V | Parcial | Sí, fragmentada |
| AGE: SIGP + Funciona + RCP + NEDAES + TRAMA | Sí | Sí | Sí | Sí | Sí | Parcial | Sí | Parcial | Sí, extensa |
| Xunta: FIDES + PORTAX | Sí | Parcial | Parcial | Sí | Sí | Sí | empleado: sí; responsable: N/V | N/V | Sí |
| Madrid: MAGMA | Sí | N/V | consulta | Sí | N/V | N/V | empleado y responsable: sí | Parcial | Parcial |
| Diputación de Granada actual | Parcial | N/V | Parcial | WCRONOS | N/V | CONVOCA/GINPIX | varios portales | N/V | por inventariar |
| SILME 2024, requisitos contractuales | Sí | Sí | Sí | Sí | Sí | opcional | Sí | Sí | Sí |
| Cataluña: ATRI | Sí | N/V | consulta | Sí | formación | N/V | empleado: sí | N/V | Parcial |

La matriz compara cobertura, no calidad, conformidad, apertura del código ni
facilidad de reutilización.

## 3. Sistemas estudiados

### 3.1 SIRHUS y VEPA, Junta de Andalucía

SIRHUS es el sistema corporativo para planificación, puestos, selección,
provisión, situaciones administrativas, nómina y acción social. La Junta sigue
publicando formación de tramitación de actos de personal en SIRHUS en 2025.

VEPA aporta el mejor referente público próximo para Bolsa:

- currículo y repositorio común de méritos;
- experiencia, formación, titulaciones y pruebas procedentes de SIRHUS;
- separación entre información oficial y declarada;
- firma, registro, pago y presentación electrónica;
- información sobre inclusión y puntuación de los méritos;
- consulta mediante Cl@ve, certificado, @firma y AutoFirma;
- reutilización de documentos en procesos posteriores.

Se adoptará el expediente de méritos reutilizable. No se copiarán estados que
se presentan como irreversibles, retrasos de sincronización sin procedencia
visible ni dependencia de un único entorno de escritorio. Una revocación o
rehabilitación será un nuevo acto motivado, no una sobrescritura.

Fuentes:

- [Norma consolidada de SIRHUS](https://ws040.juntadeandalucia.es/sedeboja/lconsolidada/eli/es-an/o/1999/09/24/%281108731653%29/dof/19991004/spa/html/LE0000030677_19991004.html).
- [Formación SIRHUS 2025](https://www.juntadeandalucia.es/sites/default/files/2025-04/Ficha_PF25PF-PHP7%20Tramitaci%C3%B3n%20Actos%20Personal%20Sirhus.pdf).
- [Manual general VEPA 2026](https://portalempleopublico.juntadeandalucia.es/vepa/cvdigital/faces/javax.faces.resource/manuales/ManualDeUsuario.pdf).
- [Manual de bolsas VEPA 2026](https://portalempleopublico.juntadeandalucia.es/vepa/cvdigital/faces/javax.faces.resource/manuales/ManualSeleccion_Personal_Bolsa.pdf).

### 3.2 GERHONTE, e_atención y VEC, Servicio Andaluz de Salud

GERHONTE reúne expediente, plantilla, nómina, turnos, ausencias, desempeño,
relaciones laborales, fiscalización y formación. e_atención muestra nóminas,
IRPF, carrera, prevención y documentos con capacidades distintas según el
método de acceso. VEC gestiona currículo, selección, bolsas y carrera, incluida
la presentación electrónica de autobaremos.

Se adoptará la separación de dominios y la adaptación de servicios a la
relación real de la persona. Se evitará su principal coste para el usuario: una
experiencia fragmentada entre distintas aplicaciones. Los módulos propios
compartirán navegación, identidad personal y bandeja, sin compartir por ello
sesión o permisos entre la zona pública y la interna.

Fuentes:

- [Descripción oficial de GERHONTE](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/ayudadigital/aplicaciones/recursos-humanos/gerhonte).
- [e_atención al profesional](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/ayudadigital/aplicaciones/recursos-humanos/e-atencion-al-profesional).
- [Seguridad y acceso a e_atención](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/atencion-al-profesional/eatencion-al-profesional/seguridad-y-acceso).
- [Ventanilla Electrónica de Profesionales](https://www.sspa.juntadeandalucia.es/servicioandaluzdesalud/profesionales/ventanilla-electronica-de-profesionales).
- [Procedimiento de carrera de 2026](https://www.juntadeandalucia.es/boja/2026/127/18).

### 3.3 SIGP, Funciona, RCP, NEDAES y TRAMA, Administración General del Estado

La AGE ofrece el patrón más claro de federación de servicios:

- SIGP tramita nombramientos, ceses, situaciones, antigüedad, grado,
  reingresos, compatibilidades, concursos, formación, acción social y RPT;
- Funciona actúa como punto de entrada del empleado;
- el Registro Central de Personal mantiene datos personales y administrativos
  canónicos;
- NEDAES gestiona nómina;
- TRAMA trata marcajes, incidencias, permisos, horarios, turnos, delegaciones y
  validación jerárquica;
- AutenticA aporta acceso y atributos organizativos comunes.

Se adoptará un registro canónico y servicios especializados conectados por
contratos. Se evitarán la multiplicación de portales y las fronteras que el
usuario no pueda entender. El catálogo funcional consultado es histórico y no
se usa como prueba de una interfaz actual; el panel oficial de mayo de 2026 sí
acredita actividad de varios de estos servicios.

Fuentes:

- [Catálogo de servicios de Administración Digital](https://administracionelectronica.gob.es/dam/jcr%3A5d070807-d512-4452-b080-0c766681d651/Catalogo_servicios_administracion_digital_v2.pdf).
- [Panel oficial de actividad, mayo de 2026](https://dataobsae.administracionelectronica.gob.es/cmobsae3/dashboard/Dashboard.action?selectedLevel=L0&selectedScope=A2&selectedTemporal=31%2F05%2F2026&selectedTemporalScope=2&selectedUnit=TOTAL).
- [Novedades de TRAMA](https://listas-ctt.administracionelectronica.gob.es/pipermail/trama-avisos/2023-March/000046.html).

### 3.4 FIDES y PORTAX, Xunta de Galicia

FIDES aporta el mejor modelo de estados del expediente de méritos:

- experiencia, idiomas, formación, titulaciones, docencia y otros méritos;
- originales electrónicos, copias auténticas o documentos verificables;
- presentación individual o por lotes;
- estados pendiente, en validación, validado, duplicado, incompleto, no
  catalogable y descartado;
- consulta de titulaciones y formación en fuentes oficiales;
- experiencia interna obtenida del repositorio central;
- subsanación de evidencia incompleta.

PORTAX permite consultar historia administrativa, puestos, reservas, grado,
cuerpos, trienios, permisos, nóminas, fichajes, flexibilidad y teletrabajo.
Parte de la información es histórica y remite de nuevo a FIDES.

Se adoptarán los estados explícitos y la obtención de oficio. Se añadirán
procedencia y fecha visibles, así como eventos inmutables de revocación,
rehabilitación, rectificación y fecha de efectos.

Fuentes:

- [Manual de actualización de méritos FIDES](https://ficheiros-web.xunta.gal/aplicacions/fides/manual-actualizacion-meritos.pdf).
- [Expediente administrativo del personal](https://www.xunta.gal/funcion-publica/expediente-administrativo).
- [Instrucción de funcionamiento](https://www.xunta.gal/dog/Publicados/2023/20231215/AnuncioG0597-051223-0010_es.html).
- [Manual de PORTAX](https://ficheiros-web.xunta.gal/portax/PORTAX_MO_manualUsuario_cas.pdf).

### 3.5 MAGMA, Comunidad de Madrid

MAGMA diferencia área personal y área restringida del responsable. Soporta
delegación por unidad y periodo, validaciones en varios niveles, consulta del
equipo, turnos, vacaciones, permisos, calendarios, contadores y seguimiento de
solicitudes.

Se adoptarán la bandeja orientada a tareas y las delegaciones temporales. Se
aplicará mayor minimización: un responsable solo verá datos necesarios para
resolver o planificar una finalidad concreta. Una solicitud ya resuelta se
rectificará mediante una versión relacionada, sin obligar a cancelar el rastro
anterior.

Fuente: [Portal del empleado de Atención Primaria](https://www.comunidad.madrid/hospital/atencionprimaria/profesionales/recursos-humanos/portal-empleado).

### 3.6 SILME y entidades locales de Menorca

El pliego de 2024 es el referente local más completo localizado. Describe como
requisitos:

- expediente, contratos, nombramientos, nómina y Seguridad Social;
- RPT, plantilla, plazas, puestos e historia;
- presupuesto del capítulo I y simulación de escenarios;
- formación, jornada, turnos, festivos y horas extraordinarias;
- portales de empleado y coordinador;
- formularios, delegaciones, sustituciones y alertas configurables;
- igualdad, estadísticas por sexo y cuadros de mando;
- conexiones con contabilidad, Delt@, Contrat@, Certific@ e ISPA;
- auditoría de accesos y cambios;
- continuidad, exportación, devolución y borrado de datos;
- tres meses de ejecución paralela de nómina durante la migración.

Un pliego acredita requisitos, no que todas las capacidades estén implantadas.
Se adoptarán la cobertura, la salida del proveedor y la ejecución paralela de
nómina. Se mejorará el control de informes: no habrá consultas libres sobre
cualquier columna, sino vistas gobernadas, seguridad por fila y columna,
finalidad declarada y auditoría de las exportaciones. Tampoco se dependerá del
proveedor para incorporar cada nuevo conector.

Fuente: [Pliego SILME 2024](https://contrataciondelestado.es/wps/wcm/connect/PLACE_es/Site/area/docAccCmpnt?amp%3BDocumentIdParam=5081dc50-13f7-4fa3-8943-df8261b5d4ab&amp%3Bcmpntname=GetDocumentsById&amp%3Bsource=library&srv=cmpnt).

## 4. Flujos compuestos para el producto objetivo

Estos flujos son decisiones de diseño. Las condiciones materiales de cada caso
procederán siempre de la norma, convenio, acuerdo, RPT, bases o resolución
aplicables y de su versión vigente.

### 4.1 Expediente personal canónico

```text
persona
→ identidades y procedencias
→ relaciones de servicio
→ plazas, puestos y ocupaciones
→ situaciones
→ documentos y evidencias
→ actos administrativos firmados
→ vistas minimizadas por finalidad
```

### 4.2 Méritos y autobaremación

```text
aportación o consulta interoperable
→ validación documental
→ incorporación al expediente de méritos
→ instantánea congelada para la convocatoria
→ aplicación de sus bases versionadas
→ autobaremo ciudadano
→ revisión técnica y aceptación o rechazo parcial
→ alegaciones y rectificaciones
→ resultado administrativo firmado
```

Se separarán cuatro conceptos que otros sistemas suelen mezclar:

1. autenticidad e integridad del documento;
2. reconocimiento administrativo del mérito;
3. aplicabilidad de ese mérito en una convocatoria;
4. puntuación conforme a las bases de esa convocatoria.

Una rectificación no borrará nada. Identificará acto rectificado, autoridad,
motivo, evidencia, firma y fecha de efectos, y provocará una nueva ejecución
reproducible del baremo cuando corresponda.

### 4.3 RPT, plazas, puestos y ocupaciones

```text
RPT versionada
→ plaza presupuestaria
→ puesto funcional e individual
→ requisitos y condiciones
→ ocupación, reserva o vacante
→ relación de servicio
→ efecto presupuestario
```

La historia bitemporal permitirá responder qué existía, qué se sabía y qué
estaba realmente cubrible en una fecha concreta.

### 4.4 Solicitud interna, jornada y nómina

```text
regla y saldo aplicables
→ solicitud
→ responsable por necesidades del servicio
→ RRHH u órgano competente
→ resolución y firma
→ jornada o expediente
→ efecto económico, si existe
→ conciliación con nómina
→ notificación y auditoría
```

Los cuadrantes, marcajes e incidencias no entrarán directamente en nómina. Una
liquidación validada, trazable e idempotente será el contrato entre ambos
dominios. La futura sustitución de nómina requerirá varios ciclos paralelos y
conciliados.

### 4.5 Selección, bolsa e incorporación

```text
información pública
→ solicitud, tasa, firma y registro
→ requisitos
→ méritos reutilizados
→ autobaremo según bases congeladas
→ revisión
→ lista provisional y alegaciones
→ resultado definitivo
→ bolsa, llamamiento y nombramiento
→ alta en Personal cuando corresponda
```

### 4.6 Formación y carrera

```text
plan formativo
→ solicitud y prioridad
→ autorización
→ realización, asistencia y evaluación
→ certificado
→ mérito oficial reutilizable
→ eventual uso en carrera, provisión o selección
```

### 4.7 Prevención y vigilancia de la salud

Prevención será un contexto especialmente aislado. Personal y responsables
solo recibirán el resultado mínimo necesario —aptitud, limitación funcional o
adaptación aplicable—, nunca la historia clínica o el diagnóstico que lo
origina.

### 4.8 Analítica

La analítica consumirá proyecciones gobernadas, agregadas o seudonimizadas. No
consultará directamente el núcleo transaccional ni ofrecerá acceso genérico a
todas sus columnas.

## 5. Fronteras de acceso resultantes

| Zona | Alcance máximo inicial |
| --- | --- |
| Público anónimo | convocatorias, bases, plazos, ayuda, publicaciones aprobadas y bot limitado al corpus público |
| Aspirante externo | únicamente datos, méritos, solicitudes, tasas, alegaciones, notificaciones y posición propios |
| Empleado | expediente propio, nómina, jornada, permisos, formación y solicitudes |
| Responsable | tareas y equipo de su unidad; solo atributos necesarios, con delegación temporal |
| Técnico de RRHH | casos de uso, procedimientos, unidades y campos expresamente autorizados |
| Tribunal u órgano técnico | espacio aislado por proceso y méritos/documentos aplicables |
| Nómina e Intervención | vistas económicas y de fiscalización, no expediente general |
| Prevención y personal sanitario | compartimento propio para categorías especiales de datos |
| Igualdad y analítica | datos agregados o seudonimizados salvo competencia legal concreta |
| Administración técnica | sin lectura permanente de negocio; elevación temporal, nominal, supervisada y auditada |
| API, CLI, MCP y asistente | mismas capacidades y finalidad que el caso de uso ordinario; nunca un atajo de autorización |

Los portales externo e interno tendrán dominios, sesiones, audiencias de
credencial, redes y políticas de autenticación separados. Podrán compartir
sistema de diseño, pero no zona de confianza.

## 6. Patrones que se adoptan

- persona y expediente canónicos, con autoridad y procedencia por atributo;
- catálogos, estados, reglas, formularios y circuitos configurables y
  versionados;
- méritos y documentos aportados o consultados una sola vez;
- instantáneas reproducibles de cada participación y autobaremo;
- bases aprobadas, firmadas, publicadas y congeladas antes de calcular;
- historia de RPT, plazas, puestos, ocupaciones y condiciones;
- delegaciones y suplencias limitadas por unidad, acción y periodo;
- decisiones y correcciones mediante eventos, nunca cambios silenciosos;
- conectores y contratos estables para almacenamiento, firma, registro,
  identidad, antivirus, pago, nómina, jornada y comunicaciones;
- separación de transacción, archivo, búsqueda y analítica;
- exportación completa, documentación de interfaces y salida del proveedor;
- acceso técnico excepcional y temporal;
- interfaz común y accesible, aunque los módulos sean independientes.

## 7. Defectos que se rechazan

- múltiples identidades y portales sin una navegación comprensible;
- contraseñas, recuperaciones o clientes de firma obsoletos;
- dependencia de un sistema operativo, navegador o complemento;
- procedimientos electrónicos que terminan necesariamente en papel;
- estados irreversibles o supresión del historial;
- sincronizaciones sin origen, fecha ni nivel de confianza;
- responsables o técnicos con acceso a expedientes completos por defecto;
- generadores de informes libres sobre toda la base de datos;
- administradores de infraestructura con acceso implícito al contenido;
- tablas de auditoría ordinarias que puedan alterarse sin detección;
- reglas jurídicas o convencionales compiladas en el programa;
- acoplamiento directo entre jornada, nómina, Bolsa y expediente;
- dependencia del fabricante para crear conectores o recuperar los datos.

## 8. Reutilización y código abierto

No se ha verificado una suite integral, actual y de código abierto que cubra por
sí sola los RRHH de una Administración pública española. El Centro de
Transferencia de Tecnología ofrece servicios y activos reutilizables, pero se
debe distinguir entre:

1. servicio común consumible;
2. producto reutilizable por otra Administración;
3. código fuente abierto con licencia, versión y mantenimiento verificables.

SIGP, TRAMA o NEDAES no se etiquetarán como software abierto mientras no se
comprueben su código y licencia. Esta cautela no impide reutilizar estándares,
servicios comunes o componentes libres especializados detrás de nuestros
puertos.

Fuentes:

- [Guía de reutilización de activos del ENI](https://administracionelectronica.gob.es/pae_Home/dam/jcr%3Ad277818a-3f2d-408f-87e3-85a925863088/2022-ENI_ReutilizacionActivos.pdf).
- [Panel de servicios del CTT](https://dataobsae.administracionelectronica.gob.es/cmobsae3/panel/Panel.action?selectedScope=A2).

## 9. Condición para convertir un patrón en requisito

Antes de programar cualquier patrón de este documento se completará la puerta
institucional:

1. fuente provincial y norma aplicable;
2. ámbito personal, material, organizativo y temporal;
3. responsable y dato maestro;
4. circuito real y excepciones;
5. autorización, segregación y conservación;
6. casos de prueba públicos o anonimizados;
7. validación del órgano competente.

Si no existe autoridad suficiente, el diseño quedará como capacidad preparada
o resultado `no determinable`. Una interfaz observada en otra Administración
nunca suplirá una decisión legal, convencional o administrativa de la
Diputación.
