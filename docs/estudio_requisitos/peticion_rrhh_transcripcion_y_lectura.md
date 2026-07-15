# Petición de RRHH: transcripción accesible y lectura técnica inicial

Fuente: [`Peticion.pdf`](../../Peticion.pdf), cuatro páginas escaneadas sin capa de
texto. Transcripción realizada mediante lectura visual el 14 de julio de 2026. El PDF
original se conserva sin modificaciones y prevalece ante cualquier discrepancia.

## Transcripción

### Página 1 — Propuesta del Servicio de selección externa

1. Gestión de bolsas y candidatos.
2. Llamamientos automáticos según bases y Reglamento de bolsas.
3. Gestión de contratos, ceses y reincorporaciones.
4. Motor de reglas configurable, sin necesidad de programar cada cambio normativo.
5. Portal de consulta para candidatos con acceso seguro.
6. Cuadro de mando para responsables y dirección.
7. Estadísticas y explotación de datos.
8. Generación automática de documentos Word y PDF.
9. Integración con correo electrónico, SMS y, si es viable, mensajería instantánea.
10. Registro completo de auditoría y trazabilidad para garantizar la transparencia y el
    control interno.

Como ejemplo de estructura de datos se proponen:

- **Bolsas:** identificador, nombre, categoría, vigencia, resolución de aprobación y
  orden de llamamiento.
- **Candidatos:** bolsa, DNI, nombre, apellidos, correo electrónico y dos teléfonos.
- **Situación en bolsas:** bolsa, candidato, posición, estado actual, fecha de
  disponibilidad y observaciones.

### Página 2 — Histórico, estados y transparencia

Se solicita un histórico de:

- contratos;
- llamamientos;
- renuncias;
- sanciones;
- cambios de estado;
- correos enviados;
- documentos generados.

Como ejemplos de estados automáticos se citan: disponible, trabajando, no disponible,
pendiente de incorporación, renuncia, excluido y disponible a partir de una fecha.

El documento pide que la aplicación ejecute reglas temporales sin recurrir a hojas de
cálculo. Incluye dos ejemplos:

- contrato terminado el 31/07/2026 y periodo de cinco meses de indisponibilidad;
- finalización de una acumulación de tareas y periodo de nueve meses.

En materia de consulta se proponen:

- un portal personal en el que cada persona vea exclusivamente su bolsa, posición,
  estado, último llamamiento, contratos y fecha de disponibilidad;
- una zona pública para consultar el estado de integrantes de una bolsa: disponible,
  trabajando, excluido, no disponible, ocupado, etc.

El original plantea DNI y clave personal como acceso. Este mecanismo se considera un
ejemplo funcional antiguo y **no un requisito técnico válido de autenticación**.

### Página 3 — Control interno, comunicaciones y avisos

El cuadro de control interno debe permitir conocer, entre otros datos:

- número de bolsas activas;
- número de candidatos por bolsa;
- personas disponibles;
- personas trabajando;
- personas excluidas;
- personas no disponibles.

Se solicita envío masivo y personalizado de comunicaciones seleccionando destinatarios
por estado. El ejemplo informa de una oferta por acumulación de tareas o sustitución,
permite solicitarla en la web, respeta el orden del reglamento y prevé un llamamiento
directo si no se cubre. También se contemplan SMS o WhatsApp cuando la normativa y la
integración lo permitan.

Como avisos automáticos se citan:

- que se haya saltado a una persona disponible en el orden de una bolsa;
- que dentro de un mes una persona alcance tres años de trabajo sin interrupción.

### Página 4 — Documentos y auditoría

Se solicita generación automática mediante plantillas de:

- contrato laboral;
- nombramiento;
- toma de posesión;
- cese;
- modificación de nombramiento;
- informes;
- resoluciones;
- otros documentos configurados.

Las plantillas deben combinar campos, incluido el coste aproximado por categoría, con
datos de GINPIX de SAVIA o de un sistema alternativo.

La trazabilidad debe registrar cada acción con:

