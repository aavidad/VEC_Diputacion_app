# CRN11: borrador de recuperación del recibo propio de un olvido

Estado: contrato para revisión de Dirección. No es una migración ni una
capacidad activada. Base: `origin/main` en
`460e120c2ec9c2d4953d1e98bdba011403fe17e2`.

El número CRN11 está reservado fuera de Git por Dirección. El 1 de octubre,
a las 04:00 CEST, Claude autorizó reservar el consumidor nominal adicional
para recuperar, decidir, resolver y aplicar. Dirección ha reservado AD138;
D custodia su orden después del paso 3 de M y debe identificar el enclave
sintético F1 admitido. La autorización permite preparar el corte, pero no
define las fachadas, sus dimensiones ni la postimagen del núcleo.

Este directorio no entra en las listas SQL ni en el arranque.
No contiene un archivo `.sql`, una
fachada V3, permisos de ejecución, ruta HTTP o cambio en el núcleo. El contrato
V3 de esta lectura sigue pendiente, después del paso 3M del núcleo. Hasta que
esa dependencia se entregue y revise, la operación debe seguir devolviendo
dependencia no disponible.

## Resultado y límites

Recuperar el recibo original de la solicitud de olvido de la persona empleada
resuelta por el servidor. Solo admite `paso=solicitud`, versión del recibo `1`
y estado histórico `pendiente_responsable`. Devuelve las referencias,
versión y fecha originales aunque existan actuaciones posteriores.

Cada recuperación requiere una concesión de lectura nueva y deja evidencia
de acceso. No crea solicitudes, actuaciones, recibos ni eventos de negocio;
no modifica fichajes ni saldo. No consulta el recibo de jefatura, RRHH o
aplicación. La recuperación no dice cuál es el estado actual del expediente.

Se reutilizan `ports.ClaveRecuperacionCorreccion`, `ports.ReciboCorreccion`
y `ServicioCorrecciones.RecuperarRecibo`. El adaptador actual
`internal/modules/cronos/adapters/postgres/empleado_solicitudes.go` deja
`RecuperarRecibo` cerrado. Su proveedor genérico por paso no basta para esta
lectura: el adaptador y la composición deberán exigir material V3 nominal de
recuperación propia, separado del permiso para solicitar el olvido.

## Entrada y vínculo de autorización propuestos

El navegador podrá aportar `solicitud_ref`, `clave_operacion` y
`paso=solicitud`. Actor, perfil y empleado proceden de identidad registrada,
ContextoActor y proyección de Personal, con exactamente un empleado vigente.
El servidor rechaza otros pasos antes de pedir autorización. El empleado no
se obtiene consultando la solicitud: conocer su referencia no concede acceso.

La clave es la de la escritura original. No se sustituye por la referencia
de solicitud ni se genera una clave de recuperación. El servidor deriva
`solicitud_ref = 'correccion:cronos:' + clave_operacion` y coteja la referencia
aportada antes de pedir V3. Aunque ambas identifiquen el mismo agregado, los
dos campos siguen siendo obligatorios en `ClaveRecuperacionCorreccion`.

El material canónico propuesto es un objeto JSON UTF-8 de hasta 4096 bytes,
con exactamente siete campos de cadena, sin claves repetidas, nulos ni
campos adicionales. El servidor lo serializa con claves ordenadas y sin
espacios. SQL valida la estructura y calcula SHA-256 sobre los mismos bytes,
sin volver a serializar el material para compararlo con V3.

| Campo | Condición |
| --- | --- |
| `actor_ref` | `^per_[-A-Za-z0-9_]{22,128}$`, actor registrado de la petición |
| `perfil_ref` | `^prf_[-A-Za-z0-9_]{22,128}$`, perfil activo registrado |
| `empleado_ref` | `^emp_[-A-Za-z0-9_]{22,128}$`, único empleado vigente resuelto por el servidor |
| `clave_operacion` | `^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`, clave original de la solicitud |
| `solicitud_ref` | Exactamente `correccion:cronos:` seguido de esa clave |
| `paso` | Exactamente `solicitud` |
| `version_recibo` | Exactamente la cadena `1` |

La clave original identifica el recibo; no es una nueva clave de escritura.
El instante del acceso no forma parte de esa clave. Cada acceso lleva una
decisión nueva, con su propio instante y evidencia.

La función CRONOS propuesta, aún sin crear, es
`vec_cronos_v1.recuperar_recibo_correccion_propia_v1`, con los once argumentos
habituales: material `text`; capacidad, decisión, motivo y contexto `bytea`;
versiones de persona y perfil `numeric`; payload, sobre, evidencia y raíz
`bytea`. Retorna `jsonb`. Las versiones deben ser enteras positivas dentro
de los límites del contrato V3 y coincidir con el contexto acreditado.

