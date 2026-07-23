# Contrato PostgreSQL de confirmación atómica O2-05

Fecha: 23 de julio de 2026.

Estado: implementación candidata; pruebas del productor verdes; pendiente de
revisión independiente. No habilita producción.

## Decisión aplicada

La primera vertical durable de contratación temporal se materializa con una
sola función SQL:

```text
vec_contratacion_temporal.confirmar_alta_atestada_v1(
  capacidad, decisión, motivo, contexto de actor,
  versiones de persona y perfil,
  payload VEC-AD-3, sobre COSE, evidencia, raíz SPKI,
  proyección canónica del alta, sellos HMAC
) -> recibo mínimo
```

Todos los documentos cruzan la frontera como bytes canónicos con esquema
cerrado y versionado. No se persiste la clave idempotente en claro. La
proyección del alta conserva identidad, flujo, fase, estado, huella de la
solicitud y actuación inicial; la solicitud funcional sigue perteneciendo al
agregado Go y su adaptador definitivo debe aportar esa proyección sin
`reflect`, mapas abiertos ni serialización accidental del dominio.

## Invariantes del único `COMMIT`

En una transacción `SERIALIZABLE` se realizan, o se revierten juntas:

1. validación de capacidad, decisión y evidencia;
2. gobierno y revocaciones HMAC y de confianza;
3. CAS nominal de autorización V3;
4. consumo único para el efecto exacto;
5. convergencia de generaciones HMAC activa y retenidas;
6. confirmación de reserva;
7. expediente y versión inicial;
8. actuación registral;
9. auditoría encadenada;
10. outbox encadenado;
11. recibo opaco.

La repetición byte a byte devuelve el mismo efecto. La misma decisión, nonce
o alias con contenido diferente produce conflicto. Un fallo después del
consumo nominal revierte también ese consumo.

## Límites de confianza

- El productor de la capacidad no se autoaprueba.
- El emisor no verifica su propia firma COSE ni decide su propia raíz.
- PostgreSQL no acepta una decisión por existir en Go o por llegar desde HTTP.
- La autoridad de identidad no procede de cookies, cabeceras libres ni JSON.
- El runtime no posee tablas, DML, CAS nominal, preparación histórica ni el
  consumidor genérico.
- Los errores exteriores son deliberadamente opacos.

El checkpoint local evita retrocesos dentro de la historia visible en la base.
La protección frente a restaurar una copia antigua completa exige un ancla
monotónica externa y permanece como puerta de producción.

## Evolución compatible

VEC-AD-2 no cambia. VEC-AD-3 usa catálogo, audiencia y esquema propios. Las
generaciones HMAC se incorporan por migración aditiva; activa y retenidas
convergen en la misma reserva. Otro motor de datos requerirá otro adaptador
que demuestre estas mismas invariantes, sin cambiar dominio ni aplicación.

## Evidencia automatizada

El vector de 37 campos se comparte entre la prueba Go y la prueba SQL. El
runner PostgreSQL real ensaya éxito, replay, concurrencia, fallos parciales,
rotaciones, revocaciones, anti-retroceso, ACL y reversión protegida.

La evidencia del productor es necesaria, pero no suficiente para un `GO`.
Dirección asignará un revisor distinto, reproducirá los mandatos y actualizará
el tablero solo después de integrar.
