# CRN12: lectura conjunta de saldo y permisos propios

Estado: borrador bloqueado para instalación y uso real. La reserva `cronos_v1
000012` ya consta en `RESERVAS_MIGRACIONES.md`; no se reserva otro número.
Este directorio contiene un contrato, un esquema comentado y un inventario de
metadatos de solo lectura. Ningún archivo es una migración `*.up.sql`.

## Fuentes y alcance

Base del borrador: `origin/main` en
`960795f3090212257d8df92791bf740e3e663c8a`.
Requisitos: `ficha_cronos_2026-09-23.md`, C3/C6/C7, y
`seguridad_y_despliegue_cronos.md`, apartados 1, 5, 8–10 y 12–14.
Se aplican E03–E07 y E10–E12 de `ESPECIFICACIONES_AGENTES.md`.
El acuerdo D→E del canal de 01/10, 03:50, exige una sola lectura nominal
`SERIALIZABLE`; FIN 05:08 mantiene pendientes CRN12, gobierno C3 y enclave.

| Fuente existente en el árbol | Qué aporta | Límite para CRN12 |
| --- | --- | --- |
| `migraciones/000001_esquema_marcajes.up.sql` y `000004_programacion_y_libro_saldo.up.sql` | `marcaje_original`, `programacion_jornada`, lector interno de libro | No contienen efectos de permisos ni adscripción a colectivo. |
| `migraciones/000007_saldo_y_fichaje_remoto_empleado.up.sql` | Vínculo propio, vigencia, huella de contexto, lector nominal AD53 y evidencia de lectura | Consume una decisión y fija `vec.cronos.empleado_ref`; no se llama desde CRN12. |
| `migraciones/000008_movimientos_y_permisos_empleado.up.sql` | Catálogo C6, solicitud, estado append-only y `estado_permiso_actual_v1` | C6 no define la regla de efecto horario C3. |
| `migraciones/000009_resolucion_permisos_y_avisos.up.sql` | Resolución, versión resultante y competencia | Motivo, resolutor y justificantes quedan fuera de la proyección. |
| `migraciones/000010_circuito_y_notificaciones_rrhh.up.sql` | Postimagen de lectura de permisos y consumo propio | Rechaza GUC propio/resolutor previo; su lista de consumidores no incluye CRN12. |
| `internal/modules/cronos/adapters/postgres/lectura_v3.go` | Transacción `SERIALIZABLE READ WRITE`, sin devolver bytes antes del COMMIT | Reutilizable después de acordar DTO y nueva consulta nominal; hoy no la implementa. |

Las rutas de migraciones son relativas a `deploy/postgresql/cronos_v1/`.
Las fuentes de código se localizaron primero en codebase-memory y se contrastaron
con el árbol de esta rama. No se ha comprobado qué definiciones tiene instalada
una base. AD53 y AD70 son antecedentes, no autorizaciones sumables.

El candidato C3 `1230661b91684464dca31cf806f5f1a8cddcfb15` contiene
`domain/efecto_permiso_saldo.go`, `ports/ensayo_saldo_permisos.go`,
`application/ensayo_saldo_permisos.go` y `adapters/catalogoefectos/json.go`
bajo `internal/modules/cronos/`. Es un ensayo sintético: exige `Demostracion`
y no acredita una fuente durable. No se copia ni se conecta en esta rama.

## Contrato mínimo que deben acordar D y E

1. D fija acción, audiencia, finalidad, recurso, ámbitos, campos, obligaciones,
   consumidor V3 y su firma/resultado. La propuesta del canal es
   `cronos.saldo_permisos.propio.consultar` /
   `vec_cronos_v1.saldo_permisos_propio.consultar.v1`. No es permiso instalado.
   D entrega número reservado, orden causal y postimagen después de M→AD138.
   AD138/CRN11 corresponden a C5; no implementan esta lectura.
2. El servidor resuelve actor, perfil, único empleado propio y zona. Canoniza
   periodo y política C3 exacta (referencia, versión y SHA256) antes de solicitar
   V3. D debe fijar la forma canónica y ligadura del recurso a esos valores.
   El navegador no aporta empleado, colectivo, concesiones, reglas ni piezas V3.
3. E y el propietario de C3 fijan la fuente durable de publicación, vigencia,
   retirada, aprobación y huella del catálogo; también la adscripción histórica
   a colectivo por día, con autoridad y versión. No existe una tabla C3 acordada
   en este borrador. Un JSON de ejemplo no sustituye esta fuente.
4. E fija el DTO versionado conjunto, límites de hechos/bytes y evidencia propia
   de acceso enlazada al consumo y auditoría centrales. Debe conservar referencia
   y versión de programación, marcajes, solicitud, estado, resolución, regla y
   colectivo; política/version/huella y un indicador explícito de integridad.
   No reutiliza `saldo_acceso` como si registrara el alcance conjunto.
5. D y E acuerdan recuperación tras COMMIT incierto, caducidad y revocación
   concurrente. Cada lectura nueva requiere autorización nueva; un reintento
   no puede devolver bytes basándose en un consumo histórico. `SERIALIZABLE`
   permite un orden lógico anterior a una revocación concurrente: no promete
   conocer toda revocación confirmada después de iniciar su instantánea.

