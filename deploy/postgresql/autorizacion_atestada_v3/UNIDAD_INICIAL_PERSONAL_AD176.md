# Auditoría común de la unidad inicial de Personal

AD176 añade dos familias a `auditoria_consumo_v3`: `unidad_inicial_personal`
y `intento_unidad_inicial_personal`. Se apoya en el CHECK instalado por AD174,
sin tocar sus funciones, filas, recibos ni bytes de eslabón. No crea otra tabla
de auditoría ni concede perfiles o permisos a personas.

Estado: candidata ensayada por Dirección en el clon PostgreSQL 18.4 conservado
post-H9 y AD174. UP y prueba SQL terminaron con código 0, sin reaplicar SQL.
Se conservaron las provisiones y los intentos anteriores. Quedan pendientes
dos revisiones independientes. No se ha instalado en la principal; el productor
no ha ejecutado PostgreSQL ni DOWN.

## Dependencias y autoridad

Personal33 publica la unidad sintética mediante su autoridad propia, con CAS,
LOGIN técnico gobernado y configuración/aprobación privadas. AD176 registra
el hecho comprobado por esa fachada; no lee las tablas de Personal ni acredita
por sí sola la fuente declarada. La acción es
`inicializar_unidad_sintetica_admin_v1`, el módulo `personal` y la finalidad
`inicializar_unidad_sintetica_admin`. No reutiliza una acción de fuentes CA/IS.

El actor es `operador_login = session_user`; no se inventa una persona, perfil,
contexto o decisión V3. El proceso exacto `postgresql` identifica al motor
observador y el canal es `operacion_tecnica_privada`. La confirmación mantiene
el alcance `sintetico_declarado`; este ejercicio no acredita producción.

Las funciones tienen EXECUTE exclusivamente para el propietario AD3 y
`vec_personal_propietario`. PUBLIC, Autorización, Identidad y los ejecutores
runtime no reciben acceso. Los permisos de las funciones existentes no cambian.

## Contratos privados

```text
registrar_unidad_inicial_personal_v1(jsonb)
registrar_intento_unidad_inicial_personal_v1(jsonb)
→ auditoria_ref, secuencia, huella_sha256, correlacion_ref, registrada_en
```

Ambas pertenecen al esquema `vec_autorizacion_atestada_v3` y exigen una
transacción SERIALIZABLE de escritura en UTC. Personal33 confirma efecto,
recibo, confirmación e intento juntos. En una denegación/error gestionados,
revierte la subtransacción del efecto y confirma únicamente el intento. Un
fallo del append aborta todo. El consumidor no entrega el sobre antes del COMMIT.

La confirmación contiene exactamente 21 cadenas, en este orden:

```text
tipo_registro
evento_ref
operador_login
plan_ref
plan_sha256
preimagen_sha256
configuracion_sha256
aprobacion_ref
alcance_fuente
accion
recurso_ref
resultado
motivo_ref
proceso
canal
finalidad_ref
correlacion_ref
fuente_ref
fuente_sha256
recibo_ref
recibo_sha256
```

`plan_ref` es la operación `pui_…` del plan aprobado, con un máximo de 128 bytes.
`recurso_ref` es `unidad:<UUID canónico>` del nodo real persistido y
`recibo_ref` es `recibo_unidad:<32 hex>`. No hay otra operación UUID inventada.
El UUID del nodo y sus metadatos quedan comprometidos por el plan, la fuente
y el recibo que comprueba Personal33.

`fuente_ref` y `fuente_sha256` ligan la fuente de entrada real aprobada y sus
bytes canónicos. `recibo_sha256` compromete el JSONB del recibo base real del
efecto, antes de añadir su propia huella y las coordenadas de auditoría. Son
huellas distintas y no hay un ciclo donde el eslabón se comprometa a sí mismo.
El resultado es `permitido` y el motivo cerrado `unidad_registrada`.

El intento contiene exactamente 12 cadenas:

```text
tipo_registro
evento_ref
operador_login
solicitud_sha256
accion
recurso_ref
resultado
motivo_ref
proceso
canal
finalidad_ref
correlacion_ref
```

`recurso_ref` es `solicitud_unidad:<32 hex>` y `solicitud_sha256` compromete la
entrada real sin almacenarla. No contiene fuente, recibo, plan, aprobación,
preimagen de efecto, persona, perfil ni V3 ficticios. Cada invocación gestionada
usa un evento propio, incluido un replay del plan.

