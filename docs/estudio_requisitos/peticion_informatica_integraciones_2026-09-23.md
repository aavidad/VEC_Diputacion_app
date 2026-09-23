# Petición a Informática — integraciones de VEC

23/09/2026. Complementa `dudas.md` (preguntas funcionales a RRHH): aquí solo lo técnico que
necesitamos de Informática de la Diputación para conectar VEC con los sistemas corporativos.
No pedimos credenciales, certificados ni datos personales reales por correo: basta con la
documentación, un entorno de pruebas y la persona responsable de cada sistema.

## Texto para enviar

Buenos días:

VEC va a sustituir a CONVOCA en la gestión de bolsas y en los procesos selectivos, y a integrar
la contratación temporal con los sistemas corporativos. Para conectar cada sistema necesitamos,
de cada uno: responsable técnico, documentación de la interfaz (versión vigente), entorno de
pruebas con datos ficticios y el procedimiento para dar de alta a VEC como aplicación cliente.

1. **Sede electrónica (MOAD).** Inscripción telemática en procesos selectivos: ¿cómo registra
   la Sede una solicitud presentada desde otra aplicación (servicio, formato del asiento, justificante
   de registro, número y fecha)? ¿Cómo se identifica el interesado (Cl@ve, certificado, DNIe) y qué
   datos de identidad recibe VEC? ¿Cómo se notifican subsanaciones y resoluciones al aspirante?
2. **Pasarela de pago de tasas.** Interfaz para iniciar el pago de la tasa de examen, confirmación
   del cobro (síncrona o por aviso), referencia y justificante, devoluciones, exenciones y
   conciliación. ¿Qué sistema es la fuente de verdad del pago?
3. **Portafirmas.** Interfaz para enviar documentos a firma, orden de firmantes, estados, devolución
   y obtención del documento firmado con su evidencia.
4. **GINPIX.** Interfaz para comunicar altas, variaciones y ceses de personal temporal y la
   respuesta que confirma la recepción.
5. **Correo corporativo (SMTP).** Servidor, puerto, TLS, cuenta remitente y límites de envío para
   las comunicaciones a candidatos; y, si existe, servicio de **SMS** corporativo.
6. **Identidad corporativa y revocación.** Directorio o IdP de empleados (altas, cambios, bajas),
   autoridades de certificación admitidas y servicio de revocación (OCSP/CRL).

Gracias. Podemos concertar una reunión breve por sistema.
