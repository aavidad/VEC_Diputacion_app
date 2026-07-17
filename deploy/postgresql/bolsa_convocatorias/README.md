# PostgreSQL: gobierno y consulta exacta de convocatorias

Este despliegue crea la primera persistencia real de convocatorias gobernadas.
No es un repositorio en memoria ni una fuente de presentacion. Conserva los
bytes canonicos sellados por el dominio, sus huellas, el flujo asociado, el
consumo unico de la decision y una auditoria encadenada.

La puerta de consulta permanece **cerrada por defecto**. La funcion existe y
se prueba, pero ningun rol de aplicacion recibe `USAGE` del esquema ni
`EXECUTE` mientras no se incorpore el registrador criptografico COSE del PDP.
Una decision V1 y su SHA-256 no prueban por si solas quien la emitio. Conceder
acceso antes de resolver esa procedencia seria una simulacion de seguridad.

## Orden de instalacion

Requiere el nucleo de autorizacion y el vinculo de autenticacion de actor:

1. `deploy/postgresql/autorizacion/roles_up.sql`.
2. `deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql`.
3. `deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`.
4. `roles_up.sql` de este directorio.
5. `migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql`.
6. `migraciones/000001_almacen_convocatorias.up.sql`.
7. `migraciones/000002_consulta_exacta_cerrada.up.sql`.

Las migraciones se aplican con una identidad nominativa miembro del grupo
migrador correspondiente. Las cuentas de aplicacion nunca reciben las
credenciales del propietario o del migrador. La transaccion de consulta debe
ser `SERIALIZABLE`.

La reversion se ejecuta en orden inverso. El `down` del almacen se niega a
destruir historia salvo que la operacion formal establezca:

```text
vec.confirmar_destruccion_bolsa_convocatorias=
DESTRUIR_HISTORIA_BOLSA_CONVOCATORIAS_IRREVERSIBLE
```

Esa confirmacion no debe formar parte de un despliegue normal.

## Contrato SQL preparado

`obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)` recibe:

- `operacion`: objeto cerrado con `esquema`, `convocatoria_id`, `secuencia`,
  `incluir_instancia_flujo`, `accion`, `recurso_ref` y `solicitada_en`;
- `prueba`: objeto cerrado con `esquema_huella`, `decision_ref`,
  `huella_decision_sha256`, `verificada_en` y `principal_ref`;
- bytes canonicos de la decision reforzada V1;
- bytes canonicos del recurso exacto
  `{"ambitos":{},"atributos":{}}`.

El esquema de operacion es
`vec.bolsa.convocatoria.consulta-postgresql.v1`. La accion se deriva de
`incluir_instancia_flujo`; no puede elegirse una combinacion distinta. La
referencia se reconstruye como `convocatoria_id#secuencia`, por lo que no
existe consulta a «la ultima version» ni selector ambiguo.

La funcion devuelve exactamente:

```text
resultado text
version_canonica bytea
huella_version_sha256 text
instancia_flujo_canonica bytea
huella_instancia_flujo_sha256 text
autorizacion_ref text
huella_autorizacion_sha256 text
atestacion_autorizacion_ref text
huella_atestacion_autorizacion_sha256 text
consumo_autorizacion_ref text
auditoria_ref text
huella_auditoria_sha256 text
consultada_en timestamptz
```

Una consulta confirmada produce `resultado=encontrada`. Una version ausente,
un flujo incompatible o una evidencia incompleta abortan la transaccion; no se
devuelve un recibo positivo parcial. Auditar `no_encontrada` requerira ampliar
el puerto de aplicacion con un recibo negativo tipado y verificable. Hasta
entonces no se inserta una falsa auditoria que el dominio no pueda validar.

## Revalidacion y consumo atomico

La frontera de autorizacion vuelve a comprobar dentro de la misma transaccion:

