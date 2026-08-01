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
f57088c  separar arnés y cerrar snapshot exacto
```

Los tres primeros corresponden a los commits productores `b3a6b02`,
`bc2e5f9` y `1c6243f`. D2c fue una corrección de dirección nacida al probar el
primer arnés H0. Después de integrarla, la rama H0 la incorporó byte a byte
como `0e2de5f` para sincronizar su dependencia; no es un origen productor
anterior. El documento final tiene 771 líneas y SHA-256
`cd9444f110fd1b77a5f28766502561eed44d66b5a8d1a3dc51b15a740558da28`.

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

La implementación exploratoria de H0 reveló después que un único shell no
dejaba margen revisable para I0. D2c se sometió a dos revisiones completas y
no se aceptó en sus primeros ciclos. Los NO-GO obligaron a cerrar:

- un write-set H0 exacto de tres artefactos: runner, helper shell privado y
  capturador Go de prueba; solo I0 podrá volver a escribir el runner;
- la carga del helper exclusivamente desde una copia privada capturada, nunca
  desde el árbol vivo, y la distinción entre el oráculo `_test.go` y el
  capturador `package main`;
- la cadena de arranque del propio capturador mediante ruta y SHA-256
  literales, copia exclusiva, toolchain local, compilación sin red y
  autoprueba con detector de carreras;
- la raíz física retenida con `os.OpenRoot(".")`, el descenso por descriptores,
  los contrastes `Lstat`/`fstat`, `nlink=1` y la lectura y huella desde el
  mismo descriptor;
- una segunda huella literal para autenticar el helper antes de `source`, no
  solo garantizar su estabilidad;
- el límite H0 de 550 líneas en el runner y la reserva mínima de 249 líneas
  para que I0 mantenga el tope final de 799.

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
- un único fichero modificado en cada corte documental;
- 771 líneas finales, por debajo del límite de 800;
- grafo, rutas, write-sets, locks, reintentos y retirada trazados hasta sus
  pruebas propietarias.

## Alcance del GO

D1, D2 y D2c, conjuntamente, autorizan implementar H0; V0 ya quedó integrado
con revisión independiente. No acreditan todavía el SQL `000007`, sus
componentes, PostgreSQL E2E ni producción. Las métricas funcionales no cambian
hasta que el corte completo se integre y supere Q1--Q3, tres pasadas limpias y
revisión independiente.
