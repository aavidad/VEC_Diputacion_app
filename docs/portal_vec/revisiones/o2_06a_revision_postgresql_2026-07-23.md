# Revisión PostgreSQL/pgx de O2-06A

Fecha: 23 de julio de 2026.

Revisor: especialista independiente PostgreSQL/pgx, en solo lectura.

## Dictamen

**GO condicionado** para el documento de diseño tras la segunda revisión.
**NO-GO** para declarar la firma ejecutable o iniciar implementación hasta que
Dirección comunique un SHA estable del consumidor SQL O2-05 y se supere la
puerta de acoplamiento.

No se leyó ningún worktree ni cambio sin commit del productor O2-05. No se
modificó ningún archivo durante la revisión.

## Evidencias

- El diseño O2-05 califica su firma como conceptual y pendiente de congelación:
  `docs/portal_vec/diseno_transaccion_atomica_alta_o2_05_2026-07-23.md`.
- El tablero conserva O2-05 en curso, consumidor SQL pendiente, y O2-06
  futura: `docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md`.
- `OrdenConfirmarAlta` solo contiene expediente, solicitud/decisión/
  confirmación V3, pares HMAC, preparación y correlación:
  `internal/modules/contrataciontemporal/ports/alta.go`.
- `registrar_decision_contexto_actor_v3` exige material y versiones que la
  orden actual no transporta:
  `deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql`.
- pgx está fijado en v5.10.0. Su `Commit` cierra la transacción y
  `ErrTxCommitRollback` acredita respuesta `ROLLBACK`; el wrapper del pool
  libera la conexión después:
  `github.com/jackc/pgx/v5@v5.10.0/tx.go` y `pgxpool/tx.go`.
- El adaptador de preparación ya demuestra `SERIALIZABLE`, `SET LOCAL`,
  rollback acotado y clasificación `40001`/`40P01`, pero carece de
  reconciliación:
  `internal/modules/contrataciontemporal/adapters/postgres/preparacion_alta.go`.
- VEC-AD-2 aporta un patrón estrecho de reconciliación y ACL, que O2-06 debe
  ampliar:
  `deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.up.sql`.
- DEC-101 exige `READ COMMITTED`, espera de la misma barrera y consulta solo
  después de terminar la escritura incierta:
  `docs/portal_vec/registro_decisiones.md`.
- En PostgreSQL 18.4 se reprodujo que una función PL/pgSQL
  `VOLATILE SECURITY DEFINER` puede adquirir
  `pg_advisory_xact_lock_shared` dentro de `READ COMMITTED READ ONLY`;
  `transaction_read_only` permaneció activo y `pg_locks` mostró el bloqueo
  compartido concedido.

## Cierres exigidos

1. Firma prevista de catorce entradas, mapeo Go↔PostgreSQL y puerta binaria
   contra el SHA estable O2-05.
2. Tres intentos totales, transacción nueva y reintento exclusivo de
   `40001`/`40P01`.
3. Distinción del punto de `COMMIT`: éxito `nil`, rollback concluyente,
   resultado indeterminado y reconciliación sin repetición.
4. Pool/LOGIN exclusivos, acreditación por conexión, `SET LOCAL`, sin DML,
   `TEMP` ni `SET ROLE`.
5. Reconciliación con todos los pares HMAC, identidad opaca,
   decisión/correlación, efecto y capacidad exacta del intento, en
   `READ COMMITTED READ ONLY` y tras la misma barrera. Espera y consulta son
   dos sentencias para renovar la instantánea.
6. `ReciboAlta` con once campos, procedencia y huellas de recibo, auditoría y
   evento; doce columnas con `resultado`, una sola fila, canon común y
   validación antes/después del `COMMIT`.
7. Reinicio sin memoria, txid, WAL/LSN ni reloj cliente.
8. Errores redactados; solo los dos SQLSTATE reintentables aparecen en
   telemetría interna.
9. Contexto V2 y manifiesto separados, con versiones de persona/perfil
   transportadas sin pérdida.
10. Cuatro resultados SQL cerrados y nulabilidad total; un `23505` genérico no
    se convierte en conflicto de dominio. Una denegación o conflicto valida
    once nulos y hace `ROLLBACK` antes de devolver el error, nunca `Commit`.
11. Reconciliación con plazo propio, cierre por `ROLLBACK` acotado y prueba
    completa no invalidada por un fallo posterior de limpieza.
12. Replay con concesión nueva consumida una vez, alias activo opcional y cero
    segundo efecto, reserva, expediente, actuación, outbox o recibo.
13. Resultado nominal `confirmada|replay`: confirmada coteja candidatos;
    replay ignora referencias CSPRNG e instante nuevos y devuelve el recibo
    histórico por alias/semántica, sin `ValidarPara(expedienteCandidato)`.
14. Wire/evidencia con framing binario exacto y vector común; `ErrTxCommitRollback`
    directo o envuelto se clasifica sin otro rollback, retry ni reconciliación.
15. Cadenas locales con génesis hex, cabeza inicializada a secuencia cero,
    orden de bloqueo, CAS de una fila, unicidades y rollback cruzado.

El diseño principal incorpora estos quince cierres.

## Matriz revisada

Se exigieron unitarias de firma, wire, mapeo, canonicidad, límites, copias,
filas 0/1/>1, recibo adulterado, cancelación y clasificación de `COMMIT`; y
PostgreSQL real para éxito, replay con candidatos nuevos, consumo exacto,
concurrencia, cadenas, rotación, revocación, expiración, snapshots obsoletos,
fallos por escritura, respuesta perdida, reinicio, ACL, timeouts, reversión y
reinstalación.

No se ejecutó el consumidor O2-05 porque no existe un SHA estable ni una firma
integrada que pueda probarse. Esta omisión es el bloqueo del dictamen, no una
prueba verde.

## Riesgo residual

El diseño no congela por sí mismo SQL. Si O2-05 cambia nombre, orden, tipos,
límites, columnas, barrera o ACL, O2-06 permanece en `NO-GO` hasta actualizar
el mapeo documental y repetir esta revisión.

**Dictamen final: GO condicionado para O2-06A como diseño; NO-GO para
implementación anticipada.**
