# VEC · Diputación de Granada

VEC es la aplicación de Recursos Humanos de la Diputación de Granada. Reúne la
gestión de bolsas de trabajo y la contratación temporal, el área del aspirante
y las bases comunes de personas, permisos, documentos, firma y auditoría.
Está en desarrollo y usa siempre datos ficticios.

## Estado al 20 de octubre de 2026

La prioridad es cerrar la **fase 1: Bolsa y Contratación temporal**. El caso de
aceptación está en [FLUJO_APP, apartado 4.0](docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación):
petición del centro, ratificación, tramitación de RRHH, fiscalización,
llamamiento desde Bolsa, resolución, incorporación, alta en Personal, cese y
vuelta a la bolsa. El área del aspirante se recorre también por separado.

Hay avances integrados en estos pasos, pero el recorrido completo con sus
variantes y la recuperación tras reiniciar aún debe acreditarse. Los criterios
de cierre incluyen permisos y auditoría, operaciones visibles que funcionen y
tiempos de respuesta medidos; están en el [plan de implementación](docs/producto/PLAN_IMPLEMENTACION.md#5-fases-y-criterios-de-cierre).

El trabajo y los ensayos se hacen **en copias locales**. Cidonia está congelada;
la siguiente instalación irá a un servidor nuevo. Lo fusionado en `main` no
acredita una instalación ni un recorrido completo. El
[estado del proyecto](ESTADO_PROYECTO.md) y el
[tablero de tareas](docs/producto/TABLERO_TAREAS.md) recogen avances y pendientes.

## Módulos

| Área | Alcance actual |
| --- | --- |
| Bolsa | Convocatorias, inscripción y admisión, candidaturas, preferencias, llamamientos y consulta en Mi Bolsa. El alcance depende de la configuración y los permisos de la copia. |
| Contratación temporal | Peticiones del centro y expediente de RRHH, documentos PDF y Word, fiscalización, firma y actuaciones de incorporación y cese. El caso completo sigue pendiente de aceptación. |
| Personal y Organización | Bases de puestos, categorías y relaciones de servicio que necesita la contratación. El resto de su alcance queda para la fase siguiente. |
| Administración y servicios comunes | Identidad por certificado, gobierno de perfiles, catálogos, documentos y auditoría, con activación específica en la copia. |
| Cronos, Dietas y demás módulos | Conservan código y planes de distinto alcance. Su desarrollo se retoma según el orden del plan, después de Bolsa y Contratación. |

La arquitectura es hexagonal. Cada módulo conserva sus datos y publica
contratos para los demás; identidad, autorización, auditoría e idiomas son
comunes. El detalle está en el [documento técnico](docs/producto/TRD.md).

## Arrancar una copia local

Sigue [Crear y usar una copia local](deploy/principal/LEEME_copia_local.md).
El guion `deploy/principal/copia_local.py` prepara y gestiona una instancia
propia con PostgreSQL y la aplicación. La receta indica requisitos, material
privado autorizado, módulos opcionales, comprobaciones, reinicio y borrado.
No basta con compilar para disponer del recorrido: firma, Administración y
Personal requieren la preparación descrita allí.

Para usar la copia desde el navegador, sigue la
[guía del recorrido](GUIA_RECORRIDO_ALBERTO.md). Cada persona entra con su
certificado y perfil. Los recibos locales no acreditan una confirmación de
GINPIX, del correo corporativo ni de otros sistemas externos.

## Documentación

| Para | Documento |
| --- | --- |
| Conocer el producto y su alcance | [Requisitos de producto](docs/producto/PRD.md) |
| Ver qué queda y en qué orden | [Plan](docs/producto/PLAN_IMPLEMENTACION.md) y [tablero](docs/producto/TABLERO_TAREAS.md) |
| Seguir los perfiles y el caso de aceptación | [Flujo de la aplicación](docs/producto/FLUJO_APP.md) |
| Usar las pantallas | [Manuales por perfil](docs/manuales/) y [guía del recorrido](GUIA_RECORRIDO_ALBERTO.md) |
| Preparar y recuperar una instancia | [Copia local](deploy/principal/LEEME_copia_local.md) y [manual de Sistemas](docs/manual_sistemas/README.md) |
| Desarrollar o integrar | [Manual del programador](docs/manual_programador/README.md), [diseño de interfaz](docs/producto/DISENO_UI_UX.md) y [esquema del backend](docs/producto/ESQUEMA_BACKEND.md) |
| Consultar decisiones pendientes de RRHH | [Preguntas abiertas](dudas.md) |

La ayuda de cada pantalla se abre con el botón «?». El uso con datos reales
requiere las autorizaciones y condiciones de la fase de paso a producción;
ningún ensayo local las sustituye.

## Licencia y autoría

Este software es obra de Alberto Avidad, desarrollado para la Diputación
Provincial de Granada. Se publica bajo la [EUPL-1.2](LICENSE), que permite su
reutilización, adaptación y redistribución con las condiciones de esa licencia.
