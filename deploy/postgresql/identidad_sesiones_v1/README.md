# Registro durable de identidad y sesiones V1

Este corte implementa el puerto `httpseguridad.RegistroSesiones` sin crear una
segunda fuente de verdad. Las sesiones y sus controles se escriben directamente
en:

- `vec_autorizacion.sesion_autenticacion_v1`;
- `vec_autorizacion.control_sesion_v1`;
- `vec_autorizacion.control_sesion_actual_v1`.

El esquema `vec_identidad_sesiones_v1` solo aporta lo que esas tablas no
modelaban: provisionamiento seudonimizado de cuentas, versiones de su estado,
aliases para rotación HMAC y consumo de aserciones de un solo uso. No contiene
DNI, nombre, correo, identificadores IdP en claro, aserciones ni claves.

## Estado de montaje

La infraestructura, el adaptador y las pruebas están cerrados, pero este corte
es **NO-GO de producción** hasta disponer de estos conectores autoritativos:

1. verificador real de aserciones protegidas;
2. HSM/KMS que implemente `SeudonimizadorAlta` con HMAC-SHA-256 y separación de
   propósito para aserción, sesión, sujeto, cuenta y cuenta ordinaria;
3. fuente IdP/maestro que autorice el provisionamiento y la rotación de aliases;
4. LOGIN y secretos distintos para cada pool, creados fuera del repositorio.

No existe implementación HMAC software de reserva. Si falta el conector, el
constructor falla cerrado. La aplicación todavía no compone este adaptador en
un proceso productivo.

## Privacidad y ligadura al emisor

`AltaSesionAtomica.EspacioIdentidad` es el emisor HTTPS protegido y ya
cotejado. La composición fija el par exacto:

```text
(EspacioIdentidad, DominioHMACRef idh_...)
```

El adaptador exige ese par antes de invocar el HSM y el resultado lo ecoa. A
PostgreSQL solo llegan `DominioHMACRef`, `kid`, versión y cinco digests de 32
bytes. El URL del emisor y los identificadores fuente no se persisten ni se
envían como parámetros SQL. Esto impide colisiones entre dos IdP que reutilicen
el mismo `CuentaID`.

Los digests de aserción, sesión, sujeto, cuenta y cuenta ordinaria deben ser
todos distintos. Go y PostgreSQL rechazan su igualdad porque revelaría que el
conector no aplicó la separación criptográfica de propósito exigida.

Una rotación de clave añade un alias a la misma `cta_` mediante
`registrar_alias_hmac_cuenta_v1`; nunca crea una identidad nueva por defecto.
La equivalencia entre las coordenadas antiguas y nuevas pertenece al
provisionador autoritativo, porque PostgreSQL no recibe el valor original.

El identificador de sesión IdP admite nuevas aserciones, pero solo una sesión
VEC no vencida por coordenada HMAC. Como el digest cambia al rotar la clave,
el cambio de clave operativa exige una fase de drenaje superior al TTL máximo
de sesión (más de cinco minutos) antes de admitir altas con la nueva versión.
Así no se solapan sesiones del mismo IdP entre épocas que PostgreSQL no puede
correlacionar sin conocer el identificador fuente. La aserción protegida sí
queda bloqueada entre versiones por su huella SHA-256 independiente de la
clave HMAC; rotar la clave nunca permite consumir otra vez la misma aserción.
PostgreSQL no puede imponer por sí solo ese drenaje porque no conoce el
identificador fuente entre épocas. Por tanto, el montaje productivo debe fallar
si el coordinador autoritativo de rotación no acredita la retirada de la época
anterior y el intervalo completo; una instrucción manual no satisface la puerta
de producción.

## Invariantes transaccionales

- Altas y revalidaciones usan transacciones `SERIALIZABLE`, `READ WRITE`, RLS y
  límites locales de bloqueo/ejecución.
- Las referencias de sesión se generan dentro de PostgreSQL con 18 bytes
  CSPRNG (`gen_random_bytes`): 144 bits antes de codificarlas.
- Cada invocación Go genera además una `opr_` CSPRNG independiente. Un `COMMIT`
  ambiguo se consulta exclusivamente por esa operación y coteja todos los
  digests, columnas, huellas e instantes. Nunca reintenta el consumo.
- Una llamada nueva genera otra `opr_`; aunque repita la aserción, la restricción
  única la rechaza y no devuelve la confirmación anterior.
- El TTL de sesión máximo se vuelve a imponer en SQL: cinco minutos.
- La frescura de autenticación se evalúa con intervalo semiabierto después de
  adquirir los bloqueos: 12 horas en superficie externa personal, 15 minutos
  en interna corporativa y 5 minutos en administración privilegiada. Se vuelve
  a comprobar con el reloj actual en cada revalidación durable.
- PostgreSQL repite la garantía mínima aunque Go ya la haya validado: externa
  personal admite `sustancial` o `alto`; interna y administración solo `alto`.
- `consumo_asercion` liga mediante FKs compuestas sesión, autenticación, ambas
  cuentas, control y revisión. No puede ensamblar filas de sesiones distintas.
