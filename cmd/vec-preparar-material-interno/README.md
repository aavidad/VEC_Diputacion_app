# vec-preparar-material-interno

Compone y valida el inventario privado `personal_b2_v3.json` (formato 4) que
carga `vec-interno` para montar el registro B2 de Personal, junto con las ocho
claves HMAC de capacidad que referencia. Sustituye al script privado usado en
el clon (plan acordado B2/gobierno V3, paso 5).

La herramienta **no publica** gobierno, raíz, configuración, claves ni
permisos (el único publicador es `vec-server`), **no amplía** el rol de
preflight y **no lee secretos** del gobierno: ninguna consulta menciona la
columna `secreto_hmac`.

## De dónde salen las claves

La clave base de capacidad CT (`clave:capacidad:ct:desarrollo:vN`) **no existe
en ningún fichero**: `vec-server` la calcula en memoria al arrancar como HMAC
del material privado de idempotencia de la generación activa, y de ella
derivan todas las claves por audiencia (CT, Bolsa, Cronos, Dietas y B2). La
herramienta no pide esa clave ni crea otra derivación:
`bootstrap.DerivarClavesPersonalB2V3DesdeMaterialDesarrollo` recorre la misma
ruta de código que `vec-server` (`cargarMaterialIdempotenciaDesarrollo` →
`nuevoDerivadorIdentidadOperacionDesarrollo` →
`nuevoMaterialAtestacionContratacionTemporalDesarrollo` →
`derivarMaterialConsumidorV3Desarrollo` con los descriptores B2) y devuelve
directamente las ocho claves B2. La clave base, la semilla Ed25519 y el
material de idempotencia solo viven en memoria durante esa llamada y nunca se
escriben ni se devuelven. Se borran las copias propias; el constructor de
confianza conserva copias internas que no se sobrescriben y quedan en memoria
hasta el recolector, igual que en vec-server. Una prueba unitaria demuestra que las
claves coinciden byte a byte con las que publica
`publicarMaterialPersonalB2Desarrollo`, y el ensayo PostgreSQL lo repite
comparando en el servidor con el secreto que dejó el publicador real.

## Qué comprueba antes de escribir nada

1. `ct_v3.json` se abre con el cargador real (`internactproveedores.CargarMaterial`).
   De él solo se toman el catálogo de motivos, el emisor (único en las cinco
   capacidades) y las coordenadas de la raíz. `ct_v3.json` **no se regenera**:
   su configuración es solo el punto de partida del lector renovable, que la
   actualiza cada día sin tocar el fichero, y no contiene nada de B2.
2. El material de idempotencia se valida como lo hace `vec-server` (formato,
   generaciones, ficheros regulares sin acceso de terceros ni enlaces) y el
   emisor derivado de la generación activa debe ser el de `ct_v3.json`.
3. Cada una de las ocho claves B2 debe estar publicada con el mismo
   identificador, audiencia, huella de gobierno, huella del secreto, emisor y
   vigencia; vigente ahora, sin ninguna revocación (ni siquiera programada),
   dentro del checkpoint y como puntero de emisión vigente de su audiencia.
   `version` y `revision_gobierno` se toman de esa fila publicada.
4. La raíz de la configuración vigente (puntero, no revocada, no caducada,
   secuencia y versión dentro del checkpoint, una sola raíz) debe ser la de
   `ct_v3.json`.
5. La identidad de la sesión es la del LOGIN de gobierno de `vec-server`,
   con los mismos atributos que exige su pool (LOGIN, INHERIT, sin
   SUPERUSER, CREATEDB, CREATEROLE, REPLICATION ni BYPASSRLS) y además
   **membresía exacta**: su única pertenencia directa es
   `vec_autorizacion_atestada_v3_migrador`, sin ADMIN, y el cierre
   transitivo solo contiene ese grupo y el propietario AD3.

Todo se lee en **una sola instantánea** (`REPEATABLE READ READ ONLY`, además
de `default_transaction_read_only=on`) con el reloj `clock_timestamp()` que
usa la sonda AD3-69. Cualquier diferencia falla cerrada.

