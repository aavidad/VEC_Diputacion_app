# CT-CUM-06 — Política ENI candidata de documento, expediente, firma, copia y conservación

Fecha de corte: 20 de agosto de 2026.

Base documental: `002805e3b7c01fd13c85fafd14574144b115ea48`.

Estado: **PENDIENTE_VALIDACION; no aprobada; cero actos o documentos reales autorizados**.

## Autoridad, capability e invariante

La
[matriz normativa de contratación temporal](matriz_normativa_contratacion_temporal_2026-07-23.md)
define `CT-CUM-06` como la política ENI de documento, expediente, firma, copia
y conservación. La puerta bloquea actos y documentos administrativos reales.

La capability de este dossier consiste exclusivamente en preparar reglas
candidatas para identificar objetos documentales, relacionarlos, conservar su
integridad y exigir evidencia antes de cualquier efecto. No constituye:

- una política corporativa ENI aprobada;
- una tabla de valoración o calendario de conservación;
- una autorización de firma, sello, registro, CSV, cotejo o notificación;
- la declaración de una copia como auténtica;
- la elección de formatos, proveedores o infraestructura reales;
- un alta, eliminación, migración o transferencia documental;
- una autorización de datos reales, preproducción o producción.

La invariante del corte es cerrada: ningún renderizador, fichero, hash, firma
de prueba o commit convierte por sí solo un objeto en documento administrativo,
copia auténtica, expediente, acto firmado o evidencia con plazo aprobado.

## Fuentes y límites

El dossier se apoya únicamente en:

- el
  [inventario candidato CT-CUM-02](inventario_tratamientos_contratacion_temporal_2026-08-20.md);
- el [dossier RAT/EIPD candidato CT-CUM-03](ct_cum_03_rat_eipd_contratacion_temporal_2026-08-20.md);
- el
  [dossier ENS/SoA candidato CT-CUM-04](ct_cum_04_categorizacion_ens_declaracion_aplicabilidad_2026-08-20.md);
- el
  [plan de riesgos candidato CT-CUM-05](ct_cum_05_analisis_tratamiento_riesgos_plan_seguridad_2026-08-20.md);
- la [especificación histórica de firma y cotejo](firma_csv_qr_y_cotejo.md),
  solo como antecedente técnico no aprobado para este módulo;
- el [expediente remitido por RRHH](expediente_contratacion_temporal_rrhh.md);
- los [objetivos y hoja de ruta](objetivos_y_hoja_ruta_rrhh_2026-07-23.md) y
  la
  [matriz de campos de análisis](matriz_campos_analisis_rrhh_contratacion_temporal_2026-07-23.md).

No se incluyen documentos, firmas, certificados, CSV, expedientes, personas,
series, plazos, ubicaciones, proveedores, cuentas, claves o rutas reales.
CT-CUM-02 a CT-CUM-05 continúan siendo candidatos locales y este documento no
cambia su estado.

## Vocabulario de estado

| Estado | Significado en el dossier |
| --- | --- |
| `PENDIENTE_VALIDACION` | Requiere aprobación de la autoridad competente. |
| `POLITICA_CANDIDATA` | Regla propuesta sin vigencia ni efecto operativo. |
| `CATALOGO_PENDIENTE` | Tipo, formato, serie o transición no están publicados por su autoridad. |
| `EVIDENCIA_NO_APORTADA` | No existe prueba aprobada para el alcance requerido. |
| `CONSERVACION_NO_DETERMINADA` | No hay serie, plazo, disposición ni autoridad aprobados. |
| `EFECTO_PROHIBIDO` | La operación no puede ejecutarse en este corte. |
| `BLOQUEANTE` | La ausencia impide actos/documentos reales. |

No se usa `VIGENTE`, `FIRMADO`, `REGISTRADO`, `COPIA_AUTENTICA`, `ARCHIVADO` o
`ELIMINADO` como estado real del sistema.

# Parte A — Objetos y relaciones documentales

## Objetos que deben permanecer distintos

