# Cargos de quien firma — AUT53

Para firmar un documento de Contratación temporal, AUT32/AUT35 exigen que la
persona tenga una asignación activa del perfil cuyo `rol_id` es el del paso del
plan nominal de firma (por ejemplo `ct_direccion_rrhh`). Hasta ahora nadie
publicaba esas versiones de rol: en desarrollo las escribe el bootstrap.

AUT53 añade una operación técnica que, con un plan aprobado, publica la versión
de rol de cada cargo, su control de vigencia y su registro como perfil
asignable. Después, la persona administradora los asigna desde la pantalla con
el lote ordinario (AD190/AUT44). Instalar AUT53 no publica ningún rol, no
concede permisos y no crea asignaciones.

## Qué publica cada cargo

Una versión de rol ordinaria con estas concesiones, todas de
`contratacion_temporal`, garantía `alto`:

| Concesión | Acción | Tipo de recurso | Finalidad | Campos | Obligaciones |
| --- | --- | --- | --- | --- | --- |
| Competencial (una por paso, 1 a 8) | la `accion_competencial` del plan, siempre `contratacion_temporal.documento.…` | la del plan, siempre `documento_…contratacion_temporal` | la del plan | ninguno | ninguna |
| `firmar_vec` (opcional) | `contratacion_temporal.documento.firma_vec.registrar` | `firma_vec_documento_contratacion_temporal` | `gestionar_contratacion_temporal` | ninguno | ninguna |
| `consultar_r5` (opcional) | `contratacion_temporal.documento.firmas_r5_v2.consultar` | `expediente_contratacion_temporal` | `gestionar_contratacion_temporal` | los 44 de `CamposConsultaFirmasR5V2` | ninguna |

La competencial es la que lee AUT32: misma acción, módulo, tipo y finalidad que
el paso, sin campos ni obligaciones. Las dos opcionales son exactamente las que
comprueba AD162 al consumir la decisión V3 de firmar con certificado VEC y de
consultar las firmas R5 V2. La firma externa (Portafirmas) no entra: AD162 la
reserva a otro rol. Una competencia sobre un tipo que consuma alguna fachada V3
(`firma_vec_…`, `firma_externa_…`, `expediente_…`) se rechaza.

El documento del rol se construye en SQL a partir del plan; su huella debe
coincidir con la que trae el plan, calculada antes en Go con
`VersionRol.HuellaSHA256()`. Fecha de publicación y publicador salen del plan
(`preparado_en` y `operacion:cargos_firma:<operacion_ref>`), para que quien
aprueba vea la huella definitiva.

## Aviso: ámbitos de las decisiones V3 de firma

El lote asigna siempre organización y unidad, y AUT32 necesita las dos. Pero el
recurso de las decisiones V3 de `firmar_vec` y `consultar_r5`
(`preparar_firma_multiple_capacidad_r5.go`, AD162) sólo lleva
`organizacion_ref`, y el PDP exige que la asignación tenga exactamente las
dimensiones del recurso. Con el código actual, un cargo asignado por el lote
acredita la competencia, pero no obtiene esas dos decisiones V3. Es el mismo
fallo que se encontró en el gobierno del plan (ver `docs/plan_modulos/firmas.md`).

Por eso las dos concesiones V2 son opcionales en el plan. Queda para dirección
elegir entre añadir `unidad_ref` al recurso de esas decisiones (Go y una
sucesora de AD162, como AD201 hizo con el gobierno) o emitirlas desde otro
perfil del operador. Las versiones de rol no cambian si se elige lo primero.

## Quién asigna

La regla de cada cargo va en el plan (`regla_asignacion`) y se traduce en
`regla_asignacion_cargo_firma_v1` a la audiencia que lo podrá asignar. Hoy sólo
existe `lote_ordinario` (el administrador de Aplicación, con el lote). La
pregunta 143 de `dudas.md` sigue abierta: si RRHH decide doble control, se añade
otra regla cuando exista su circuito. Las filas de reglas no se cambian ni se
borran.

## Plan

Texto canónico de `jsonb` (sin claves repetidas), de 1 a 16 cargos sin repetir
`rol_id`, que caduca como mucho un día después de prepararse:

```json
{"esquema":"vec.admin.cargos-firma.plan.v1","operacion_ref":"rpa_cf_…","preparado_en":"…Z","caduca_en":"…Z",
 "cargos":[{"rol_id","version","nombre","version_anterior_sha256","operaciones_v2","competencias",
            "version_rol_sha256","regla_asignacion","organizacion_ref","vigente_desde","vigente_hasta",
            "duracion_propuesta_segundos"}]}
```

- `version` 1 exige que el rol no exista; la N exige que la última publicada sea
  N-1 y que su huella sea `version_anterior_sha256` (vacía en la 1).
  Además, la N-1 tiene que haberla publicado AUT53: debe aparecer, con esa
  misma huella, en el recibo de una operación anterior de este circuito. Así
  no se saca la versión siguiente de un rol ajeno, ni de Dietas ni de un rol
  de RRHH de Contratación temporal que no es un cargo.