**Checkpoint recién publicado.** El publicador de `vec-server` asigna a cada
clave una revisión nueva, pero el checkpoint (`avanzar_checkpoint`, AD3-2)
solo avanza cuando se publica una configuración o una raíz nueva. Unas claves
B2 recién publicadas pueden quedar con `revision_gobierno` mayor que la del
checkpoint: la sonda de `vec-interno` las rechazaría y la herramienta también
(«lo derivado no coincide con el gobierno publicado vigente»). Hay que
esperar a que el checkpoint las alcance; la herramienta no lo adelanta.

Después compone el material en un directorio temporal hermano de la salida,
lo abre con el cargador real de formato 4
(`internactproveedores.CargarMaterialPersonalB2`) y reconstruye cada clave con
los mismos constructores y comprobaciones que la composición de
`vec-interno` (huella del fichero, clave HMAC de emisión, audiencia y
vigencia). Solo si todo es correcto activa la salida con un único
`renameat(2)`.

El directorio padre de la salida se abre **una vez por descriptor**
(`O_DIRECTORY|O_NOFOLLOW` y `os.Root` sobre el mismo inodo): el temporal se
crea, se escribe, se retira y se activa relativo a ese descriptor. Antes de
validar y de activar se vuelve a comprobar que la ruta del padre sigue
designando ese mismo inodo, del usuario y sin escritura de grupo ni otros; si
no, no se activa en ningún sitio y el temporal se retira del directorio
original.

Si falla o se interrumpe (SIGINT, SIGTERM, SIGHUP) antes de activar, el
temporal se borra y la salida no existe o sigue vacía; un `kill -9` puede
dejar un temporal oculto `.vec-preparar-material-*` en el directorio padre,
nunca material parcial en la salida. Si el `renameat(2)` se hizo pero el
`fsync` del padre falla, el material **ya está activado**: la herramienta
termina con código 0 y el mensaje «material Personal B2 activado; fsync del
padre no confirmado». No hay que repetir la preparación; basta con `sync` o
comprobar la salida tras un reinicio.

## Entradas privadas

| Entrada | Forma | Requisitos |
| --- | --- | --- |
| `-inventario-ct` | ruta de `ct_v3.json` existente | ruta absoluta sin enlaces; el directorio lo valida el cargador real (0700, fuera de Git, fichero 0600) |
| `-material-idempotencia` | subdirectorio `idempotencia` del material de desarrollo de `vec-server` | ruta absoluta canónica sin enlaces y fuera de Git; se llama exactamente `idempotencia`; 0700 y del usuario; su directorio padre, del usuario y sin acceso de grupo ni otros; ficheros según el cargador de `vec-server` |
| `-motivos` | fichero JSON | 0600; formato abajo |
| `-salida` | directorio **nuevo** | inexistente, o vacío con 0700 y del usuario; padre del usuario sin escritura de grupo/otros, sin enlaces y fuera de cualquier árbol Git |
| DSN del gobierno | `-dsn-archivo` (fichero 0600, una línea; **recomendado**) **o** variable `VEC_PREPARAR_MATERIAL_GOBIERNO_DSN` | exactamente uno; nunca como argumento |

**DSN por fichero.** La variable de entorno se retira del entorno del proceso
para que no pase a procesos hijos, pero eso **no la borra**: la copia inicial
sigue visible en `/proc/<pid>/environ` para el mismo usuario y root mientras
dure el proceso, y en la shell que la exportó. Use `-dsn-archivo`.

**Motivos por fichero, no por parámetros.** Son 8 referencias de cuatro campos
cada una: un fichero 0600 permite decodificación estricta (campos
desconocidos, claves duplicadas, documento único), no aparece en la lista de
procesos ni en el historial de la shell y usa los mismos nombres que el
inventario. Cada referencia debe ser un motivo V2 válido (clave opaca
`motivo_<32 hex>`) del mismo catálogo que `ct_v3.json`:

```json
{
  "ficha":              {"catalogo_id": "…", "catalogo_version": 1, "catalogo_huella_sha256": "<64 hex>", "entrada_clave": "motivo_<32 hex>"},
  "vacantes":           {"…": "…"},
  "alta":               {"…": "…"},
  "hecho":              {"…": "…"},
  "catalogo_consultar": {"…": "…"},
  "catalogo_publicar":  {"…": "…"},
  "catalogo_retirar":   {"…": "…"},
  "empleados":          {"…": "…"}
}
```

