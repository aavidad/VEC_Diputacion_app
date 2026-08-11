# Checkpoint F0-H0b/C4b-2/G2-O — O4B-P0-CONTRATO

Fecha: 12 de agosto de 2026.

Estado: **CANDIDATO A REVISIÓN**.

## Corte

- Base exacta: `1946b671afbcc47c681d6d6bb22e3c1122935247`.
- Rama: `trabajo/o4b-p0-contrato-20260812`.
- CI base: [31543335809](https://github.com/aavidad/VEC_Diputacion_app/actions/runs/31543335809), `Success` 5/5.
- Decisión: [O4b señales funcionales de grupo](../decision_f0_h0b_c4b2_g2o_o4b_senales_grupo_2026-08-12.md).
- Write-set: decisión, este checkpoint y dos actas independientes; todos son Markdown nuevos.
- Código, pruebas, herramientas, workflows, SQL, O3 y O4a: byte-inmutables.

## Resultado contractual

O4b queda delimitado como ejecutor único de STOP/TERM/CONT/KILL de grupo por
pidfd primario fiable y autorización O4a opaca one-shot. No decide causa,
incidente, etapa o plazo; no recoge, drena, escribe TERMINAL ni libera.

La decisión cierra:

- máquina OB0--OBF, consumo, carrera y fatalidad;
- señalización exclusiva `pidfd_send_signal` con flag de grupo;
- permiso lease distinto por cada syscall y consolidación previa;
- tabla total STOP, TERM→CONT, STOP final y KILL;
- deadlines prestados y KILL inmediato antes del borde;
- evidencia mínima no recolectora mediante primitivas O3 existentes;
- resultado opaco mínimo hacia O4a;
- DAG `O4B-P0→P1→…→P7` y dependencias sin ciclo con O4A/O4C.

## Bloqueos

`O4A-P4-ETAPAS` y toda implementación O4b permanecen bloqueados hasta doble
GO, push y CI 5/5 de este corte. `O4C-P0` depende solo de los contratos O4a y O4b;
`O4A-P5` depende además del contrato O4c y del resultado O4b material.

No se modifican Contratación temporal `24/46`, Bolsa `1/14` ni producción
`NO-GO`. No se integra ni fusiona master.

## Puertas pendientes de cierre

1. revisión funcional independiente completa;
2. revisión de seguridad independiente completa;
3. corrección y relectura de cualquier NO-GO;
4. hashes/enlaces/formato/Gitleaks/diff-check verdes;
5. commit documental pequeño, push normal y CI exacta 5/5.
