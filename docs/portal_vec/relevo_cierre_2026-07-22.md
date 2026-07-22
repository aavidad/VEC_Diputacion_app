# Relevo de cierre — 22 de julio de 2026

## Motivo y criterio

La sesión se detuvo por petición expresa del responsable para conservar cuota.
Este documento fija únicamente evidencia comprobada. C3 no recibe todavía GO:
las auditorías finales fueron interrumpidas y sus ramas aún no se han integrado
en la rama de convergencia.

## Estado preservado en Git

| Trabajo | Rama | SHA | Estado al cierre |
|---|---|---|---|
| C3 PostgreSQL público multiversión | `real/c3-disponibilidad-lectores` | `86d40ce` | Limpia, publicada y con seguimiento remoto |
| Web de categorías históricas y doble huella | `real/c3-web-historico` | `7047972` | Limpia, publicada y con seguimiento remoto |
| Disponibilidad real de la proyección | `real/c3-readiness` | `bb12c7b` | Limpia, publicada y con seguimiento remoto |
| Convergencia C1/C2/C4 | `real/c2-artefactos-fisicos` | anterior a este documento: `a8a5d7f` | No contiene aún las tres ramas C3 |
| Presentación y borde remoto TLS | `vec-orquesta-20260619` | `96c6396` | Limpia y publicada |

## Evidencia ejecutada en esta sesión

- C3 focal:
  `go test ./internal/modules/bolsa/publico/... ./internal/modules/bolsa/adapters/postgrespublico/... ./internal/app/composicion/publica/...` — superado.
- C3 completo: `go test ./...` — superado.
- PostgreSQL público:
  `deploy/postgresql/bolsa_publica/probar_integracion.sh` — superado con
  PostgreSQL 18 y TLS verificado; el runner terminó con código cero.
- Web C3: el agente implementador informó
  `node --test web/static/bolsa/*.test.mjs` — 19/19 superadas sobre `7047972`.
- Presentación remota: `bash -n` del generador TLS y
  `docker compose config -q` — superados antes de `96c6396`.
- `scripts/verificar_calidad.sh` se interrumpió por la orden de cierre mientras
  ejecutaba su pasada con carreras. La salida anterior era verde; la
  interrupción produjo `signal: interrupt` en `internal/vec/ports`. No debe
  registrarse como fallo funcional ni como puerta superada.

## Cambios funcionales ya preservados

- Lecturas públicas bajo MVCC sin bloquearse con el publicador.
- Catálogos profesionales multiversión y resolución histórica exacta.
- Referencia vinculada por versión, huella gobernada y huella de proyección.
- Conjunto vigente actual independiente del material histórico.
- Redacción de errores PostgreSQL y requisito operativo de ocultación de
  parámetros en el usuario `LOGIN` real.
- Web que falla cerrada si falta o cambia cualquiera de las dos huellas.
- `livez`, `readyz` y comprobación de integridad de la proyección en la rama de
  disponibilidad.

## Trabajo exacto para la siguiente sesión

1. Reanudar y terminar las tres auditorías independientes interrumpidas:
   backend C3, web de doble huella y disponibilidad. Exigir cero hallazgos
   medios o superiores.
2. Corregir cualquier hallazgo y repetir las pruebas focales, con carreras y el
   runner PostgreSQL 18/TLS.
3. Integrar en una rama limpia basada en `real/c2-artefactos-fisicos`, en este
   orden: `86d40ce`, `7047972` y `bb12c7b`; resolver la compatibilidad del DTO y
   de disponibilidad contra el contrato C3 definitivo.
4. Ejecutar Node, `go test ./...`, `scripts/verificar_calidad.sh`, runner
   PostgreSQL y las puertas de artefactos/Docker de superficies separadas.
5. Solo tras todo lo anterior, cambiar C3 a GO en
   `matriz_estado_operativo_bolsa_2026-07-18.md` y actualizar porcentajes.
6. Continuar con C5 (identidad interna fuerte) y las capacidades restantes de
   Bolsa. La demo no sustituye ningún adaptador productivo.

## Invariantes de continuación

- Denegación por defecto y separación exterior/interior.
- Sin cookies de autenticación en la superficie interna prevista para cliente
  de escritorio.
- Sin datos personales, secretos ni material criptográfico en Git.
- Ninguna rama se declara productiva solo porque compile o tenga una demo.
- Los cambios de contrato, seguridad o persistencia requieren auditoría
  independiente antes de converger.