La herramienta valida su forma y su catálogo; su existencia y vigencia en el
catálogo gobernado las comprueba el PDP en cada decisión.

**LOGIN: el de gobierno de `vec-server`, no uno nuevo.** No se crea ningún
LOGIN para esta herramienta. Se reutiliza el LOGIN que `vec-server` usa para
publicar el gobierno (el de `VEC_CT_GOBIERNO_DATABASE_URL`), miembro de
`vec_autorizacion_atestada_v3_migrador`; ese grupo puede asumir el
propietario AD3 (`SET`, sin `INHERIT`), único rol que ve las tablas de
gobierno con RLS forzada. La herramienta lo asume con `SET LOCAL ROLE` dentro
de la transacción de solo lectura. El rol de preflight de `vec-interno` no
sirve y no se amplía. Si ese LOGIN tuviera otra membresía o atributo, la
herramienta lo rechaza: se corrige el LOGIN, no la herramienta.

## Salida

```text
<salida>/                         0700
  personal_b2_v3.json             0600  formato 4, sin secretos
  personal_b2_ficha.hmac          0600  secreto HMAC en bruto (32 bytes)
  personal_b2_vacantes.hmac       …
  personal_b2_alta.hmac
  personal_b2_hecho.hmac
  personal_b2_catalogo_consultar.hmac
  personal_b2_catalogo_publicar.hmac
  personal_b2_catalogo_retirar.hmac
  personal_b2_empleados.hmac
```

En la salida estándar solo aparece una línea de confirmación; los errores
son mensajes fijos sin rutas, DSN ni secretos. Código 0 = activado (incluido
el aviso de fsync del padre no confirmado), 1 = rechazado, 2 = uso incorrecto.

## Procedimiento en el despliegue

Se ejecuta como el usuario del servicio, en la máquina donde corren
`vec-server` y `vec-interno`, después de que `vec-server` haya arrancado con
B2 activo (y, por tanto, publicado las ocho claves) y de que el checkpoint las
haya alcanzado.

Directorios que intervienen (los valores reales son privados y no están en
Git):

- **`MATERIAL_VEC_SERVER`**: el directorio del material de desarrollo de
  `vec-server`, es decir, el valor de `VEC_DEVELOPMENT_MATERIAL_DIR` en su
  entorno de arranque. El operador lo localiza en el fichero de entorno o en
  el guion de arranque del servicio (por ejemplo, la línea
  `VEC_DEVELOPMENT_MATERIAL_DIR=` del entorno privado descrito en
  `deploy/principal/03_entorno.md`). `vec-server` no tiene valor por
  defecto: sin esa variable no arranca en desarrollo. Los guiones de
  `deploy/principal` que la necesitan usan, si falta,
  `$XDG_STATE_HOME/vec-diputacion/desarrollo` o
  `$HOME/.local/state/vec-diputacion/desarrollo`. Debe contener
  `idempotencia/configuracion.json` y los ficheros `idempotencia/g<N>-*.bin`
  que generó `scripts/generar_credenciales_desarrollo.sh`. La entrada de la
  herramienta es **`$MATERIAL_VEC_SERVER/idempotencia`**.
- **`MATERIAL_INTERNO`**: el directorio privado de `vec-interno`
  (`VEC_INTERNO_MATERIAL_DIR`, con `ct_v3.json`).
- **`PRIVADO`**: un directorio 0700 del operador fuera de Git para el binario,
  el DSN, los motivos y la salida.

```bash
# 1. Compilar fuera del repositorio.
go build -o "$PRIVADO/vec-preparar-material-interno" ./cmd/vec-preparar-material-interno

# 2. DSN del LOGIN de gobierno de vec-server (el mismo valor que su
#    VEC_CT_GOBIERNO_DATABASE_URL) en un fichero 0600; nunca en argumentos.
install -m 0600 /dev/null "$PRIVADO/gobierno.dsn"
"$EDITOR" "$PRIVADO/gobierno.dsn"

# 3. Motivos B2, 0600.
chmod 0600 "$PRIVADO/motivos_b2.json"

# 4. Preparar en un directorio nuevo.
"$PRIVADO/vec-preparar-material-interno" \
  -inventario-ct "$MATERIAL_INTERNO/ct_v3.json" \
  -material-idempotencia "$MATERIAL_VEC_SERVER/idempotencia" \
  -motivos "$PRIVADO/motivos_b2.json" \
  -dsn-archivo "$PRIVADO/gobierno.dsn" \
  -salida "$PRIVADO/b2-$(date -u +%Y%m%dT%H%M%SZ)"
```

