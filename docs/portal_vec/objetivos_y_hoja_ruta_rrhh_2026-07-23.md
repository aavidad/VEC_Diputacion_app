# Objetivos y hoja de ruta del frente RRHH

Última actualización: 23 de julio de 2026.

Este documento es la referencia de dirección del procedimiento recibido de
RRHH. Evita dirigir el trabajo por códigos internos y permite saber qué se ha
cerrado, qué falta y qué evidencia exige cada cierre.

La descomposición de cada objetivo en commits verificables se mantiene en el
[tablero de tareas](tablero_tareas_contratacion_temporal_2026-07-23.md).
Las dependencias y los carriles que pueden ejecutarse a la vez se visualizan
en el
[mapa de objetivos y paralelización](mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md).
Las puertas europeas, estatales y andaluzas están trazadas en la
[matriz normativa del módulo](matriz_normativa_contratacion_temporal_2026-07-23.md).

## Objetivo de producto

Incorporar al portal VEC un expediente completo de contratación temporal desde
la petición del centro hasta la incorporación, GINPIX, seguimiento y cierre,
sin sustituir ni duplicar Bolsa, Personal, documentos, firma, comunicaciones o
auditoría.

El resultado debe ser:

- modular y hexagonal;
- gobernable desde la aplicación mediante catálogos y flujos versionados;
- seguro por denegación predeterminada;
- trazable y recuperable;
- intercambiable en identidad, base de datos, documentos, firma,
  comunicaciones y sistemas externos;
- utilizable desde web y, mediante la misma aplicación, desde futuras
  superficies API, CLI o MCP autorizadas.

## Relación con Bolsa

El trabajo es complementario. No se borra ninguna capacidad de Bolsa.

| Autoridad | Conserva |
| --- | --- |
| Bolsa | Convocatorias, integrantes, posiciones, disponibilidad, reglas y llamamientos. |
| Contratación temporal | Expediente administrativo, fases, tareas, decisiones y coordinación. |
| Personal | Relación jurídica, ocupación, puesto, incorporación y cese. |
| Documentos/firma | Generación, custodia, firma, CSV/QR y cotejo. |
| VEC común | Identidad, autorización, auditoría transversal, catálogos, eventos e i18n. |

Solo se retirarán componentes de demostración duplicados cuando la capacidad
real alcance paridad, exista migración verificable y no se pierda historia.

## Objetivos verificables

### O1. Base del módulo

Condición de cierre:

- especificación normalizada desde el documento de RRHH;
- módulo y manifiesto sin concesión implícita de permisos;
- agregado versionado, cronología de solo adición y copias defensivas;
- fases y opciones funcionales gobernadas, no listas compiladas.

Estado: **cerrado y probado**.

### O2. Primera vertical real: alta de la solicitud

Condición de cierre completa:

```text
pantalla interna
→ API
→ identidad de garantía alta
→ autorización VEC
→ flujo gobernado
→ idempotencia
→ PostgreSQL
→ auditoría y outbox
→ recibo
```

| Puerta | Estado | Evidencia |
| --- | --- | --- |
| Dominio y validaciones | ✅ | Pruebas unitarias y de carrera. |
| Caso de uso y puertos | ✅ | Reintento, denegación, adulteración y cancelación. |
| Reserva idempotente PostgreSQL | ✅ | Rotación HMAC v1→v2, replay, concurrencia, ACL, reintentos y límites reales; PostgreSQL efímero 3/3. |
| Autorización VEC durable + confirmación | 🚧 | O2-04 y la función SQL O2-05 están cerradas; falta el adaptador/reconciliación O2-06 para completar esta puerta vertical. |
| API interna | 🚧 | Adaptador O2-08B revisado con GO e integrado; sin autoridad reconstruida desde HTTP ni cookies. Falta registrar la ruta mediante O2-07. |
| Pantalla definitiva | ⬜ | Misma web final; adaptador real registrado por composición. |
| E2E y aceptación | ⬜ | Reintento, concurrencia, reinicio, fallo y prueba de RRHH. |

Progreso del hito: **3 de 7 puertas (43 %)**. No representa un porcentaje de
toda la aplicación.

### O3. Análisis RRHH y RC

RRHH valida categoría, causa, periodo, jornada, RC y coste usando fuentes
autoritativas. Cada rectificación conserva autor, motivo, versión y recibo.

Estado: dominio, aplicación, fuentes atestadas y confirmación PostgreSQL
atómica cerrados hasta O3-04, incluido recorrido Go → PostgreSQL, resultados
ambiguos, reinicio y revisión independiente. La interfaz O3-05 continúa
pendiente.

### O4. Decisión de vía de cobertura

Comprobar Bolsa, agotamiento, candidaturas, SAE y nueva convocatoria; justificar
la vía elegida y conservar las fuentes utilizadas.

Estado: catálogo O4-01 y consultas atestadas O4-02 cerrados. O4-03 está activo
mediante microcortes de política/propuesta, evidencia completa y decisión
humana autorizada; persistencia, interfaz y E2E corresponden a O4-04 y O4-05.

### O5. Asignación, informes y fiscalización

Asignar unidad y responsable, elaborar informe jurídico, tramitar fiscalización
y resolver incidencias sin saltos de fase no autorizados.

Estado: asignación modelada inicialmente; resto pendiente.

### O6. Llamamiento y formalización

Reutilizar Bolsa para seleccionar y llamar; gestionar aceptación, renuncia,
documentación, propuesta, firmas múltiples y documentos descargables.

Estado: Bolsa aporta contratos reutilizables; integración pendiente.

### O7. Incorporación, Personal y GINPIX

Confirmar relación/ocupación en Personal y entregar a GINPIX mediante un puerto
con adaptadores API o fichero, versión explícita de mapeo y reintento
recuperable.

Estado: especificado; pendiente.

### O8. Seguimiento, cierre y conservación

Gestionar vigencia, prórroga, cese, archivo, conservación, expurgo y acceso al
histórico conforme a política aprobada.

Estado: especificado; pendiente.

## Orden de ataque

1. Terminar O2 sin abrir atajos de demostración en la ruta real.
2. Validar O2 con RRHH y Sistemas.
3. Replicar el mismo patrón durable en O3 y O4.
4. Cerrar O5 y O6 reutilizando documentos, firma y Bolsa.
5. Cerrar O7 y O8 con los conectores institucionales autorizados.
6. Ejecutar aceptación funcional, seguridad, accesibilidad, recuperación y
   operación antes de declarar producción.

## Definición común de terminado

Una capacidad solo está terminada si dispone de:

1. dominio e invariantes;
2. caso de uso;
3. puertos;
4. adaptador durable;
5. autorización por operación;
6. auditoría, idempotencia y outbox cuando produce efectos;
7. API;
8. interfaz conectada;
9. pruebas unitarias, concurrencia, fallo y E2E;
10. documentación técnica, funcional y de operación;
11. aceptación del área competente.

Un contrato, una tabla, una pantalla o una prueba aislada no bastan.

## Próxima puerta exacta

Integrar la autorización durable de VEC en la orden de alta y añadir la función
PostgreSQL que, dentro de la misma transacción:

- coteje y consuma la concesión exacta;
- confirme la reserva;
- inserte expediente y actuación inicial;
- escriba auditoría;
- publique el evento outbox;
- devuelva un recibo verificable.

Hasta que esa puerta cierre, la cuenta runtime solo puede preparar referencias;
no puede confirmar expedientes.