- La revisión de cada cuenta al crear la sesión queda fijada. Inactivar y luego
  reactivar una cuenta no resucita sesiones anteriores.
- Revalidar bloquea primero el puntero de control y después los punteros de
  cuenta en orden canónico. Revocación y cambios de estado bloquean los mismos
  registros.
- La historia (`cuenta`, aliases, estados, consumos, sesión y controles) es
  append-only. Tanto el `down` de operaciones como el del esquema se niegan
  antes de retirar funciones u objetos si existe historia.
- La exclusión de otra sesión IdP todavía activa usa un índice compuesto por
  coordenada HMAC y `sesion_id_hmac`; no recorre el ledger append-only.
- Los `timestamptz` que devuelve pgx se convierten expresamente a UTC y
  microsegundos antes de construir la confirmación tipada.

## Capacidades PostgreSQL

Todos son grupos `NOLOGIN`, sin superusuario, `BYPASSRLS`, creación de roles ni
acceso directo a tablas:

| Grupo | Capacidad exclusiva |
|---|---|
| `vec_identidad_sesiones_v1_provisionador` | provisionar cuenta y rotar alias HMAC |
| `vec_identidad_sesiones_v1_registrador` | consumir aserción, registrar y reconciliar su propia operación |
| `vec_identidad_sesiones_v1_revalidador` | revalidar sesión/control/cuentas bajo bloqueo |
| `vec_identidad_sesiones_v1_revocador` | versionar estado de cuenta o revocar sesión |

Cada pool de ejecución debe usar un LOGIN diferente y heredar un único grupo.
No se admite `SET ROLE`, reutilizar el pool de registro para revalidar ni dar un
LOGIN al propietario/migrador. En PostgreSQL 18 la concesión se realiza de
forma explícita, por ejemplo:

```sql
GRANT vec_identidad_sesiones_v1_registrador TO vec_identidad_registrador
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
```

El constructor consulta PostgreSQL antes de aceptar los pools: exige
`session_user = current_user`, LOGIN sin privilegios administrativos,
membresía directa con `INHERIT TRUE`, `SET FALSE`, `ADMIN FALSE`, ausencia de
las otras capacidades runtime y usuarios distintos para registrar y
revalidar. Esta comprobación no sustituye la creación de LOGIN y secretos por
Sistemas ni autoriza el montaje productivo.

## Orden obligatorio de despliegue

Prerequisitos: PostgreSQL 18, base VEC dedicada, `pgcrypto` en `public` y el
nucleo de autorización hasta
`000002_vinculo_autenticacion_actor_actual.up.sql`.

Después se aplica exactamente este orden:

1. `roles_up.sql` como DBA;
2. `migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql` con el migrador
   de autorización;
3. `migraciones/000001_registro_base_v1.up.sql`;
4. `migraciones/000002_operaciones_v1.up.sql`.

El paso 2 crea la `UNIQUE` compuesta que necesitan las FKs del paso 3. El runner
demuestra que invertirlos falla y que la transacción no deja objetos parciales.

Para un `down` limpio se usa el orden inverso y al final `roles_down.sql`. Si ya
existen cuentas o consumos, `000002_operaciones_v1.down.sql` se detiene antes
de retirar las APIs y `000001_registro_base_v1.down.sql` mantiene una segunda
barrera. La retención/borrado de historia deberá resolverse mediante expediente
y política de conservación, no con una migración destructiva.

`roles_down.sql` no devuelve a `PUBLIC` la ejecución de
`public.gen_random_bytes` ni `public.digest`. Esa revocación forma parte del
endurecimiento de la base VEC dedicada y se conserva al retirar el componente;
restaurar ACL globales deberá ser una decisión DBA explícita, no un efecto
lateral de una migración de aplicación.

## Pruebas

```bash
go test -race -count=1 ./internal/vec/adapters/httpseguridad/postgres
deploy/postgresql/identidad_sesiones_v1/probar_integracion.sh
```

El runner levanta PostgreSQL 18 efímero y verifica, entre otros puntos:

- orden y ciclo limpio de migraciones, y barrera de `down` con historia;
- RLS `FORCE`, `PUBLIC` sin acceso, índice de exclusión de sesión y ACL de
  funciones/tablas;
- LOGIN separados, membresía única y prohibición efectiva de `SET ROLE`;
- alta y revalidación reales mediante pools pgx distintos;
- normalización UTC real de `timestamptz`;
- carrera de dos consumos de la misma aserción y carrera de dos aserciones para
  la misma sesión IdP, sin temporizadores: exactamente uno confirma;
- replay incluso tras rotar HMAC, reconciliación exacta, TTL, revocación y
  transición de cuentas;
- frescura por superficie y rechazo cuando caduca mientras espera un bloqueo;
- rotación HMAC que conserva la `cta_`;
- cuenta inactiva/reactivada que no recupera una sesión anterior.

Las variables `VEC_POSTGRES_TEST_IDENTIDAD_*_DSN` solo se usan para ejecutar la
prueba Go contra una instalación externa. El runner las crea con credenciales
efímeras y las elimina al terminar.
