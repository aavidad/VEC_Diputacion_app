# Enmienda F0-H0b/C4b-2 G2-S: ledger físico real de 800 líneas

Fecha: 5 de agosto de 2026.

Estado: **aceptada** tras doble revisión independiente con `P0=P1=P2=0`.

## Motivo

La enmienda G2-S aceptada proyectó un runner de 797 líneas sobre una base de
789. El primer candidato `ccd72365a86b9afe623806e3f4bebd511ec13830`
demostró que la separación funcional era viable, pero recibió doble `NO-GO`:
sus dos builds compartían `HOME`, `TMPDIR` y `GOCACHE`, el runner ocupaba 799
líneas frente al ledger previsto y dos controles se habían condensado para
caber.

La corrección `dd01b44449ec3953e9706991ecfd59751337a78e` no elimina ni
minifica controles. Introduce raíces privadas distintas para los dos builds,
restaura el formato legible de asignaciones y estados y deja el runner en el
tope duro de 800 líneas fijado por DEC-051 y por la propia enmienda G2-S.

## Ledger comprobable

Desde la base estable `0a6de822dd735f9e0f940230b9f6a6bf7a822608` hasta la
corrección candidata:

```text
41  30  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
3   74  supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go
91  0   supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go
```

| Unidad | Base | Candidato corregido | Máximo de este corte |
| --- | ---: | ---: | ---: |
| Runner | 789 | 800 | 800 |
| Supervisor G1 | 754 | 683 | 790 |
| Supervisor G2 | inexistente | 91 | 160 |
| Capturador | 799 | 799, invariante | 799 exacto |
| Adaptador M38 | 527 | 527, invariante | 527 exacto |

Las tres líneas adicionales frente a la previsión documental de 797 son
necesarias para acreditar aislamiento físico y conservar controles legibles.
No constituyen reserva para código posterior.

## Condiciones de aceptación

La autorización de 800 líneas queda limitada a G2-S y al runner exacto de la
corrección. Exige que la revisión independiente reproduzca:

- dos raíces nuevas, privadas, no simbólicas, distintas y con modo `0700`;
- `HOME`, `TMPDIR` y `GOCACHE` disjuntos por build, neutralizando valores del
  entorno de entrada;
- cachés inicialmente vacías y compilación forzada en ambos builds;
- forma y SHA-256 de G1 y G2 antes y después de compilar;
- forma, enlace único y SHA-256 literal e idéntica de ambos binarios;
- G1 100/100, modos cerrados en 64, FD e hijos invariantes y residuos cero;
- manifiesto exacto de seis, mutantes de captura y huellas invariantes;
- ausencia de G2-O, Docker, PostgreSQL, red, SQL o autoridad productiva.

El runner no dispone de una línea libre. G2-O, C4b-3 o cualquier trabajo
posterior que necesite modificarlo requiere una separación o topología nueva
aprobada antes de programar; no se comprimen ni retiran controles para crear
espacio.

Esta enmienda no cierra G2-S, C4b-2, C4b, H0b, C2, F0, O4-05 ni producción y
no modifica métricas.
