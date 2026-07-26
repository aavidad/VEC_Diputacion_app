# Revisión independiente O2-06 — NO-GO

Fecha: 24 de julio de 2026.

## Alcance revisado

- Base inmediata: `1d2cf7ed0dd96c7f291ca063bd6a9703992d3226`.
- Candidato: `157a589ddd5113a0d119f42c4e8fd7730f39952e`.
- Rango: `1d2cf7e..157a589`.
- Rama publicada: `origin/agent/ct-o2-06-implementacion`.

El revisor fue distinto del productor y no modificó el candidato.

## Veredicto

`NO-GO`, con dos hallazgos altos bloqueantes.

### CT-O2-06-R1 — Replay tras reinicio no conserva la candidatura

`ServicioRegistroSolicitud.Registrar` genera referencias aleatorias nuevas en
cada invocación. La confirmación O2-05 liga el canon exacto, que incluye esas
referencias. Una repetición semántica con la misma clave idempotente después de
perder la respuesta construye por tanto otro efecto y no puede recuperar el
recibo anterior.

Las pruebas verdes no acreditan el recorrido real: el doble del caso de uso
devuelve siempre las mismas referencias y las pruebas PostgreSQL reutilizan
directamente los mismos doce parámetros.

Corrección exigida:

1. resolver una candidatura técnica, durable y no autoritativa con las mismas
   referencias para el mismo ámbito HMAC y la misma huella de petición;
2. rechazar la misma clave y ámbito con huella distinta;
3. no restaurar `preparar_alta_v2` ni sus privilegios;
4. probar dos instancias nuevas del servicio y la frontera pública
   `ConfirmarAlta` contra PostgreSQL 18.

### CT-O2-06-R2 — Una cancelación segura pre-COMMIT puede confirmar

El adaptador trata como indeterminado cualquier error de `Commit` que no sea
`PgError` y reejecuta con un contexto sin cancelación. `pgconn.SafeToRetry`
indica precisamente cuándo pgx garantiza que no llegó a enviar datos. En ese
caso no existe ambigüedad que reconciliar: reejecutar puede crear el alta pese
a que la cancelación ocurrió antes de enviar `COMMIT`.

Corrección exigida:

1. `ErrTxCommitRollback` y `pgconn.SafeToRetry(err)` son resultados
   determinados sin efecto y no se reconcilian;
2. `40001` y `40P01` son reintentables con transacción nueva;
3. `08007`, la resolución realmente incierta o el transporte posiblemente
   enviado se reconcilian de forma acotada;
4. PostgreSQL 18 debe probar cancelación entre la fila y el envío de `COMMIT`,
   `08007`, una segunda ambigüedad y ausencia de falso éxito.

## Evidencia favorable que se conserva

El candidato sí superó PostgreSQL 18 para los mismos bytes, canon Go/SQL,
doce entradas y ocho columnas, concurrencia, respuesta perdida, recibo
adulterado y ACL; también pruebas focales y de carrera, globales, `go vet`,
compilación, revisión del árbol de fusión, tamaños y Gitleaks.

Estas evidencias son reutilizables, pero no compensan los dos bloqueantes. La
tarea O2-06 continúa abierta y el avance verificado permanece en 16 de 46.