Para acordar con la autoridad V3 se propone:

| Vínculo | Valor |
| --- | --- |
| Acción | `cronos.correccion.recibo.consultar`, constante ya existente en aplicación |
| Módulo | `cronos` |
| Tipo de recurso | `correccion_recibo_propio` |
| Finalidad | `recuperar_recibo_correccion_propia` |
| Referencia del recurso | La `solicitud_ref` exacta |
| Ámbito | `empleado_ref` exacto |
| Atributo | `material_sha256`, huella del material anterior |

Tipo y finalidad son una propuesta, no entradas publicadas del catálogo.
La audiencia general `AudienciaCorreccionMarcaje` de Go no basta para dar
por aprobada una audiencia de lectura propia.
La huella de contexto de recurso debe ligar el empleado y el material
completo, incluidos solicitud, clave, paso y versión. Puede reutilizar la
codificación de `huella_contexto_empleado_v1` si el contrato V3 aprobado
mantiene esa misma codificación. Cambiar cualquier campo invalida la decisión.

Falta acordar y entregar audiencia, perfil de consumo, fachada nominal,
catálogo/motivo, emisor, reconocimiento, validación de payload/sobre y
permisos técnicos mínimos. No se propone un nombre de fachada activable.
La firma, evidencia, raíz de confianza, emisor, audiencia, vigencia y
vínculo del payload/sobre deben verificarse con la autoridad existente;
comparar campos JSON de la decisión no sustituye esas verificaciones.

`consumir_propio_v1`, en CRONOS 000010, solo permite las cuatro fachadas de
AD3-70. No se ampliará su lista ni se usará
`registrar_y_consumir_cronos_correccion_v3_atestada` para leer un recibo.
La concesión histórica de solicitud tampoco autoriza la recuperación.

## Preimagen que debe comprobar la futura migración

Las definiciones fuente de esta base son:

| Archivo en `migraciones/` | SHA-256 |
| --- | --- |
| `000001_esquema_marcajes.up.sql` | `f3c7f70308288cc5ea1b1193655f64398b29a28531d0f39b5a54f571164c603e` |
| `000008_movimientos_y_permisos_empleado.up.sql` | `8fa6f754ce18f2e8d9a9d73dbe47bc85c51755fa7639d1dea72e9b768399fc3f` |
| `000010_circuito_y_notificaciones_rrhh.up.sql` | `2aaa5b73b8adb37b836fe054f824992110e5be6e78f805f9fd47cb99255d04aa` |

Estas huellas identifican fuentes. No acreditan instalación ni son la huella
del catálogo PostgreSQL de una base. Antes de convertir el contrato en SQL,
Dirección debe obtener en el clon autorizado la preimagen de columnas,
restricciones, índices, políticas, triggers, ACL, propietario y funciones
afectadas, y exigir coincidencia exacta con la candidata revisada.

`correccion_solicitud` tiene, en este orden, estas columnas NOT NULL:
`solicitud_ref text`, `empleado_ref text`, `clave_operacion text`,
`actor_ref text`, `perfil_ref text`; después `marcaje_original_ref text`
nullable, y después las columnas NOT NULL `hueco_declarado boolean`,
`movimiento text`, `fecha_civil date`, `hora_pretendida text`,
`motivo_codigo text`, `material_sha256 text`, `decision_ref text`,
`auditoria_ref text`, `consumo_huella_sha256 text` y
`solicitada_en timestamptz(6)`. PK de solicitud; clave única; FK opcional al
original; referencia derivada de la clave; las tres referencias de consumo
son únicas. Motivo fijo `olvido_marcaje`. El original puede ser nulo solo
cuando hay hueco declarado.

`correccion_actuacion` tiene, en este orden y todas NOT NULL:
`actuacion_ref text`, `solicitud_ref text`, `empleado_ref text`,
`version integer`, `paso text`, `estado text`, `recibo_ref text` y
`registrada_en timestamptz(6)`. PK de actuación, FK a solicitud,
`UNIQUE(solicitud_ref,version)` y recibo único. Admite versiones 1–4;
`version=1` si y solo si `paso=solicitud`, y en versión 1 exige
`estado=pendiente_responsable`.

Los pasos existentes son `solicitud`, `decision_responsable`,
`resolucion_rrhh` y `aplicacion`. Los estados existentes son
`pendiente_responsable`, `pendiente_rrhh`, `denegada_responsable`,
`denegada_rrhh`, `pendiente_aplicacion` y `aplicada`. Este corte solo lee
el par de versión 1; no añade transiciones.

