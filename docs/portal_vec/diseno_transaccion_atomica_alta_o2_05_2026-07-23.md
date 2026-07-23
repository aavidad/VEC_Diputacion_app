# Diseño de la transacción atómica de alta O2-05

Fecha: 23 de julio de 2026.

Estado: diseño previo a implementación. No habilita producción ni cierra
O2-05.

## Resultado perseguido

Una única transacción `SERIALIZABLE` debe:

1. autenticar criptográficamente el origen de la decisión VEC V3;
2. revalidar su contexto de actor, políticas, motivo y vigencia;
3. consumir una sola vez la concesión;
4. resolver los alias HMAC activos y retenidos de la idempotencia;
5. crear o recuperar exactamente una reserva y un expediente;
6. registrar actuación, auditoría y evento de salida;
7. confirmar todo en un solo `COMMIT`.

Ningún recibo, confirmación nominal Go, clave de idempotencia, cookie, cabecera
HTTP ni dato del navegador concede autoridad por sí mismo.

## Componentes existentes que se reutilizan

No se creará un segundo sistema de autorización. El diseño parte de:

- `SolicitudAutorizacionLigadaV3`,
  `DecisionAutorizacionLigadaV3` y
  `ConfirmacionRegistroConcesionAutorizacionLigadaV3`;
- `ResultadoContextoActorRegistradoV2` y
  `VinculoAutenticacionActorV2`;
- la representación canónica y la huella de decisión V3;
- `vec_autorizacion.registrar_decision_contexto_actor_v3`, que ya realiza el
  CAS nominal de contexto de actor y decisión;
- el patrón ejecutable de
  `vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada`;
- la separación de roles emisor/consumidor, capacidad HMAC de cinco segundos,
  verificación COSE, cotejo de confianza, revocación, cadena de auditoría y
  reconciliación de VEC-AD-2;
- la migración de preparación de contratación temporal y su convivencia de
  generaciones HMAC.

La preparación actual seguirá siendo una pieza privada. El runtime no tendrá
permiso para reservar por separado cuando se active O2-05.

## Incompatibilidad que debe resolverse de forma versionada

La puerta atestada existente acepta la representación canónica de una decisión
ligada V2 y llama a
`registrar_decision_solicitud_ligada_v2_si_vigente`. La vertical nueva usa una
decisión V3 ligada a contexto de actor V2. Cambiar silenciosamente el
significado de VEC-AD-2 rompería verificaciones históricas.

Se añadirá una versión de mensaje y capacidad, con:

- esquema distinto y audiencia exacta;
- representación canónica de `DecisionAutorizacionLigadaV3`;
- representación canónica del motivo catalogado;
- evidencia completa y durable de contexto de actor V2;
- referencias y huellas de solicitud, decisión, vínculo, contexto y motivo;
- organización, perfil, acción, finalidad, recurso y correlación;
- revisión de persona, perfil, rol, catálogo de políticas y confianza;
- referencia y huella exactas del efecto de contratación temporal.

La versión V2 actual permanecerá inmutable y verificable. La nueva versión no
admitirá conversión, reconstrucción ni downgrade desde V3.

## Frontera SQL propuesta

El runtime de contratación recibirá únicamente `EXECUTE` sobre una función
cerrada, conceptualmente:

```text
confirmar_alta_atestada_v1(
    decision_v3_canonica,
    motivo_canonico,
    contexto_actor_canonico,
    payload_atestado,
    sobre_cose_sign1,
    evidencia_verificacion,
    raiz_publica_spki,
    capacidad_breve,
    identidades_hmac,
    expediente_canonico,
    huella_efecto
) -> recibo_minimo
```

El nombre definitivo y cada tipo se congelarán en la migración. No se usará
`jsonb` abierto para el expediente ni para las identidades HMAC: tendrán
esquema cerrado, límite de tamaño, número máximo de generaciones y rechazo de
claves desconocidas o duplicadas.

La respuesta se limitará a referencias opacas, número visible, versión,
instante confirmado y huellas de recibo/auditoría/evento. No devolverá
identidad, clave idempotente, material HMAC, políticas ni datos personales.

## Orden obligatorio de comprobación y bloqueo

Para evitar interbloqueos y ventanas de revocación, todas las rutas usarán el
mismo orden:

