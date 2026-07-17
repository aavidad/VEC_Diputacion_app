# Registro y consumo atestado VEC-AD-2

Estado: corte real y ejecutable, pero **NO-GO para producción** hasta cerrar el
broker Go, HSM/KMS, ancla anti-restauración y composición atómica con el efecto
de Bolsa. Este paquete no convierte una firma válida en una autorización.

La puerta hace una sola operación transaccional:

1. valida una capacidad HMAC de cinco segundos emitida por el broker aislado;
2. liga sus 39 campos exactos a decisión, motivo, payload VEC-AD-2, COSE,
   evidencia de verificación, actor, sujeto pseudonimizado, acción, finalidad,
   recurso, contexto, correlación, efecto y catálogo histórico de confianza;
3. llama al registrador nominal V2 existente, que vuelve a comprobar
   asignación, RBAC/ABAC, políticas, sesión, motivo y vigencia actuales;
4. guarda la prueba criptográfica re-verificable y consume una sola vez tanto
   la capacidad/nonce como la decisión;
5. añade una entrada a la cadena de auditoría y devuelve referencias opacas.

Todo ocurre en la transacción del llamador. Si cualquier inserción, FK,
revalidación o efecto posterior falla y la transacción se revierte, también se
revierte el registro nominal y el consumo. El futuro adaptador de cálculo debe
invocar esta puerta y confirmar el resultado oficial en la misma transacción
`SERIALIZABLE`; llamar y confirmar en transacciones distintas es un NO-GO.

## Separación de autoridad y criptografía

El catálogo `vec_confianza_atestacion_v2` conserva la lista positiva pública.
Un LOGIN dedicado, miembro únicamente de su rol lector, obtiene esa instantánea
y el verificador Go valida COSE Ed25519 y el mensaje binario producido por
`SerializarMensajeAtestacionAutorizacionV2`. Un segundo LOGIN, miembro únicamente
de `vec_autorizacion_atestada_v2_emisor_capacidad`, obtiene la clave HMAC de
vida operativa y emite una capacidad ligada a la verificación. Las credenciales
no se comparten y la cuenta consumidora no puede obtener el secreto.

PostgreSQL conserva payload, COSE, SPKI y evidencia exactos, sus SHA-256 y FK
compuestas a la decisión nominal, configuración, raíz y consumo exactos. La
migración compañera instala en el propietario del catálogo un cotejo booleano
estrecho: toma su mismo advisory lock compartido, reconstruye la configuración
completa, exige que siga siendo la actual y coteja huellas, membresía, versión,
suite, audiencia, ventanas y ausencia de revocación de configuración y raíz.
El lock vive hasta el `COMMIT`, por lo que rotaciones y revocaciones quedan
serializadas con el consumo. Un checkpoint mutable en cada gobierno fuerza
`40001` si una transacción `SERIALIZABLE` intenta leer desde un snapshot anterior
a una revocación ya comprometida. Ni consumidor ni emisor pueden invocar esos
cotejos ni el lector estrecho del instante nominal.

La FK prueba procedencia referencial y el cotejo prueba vigencia del catálogo;
ninguno demuestra por sí solo que la firma sea correcta. La capacidad HMAC
acredita el paso por el broker configurado; no es una firma del PDP, no reemplaza
VEC-AD-2 y no concede autoridad por sí sola. La llamada al registrador nominal
dentro de la misma transacción sigue siendo obligatoria.

El sujeto del efecto solo admite
`hmac-sha256:<espacio>:<64-hex-minúsculas>`. La capacidad liga además su valor al
contexto de recurso verificado por el broker. PostgreSQL no intenta reconstruir
el mapa canónico de ámbitos: esa comprobación pertenece al códec Go cerrado y
debe estar cubierta por las pruebas del adaptador productivo.

## Roles y ACL

- `..._propietario`: propietario `NOLOGIN`, aislado;
- `..._migrador`: solo puede asumir al propietario durante migraciones;
- `..._emisor_capacidad`: solo ejecuta `obtener_material_emisor_capacidad()`;
- `..._consumidor`: solo ejecuta la puerta y la reconciliación estrecha.

Runtime no recibe `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`,
`REFERENCES`, secretos, registrador nominal ni lectura del catálogo. Las nueve
tablas usan `ENABLE/FORCE RLS`, política de propietario exacto, historia
append-only y rechazo de `TRUNCATE`. Las funciones son `SECURITY DEFINER` con
`search_path=pg_catalog`; `PUBLIC` no recibe `EXECUTE` ni acceso al esquema.

