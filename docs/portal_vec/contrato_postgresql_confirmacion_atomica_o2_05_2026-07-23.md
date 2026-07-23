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
proyección `efecto-alta.v2` conserva, en un canon cerrado, identidad y
referencias, flujo, fase y estado, solicitud completa, periodo, RC, importe,
documentos, observaciones, fechas y actuación inicial. Su huella entra en el
contexto de recurso autorizado; cambiar por separado cualquiera de sus 45
campos o nodos probados invalida el efecto antes de persistir.

## Invariantes del único `COMMIT`

En una transacción `SERIALIZABLE` se realizan, o se revierten juntas:

1. validación de capacidad, decisión y evidencia;
2. gobierno y revocaciones HMAC y de confianza;
3. reconciliación histórica y revalidación viva de sesión, contexto,
   asignación, rol, políticas, motivo y ventana;
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

El runner no confunde una remarshalización con interoperabilidad: entrega a
Go una capacidad efímera, el emisor MAC de producción genera la MAC real y
los bytes canónicos, y PostgreSQL los consume con la clave durable de la base.
Una adulteración posterior se rechaza sin consumo.

Además ensaya:

- éxito, replay y recibo idéntico con `DateStyle` ISO y German;
- mutación individual del canon completo y tipos JSON estrictos;
- límite entero interoperable `2^53-1`;
- clave activa, retenida, revocada y provisionada nunca activada;
- configuración futura sin avance prematuro del checkpoint;
- restauraciones obsoletas de configuración y raíz rechazadas
  específicamente por el checkpoint;
- decisión ya durable seguida de revocación viva de sesión;
- concurrencia con reintento exclusivo de `40001`/`40P01` y cabezas de
  auditoría/outbox;
- fallos parciales, ACL, retirada protegida, destrucción explícita,
  reinstalación y segunda retirada sin inventario residual.

La migración transversal de revalidación viva tiene ciclo propio en el runner
de autorización. Su `down` falla cerrado mientras siga instalado el
consumidor O2-05, por lo que el orden obligatorio es
`consumidor -> revalidación viva`.

La evidencia del productor es necesaria, pero no suficiente para un `GO`.
Dirección asignará un revisor distinto, reproducirá los mandatos y actualizará
el tablero solo después de integrar.

## Incidencias detectadas durante el endurecimiento

Se conservan en la memoria técnica los fallos intermedios porque también son
evidencia de la utilidad del runner:

1. una clave provisionada de prueba reutilizaba una huella secreta protegida
   por `UNIQUE`; se corrigió generando material efímero independiente;
2. Docker copiaba el JSON de capacidad Go con modo `0600` y PostgreSQL no
   podía leerlo; se hizo legible únicamente ese JSON, nunca el secreto;
3. el fixture seleccionaba un puntero futuro que el consumidor correctamente
   ignoraba; se alineó con `establecida_en <= clock_timestamp()`;
4. la primera reinstalación intentó aplicar la migración después de que el
   `down` hubiera eliminado el esquema; ahora repite el bootstrap completo de
   roles y permisos antes de reinstalar;
5. la primera prueba de retroceso chocaba con una restricción `UNIQUE` antes
   de alcanzar el checkpoint; se sustituyó por una restauración obsoleta
   simulada en el contenedor efímero y se afirma expresamente que el valor
   presentado es menor que el mínimo;
6. dos aserciones shell comparaban el booleano de `psql -At` con `true` en vez
   de `t`; se corrigió la expectativa y se repitió el ciclo completo.

Después de cada corrección se ejecutó de nuevo el runner desde una base
PostgreSQL 18 vacía. Ninguno de estos fallos se oculta ni se interpreta como
un `GO`.