| Objeto lógico candidato | Invariante | Decisión pendiente |
| --- | --- | --- |
| Documento aportado | Se conservan los bytes exactos recibidos y su recibo; una conversión no sustituye al original. | Formatos admitidos, canal, controles y autoridad de incorporación. |
| Documento generado | La fuente gobernada y cada representación son objetos relacionados y versionados. | Plantilla, formato, metadatos y autoridad emisora. |
| Representación de consulta | Es un derivado accesible o seguro, nunca el original ni prueba de autenticidad. | Perfil de transformación y acceso. |
| Versión preparada para firma | Bytes inmutables ligados a política, expediente, acto y huella. | Circuito, formato y validaciones previas. |
| Revisión firmada o sellada | Cada paso es una nueva revisión inmutable; no se reescribe una firma anterior. | Política, competencia, certificados y validación. |
| Documento emitido | Solo existe tras completar controles y efecto formal autorizado. | Registro, firma/sello, emisión y autoridad competente. |
| Copia | Mantiene vínculo con origen, propósito, transformación y huellas. | Clase de copia y órgano habilitado. |
| Evidencia de validación | Informe separado, versionado y ligado a bytes exactos. | Política de confianza, vigencia y custodia. |
| Índice de expediente | Manifiesto ordenado y versionado de objetos y relaciones. | Perfil ENI, firma y autoridad de cierre. |
| Recibo | Evidencia de una operación exacta, no sustituto del documento. | Esquema, conservación y acceso. |

Una regeneración a partir de datos actuales no recupera una versión emitida.
Una apariencia visible, QR o CSV tampoco prueba firma, autenticidad o copia.

## Identidad documental candidata

Cada objeto deberá tener una referencia opaca no significativa y conservar,
cuando proceda:

| Grupo | Metadatos mínimos candidatos |
| --- | --- |
| Identidad | Referencia, tipo documental gobernado, versión y estado del catálogo. |
| Contexto | Expediente, procedimiento, fase, actuación y finalidad por referencia. |
| Autoridad | Órgano, unidad, competencia y actor por referencias atestadas. |
| Procedencia | Fuente, canal, recibo e instante fiable. |
| Contenido | Formato, versión del formato, tamaño acotado y huella de bytes. |
| Relaciones | Original, derivado, revisión, sustitución, copia, anexo e índice mediante referencias. |
| Firma | Política, firmante competente por referencia, clase, sello, tiempo e informe de validación. |
| Acceso | Clasificación, relación, finalidad, representación y decisión de acceso. |
| Conservación | Serie, disposición, bloqueo, transferencia y revisión, todos pendientes. |
| Trazabilidad | Correlación, idempotencia, auditoría y recibos sin contenido documental. |

La enumeración es una estructura candidata. Los nombres, cardinalidades y
perfiles definitivos requieren catálogos versionados y aprobación.

## Versiones y derivaciones

1. Cada cambio de bytes crea una versión u objeto derivado nuevo.
2. La fuente, transformación, herramienta, política y huellas de entrada y
   salida quedan ligadas sin incluir contenido en auditoría.
3. Un objeto firmado no se normaliza, estampa ni reserializa fuera de las
   revisiones permitidas por la política aprobada.
4. Una rectificación, sustitución o revocación no borra la versión probatoria;
   añade una relación y una nueva decisión.
5. La vista previa y la representación accesible son derivaciones; no alteran
   el original ni conceden acceso a él.
6. El fallo de una transformación obligatoria se interpreta como fallo total,
   nunca como emisión degradada.

# Parte B — Expediente electrónico candidato

## Estructura mínima

| Elemento | Regla candidata | Estado |
| --- | --- | --- |
| Identificador | Referencia gobernada del expediente, separada del número visible cuando exista. | `POLITICA_CANDIDATA` |
| Procedimiento y órgano | Referencias a catálogos y competencia vigentes. | `PENDIENTE_VALIDACION` |
| Interesados y representación | Relaciones atestadas y versionadas; no se infieren de un identificador. | `PENDIENTE_VALIDACION` |
| Documentos | Referencias a objetos y versiones exactas, no rutas ni copias embebidas. | `POLITICA_CANDIDATA` |
| Actuaciones | Tipo, autoridad, instante, motivo, versión y recibo. | `CATALOGO_PENDIENTE` |
| Orden | Criterio estable, explícito y reproducible. | `CATALOGO_PENDIENTE` |
| Índice | Manifiesto completo con metadatos y huellas de los objetos incluidos. | `EVIDENCIA_NO_APORTADA` |
| Cierre y reapertura | Actuaciones separadas, competentes y trazables. | `PENDIENTE_VALIDACION` |
| Acceso y entrega | Proyección y paquete según relación, finalidad y clasificación. | `PENDIENTE_VALIDACION` |
| Conservación | Serie, bloqueo, transferencia, acceso y disposición aprobados. | `CONSERVACION_NO_DETERMINADA` |

## Índice y paquete

El perfil aprobable deberá determinar:

- qué objetos y versiones forman parte del expediente;
- orden y relaciones entre documentos y actuaciones;
- metadatos obligatorios y esquema versionado;
- huellas, algoritmo y manifestación de errores;
- firma o sello del índice cuando corresponda;
- tratamiento de anexos, documentos voluminosos y formatos no incorporables;
- actualización, cierre, reapertura y nueva versión del índice;
- representación accesible y entrega probatoria;
- validación independiente y conservación a largo plazo.

Este dossier no genera un índice ni declara compatible un formato concreto.

# Parte C — Firma, sello, tiempo, CSV y cotejo

## Política de firma pendiente

Una política publicada deberá vincular de forma exacta:

| Dimensión | Decisión requerida |
| --- | --- |
| Acto | Procedimiento, fase, tipo documental y resultado que se firma. |
| Competencia | Puesto o función, titularidad, suplencia/delegación y vigencia. |
| Acceso | Permiso técnico separado de la competencia para firmar. |
| Circuito | Orden, cardinalidad, incompatibilidades, rechazo y cancelación. |
| Formato | Perfil, nivel, firma/sello, tiempo y restricciones de modificación. |
| Evidencia | Bytes de entrada/salida, huellas, certificado público, validación y recibo. |
| Vigencia | Fecha de publicación, versión, sustitución y revisión. |

Un rol técnico no concede competencia y una competencia no abre por sí sola el
acceso. CT-CUM-07 debe resolver actuaciones humanas y automatizadas antes de
cualquier efecto.

## Validación y custodia

- Se valida cada revisión contra los bytes exactos y la política aplicable.
- Firma, cadena, revocación, tiempo, modificaciones y formato producen un
  informe separado y conservable.
- Los certificados, claves y secretos permanecen fuera de Git y de los
  metadatos públicos.
- Un cliente o conector no declara válida su propia salida sin verificación en
  la frontera confiable.
- Una indisponibilidad de validación nunca equivale a firma válida.
- La evidencia de validación requiere política de confianza y conservación
  aprobadas; este dossier no las fija.

## CSV, QR y cotejo

CSV y QR permanecen `EFECTO_PROHIBIDO`. Si una política futura los autoriza:

1. un código opaco se liga a una sola versión emitida y a sus bytes exactos;
2. el QR solo facilita acceso y no sustituye autenticidad, firma ni texto;
3. conocer el código no concede acceso universal al contenido;
4. estados reservado, activo, retirado, sustituido o inexistente no permiten
   enumerar personas o expedientes;
5. el servicio devuelve la versión emitida, nunca una regeneración;
6. código, URL y contenido no se copian a logs, métricas o trazas;
7. sede, disponibilidad, acceso y conservación requieren aprobación formal.

# Parte D — Copias y representaciones

## Clasificación pendiente

| Clase candidata | Requisitos antes de usarla | Estado |
| --- | --- | --- |
| Copia simple o representación | Origen, finalidad, transformación, huellas y advertencia de alcance. | `PENDIENTE_VALIDACION` |
| Copia electrónica auténtica | Habilitación, órgano, metadatos, procedimiento, firma/sello y vínculo verificable con original. | `EFECTO_PROHIBIDO` |
| Copia digitalizada auténtica | Reglas de digitalización, calidad, cotejo, metadatos y autoridad. | `EFECTO_PROHIBIDO` |
| Copia en papel verificable | Política de CSV/cotejo, integridad y acceso aprobados. | `EFECTO_PROHIBIDO` |
| Representación accesible o testada | Transformación gobernada, revisión, relación con fuente y acceso separado. | `POLITICA_CANDIDATA` |

Una ocultación o testado produce un objeto derivado con nueva huella. Nunca se
modifica el original firmado para crear una vista parcial.

## Acceso y entrega

Cada acceso debe ligarse a objeto, versión, representación, expediente,
relación, acción, finalidad, actor, competencia, instante y vigencia. Las
capacidades de ver metadatos, previsualizar, descargar, cotejar y exportar son
separadas.

No se concede acceso por conocer una URL, referencia, CSV o número visible.
Los intentos y resultados se auditan con metadatos mínimos y sin contenido,
secretos o autorizaciones temporales completas.

# Parte E — Conservación, transferencia y disposición

## Matriz pendiente por serie

No se fija ningún plazo. Para cada serie, la autoridad de Archivo con DPD,
Jurídico y responsables competentes deberá aprobar:

