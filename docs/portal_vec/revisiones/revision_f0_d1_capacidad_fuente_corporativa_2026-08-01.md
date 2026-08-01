# Revisión F0-D1: capacidad de fuente corporativa

Fecha: 1 de agosto de 2026.

## Resultado

**GO documental**, con dos revisiones independientes finales y
`P0=0`, `P1=0`, `P2=0`.

La serie integrada es:

```text
c4fc55c  fijar capacidad de fuente corporativa
3a8a553  cerrar ligadura de fuente corporativa
1236c0b  asegurar retirada y evento atómico
```

El documento resultante tiene SHA-256
`cc770d46ada4f6b2a18b151224b942d20f13b1b0c99ee3f41216b2b97e731c74`.

## Hallazgos corregidos antes del GO

Las revisiones rechazaron los candidatos anteriores hasta cerrar:

- autoridad de fuente y evento derivada dentro de F0, no aportada por la
  fachada ni por un DTO;
- tipos canónicos exactos, incluido `configuracion_revision` textual;
- ligadura inequívoca de `efecto_ref` al recurso opaco C2.3;
- rotación gobernada de fuente, privacidad seudonimizada y conservación
  limitada;
- uso de las claves primarias existentes sin restricciones redundantes;
- evento atómico de un solo efecto y unicidad estable
  `(fuente_ref,evento_fuente_ref)` entre rotaciones;
- separación entre mecánica criptográfica y aislamiento por catálogo, raíz
  y audiencia;
- dependencia catalogal que impide retirar `000006` mientras F0 siga
  instalado.

## Retirada inversa

La decisión exige que `000007` cree un disparador centinela deshabilitado,
dependiente de
`serializar_revocacion_consultas_rrhh_v3()`. El `000006.down.sql` existente
ya rechaza dependencias adicionales mediante `pg_depend`; por tanto no se
modifica una migración histórica.

Dirección reprodujo la mecánica en PostgreSQL 18.4: el disparador quedó con
`tgenabled='D'` y una dependencia normal; `DROP FUNCTION ... RESTRICT` falló
mientras existía y el rollback conservó el disparador ordinario. Tras retirar
el centinela, la función pudo eliminarse. Esta comprobación valida el diseño,
no sustituye el futuro runner completo de `000007`.

## Comprobaciones reproducidas

- `git diff --check 6201541..54f92c4`;
- `git show --check` de los tres commits productores;
- Gitleaks sobre tres commits y 24,54 KB, sin fugas;
- enlaces relativos existentes;
- 442 líneas, por debajo del límite de 800;
- contraste estático con `000001..000006` y C2.3-D0;
- worktrees productor e integrador limpios antes de integrar.

## Alcance del GO

El GO de D1 cerró exclusivamente el contrato funcional y de seguridad. Por
sí solo **no autorizó escribir SQL**: aún faltaban el paquete componible, el
grafo de minitareas, los envoltorios atómicos, el plan exhaustivo de locks y
la retirada probatoria. Esas condiciones se cerraron después en F0-D2,
F0-D2a y F0-D2b, cuya revisión independiente se conserva en
[revision_f0_d2_implementabilidad_2026-08-01.md](revision_f0_d2_implementabilidad_2026-08-01.md).

D1 no acredita SQL, PostgreSQL E2E ni producción. Continúan cerradas las
puertas de fuente real, HSM/KMS, raíces, identidades técnicas, TLS/mTLS,
ENS/EIPD y aprobaciones formales.
