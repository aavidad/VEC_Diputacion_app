# Checkpoint O4A-P0-CONTRATO

Fecha: 11 de agosto de 2026.

Estado: **GO DOCUMENTAL PARA PUBLICACIÓN, P0=P1=P2=0**. No autoriza código,
integración, O4b/O4c efectivo, producción ni cambio de métricas.

## Base y alcance

- Base exacta: `0029cfe03b2f2f637169be8340985e38b1fa6557`.
- Rama de origen: `trabajo/o3c-p7-evidencia-20260811`.
- CI de origen: `31509858146`, `Success`, cinco jobs concluidos.
- Write-set: decisión O4a, este checkpoint y dos actas de revisión; cero
  código, pruebas de implementación, herramientas, migraciones o documentos
  transversales.
- O3a/O3b/O3c permanecen byte-inmutables.

## DAG propuesto

`O4A-P0→P1-AUTORIDAD→P2-SEMILLA→P3-ARBITRAJE→P4-ETAPAS→P5-HANDOFF→P6-CONDUCTOR→P7-EVIDENCIA`.

`O4B-P0-CONTRATO` depende del contrato O4a publicado y CI 5/5.
`O4C-P0-CONTRATO` depende de los contratos O4a y O4b publicados y CI 5/5.

## Snapshot material y revisión

La decisión final tiene 535 líneas y SHA-256
`ffe3d570b3fe51be96948f8aeb0b163e2ff071d2c2cca4fbdba852a5f9dafebc`.
Las revisiones [funcional](revision_funcional_f0_h0b_c4b2_g2o_o4a_p0_contrato_2026-08-11.md)
y de [seguridad](revision_seguridad_f0_h0b_c4b2_g2o_o4a_p0_contrato_2026-08-11.md)
releyeron esos bytes completos y emitieron `GO`, `P0=P1=P2=0`.

Queda condicionado a commit documental pequeño, push normal y CI 5/5 sobre el
SHA exacto. No hay merge ni apertura de implementación.

## Límites y siguiente arista

O4a posee solo causa, precedencias, tiempo y permisos opacos. O4b posee señales
funcionales; O4c, terminalidad y limpieza. Tras publicación/CI, dirección puede
asignar únicamente `O4A-P1-AUTORIDAD` o el contrato dependiente O4b; este corte
no los abre por sí mismo.
