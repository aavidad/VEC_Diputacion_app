# Documentos comunes B5: persistencia

Este módulo conserva metadatos, numeración interna, referencias a originales
inmutables, instantánea de conservación, preparaciones de notificación,
auditoría y outbox. Los bytes originales permanecen en `AlmacenObjetos` y el
adaptador comprueba su recibo antes de confirmar el documento. El número
`VEC-AAAA-N` no es un asiento de registro general.

## Orden de instalación

AD3-60 y AD3-62 no dependen de AD3-53–59, 61, 69, 70 ni 80: se ensayaron
instaladas antes y después de ellas sobre el núcleo AD3 real.

1. `roles_up.sql` como DBA, una sola vez.
2. `../autorizacion_atestada_v3/migraciones/000060_documentos_comunes.up.sql` por su migrador.
3. `migraciones/000001_documentos_comunes.up.sql` por el migrador documental.
4. `../autorizacion_atestada_v3/migraciones/000062_replay_documentos_comunes.up.sql`.
5. `migraciones/000002_replay_autorizado.up.sql`.
6. `migraciones/000003_custodia_externa.up.sql`, en la misma ventana y antes
   de la primera alta: su precondición exige que la numeración interna no se
   haya usado, porque el registro de identificadores parte vacío.
7. `roles_000004_up.sql` como DBA, una sola vez (rol `vec_documentos_auditor`).
8. `migraciones/000004_efecto_contexto_y_frontera.up.sql`.
9. Crear fuera del repositorio dos LOGIN de aplicación: uno con **solo** la
   membresía `vec_documentos_ejecutor` y otro con **solo**
   `vec_documentos_auditor`; no conceder propiedad, migración ni acceso a tablas.

## Documentos-4

El PDP V3 real fija `huella_efecto_sha256` y `contexto_recurso_huella_sha256`
como SHA-256 del contexto canónico del recurso autorizado, no de la
preimagen. `consumir_v3_v1/v2` (000001/000002, nunca instaladas, corregidas
en su sitio) exigen esa huella del recurso
`{"ambitos":{},"atributos":{"preimagen_sha256":"<hex>"}}`, que es la que
construye `ports.RecursoV3` en Go. 000004 ya no las reescribe: comprueba
antes de crear nada que el SHA-256 de su cuerpo instalado (`prosrc`)
coincide con el de los cuerpos exactos de 000001/000002, que ligan esa huella,
y publica la misma expresión como `huella_efecto_v1`. Si se corrige alguno de
esos cuerpos, hay que recalcular y fijar su huella en 000004. La columna
`huella_preimagen_sha256` sigue guardando `SHA-256(preimagen)`.

`confirmar_alta_v1` y `preparar_notificacion_v1` no se conceden al ejecutor:
el alta y la preparación solo entran por sus versiones v2 (replay autorizado).

También crea `denegacion_frontera` (solo adición, RLS forzada, sin lectura)
y `registrar_denegacion_frontera_v1`, que solo puede ejecutar un LOGIN con la
única membresía `vec_documentos_auditor` (exactamente una, heredada, sin
`SET` ni `ADMIN`, como AD3-60 exige al ejecutor). La frontera HTTP de
`/api/vec/documentos/` registra ahí sus denegaciones con valores cerrados.

AD3-60 y AD3-62 no se han instalado nunca en ninguna base: el 25/09/2026 se
ampliaron en su sitio con la acción `documentos.externo.registrar`. AD3-60
toma también el cerrojo común `vec_autorizacion_atestada_v3:nucleo`, como las
demás migraciones que reescriben el núcleo (53, 54, 59, 61, 70, 80). La AD3-61
de `main` es de Dietas (competencias y rectificación) y no guarda relación
con este módulo.

No reaplicar ni ejecutar `DOWN` sobre bases con historia. Cada fachada exige
transacción `SERIALIZABLE` y material V3 nominal. El alta inicial consume la
decisión en el mismo `COMMIT` que estado, auditoría y outbox. Un replay exacto
puede recibir `consumo_nuevo=false` solo mientras sigan vigentes capacidad,
clave, configuración, raíz y decisión; se cotejan principal, decisión y
auditoría originales, preimagen y recibo objeto. No crea otro documento ni
outbox. El material V3 original caduca como máximo a los cinco segundos: una
recuperación posterior exige una **decisión V3 fresca**, ligada a la misma
preimagen, recurso, principal y clave idempotente. Esta decisión produce su
propio consumo y auditoría de autorización; el documento, número, recibo y
outbox originales permanecen intactos. La consulta exige consumo nuevo y
registra el acceso antes de resolver el objeto por su referencia y versión
inmutables.