Este paquete no cambia globalmente los ACL de `pgcrypto`. `roles_up.sql` exige
que el bootstrap de la base dedicada ya haya retirado `EXECUTE` de
`public.hmac(bytea,bytea,text)` a `PUBLIC`. Así no modifica silenciosamente a
otros módulos ni pretende restaurar un ACL previo que desconoce.

## Gestión de la clave de capacidad

La clave HMAC se publica como una versión append-only y se activa mediante un
puntero monotónico. Publicación, rotación y revocación adquieren el advisory lock
`vec_autorizacion_atestada_v2:gobierno_clave:v1`; las lecturas lo adquieren en
modo compartido y bloquean el checkpoint antes de tomar el reloj. Los punteros
futuros se rechazan y el lector elige el de mayor orden que ya sea elegible.
`registrada_en` se vuelve a sellar dentro del lock exclusivo y constituye el
eje de conocimiento; `establecida_en`/`revocada_en` es el eje efectivo. Toda
lectura exige ambos ejes anteriores o iguales a su instante autoritativo. Las
revocaciones no pueden fecharse antes del sello, por lo que una espera de lock
no reescribe retroactivamente la historia de un consumo válido.

Producción debe mover el secreto a HSM/KMS o a un broker con envoltura
institucional: el `bytea`
almacenado en PostgreSQL no demuestra cifrado en reposo, es solo un corte de
integración y mantiene el NO-GO.

La cuenta emisora debe tener exactamente una membresía directa. El lector del
catálogo exige también identidad aislada; por ello el broker usa dos pools de
conexión distintos y nunca una cuenta con ambos roles.

## Instalación y retirada

Orden requerido:

```bash
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/autorizacion_atestada_v2/roles_up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/autorizacion_atestada_v2/migraciones_autorizacion/000001_vinculo_decision_atestada_v2.up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/autorizacion_atestada_v2/migraciones_confianza/000001_cotejo_consumo_atestado_v2.up.sql
psql -X -v ON_ERROR_STOP=1 -f deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.up.sql
```

El down requiere superusuario, `lock_timeout`, inventario exacto sin `CASCADE`
y la confirmación de sesión:

```text
vec.confirmar_destruccion_autorizacion_atestada_v2=DESTRUIR_AUTORIZACION_ATESTADA_V2_IRREVERSIBLE
```

La retirada usa el orden inverso: down del registro (con token), down del cotejo
en confianza, down del vínculo en autorización y finalmente `roles_down.sql`.
El token no sustituye el expediente
formal de retención/destrucción ni autoriza retirar datos reales.

## Barreras NO-GO restantes

- broker Go que verifique realmente COSE/VEC-AD-2 y emita la capacidad exacta;
- HSM/KMS, rotación operacional y separación de credenciales aprobada;
- ancla monotónica externa contra restauración/failover atrasado;
- adaptador PostgreSQL del cálculo que consuma y cree el efecto en un único
  `COMMIT`, con reconciliación tras respuesta ambigua;
- política aprobada de retención, backup cifrado, recuperación y destrucción.

La cabeza hash de auditoría es deliberadamente una fila única bloqueada con
`FOR UPDATE`: garantiza orden total, pero constituye un cuello de botella global
que debe medirse. Fragmentarla exige un diseño formal de múltiples cadenas y
anclaje; hacerlo solo para ganar rendimiento es NO-GO. También es NO-GO aplicar
el efecto de negocio fuera de la misma transacción: un consumo sin efecto, o un
efecto sin consumo, crea un «efecto fantasma» aunque después se intente reparar.

Hasta entonces no se concede `EXECUTE` de esta puerta a la aplicación de Bolsa
en un despliegue real y el arranque productivo debe fallar cerrado.

## Prueba

```bash
deploy/postgresql/autorizacion_atestada_v2/probar_integracion.sh
```

El runner usa PostgreSQL 18 efímero, dependencias reales, LOGIN separados y
casos adversariales de ACL, rol puente, HMAC, nonce, vínculos alterados,
puntero futuro, revocación efectiva durante espera, snapshots obsoletos con
`40001` para clave y confianza, inmutabilidad, down protegido y reinstalación
limpia.
