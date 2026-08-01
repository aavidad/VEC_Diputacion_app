# Revisión F0-D2: implementabilidad y concurrencia

Fecha: 1 de agosto de 2026.

## Resultado

**GO documental**, confirmado por dos revisores independientes con
`P0=0`, `P1=0` y `P2=0`.

La serie integrada es:

```text
a5ba276  cerrar concurrencia y paquete componible
d0e6136  cerrar ensayos dormidos
cebc8bd  cerrar grafo y concurrencia
```

Corresponde a los commits productores `b3a6b02`, `bc2e5f9` y `1c6243f`.
El documento final tiene 673 líneas y SHA-256
`22507dbfc8efcc76f50605c7b4eacf260dd2b73dc295d4ebed2804c2fd220732`.

## NO-GO corregidos antes del cierre

La primera versión no fue aceptada. Las contrarrevisiones obligaron a cerrar:

- el grafo completo `H0`--`I0`, sus dependencias y write-sets literales;
- la ampliación de audiencias `3→7` antes de los consumos positivos;
- la retirada separada `R2a/R2b` y el orden inverso exacto;
- la entrada operativa única por runner, una transacción y un `txid` para
  cada envoltorio, y el rechazo léxico de todo control transaccional, incluido
  `ABORT`;
- la clasificación estructurada de SQLSTATE, sin interpretar mensajes
  localizados, y los reintentos acotados del envoltorio completo;
- el plan cerrado de locks `NOWAIT`, incluido `checkpoint_gobierno`, los
  modos finales requeridos por las FK y la prohibición de ascensos;
- la separación entre carreras controladas, donde `40P01` es un fallo, y la
  prueba defensiva aislada de recuperación ante ese SQLSTATE;
- `max_prepared_transactions=0`, ausencia de `pg_prepared_xacts` y limpieza
  byte a byte después de cada ensayo dormido.

Ningún revisor modificó el candidato. El productor corrigió cada NO-GO y
volvió a presentar el diff completo hasta obtener el doble GO final.

## Evidencia PostgreSQL de dirección

Dirección reprodujo en PostgreSQL 18.4 dos premisas antes de cerrar D2:

1. el disparador centinela deshabilitado crea una dependencia normal real a
   `serializar_revocacion_consultas_rrhh_v3()`; `DROP FUNCTION RESTRICT`
   falla mientras existe y el rollback conserva el estado anterior;
2. crear una tabla hija con FK toma en la tabla padre `AccessShareLock` y
   `ShareRowExclusiveLock`, lo que justifica preadquirir directamente el modo
   final y no ascenderlo durante la migración.

Esta evidencia valida las premisas del contrato. No sustituye las pruebas
Q1--Q3 ni las tres pasadas completas que deberá ejecutar la implementación.

## Comprobaciones del candidato final

- dos revisiones completas: GO y `P0=P1=P2=0`;
- `git diff --check` y `git show --check`, correctos;
- Gitleaks sobre el commit final, sin fugas;
- un único fichero modificado por el productor;
- 673 líneas, por debajo del límite de 800;
- grafo, rutas, write-sets, locks, reintentos y retirada trazados hasta sus
  pruebas propietarias.

## Alcance del GO

D1 y D2, conjuntamente, autorizan comenzar la implementación por las
minitareas paralelas `H0` y `V0`. No acreditan todavía el SQL `000007`, sus
componentes, PostgreSQL E2E ni producción. Las métricas funcionales no cambian
hasta que el corte completo se integre y supere Q1--Q3, tres pasadas limpias y
revisión independiente.