Si la herramienta responde que lo derivado no coincide con el gobierno
publicado, las causas posibles son: `vec-server` todavía no ha publicado B2,
el checkpoint aún no alcanza sus revisiones, el material de idempotencia no es
el que usa el `vec-server` en marcha, o alguna clave B2 está revocada o ya no
es la de emisión. No hay que tocar el gobierno a mano.

Instalación en el material de `vec-interno` (mismo sistema de ficheros, con el
servicio detenido o antes de arrancarlo). `vec-interno` lee
`personal_b2_v3.json` del mismo directorio que `ct_v3.json`; las claves se
copian primero y el inventario el último, con `rename`, para que nunca exista
un inventario que apunte a claves ausentes:

```bash
SALIDA="$PRIVADO/b2-…"
for f in "$SALIDA"/personal_b2_*.hmac; do
  install -m 0600 "$f" "$MATERIAL_INTERNO/.$(basename "$f").nuevo" &&
  mv -f "$MATERIAL_INTERNO/.$(basename "$f").nuevo" "$MATERIAL_INTERNO/$(basename "$f")"
done
install -m 0600 "$SALIDA/personal_b2_v3.json" "$MATERIAL_INTERNO/.personal_b2_v3.json.nuevo"
mv -f "$MATERIAL_INTERNO/.personal_b2_v3.json.nuevo" "$MATERIAL_INTERNO/personal_b2_v3.json"
```

Al arrancar, `vec-interno` vuelve a sondear el material con
`comprobar_material_emision_interna_v2('personal_b2', …)`; si el gobierno cambió
entre la preparación y el arranque, B2 queda fuera (404) sin afectar a CT.
Si el material de idempotencia de `vec-server` rota de generación, las claves
B2 cambian: hay que volver a preparar tras la nueva publicación.

## Pruebas

- Unitarias de la derivación (`go test ./internal/app/bootstrap/ -run 'ClavesB2|PersonalB2'`):
  coincidencia byte a byte con la publicación de `vec-server` desde el mismo
  material (también otro día) y rechazo de rutas relativas, no canónicas, con
  enlaces, sin material, fuera de vigencia o con ficheros legibles por
  terceros.
- Unitarias de la herramienta (`go test ./cmd/vec-preparar-material-interno/`):
  gobierno inconsistente (clave ausente, puntero no vigente, revocación,
  checkpoint, caducidad, huella, audiencia, identificador, emisor, vigencia,
  acto ajeno, raíz distinta o no vigente), material de idempotencia divergente
  o rotado, permisos, enlaces simbólicos, nombre y ubicación del directorio de
  idempotencia, salida con contenido o reaparecida en carrera, padre
  sustituido antes de activar, fsync del padre no confirmado tras activar,
  interrupción antes de activar, motivos inválidos, DSN y ausencia de secretos
  en la salida, con carga final mediante los cargadores reales.
- PostgreSQL 18.4 desechable: `cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh`
  instala la cadena canónica AD3 1/2, crea un material de idempotencia
  sintético y un LOGIN de gobierno como el de `vec-server`, **publica con el
  publicador real de `vec-server`** (pool con su comprobación de identidad,
  CT y las ocho claves B2, dos veces para comprobar la idempotencia, y
  comparación byte a byte en el servidor) y ejecuta la herramienta: rechazo
  mientras el checkpoint no alcanza las claves, positivo tras adelantarlo
  (el DBA solo toca el checkpoint), transacción de solo lectura (escritura
  rechazada con 25006), identidad (sin grupo, superusuario, BYPASSRLS,
  CREATEROLE, membresía de más, NOINHERIT), material divergente, revocación
  programada y puntero rotado. Requiere Docker (o
  `VEC_CONTENEDOR_MOTOR=podman`) y la imagen `postgres:18.4-alpine`.
