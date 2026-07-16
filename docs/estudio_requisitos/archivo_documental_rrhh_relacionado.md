# Archivo documental relacionado de Recursos Humanos

Fecha de referencia: 16 de julio de 2026.

Estado: especificación funcional y arquitectónica adoptada como base de
trabajo. Requiere validación de Archivo, Secretaría, RRHH, DPD y Seguridad y la
aprobación de la política de gestión documental antes de custodiar expedientes
reales.

## Decisión

El portal incorporará una capacidad transversal de archivo lógico y gobierno
documental para conservar y consultar cualquier documentación de Recursos
Humanos: bases, RPT, plantillas, acuerdos de Pleno, BOP, BOJA, BOE, informes,
actas, resoluciones, baremaciones, nombramientos, contratos, recursos y demás
evidencias.

No será una carpeta compartida ni un segundo almacén de ficheros. Se apoyará en
las capacidades existentes:

- `DocumentoLogico` para identidad y versión administrativa;
- `RepresentacionDocumento` para original, PDF, firma, copia u otro artefacto;
- `AlmacenObjetos` para bytes cifrados, inmutables y referenciados de forma
  opaca;
- firma, sello de tiempo, CSV/QR, cotejo y antivirus mediante puertos;
- autorización, auditoría y expediente electrónico comunes.

Cada módulo crea o incorpora documentos mediante casos de uso documentales.
Bolsa, Personal, RPT, Provisión, Nómina y Cronos no guardan copias propias ni
conocen buckets, rutas o productos de almacenamiento.

## Tres conceptos que no deben confundirse

### Contenido almacenado

Son los bytes exactos de una representación. Se identifican por referencia
opaca y huella, se cifran y se conservan conforme a su política. El mismo
contenido puede ser reutilizado sin duplicar físicamente los bytes.

### Documento lógico

Es una unidad administrativa con productor, contexto, versión, estado,
metadatos, clasificación y relaciones. Dos documentos con bytes idénticos no
son necesariamente el mismo documento: pueden haber sido incorporados por
órganos, actos o expedientes distintos.

### Expediente y dossier

- El **expediente electrónico** es la agregación ordenada reglada, con índice
  electrónico autenticado y documentos incorporados en un momento concreto.
- El **dossier transversal de RRHH** es una vista de consulta que relaciona
  expedientes y documentos —por ejemplo, toda la historia de una RPT— sin
  alterar, refundir ni sustituir los expedientes legales.

Un documento puede pertenecer a varios expedientes. Cada incorporación
conserva rol, orden, fecha y el índice firmado en el que quedó incluida.

## Base normativa y archivística

La validación final corresponde a los órganos competentes. La base comprobada
incluye:

- [Ley 39/2015](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565), artículos 17, 26, 27 y 70: archivo electrónico, documento, copia auténtica e índice de expediente;
- [Ley 40/2015](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566), artículo 46: conservación electrónica con identidad, integridad, reproducción y acceso;
- [Real Decreto 203/2021](https://www.boe.es/buscar/act.php?id=BOE-A-2021-5032), artículos 51 a 55: índice, acceso, originales en papel, conservación y archivo electrónico único;
- [Esquema Nacional de Interoperabilidad](https://www.boe.es/buscar/act.php?id=BOE-A-2010-1331), en especial su artículo 21;
- [Esquema Nacional de Seguridad](https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191);
- [Ley 7/2011 de Documentos, Archivos y Patrimonio Documental de Andalucía](https://www.boe.es/buscar/act.php?id=BOE-A-2011-18654);
- [Ley 19/2013 de transparencia](https://www.boe.es/buscar/act.php?id=BOE-A-2013-12887) y [Ley andaluza 1/2014](https://www.boe.es/buscar/act.php?id=BOE-A-2014-7534);
- RGPD y [LOPDGDD](https://www.boe.es/buscar/act.php?id=BOE-A-2018-16673), incluido el archivo en interés público del artículo 26.

Normas técnicas directamente aplicables:

- [NTI de Documento Electrónico](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13169);
- [NTI de Expediente Electrónico](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13170);
- [NTI de Digitalización](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13168);
- [NTI de Copiado Auténtico y Conversión](https://www.boe.es/buscar/act.php?id=BOE-A-2011-13172);
- [NTI de Política de Gestión de Documentos Electrónicos](https://www.boe.es/buscar/act.php?id=BOE-A-2012-10048);
- [e-EMGDE 3.0, noviembre de 2024](https://administracionelectronica.gob.es/pae_Home/dam/jcr%3Adc0d4b22-81f5-4d84-9461-1da5eee8993e/ENI-Esquema_Metadatos_Gestion_Doc_v3-acc3.pdf), como referencia para el perfil institucional ampliado.

No se afirmará que una NTI futura está aprobada sin la resolución publicada en
BOE. Tampoco se aplicará anticipadamente una modificación legal cuya entrada
en vigor sea posterior. El texto consolidado de la Ley andaluza 7/2011 muestra
una modificación publicada el 7 de abril de 2026 con efectos previstos para el
7 de abril de 2027; la versión aplicable se comprobará de nuevo en cada
implantación.

## Modelo funcional

### `SerieDocumental`

- código y denominación institucional;
- productor y función;
- procedimiento y periodos a los que se aplica;
- tabla de valoración y versión, si existe;
- regla de acceso, transferencia, conservación y eliminación;
- estado de aprobación y revisiones.

La clasificación será funcional. No se organizará por extensión, nombre de la
persona o carpetas decididas por cada módulo.

### `ExpedienteElectronico`

- identificador, productor DIR3 y procedimiento;
- apertura, cierre y reaperturas justificadas;
- personas interesadas mediante referencias opacas;
- clasificación y serie;
- estado, acceso y retención;
- índices firmados y transferencias de custodia.

### `IndiceExpedienteSnapshot`

Cada cierre o remisión conserva el índice exacto, su orden, metadatos, firma o
sello y huella. Nunca se regenera después a partir de «lo que haya ahora».
Correcciones, recursos o reaperturas generan un nuevo índice relacionado y
justificado sin destruir el anterior.

### `IncorporacionDocumentoExpediente`

Relación muchos a muchos con:

- documento y versión exactos;
- expediente e índice;
- rol documental y orden;
- fecha, origen y responsable de incorporación;
- estado de elaboración y autenticidad;
- motivo de retirada o sustitución, cuando proceda.

### `DossierRRHH`

Vista lógica versionada que puede reunir:

- una RPT y todas sus modificaciones;
- una plaza o puesto y los actos que explican su historia;
- una convocatoria desde la OEP hasta el nombramiento;
- un expediente personal autorizado;
- una regla de jornada con acuerdos, calendarios y resoluciones;
- un procedimiento de provisión, baremación y adjudicación.

El dossier no concede acceso adicional. Cada elemento se autoriza de forma
independiente y el resultado informa de documentos ocultos sin revelar sus
metadatos sensibles cuando la política lo exija.

### `RelacionDocumentalTipada`

El catálogo, gobernado desde la aplicación, admitirá relaciones como:

- `forma_parte_de`, `anexo_de`, `publica`, `ejecuta`;
- `modifica`, `rectifica`, `anula`, `sustituye`, `desarrolla`;
- `deriva_de`, `copia_de`, `conversion_de`, `version_de`;
- `acredita`, `motiva`, `resuelve`, `notifica`;
- relaciones con proceso, expediente, RPT, plantilla, categoría, plaza,
  puesto, persona, relación de servicio o acto.

Los nombres no se fijan en condicionales dispersos. Registrar un nuevo tipo o
rol permitido no exige recompilar; una semántica con efectos nuevos sí requiere
extensión tipada, revisión y pruebas.

### `EvidenciaPublicacion`

Para BOP, BOJA, BOE, sede y portal de transparencia:

- diario, editor, número, fecha, sección y páginas;
- identificador, ELI, CVE o CSV;
- URL oficial y fecha de captura/verificación;
- bytes exactos y huella del artefacto;
- acto publicado, fecha de publicación y efectos;
- correcciones, sustituciones o anulaciones relacionadas;
- resultado de validación de firma o sello.

El [BOP es oficial y auténtico conforme a la Ley 5/2002](https://www.boe.es/buscar/act.php?id=BOE-A-2002-6467).
La edición del [BOE se regula por el Real Decreto 181/2008](https://www.boe.es/eli/es/rd/2008/02/08/181)
y el [BOJA electrónico por el Decreto 188/2018](https://www.juntadeandalucia.es/boja/2018/199/3).
La URL sola no es evidencia de conservación a largo plazo.

### Acceso, retención y bloqueo

`ReglaAcceso`, `ReglaRetencion` y `BloqueoLegal` serán objetos versionados
distintos. Conservar permanentemente no significa publicar indefinidamente.
Que un documento estuviera en Internet no autoriza cualquier reutilización o
indexación nominal posterior.

## Metadatos institucionales mínimos

Además de los mínimos ENI:

- identificador persistente y versión;
- órgano productor DIR3 y agente incorporador;
- serie, procedimiento y tipo documental;
- fechas de creación, captura, incorporación, vigencia y cierre;
- estado de elaboración y autenticidad;
- formato, firma y validaciones históricas;
- expedientes, índices y dossiers relacionados;
- original, copia, conversión, publicación o corrección de origen;
- clasificación de seguridad y reglas de acceso;
- valoración, retención, transferencia y bloqueos;
- huella, referencia opaca y eventos de integridad;
- calidad de extracción u OCR, si existe.

Los metadatos se protegen igual que el contenido. Un nombre de fichero, título,
vista previa o relación puede revelar salud, afiliación sindical, sanciones,
domicilio, cuenta bancaria o circunstancias familiares.

## Originales, copias y representaciones

Estados o clases de representación que debe soportar el contrato versionado:

- original electrónico nativo;
- copia electrónica auténtica;
- copia auténtica de papel;
- imagen digital ordinaria;
- representación de trabajo;
- representación de visualización accesible;
- representación para firma;
- conversión de preservación;
- rendición publicada;
- copia parcial o expurgada;
- OCR o texto extraído;
- audio de ayuda, cuando corresponda.

Reglas invariables:

1. El original nativo conserva firmas, sellos y evidencia de validación.
2. Un escaneo no es copia auténtica sin el procedimiento competente.
3. OCR, HTML, audio o PDF convertido nunca sustituyen al original.
4. Cada derivado identifica origen, herramienta/adaptador, versión, fecha,
   responsable, huellas y validación.
5. El OCR muestra el aviso `texto_extraido_no_autentico` y su nivel de calidad.
6. Una copia expurgada es otro objeto relacionado y documenta el fundamento y
   las partes ocultadas sin copiarlas a logs.
7. La migración de formato conserva origen y destino y prueba la posibilidad de
   reproducción.

## Aplicación a RPT y plantilla

Cada versión o modificación de RPT relacionará, cuando existan:

- propuesta, memoria y datos técnicos;
- negociación y acuerdos;
- informes de RRHH, Secretaría e Intervención;
- dictámenes;
- acuerdo inicial y definitivo del Pleno;
- información pública y alegaciones;
- BOP, correcciones y evidencia de publicación;
- instantánea estructurada exacta aprobada;
- remisiones, firmas y justificantes.

La [tabla CAVAD 265 para Catálogo/RPT de diputaciones](https://www.juntadeandalucia.es/boja/2023/183/36)
prevé conservación permanente y transferencia del expediente y del documento
o base técnica. La [ficha completa](https://www.juntadeandalucia.es/sites/default/files/2023-09/01_02_02_265TV_L%20Cat%20RPT%20Diputaciones.pdf)
debe validarse por Archivo antes de parametrizarla.

No se duplica la base viva. Se conserva una instantánea probatoria de los datos
tal como fueron aprobados, con esquema, catálogos, reglas de reconstrucción,
manifiesto, huellas y fuente. La vigencia jurídica y el tiempo de conocimiento
siguen el modelo bitemporal de RPT.

## Aplicación a selección, Bolsa y provisión

El dossier puede enlazar:

- OEP o instrumento habilitante;
- plazas, puestos y versiones RPT;
- resolución de aprobación y bases firmadas;
- publicaciones y correcciones;
- solicitudes, admisión y exclusión;
- tribunal o comisión;
- pruebas, méritos, baremaciones, actas y votos;
- alegaciones, revisiones y recursos;
- resultados, llamamientos, nombramientos o contratos.

No basta con conservar el PDF final. Tampoco se publica en bloque el expediente
que contiene datos de aspirantes. La tabla CAVAD 61 de selección municipal no
se aplicará automáticamente a una Diputación: productor, serie y periodo deben
coincidir. Sin tabla competente, el estado será `pendiente_valoracion` y la
eliminación quedará prohibida.

## Captura e incorporación

Canales permitidos:

- carga manual por personal autorizado;
- generación interna del portal;
- registro o sede mediante conector;
- conectores homologados de BOP, BOJA, BOE y transparencia;
- importación gobernada desde archivo o sistema corporativo;
- digitalización controlada de papel.

Flujo:

```text
captura de bytes y procedencia
→ cuarentena y límites
→ antivirus
→ identificación de formato y firma
→ extracción de metadatos/OCR auxiliar
→ propuesta de clasificación y relaciones
→ revisión humana
→ incorporación transaccional a documento y expediente
→ índice/manifestación cuando corresponda
→ retención, auditoría y outbox
```

Un conector o una IA puede proponer tipo y relaciones, pero no declarar una
copia auténtica, cambiar clasificación, publicar, expurgar o eliminar sin la
decisión competente.

## Consulta y experiencia de usuario

El panel interno `Archivo de RRHH` ofrecerá:

- búsqueda de texto y metadatos con autorización previa;
- filtros por serie, tipo, órgano, proceso, fecha, estado, RPT, categoría,
  plaza, puesto y clasificación;
- cronología de actos, publicaciones, correcciones y versiones;
- árbol de expediente e índices firmados;
- grafo pequeño de relaciones documentales relevantes;
- comparación de versiones y metadatos;
- vista del original y derivados claramente etiquetados;
- cobertura, fuente, integridad, firma, acceso y retención visibles;
- generación de dossier o paquete autorizado con recibo.

La búsqueda no obtiene primero todos los resultados para filtrarlos después.
La autorización y seguridad por filas/campos se aplican en el origen. El texto
OCR no crea un canal lateral para contenido restringido.

## Canales de acceso separados

1. Acceso interno por función, finalidad y ámbito.
2. Acceso de la persona interesada a su expediente.
3. Transparencia, publicidad activa y cotejo público.
4. Consulta archivística o investigación conforme a sus reglas.

Ser interesado no concede notas internas ni datos de terceros. La descarga
parcial usa una copia expurgada separada. Toda consulta, previsualización,
descarga, exportación y cotejo queda auditada según riesgo.

El portal externo solo lee una proyección de documentos marcados y aprobados
como públicos. No comparte repositorio de consultas, credenciales ni rutas con
el archivo interno.

## Retención, transferencia y eliminación

Una regla identifica:

- autoridad y tabla exacta;
- productor, serie y periodo;
- hito que inicia el cómputo;
- plazo de transferencia;
- conservación permanente, eliminación o muestreo autorizado;
- régimen de acceso y revisión;
- excepciones, recursos, auditorías, litigios y bloqueos.

Las copias de seguridad aportan continuidad, pero no sustituyen al archivo.
No habrá eliminación automática por alcanzar una fecha. Flujo mínimo:

1. cálculo de elegibilidad;
2. validación de tabla, ámbito y cobertura;
3. comprobación de bloqueos y procedimientos pendientes;
4. revisión de Archivo y órganos afectados;
5. autorización formal;
6. eliminación segura y coordinada en repositorios y réplicas gestionadas;
7. acta firmada con cantidades, objetos, huellas, autoridad y resultado;
8. conservación del asiento probatorio sin contenido eliminado.

## Puertos hexagonales

- `RepositorioDocumentosLogicos`;
- `RepositorioExpedientesElectronicos`;
- `RepositorioIndicesExpediente`;
- `RepositorioRelacionesDocumentales`;
- `RepositorioSeriesYRetencion`;
- `BuscadorDocumentalAutorizado`;
- `FuenteDocumentosInstitucionales` por proveedor;
- `VerificadorDocumentoOficial`;
- `GeneradorCopiaAutentica`;
- `ExtractorTextoDocumental`;
- `ExpurgadorDocumentalAtestado`;
- `TransferidorArchivo`;
- `ExportadorExpedienteENI`;
- puertos comunes de almacenamiento, firma, autorización y auditoría.

PostgreSQL será el primer adaptador de metadatos. El contenido puede residir
en S3 compatible, filesystem dedicado o gestor documental detrás de
`AlmacenObjetos`. Oracle o InSiDe/Archive implementarán adaptadores sin cambiar
los casos de uso.

## Encaje con el código actual

La base existente se reutiliza, pero necesita una evolución aditiva:

1. `TipoRelacionDocumento` ya es extensible y permite añadir referencias a
   RPT, plaza, puesto, norma o publicación mediante catálogo gobernado.
2. `DocumentoLogico` exige hoy una `Plantilla`. Un documento oficial capturado
   no debe recibir una plantilla ficticia. Un contrato V2 separará procedencia
   `generado_desde_plantilla`, `capturado` o `digitalizado` y mantendrá
   compatibilidad con V1 durante la migración.
3. Los tipos actuales de `RepresentacionDocumento` no cubren original,
   digitalización, copia auténtica, OCR, publicación o expurgo. Se ampliarán en
   V2 de forma tipada; no se sobrecargará `visualizacion`.
4. El estado administrativo del documento no se mezclará con ciclo de archivo,
   retención, transferencia o bloqueo. Se añadirán agregados relacionados.
5. El contenido direccionado por huella permite deduplicación física, pero no
   fusionará identidades documentales distintas.

La migración crea tablas y contratos V2 en paralelo, adapta documentos V1 con
procedencia conocida, compara manifiestos y retira la compatibilidad solo tras
pruebas y copia restaurada. No obliga a rehacer almacenamiento, firma o núcleo.

## Pruebas de aceptación

1. Un BOP relacionado con RPT y Pleno usa un único contenido almacenado y dos
   incorporaciones documentales sin perder contexto.
2. Dos documentos con bytes iguales y productores distintos mantienen dos
   identidades.
3. Un documento pertenece a varios expedientes y cada índice firmado conserva
   su incorporación exacta.
4. Cerrar de nuevo un expediente no altera un índice anterior.
5. Una corrección de BOP enlaza la publicación corregida y no la sobrescribe.
6. Una RPT histórica conserva datos, esquema, manifiesto y documentos que la
   aprobaron.
7. OCR permite buscar, pero se etiqueta como no auténtico y nunca sustituye la
   imagen.
8. Un escaneo ordinario no aparece como copia auténtica.
9. Una persona externa solo obtiene la rendición publicada aprobada.
10. Una parte interesada recibe una copia parcial cuando existen datos ajenos.
11. Un documento permanente puede dejar de ser públicamente indexable sin ser
    eliminado.
12. Una tabla de valoración de otro productor no habilita eliminación.
13. Un bloqueo legal impide eliminación y registra el intento.
14. Un paquete ENI contiene documentos, metadatos, índice y firmas exactos.
15. Cambiar el adaptador de objetos no cambia la identidad documental.
16. Una búsqueda sin autorización no revela existencia, título, OCR, tamaño ni
    tiempos diferentes aprovechables.
17. Toda descarga y exportación genera recibo con actor, finalidad, versión y
    huella sin copiar datos personales al log.

## Información pendiente de la Diputación

- política de gestión de documentos electrónicos vigente;
- cuadro de clasificación y perfil de metadatos;
- series y tablas de valoración aplicables a cada productor y periodo;
- archivo electrónico y gestor documental corporativos;
- DIR3, órganos productores y responsables de transferencia;
- sistemas de registro, Pleno, resoluciones, BOP, sede y transparencia;
- formatos de intercambio e índices usados hoy;
- historia de RPT, plantilla y selección disponible;
- criterios de acceso, transparencia, expurgo y reutilización;
- procedimiento de copia auténtica y digitalización;
- RTO/RPO, conservación de firmas y estrategia de preservación;
- responsables de clasificación, publicación, transferencia y eliminación.

Hasta recibirlos podemos construir el contrato, los casos de uso y la interfaz
con expedientes de prueba, pero no declarar conforme ni transferir/eliminar
documentación real.