Ambas tablas pertenecen a `vec_cronos_v1_propietario`, rol NOLOGIN,
NOSUPERUSER y NOBYPASSRLS. Deben conservar ENABLE y FORCE ROW LEVEL SECURITY,
la política `lectura_propia` para ese propietario con
`empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'')`,
la política de adición propia y el trigger `historia_inmutable` que rechaza
UPDATE/DELETE/TRUNCATE. Sin acceso directo para PUBLIC, ejecutor, migrador ni
auditor. Las versiones 000009/000010 no añaden políticas de jefatura a estas
dos tablas. Rechazar políticas permisivas adicionales o cambios de ACL.

También se comprobarán las firmas y definiciones reales de
`acreditar_empleado_contexto_v1`, `vence_autorizacion_v1`,
`comprobar_decision_cronos_v1`, `huella_contexto_empleado_v1` y
`rechazar_mutacion_historia`, junto con su propietario, configuración y ACL.
La futura fachada V3 es una precondición independiente. Sin ella no se
instala ni se publica esta función.

## Orden transaccional y consulta minimizada

1. Validar material y límites de todos los argumentos antes de parsear los
   sobres. Rechazar un contexto de empleado o resolutor ya fijado en la
   transacción. Resolver el vínculo propio desde el contexto acreditado.
2. Verificar la decisión nominal exacta y consumir una concesión nueva con
   payload/sobre/evidencia/raíz. El consumo, la lectura y su evidencia deben
   poder confirmar o revertir juntos. No llamar a una autoridad remota y
   presentar dos commits independientes como una transacción.
3. Exigir `consumo_nuevo=true`, recurso y huella exactos, y referencias
   canónicas de decisión, auditoría y consumo. Revalidar vínculo y vigencia
   con reloj vivo después de cualquier espera. Fijar el empleado acreditado
   mediante `set_config(..., true)`; un GUC aportado por el cliente no sirve.
4. Leer exactamente una fila con la consulta siguiente. Cero filas significa
   rechazo uniforme; más de una es dependencia inconsistente. El adaptador
   nunca transforma una ausencia en un recibo vacío o una solicitud nueva.
5. Añadir evidencia de lectura en CRONOS, sin reescribir el recibo. Revalidar
   vigencia antes del retorno. Si falla auditoría, consumo o commit, no hay
   éxito. El transporte solo devuelve el resultado después del commit.

Esta consulta es un fragmento para revisar dentro de la futura función
autorizada. No es una consulta destinada a ejecución directa ni concede
permiso. `$1` es el empleado ya acreditado; `$2` la solicitud ligada a V3 y
`$3` la clave original validada. No contiene entrada libre de nombre de tabla,
columna o función.

```sql
SELECT jsonb_build_object(
  'solicitud_ref', s.solicitud_ref,
  'actuacion_ref', a.actuacion_ref,
  'recibo_ref', a.recibo_ref,
  'estado', a.estado,
  'version', a.version,
  'instante_utc', a.registrada_en,
  'replay', true
)
FROM vec_cronos_v1.correccion_solicitud AS s
JOIN vec_cronos_v1.correccion_actuacion AS a
  ON a.solicitud_ref = s.solicitud_ref
 AND a.empleado_ref = s.empleado_ref
WHERE s.empleado_ref = $1
  AND s.empleado_ref = nullif(
    current_setting('vec.cronos.empleado_ref', true), '')
  AND s.solicitud_ref = $2
  AND s.clave_operacion = $3
  AND s.solicitud_ref = 'correccion:cronos:' || $3
  AND a.version = 1
  AND a.paso = 'solicitud'
  AND a.estado = 'pendiente_responsable';
```

La fecha de recibo procede de `a.registrada_en`, en UTC con precisión de
microsegundos. El adaptador debe validar las referencias UUID v4 de actuación
y recibo, el estado y la versión. `replay=true` indica recuperación del
recibo inicial; no reutiliza una decisión V3 anterior.

La evidencia de lectura propuesta es una tabla CRONOS nueva de solo adición,
con decisión como PK y referencias únicas de auditoría y huella de consumo.
Conserva empleado, solicitud, actuación, recibo y versión histórica 1, huella
del material e instante del acceso. No conserva motivo, hora declarada,
documentos ni bytes V3. El fragmento DDL siguiente prepara su estructura;
su aprobación queda pendiente del contrato de consumo.
Tendrá propietario NOLOGIN, FORCE RLS y políticas propias explícitas, trigger
inmutable y ninguna concesión directa a cuentas runtime. No se reutiliza
`solicitud_replay`: esa tabla documenta recuperación de escrituras.