- `operaciones_v2`: subconjunto ordenado de `["consultar_r5","firmar_vec"]`.
- `competencias`: de 1 a 8 objetos `{accion, tipo_recurso, finalidad}`.
- El perfil asignable lleva `ambitos_fijos` = la organización del cargo y
  `unidad_requerida` verdadero; la unidad la pone cada asignación.

Además de lo anterior se aplica la defensa de AUT49 sobre cada versión
publicada (`validar_perfil_asignable_admin_v1`): nada de administración,
Sistemas, roles sensibles o fijos, Intervención, fiscalización, aspirantes ni
usuarios externos. Si un cargo falla, no se publica ninguno.

## Procedimiento

1. **Plan.** Se genera desde un fichero de cargos, con las huellas calculadas
   en Go. La CLI `vec-publicar-cargos-firma` llega en la PR siguiente.
2. **Aprobación.** Alberto aprueba la huella SHA256 de los bytes exactos del
   plan.
3. **DBA.** Crea un LOGIN nuevo, miembro único de
   `vec_admin_cargos_firma_ejecutor` (`WITH INHERIT TRUE, SET FALSE, ADMIN FALSE`),
   sin otros permisos, e inserta la fila de
   `vec_autorizacion.config_cargos_firma_admin_v1` con la huella del plan, la
   referencia y la huella de la aprobación, entorno `desarrollo` y una ventana
   corta (como mucho un día).
4. **Aplicar.** Con ese LOGIN, en una transacción `SERIALIZABLE` y UTC:
   `SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(plan, huella)`.
5. **Replay.** Repetir el paso devuelve el mismo recibo con `replay: true` y
   añade sólo un intento auditado. Si alguna versión se retiró después por su
   control, el replay se deniega.
6. **Cierre.** `ALTER ROLE <login> NOLOGIN`.

## Auditoría

Se reutiliza la familia de AD196 en la auditoría común
(`perfiles_asignables_admin` y su intento), sin tocar el núcleo ni el CHECK de
`autorizacion_atestada_v3`. Para saber qué registros comunes son de AUT53 y no
de AUT49, cada uno se anota en `operacion_cargos_firma_admin_v1` por
`auditoria_ref`, en la misma transacción (igual que AD198 con AD188). El plan
exacto y el recibo quedan en `registro_cargos_firma_admin_v1`. Todas las tablas
son de sólo adición y sólo las lee el propietario de Autorización.

## Instalación

`deploy/principal/lista_sql_claude_aut53_cargos_firma_20261005.txt`, una vez y
sin DOWN, después de AUT49 y AD196. La migración comprueba que
`validar_perfil_asignable_admin_v1` es exactamente la de AUT49 (huella del
cuerpo `4ca98d16…`).

## Ensayo del 5 de octubre de 2026

PostgreSQL 18.4 desechable restaurado de la copia fría posterior a AD197
(con AD190, AUT44, CA35, AUT49 y AD196). AUT53 se instala con salida 0 y una
segunda aplicación se para en `dependencias`. El vector
`pruebas_sql/cargos_firma_000053.sql` (en ROLLBACK) da 31/31: positivo con dos
cargos, versión siguiente de un rol ajeno (de Dietas o un rol CT que no publicó AUT53; este último caso falla con la versión anterior del filtro), replay tras retirar la versión, concesiones exactas, consulta de AUT32, replay con el mismo recibo,
CAS de versión 1 y 2, huella de rol y aprobación divergentes, regla no
configurada, competencias sobre tipos V3 o de administración, Intervención por
`rol_id` y por nombre, Sistemas en el plan sin efecto parcial, plan caducado o
con claves repetidas, LOGIN sin configuración, LOGIN de AUT49 sin EXECUTE,
LOGIN con permisos de más, ACL de las funciones internas, inmutabilidad y
cadena común enlazada.

Medido de nuevo el 06/10 sobre main, con AD208 y AD207 (sellado diferido de la
cadena), en un PostgreSQL 18.4 desechable con la copia fría posterior a AD197
más AD198, AD200 a AD202, AUT52, CC9, CT179, CT180, AD208, AD207 y CT183. AUT53
se instala con salida 0. El vector da 31/31: su comprobación de cadena admite
las dos formas. Antes de AD207 la cadena queda enlazada con la cabeza en el
control; con AD207 cada asiento nuevo lleva el marcador fijo y está en la cola
de sellado. AUT53 no mide ninguna preimagen del núcleo y escribe la auditoría
con `registrar_perfiles_asignables_admin_v1` y su intento, que AD207 reescribe.

## Lo que no hace

- No asigna el cargo a nadie: eso es el lote de Administración.
- No publica el certificado vinculado (AD165) ni el cargo en Personal (AD166);
  AUT32 también los exige.
- No retira versiones: se retiran por el control de vigencia existente.