- decision durable, representacion canonica y SHA-256;
- accion, recurso exacto, finalidad y lista exacta de campos;
- garantia minima y observada `alto`;
- metodo distinto de `demo`;
- superficie `interna_corporativa` no privilegiada o
  `administracion_privilegiada` con cuenta privilegiada;
- asignacion, version y control de vigencia del rol;
- concesion RBAC exacta y sin obligaciones pendientes;
- revision, huella y manifiesto completo de politicas ABAC;
- sesion y `ContextoActor` actuales;
- ventana maxima de 30 segundos y reloj autoritativo de PostgreSQL.

Despues bloquea la atestacion activa, lee la version exacta, consume una sola
vez `decision_ref`, bloquea el checkpoint de auditoria, inserta el registro
encadenado y avanza el checkpoint. Cualquier ausencia, replay, carrera o
cambio concurrente aborta todo el efecto.

## Modelo durable

- `version_convocatoria`: historia inmutable, bytes canonicos y SHA-256;
- `instancia_flujo_version`: flujo exacto ligado a esa version;
- `atestacion_autorizacion_version`: evidencia COSE versionada y append-only;
- `atestacion_autorizacion_actual`: puntero gobernado a activa o revocada;
- `uso_decision_consulta`: consumo unico y preimagen del efecto;
- `auditoria`: registros inmutables con huella anterior;
- `auditoria_actual`: checkpoint serializado de la cadena.

Todas las tablas tienen RLS habilitada y forzada. La unica politica positiva
exige que `current_user` sea el propietario `NOLOGIN`. Los roles runtime no
poseen privilegios de tabla, secuencia, tipo, esquema o funcion. Las funciones
definidoras fijan `search_path` y no resuelven objetos controlados por el
llamador.

## Roles

| Rol | Situacion actual |
|---|---|
| `vec_bolsa_convocatorias_propietario` | `NOLOGIN`; posee esquema y funciones. |
| `vec_bolsa_convocatorias_migrador` | Solo puede asumir al propietario durante migraciones. |
| `vec_bolsa_convocatorias_ejecutor_consulta` | Reservado, sin `USAGE` ni `EXECUTE`. |
| `vec_bolsa_convocatorias_proyector_gobierno` | Reservado para los comandos durables futuros; sin acceso. |
| `vec_bolsa_convocatorias_registrador_atestacion` | Reservado para el verificador COSE; sin acceso. |

## Condicion para abrir la consulta

Una migracion posterior, independiente y revisable debe:

1. registrar una atestacion solo desde una capacidad efimera emitida por el
   verificador COSE aislado;
2. comprobar suite, audiencia, clave versionada, ventanas, rotacion,
   revocacion y configuracion de confianza;
3. ligar decision completa, efecto `consulta-version-exacta` y despliegue;
4. evitar que el portal, el ejecutor o el registrador fabriquen filas;
5. probar replay, clave revocada, firma alterada, carrera de revocacion y ACL;
6. conceder, solo al final, `USAGE` del esquema y `EXECUTE` de esta unica
   funcion al ejecutor de consulta.

No se debe insertar atestaciones mediante fixtures, triggers permisivos ni
credenciales compartidas en un entorno productivo.

## Prueba reproducible

```bash
./deploy/postgresql/bolsa_convocatorias/probar_integracion.sh
```

La prueba usa PostgreSQL 18.4 fijado por digest, instala las dependencias,
compila las migraciones, crea identidades LOGIN distintas y demuestra que:

- ejecutor, proyector y registrador no leen ni escriben tablas;
- el ejecutor no puede invocar la funcion cerrada;
- `PUBLIC` carece de acceso;
- todas las tablas tienen RLS forzada;
- una huella falsa se rechaza;
- una version confirmada no se puede mutar;
- el ciclo ascendente y descendente es reproducible sin datos.

El script solo contiene datos sinteticos efimeros dentro de una transaccion
revertida; no crea una fuente de datos alternativa para la aplicacion.
