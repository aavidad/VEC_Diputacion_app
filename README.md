# VEC · Diputación de Granada

VEC es un portal modular para los procedimientos de personal de la Diputación de Granada. Reúne trabajo de Recursos Humanos, gestión de bolsas y consultas de las personas interesadas en superficies con competencias distintas. El repositorio contiene una aplicación Go, una interfaz web y adaptadores PostgreSQL. **Su estado es de desarrollo y presentación con datos sintéticos; no acredita una tramitación administrativa en producción.**

## Qué aporta el proyecto

- **Continuidad del expediente.** Los recorridos conectados de Contratación temporal conservan versiones, actuaciones y recibos; las operaciones recuperables evitan duplicar efectos al repetir una petición con su clave.
- **Responsabilidades claras.** Contratación coordina el expediente; Bolsa conserva sus convocatorias, candidaturas, orden y llamamientos; Personal conserva sus relaciones y datos propios. Se intercambian referencias y capacidades por contratos, sin convertir una pantalla en autoridad sobre los datos de otro módulo.
- **Decisiones explicables.** Los catálogos configurables y versionados ya conectados, como opciones de Contratación y categorías de Bolsa, fijan las alternativas disponibles. El expediente conserva la procedencia de sus decisiones. No existe un editor universal de todas las reglas del portal.
- **Acceso controlado y trazabilidad.** Las operaciones conectadas exigen una concesión positiva, exacta y vigente en el servidor. El menú y el perfil mostrado no conceden acceso. Los recorridos duraderos documentados conservan historia, auditoría y, donde corresponde, bandeja de salida e idempotencia; las lecturas y efectos tienen alcances distintos.

Estas ventajas describen capacidades presentes en partes del sistema. La disponibilidad de cada operación depende de su composición, permisos, configuración y despliegue; un módulo o una pantalla por sí solos no la acreditan.

## Estado de las áreas

| Área | Disponible en el código de `main` | Límite |
| --- | --- | --- |
| Contratación temporal | Bandeja y expediente; petición del centro, análisis, cobertura con Bolsa, asignación, documentos de desarrollo, fiscalización y actuaciones posteriores con recorridos sintéticos documentados. | Firma, envío corporativo, plazos y eficacia de los actos requieren sus autoridades y validaciones. Un borrador o una aceptación manual de ejercicio no son un documento firmado ni una entrega acreditada. |
| Bolsa | Gestión interna de convocatorias, candidaturas y llamamientos; consulta pública de información publicada. | Las superficies de demostración y los procesos aún parciales no equivalen a una gestión completa de todas las convocatorias. |
| Personal, Cronos y Dietas | Módulos con dominio, interfaces o integraciones de desarrollo de distinto alcance. | Una ficha informativa, un cálculo de ruta o un prototipo de jornada no acreditan nómina, liquidación ni control horario corporativos. |

El [estado del proyecto](ESTADO_PROYECTO.md) distingue código integrado, instalación, recorrido comprobado y asuntos pendientes. La [guía de recorrido](GUIA_RECORRIDO_ALBERTO.md) conserva las pruebas sintéticas y sus límites; sus antecedentes fechados no son instrucciones para crear otra instancia.

## Arquitectura

El diseño es hexagonal: `domain` expresa reglas, `application` coordina casos de uso, `ports` define contratos y `adapters` conecta HTTP, PostgreSQL y proveedores. La composición decide qué adaptadores y rutas se activan. Los módulos se registran en el portal común y mantienen autoridad sobre sus propios datos. Compartir identidad, autorización, auditoría, traducciones y tema evita duplicar esas funciones al ampliar el sistema.

| Ubicación | Contenido |
| --- | --- |
| [internal/modules](internal/modules) | Módulos funcionales, entre ellos Contratación temporal, Bolsa, Personal, Cronos y Dietas. |
| [internal/vec](internal/vec) | Capacidades comunes y contratos del portal. |
| [internal/app](internal/app) | Composición, arranque y servidor. |
| [web/static](web/static) | Portal web, área personal y consulta pública. |
| [deploy/postgresql](deploy/postgresql) | Migraciones y funciones de persistencia por autoridad. |
| [cmd](cmd) | Entradas de servidor y utilidades operativas. |