| Campo | Decisión pendiente |
| --- | --- |
| Serie y alcance | Tipos documentales, expedientes, versiones y evidencias incluidos. |
| Inicio del cómputo | Evento jurídico o administrativo gobernado. |
| Plazo y fundamento | Duración aprobada y norma o tabla de valoración aplicable. |
| Acceso durante conservación | Perfiles, finalidades, restricciones y revisiones. |
| Bloqueos | Litigio, investigación, fiscalización, derechos u otra causa formal. |
| Transferencia | Destino archivístico, momento, formato, paquete y recibo. |
| Disposición | Conservación permanente, eliminación u otra decisión aprobada. |
| Eliminación | Autoridad, doble control cuando proceda, evidencia y reconciliación. |
| Migración | Formatos, validación, huellas y preservación de autenticidad. |
| Revisión | Responsable, vigencia y tratamiento de cambios normativos. |

Estado de todas las series: `CONSERVACION_NO_DETERMINADA`.

## Reglas mientras falta aprobación

1. No se programa expurgo ni eliminación silenciosa.
2. No se inventa un plazo por defecto ni se interpreta la historia de solo
   adición como conservación permanente.
3. Cualquier eliminación futura requiere política, autoridad, bloqueo,
   alcance, recibo y prueba propios.
4. Un bloqueo impide eliminación, pero no amplía acceso ni finalidad.
5. Las copias y restauraciones respetan serie, clasificación, bloqueo y
   disposición; no crean un plazo alternativo.
6. La migración conserva origen, transformación, huellas, validación y
   relación, sin destruir la evidencia previa por conveniencia.

# Parte F — Evidencia, gobierno y cierre

## Evidencia de una política aplicable

Cada afirmación futura deberá enlazar:

1. requisito y versión normativa;
2. catálogo o política publicada;
3. objeto, versión y alcance exactos;
4. autoridad, competencia y aprobación;
5. procedimiento y configuración versionados;
6. pruebas positivas, negativas, de fallo y recuperación;
7. informe de firma, integridad, acceso o disposición según corresponda;
8. auditoría independiente, hallazgos y remediación;
9. vigencia, sustitución y próxima revisión.

Una prueba local acredita solo su predicado. No acredita ENI, autenticidad,
conservación o efecto administrativo completos.

## Decisiones reservadas

| Decisión | Autoridad requerida | Estado |
| --- | --- | --- |
| Política ENI corporativa | Órganos y responsables competentes | `PENDIENTE_VALIDACION` |
| Catálogo de documentos y metadatos | Secretaría, Archivo, RRHH y autoridades funcionales | `CATALOGO_PENDIENTE` |
| Firma, sello y actuación automatizada | Competencia formal y CT-CUM-07 | `EFECTO_PROHIBIDO` |
| Series, plazos y disposición | Archivo, DPD y autoridad competente | `CONSERVACION_NO_DETERMINADA` |
| CSV y cotejo | Política formal, sede y autoridades competentes | `EFECTO_PROHIBIDO` |
| Formatos y preservación | Archivo, Sistemas, Seguridad y responsables | `PENDIENTE_VALIDACION` |
| Actos/documentos administrativos reales | Autoridad formal tras cerrar dependencias | `BLOQUEANTE` |

## Bloqueos conservados

- CT-CUM-06 permanece abierto y no aprobado.
- CT-CUM-07 a CT-CUM-10 conservan sus puertas propias.
- CT-CUM-02 a CT-CUM-05 siguen siendo candidatos locales hasta integración y
  aprobación por dirección y autoridades competentes.
- O4 y los demás carriles técnicos conservan sus NO-GO y dependencias.
- Datos reales, firma, sello, registro, CSV, cotejo, copia auténtica, expurgo,
  actos, altas, envíos, comunicaciones, preproducción y producción siguen
  prohibidos.
- El tablero, métricas y documentos transversales no cambian.

## Criterio técnico de revisión

La revisión independiente solo puede comprobar que el candidato:

1. separa original, derivado, versión firmada, copia, evidencia e índice;
2. exige metadatos, relaciones, huellas y autoridad sin inventar valores;
3. no declara firma, sello, CSV, copia auténtica o acto real;
4. mantiene series, plazos, disposición y expurgo sin decidir;
5. conserva CT-CUM-06 y los actos/documentos reales como bloqueos;
6. no contiene personas, documentos, certificados, proveedores, secretos,
   ubicaciones o infraestructura reales;
7. resuelve enlaces locales, cita una base Git existente y supera
   `git diff --check`.

Un `GO` documental no cierra CT-CUM-06, no aprueba política ENI y no autoriza
ningún efecto documental.
