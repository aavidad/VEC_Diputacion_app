# Registro común de intentos fallidos de copias

`Auditor.Registrar` conecta el callback HTTP con `RegistradorIntentosAuditoria`.
La composición aporta una fuente confiable del intento exacto por correlación,
el registrador común, el auditor de frontera anterior a sesión y la configuración
del servidor. El paquete no monta endpoints, no configura SQL y no concede permisos.

La fuente devuelve la correlación original, el resultado de contexto registrado
y su vínculo V2. Se cotejan el par, la Persona, el perfil y el canal ADMIN. Se puede
conservar la acreditación original de un intento revocado sin conceder una sesión
nueva ni exigir permiso vigente para registrar el fallo. Una consulta únicamente
por Persona y perfil no cumple este contrato.

La configuración exige motivos de catálogo para los siete códigos HTTP cerrados
y acción, finalidad y recurso de reserva para las siete operaciones fijas.
Las rutas desconocidas con identidad resuelta usan `FronteraNominal` y conservan
esa identidad. Antes de sesión se llama al auditor no nominal separado sin fabricar
Persona. El proceso, canal y referencias no los elige la petición. El canal debe
ser `administracion_privilegiada`; el plazo debe ser positivo y como máximo
30 segundos. Un recurso HTTP que no cumple el contrato común se registra con
el recurso opaco de reserva configurado, sin perder el intento ni conservar
el valor rechazado.

El registro usa `WithoutCancel` y un plazo propio que abarca fuente, append y
frontera. Forma la orden común y ejecuta un solo append, sin reintentar efectos.
Sólo un acuse que supera `ValidarPara` confirma el registro. Los errores devuelven
`httpcopias.ErrNoDisponible`, que la frontera traduce a 503, sin causas del proveedor
ni serialización de las estructuras opacas.

Quedan pendientes la fuente operativa de K y los catálogos y registrador de L,
su configuración por servidor y las revisiones sensibles del conjunto final.
Las fábricas selladas del paquete de pruebas sólo se usan en pruebas automatizadas;
no son una fuente nominal para producción. El callback no sustituye la auditoría
atómica de los efectos de negocio ni demuestra persistencia SQL por sí solo.
