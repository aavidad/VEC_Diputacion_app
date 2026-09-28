# Manual de administración de VEC

**Alcance comprobado:** `main` en `7247682cbd1e6e630c86c290e3ddeca281456a94`, 26 de septiembre de 2026. VEC se usa para una presentación con datos sintéticos; la producción con datos reales sigue en **NO-GO**. Este manual distingue las capacidades conectadas de las reglas de gobierno previstas. La [situación del proyecto](../../ESTADO_PROYECTO.md) prevalece si cambia el despliegue.

## Dos responsabilidades distintas

| Responsabilidad | Qué gobierna | Límite |
| --- | --- | --- |
| **ADMIN de sistema y seguridad** | Identidades y asignaciones, políticas técnicas, conectores, claves, despliegue y observación de servicios, según una concesión específica. | Operar la plataforma no concede acceso funcional a expedientes, documentos o decisiones de RRHH. La superficie ADMIN segregada y sus acciones de configuración **aún no están conectadas**. |
| **Gobierno funcional de RRHH** | Procedimientos, catálogos, reglas, plantillas y decisiones del módulo que le corresponda, dentro de su unidad, fase y finalidad. | Un perfil de RRHH no administra cuentas técnicas, claves o infraestructura. Preparar una regla no equivale a publicarla, ni tramitar un expediente equivale a firmarlo. |