El catálogo técnico cerrado de datos SQL y el catálogo JSON del verificador
admiten únicamente estos pares:

| Resultado | Motivo |
| --- | --- |
| permitido | unidad_registrada |
| permitido | unidad_replay |
| denegado | unidad_denegada |
| error | unidad_error |

## Cadena y recuperación

La preimagen exacta es `auditoria_tipo_disjunto_v2` posterior a AD174.
La huella SHA256 de `pg_get_constraintdef(oid, false)`, en UTF-8 y sin salto
final, observada por Dirección en el clon PostgreSQL 18.4 es
`c7dc8abc0c0ea178cadb22960976af57a7f711e158a076c7068227d5718efbb8`.
AD176 la comprueba y envuelve literalmente la condición existente. Las
columnas nuevas son nulas para todas las familias anteriores. No predice
columnas ni familias futuras de AD172/173.

Se reutilizan `encuadrar_mac`, la cabeza y el cerrojo de eventos comunes.
La confirmación usa material `vec.auditoria.unidad-inicial-personal.v1`,
eslabón `vec.auditoria.eslabon.unidad-inicial-personal.v1` y referencia
`aud_v3_u_<32 hex>`. El intento usa material
`vec.auditoria.intento-unidad-inicial-personal.v1`, eslabón
`vec.auditoria.eslabon.intento-unidad-inicial-personal.v1` y referencia
`aud_v3_ui_<32 hex>`. Los dominios son protocolos técnicos versionados,
no textos visibles ni una firma jurídica.

Evento y material idénticos recuperan las coordenadas originales. Una
divergencia, o reutilizar un evento entre familias, se rechaza sin avanzar la
cabeza. El replay del append no equivale al replay del plan de negocio: este
último conserva el recibo original y añade otro intento.

## Verificación

La CLI admite el esquema propio
`vec.auditoria.verificacion.unidad-inicial-personal.v1`, con objetos
`unidad_inicial` e `intento_unidad_inicial`. Conserva consumo histórico v1,
AD169, AD171 y las dos familias AD174. Los esquemas anteriores rechazan AD176.
El informe distingue material de unidad y de intento, recalculados; no autentica
el LOGIN, el checkpoint ni la fuente y no declara cubiertas familias futuras.

Los seis vectores independientes (incluido UTF-8) se conservan en
`pruebas_sql/ad176_vectores_cadena.json` y en
`cmd/vec-auditoria-verificar/testdata/unidad_inicial_ad176.json`. La prueba
`pruebas_sql/ad176_unidad_inicial.sql` coteja material/eslabón con PostgreSQL,
ACL, dos confirmaciones sintéticas, los cuatro resultados de intento, replay,
cruces, LOGIN ajeno, datos nulos, acción/proceso falsos, inmutabilidad e historia
completa. Todo hace ROLLBACK en el clon, sin DOWN.

La revisión de seguridad focal del productor sigue frontera de propietario,
material acotado, campos disjuntos, mínima proyección, parametrización y
atomicidad del append. No sustituye las dos revisiones independientes ni el
recorrido real de Personal33. Conexiones fallidas, cancelaciones y fallos que
no alcanzan COMMIT no quedan acreditados por este contrato.

Comprobaciones locales ejecutadas:

- `GOCACHE=$HOME/.cache/go-build go test -p 8 ./internal/vec/auditoria ./cmd/vec-auditoria-verificar`: código 0.
- Los mismos paquetes con `go test -race -p 8` y `go vet -p 8`: código 0.
- `gosec -quiet -fmt text` solo sobre esos dos paquetes: sin hallazgos.
- Semgrep `--metrics=off`, reglas locales `pruebas_sql/ad176_semgrep_local.yml`: seis archivos productivos, tres reglas y sin hallazgos.
- `git diff --check`: código 0. gopls resolvió el modelo compartido de evento AD171 desde el nuevo verificador.

SQL UP ensayada: SHA256
`da00e3cb266d5e634fb8c1cd17775f7e1a2927fbef2df65067c8deba7af24098`.
Prueba SQL ensayada: SHA256
`c78ad97b3155ab65b7c594423833db01843a2deebae1a922fa9edaf721bd2a6d`.
Los logs del ensayo quedan en el estado privado de Dirección, fuera de Git.
La lista causal de esta rama sirve para un frío sin AD174; en un clon que ya
la conserve se aplica únicamente AD176. Nunca se reaplica la dependencia.
