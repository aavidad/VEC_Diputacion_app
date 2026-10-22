# Certificados y analítica: inventario y continuación

## Vigencia al 20 de octubre de 2026

Este módulo conserva su alcance y sigue después de Bolsa y Contratación
según el [plan general](../producto/PLAN_IMPLEMENTACION.md). El estado de cada
tarea lo fija el [tablero](../producto/TABLERO_TAREAS.md). Se trabaja en
copias locales con datos ficticios; cidonia está congelada.

El inventario de abajo conserva sus fechas, contratos y requisitos. Sus
ramas, responsables, órdenes de reanudación, horas y cifras pertenecen a esos
cortes: no son un encargo actual ni acreditan una instalación o un recorrido.
Antes de reanudar, dirección debe contrastar qué se ha integrado en `main`
y fijar una tarea con consumidor visible y comprobación propia. Lo pendiente
de RRHH sigue en [dudas.md](../../dudas.md), con valor provisional configurable
cuando corresponda; ninguna prueba con datos ficticios autoriza datos reales.

## Inventario histórico anterior al 20 de octubre de 2026

**Aparcado hasta cerrar Bolsa y CT (según la prioridad del plan).** Los objetivos vigentes están en [plan de implementación](../producto/PLAN_IMPLEMENTACION.md).

Estado comprobado sobre `origin/main` `0a62a3ea6` el 1 de octubre de 2026. El catálogo funcional es `docs/producto/PRD.md`; este plan no da por emitido ningún certificado ni por registrada una descarga.

## Qué existe

| Pieza en main | Rutas y estado |
| --- | --- |
| Certificados | `internal/modules/personal/manifest.go` declara hoy un enlace al menú; **Certificados conserva la autoridad del módulo y de la entrada de Certificados**, Personal solo enlaza. `web/static/area-personal/` presenta certificados, pero su catálogo propio llega vacío. No hay emisión de servicios conectada. |
| Documentos y firma | `internal/vec/adapters/documentos/pdf/renderizador.go` genera PDF; `internal/modules/contrataciontemporal/application/firma_documento.go` y `ports/custodia_firmado.go` aportan contratos de firma y custodia para CT. Son capacidades reutilizables mediante puertos; no acreditan por sí solas un certificado firmado. |
| Estadísticas CT | `internal/modules/contrataciontemporal/adapters/postgres/estadisticas_rrhh_postgresql.go`, migración CT 000107, `adapters/httpinterno/estadisticas_rrhh.go` y `web/static/portal-empleado/modulos/contratacion-temporal/{cliente-http-estadisticas,contrato-estadisticas,vista-estadisticas}.js` dan consulta agregada por semana, mes o año y ámbito autorizado, con gráfico, tabla y CSV local. Funciona como cuadro CT; no es un cuadro transversal. |

## Huecos frente al catálogo y minitareas

