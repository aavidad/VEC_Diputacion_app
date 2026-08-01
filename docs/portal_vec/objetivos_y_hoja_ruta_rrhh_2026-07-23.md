# Objetivos y hoja de ruta del frente RRHH

Última actualización: 1 de agosto de 2026.

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

Estado: O4-01 a O4-04 están cerrados. O4-05 conserva tres de cinco hitos:
contratos HTTP/web, proyecciones protegidas, autorización, registro durable,
cursores, prueba durable, motor privado CT-000044 y fachadas nominales
CT-000045, junto con el adaptador CT-000046, están integrados y revisados.
CT-000047A cierra los manejadores HTTP de cuadro y detalle; CT-000047B cierra
las piezas nominales y el adaptador PostgreSQL de motivos; C1 cierra la
cápsula de identidad; C2.1a el rol selector mínimo; C2.1b la fachada mínima de
Identidad; S0.1/S0.2 las retiradas ContextoActor; C2.2-A la historia y el
puntero organizativos; y C2.2-B la historia y el puntero del vínculo
corporativo. B queda cerrada técnicamente en `de6e7df`, con revisión B5 en
`8c27c72`, GO y P0=P1=P2=0. B6 está publicada en `51a4390`, con CI
`30650778125` completamente verde. C2.3-D0 queda integrado en `5fbf6a7`–
`abfdc21` con doble GO y sin aumentar métricas. La decisión F0-D1 queda
integrada en `c4fc55c`–`1236c0b`; D2/D2a/D2b, que cierran su paquete,
concurrencia, retirada y grafo implementable, quedan en `a5ba276`–`cebc8bd`.
Ambas obtuvieron doble GO. D2c queda integrada en `f57088c` y acreditada en
`7014d1d`; D2d queda publicada en `4b3d01d`–`55f6001`; V0 queda integrado en
`e68dbe0`, revisado y con CI verde. H0 queda integrado en `a0d63df` con doble
GO independiente y PostgreSQL 18.4 real. H0a corrige en `eb21fdd` la
autoprueba sintética que el primer A1 demostró incompatible con una clausura
real y obtuvo doble GO, `P0=P1=P2=0`. A1 queda integrado después en `169a055`,
con doble GO final y PostgreSQL 18.4; dejó listos A2, A3, A4 y B1.
A2 queda integrado en `0ac8fe4` tras
[doble GO](revisiones/revision_f0_a2_canon_manifiesto_2026-08-01.md) y
comparación byte a byte con V0. A3 queda integrado en `806691b` tras
[doble GO](revisiones/revision_f0_a3_canon_capacidad_2026-08-01.md), HMAC y
comparación constante acreditados. A4 queda integrado en `4dd2ff9` con
[doble GO](revisiones/revision_f0_a4_canon_consumo_2026-08-01.md) y B1 en
`00ff427` con
[doble GO](revisiones/revision_f0_b1_catalogo_checkpoint_2026-08-01.md).
B2 queda integrado en `7519063`, con CI `30719044843` completamente verde y
[doble GO](revisiones/revision_f0_b2_atestacion_consumo_2026-08-01.md). C1
corrige un `NO-GO` independiente de cronología y cobertura. Después siguen los demás
componentes hasta Q3, I0, R0/M5–M7, C2.4 selección y
recibo, C2.5 fachada y reconciliación, PDP, composición raíz, TLS/mTLS viva,
la misma web definitiva y el E2E.

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

1. Cerrar O4-05 sin abrir atajos de demostración en la ruta real.
2. Retomar O2-06 y completar la primera vertical de alta con el mismo patrón.
3. Cerrar O3-05 y continuar O5 y O6 reutilizando documentos, firma y Bolsa.
4. Cerrar O7 y O8 con los conectores institucionales autorizados.
5. Ejecutar aceptación funcional, seguridad, accesibilidad, recuperación y
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

Completar, revisar e integrar C1 de C2.3-F0, limitado a su par de componentes.
B2 ya crea las historias minimizadas de atestación y consumo; C1 debe cerrar
el segundo cruce temporal y su matriz adversarial. Después se continúa con C2,
que compone A4+B2+C1, y el resto del DAG
hasta el consumidor nominal de la
capacidad breve de fuente, ejecutable dentro de la misma transacción
`SERIALIZABLE READ WRITE` que el
efecto y con rollback total. A continuación se
implementan R0 y las tres migraciones ContextoActor M5–M7 ya fijadas por D0,
sin seleccionar candidatos ni asumir funciones del PDP. C2.2-A, C2.2-B y el
contrato C2.3-D0, junto con F0-D1 y F0-D2, son dependencias cerradas, no
trabajo pendiente.