La futura función será SECURITY DEFINER con nombres de objetos cualificados,
`search_path=pg_catalog,pg_temp`, `row_security=on`, `timezone=UTC`,
`lock_timeout=2s` y `statement_timeout=5s`, y pertenecerá al propietario
NOLOGIN/NOBYPASSRLS. No precisa bloquear ni actualizar la solicitud para leer
su versión inmutable 1. La instalación tendrá su bloqueo de migración y
preimagen. Se revocará EXECUTE de PUBLIC y de todos los roles runtime; la
activación posterior necesita una entrega expresa con consumidor visible.
No se añaden privilegios ni tipos de fila al núcleo en este borrador.

## Fragmentos SQL para revisar

Estos bloques no forman una función ni una migración. No se extraen a las
listas SQL. Falta la fachada AD138 revisada que verifica y consume V3; por eso
no hay una llamada de consumo ni una función ejecutable de recuperación.

La validación siguiente es un fragmento PL/pgSQL previo al consumo. Usa
`bruto json`, `m jsonb`, `cantidad bigint`, `distintas bigint` y
`material_sha256 text` como variables locales. Comprueba claves repetidas
antes de convertir a `jsonb`; esa conversión por sí sola las ocultaría.
`material_propio_v1` no sirve aquí: exige `zona_horaria`, que esta lectura
histórica no necesita.

```sql
IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
  RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE = 'PC001';
END IF;
BEGIN
  bruto := p_material::json;
EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE = 'PC001';
END;
IF json_typeof(bruto) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE = 'PC001';
END IF;
SELECT count(*), count(DISTINCT e.key) INTO cantidad, distintas
  FROM json_each(bruto) AS e;
IF cantidad <> 7 OR distintas <> 7 THEN
  RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE = 'PC001';
END IF;
BEGIN
  m := p_material::jsonb;
EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE = 'PC001';
END;
IF m - ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion',
             'solicitud_ref','paso','version_recibo'] <> '{}'::jsonb
   OR EXISTS (SELECT 1 FROM jsonb_each(m) AS e
              WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
   OR m->>'actor_ref' !~ '^per_[-A-Za-z0-9_]{22,128}$'
   OR m->>'perfil_ref' !~ '^prf_[-A-Za-z0-9_]{22,128}$'
   OR m->>'empleado_ref' !~ '^emp_[-A-Za-z0-9_]{22,128}$'
   OR m->>'clave_operacion' !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
   OR m->>'solicitud_ref' IS DISTINCT FROM
      'correccion:cronos:' || (m->>'clave_operacion')
   OR m->>'paso' IS DISTINCT FROM 'solicitud'
   OR m->>'version_recibo' IS DISTINCT FROM '1' THEN
  RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE = 'PC001';
END IF;
material_sha256 := encode(sha256(convert_to(p_material, 'UTF8')), 'hex');
```

La tabla propuesta conserva el vínculo al consumo central y al recibo leído.
No reemplaza la auditoría central ni crea una nueva autoridad. Las referencias
de decisión, auditoría y consumo vienen exclusivamente de AD138, una vez
validado su contrato. El INSERT usará la única fila histórica recuperada,
el material validado y el instante vivo del acceso; ninguna referencia se
genera para fingir un consumo. No se usa `ON CONFLICT DO NOTHING`.

```sql
CREATE TABLE vec_cronos_v1.correccion_recibo_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
  actuacion_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_actuacion,
  recibo_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_actuacion(recibo_ref),
  version_recibo integer NOT NULL CHECK (version_recibo = 1),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE
    CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
  consultada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_cronos_v1.correccion_recibo_acceso
  OWNER TO vec_cronos_v1_propietario;
ALTER TABLE vec_cronos_v1.correccion_recibo_acceso ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_recibo_acceso FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_propia ON vec_cronos_v1.correccion_recibo_acceso
  FOR SELECT TO vec_cronos_v1_propietario
  USING (empleado_ref = nullif(current_setting('vec.cronos.empleado_ref', true), ''));
CREATE POLICY adicion_propia ON vec_cronos_v1.correccion_recibo_acceso
  FOR INSERT TO vec_cronos_v1_propietario
  WITH CHECK (empleado_ref = nullif(current_setting('vec.cronos.empleado_ref', true), ''));
CREATE TRIGGER historia_inmutable
  BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_recibo_acceso
  FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_recibo_acceso
  FROM PUBLIC, vec_cronos_v1_ejecutor, vec_cronos_v1_migrador, vec_cronos_v1_auditor;
REVOKE ALL ON TYPE vec_cronos_v1.correccion_recibo_acceso
  FROM PUBLIC, vec_cronos_v1_ejecutor, vec_cronos_v1_migrador, vec_cronos_v1_auditor;
```