1. identidad efectiva del rol consumidor y límites de sesión;
2. forma, tamaños, esquema, audiencia, hashes y MAC de la capacidad breve;
3. gobierno de clave de capacidad y su revocación;
4. configuración y raíz de confianza de la atestación;
5. CAS y registro nominal V3 de contexto de actor y decisión;
6. comprobación final de clave, confianza, sesión, persona, perfil, rol,
   políticas y vigencia en el instante autoritativo;
7. consumo único de la decisión para el efecto exacto;
8. resolución de todos los alias HMAC, en orden de generación;
9. reserva y expediente ganadores;
10. actuación, auditoría y outbox;
11. actualización de cadenas o anclas;
12. `COMMIT`.

Las filas se bloquearán siempre en ese orden. Un alias ya ligado a otra
reserva, dos alias que apunten a reservas distintas o una misma decisión
ligada a otro efecto producirán denegación completa.

## Reglas de idempotencia y rotación

- Solo la generación activa se emite en el recurso autorizable V3.
- La colección activa más retenidas se usa para localizar historia.
- Ámbito y huella deben pertenecer a la misma generación.
- Repetir la petición exacta bajo una generación retenida devuelve el mismo
  expediente y recibo e incorpora el alias activo.
- La misma clave lógica con otra huella se rechaza.
- Ningún alias ni huella HMAC se acepta fuera de la colección gobernada.
- La clave idempotente en claro no se persiste, registra ni devuelve.
- La retirada de una generación queda prohibida mientras exista historia que
  dependa exclusivamente de ella.

## Cancelación, reintentos y resultado indeterminado

Antes de intentar `COMMIT`, una cancelación aborta y no deja efectos. Después
de intentar `COMMIT`, un error de transporte es indeterminado y nunca se
traduce automáticamente a cancelación.

El adaptador O2-06 reconciliará usando:

- todos los alias HMAC admitidos;
- huellas de petición de la misma generación;
- organización, actor y perfil revalidados;
- referencia/huella de decisión y correlación V3;
- referencia y huella esperadas del efecto.

Solo se reintentará toda la transacción ante `40001` o `40P01`, con una
transacción nueva y un máximo acotado. No se reintentará un `COMMIT`
indeterminado sin reconciliar.

## Privilegios y topología

Autorización y contratación temporal residirán en la misma instancia y base
PostgreSQL para conservar atomicidad, con propietarios y migradores separados.

El runtime:

- no tendrá DML directo sobre tablas;
- no podrá `SET ROLE`, crear funciones ni cambiar `search_path`;
- no leerá tablas de autorización, auditoría o contratación;
- no ejecutará las funciones internas de preparación, CAS o consumo por
  separado;
- usará una credencial exclusiva de esta vertical.

La función será `SECURITY DEFINER`, con `search_path = pg_catalog`, UTC y
límites de sentencia, bloqueo e inactividad ya armados en la transacción por
el adaptador. La función comprobará que dichos límites existen y no exceden
los máximos aprobados; no confiará en un `SET statement_timeout` local dentro
de la propia función.

## Matriz mínima de pruebas en PostgreSQL real

- éxito y repetición exacta;
- dos y más sesiones simultáneas;
- rotación HMAC v1→v2 y replay histórico;
- mezcla de ámbito/huella entre generaciones;
- alias convergentes y divergentes;
- decisión reutilizada para otro efecto;
- actor, perfil, organización, acción, finalidad, recurso, motivo, flujo,
  contexto, versión o correlación distintos;
- expiración y revocación antes y durante una espera de bloqueo;
- restauración o checkpoint de gobierno obsoleto;
- cancelación antes de `COMMIT`;
- respuesta perdida después de `COMMIT` y reconciliación tras reinicio;
- fallos inyectados en cada escritura sin estado parcial;
- ACL negativas para runtime, migrador, emisor y consumidor;
- `down` protegido, inventario exacto y reinstalación limpia.

Cada prueba deberá comprobar también la ausencia de claves en claro, datos
personales y mensajes internos en errores, trazas y recibos.

## Puertas externas de producción

Aunque el código y las pruebas sean correctos, la producción permanece
cerrada hasta disponer de:

- broker separado que verifique COSE y emita la capacidad exacta;
- HSM/KMS, rotación y custodia de claves aprobados;
- ancla monotónica externa contra restauraciones atrasadas;
- política de copias, recuperación, retención y destrucción;
- conformidad de Sistemas, Seguridad, DPD y responsable funcional de RRHH.

O2-05 solo se marcará terminado tras implementación, revisión independiente,
pruebas PostgreSQL efímeras repetidas y actualización del tablero. Este
documento por sí solo no aumenta el porcentaje.
