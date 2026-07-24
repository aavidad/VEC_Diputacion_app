# Corrección candidata O2-06

Fecha: 24 de julio de 2026.

Estado: `LISTA PARA REVISIÓN INDEPENDIENTE`. Este documento no sustituye por
sí solo el `NO-GO` previo ni autoriza integración o producción.

Commits principales: corrección de código `1ca248b` y documentación candidata
`fa65da4`.

## Alcance

La corrección parte de `10ed26d` e incorpora los commits de implementación
O2-06 más dos correcciones bloqueantes:

- candidatura técnica durable, opaca y no autoritativa antes de proyectar y
  solicitar la decisión V3;
- clasificación determinada de `ErrTxCommitRollback` y
  `pgconn.SafeToRetry`, y reconciliación solo ante resultado realmente
  indeterminado.

La migración `000006_candidatura_tecnica_o2_06` añade:

- tablas técnicas inmutables separadas de expedientes, reservas, auditoría y
  outbox;
- alias HMAC por generación para rotación sin cruzar pares;
- resolver idempotente con privilegio nominal y sin lectura directa;
- fachada `confirmar_alta_atestada_v2`, que convierte la candidatura en
  identidad oficial y confirma el agregado dentro del mismo `COMMIT`;
- reversión protegida cuando existen candidaturas aún no confirmadas.

`preparar_alta_v2` continúa revocada. El ejecutor tampoco puede invocar
directamente `confirmar_alta_atestada_v1`, leer tablas ni ejecutar el
reconciliador interno.

## Respuesta a los bloqueantes

### CT-O2-06-R1

El caso de uso recibe un `ResolutorCandidaturaAlta`. La primera invocación
estabiliza reserva, expediente, número visible y recibo; otra instancia del
servicio recupera exactamente esas referencias aunque su generador proponga
otras. PostgreSQL 18 prueba además:

- recuperación desde un pool nuevo;
- misma clave y huella con propuesta distinta;
- rechazo de la misma clave con huella distinta en todas las generaciones;
- rechazo si solo diverge la generación activa;
- conversión oficial exclusivamente dentro de la confirmación atestada.

### CT-O2-06-R2

El adaptador trata como determinados `pgx.ErrTxCommitRollback` y los errores
que `pgconn.SafeToRetry` acredita como no enviados. Solo `40001` y `40P01`
abren otra transacción ordinaria. `08007` y los fallos de transporte
potencialmente posteriores al envío pasan a una reconciliación acotada con los
mismos doce bytes. Una segunda ambigüedad nunca se presenta como éxito.

## Evidencia local

Superado:

- pruebas unitarias y de carrera del adaptador PostgreSQL;
- `go vet` del paquete;
- PostgreSQL 18 efímero fijado por digest, sin red, repetido 3/3 sobre el
  árbol candidato;
- canon Go/SQL, alta, replay, cuatro confirmaciones concurrentes, cancelación
  pre-`COMMIT`, respuesta perdida, segunda ambigüedad y recibo adulterado;
- candidatura y confirmación tras pool/proceso lógico nuevos;
- reutilización de ámbito con huella divergente;
- ACL de mínimo privilegio y lecturas/preparación/reconciliación rechazadas;
- `down` protegido con candidatura huérfana;
- migración ascendente por debajo del límite duro de 800 líneas;
- pruebas globales, carrera global (`internal/vec/ports`: 488,252 s),
  `go vet ./...`, aislamiento de grafos/manifiestos, TLS no privilegiado,
  análisis de vulnerabilidades, tamaños y `git diff --check`;
- Gitleaks sobre los ocho commits `10ed26d..fa65da4`: cero filtraciones.

Antes de integrar aún se exige:

1. revisión independiente del rango candidato;
2. registrar hash, evidencia y veredicto independiente en tablero y relevo.

## Trabajo posterior

O2-07 compondrá el pool exclusivo, el resolver, el proveedor de material V3 y
la transacción de altas con fallo cerrado. O2-08 y O2-09 ya disponen de
adaptadores revisados, pero la ruta y el E2E O2-10 siguen abiertos.
