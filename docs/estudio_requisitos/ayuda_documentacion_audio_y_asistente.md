# Accesibilidad personalizable, ayuda, audio y asistente

Estado: requisito transversal incorporado al estudio.

Fecha: 14 de julio de 2026.

## 1. Decisión

La plataforma tendrá un módulo transversal denominado provisionalmente **Ayuda,
Conocimiento y Asistencia**. No pertenecerá a Bolsa, Dietas, Cronos ni a otro módulo
concreto. Todos ellos lo consumirán mediante los mismos casos de uso.

Este módulo proporcionará, desde una única fuente aprobada:

- ayuda contextual de pantallas, campos, errores y trámites;
- guías paso a paso y preguntas frecuentes;
- manual HTML accesible y exportación documental;
- lectura fácil cuando proceda;
- audio por página y apartado;
- transcripción y descripción textual de elementos visuales;
- buscador de ayuda;
- base de conocimiento para el asistente público y los futuros asistentes autenticados;
- derivación a soporte humano.

La lectura en voz alta será una ayuda adicional. No sustituirá la obligación de que toda la
interfaz funcione de forma nativa con lectores de pantalla, teclado, ampliación y otras
tecnologías de apoyo. No se creará una segunda «web para personas ciegas» que pueda quedar
incompleta o desactualizada.

## 2. Base de accesibilidad a fecha del estudio

