# Guion de demostración para RRHH — septiembre de 2026

Este guion separa lo publicado y recorrible de los ejemplos sintéticos y de lo
pendiente. El punto de entrada de la reunión es `https://cidonia.cloud/portal-empleado/`:
responde en esta fecha, pero las pantallas B-BACK de situaciones, contactos y nuevo
llamamiento siguen apagadas allí mientras Dirección corrige su arranque. No se deben
simular ni afirmar como demostradas hasta que Dirección las reactive.

## Orden de la demostración

1. Abrir el portal de empleado de cidonia y recorrer las lecturas que estén accesibles.
2. Mostrar el cuadro de Bolsa y la lista B5/B6 únicamente si el servidor entrega sus
   contratos reales y autorizados.
3. Cuando Dirección reactive B-BACK, abrir el recorrido B7: bolsa, candidatos ordenados,
   configuración y revisión/envío. La selección masiva de este corte está preparada en
   código, pero no se presenta como publicada hasta su despliegue.
4. Cerrar con los límites: correo corporativo, avisos, reglas temporales, documentos
   administrativos finales e integración GINPIX requieren trabajo o decisión adicional.

## Petición, pantalla y estado

| Petición RRHH | Pantalla y qué pulsar | Qué se enseña hoy | Pendiente y responsable | Estado |
| --- | --- | --- | --- | --- |
| 1. Gestión de bolsas y candidatos | Bolsa → Candidatos (B5) → abrir una bolsa | Lectura de bolsa, candidatura y situación, cuando B-BACK esté activo; no usar datos de ejemplo como operación real. | Activar B-BACK en cidonia: Dirección. | Parcial, pantalla apagada en cidonia. |
| 2. Llamamientos automáticos | Bolsa → Nuevo llamamiento (B7) | Emisión real B7 se prepara por pasos sobre el orden B6; no decide automáticamente a quién llamar. | Reglas aprobadas y automatización: RRHH; implementación: Desarrollo. | Parcial. |
| 3. Contratos, ceses y reincorporaciones | No hay pantalla de demostración | No se muestra una operación inexistente. | Reglas, autoridad e integración de Personal: RRHH y Desarrollo. | Pendiente. |
| 4. Motor de reglas configurable | B6 → ver rótulo de política provisional | Se muestra el orden vigente y que sus parámetros son provisionales. | Dudas 13–14 y gobierno de reglas: RRHH; motor versionado: Desarrollo. | Parcial. |
| 5. Portal seguro de candidato | Portal del empleado → Mi bolsa | La web consulta primero su fuente real; el acceso externo de candidato sigue pendiente de construir en VEC. | Superficie externa con certificado/DNIe: Desarrollo. | Parcial, sin acceso real de candidato. |
| 6. Cuadro de mando | Bolsa → Cuadro de mando / Estadísticas | Indicadores agregados conectados si el contrato está disponible; no habilitan acciones. | Arranque y autorización en cidonia: Dirección. | Parcial. |
| 7. Estadísticas | Bolsa → Estadísticas → abrir una cifra | Desglose agregado por bolsa y estado; cada cifra abre la lista filtrada. | Fuente autorizada disponible en servidor: Dirección. | Parcial. |
| 8. Word y PDF | Documentos | No se atribuye generación automática de plantillas Bolsa. | Plantillas, custodia, firma y conector: Desarrollo con criterios RRHH. | Pendiente. |
| 9. Correo, SMS y mensajería | B7 → Configurar llamamiento | B7 usa relay de desarrollo y rotula que no acredita buzón corporativo; SMS/WhatsApp no se muestran. | Correo corporativo y política/canales: Dirección/RRHH; conectores: Desarrollo. | Parcial. |
| 10. Auditoría y trazabilidad | B5 → Histórico; B7 → recibo tras emitir | Las operaciones conectadas conservan recibo e historial; el ejemplo no acredita una auditoría completa de todos los módulos. | Extender la cobertura por proceso y política IP/equipo: Desarrollo, Seguridad y RRHH. | Parcial. |
| Histórico de contratos, llamamientos, renuncias, sanciones, estados, correos y documentos | B5 → Histórico; B2/B3/B7 cuando estén activos | Histórico de contactos y llamamientos B3/B7 existe en su recorrido; no se suplanta histórico de contratos, sanciones o documentos. | Resto de históricos: Desarrollo; criterios de retención: RRHH/Seguridad. | Parcial. |
| Estados: disponible, trabajando, no disponible, pendiente, renuncia, excluido y desde fecha | Bolsa → Candidatos → filtro / ficha → cambiar situación | Catálogo B2 y cambio con motivo y recibo, condicionado a que B-BACK esté activo. | Activar la pantalla en cidonia: Dirección; transiciones definitivas: RRHH. | Parcial, pantalla apagada en cidonia. |
| Reglas de cinco y nueve meses sin hojas de cálculo | No hay pantalla de demostración | No se inventa el cómputo temporal ni se muestran alertas ficticias. | Regla, vigencia y excepciones: RRHH; cálculo reproducible: Desarrollo. | Pendiente. |
| Portal personal: bolsa, posición, estado, último llamamiento, contratos y disponibilidad | Portal del empleado → Mi bolsa | La pantalla diferencia datos reales de fallback sintético; no concede acceso real de candidato. | Identidad externa y lecturas autorizadas: Desarrollo. | Parcial. |
| Zona pública de estados | No hay pantalla de demostración | No se publica identidad ni situación individual sin decisión de minimización. | Alcance publicable y seudonimización: RRHH, Secretaría, Jurídica y DPD. | Pendiente. |
| Cuadro interno: bolsas, candidatos y estados | Bolsa → Cuadro de mando / Estadísticas | Muestra agregados autorizados, no identidades ni acciones administrativas. | Disponibilidad de B-BACK en cidonia: Dirección. | Parcial. |
| Envío masivo por estado | B7 → paso 2 → «Seleccionar todas las que cumplen el filtro» | Selecciona todas las participaciones del filtro en el orden B6; si superan 100, conserva las cien primeras y lo rotula. | Publicar el corte en cidonia: Dirección. | Preparado, pendiente de despliegue. |
| Personalización de la comunicación | B7 → paso 3 | El asunto y texto son comunes a todas las personas; el rótulo visible dice que no hay personalización. | Diseño y desarrollo por persona: Desarrollo; contenido/plantillas: RRHH. | Pendiente. |
| Oferta, solicitud web, orden y llamamiento directo | B7 → pasos 2–4 | Se emite una comunicación por el orden B6; no se presenta como solicitud del candidato ni como llamamiento directo automático. | Reglas de respuesta, salto y directo: RRHH; desarrollo posterior. | Parcial. |
| Avisos de salto de orden y tres años | No hay pantalla de demostración | No se sustituyen por avisos de ejemplo. | D3-AV: Módulos; reglas y destino del aviso: RRHH. | En desarrollo. |
| Plantillas: contrato, nombramiento, toma de posesión, cese, modificación, informes y resoluciones | Documentos | No se muestran documentos sintéticos como administrativos ni firmados. | Plantillas, datos, firma/sello, CSV y custodia: RRHH y Desarrollo. | Pendiente. |
| Coste por categoría e integración GINPIX/SAVIA o alternativa | No hay pantalla de demostración | No se calcula ni muestra coste aparente. | Fuente admitida, contrato y autorización: RRHH/Dirección; conector: Desarrollo. | Pendiente. |
| Auditoría: autor, fecha/hora, antes/después, motivo, expediente e IP/equipo | B2/B3/B7 → recibo e histórico | Se enseñan recibo e historia de las operaciones disponibles; IP/equipo solo se incorporará con política de seguridad. | Política y modelo probatorio completo: Seguridad/RRHH; extensión técnica: Desarrollo. | Parcial. |

## Frases de cierre

- «Lo visible como conectado solo se afirma cuando la pantalla obtiene su contrato real y autorizado.»
- «El relay de desarrollo no acredita envío ni entrega en correo corporativo.»
- «La selección masiva no altera el orden: si hay más de cien destinatarios, se realizan envíos separados con las cien primeras personas de cada orden.»
- «Un certificado identifica el acceso; no equivale a firma de un documento administrativo.»
