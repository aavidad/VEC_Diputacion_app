# Revisión de seguridad O4C-P0-CORRECCIÓN-INVENTARIO

Fecha: 13 de agosto de 2026.

Revisor: agente independiente de seguridad, rama
`revision/o4c-p0-correccion-inventario-seguridad-20260813`, sin edición del
worktree ni de la rama productora.

Dictamen: **GO**, `P0=0`, `P1=0`, `P2=0`.

## Corte exacto y aislamiento

Se revisó exclusivamente el candidato
`de4ee5a11c3611e56ab09899da860ef88647ea0c`, con padre exacto
`64351a469df4e7bba4850ef1500d9e2c4bf378de` y árbol
`6be07800da0f90f1bd61f370ecfe57c9a9f6d108`. El único commit es
`docs(O4C-P0): corrige inventario de cinco FD`.

El delta corrector modifica solo los dos documentos O4c declarados, con 32
inserciones y 11 borrados:

| Documento corregido | Delta | Líneas finales | SHA-256 final |
| --- | ---: | ---: | --- |
| `docs/portal_vec/decision_f0_h0b_c4b2_g2o_o4c_terminalidad_limpieza_2026-08-13.md` | `+9/-6` | 555 | `55e970da04933d7eb1287ec06d7b8f24767c25712d04d5111eedf6eb52b687bc` |
| `docs/portal_vec/revisiones/checkpoint_f0_h0b_c4b2_g2o_o4c_p0_contrato_2026-08-13.md` | `+23/-5` | 115 | `9132d26c3fd2cb1e346ca6d67bef97535472ab3b628af273d7d0402301275746` |

Respecto de la base normativa original
`5345d5d097b51ab3567983f048feabeceaf2957b`, el candidato completo sigue
teniendo únicamente esas dos altas. Por tanto, el checkpoint distingue sin
contradicción el write-set corrector de dos modificaciones y la puerta del
candidato completo de dos altas. La decisión conserva la rama de producción
original y el checkpoint fija además padre y rama correctores exactos.

`AGENTS.md`, lectura obligatoria, decisiones O1a/O3a/O3b/O3c/O4a/O4b,
código, pruebas, herramientas, roadmap, handoffs, ledger transversal y
métricas son byte-inmutables frente al padre. Sus bytes ya leídos completos en
esta misma sesión se revalidaron antes de editar; también se leyó completa la
decisión corregida, su checkpoint y la revisión funcional que originó el P1.

## Cierre exacto del hallazgo heredado

La corrección elimina la cardinalidad imposible del padre y fija un conjunto
único de cinco descriptores físicos:

```text
{pidfdOpaco, pidfdPrimario, pidfdReserva, CONTROL, TERMINAL}
```

La cifra y los nombres concuerdan con todas las autoridades aplicables:

- O3a establece dos pidfd explícitos y el handle opaco de `Process`, contado
  como tercera y única referencia adicional; `cmd.Wait` libera ese handle;
- O3c conserva exactamente tres referencias pidfd, sin cuarta, además de
  CONTROL y TERMINAL;
- el helper O3c ya acreditado materializa un arreglo `[5]int` compuesto por
  primario, reserva, opaco, CONTROL y TERMINAL para cotejar el inventario
  liberado;
- O4c consume `pidfdOpaco` mediante el Wait único y luego cierra, en orden y
  bajo permisos separados, primario, reserva, CONTROL y TERMINAL.

Así, al llegar al snapshot final han desaparecido exactamente cinco FD, no
seis. `Cmd` y `Process` se anulan después como referencias Go y no constituyen
otro descriptor ni autorizan restar una segunda vez el handle opaco.

La misma cardinalidad queda sellada de extremo a extremo:

- la secuencia positiva enumera los cinco nombres y prohíbe cambiar un FD
  ajeno;
- OC02 exige owners, estados, registro, generación, TID, `pending` y los cinco
  FD exactos antes de adquirir autoridad;
- OC16 exige que cada uno se cierre o consuma una vez y que el snapshot reste
  exactamente esos cinco;
- los mutantes OC02/OC16 cubren omisión de cada miembro, duplicación,
  reordenación, conteo de seis y modificación de un FD ajeno.

No queda un hueco de cardinalidad, alias o doble cierre entre autoridad,
conducta, oráculo y mutantes.

## Seguridad y fallo cerrado preservados

La corrección no altera el resto del contrato y conserva:

- agregado privado, preasignado, no serializable y one-shot; transferencia
  observador→lease y consumo mediante CAS con un solo ganador;
- owners O4C, estados físicos 2/3, registro, generaciones, TID, `pending`,
  identidad, flags, huellas y límites exactos antes de cualquier syscall;
- terminalidad natural concordante de primario/reserva antes del único
  `cmd.Wait`, sin señal cero, PID/PGID, `/proc`, fallback ni segundo Wait;
- Wait4 no bloqueante hasta ECHILD y prueba de grupo exclusivamente por ESRCH;
  PID cero, EPERM, EINTR, límite o duda nunca acreditan limpieza;
- permisos lease separados y consolidación crítica previa a interpretar cada
  raw, cierre único sin retry incierto y TERMINAL normal solo con
  postcondiciones completas;
- inventario previo a liberar observador; lease como última capacidad, sin
  syscall, lectura, escritura, log o asignación falible posterior;
- incidente/cuarentena 65 sin sustituir causa, autoridad, estado ni
  postausencia y sin exponer PID, pidfd, nonce, ticket o error libre.

La indisponibilidad y la discrepancia de inventario siguen conduciendo a OCF,
cero caso siguiente y ninguna reparación ficticia.

## Autoridades, enlaces y puertas

Se reprodujeron sin divergencias los siete hashes citados de O4a, checkpoint
O4a, O4b, checkpoint O4b, O3c, ledger O3c y corrección O1a. También quedaron
verdes:

- HEAD, padre, árbol, ancestry y único commit exactos;
- delta corrector de dos modificaciones, `+32/-11`, y delta completo de dos
  altas respecto de `5345d5d`;
- SHA-256 y líneas de decisión/checkpoint corregidos;
- existencia de todos los enlaces Markdown locales;
- `git diff --check` sobre
  `64351a469df4e7bba4850ef1500d9e2c4bf378de..de4ee5a11c3611e56ab09899da860ef88647ea0c`;
- búsqueda focal de secretos, tokens, claves, DSN, rutas privadas y datos
  personales: solo menciones normativas;
- Gitleaks sobre el mismo rango: un commit, 1,84 KB, cero filtraciones;
- worktree candidato limpio antes de esta acta.

No aplican focal Go normal/race, gofmt, vet, mutantes, PostgreSQL ni E2E: este
corrector modifica únicamente dos Markdown y no cambia paquete, fuente,
ejecutable, SQL ni composición. El contrato mantiene esas puertas obligatorias
para los cortes materiales P1--P7; este GO no las declara ejecutadas.

## Cierre

No quedan hallazgos P0, P1 ni P2 en el candidato exacto. El GO acredita solo
la corrección documental del inventario de cinco FD. No autoacredita el padre,
O4A-P4, O4A-P5, O4C-P1 ni ningún descendiente, y no autoriza integración,
publicación, despliegue, producción o cambio de métricas. Dirección deberá
exigir el otro GO independiente y aplicar sus reglas de integración antes de
considerar cerrado O4C-P0.