La conformidad legal se evaluará contra el
[Real Decreto 1112/2018](https://www.boe.es/eli/es/rd/2018/09/07/1112/con), la norma
armonizada aplicable y el resto de obligaciones que correspondan a cada superficie y
contenido.

A 14 de julio de 2026, la Comisión Europea identifica
[EN 301 549 V3.2.1](https://digital-strategy.ec.europa.eu/en/policies/web-accessibility-directive-standards-and-harmonisation)
como última versión armonizada para la Directiva de accesibilidad web. Como objetivo de
ingeniería adicional se adoptará
[WCAG 2.2 nivel AA](https://www.w3.org/TR/WCAG22/), junto con criterios AAA seleccionados
cuando aporten una mejora real. No se declarará conformidad AAA global, porque no resulta
apropiada ni alcanzable para todos los contenidos.

La plataforma deberá disponer también de:

- declaración de accesibilidad enlazada desde todas las páginas;
- mecanismo accesible para comunicar incumplimientos o solicitar información accesible;
- procedimiento de reclamación;
- responsable y calendario de revisión;
- registro y seguimiento de incidencias de accesibilidad;
- revisiones periódicas y después de cambios materiales.

La norma y sus versiones se parametrizarán en las puertas de calidad. Si cambia la versión
armonizada, la actualización no dependerá de buscar reglas dispersas por cada módulo.

## 3. Tres capacidades distintas

### 3.1 Accesibilidad nativa

Toda la aplicación deberá ser perceptible, operable, comprensible y robusta sin activar un
tema especial:

- HTML semántico y orden de lectura lógico;
- encabezados, regiones, listas, tablas y formularios correctamente estructurados;
- nombres, funciones, valores y estados expuestos a las tecnologías de apoyo;
- navegación completa por teclado, foco visible y gestión correcta del foco;
- enlaces para saltar al contenido y evitar bloques repetidos;
- mensajes dinámicos y errores anunciados sin robar el foco indebidamente;
- ampliación, redistribución y zoom sin pérdida de información o funcionalidad;
- alternativas textuales útiles para imágenes, gráficos, firmas y capturas;
- subtítulos, transcripciones y audiodescripción cuando correspondan;
- compatibilidad probada con lectores de pantalla de escritorio y móvil.

### 3.2 Preferencias personales

La personalización visual se rige por
[`sistema_diseno_y_temas.md`](sistema_diseno_y_temas.md). La persona podrá sincronizar
tema, contraste, escala, legibilidad, densidad y movimiento reducido sin que estos ajustes
se conviertan en diagnósticos o datos disponibles para RRHH.

Las preferencias específicas de lectura podrán incluir:

- idioma y voz entre las autorizadas;
- velocidad de reproducción;
- reproducción de una sección o lectura continuada;
- resaltado sincronizado del texto;
- posición de reproducción;
- preferencia de transcripción visible;
- no iniciar nunca audio automáticamente.

### 3.3 Lectura en voz alta integrada

La aplicación ofrecerá «Escuchar esta ayuda», «Escuchar este apartado» y, cuando resulte
útil, «Escuchar la página». El reproductor será un componente común, accesible y
tematizable; no se implementará de forma distinta en cada módulo.

La lectura de controles, formularios y estados dinámicos seguirá dependiendo en primer
lugar de su semántica accesible. El audio pregenerado se reservará principalmente para
ayuda, instrucciones, contenidos públicos estables y textos aprobados.

## 4. Una fuente, varios formatos

El principio será publicación desde una única fuente:

```text
contenido canónico estructurado y aprobado
├── HTML accesible y ayuda contextual
├── manual imprimible y PDF accesible
├── versión de lectura fácil revisada
├── audio por apartados y transcripción
├── índice de búsqueda
└── corpus autorizado del asistente
```

No se redactará por separado un manual web, otro PDF, otro guion de audio y otra respuesta
para el bot. Eso produciría contradicciones y versiones obsoletas. Cada derivado conservará
la referencia y huella de la versión canónica que lo originó.

## 5. Modelo de contenido de ayuda

Cada unidad de ayuda tendrá, al menos:

```text
ayuda_id
clave_contexto_estable
título y contenido estructurado
tipo: guía | campo | error | trámite | pregunta frecuente | glosario | aviso
audiencia y superficie
módulo y caso de uso
procedimiento, convocatoria o configuración relacionada
idioma
clasificación: pública | aspirante | empleado | responsable | RRHH | administración
fuentes oficiales
versión y huella del contenido
vigencia desde/hasta
relación con la versión sustituida
propietario editorial y responsables de revisión
estado y fechas de revisión/publicación
```

Pantallas, pasos, campos, estados y errores complejos tendrán claves de contexto estables.
Así podrá abrirse la ayuda exacta aunque cambie la ruta web o exista otro cliente de
escritorio.

La clasificación de acceso se aplicará antes de buscar o recuperar el contenido. Un texto
interno de RRHH no podrá aparecer porque el bot lo recupere y después intente ocultarlo.

## 6. Edición y gobierno desde la aplicación

Las guías de usuario, preguntas frecuentes y ayudas contextuales serán contenido
administrable desde el área autorizada. Su ciclo será:

```text
borrador
→ revisión funcional
→ revisión jurídica, de privacidad y accesibilidad cuando proceda
→ aprobación
→ publicación inmediata o programada
→ revisión periódica
→ sustitución, archivo o retirada
```

Todas las transiciones, autores, revisores, comparaciones y fechas quedarán auditados. La
edición permitirá bloques estructurados, componentes aprobados y recursos analizados, pero
no HTML, CSS, JavaScript, instrucciones para modelos ni recursos remotos arbitrarios.

La documentación técnica —arquitectura, seguridad, operación, API, datos, despliegue y
recuperación— se mantendrá como documentación junto al código. La documentación de usuario
se administrará como contenido gobernado. Ambas compartirán identificadores, versiones,
fuentes, responsables y puertas de actualización.

Ninguna función se considerará terminada si faltan:

- ayuda de la acción normal;
- explicación y recuperación de errores;
- guía para operaciones jurídicas o irreversibles;
- actualización del manual correspondiente;
- material de soporte para la unidad que atenderá incidencias.

## 7. Ayuda exhaustiva y atención

La ayuda se organizará por tareas reales, no solo por menús. Incluirá:

- primeros pasos y orientación por tipo de usuario;
- procedimientos completos con requisitos, plazos, documentos y resultado esperado;
- explicación de cada estado y siguiente acción;
- guías de identidad, firma, registro, pago, documentos y notificaciones;
- ayuda específica de Bolsa, méritos, autobaremación, subsanación y llamamientos;
- ayuda de empleado para certificados, Dietas, Cronos y futuros módulos;
- preguntas frecuentes vinculadas a la versión de la convocatoria o procedimiento;
- glosario administrativo en lenguaje claro;
- lectura fácil en recorridos críticos, revisada por personas;
- diagnóstico guiado de incidencias;
- estado del servicio e incidencias conocidas;
- contacto humano y creación de solicitud de soporte;
- mecanismo para valorar, corregir o comunicar que una ayuda no resuelve la duda.

El diagnóstico distinguirá, al menos, duda funcional, problema técnico, discrepancia de
datos, identidad, firma, pago, fichero, accesibilidad y consulta jurídica o procedimental.
Nunca pedirá enviar DNI, documentos personales o credenciales por correo ordinario.

## 8. Generación de manuales con capturas

Se reutilizará el patrón observado en OPES/USO:

- escenario declarativo con audiencia, versión, pasos y resoluciones;
- capturas de escritorio y móvil;
- anotaciones numeradas fuera de la imagen para no tapar controles;
- originales separados de las imágenes anotadas;
- HTML como salida canónica revisable;
- generación de la versión imprimible desde el HTML;
- manifiesto de ficheros y versión de la aplicación mostrada;
- revisión visual antes de publicar.

Los manuales privados usarán preferentemente entornos de demostración y datos sintéticos.
No se capturarán sesiones reales por comodidad. Si excepcionalmente se usa una pantalla
real, se eliminarán de forma irreversible datos personales, identificadores, cookies,
tokens, rutas privadas y metadatos. El difuminado no será suficiente para secretos o datos
de alto riesgo; se preferirá sustitución sintética o tachado opaco verificado.

Las capturas originales se tratarán como material potencialmente sensible, con acceso y
retención propios. El manual declarará versión de aplicación, tema, idioma, perfil y tamaño
de pantalla. Un cambio material de interfaz marcará las guías afectadas como pendientes de
regeneración; no se presentará una captura antigua como vigente.

El PDF será un derivado accesible y etiquetado, no una sucesión de imágenes. La guía HTML
será siempre la alternativa principal y más actualizable.

## 9. Canal de audio

### 9.1 Flujo de generación

```text
contenido aprobado
→ extracción de bloques semánticos
→ normalización de pronunciación
→ cálculo de huella por bloque
→ reutilización o síntesis asíncrona
→ comprobación automática mediante transcripción
→ escucha humana según riesgo y muestreo
→ publicación atómica de manifiesto y audios
```

El texto se dividirá por apartados, no en numerosos botones por párrafo. También podrán
tener lectura propia las tablas, diagramas e infografías cuando su alternativa textual
aporte información distinta del cuerpo principal.

Un cambio solo regenerará los bloques cuya huella haya cambiado. La versión anterior no se
servirá como vigente mientras el nuevo audio esté pendiente. Si la síntesis falla, el texto
seguirá disponible y el trámite no quedará bloqueado.

### 9.2 Manifiesto de audio

Cada recurso conservará:

```text
audio_id y bloque_id
contenido_origen_id, versión y huella
idioma
perfil de voz autorizado
motor y versión del adaptador
formato, duración y tamaño
referencia de almacenamiento
estado de generación
resultado de transcripción y control de calidad
estado de escucha humana cuando se exija
fecha de publicación y sustitución
clasificación y política de acceso heredadas
```

La interfaz privada no necesita recibir en el manifiesto el texto personal completo. Las
URLs serán autorizadas, breves y vinculadas al usuario, recurso y finalidad.

### 9.3 Conectores

Se definirán puertos de aplicación independientes, con nombres coherentes en castellano:

- `SintetizadorVoz`;
- `TranscriptorAudio`;
- `ExtractorContenidoLectura`;
- `ValidadorAudio`;
- `AlmacenAudio`;
- `RepositorioManifiestosAudio`;
- `PublicadorAudio`;
- `PoliticaPrivacidadSintesis`.

Edge TTS y Whisper podrán ser adaptadores iniciales para pruebas o contenido público si se
aprueban sus condiciones. No formarán parte del dominio ni serán obligatorios. Se deberá
poder usar un motor completamente local o contratado sin modificar los casos de uso.

### 9.4 Privacidad de la síntesis

No se enviarán a un servicio de voz externo páginas que contengan DNI, domicilio, nómina,
salud, discapacidad, expediente, documentos u otros datos personales. Esas páginas se
leerán mediante tecnología del dispositivo o mediante infraestructura local expresamente
autorizada, sin caché compartida.

El texto privado no se pasará como argumento visible de un proceso ni se guardará en
registros técnicos. Los trabajos recibirán referencias autorizadas y el contenido mínimo en
un canal seguro. Los audios heredarán acceso, conservación, borrado y clasificación del
texto origen.

## 10. Asistente de consultas

### 10.1 Primera fase: asistente público

La primera entrega será de solo consulta y responderá sobre:

- convocatorias y procesos publicados;
- requisitos y documentación;
- plazos vigentes;
- funcionamiento del portal;
- inscripción, firma, registro, subsanación y alegaciones;
- preguntas frecuentes y canales de ayuda.

Solo utilizará contenido público aprobado y la proyección pública de los módulos. No tendrá
acceso a personas, expedientes, documentos privados, bases operativas ni herramientas
administrativas.

Los datos dinámicos —por ejemplo, si un plazo está abierto hoy— se obtendrán mediante un
caso de uso público y determinista. El corpus explicará la regla, pero no sustituirá la
fuente estructurada de fechas y estados.

Una consulta que contenga un DNI u otro dato personal no activará ninguna búsqueda
individual. El asistente indicará que no debe incluirse ese dato y dirigirá al acceso seguro
o al canal humano correspondiente.

### 10.2 Respuestas fundamentadas

Cada respuesta mostrará:

- fuente oficial enlazada;
- apartado concreto;
- versión o convocatoria;
- fecha de vigencia o revisión;
- indicación clara cuando sea una explicación y no una resolución administrativa.

Si no hay evidencia suficiente, existe contradicción o la fuente está caducada, el
asistente lo reconocerá y ofrecerá búsqueda o soporte humano. No completará huecos con una
respuesta plausible.

No decidirá admisión, puntuación, exclusión, nombramiento ni derechos. Puede explicar un
baremo publicado, pero el cálculo oficial seguirá siendo determinista, versionado y
auditable.

### 10.3 Futuro asistente autenticado

Si se autoriza más adelante, será una superficie, credencial, índice y memoria separados
del asistente público. Aplicará autorización antes de recuperar información y solo podrá
invocar herramientas cerradas como `ConsultarMisSolicitudes` o `ConsultarMisPlazos`.

No existirá una herramienta «consultar la base de datos» ni una consulta genérica de
personas. La representación de otra persona exigirá apoderamiento vigente. Cualquier acción
futura requerirá confirmación explícita, las mismas garantías que la interfaz ordinaria y
un justificante; conversar con el bot no equivaldrá a presentar una solicitud.

### 10.4 Protección del asistente

- corpus separados por clasificación, audiencia y vigencia;
- documentos tratados como datos, nunca como instrucciones para el modelo;
- cuarentena y revisión antes de indexar;
- exclusión de borradores, prompts, registros, notas internas y ficheros aportados por
  usuarios salvo un caso de uso aprobado;
- defensa frente a inyección de instrucciones y contenido hostil;
- herramientas permitidas por lista cerrada y autorización fuera del modelo;
- aislamiento de conversaciones y cachés entre personas;
- minimización, redacción y retención limitada de consultas;
- prohibición de reutilizar conversaciones con datos personales como aprendizaje estable;
- límites de uso, protección ante automatización abusiva y monitorización;
- revisión humana y retirada de respuestas erróneas;
- evaluación de alucinación, citas, fugas de información y ataques antes de cada versión.

La interfaz del asistente será accesible: etiquetas visibles, historial con semántica de
conversación, mensajes anunciados, estado de espera, control de foco, teclado, transcripción
y lectura de respuestas. Si la IA está caída, seguirán disponibles el buscador, las
preguntas frecuentes, los manuales y el soporte humano.

## 11. Arquitectura conceptual

```text
Módulo Ayuda, Conocimiento y Asistencia
├── gestión editorial y vigencia
├── publicación multiformato
├── ayuda contextual y búsqueda
├── generación y validación de audio
├── construcción de corpus autorizados
├── evaluación del asistente
└── derivación a soporte humano
        │
        ├── portal público y aspirante
        ├── portal del empleado
        ├── espacio del responsable
        ├── backoffice de RRHH
        └── API, escritorio y futuros canales
```

Los adaptadores de HTML, PDF, almacenamiento, voz, transcripción, búsqueda, modelo de
lenguaje, Telegram o MCP serán intercambiables. El núcleo conservará contenido, permisos,
vigencia, fuentes, flujo editorial y evidencia; no conocerá marcas de proveedores.

## 12. Reutilización del trabajo de OPES

La revisión de `/home/alberto/Trabajo/OPES` ha encontrado piezas aprovechables:

- `skills/opes-audio-edge-whisper/SKILL.md`: cierre textual, bloques, huellas, generación
  selectiva y QA con Whisper;
- `opes-uso/internal/application/ports/services.go`: puertos Go para generación y
  transcripción;
- `opes-uso/internal/application/job/generate_audio_asset.go`: trabajo asíncrono,
  almacenamiento, manifiesto, idempotencia y revisión auditiva pendiente;
- `skills/opes-rag-tutor-bots/SKILL.md`: fuentes aprobadas, anclas de sección, citas y
  abstención;
- `opes-salidas/coordinacion_temarios/tools/build_opes_course_rag_from_html.py`: extracción
  de HTML por secciones;
- `/home/alberto/Trabajo/USO/web/scripts/build_help_manual.py`: escenarios de capturas y
  salida HTML/Markdown/PDF;
- `/home/alberto/Trabajo/USO/web/docs/SCREENSHOT_HELP_MANUALS.md`: proceso de privacidad y
  revisión de manuales.

No existe todavía una única implementación que genere de extremo a extremo manual,
audio y asistente. El paquete de manual inspeccionado produce HTML, Markdown, PDF y
capturas, mientras el audio se genera por otro flujo. En VEC se unificarán mediante el
contenido canónico y sus manifiestos.

Se reutilizarán contratos, pruebas y patrones después de revisar propiedad, licencias,
dependencias y calidad. No se copiarán sin más las siguientes limitaciones observadas:

- Edge TTS depende de un servicio externo no oficial y no es apto para datos personales;
- el adaptador actual pasa texto como argumento de proceso;
- la comprobación automática de similitud de audio es insuficiente para contenido crítico;
- la evaluación RAG de ejemplo tiene pocas preguntas y tolera resultados mal ordenados;
- existen esquemas de corpus incompatibles;
- faltan defensas explícitas completas frente a inyección de instrucciones;
- algunos contenidos y respuestas están fijados directamente en código;
- la interfaz de audio y chat todavía necesita mejoras de accesibilidad;
- los directorios de manual de usuario localizados en numerosos paquetes OPES están vacíos;
  el generador gráfico de USO es una referencia parcial, no un subsistema acabado.

## 13. Puertas de calidad

Antes de publicar una versión se comprobará:

- accesibilidad automática y revisión manual;
- teclado, foco, ampliación y redistribución;
- lectores de pantalla representativos en escritorio y móvil;
- todos los temas y preferencias personales admitidos;
- enlaces y claves de ayuda contextual;
- contenido sin responsable, caducado o contradictorio;
- correspondencia entre aplicación, capturas y versión del manual;
- ausencia de datos reales en capturas y derivados;
- huellas de texto, manifiestos y audios;
- pronunciación, comienzo correcto, cortes y duración;
- transcripción automática y escucha humana según política;
- exactitud de búsqueda y recuperación mediante un banco amplio de consultas;
- preguntas sin respuesta, ambiguas y adversariales;
- soporte de cada afirmación y corrección de las citas;
- separación entre corpus público, interno y personal;
- fuga entre usuarios, módulos y perfiles;
- funcionamiento degradado sin voz, buscador avanzado o IA;
- derivación real a soporte humano.

Las métricas de recuperación incluirán precisión, posición del resultado correcto y
cobertura, no solo que aparezca en algún punto de los primeros resultados. Las preguntas de
prueba se versionarán junto al contenido.

## 14. Requisitos verificables iniciales

### Accesibilidad

- **ACC-001.** No existirá una versión accesible separada: la interfaz ordinaria será
  semántica y utilizable con tecnologías de apoyo.
- **ACC-002.** Todo caso de uso será operable por teclado y conservará un orden de foco
  lógico y visible.
- **ACC-003.** La aplicación soportará ampliación, redistribución y preferencias del
  sistema sin pérdida de información o funcionalidad.
- **ACC-004.** Las pruebas combinarán herramientas automáticas, revisión experta y personas
  usuarias con discapacidad.
- **ACC-005.** Declaración, comunicación, solicitud de formato accesible y reclamación serán
  accesibles desde todas las superficies aplicables.
- **ACC-006.** WCAG 2.2 AA será el objetivo de ingeniería sin confundirlo con la versión
  armonizada legal que corresponda en cada fecha.

### Ayuda y documentación

- **AYU-001.** El HTML accesible será la fuente principal; PDF, impresión, audio, búsqueda y
  corpus se derivarán de la misma versión aprobada.
- **AYU-002.** Ninguna función se cerrará sin ayuda normal, recuperación de errores y
  actualización de manuales técnicos y de usuario.
- **AYU-003.** Cada pantalla, paso, campo complejo y error tendrá una clave de ayuda estable.
- **AYU-004.** Todo contenido declarará audiencia, clasificación, versión, vigencia,
  responsable, fuentes y fecha de revisión.
- **AYU-005.** Borrador, revisión, aprobación, publicación, sustitución y retirada serán
  estados auditados y reversibles.
- **AYU-006.** Las bases de conocimiento pública, de aspirante, empleado y áreas internas se
  mantendrán separadas y autorizadas antes de recuperar contenido.
- **AYU-007.** Los recorridos críticos tendrán lenguaje claro y, cuando corresponda,
  lectura fácil revisada por personas.
- **AYU-008.** Las preguntas frecuentes se vincularán a la versión de convocatoria,
  procedimiento o configuración que explican.
- **AYU-009.** Los manuales gráficos usarán datos sintéticos y declararán la versión exacta
  de la interfaz mostrada.
- **AYU-010.** El gestor de contenidos no admitirá código ejecutable ni recursos remotos
  arbitrarios.
- **AYU-011.** Enlaces rotos, ayuda crítica ausente o contenido caducado bloquearán la
  publicación del trámite afectado.
- **AYU-012.** Siempre existirán buscador, mapa de ayuda, canal humano y valoración de la
  utilidad del contenido.

### Audio

- **AUD-001.** El audio se generará desde contenido canónico aprobado y mantendrá su
  referencia y huella.
- **AUD-002.** Se podrá escuchar página, apartado y lectura continuada, con pausa, parada,
  velocidad y sin reproducción automática.
- **AUD-003.** El reproductor será un componente central, accesible y no interferirá con los
  lectores de pantalla.
- **AUD-004.** Todo audio tendrá texto equivalente, idioma, duración, origen, estado y datos
  de control de calidad.
- **AUD-005.** Cambiar un bloque invalidará únicamente su audio y nunca servirá una versión
  antigua como vigente.
- **AUD-006.** Síntesis, transcripción, almacenamiento y publicación serán conectores
  sustituibles ejecutados de forma asíncrona.
- **AUD-007.** Contenido personal o interno no saldrá hacia un sintetizador externo sin base,
  contrato, evaluación y autorización expresa.
- **AUD-008.** Los audios heredarán clasificación, permisos, conservación y borrado del
  contenido de origen.
- **AUD-009.** Los registros técnicos no conservarán texto personal leído ni el texto se
  expondrá como argumento de proceso.
- **AUD-010.** La indisponibilidad del audio no impedirá leer, comprender ni completar el
  trámite.
- **AUD-011.** La política podrá exigir transcripción automática y escucha humana antes de
  publicar.
- **AUD-012.** La caché se identificará por huella, idioma, voz y versión del sintetizador.

### Asistente

- **BOT-001.** La primera fase será pública, de solo consulta y limitada a información
  aprobada y publicada.
- **BOT-002.** Usará proyecciones y casos de uso públicos; no accederá a personas,
  expedientes, documentos privados o bases operativas.
- **BOT-003.** Plazos y estados dinámicos procederán de datos estructurados del módulo, no
  exclusivamente del corpus textual.
- **BOT-004.** Cada respuesta indicará fuente, apartado, versión y fecha; sin evidencia
  suficiente se abstendrá.
- **BOT-005.** El asistente no decidirá admisión, puntuación, exclusión, selección o
  derechos.
- **BOT-006.** Introducir un DNI en el asistente público no provocará una consulta personal.
- **BOT-007.** Un futuro asistente autenticado tendrá credencial, memoria, corpus y
  herramientas separados y solo operará sobre datos autorizados de la propia persona.
- **BOT-008.** No habrá herramientas genéricas de base de datos, sistema de ficheros o
  consulta de personas.
- **BOT-009.** Toda acción futura requerirá autorización ordinaria, confirmación expresa y
  justificante.
- **BOT-010.** Se aplicarán controles de inyección, clasificación, aislamiento, minimización
  y retención antes y después del modelo.
- **BOT-011.** No se aprenderá ni alimentará el corpus con conversaciones que contengan
  datos personales o respuestas no revisadas.
- **BOT-012.** El asistente será accesible y podrá leer sus respuestas mediante el mismo
  servicio de voz autorizado.
- **BOT-013.** Existirá derivación a soporte humano con consentimiento para transferir el
  contexto útil.
- **BOT-014.** Sin IA seguirán funcionando búsqueda, preguntas frecuentes, manuales y
  soporte.
- **BOT-015.** Cada versión superará pruebas de exactitud, abstención, citas, inyección,
  fuga entre usuarios y accesibilidad.

## 15. Decisiones pendientes

- Unidad responsable y flujo de aprobación de contenidos de ayuda.
- Herramienta de edición estructurada y formatos canónicos.
- Idiomas iniciales y alcance de lectura fácil.
- Motores de síntesis y transcripción permitidos, condiciones, licencias y ubicación.
- Política exacta para lectura de contenido personal en el dispositivo o infraestructura
  local.
- Formatos, calidad, almacenamiento y conservación de audio.
- Lectores de pantalla y dispositivos que formarán la matriz oficial de pruebas.
- Casos que exigirán escucha humana completa o por muestreo.
- Sistema de tickets y unidad de soporte responsable.
- Modelo y proveedor del asistente, si procede, después de la evaluación de impacto.
- Alcance y fecha del futuro asistente autenticado.
- Política de conservación y anonimización de conversaciones.

Estas decisiones no impiden fijar desde ahora las fronteras, metadatos, permisos y puertas
de calidad.