Las tres FK no garantizan por sí solas que solicitud, actuación, recibo y
empleado correspondan al mismo hecho. Lo exige la consulta conjunta anterior
y el INSERT de esa fila dentro de la función autorizada. La candidata final
comprobará también propietario y ACL del tipo de fila nuevo, sin modificar
las restricciones ni los tipos de las tablas instaladas.

## Corte posterior de escritura

CRN11 sigue siendo lectura propia. Decidir, resolver y aplicar requieren otra
migración Cronos porque añaden historia de negocio y, al aplicar, un asiento
compensatorio. Se propone CRN12, pendiente de reserva por Dirección; este
borrador no la reserva ni contiene su SQL. AD138 puede reunir fachadas
separadas si D acuerda sus contratos; compartir número no comparte permisos.

Ese corte recibirá la solicitud, una clave de escritura nueva por paso y la
versión esperada (1, 2 o 3), con el resultado del responsable o RRHH cuando
corresponda. Debe resolver al empleado gobernado por autoridad vigente, sin
inferirlo de la identidad del decisor ni de una referencia enviada. Después
de cualquier espera revalidará competencia, sesión y V3. Una repetición
exacta recuperará su propio recibo histórico con autorización actual; cambiar
material o versión con esa clave será conflicto. Historia, recibo, consumo,
auditoría y outbox confirmarán juntos. La aplicación añadirá compensación y
afectará la proyección de jornada sin borrar ni cambiar el marcaje original.
Sin puente durable si hay bases separadas, continuará cerrada.

## Rechazos y aceptación pendiente

Material inválido: rechazo nominal de entrada. Paso distinto de solicitud:
rechazo sin cambiar estado. Identidad ajena, solicitud inexistente o clave
incompatible: respuesta uniforme sin recibo ni confirmación de existencia.
Decisión de otra solicitud, ámbito, acción, finalidad o versión: rechazo;
el permiso de solicitar no sirve. Error técnico, concesión vencida, firma
inválida, fallo de autoridad o de auditoría: dependencia no disponible o
denegación nominal según el contrato real; nunca éxito.

La frontera debe auditar denegaciones con el mecanismo existente y datos
mínimos. El rollback de la recuperación no acredita por sí solo que una
denegación quede registrada; ese registro corresponde a la frontera y debe
comprobarse en el recorrido posterior. No añadir un endpoint a su lista en
este corte.

Cuando estén entregados los contratos pendientes, ensayar con datos sintéticos:

- Recuperación propia: los siete campos históricos del recibo coinciden con
  la solicitud original y `replay=true`; nueva evidencia de acceso.
- Cambio de empleado, solicitud, clave, paso, versión o bytes de material
  usando el mismo sobre: rechazo sin datos de otra persona.
- Concesión histórica de escritura o contexto de empleado ya fijado: rechazo.
- Revocación/caducidad durante una espera y fallo de consumo/auditoría/commit:
  no se confirma la recuperación; no quedan efectos parciales.
- Dos accesos simultáneos con decisiones nuevas: mismo recibo, dos evidencias,
  sin otra solicitud, actuación, recibo ni outbox. Mismo sobre consumido dos
  veces: rechazo; no eludirlo con `ON CONFLICT DO NOTHING`.
- Actuaciones posteriores: sigue devolviendo la actuación inicial v1. Paso
  responsable/RRHH/aplicación continúa fuera de alcance.
- Reinicio de aplicación y PostgreSQL: idénticos recibo y fecha originales,
  con autorización vigente nueva; ninguna escritura de negocio duplicada.
- ACL/RLS: PUBLIC, ejecutor y cuentas runtime no pueden ejecutar la futura
  función ni leer tablas; el propietario no puede saltar FORCE RLS.

No se ha ejecutado SQL, ensayo en clon, prueba Go, recorrido ni puerta global.
Este corte documental requiere revisión del contrato. La futura candidata
ejecutable necesitará dos revisiones independientes SQL/identidad sobre su
hash exacto, ensayo por Dirección en el clon autorizado y recorrido real.
C5 completo sigue pendiente de decisión de jefatura, resolución RRHH,
aplicación compensatoria y sus autorizaciones nominales; CRN11 solo prepara
la recuperación del recibo inicial propio.
