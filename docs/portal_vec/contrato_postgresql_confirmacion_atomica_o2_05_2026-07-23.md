# Contrato PostgreSQL de confirmación atómica O2-05

Fecha: 23 de julio de 2026.

Estado: implementación candidata corregida tras NO-GO; pruebas del productor
verdes; pendiente de dos revisiones independientes. No habilita producción.

Las migraciones de contratación conservan una secuencia única y monótona:
`000003_expediente_confirmacion_atestada`,
`000004_integridad_agregado_alta` y
`000005_funcion_confirmar_alta_atestada`. La puerta rechaza antes de crear el
contenedor cualquier prefijo numérico duplicado.

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

El consumidor VEC-AD-3 distingue internamente consumo nuevo y replay. La rama
de replay no ejecuta preparación, alias, reserva ni inserciones de efecto. Solo
puede devolver éxito si `reconciliar_agregado_alta_v1` acredita el agregado
durable completo; una capacidad consumida nunca se reutiliza para completar
tarde una actuación, auditoría, evento o marcador ausentes.

## Identidad y prueba portable del agregado

`confirmacion_agregado_alta` se crea dentro de la misma transacción que el
efecto. Su referencia determinista y `agregado_huella_sha256` comprometen:

- identidad y revisión exacta de reserva confirmada;
- expediente, número, recibo, decisión, efecto y consumo;
- versión 1, canon y huella del alta;
- actuación inicial y huella del canon completo de actuación;
- auditoría, decisión, consumo, secuencia, anterior y huella;
- outbox, payload, secuencia, anterior y huella;
- instante confirmado y huella del recibo.

FK diferibles en ambos sentidos ligan el marcador con reserva, expediente,
versión, actuación, auditoría y outbox en el único `COMMIT`. La prueba no usa
identificadores de transacción, LSN ni WAL y es portable a otro adaptador que
demuestre las mismas invariantes.

La frontera VEC-AD-3 permanece intercambiable: contratación invoca su función
consumidora y conserva el recibo autenticado devuelto, pero no lee, escribe ni
referencia por FK las tablas internas de autorización. Dominio y aplicación
no incorporan dependencias PostgreSQL ni de transporte.

La reconciliación compara con `IS DISTINCT FROM` todas las piezas y falla ante
ausencia, divergencia o `NULL`. Recompone canon, payload y huellas. Para
cadenas con efectos posteriores valida el predecesor, el sucesor inmediato y
la cabeza vigente recompuesta, sin suponer que el efecto reintentado sea la
cabeza. La huella del marcador conserva la prueba durable de su eslabón
original y evita un recorrido histórico no acotado en cada replay.

## Límites de confianza

- El productor de la capacidad no se autoaprueba.
- El emisor no verifica su propia firma COSE ni decide su propia raíz.
- PostgreSQL no acepta una decisión por existir en Go o por llegar desde HTTP.
- La autoridad de identidad no procede de cookies, cabeceras libres ni JSON.
- El runtime no posee tablas, DML, CAS nominal, preparación histórica ni el
  consumidor genérico.
- Los errores exteriores son deliberadamente opacos.

## Criterios transversales verificados

- Vocabulario propio del dominio en castellano coherente; `outbox`, HMAC,
  COSE, SQLSTATE y replay se limitan a términos técnicos del protocolo.
- No se añade texto funcional de interfaz: los mensajes SQL son diagnósticos
  internos opacos para el cliente y no sustituyen claves i18n.
- El contrato es neutral para web, escritorio, CLI y MCP; no usa cookies,
  almacenamiento de navegador ni autoridad aportada por el cliente.
- La única integración entre autoridades es el conector SQL mínimo
  autenticado; no hay lectura ni escritura cruzada de tablas y otro adaptador
  puede acreditar las mismas invariantes portables.
- Privilegio mínimo, denegación predeterminada, referencias opacas,
  trazabilidad encadenada y ausencia de secretos o datos personales reales se
  verifican en migraciones, fixtures, recibos, errores y logs.

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
- contadores y ligaduras de las ocho piezas en éxito, rollback y concurrencia;
- replay después de expirar solo para agregado íntegro;
- retirada o deriva aislada de reserva, puntero, expediente, versión,
  actuación, auditoría, outbox y marcador;
- refs, canones, payloads, huellas, eslabones y cabezas divergentes sin
  reparación ni inserción posterior;
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
7. la revisión independiente detectó que el replay omitía actuación, refs de
   reserva y cadenas; se sustituyó la inferencia parcial por señal
   nuevo/replay, marcador común y reconciliación exhaustiva;
8. `go test` ocultaba `stdout`; el runner ahora captura ambos canales y vuelca
   hasta 240 líneas cuando falla, sin mostrar salida en el éxito.
9. el generador Go de capacidad falló de forma intermitente al interpretar su
   entrada V3; dos repeticiones mostraron el diagnóstico completo y la tercera
   superó el ciclo. No se atribuye al contrato SQL ni se silencia como éxito:
   queda como inestabilidad separada que la revisión debe reproducir.

Después de cada corrección se ejecutó de nuevo el runner desde una base
PostgreSQL 18 vacía. Ninguno de estos fallos se oculta ni se interpreta como
un `GO`.
