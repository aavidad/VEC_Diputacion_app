# Mapa de objetivos, tareas y paralelización

Última actualización: 24 de julio de 2026.

Vista de dirección para revisar el avance del procedimiento de contratación
temporal. El detalle verificable de cada tarea está en el
[tablero de tareas](tablero_tareas_contratacion_temporal_2026-07-23.md).

## Lectura rápida

| Indicador | Estado actual |
| --- | --- |
| Objetivo activo | O2 — primera vertical real de alta |
| Camino crítico | O2-05 confirmación SQL → O2-06 adaptador → O2-07 composición → O2-08 API → O2-09 web → O2-10 E2E |
| Primera vertical | 5 de 10 tareas cerradas (50 %); O2-06 es el siguiente cierre |
| Procedimiento completo | 16 de 46 tareas cerradas (35 %); O4-02 ya está integrado |
| Último commit verificado | `6fb6cc6` — matriz web RRHH 1..17 verde en 51/51 capturas locales; no cierra la integración O2-09 |
| Trabajo local en revisión | O2-06 tiene corrección candidata con PostgreSQL 18 verde y espera revisión independiente; primer corte O5-01 pendiente. La superficie RRHH ya superó la revisión visual local, pero sigue separada del E2E real. |
| Bloqueo externo actual | Ninguno para programar; producción sigue sujeta a las conformidades formales |
| Producción | No autorizada; no se usarán datos reales |

La primera cifra mide las diez tareas de O2. La segunda mide las 46 tareas del
procedimiento temporal completo; ninguna representa por sí sola el porcentaje
de todo VEP.

## Mapa de objetivos

```mermaid
flowchart LR
    O1["✅ O1<br/>Base del módulo"]
    O2["🚧 O2<br/>Alta de solicitud"]
    O3["🚧 O3<br/>Análisis RRHH y RC"]
    O4["🚧 O4<br/>Vía de cobertura"]
    O5["— O5<br/>Asignación, informe y fiscalización"]
    O6["🚧 O6<br/>Llamamiento y formalización"]
    O7["— O7<br/>Incorporación, Personal y GINPIX"]
    O8["🚧 O8<br/>Seguimiento, cierre y conservación"]

    O1 --> O2
    O2 --> O3
    O2 --> O4
    O3 --> O5
    O4 --> O5
    O5 --> O6
    O6 --> O7
    O7 --> O8
```

O3 y O4 pueden desarrollarse en paralelo una vez estabilizado el patrón
durable de O2. O5 necesita ambos resultados. El contrato de integración de
Bolsa de O6 y el modelo canónico GINPIX de O7 pueden adelantarse sin activar
sus efectos.

## Mapa del camino crítico O2

```mermaid
flowchart TD
    O201["✅ O2-01<br/>Caso de uso"]
    O202["✅ O2-02<br/>Adaptador Go de preparación"]
    O203["✅ O2-03<br/>PostgreSQL y rotación"]
    O204["✅ O2-04<br/>Autorización VEC durable"]
    O205["✅ O2-05<br/>Confirmación SQL atómica"]
    O206["🚧 O2-06<br/>Adaptador y reconciliación"]
    O207["— O2-07<br/>Composición real"]
    O208["— O2-08<br/>API interna"]
    O209["— O2-09<br/>Web definitiva"]
    O210["— O2-10<br/>E2E y aceptación"]

    O201 --> O202
    O201 --> O203
    O202 --> O204
    O203 --> O205
    O204 --> O205
    O205 --> O206
    O206 --> O207
    O207 --> O208
    O208 --> O209
    O209 --> O210
```

O2-02, O2-03, O2-04 y O2-05 ya superaron revisión independiente. O2-03 demostró en
tres ejecuciones PostgreSQL reales la convivencia HMAC v1→v2, el replay
exacto, la concurrencia, las ACL y los límites de sentencia e inactividad.
O2-04 demostró que esa preparación solo puede ejecutarse tras una concesión
V3 durable, ligada al par HMAC activo exacto y sin autoridad reconstruida
desde el cliente.

## Olas de trabajo paralelo

### Ola 0 — cierre actual

| Carril | Tarea | Archivos principales | Dependencia | Integración |
| --- | --- | --- | --- | --- |
| Dirección | O1-04 | documentación de objetivos, tareas y relevo | O1-03 | Commit documental aislado. |
| Adaptador | O2-02 | `internal/modules/contrataciontemporal/adapters/postgres` y puerto de infraestructura | O2-01 | Unitarias, carrera y vet. |
| Base de datos | O2-03 | `deploy/postgresql/contratacion_temporal` | Contrato O2-01 | PostgreSQL real efímero, ACL y down protegido. |

