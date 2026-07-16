# Calendario hábil, laboral y de jornada con historia reproducible

Fecha de referencia: 16 de julio de 2026.

Estado: especificación funcional y arquitectónica adoptada como base de trabajo.
Requiere validación de Secretaría, RRHH, Archivo, Seguridad y DPD antes de
utilizarse para producir actos o controlar jornada real.

## Decisión

El portal tendrá un contexto transversal `Calendarios`. No será una tabla de
festivos dentro de Cronos ni una utilidad con fines de semana compilados.

El sistema debe contestar por separado estas preguntas:

1. si una fecha es día natural;
2. si es hábil para un plazo administrativo concreto;
3. si un centro o servicio está abierto;
4. si una persona tenía trabajo programado y durante cuántos minutos;
5. qué fuentes y versiones justifican cada respuesta.

Una misma fecha puede ser inhábil administrativamente, estar abierto un centro
con servicio continuo y ser laborable para una persona a turnos. Reducir esas
realidades a un único booleano `es_laborable` produciría errores jurídicos y de
nómina, permisos o fichajes.

## Alcance transversal

`Calendarios` presta el mismo servicio versionado a:

- Bolsa y procesos selectivos, para apertura, cierre y vencimiento de plazos;
- Registro y ventanilla única, para fecha de presentación y efectos;
- Personal, para calendarios institucionales y adscripciones;
- Cronos, para jornada teórica, permisos y contraste con fichajes;
- Nóminas, solo cuando una regla retributiva aprobada requiera el calendario;
- Dietas y desplazamientos, para reglas que distingan día laborable;
- firma, notificación y expediente, para explicar fechas de un acto;
- API, web, CLI y MCP, siempre a través de los mismos casos de uso.

Las fronteras de responsabilidad serán:

| Contexto | Responsabilidad |
|---|---|
| Calendarios | Fuentes oficiales, versiones, reglas de calendario, políticas de cómputo y cálculos explicables. |
| Personal | Centro, municipio de cabecera, relación de servicio y adscripción histórica de una persona. |
| Cronos | Cuadrantes, turnos asignados, fichajes y jornada efectivamente realizada. |
| Procedimiento consumidor | Regla jurídica concreta y versión de política que debe aplicar. |
| Documentos | Original firmado, metadatos, representaciones y conservación de la fuente y del cálculo. |

Calendarios no guardará el motivo médico, familiar, sindical o disciplinario de
una reducción. Personal le entregará una adscripción o atestación mínima y
opaca con el efecto horario autorizado.

## Distinciones obligatorias

### Día natural

Es una fecha civil del calendario gregoriano. No implica que sea hábil,
laborable ni que tenga veinticuatro horas exactas en una zona horaria.

### Día hábil administrativo

Se determina para una finalidad, órgano, sede, persona interesada y política
de cómputo. La exclusión de sábados, domingos y festivos prevista por la Ley
39/2015 no se sustituirá por una regla genérica de lunes a viernes sin fuentes.

El calendario del registro electrónico y el cómputo que compara residencia de
la persona y sede del órgano son contextos distintos. La presentación
electrónica puede estar disponible todos los días aunque sus efectos se
produzcan conforme al siguiente día hábil.

### Apertura de centro o servicio

Describe si una unidad está abierta total o parcialmente. Una apertura
excepcional no convierte el día en hábil administrativo; un cierre interno no
lo convierte por sí solo en inhábil para toda la ciudadanía.

### Día y minutos de trabajo programados

Resultan del calendario laboral, perfil horario, cuadrante y adscripción
vigentes. Una media jornada puede ser menos tiempo cada día o días completos
alternos: nunca se representará automáticamente como `0,5 días`.

### Tiempo civil y tiempo instantáneo

Las fechas y los intervalos en días se expresan sin hora ni zona. Las horas,
turnos, registros y vencimientos instantáneos usan `Europe/Madrid`, instante
UTC y desplazamiento aplicado. Los días de cambio horario pueden tener 23 o 25
horas y no se calcularán multiplicando días civiles por 24.

## Base normativa y fuentes de autoridad

La validación jurídica final corresponde a los órganos competentes. La base
actual comprobada incluye:

- [Ley 39/2015, artículos 30 y 31](https://www.boe.es/eli/es/l/2015/10/01/39/con): cómputo de plazos, días hábiles, calendario de sede y registro electrónico;
- [Real Decreto 203/2021](https://www.boe.es/eli/es/rd/2021/03/30/203/con): sede, fecha y hora oficial, registro e incidencias;
- [TREBEP, artículos 37.1.m y 47](https://www.boe.es/eli/es/rdlg/2015/10/30/5/con): negociación y establecimiento de jornada;
- [Estatuto de los Trabajadores, artículo 37.2](https://www.boe.es/eli/es/rdlg/2015/10/23/2/con) y [Real Decreto 2001/1983](https://www.boe.es/eli/es/rd/1983/07/28/2001/con): fiestas laborales y fiestas locales;
- [Código Civil, artículo 5](https://www.boe.es/eli/es/rd/1889/07/24/%281%29/con): cómputo civil, distinto del administrativo;
- [Real Decreto 236/2002](https://www.boe.es/eli/es/rd/2002/03/01/236): hora legal y cambios estacionales;
- [Reglamento regulador del tiempo de trabajo de la Diputación](https://www.dipgra.es/export/sites/diputaciongranada/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/.galleries/DIPUTACION-Delegaciones-Galerias-Normativa-RRHH/REGLAMENTO-TIEMPO-DE-TRABAJO.pdf), sujeto a sus modificaciones y al texto vigente en cada fecha.

La cadena de fuentes será configurable y se conservará, como mínimo, para:

1. BOE: fiestas nacionales y disposiciones estatales;
2. BOJA: fiestas de Andalucía, días inhábiles y publicación de fiestas locales;
3. Ayuntamiento y BOJA: acuerdo y publicación del municipio;
4. BOP y Diputación: acuerdos, reglamentos y calendarios internos;
5. resoluciones aprobadas de centro, servicio, turno o situación excepcional.

La fuente estructurada facilita la carga, pero el documento oficial exacto es
la evidencia. Una URL externa no basta: se archivarán sus bytes, huella,
firma/CVE cuando existan, fecha de captura, procedencia y resultado de
validación.

## Evidencia histórica inicial

Como vectores de contraste, no como valores compilados, las fiestas locales
publicadas para Granada capital incluyen:

| Año | Fechas locales publicadas |
|---|---|
| 2024 | 2 de enero y 30 de mayo |
| 2025 | 2 de enero y 19 de junio |
| 2026 | 2 de enero y 4 de junio |

Fuentes: [Ayuntamiento de Granada 2024](https://www.granada.org/inet/wordenanz.nsf/947df60665eef1b1c1256e280062ae7e/1ea307fe6a934fd7c1258a3e0045637b%21OpenDocument), [BOJA 2025](https://www.juntadeandalucia.es/boja/2024/198/35) y [BOJA 2026](https://www.juntadeandalucia.es/boja/2025/197/28).

Andalucía trasladó expresamente determinados descansos de 2026 al 2 de
noviembre y al 7 de diciembre. El motor aplicará lo publicado en el
[Decreto 101/2025](https://www.juntadeandalucia.es/boja/2025/93/1.html), no
inventará una regla general de traslado. El calendario administrativo consta
en la [Orden de 24 de septiembre de 2025](https://www.juntadeandalucia.es/boja/2025/187/26).

Las fuentes pueden recibir correcciones posteriores. La
[modificación de 20 de abril de 2026](https://www.juntadeandalucia.es/boja/2026/79/19)
es un caso que obliga a conservar tanto la vigencia jurídica como el momento en
que el sistema conoció la corrección.

## Modelo de dominio

### `VersionCalendario`

- identificador estable y número de versión;
- finalidad: administrativo general, registro electrónico, laboral
  institucional, apertura de centro, perfil o turno;
- ámbito territorial y organizativo;
- vigencia administrativa desde/hasta;
- conocimiento del sistema desde/hasta;
- estado: borrador, en validación, publicado, sustituido o anulado;
- documento y acto fuente, ELI/CVE/CSV si existen, huella y firma;
- responsable de validación y evidencia de publicación.

### `ReglaCalendario`

- fecha civil o intervalo;
- efecto tipado e independiente: festivo, inhábil administrativo, cierre,
  apertura excepcional, jornada reducida o trabajo programado;
- ámbito, prioridad normativa y disposición concreta;
- versión de calendario a la que pertenece;
- relación con la regla corregida, sustituida o anulada;
- explicación pública y explicación interna, separadas por clasificación.

Una fecha con varios motivos conserva todos ellos, pero el contador no duplica
el día.

### `PerfilHorario`

- ciclos e intervalos de trabajo;
- pausas y descansos computables o no computables;
- zona horaria y vigencia;
- modalidad general, partida, turnos u otra gobernada;
- reglas aprobadas para periodos especiales;
- fuente y versión.

No contendrá nombres de personas. La asignación se realiza mediante una
referencia desde Personal.

### `AdscripcionCalendario`

- referencia opaca a persona o relación de servicio;
- centro, servicio y municipio de cabecera versionados;
- perfil horario y cuadrante aplicables;
- vigencia y procedencia;
- atestaciones mínimas de excepciones autorizadas.

### `PoliticaComputo`

Configura una semántica jurídica soportada y tipada:

- unidad: horas, días naturales, días hábiles, meses o años;
- cantidad;
- inclusión o exclusión del día inicial;
- regla de vencimiento y prórroga;
- dirección del cálculo;
- finalidad y calendarios que deben combinarse;
- tratamiento permitido de incidencias de sede;
- versión de esquema y casos de prueba aprobados.

No admite código Go, JavaScript, SQL, expresiones libres ni consultas
configuradas por el usuario. Una nueva semántica jurídica se incorpora como
operador revisado y probado; una nueva combinación de operadores existentes se
publica desde la aplicación sin recompilar.

### `CalculoCalendario`

Un cálculo con relevancia administrativa es inmutable y conserva:

- entrada canónica y política exacta;
- fecha de vigencia y fecha de conocimiento consultadas;
- versiones de todos los calendarios, perfiles y adscripciones;
- reglas aplicadas y descartadas con motivo;
- resultado y desglose día a día;
- fuentes, cobertura y discrepancias;
- huella canónica, actor, finalidad, autorización y correlación;
- firma o sello de tiempo cuando forme parte de un acto.

Una vista previa puede ser efímera. En cuanto fundamente una decisión,
notificación, permiso, nómina o resolución debe materializarse y enlazarse al
expediente.

## Bitemporalidad y correcciones

Cada versión mantiene dos ejes:

- **vigencia**: cuándo produjo efectos la disposición;
- **conocimiento**: cuándo fue recibida, validada, corregida o sustituida en el
  sistema.

Esto permite reproducir «qué era aplicable el 15 de marzo» y «qué sabía el
sistema cuando calculó el plazo». Nunca se sobrescribe un calendario
publicado.

Una corrección retroactiva:

1. crea una versión sucesora;
2. identifica la fuente y el alcance corregido;
3. localiza cálculos y plazos abiertos potencialmente afectados;
4. abre incidencias para revisión competente;
5. no modifica automáticamente actos ya dictados ni borra su evidencia.

## Casos de uso y puertos

Casos de uso mínimos:

```text
ClasificarFecha(fecha, finalidad, ámbito, conocidoEn)
CalcularVencimiento(inicio, politicaVersion, contexto)
ContarDias(intervalo, politicaVersion, contexto)
SiguienteDiaHabil(fecha, contexto)
CalcularJornadaTeorica(intervalo, adscripcionVersion)
ExplicarCalculo(calculoID)
CompararVersiones(calendarioA, calendarioB)
PublicarVersion(borrador, revisiones, autorización)
```

Puertos hexagonales:

- `RepositorioCalendarios`;
- `RepositorioPoliticasComputo`;
- `RepositorioCalculosCalendario`;
- `FuenteCalendarioOficial` para cada proveedor homologado;
- `ProveedorAdscripcionLaboral`;
- `ProveedorCuadranteTurnos`;
- `ProveedorZonaHoraria`;
- `VerificadorDocumentoOficial`;
- puertos comunes de autorización, auditoría, documentos y firma.

PostgreSQL será el primer adaptador. Oracle u otro motor implementará los
mismos puertos sin alterar los casos de uso. Los conectores oficiales podrán
descargar cambios, pero nunca publicar directamente: entran en preparación y
requieren validación.

## Contrato de resultado

El resultado no devuelve un único `esLaborable`. Como mínimo separa:

```text
esFestivoOficial
esInhabilAdministrativo
centroAbierto
personaProgramada
minutosTeoricos
reglasAplicadas
fuentesOficiales
versionesUtilizadas
cobertura
estadoDeterminacion
```

`estadoDeterminacion` será `determinado`, `parcial` o `indeterminado`. Si falta
el calendario oficial del año, municipio o finalidad requerida, se falla de
forma cerrada: nunca se supone silenciosamente que de lunes a viernes es hábil.

## Portal interno y proyección pública

El portal público puede mostrar calendarios oficiales ya publicados, plazos y
explicaciones sin datos personales. El área interna añade:

- bandeja de fuentes importadas y pendientes de revisión;
- calendario anual por capas y ámbitos;
- comparación y diferencias entre versiones;
- editor gobernado de reglas y políticas;
- simulador con explicación día a día;
- incidencias por ausencia, contradicción o corrección;
- impacto sobre cálculos y expedientes;
- gestión de centros, perfiles y adscripciones según rol.

Turnos, reducciones, adscripciones y cálculos nominales permanecen únicamente
en la red y superficie interna. Un responsable solo consulta su ámbito
organizativo vigente. Un técnico de calendario no obtiene por ello acceso a
fichajes ni a causas protegidas.

## Gobierno y separación de funciones

Flujo mínimo:

```text
capturado
→ preparado
→ validación RRHH para efectos laborales
→ validación Secretaría para cómputo administrativo
→ doble control de publicación
→ publicado
→ sustituido o anulado
```

Una sola versión puede contener únicamente el ámbito que haya superado sus
validaciones. No se retrasará un calendario público correcto por una regla
interna de turnos pendiente, ni se publicará esta última aprovechando la
aprobación del primero.

Permisos positivos y mínimos:

- importador: captura, pero no valida ni publica;
- gestor RRHH: prepara y valida contenido laboral;
- Secretaría: valida cómputo administrativo;
- publicador: publica una versión previamente aprobada;
- gestor de centro: mantiene perfiles de su ámbito, sin acceso transversal;
- auditor: consulta evidencia, sin mutar;
- consumidor técnico: calcula solo para la finalidad autorizada.

## Seguridad, privacidad y cumplimiento

- autorización RBAC+ABAC positiva por acción, finalidad, ámbito y versión;
- superficies pública e interna, identidades y repositorios de lectura
  separados conforme al diseño general del portal;
- cifrado, referencias opacas y auditoría de toda consulta nominal o
  exportación;
- registros sin nombres, causas sensibles, turnos completos ni ubicaciones;
- proyección pública generada desde versiones explícitamente publicadas;
- fuentes y cálculos inmutables, retención y bloqueo coordinados con Archivo;
- ninguna IA decide el calendario aplicable ni publica una corrección; puede
  ayudar a extraer una fuente, siempre con revisión humana y evidencia;
- ENS, ENI, RGPD y LOPDGDD se aplican como puertas del producto, no como una
  declaración automática de conformidad.

## Pruebas de aceptación obligatorias

### Fecha y cómputo

- años bisiestos, fin de mes y fin de año;
- intervalos semiabiertos sin doble conteo;
- plazos en días naturales, hábiles, meses y años;
- residencia y sede con calendarios diferentes;
- calendario propio del registro electrónico;
- ausencia de fuente y conflicto entre fuentes.

### Calendario real

- fiestas locales de Granada de 2024, 2025 y 2026;
- traslados expresos del 2 de noviembre y 7 de diciembre de 2026;
- 24 y 31 de diciembre como posible no laborable interno, sin inferir
  inhabilidad administrativa;
- centro cuyo municipio de cabecera no es Granada;
- persona a turnos que trabaja en festivo;
- media jornada diaria y días completos alternos;
- servicio abierto durante un cierre general.

### Hora e historia

- cambios horarios del 29 de marzo y 25 de octubre de 2026 en `Europe/Madrid`;
- corrección oficial conocida después de un cálculo anterior;
- reproducción por fecha de vigencia y de conocimiento;
- firma y huella idénticas para la misma entrada canónica;
- identificación de cálculos potencialmente afectados sin alterarlos.

### Propiedades

- un motivo adicional no duplica un día;
- contar un intervalo dividido en partes adyacentes produce el mismo total;
- el siguiente día hábil satisface la política exacta utilizada;
- ningún resultado determinado carece de fuente publicada;
- API, CLI, MCP y web ofrecen el mismo resultado canónico autorizado.

## Migración e implantación

1. Inventariar calendarios y reglas hoy distribuidos en Cronos, hojas,
   acuerdos y procedimientos.
2. Solicitar a RRHH los calendarios anuales firmados de 2024-2026, texto
   consolidado del reglamento, calendarios de centros, ciclos de turno y
   relación centro-municipio de cabecera.
3. Solicitar a Secretaría las políticas de cómputo y calendario de sede y
   registro realmente vigentes.
4. Importar fuentes oficiales a preparación y reconciliarlas a cuatro ojos.
5. Ejecutar el nuevo motor en sombra y comparar resultados históricos.
6. Resolver discrepancias; nunca corregirlas con una regla oculta.
7. Publicar primero la consulta pública y los cálculos administrativos
   controlados.
8. Integrar después Personal y Cronos con adscripciones y cuadrantes reales.

La `FechaCivil` exacta ya iniciada para baremación debe convertirse, antes de
ser consumida por varios contextos, en un valor neutral compartido. Esta es una
reubicación interna pequeña y temprana, no un cambio de semántica ni una
reescritura del núcleo.

## Puertas de salida de NO-GO

No se usará el módulo para un acto o cálculo de jornada real hasta acreditar:

- fuentes oficiales completas y diccionario de ámbitos;
- políticas jurídicas aprobadas y casos de prueba firmados;
- persistencia bitemporal y migraciones reversibles;
- autorización por ámbito, segregación de funciones y auditoría durable;
- archivo del original y trazabilidad de cada regla;
- comparación satisfactoria con expedientes históricos;
- revisión de Secretaría, RRHH, Seguridad, Archivo y DPD;
- pruebas de carga, copia y restauración del adaptador productivo.