## Custodia externa

Cuando el original ya lo custodia otro sistema (justificantes de Dietas o de
Bolsa, correos, un gestor documental), VEC registra solo su referencia opaca
en el custodio (`custodio_id` + `custodia_ref`), la huella SHA-256 y los
metadatos gobernados: tipo documental, versión y política de conservación.
MIME y tamaño son opcionales porque no todos los custodios los declaran. La
tabla `referencia_externa` no tiene columnas de objeto: no se sube contenido
y la descarga sigue limitada a originales custodiados por VEC. Un mismo
identificador no puede nombrar a la vez un original y una referencia externa
(`identificador_documental`). La lista v2 devuelve ambos tipos con el campo
`custodia` (`vec` o `externa`), con el mismo cursor.

## Política de conservación provisional

El catálogo local de conservación (`internal/vec/adapters/conservacion`) es
**provisional**: sus plazos son de desarrollo, pendientes de RRHH (dudas.md,
preguntas 60 y 61). Mientras lo sea, **no se fija retención en el proveedor**:

- El resolutor emite la política con estado `provisional`, nunca `aprobada`.
  `ResolverPoliticaConservacionDocumental` la deniega; solo Documentos usa la
  variante `...AdmitiendoProvisional`, que conserva el estado.
- `estado_politica` (`aprobada` | `provisional`) forma parte de la preimagen
  autorizada de `confirmar_alta_v2` y `registrar_referencia_externa_v1`, se
  guarda en `documento` y `referencia_externa` y aparece en recibos y listas
  (`conservacion` en la consulta HTTP). Otro valor es 42501; la misma clave con
  el estado cambiado es conflicto 23505.
- Un original con política provisional se confirma con
  `objeto_retenido_hasta` nulo, sin inmovilizar y solo con protección
  ordinaria; con política aprobada se sigue exigiendo retención hasta el plazo.
  La tabla lo impone con un `CHECK`, también al superusuario.
- El arranque solo acepta el catálogo provisional en el perfil de desarrollo
  con doble llave y con un conector que declare no fijar retención al escribir
  (`RetencionAlEscribir() == false`): `ficheros` con `retencion_minima_dias: 0`.
  El conector S3 (Object Lock) la fija siempre y falla cerrado al componer.

### Tarea pendiente: aplicar el catálogo definitivo

No está implementada. Cuando RRHH apruebe los plazos (preguntas 60 y 61) hará
falta una operación durable, autorizada (V3 nominal propia) y auditada que,
por cada documento con `estado_politica = 'provisional'`:

1. resuelva la política aprobada que le corresponde y calcule el plazo según
   la regla de cómputo aprobada;
2. aplique la retención en el proveedor (`AplicarRetencion`) y verifique su
   recibo;
3. registre en historia de solo adición la nueva política, el plazo y la
   retención, conservando la decisión provisional original (hoy las tablas
   son inmutables: la operación añadirá su propia tabla de aplicaciones y la
   proyección combinará ambas);
4. todo ello en la misma transacción que autorización, auditoría y outbox, con
   idempotencia y reanudación si el proveedor falla a mitad.

Con el catálogo definitivo instalado, el montaje volverá a admitir un conector
que fije retención al escribir; los documentos provisionales ya incorporados
seguirán sin retención hasta ejecutar esta operación.

## Ensayos

`probar_integracion_pg18.sh` crea y destruye su propio contenedor PostgreSQL
18.4. Comprueba `ROLLBACK`/`COMMIT`, RLS forzada y ACL. Prueba el replay
inmediato con el mismo material dentro de su TTL y, después de reiniciar y
dejarlo caducar, una decisión sintética nueva con idénticos efecto y clave:
devuelve el mismo recibo, mantiene un documento y un outbox, y registra dos
consumos de autorización distintos. Hace lo mismo con un registro de custodia
externa (replay; clave reutilizada, identificador compartido con un original,
referencia con ruta y finalidad ajena rechazados) y con la lista v2 paginada
sobre ambas custodias. Con la política provisional (en un expediente propio)
confirma un alta sin retención y su replay, rechaza con 42501 estados fuera de
catálogo, un provisional con retención o inmovilizado y un aprobado sin
retención, y con 23505 el replay de la misma clave con el estado cambiado; lo
mismo para una referencia externa; y comprueba que el `CHECK` de la tabla
rechaza filas incoherentes incluso al superusuario. Al final ejecuta el repositorio Go (pgx) contra esa base
con el LOGIN ejecutor para cotejar preimagen, proyección y lista v2
(`VEC_DOCUMENTOS_SIN_GO=1` lo omite). Además rechaza con 42501 una decisión
fresca con la huella de otra preimagen, el replay AD3-62 de una decisión
retirada, el UPDATE/DELETE como superusuario de `documento`, `outbox` y
`auditoria_operacion`, y el registro de denegaciones por un LOGIN auditor con
otra membresía o con `SET`. Corre sobre una **preimagen sintética** AD3-50 seguida
de AD3-51/52/60/62 reales, y sustituye en esa base las fachadas AD3 por
recibos sintéticos: no acredita COSE.