Cada módulo conserva la autoridad sobre sus propios datos: Contratación coordina el expediente; Bolsa conserva orden y llamamientos; Personal conserva sus relaciones y datos propios. Identidad, autorización y auditoría son servicios comunes. Esta separación permite cambiar un catálogo o ampliar un módulo sin atribuir a ADMIN una decisión funcional que corresponde a RRHH. El [contrato de módulos](../portal_vec/contrato_modulos_vec.md#criterio-vigente-de-ampliación--19-de-septiembre-de-2026) distingue el registro de un módulo de su instalación, activación y exposición por red.

## Acceso y permisos

La entrada administrativa prevista exige **DNIe o certificado digital admitido y vigente**, comprobación de revocación y **permiso administrativo nominal, exacto y vigente**. Autenticarse solo identifica al actor: no firma documentos ni concede por sí mismo acceso a ADMIN. En producción la superficie administrativa debe estar separada del portal RRHH y restringida a la red de gestión. La entrada de pruebas prevista por Internet conserva certificado, revocación y una lista privada de identidades autorizadas. Las cuentas administrativas son nominativas y separadas de la cuenta ordinaria. SSH sirve para operar el servidor; no es un factor de acceso al portal.

**Estado en este corte:** la entrada ADMIN de pruebas permanece cerrada. El módulo tiene un manifiesto y una vista web de estado, pero las secciones de roles, catálogos, calendarios, reglas, conectores, módulos y privacidad indican «No configurado» y sus acciones están deshabilitadas. La elección de apariencia es solo una vista previa local; no publica un tema. Por tanto, este manual no ofrece pasos de alta, publicación o revocación desde una consola ADMIN operativa. Un menú visible, un certificado válido o una respuesta `403` tampoco prueban que la frontera de red y autorización esté desplegada.

Para cualquier capacidad funcional conectada, el servidor debe comprobar un único perfil activo, la acción, el recurso, el ámbito, la finalidad y los campos permitidos. No suma perfiles ni interpreta un ámbito vacío como acceso general. Si falta una concesión o falla su autoridad, la operación se deniega. La [matriz de roles y ámbitos](../portal_vec/matriz_roles_y_ambitos.md) describe la política completa; su tabla de perfiles es una línea base pendiente de validación institucional, no una lista de cuentas activas.

## Catálogos y versiones

VEC contiene catálogos y reglas con referencias de versión y procedencia. Por ejemplo, el corte de presentación conserva un **paquete de reglas de ejemplo** para las decisiones RRHH aún pendientes; cambiar ese paquete requiere el circuito de revisión y despliegue correspondiente. Algunos catálogos de Personal se consultan desde la API. La existencia de esos lectores, ficheros o pantallas **no habilita una edición o publicación desde ADMIN**.

El gobierno previsto para un cambio funcional es: identificar fuente y responsable, preparar una nueva versión, comprobar el efecto sobre expedientes, revisar y aprobar según la competencia aplicable, publicar, y conservar la versión anterior y su huella. Un expediente conserva la referencia de la regla aplicada; una corrección posterior no reescribe silenciosamente decisiones previas. Las reglas ordinarias son declarativas y tipadas: no se admite código o SQL enviado desde un formulario de administración. Véanse los [principios de reglas configurables](../estudio_requisitos/analisis_integral_rrhh.md#7-principios-para-las-reglas-configurables) y el [flujo de Contratación](../portal_vec/expediente_contratacion_temporal_rrhh.md).

## Separación de funciones, revocaciones y auditoría

- Una misma persona no satisface un doble control cambiando de perfil. Preparación, revisión, publicación, firma y auditoría requieren las competencias y personas que marque cada circuito.
- La baja, el cambio de unidad o la retirada de una función deben revocar las asignaciones afectadas. Las autorizaciones se revalidan antes del efecto; una política retirada, caducada o no disponible deniega. **No hay en este corte un botón ADMIN operativo para ejecutar esa revocación.** La actuación debe pasar por la autoridad de identidades y el procedimiento aprobado, con evidencia del resultado.
- La auditoría debe permitir reconstruir actor, acción, recurso, resultado, instante, correlación y versión sin copiar secretos ni documentos completos. El acceso a auditoría requiere permiso específico. Los registros de seguridad y la auditoría funcional tienen destinos y custodios diferenciados; quien configura una política no debe poder alterar su evidencia.

Estos controles forman parte de la [política de acceso interno](../estudio_requisitos/acceso_interno_tecnicos_administracion.md), la [matriz de roles](../portal_vec/matriz_roles_y_ambitos.md) y la [seguridad de PostgreSQL](../portal_vec/seguridad_persistencia_postgresql.md). La existencia de contratos y pruebas de autorización o auditoría en `main` no acredita por sí sola una consola administrativa completa, una certificación normativa ni la instalación de todos esos controles en producción.

## Incidencias y recuperación

Si una acción funcional devuelve denegación, compruebe con el responsable del procedimiento la identidad activa, la asignación vigente, la unidad y la finalidad; no cambie permisos o datos directamente en la base para sortearla. Ante un servicio de identidad, políticas o auditoría indisponible, preserve el error y la correlación para soporte: el fallo no habilita la operación.

Al reintentar una operación de negocio que ya pudo surtir efecto, use la **misma clave de operación** y consulte el recibo e historial antes de crear otra. Los recorridos sintéticos de Bolsa y Contratación documentados en el [estado del proyecto](../../ESTADO_PROYECTO.md) comprobaron recuperación de recibos tras reiniciar aplicación y PostgreSQL; eso no equivale a una restauración integral de la plataforma. Las copias, migraciones y recuperación de servicios requieren el procedimiento técnico aprobado, conservación de historia, ACL y verificación del estado antes de intervenir. No se deben revertir migraciones instaladas sobre expedientes conservados.

## Ventajas y límites del diseño

La arquitectura hexagonal separa reglas, casos de uso y adaptadores. Esto permite que web, un cliente de escritorio o CLI puedan invocar los **mismos casos de uso y controles**, sin duplicar reglas por interfaz. Hoy esa igualdad es una regla arquitectónica y **no acredita clientes de escritorio o CLI administrativos operativos**. Los módulos y los catálogos versionados conservan propietario y procedencia; las concesiones exactas y la auditoría hacen explicables las decisiones y reducen accesos accidentales. Los recibos y la historia facilitan revisar y recuperar operaciones acreditadas.

Quedan pendientes la superficie ADMIN segregada y conectada, sus fuentes duraderas de configuración, el circuito de publicación y revocación, y la validación institucional de perfiles y políticas. La presentación usa datos sintéticos y no demuestra firma legal, envío acreditado, cumplimiento normativo ni autorización para datos reales.
