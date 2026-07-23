# Mapa de objetivos, tareas y paralelización

Última actualización: 23 de julio de 2026.

Vista de dirección para revisar el avance del procedimiento de contratación
temporal. El detalle verificable de cada tarea está en el
[tablero de tareas](tablero_tareas_contratacion_temporal_2026-07-23.md).

## Lectura rápida

| Indicador | Estado actual |
| --- | --- |
| Objetivo activo | O2 — primera vertical real de alta |
| Camino crítico | O2-04 autorización VEC → O2-05 confirmación SQL → O2-06 adaptador → O2-07 composición → O2-08 API → O2-09 web → O2-10 E2E |
| Primer hito funcional | 2 de 7 puertas publicadas (29 %); tercera validada localmente |
| Último commit publicado | `40105d2` — caso de uso seguro |
| Trabajo local en revisión | O1-04, O2-02 y O2-03 |
| Bloqueo externo actual | Ninguno para el siguiente corte |
| Producción | No autorizada; no se usarán datos reales |

El porcentaje solo mide las siete puertas de la primera vertical. No es el
porcentaje de toda la aplicación ni del procedimiento completo.

## Mapa de objetivos

```mermaid
flowchart LR
    O1["✅ O1<br/>Base del módulo"]
    O2["🚧 O2<br/>Alta de solicitud"]
    O3["⬜ O3<br/>Análisis RRHH y RC"]
    O4["⬜ O4<br/>Vía de cobertura"]
    O5["— O5<br/>Asignación, informe y fiscalización"]
    O6["— O6<br/>Llamamiento y formalización"]
    O7["— O7<br/>Incorporación, Personal y GINPIX"]
    O8["— O8<br/>Seguimiento, cierre y conservación"]

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
    O202["🚧 O2-02<br/>Adaptador Go de preparación"]
    O203["🚧 O2-03<br/>Migración de preparación"]
    O204["⬜ O2-04<br/>Autorización VEC durable"]
    O205["— O2-05<br/>Confirmación SQL atómica"]
    O206["— O2-06<br/>Adaptador y reconciliación"]
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

O2-02 y O2-03 son trabajos paralelizables porque uno prueba el contrato Go y
el otro el contrato SQL. Solo se integran después de superar sus revisiones
independientes.

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

Ejemplo actual:

```text
O2: 2/7 puertas publicadas = 29 %
Tareas cerradas publicadas: O1-01, O1-02, O1-03, O2-01
Tareas locales en revisión: O1-04, O2-02, O2-03
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
