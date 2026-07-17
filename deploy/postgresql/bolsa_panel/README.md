# PostgreSQL: panel interno real de bolsas

Este despliegue respalda `ConsultaPanelInternoPostgreSQL` con una proyección
operativa durable, versionada y sin datos personales. La lectura se realiza en
una única transacción `SERIALIZABLE READ WRITE`: revalida la decisión V2 y su
motivo, exige el alcance exacto, comprueba la atestación activa, fija la
revisión leída, consume la decisión y confirma una auditoría encadenada.

No es una fuente de demostración. El único origen admitido es una proyección
publicada por el conector interno autorizado y la respuesta fija siempre
`demostracion=false`.

## Estado de apertura

La función `vec_bolsa_panel.consultar_panel_interno_v1` está instalada pero
**cerrada por defecto**. Ni el ejecutor de consulta ni `PUBLIC` reciben
`USAGE` del esquema o `EXECUTE` de la función. Una decisión, sus bytes y su
SHA-256 acreditan integridad, pero no demuestran por sí solos qué PDP la
emitió.

La apertura productiva exige una migración posterior que conecte el
verificador COSE aislado, gestione claves y revocaciones y registre
atestaciones mediante una capacidad efímera. Hasta entonces no existe un
camino runtime que pueda fabricar esa autoridad. El rol reservado
`vec_bolsa_panel_registrador_atestacion` tampoco posee privilegios.

## Orden de instalación

Requiere PostgreSQL con `sha256(bytea)` disponible y el núcleo V2 de
autorización ya desplegado:

1. `deploy/postgresql/autorizacion/roles_up.sql`.
2. `deploy/postgresql/autorizacion/roles_v2_up.sql`.
3. `deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql`.
4. `deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql`.
5. `deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql`.
6. `deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql`.
7. `roles_up.sql` de este directorio.
8. `migraciones_autorizacion/000001_revalidacion_panel_v2.up.sql`.
9. `migraciones/000001_proyeccion_panel.up.sql`.
10. `migraciones/000002_publicador_proyeccion.up.sql`.
11. `migraciones/000003_consulta_panel_cerrada.up.sql`.

Las migraciones se ejecutan con identidades nominativas miembros de los
grupos migradores. Las cuentas de aplicación no reciben credenciales del
propietario ni del migrador. La reversión usa el orden inverso y se niega a
eliminar historia durable.

## Proyección de lectura

`publicar_proyeccion_panel_v1(jsonb)` es la única entrada concedida al rol
proyector. Recibe un documento cerrado con:

- esquema `vec.bolsa.panel.proyeccion.v1`;
- selector exacto de organización o unidad de gestión;
- revisión opaca `rev_…` y fecha de actualización UTC;
- los doce indicadores agregados del contrato de dominio;
- hasta 40 resúmenes de convocatoria;
- hasta 80 actuaciones pendientes.

Los resúmenes solo admiten referencias opacas, claves de catálogo, fechas y
contadores. No existen campos para nombre, documento identificativo, correo,
teléfono, dirección o candidatos. Las claves adicionales se rechazan. Una
revisión es inmutable; republicar exactamente el mismo documento es
idempotente y reutilizar la referencia con otro contenido falla.

El publicador mantiene un puntero monotónico por alcance. La consulta fija ese
puntero y sus filas hijas dentro de la misma instantánea serializable. Los
módulos dueños de convocatorias, bolsas, llamamientos, documentos e
incidencias alimentarán esta proyección mediante el puerto interno; no se
duplican sus modelos de escritura.

## Contrato de consulta

La firma SQL coincide con el adaptador Go:

```text
consultar_panel_interno_v1(
  operacion jsonb,
  prueba jsonb,
  decision_canonica bytea,
  motivo_canonico bytea,
  correlacion_ref text
) RETURNS TABLE (panel_canonico bytea)
```

`operacion` usa el esquema
`vec.bolsa.panel.interno.consulta-postgresql.v1`. La función reconstruye de
forma positiva:

- clase, organización y unidad opcional;
- referencia `panel:<organización>[:<unidad>]`;
- acción `bolsa.panel_interno.consultar`;
- contexto canónico y su SHA-256, incluidos identificador, versión, huella y
  entrada del motivo catalogado.

La frontera de autorización vuelve a comprobar en la transacción:

- decisión V2 registrada y bytes exactos;
- motivo canónico, publicado, vigente y no retirado;
- acción, módulo, tipo, recurso, finalidad, correlación y contexto exactos;
- único campo permitido `panel_agregado_sin_datos_personales` y cero
  obligaciones pendientes;
- garantía mínima y observada `alto`, método distinto de `demo` y superficie
  interna corporativa o administración privilegiada coherente;
- asignación actual, versión y vigencia del rol, concesión RBAC exacta;
- revisión y manifiesto ABAC actual;
- sesión y `ContextoActor` actuales;
- ventanas máximas de 30 segundos con reloj autoritativo de PostgreSQL.

Después exige una atestación activa, lee la proyección, bloquea el checkpoint
de auditoría, inserta el eslabón y el consumo y devuelve el panel. Cualquier
fallo revierte el conjunto completo.

## Recibo, replay y auditoría

Cada respuesta contiene referencias técnicas `lec_…` y `aud_…`, secuencia,
decisión, huella, correlación y fecha confirmada. El panel canónico queda
guardado con su huella para resolver un reintento exacto sin crear un segundo
efecto. La repetición solo se acepta mientras la decisión, sesión, motivo y
atestación continúan vigentes; cambiar operación, alcance, motivo o
correlación con la misma decisión se rechaza.

La auditoría es append-only. Cada huella se calcula sobre la huella anterior
decodificada y el registro canónico actual. El checkpoint se bloquea y avanza
una sola posición, por lo que dos transacciones no pueden confirmar el mismo
eslabón. Esta cadena aporta integridad y detección de alteraciones; no se
presenta como firma electrónica.

## RLS y privilegios

Las nueve tablas tienen RLS habilitada y forzada. La única política positiva
exige el propietario exacto `NOLOGIN`. Ningún rol runtime posee privilegios de
tabla, secuencia o tipo.

| Rol | Privilegio actual |
|---|---|
| `vec_bolsa_panel_propietario` | Propietario `NOLOGIN`; usado por funciones definidoras. |
| `vec_bolsa_panel_migrador` | Puede asumir el propietario solo para migraciones. |
| `vec_bolsa_panel_proyector` | `USAGE` y `EXECUTE` únicamente del publicador cerrado. |
| `vec_bolsa_panel_ejecutor_consulta` | Reservado; sin acceso hasta autoridad COSE productiva. |
| `vec_bolsa_panel_registrador_atestacion` | Reservado; sin acceso hasta el verificador aislado. |

Las funciones definidoras fijan `search_path=pg_catalog, pg_temp` y UTC. No
resuelven objetos controlados por el llamador.

## Prueba reproducible

```bash
./deploy/postgresql/bolsa_panel/probar_integracion.sh
```

La prueba usa PostgreSQL 18.4 fijado por digest, instala todas las
dependencias, verifica RLS y ACL con identidades LOGIN distintas, publica una
proyección real a través del único puerto y recorre ascenso y descenso.

Para probar composición, recibo, idempotencia y cadena sin fingir una
autoridad productiva, una prueba SQL sustituye temporalmente la frontera de
autorización dentro de una transacción que termina en `ROLLBACK` e inserta una
atestación sintética solo como propietario. El ejecutor runtime permanece sin
`EXECUTE` en todo momento; esa prueba no valida firmas, claves ni procedencia
del PDP.