| Orden | Capacidad y tarea pequeña | Responsable, archivos previstos y dependencia | Cierre verificable |
| --- | --- | --- | --- |
| 1 | **ANA-001:** mostrar definición y fórmula observable de SQL107, fuente, periodo, calidad y versión de los cinco indicadores CT existentes. El responsable institucional y la aprobación siguen pendientes de RRHH. | `web/static/portal-empleado/modulos/analitica/`, `vista-estadisticas.js`, textos ES/EN. Terminar la cadena `?v=` y los manifiestos con sus dependencias de carga. | Ayuda visible junto al cuadro real, sin cambiar cifras; Chrome escritorio/móvil, revisión independiente, calidad y CI verdes. |
| 2 | **CER-001, preparación:** plantilla de servicios versionada y borrador PDF con ejemplo sintético, sin emisión oficial. La CLI es el preparador que servirá como **vista previa de RRHH en el producto**; el corte actual solo admite ensayo sintético local y aún no está montado en el portal. | `internal/modules/certificados/`, `cmd/vec-certificados-borrador/`, `data/certificados/plantillas/`, textos ES/EN. Reutiliza PDF común; entrada autorizada de servicios depende de Personal. | La CLI prepara y comprueba un borrador identificado como tal; exige una muestra marcada como sintética, sin poder verificar que un nombre sea ficticio. No afirma firma, código de verificación ni entrega. |
| 2b | **CER-002 (hecho):** el borrador acepta la respuesta del contrato V1 de Personal (`LectorServiciosParaCertificadosV1`): periodos semiabiertos o abiertos, cobertura, certeza y acto, sin días porque V1 no los trae; marca qué servicios pueden sustentar un certificado. | `internal/modules/certificados/adapters/personalv1/`, dominio, aplicación, CLI y textos ES/EN. El lector de Personal sigue sin implementación. | El CLI prepara el borrador desde una muestra sintética con forma V1 y la traduce con el mismo adaptador que usará la respuesta real. |
| 3 | **ANA-002:** hacer visible en el cuadro CT el rango efectivo, zona horaria y corte recibidos. | Integración de `vista-estadisticas.js` y catálogos CT; sin nueva pantalla ni SQL. El servidor ya determina el ámbito y no devuelve su etiqueta. | Una consulta autorizada muestra contexto y totales coherentes; carga, error, denegación y móvil comprobados. |
| 4 | **CER-001, emisión:** obtener hechos de servicios autorizados de Personal por un puerto, revisión RRHH, documento original, firma/sello, registro, código seguro de verificación (CSV), entrega y conservación. | Personal posee servicios; Certificados posee plantilla/expediente; Documentos/firma aporta capacidades de PDF, firma y custodia por puertos nuevos o adaptados. Requiere permisos fijos, SQL nuevo **solo en borrador**, número reservado y orden en `ORDEN_SQL_NUCLEO.md` y ensayar en clon y doble revisión antes de integrar. La verificación por código no abre la ficha Personal; RRHH debe aprobar sus datos visibles. | Navegador → identidad/permiso → datos de oficio → revisión → firma real → documento y código verificables → recibo e historia conservados tras reinicio. |
| 5 | **ANA-005:** registrar consulta y exportación nominales antes de entregar resultados. | Propietario CT Contratación temporal, con Certificados para el consumidor y Autorización para orden SQL; puerto de consulta atestada y exportación separada con `corte_esperado`. SQL 000107 solo lee el corte actual; el archivo CSV de hoy se genera en el navegador y carece de registro de descarga. SQL nuevo queda en borrador hasta reserva, ensayo y revisiones. Alertas de uso anómalo quedan pendientes de regla aprobada. | Revalidar identidad, permiso y corte; confirmar antes de devolver datos un registro durable mínimo de actor, finalidad, ámbito, versión, corte, campos, filas y resultado, sin contenidos de certificados ni filas nominales. Cambio de corte rechaza exportar lo visto antes; la traza acredita la puesta a disposición o el intento de respuesta, no recepción ni guardado. |

**Fuera del corte actual:** ANA-003 (transparencia) y ANA-004 (informes regulatorios/ISPA). Requieren proyección, plantillas, autoridad y decisiones propias; no están incluidas en las horas siguientes ni se deducen del cuadro CT.

## Decisiones pendientes de RRHH

Las preguntas se contrastan con `dudas.md` antes de añadirlas, conservando su numeración:

1. ¿Qué certificados de servicios y plantillas aprueba RRHH, quién revisa y firma/sella cada tipo, y qué datos de oficio son obligatorios?
2. ¿Qué fuentes acreditan periodos, jornada e interrupciones y cómo se distinguen los servicios declarados, comprobados y reconocidos?
3. ¿Quién responde de cada indicador, aprueba su fórmula y fija calidad, periodicidad, umbrales de revelación y ámbitos del cuadro?
4. ¿Qué finalidades y perfiles permiten consultar y descargar agregados, cuánto se conserva la traza y quién revisa alertas de uso anómalo?

## Condiciones funcionales conservadas

Personal conserva la autoridad de los servicios; Certificados gobierna su expediente; Documentos aporta generación, firma y custodia por contrato. Los indicadores necesitan fuente, periodo, fórmula aprobada y control de identificación indirecta. Toda consulta o descarga se autoriza y registra antes de responder. Un borrador no es un certificado oficial.