## Secuencia de la futura operación

Una transacción `SERIALIZABLE READ WRITE` es necesaria para consumo y auditoría.
La función nominal será `VOLATILE SECURITY DEFINER`, con propietario sin LOGIN,
sin BYPASSRLS, `search_path=pg_catalog`, `row_security=on` y tiempos acotados.
Rechazará contexto previo de empleado o resolutor. Revalidará vínculo propio,
decisión exacta y vigencia; consumirá una única fachada V3 literal acordada con D;
revalidará tras esperas y fijará una sola vez el contexto local de RLS.

La función leerá las cuatro fuentes en esa misma transacción: programación,
fichajes, concesiones y política/reglas C3. El lector interno sin consumo
`consultar_libro_saldo_interno_v1(text,date,date,text)` es candidato a reutilizar
tras revisar su postimagen; las fachadas nominales AD53/AD70 y
`consumir_propio_v1` quedan fuera de la secuencia. No se vacía el GUC para
sortear sus guardas ni se cambia su lista de fachadas.
Si C5 añade hechos de corrección aplicados, la fuente de fichajes efectivos
debe incorporarlos por su contrato conservando los originales; el lector actual
de `marcaje_original` no se declarará completo para esa postimagen sin revisión.

El periodo usa fechas civiles inclusivas y zona explícita. Las solicitudes
se seleccionan por `s.desde <= hasta AND s.hasta >= desde`, incluso si empiezan
en otro año. Para cada solicitud se obtiene el estado de mayor versión de la
instantánea, y una concesión exige resolución final concordante con esa versión
y estado. No basta una aprobación de responsable en un circuito J-A ni una
resolución histórica seguida de cancelación.

La proyección mínima de concesión contiene referencias de solicitud/resolución,
permiso y catálogo C6, fechas, unidad/horas necesarias, versión del estado y
evidencia acordada de jornada completa y colectivo histórico. No contiene nombre
del permiso, motivo, salud, actividad sindical, adjunto ni identidad del resolutor.
La matriz positiva de campos debe aprobarse también para el empleado propio.

El cálculo permanece en dominio y distingue trabajo de crédito de permiso.
El primer motor solo admite jornada completa acreditada, programación exacta,
regla vigente para permiso/catalogo/colectivo y ausencia acreditada de trabajo,
anomalías y otros permisos solapados. Cero minutos redondeados no demuestra
ausencia de trabajo. Una regla ausente, fuente parcial, varias programaciones,
colectivo ambiguo, límite excedido o solape no se convierte en crédito cero ni
saldo definitivo. El DTO distinguirá dato ausente, conjunto vacío completo y
efecto no disponible; una colección truncada nunca se presenta como completa.

Antes de devolver, se confirma la evidencia conjunta y se revalida la vigencia
con reloj vivo. El adaptador retiene el resultado hasta COMMIT confirmado.
Un fallo de consumo, integridad, auditoría, serialización o confirmación devuelve
un error sin datos. Las denegaciones se registran por el canal de auditoría
autorizado fuera de la transacción abortada, sin hechos personales en logs.

## Preflight y puerta de instalación pendiente

`preflight.borrador.sql` inventaría solo catálogos de PostgreSQL y siempre informa
`bloqueado`. No consulta tablas personales ni llama funciones de Cronos/V3.
No se ha ejecutado; Dirección puede usarlo por el canal de solo lectura del clon
cuando autorice la comprobación. No compara aún una postimagen aprobada y no da GO.

Para convertir el esquema comentado en migración faltan los cinco acuerdos
anteriores. Después se necesitan preimagen/postimagen exactas, huellas de funciones
y ACL, RLS/trigger append-only y restricciones de integridad, ensayo PostgreSQL
real en el clon, dos revisiones sensibles independientes sobre el hash final y
orden aprobado. Las nuevas funciones/tipos revocarán permisos a PUBLIC y a todos
los roles no autorizados; la activación de EXECUTE será un paso expreso posterior.

Casos mínimos del ensayo futuro: empleado ajeno, GUC previo, decisión AD53/AD70,
periodo o huella cambiados, permiso iniciado antes del periodo, cancelación y
aprobación parcial, colectivo/regla ausentes, programación ambigua, trabajo
inferior a un minuto, solapes, noches y cambio horario, fuente truncada,
publicación/revocación concurrentes, caducidad tras espera, error de auditoría,
`40001`, COMMIT incierto y recuperación. Nunca se devuelve una mezcla de versiones.

## Comprobaciones de esta entrega

Revisión estática propia: fuentes, ligadura nominal, contexto RLS, minimización,
solapes y límites revisados. El esquema SQL solo contiene comentarios; el preflight
contiene consultas de metadatos. `git diff --check` se ejecuta antes del commit.
La revisión propia no sustituye las dos revisiones independientes.
No hay ensayo SQL, SQL instalado, contenedores, servicio, HTTP, navegador, PR ni push.
El cierre formal de Cronos no cambia. La activación sigue bloqueada por el contrato
V3, gobierno C3, fuentes históricas y enclave interno.
