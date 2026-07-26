# Relevo de cierre de sesión — 25 de julio de 2026

Actualizado el 26 de julio de 2026 tras recuperar y revisar los cortes
interrumpidos. Este fichero conserva el nombre original para no romper sus
enlaces.

> Documento histórico superado por el
> [cierre O4-04E](o4_04e_informe_cierre_confirmacion_durable_2026-07-26.md).
> La métrica vigente es 19/46 (41 %), no la cifra WIP conservada más abajo.

## Referencias conservadas

- Rama aprobada: `real/ct-o4-03`, commit `0775853`.
- Rama de resguardo no aprobada: `resguardo/ct-o4-04-20260725`.
- Rama de corrección publicada:
  `correccion/ct-o4-04-20260726`, commit `d4eb639`.
- O4-04A con revisión independiente `GO`: `b819961`.
- O4-04B con revisión independiente `GO`: `54a755e`.
- O4-04C con revisión independiente `GO`: `a540522`.
- Aislamiento Docker y documentación: `577f157`.
- Aislamiento web y corrección visual: `0775853`.

La rama de corrección está publicada en GitHub. Los commits nuevos fueron
revisados, probados y analizados con Gitleaks antes del envío.

## Trabajo interrumpido recuperado

La rama de resguardo conserva los dos commits `WIP` originales como evidencia
de recuperación, pero ya no son por sí solos la referencia de integración:

- `f480ad5`: corrección O4-04A interrumpida durante la tercera vuelta. La
  carencia de `go/types` se cerró en `b819961`, que enumera aliases,
  reexportaciones, interfaces embebidas, declaraciones y tipos anónimos dentro
  y fuera del paquete. Obtuvo `GO` independiente.
- `f274f76`: corrección O4-04B interrumpida antes de congelar el candidato y
  ejecutar su revisión final. `54a755e` cerró los cánones Go/SQL, las dieciséis
  funciones exactas sin `EXECUTE PUBLIC`, los `down` fuera de orden y la
  advisory lock común. Obtuvo `GO` independiente sobre PostgreSQL 18.4.

Los commits WIP continúan sin contarse como evidencia aislada. Las correcciones
posteriores sí son candidatos aprobados, aunque `O4-04` completo todavía no
puede cerrarse hasta terminar D/E.

## Estado del camino crítico

| Corte | Estado al cerrar |
| --- | --- |
| O4-04A | Cerrado, commit `b819961`, GO independiente |
| O4-04B | Cerrado, commit `54a755e`, GO independiente y PostgreSQL 18.4 |
| O4-04C | Cerrado, commit `a540522`, GO independiente |
| O4-04D | Cerrado, commits `dacf0e1`–`a2bb302`, GO independiente y PostgreSQL 18.4 |
| O4-04E | Listo: el canon durable de D ya está congelado |

O4-04D usa `000023`–`000024`, exige la versión de barrera `3` creada por C y
no modifica `000017`–`000022`. El wrapper VEC se concede únicamente al
propietario `NOLOGIN`; solo la futura función exterior de E será ejecutable por
el rol de ejecución.

## Presentación web

La presentación aprobada superó:

- 361 pruebas web;
- 9 pruebas Python de composición y aislamiento;
- 51/51 recorridos visuales de Contratación temporal;
- capturas reales de Bolsa y Contratación sin errores de consola;
- caída de cartografía con Bolsa y Contratación todavía disponibles.

Las evidencias de navegador permanecen locales e ignoradas por Git en
`.evidencias-locales/`; no contienen credenciales ni forman parte del
artefacto productivo.

## Reanudación recomendada

1. Continuar desde `correccion/ct-o4-04-20260726`.
2. Implementar O4-04E con una sola función exterior, recibo durable y
   reconciliación primaria sin reintento automático.
3. Ejecutar PostgreSQL 18, reinicio, concurrencia, fallos inyectados, ACL,
   carrera y puertas globales.
4. Solo entonces integrar el corte completo en la rama aprobada y actualizar
   el cómputo de tareas.

El procedimiento de Contratación temporal continúa oficialmente en 18/46
tareas verificadas (39 %). No se incrementa por trabajo WIP.
