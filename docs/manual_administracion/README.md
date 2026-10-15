# Manual de administración de VEC

**Alcance comprobado:** `main` el 15 de octubre de 2026. La versión anterior de este manual (26 de septiembre) describía la consola de administración como cerrada; desde entonces existe `vec-admin`. VEC se usa con datos sintéticos y la producción con datos reales sigue en **NO-GO**. El estado de cada tarea de administración está en el [flujo de la aplicación, apartado 4.4](../producto/FLUJO_APP.md), que prevalece si hay diferencias.

## Dos responsabilidades distintas

| Responsabilidad | Qué gobierna | Límite |
| --- | --- | --- |
| **Administrador de Aplicación** | Asigna perfiles y cargos, lleva catálogos y configuración funcional, activa módulos y consulta la auditoría. | No ve expedientes ni bolsas. Nadie se asigna a sí mismo un perfil de administrador. |
| **Administrador de Sistemas** | Instala, gestiona versiones, hace copias y propone restauraciones. | Sin datos personales ni facultad de dar perfiles. Una restauración la propone Sistemas y la aprueba Aplicación. |
| **Gobierno funcional de RRHH** (fuera de Administración) | Los catálogos funcionales de RRHH (plazos de Contratación temporal, política de ofertas y de cese de Bolsa, baremo, plantillas) se editan desde el portal de RRHH con versión, motivo, recibo y auditoría. | Un perfil de RRHH no administra cuentas técnicas, claves o infraestructura. Preparar una regla no equivale a publicarla, ni tramitar un expediente equivale a firmarlo. |

Los primeros administradores se crean en la instalación (arranque «2+1»: dos de Aplicación y uno de Sistemas); los siguientes requieren dos personas, y siempre queda al menos uno de cada tipo.

Cada módulo conserva la autoridad sobre sus propios datos: Contratación coordina el expediente; Bolsa conserva orden y llamamientos; Personal conserva sus relaciones y datos propios. Identidad, autorización y auditoría son servicios comunes. Esta separación permite cambiar un catálogo o ampliar un módulo sin atribuir a ADMIN una decisión funcional que corresponde a RRHH. El [contrato de módulos](../portal_vec/contrato_modulos_vec.md) distingue el registro de un módulo de su instalación, activación y exposición por red.

## Acceso y permisos

La entrada administrativa prevista exige **DNIe o certificado digital admitido y vigente**, comprobación de revocación y **permiso administrativo nominal, exacto y vigente**. Autenticarse solo identifica al actor: no firma documentos ni concede por sí mismo acceso a ADMIN. En producción la superficie administrativa debe estar separada del portal RRHH y restringida a la red de gestión. La entrada de pruebas prevista por Internet conserva certificado, revocación y una lista privada de identidades autorizadas. Las cuentas administrativas son nominativas y separadas de la cuenta ordinaria. SSH sirve para operar el servidor; no es un factor de acceso al portal.

**Estado a 15/10:** `vec-admin` funciona en un subdominio propio, en cidonia desde el 10/10. Exige TLS 1.3 con certificado de la CA de administración, lista de revocación leída en cada petición, nombre de host exacto, redes permitidas y cuenta privilegiada; no admite identidad por cabeceras. Solo arranca en `desarrollo` o `cidonia`; en producción se niega hasta tener Kerberos. Tiene selector de perfil activo, auditado y guardado en el servidor. Lo que se puede hacer hoy, y por qué vía:

| Tarea | Cómo se hace | Estado |
| --- | --- | --- |
| Ver personas y su ficha | `/admin/usuarios/`, lista y ficha | En cidonia |
| Asignar o retirar un perfil | Ficha → perfil, unidad, vigencia y motivo → revisar y confirmar | En `main`; en cidonia *(por verificar)* |
| Dar o quitar el perfil de administrador; alta y baja de Intervención | Propuesta de una persona y cierre por otra distinta | Solo API, sin pantalla |
| Publicar una versión nueva de un rol | Propuesta y cierre por dos administradores | API y línea de órdenes |
| Admitir o ratificar catálogos de acciones | `vec-catalogo-acciones-admitir`, `vec-ratificar-catalogo-admin`, con aprobación por huella | Línea de órdenes |
| Publicar cargos de firma | `vec-publicar-cargos-firma`; la asignación, por pantalla | Línea de órdenes |
| Activar o desactivar módulos | `/admin/modulos/` solo prepara; se aplica con `VEC_PORTAL_MODULOS_VISIBLES` y reinicio | Pantalla sin montar |
| Copias de seguridad | `vec-copias-*`; la pantalla `/admin/copias/` no tiene página ni ruta | Aparcado |
| Consultar la auditoría por persona | Módulo de auditoría del portal | Falta encenderla (tarea F0-01) |

Tras cambiar la versión de su rol, cada administrador vuelve a elegir perfil. La confianza de gobierno de los administradores dura hoy 24 horas y se renueva con claves nuevas por dos administradores; su vida configurable está en la tarea F0-03. Un menú visible, un certificado válido o una respuesta `403` no prueban por sí solos que la frontera de red y autorización esté desplegada.