Ambos scripts aceptan `VEC_PG18_DATOS_DIR` (p. ej. `/dev/shm`) para guardar
los datos del PostgreSQL desechable en un subdirectorio temporal de ese
directorio, que borran al salir; no crean volúmenes con nombre.

`probar_cadena_real_pg18.sh BASE.dump ROLES.sql [main-60|60-main]` restaura
una base VEC sintética con el núcleo AD3 real (la misma preimagen del ensayo
Dietas 000008) e instala encima la cadena Dietas, AD3-53/54/70/80 y las
migraciones documentales en los dos órdenes posibles. Comprueba cada
exclusión del núcleo una sola vez, la fachada AD3-60 real con el LOGIN exacto
hasta la clave de capacidad (alta y externa), el rechazo con dos grupos o con
login ajeno, RLS/ACL y la persistencia tras reiniciar. Tampoco acredita COSE:
la base no tiene clave publicada para `vec_documentos.operacion.v1`.

`estado_firma` permanece `pendiente_proveedor` e inmutable. La integración con
AutoFirmaV2 requiere recibo verificable del verificador y otro consumidor V3
nominal; la referencia del objeto firmado o un resultado booleano no habilitan
la transición. La preparación de notificación no declara envío ni entrega.

## Montaje en vec-server

Selector de despliegue `VEC_DOCUMENTOS_ENABLED` (`true`/`false`; ausente =
apagado), sujeto a la doble llave de desarrollo. Apagado no cambia nada: ni
rutas, ni catálogo `/api/vec/modules`, ni material V3. Encendido, cualquier
pieza ausente impide arrancar (`bootstrap: Documentos no disponible`, con
fichero y línea, sin DSN ni rutas).

Encendido, vec-server publica en el gobierno V3 único la clave de la
audiencia `vec_documentos.operacion.v1` (AD3-60) y lee el material privado
`identidad/documentos.json` del directorio de material de desarrollo (fuera de
Git, 0600):

```json
{
  "version": 1,
  "autoridad": "no_autoritativo",
  "cuentas": [{"certificado_sha256": "<64 hex>", "sujeto": "...", "cuenta_ref": "...", "perfil_ref": "prf_..."}],
  "dsn_registro_identidad": "...", "dsn_revalidacion_identidad": "...",
  "dsn_contexto": "...", "dsn_fuente_autorizacion": "...",
  "dsn_registro_autorizacion": "...", "dsn_motivos": "...",
  "dsn_documentos": "<LOGIN con solo vec_documentos_ejecutor>",
  "dsn_documentos_auditor": "<LOGIN con solo vec_documentos_auditor>",
  "motivos": {"listar": {"catalogo_id": "...", "catalogo_version": 1, "catalogo_huella_sha256": "...", "entrada_clave": "..."}},
  "almacen": {"tipo": "ficheros", "directorio": "/ruta/absoluta/privada", "tamano_maximo": 16777216, "retencion_minima_dias": 0}
}
```

`almacen.tipo` admite `ficheros` (predeterminado: directorio propio del
proceso, 0700) o `s3` con el mapa `s3` del conector S3 existente. Con el
catálogo de conservación provisional solo arranca `ficheros` con
`retencion_minima_dias: 0` (sin retención al escribir; véase «Política de
conservación provisional»). Los DSN
exigen TLS verificado y LOGIN distintos. Las cuentas deben existir ya en la
identidad de desarrollo; el catálogo de motivos y las concesiones de
`documentos.expediente.listar` (tipo `expediente_documental`, campos
`["items","siguiente_cursor"]`) son datos de autorización, no de este montaje.

Queda publicada la consulta `POST /api/vec/documentos/expedientes/consultas`.
La descarga de originales no se publica todavía: leer el original exige una
decisión de almacén propia (`NuevoContextoLeerDocumentoGeneradoAlmacen`) que
la raíz no puede obtener del PDP V3; la lista marca `descargable:false`.