- autor del cambio;
- fecha y hora exactas;
- valor anterior y nuevo;
- motivo;
- documento o expediente relacionado;
- dirección IP o equipo, cuando la política de seguridad lo permita.

El objetivo expresado es justificar cualquier modificación e impedir alteraciones sin
rastro.

## Lectura técnica inicial

### Requisitos que se conservan íntegramente

- Motor de reglas funcionalmente configurable y versionado.
- Ciclo completo de la bolsa, incluido el llamamiento y la incorporación, no solo la
  inscripción.
- Histórico inmutable y reconstruible.
- Detección de saltos del orden y de vencimientos.
- Portal propio del candidato y cuadro operativo interno.
- Plantillas documentales configurables.
- Comunicaciones segmentadas y avisos automáticos.
- Auditoría con actor, instante, antes/después, motivo y expediente.
- Integración con el sistema de personal existente o con su sustituto mediante un
  conector.

### Aspectos que deben elevarse a un diseño profesional

1. **Persona única.** No se repetirá un candidato completo dentro de cada bolsa. Se
   modelarán persona, participación en bolsa, posición, preferencias y situación como
   conceptos relacionados y versionados.
2. **DNI protegido.** El DNI/NIE no será clave técnica ni contraseña. Se utilizará un
   identificador interno y proveedores de identidad admitidos.
3. **Estados configurables pero gobernados.** Los estados y transiciones tendrán versión,
   vigencia, condiciones, simulación, revisión, aprobación y trazabilidad. No serán una
   lista estática ni texto libre.
4. **Reglas reproducibles.** Cada transición automática conservará la versión del
   reglamento, la regla, los datos de entrada, el cálculo, el resultado y las excepciones.
5. **Publicación minimizada.** La zona pública no expondrá identidades y estados
   individuales por defecto. Se definirá con RRHH, Secretaría, asesoría jurídica y DPD qué
   datos son legalmente publicables, durante cuánto tiempo y con qué seudonimización.
6. **Aviso no equivale a notificación.** Correo, SMS, Telegram o mensajería serán
   conectores de aviso salvo que el canal cumpla formalmente los requisitos de una
   notificación administrativa.
7. **Llamamiento automático con control humano.** El sistema propondrá y explicará la
   siguiente persona según reglas congeladas; las excepciones, saltos, sanciones y
   exclusiones requerirán competencia, motivo y evidencia.
8. **Documentos, no solo Word.** Las plantillas podrán producir formatos editables cuando
   sea necesario, pero el documento administrativo final tendrá versión cerrada, firma o
   sello, CSV, metadatos, registro y custodia según proceda.
9. **Auditoría probatoria.** La IP será solo un dato auxiliar sujeto a política. La evidencia
   principal incluirá identidad efectiva, perfil, ámbito, autorización, acción, objeto,
   versión, resultado, correlación y sello temporal, con protección contra alteración.
10. **Estadística separada.** Los cuadros operativos, los informes de dirección y la
    publicación estadística utilizarán proyecciones diferentes y datos minimizados.

## Correspondencia de dominio propuesta

| Concepto de la petición | Modelo objetivo |
| --- | --- |
| Candidato repetido por bolsa | Persona canónica + perfil candidato + participación en bolsa |
| Situación actual | Periodo de situación con inicio, fin, causa, regla, resolución y evidencia |
| Posición | Instantánea versionada de lista y posición, con criterio de desempate |
| Orden de llamamiento | Política de llamamiento versionada + ejecución reproducible |
| Correos enviados | Comunicación, canal, plantilla, destinatario, intentos, entrega y relación con expediente |
| Documento generado | Documento versionado, plantilla aplicada, datos de entrada, firma/sello, CSV y huella |
| Observaciones | Actuación o anotación tipificada, con acceso restringido; no un campo libre universal |
| Regla de cinco o nueve meses | Regla con vigencia, unidad temporal, calendario, condición, simulación y prueba |
| Cuadro de control | Proyección agregada autorizada, separada de los datos operacionales |

Esta lectura conserva el propósito de RRHH, pero evita que el ejemplo inicial de tablas o
autenticación se convierta accidentalmente en una limitación del portal transversal.
