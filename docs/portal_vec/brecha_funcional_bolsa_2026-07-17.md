# Brecha funcional verificada de Bolsa a 17 de julio de 2026

## Finalidad

Este documento separa las capacidades demostrables del producto de los modelos,
contratos, dobles de prueba y pantallas que todavía no están conectados a una
composición productiva. Su objetivo es impedir que una interfaz visible o un
dominio bien probado se contabilicen como un recorrido terminado.

Los porcentajes son una estimación de alcance funcional, no una métrica de
calidad ni de líneas de código:

- recorridos de Bolsa utilizables sin datos de demostración: **20-25 %**;
- cimentación técnica reutilizable: **50-60 %**;
- recorridos funcionales todavía pendientes: **75-80 %**.

La consulta pública es el único recorrido actualmente conectado de extremo a
extremo. Aun así, la composición de serie publica dos convocatorias sintéticas,
por lo que no se considerará productiva hasta que lea la proyección durable de
las convocatorias aprobadas por RRHH.

## Evidencia por capacidad

| Prioridad | Capacidad | Evidencia disponible | Brecha para considerarla terminada |
| --- | --- | --- | --- |
| P0 | Composición productiva | El servidor monta la carcasa VEC y la consulta pública. La API heredada de candidatos solo se activa en modo `fake` en `internal/app/bootstrap/bootstrap.go`. | Montar exclusivamente casos de uso reales, con PostgreSQL y conectores productivos; ninguna ruta administrativa puede depender de semillas locales. |
| P0 | Gobierno de convocatorias | El dominio representa borrador, publicación, sustitución y retirada; existen contratos exactos en `internal/modules/bolsa/ports/convocatorias_gobierno_*`. | Servicio de aplicación, repositorio transaccional, autorización, idempotencia, auditoría, outbox y API RRHH. |
| P0 | Bases y reglas de baremo | `domain/reglasbaremo` modela versiones, topes, jornada, restos, redondeos y concurrencia; `domain/calculoexperiencia` está construyendo el motor exacto. | Persistencia y gobierno productivos, simulador y editor RRHH; ampliar después experiencia con titulaciones, cursos, exámenes y los restantes méritos tipados. |
| P0 | Portal privado y candidatura | Hay un núcleo heredado en `internal/candidate` y modelos reutilizables. | Alta, borrador, firma, registro, subsanación, desistimiento y expediente conectados sin reglas fijas ni datos sintéticos. |
| P0 | Documentación segura | Existen contratos de almacenamiento S3 compatible, cuarentena, análisis y custodia. | Reserva de subida directa, confirmación, análisis antimalware, firma, registro y recuperación conectados al flujo de candidatura. La ruta heredada devuelve `503` deliberadamente mientras falten estas garantías. |
| P0 | Autobaremación y revisión | El dominio admite aceptación, desestimación, rectificación, revocación, rehabilitación y decisiones firmadas. | Fuente de méritos, cálculo oficial, firma, sello de tiempo, custodia, autorización y repositorio productivos; API y bandeja RRHH. |
| P0 | Llamamientos | Existe selección segura del primer candidato elegible y una transacción PostgreSQL de propuesta. | Fuentes autoritativas y composición; confirmación, comunicación, respuesta, renuncia, no comparecencia, contratación y reincorporación. |
| P0 | Alegaciones | El legado conserva un modelo inicial. | Presentación registrada, documentación cotejada, revisión, resolución firmada, notificación y recurso. La ruta heredada permanece cerrada con `503`. |
| P0 | Trazabilidad integral | Los dominios aplican inmutabilidad, huellas, idempotencia y denegación por defecto. | Auditoría durable en todos los recorridos y pruebas integrales HTTP, navegador y PostgreSQL, incluida recuperación tras reinicio. |
| P1 | Operación profesional | Hay contratos y diseños parciales. | Notificación multicanal, operaciones masivas, exportaciones, reconciliación, conservación y expurgo, accesibilidad real y firma longeva o múltiple. |
| P2 | Ampliaciones | La arquitectura reserva adaptadores intercambiables. | CLI, MCP e IA, analítica, formatos y canales adicionales, nube y nuevos motores de persistencia. |

## Estado de las pantallas

La carcasa administrativa, la navegación lateral y la densidad de información
son reutilizables. No acreditan por sí mismas una capacidad de negocio:

- el panel RRHH solicita `/api/vec/bolsa/panel` y
  `/api/vec/bolsa/propuestas-llamamiento`, pero esas rutas aún no están
  registradas en la composición productiva;
- varios botones muestran diálogos informativos y no ejecutan un caso de uso;
- guardar una convocatoria y confirmar un llamamiento siguen siendo acciones
  pendientes;
- la consulta pública usa `GET` y `HEAD` reales, pero su fuente de serie es de
  demostración y declara que no tramita solicitudes.

Todo valor sintético o acción no conectada debe seguir etiquetado como
demostración. Su retirada forma parte del criterio de aceptación, no de una
limpieza opcional posterior.

## Dependencias que no se pueden invertir

El camino crítico del producto es:

1. identidad y autorización central con denegación por defecto;
2. PostgreSQL, auditoría durable, outbox y custodia documental;
3. convocatoria y reglas aprobadas y versionadas;
4. candidatura, documentos y registro;
5. autobaremación, cálculo oficial y revisión firmada;
6. listas provisionales, alegaciones y lista definitiva;
7. bolsa constituida y llamamientos.

La auditoría, el control de acceso por finalidad y la evidencia transaccional
atraviesan todos los pasos. No son una fase final.

## Siguiente corte vertical definitivo

El siguiente entregable demostrable debe reutilizar código de producción y
recorrer la misma información desde RRHH hasta el portal público:

1. crear y editar una convocatoria desde el área interna;
2. configurar reglas reales de experiencia;
3. ejecutar simulaciones contra el motor exacto;
4. adjuntar y custodiar las bases;
5. aprobar y publicar con autorización, idempotencia, PostgreSQL, auditoría y
   outbox en una única transición coherente;
6. proyectar esa misma versión en la consulta pública, sin JSON de ejemplo;
7. probar denegación por defecto, publicación concurrente y conservación tras
   reinicio completo.

Este corte no completa todavía candidaturas ni llamamientos, pero elimina una
cadena de demostración completa y crea la base productiva que ambos recorridos
necesitan.

## Criterio de comunicación del avance

En informes y demostraciones se distinguirán siempre cuatro estados:

- **modelo**: reglas e invariantes representadas en el dominio;
- **probado en aislamiento**: dominio o adaptador con pruebas, sin composición;
- **integrado**: recorrido conectado con dependencias reales;
- **productivo verificado**: integrado, durable, autorizado, auditable y
  probado tras reinicio y bajo concurrencia.

Solo el último estado se expresará como funcionalidad terminada.