La separación de casos de uso y transportes permite incorporar otros clientes con las mismas reglas y autorización. **En `main` existe el cliente web y hay comandos de operación acotados; no hay un cliente de escritorio general ni una CLI con paridad funcional con el portal.** El [contrato de módulos](docs/portal_vec/contrato_modulos_vec.md#criterio-vigente-de-ampliación--19-de-septiembre-de-2026) explica el registro real y sus límites: añadir un módulo requiere composición, dependencias y revisión, y no supone instalación dinámica ni publicación automática de sus rutas.

La autorización deniega por defecto: exige actor, acción, recurso, ámbito, finalidad y campos permitidos. PostgreSQL añade identidades técnicas y privilegios separados; la decisión funcional sigue en la autoridad del servidor. La auditoría, las versiones y los recibos permiten comprobar qué ocurrió, sin convertir la autenticación en firma ni una referencia de correo en prueba de entrega. Véanse [roles y ámbitos](docs/portal_vec/matriz_roles_y_ambitos.md), [seguridad y cumplimiento](docs/portal_vec/cumplimiento_y_seguridad.md) y [persistencia PostgreSQL](docs/portal_vec/seguridad_persistencia_postgresql.md).

## Documentación

| Para | Empezar por |
| --- | --- |
| Usar Contratación temporal | [Manual de usuario de Contratación](docs/manual_usuario/manual_contratacion_temporal.md) y [manual funcional de RRHH](docs/manual_rrhh/README.md). |
| Usar Bolsa y el portal | [Manual de usuario del portal y Bolsa](docs/manual_usuario/manual_portal_bolsas.md). |
| Administrar y operar el entorno | [Manual de Sistemas](docs/manual_sistemas/README.md). |
| Entender la administración técnica y el gobierno funcional de RRHH | [Manual de administración](docs/manual_administracion/README.md): alcance y límites actuales; la consola ADMIN aún no está operativa. |
| Desarrollar o integrar módulos | [Manual del programador](docs/manual_programador/README.md), [arquitectura técnica](docs/portal_vec/arquitectura_tecnica.md) y [especificaciones para agentes](ESPECIFICACIONES_AGENTES.md). |
| Consultar el procedimiento de referencia | [Expediente de Contratación temporal remitido por RRHH](docs/portal_vec/expediente_contratacion_temporal_rrhh.md). |

La ayuda contextual del portal se abre desde el botón **«?»** cuando está disponible en la pantalla. Los manuales describen el uso y los límites de cada recorrido; no sustituyen el permiso del actor ni la comprobación de un entorno concreto.

## Despliegue y límites

El repositorio separa la portada pública, el portal interno y las superficies de presentación. La documentación de seguimiento recoge recorridos privados con datos sintéticos y despliegues parciales; **la presencia de código en `main` no demuestra que ese mismo commit esté activo en un servidor**. Consulte el [estado](ESTADO_PROYECTO.md) y el [manual de Sistemas](docs/manual_sistemas/README.md) antes de preparar o comprobar un entorno. No se publican aquí credenciales, certificados, datos reales ni rutas privadas.

VEC no se presenta como producto legalmente operativo. La firma oficial, la entrega externa, los plazos y las conexiones institucionales solo podrán afirmarse cuando el circuito correspondiente esté aprobado, conectado y comprobado. Los datos y documentos de demostración no adquieren validez administrativa por aparecer en una pantalla o generar un recibo.

## Licencia y autoría

Este software es obra de Alberto Avidad (avidad@dipgra.es), desarrollado para la Diputación Provincial de Granada, y se publica bajo la [EUPL-1.2](LICENSE) para que cualquier administración pública u organización pueda reutilizarlo, adaptarlo y redistribuirlo.

La reutilización debe conservar los avisos de autoría y licencia, indicar los cambios y distribuir las obras derivadas bajo la EUPL o una licencia compatible de su apéndice. El archivo [LICENSE](LICENSE) contiene las versiones oficiales en español e inglés.

**Descripción corta sugerida para GitHub:** Portal modular de RRHH de la Diputación de Granada: Contratación temporal y Bolsa con arquitectura hexagonal, permisos explícitos e historia auditable. En desarrollo con datos sintéticos.