Estos tres carriles no deben modificar los mismos ficheros salvo README,
estado y relevo, que actualiza únicamente el integrador.

### Ola 1 — tras publicar la ola 0

| Carril | Tarea | Puede avanzar junto a | Condición de entrega |
| --- | --- | --- | --- |
| Seguridad | O2-04 | O3-01, O4-01, O6-01 | Capacidad durable común de VEC; sin segunda autoridad local. |
| Dominio RRHH | O3-01 | O2-04, O4-01, O6-01 | Análisis/RC completo y probado, sin adaptadores. |
| Catálogos | O4-01 | O2-04, O3-01, O6-01 | Vías/comprobaciones versionadas sin recompilar. |
| Integración Bolsa | O6-01 | O2-04, O3-01, O4-01 | Solo contrato y eventos; sin tablas cruzadas. |

Cada carril usa un worktree propio. Seguridad no comparte ficheros con dominio
funcional. El integrador revisa y aplica los commits uno por uno.

### Ola 2 — contrato de autorización estabilizado

| Carril | Tarea | Dependencia |
| --- | --- | --- |
| PostgreSQL de efectos | O2-05 | O2-03 + O2-04 |
| Diseño de interfaz | parte visual de O2-09 | contrato de entrada/salida O2-01; no se simula autoridad |
| Conectores RRHH | O3-03 | O3-01 |
| Consultas de cobertura | O4-02 | O4-01 + O6-01 |
| Modelo GINPIX | O7-03 | especificación RRHH; sin activar envío |

La interfaz puede avanzar como presentación del mismo modelo, pero no se marca
conectada hasta O2-08. Los adaptadores DEMO permanecen en una composición
separada y no entran en la ruta real.

### Ola 3 — primer efecto durable

| Carril | Tarea | Dependencia |
| --- | --- | --- |
| Aplicación/PostgreSQL | O2-06 | O2-05 |
| Análisis RRHH | O3-02 | O3-01 |
| Cobertura | O4-03 | O4-01 + O4-02 |
| Firma documental | O5-02/O5-03, solo contrato inicial | conectores comunes ya existentes |

### Ola 4 — integración de la primera vertical

```text
O2-07 composición
      ↓
O2-08 API ─── revisión de seguridad
      ↓
O2-09 web ─── revisión visual y accesibilidad
      ↓
O2-10 E2E ─── aceptación RRHH
```

La secuencia final es deliberadamente lineal: evita que una pantalla o una API
oculten una composición aún falsa.

## Regla productor–revisor–integrador

Cada tarea admite hasta tres trabajos distintos:

| Función | Puede modificar | No puede hacer |
| --- | --- | --- |
| Productor | Solo el alcance declarado de la tarea y sus pruebas. | Integrar su propio trabajo en la rama de convergencia. |
| Revisor | Pruebas adicionales, informe y correcciones en commit separado si se autorizan. | Rebajar requisitos o declarar aceptación funcional. |
| Integrador | Documentación transversal, resolución de conflictos y commit/merge final. | Omitir una puerta fallida por ganar tiempo. |

El revisor debe ser distinto del productor para seguridad, PostgreSQL,
autorización, documentos/firma y fronteras HTTP.

## Ficha mínima visible por tarea

Cada fila del tablero se actualizará con:

```text
ID:
objetivo:
estado:
responsable/productor:
revisor:
dependencias:
resultado:
pruebas:
commit:
incidencias:
siguiente tarea desbloqueada:
```

El campo responsable identifica el trabajo, no concede un rol de aplicación.

## Cómo calcular el avance

Se publican dos métricas:

1. **Puertas funcionales:** porcentaje principal. Solo avanza cuando existe una
   pieza completa del recorrido.
2. **Tareas verificadas:** indicador operativo. Sirve para saber carga y
   paralelización, pero no se presenta como porcentaje de producto.

Estado actual:

```text
O2: 3/7 puertas funcionales publicadas = 43 %
Tareas verificadas del procedimiento: 16/46 = 35 %
Tareas locales en revisión: corrección candidata O2-06 y O5-01
Web RRHH: revisión visual local superada; O2-09 sigue abierta por O2-07/O2-10
```

Una tarea local no cuenta como cerrada hasta que el commit esté publicado y el
tablero contenga su evidencia.

## Actualización obligatoria

Al cerrar cualquier tarea, el integrador actualiza en el mismo corte:

1. estado de la tarea;
2. hash del commit;
3. prueba ejecutada;
4. tareas desbloqueadas;
5. indicador de puertas si realmente cambia;
6. relevo para el siguiente agente.