Para cualquier capacidad funcional conectada, el servidor debe comprobar un único perfil activo, la acción, el recurso, el ámbito, la finalidad y los campos permitidos. No suma perfiles ni interpreta un ámbito vacío como acceso general. Si falta una concesión o falla su autoridad, la operación se deniega. El [flujo de la aplicación, apartado 6](../producto/FLUJO_APP.md) describe la política completa; su tabla de perfiles es una línea base pendiente de validación institucional, no una lista de cuentas activas.

## Catálogos y versiones

VEC contiene catálogos y reglas con referencias de versión y procedencia. Por ejemplo, el corte de presentación conserva un **paquete de reglas de ejemplo** para las decisiones RRHH aún pendientes; cambiar ese paquete requiere el circuito de revisión y despliegue correspondiente. Algunos catálogos de Personal se consultan desde la API. La existencia de esos lectores, ficheros o pantallas no habilita su edición desde Administración: los catálogos funcionales se cambian desde el portal de RRHH y el catálogo de acciones por la línea de órdenes protegida.

El gobierno previsto para un cambio funcional es: identificar fuente y responsable, preparar una nueva versión, comprobar el efecto sobre expedientes, revisar y aprobar según la competencia aplicable, publicar, y conservar la versión anterior y su huella. Un expediente conserva la referencia de la regla aplicada; una corrección posterior no reescribe silenciosamente decisiones previas. Las reglas ordinarias son declarativas y tipadas: no se admite código o SQL enviado desde un formulario de administración. Véanse los [principios de reglas configurables](../estudio_requisitos/analisis_integral_rrhh.md) y el [flujo de Contratación](../producto/FLUJO_APP.md).

## Separación de funciones, revocaciones y auditoría

- Una misma persona no satisface un doble control cambiando de perfil. Preparación, revisión, publicación, firma y auditoría requieren las competencias y personas que marque cada circuito.
- La baja, el cambio de unidad o la retirada de una función deben revocar las asignaciones afectadas. Las autorizaciones se revalidan antes del efecto; una política retirada, caducada o no disponible deniega. La retirada de un perfil se hace desde la ficha de la persona en `vec-admin`, con confirmación y recibo; una asignación retirada no se revive, se otorga una nueva. La retirada de un administrador exige propuesta y cierre por dos personas.
- La auditoría debe permitir reconstruir actor, acción, recurso, resultado, instante, correlación y versión sin copiar secretos ni documentos completos. El acceso a auditoría requiere permiso específico. Los registros de seguridad y la auditoría funcional tienen destinos y custodios diferenciados; quien configura una política no debe poder alterar su evidencia.

Estos controles forman parte de la [política de acceso interno](../estudio_requisitos/acceso_interno_tecnicos_administracion.md), la [matriz de roles](../producto/FLUJO_APP.md) y la [seguridad de PostgreSQL](../portal_vec/seguridad_persistencia_postgresql.md). La existencia de contratos y pruebas de autorización o auditoría en `main` no acredita por sí sola una consola administrativa completa, una certificación normativa ni la instalación de todos esos controles en producción.

## Incidencias y recuperación

Si una acción funcional devuelve denegación, compruebe con el responsable del procedimiento la identidad activa, la asignación vigente, la unidad y la finalidad; no cambie permisos o datos directamente en la base para sortearla. Ante un servicio de identidad, políticas o auditoría indisponible, preserve el error y la correlación para soporte: el fallo no habilita la operación.

Al reintentar una operación de negocio que ya pudo surtir efecto, use la **misma clave de operación** y consulte el recibo e historial antes de crear otra. Los recorridos sintéticos de Bolsa y Contratación documentados en el [estado del proyecto](../../ESTADO_PROYECTO.md) comprobaron recuperación de recibos tras reiniciar aplicación y PostgreSQL; eso no equivale a una restauración integral de la plataforma. Las copias, migraciones y recuperación de servicios requieren el procedimiento técnico aprobado, conservación de historia, ACL y verificación del estado antes de intervenir. No se deben revertir migraciones instaladas sobre expedientes conservados.

## Ventajas y límites del diseño

La arquitectura hexagonal separa reglas, casos de uso y adaptadores. Esto permite que web, un cliente de escritorio o CLI puedan invocar los **mismos casos de uso y controles**, sin duplicar reglas por interfaz. Hoy esa igualdad es una regla arquitectónica y **no acredita clientes de escritorio o CLI administrativos operativos**. Los módulos y los catálogos versionados conservan propietario y procedencia; las concesiones exactas y la auditoría hacen explicables las decisiones y reducen accesos accidentales. Los recibos y la historia facilitan revisar y recuperar operaciones acreditadas.

Quedan pendientes las pantallas de doble control (dar o quitar administradores), de módulos y de copias; la auditoría por persona encendida; la fuente durable auditada del selector de perfil (sin ella responde 503); y la validación institucional de perfiles y políticas. La presentación usa datos sintéticos y no demuestra firma legal, envío acreditado, cumplimiento normativo ni autorización para datos reales.
