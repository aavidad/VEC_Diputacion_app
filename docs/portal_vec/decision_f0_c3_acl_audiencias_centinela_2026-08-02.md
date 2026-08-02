# Decisión F0-C3: ACL, audiencias y centinela

Fecha: 2 de agosto de 2026.

## Finalidad

C3 publica el consumidor privado C2 exclusivamente a la autoridad propietaria
de ContextoActor y acredita la clausura catalogal de F0. No crea otra fachada,
no consume una capacidad y no implementa la retirada final.

Depende de C2 y modifica solo:

- `M/090_acl_audiencias_centinela.sql`;
- `T/090_acl_audiencias_centinela.sql`.

R1, R2a, R2b, T1 y T2 permanecen en cortes posteriores.

## Orden respecto de R0

M090 se instala correctamente con los tres grupos R0 ausentes. No los crea,
altera, elimina, concede ni inspecciona. R0 es un corte posterior y la
[decisión C2](decision_f0_c2_consumidor_nominal_2026-08-02.md) hace que toda
llamada falle cerrada hasta que existan los grupos y el `LOGIN` canónicos.

El arnés focal instala primero F0; después puede crear sustitutos R0 y `LOGIN`
dentro de PostgreSQL efímero para probar llamadas anidadas. Esos sustitutos no
son una precondición productiva ni forman parte de M090.

## Publicación mínima

M090 reacredita en catálogo la firma C2 exacta, propietario
`vec_autorizacion_atestada_v3_propietario`, `SECURITY DEFINER`, `VOLATILE`, no
`STRICT`, `PARALLEL UNSAFE`, no `LEAKPROOF`, `search_path=pg_catalog` y
`lock_timeout=2s`.

M090 no repara deriva. Antes de mutar exige que la ACL efectiva de C2 sea
exclusivamente propietaria. Después exige únicamente al propietario V3 y al
propietario ContextoActor; cualquier otro beneficiario o `grant option` hace
fallar y revierte todo. C3 no barre dinámicamente `LOGIN` ni roles R0.

La ACL exacta previa del esquema contiene al propietario V3 con
`CREATE+USAGE` y a `vec_contratacion_temporal_propietario` con `USAGE`. La ACL
posterior conserva esas dos entradas y añade solo
`vec_contexto_actor_v1_propietario` con `USAGE`. En las tres, el otorgante es
`vec_autorizacion_atestada_v3_propietario` y no hay `grant option`; cualquier
entrada o privilegio distinto deniega antes de dejar efectos.

La única publicación positiva de C3 es:

1. `USAGE` del esquema `vec_autorizacion_atestada_v3` para
   `vec_contexto_actor_v1_propietario`;
2. `EXECUTE` de la firma exacta C2 para ese mismo propietario.

Ambas concesiones tienen como otorgante exacto
`vec_autorizacion_atestada_v3_propietario` y carecen de `grant option`.

No se conceden tablas, secuencias, tipos, DML, otras funciones V3 ni capacidad
de cambiar de rol. R0 nunca obtiene `EXECUTE` directo. La futura fachada
ContextoActor invocará C2 como `SECURITY DEFINER` bajo su propietario y dentro
de la misma transacción exterior que el efecto.

## Audiencias

C3 acredita, pero no modifica, que el `CHECK` creado por B1 para
`clave_capacidad_version.audiencia_consumo` admite exactamente:

```text
vec_contratacion_temporal.confirmar_alta_atestada.v1
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1
vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1
vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1
vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1
vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1
vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1
```

Cualquier ausencia, valor adicional, cambio de nombre o expresión catalogal
distinta hace fallar el corte. No se reconstruye el `CHECK` desde una lista
aportada por configuración.

## Centinela catalogal

M090 crea en `revocacion_raiz` el trigger
`dependencia_f0_fuente_corporativa_v1`, dirigido por `tgfoid` a
`serializar_revocacion_consultas_rrhh_v3()`, como `BEFORE INSERT FOR EACH ROW`,
sin argumentos y con `tgenabled='D'`.

El centinela aporta exactamente una dependencia normal propia desde su fila de
`pg_trigger` a `serializar_revocacion_consultas_rrhh_v3()`. No se exige
unicidad global de las dependencias normales de esa función: los tres triggers
históricos conservan las suyas. C3 tampoco los modifica ni avanza el
checkpoint.

La función del centinela es impedir una retirada parcial con
`DROP FUNCTION ... RESTRICT`. C3 solo demuestra esa propiedad dentro de un
subensayo revertido; la acreditación y retirada completas pertenecen a
R1/R2a/R2b.

## Criterio combinado H0b y T090

El subensayo H0b instala M080/M090 sin R0 y acredita la denegación C2 sin
efectos. La clausura focal posterior con T090 y el fixture R0 canónico debe
acreditar:

- firma, propietario, atributos, configuración y ACL exactos de C2;
- ACL pre/post exacta del esquema, incluidos Contratación y los otorgantes;
- concesión mínima al único propietario ContextoActor, otorgante V3 exacto;
- rollback ante cualquier ACL previa no propietaria o `grant option`;
- R0/`LOGIN` sintéticos canónicos tras instalar F0;
- llamada directa denegada y llamada anidada aceptada para los cuatro cruces;
- denegación sin R0, con R0 cruzado o adicional, como despachador, `PUBLIC`,
  mediante DML o `SET ROLE`;
- siete audiencias exactas sin reescritura del `CHECK`;
- forma, estado, función y dependencia propia exactos del centinela;
- bloqueo por `RESTRICT` en un subensayo revertido, sin residuo.

La fachada sintética usada por T090 se crea y elimina dentro de la prueba. No
se incorpora a M090 ni se presenta como composición productiva.

## Criterio de cierre

C3 solo se integra después de C2, PostgreSQL 18.4 real, pruebas adversariales,
revisión independiente y `P0=P1=P2=0`. Su cierre no autoriza aún R0/M5–M7,
ContextoActor, la raíz HTTP ni producción.
