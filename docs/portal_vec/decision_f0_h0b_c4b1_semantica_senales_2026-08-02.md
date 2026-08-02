# Decisión F0-H0b C4b-1: semántica de INT y TERM

Fecha: 2 de agosto de 2026.

Estado: contrato documental pendiente de contrarrevisión independiente. No
cierra C4b-1, H0b, C2, F0 ni producción.

## Hecho técnico y evidencia

`signal(7)` establece dos límites relevantes para las señales estándar:
varias instancias pendientes de una misma señal no se encolan y el orden de
entrega de señales pendientes no está especificado. Por tanto, el orden de dos
órdenes `kill` no acredita el orden en que Bash iniciará sus manejadores.

Las pruebas reales ejecutadas sobre `1524feb` confirman esa diferencia: el
orden de `kill` no equivale necesariamente al orden de los `trap`. La
implementación Bash puede enclavar la primera señal que observa al comenzar un
manejador, pero no puede reconstruir de forma fiable cuál fue enviada primero
antes de que exista ese marco de ejecución.

La serie local conserva estos `NO-GO` y no debe integrarse por partes:

| Candidato | Resultado que no acredita |
| --- | --- |
| `db240a5` | Difiere señales, pero no fija una causal estable frente a señales sucesivas. |
| `075610f` | Enclava una causal ya escrita, pero una reentrada antes de la primera orden del manejador puede sustituirla. |
| `1524feb` | Impide esa reentrada cuando ya existe el marco del manejador; no puede prometer que el primer `kill` sea la primera señal entregada. |

## Contrato garantizable

1. Una única `INT` conserva estado 130 y una única `TERM` conserva estado 143.
2. La primera señal entregada y observada al iniciarse un manejador queda
   enclavada. Una reentrada o señal posterior no la sustituye ni abre otra
   cancelación.
3. En una ráfaga anterior al marco del manejador, el resultado admisible es
   exactamente uno de `{130, 143}`. No se atribuye prioridad al orden de
   envío.
4. Toda ráfaga produce exactamente una cancelación, cero efectos o trabajos
   nuevos y una limpieza convergente. Las señales posteriores solo pueden
   despertar una espera ya existente; no cambian la causal enclavada.
5. La acción de cada `trap` es una única llamada directa al manejador. La
   guardia de reentrada inspecciona el marco completo y no convierte
   `FUNCNAME[1]`, una función auxiliar intermedia o el orden del planificador
   en autoridad causal.

Este contrato no rebaja silenciosamente «gana el primer `kill`»: elimina esa
promesa porque las señales estándar no la sostienen. Si negocio necesitase
orden cronológico de solicitudes de cancelación, `INT`/`TERM` no serían el
canal adecuado. Haría falta un canal ordenado y autenticado y una decisión
nueva; no un quinto auxiliar que intentase inferir orden perdido.

## Criterio de contrarrevisión

Antes de aceptar una implementación C4b-1, un revisor distinto debe:

- comprobar por separado `INT` única = 130 y `TERM` única = 143;
- repetir al menos cien veces ambos órdenes y las ráfagas de tres señales, sin
  exigir que el orden de `kill` determine el resultado anterior al marco y
  exigiendo siempre un único valor de `{130, 143}`;
- inyectar reentrada al inicio del manejador de forma directa y mediante
  marcos `DEBUG`, función auxiliar, `ERR` y `RETURN`, y demostrar que la
  primera señal ya observada no cambia;
- acreditar exactamente una cancelación, cero efectos/trabajos nuevos,
  restauración del régimen shell y residuos cero;
- inspeccionar que ningún oráculo, mensaje o acta vuelva a presentar el primer
  envío como la primera entrega, y emitir `P0=0`, `P1=0` y `P2=0`.

## Alcance pendiente

Esta decisión solo corrige el contrato de señales de C4b-1. C4b-2
(hijo/grupo y plazo), C4b-3 (Docker, temporal y epílogo), C4c, C4d, las
métricas y la autorización de producción permanecen abiertos. Producción
sigue en `NO-GO`.
