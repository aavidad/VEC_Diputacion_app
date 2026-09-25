# vec-preparar-material-interno

Compone y valida el inventario privado `personal_b2_v3.json` (formato 4) que
carga `vec-interno` para montar el registro B2 de Personal, junto con las ocho
claves HMAC de capacidad que referencia. Sustituye al script privado usado en
el clon (plan acordado B2/gobierno V3, paso 5).

La herramienta **no publica** gobierno, raíz, configuración, claves ni
permisos (el único publicador es `vec-server`), **no amplía** el rol de
preflight y **no lee secretos** del gobierno: ninguna consulta menciona la
columna `secreto_hmac`. Tampoco crea otra derivación: usa
`bootstrap.DerivarClavesPersonalB2V3Desarrollo`, la misma que aplica el
publicador.

## Qué comprueba antes de escribir nada

1. `ct_v3.json` se abre con el cargador real (`internactproveedores.CargarMaterial`).
   De él solo se toman el catálogo de motivos, el emisor (único en las cinco
   capacidades) y las coordenadas de la raíz. `ct_v3.json` **no se regenera**:
   su configuración es solo el punto de partida del lector renovable, que la
   actualiza cada día sin tocar el fichero, y no contiene nada de B2.
2. La clave base CT (`clave:capacidad:ct:…`) se localiza en el gobierno por la
   huella SHA-256 de su secreto. Debe ser la clave base del publicador de
   desarrollo (audiencia `vec_contratacion_temporal.confirmar_alta_atestada.v1`,
   acto `acto:ct:desarrollo:clave-capacidad:`), no revocada y con el mismo emisor
   que `ct_v3.json`. Su identificador, emisor y vigencia salen del gobierno.
3. Se derivan las ocho claves B2 y cada una debe estar publicada con el mismo
   identificador, audiencia, huella de gobierno, huella del secreto, emisor y
   vigencia; vigente ahora, sin ninguna revocación (ni siquiera programada),
   dentro del checkpoint y como puntero de emisión vigente de su audiencia.
   `version` y `revision_gobierno` se toman de esa fila publicada.
4. La raíz de la configuración vigente (puntero, no revocada, no caducada,
   secuencia y versión dentro del checkpoint, una sola raíz) debe ser la de
   `ct_v3.json`.

Todo se lee en **una sola instantánea** (`REPEATABLE READ READ ONLY`, además
de `default_transaction_read_only=on`) con el reloj `clock_timestamp()` que
usa la sonda AD3-69. Cualquier diferencia falla cerrada.

Después compone el material en un directorio temporal hermano de la salida,
lo abre con el cargador real de formato 4
(`internactproveedores.CargarMaterialPersonalB2`) y reconstruye cada clave con
los mismos constructores y comprobaciones que la composición de
`vec-interno` (huella del fichero, clave HMAC de emisión, audiencia y
vigencia). Solo si todo es correcto activa la salida con un único
`rename(2)`. Si falla o se interrumpe (SIGINT, SIGTERM, SIGHUP) antes de ese
punto, el temporal se borra y la salida no existe o sigue vacía; un `kill -9`
puede dejar un temporal oculto `.vec-preparar-material-*` en el directorio
padre, nunca material parcial en la salida.

## Entradas privadas

| Entrada | Forma | Requisitos |
| --- | --- | --- |
| `-inventario-ct` | ruta de `ct_v3.json` existente | ruta absoluta sin enlaces; el directorio lo valida el cargador real (0700, fuera de Git, fichero 0600) |
| `-clave-base` | fichero con la clave base CT **en bruto** (32–256 bytes) | 0600, regular, del usuario, sin enlaces |
| `-motivos` | fichero JSON | 0600; formato abajo |
| `-salida` | directorio **nuevo** | inexistente, o vacío con 0700 y del usuario; padre del usuario sin escritura de grupo/otros, sin enlaces y fuera de cualquier árbol Git |
| DSN de lectura | `-dsn-archivo` (fichero 0600, una línea) **o** variable `VEC_PREPARAR_MATERIAL_GOBIERNO_DSN` | exactamente uno; nunca como argumento; la variable se borra del entorno al arrancar |

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

**LOGIN de lectura.** Las tablas de gobierno tienen RLS forzada solo para
`vec_autorizacion_atestada_v3_propietario`. El LOGIN debe poder asumir ese rol
(`SET ROLE`), como el pool de gobierno de `vec-server`; la herramienta lo hace
con `SET LOCAL ROLE` dentro de la transacción de solo lectura. El rol de
preflight de `vec-interno` no sirve y no se amplía.

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
son mensajes fijos sin rutas, DSN ni secretos. Código 0 = activado, 1 =
rechazado, 2 = uso incorrecto.

## Uso en el clon

`MATERIAL` es el directorio privado que ya usa `vec-interno`
(`VEC_INTERNO_MATERIAL_DIR`, con `ct_v3.json`). `PRIVADO` es un directorio
0700 del operador fuera de Git.

```bash
# 1. Compilar fuera del repositorio.
go build -o "$PRIVADO/vec-preparar-material-interno" ./cmd/vec-preparar-material-interno

# 2. DSN del LOGIN que puede asumir el propietario AD3 (el del gobierno de
#    vec-server), en fichero 0600; nunca en argumentos.
install -m 0600 /dev/null "$PRIVADO/gobierno.dsn"
"$EDITOR" "$PRIVADO/gobierno.dsn"

# 3. Clave base CT en bruto y motivos, ambos 0600 (material privado existente).
chmod 0600 "$PRIVADO/clave_base_ct.bin" "$PRIVADO/motivos_b2.json"

# 4. Preparar en un directorio nuevo junto al material.
"$PRIVADO/vec-preparar-material-interno" \
  -inventario-ct "$MATERIAL/ct_v3.json" \
  -clave-base "$PRIVADO/clave_base_ct.bin" \
  -motivos "$PRIVADO/motivos_b2.json" \
  -dsn-archivo "$PRIVADO/gobierno.dsn" \
  -salida "$PRIVADO/b2-$(date -u +%Y%m%dT%H%M%SZ)"
```

Instalación en el material de `vec-interno` (mismo sistema de ficheros, con el
servicio detenido o antes de arrancarlo). `vec-interno` lee
`personal_b2_v3.json` del mismo directorio que `ct_v3.json`; las claves se
copian primero y el inventario el último, con `rename`, para que nunca exista
un inventario que apunte a claves ausentes:

```bash
SALIDA="$PRIVADO/b2-…"
for f in "$SALIDA"/personal_b2_*.hmac; do
  install -m 0600 "$f" "$MATERIAL/.$(basename "$f").nuevo" &&
  mv -f "$MATERIAL/.$(basename "$f").nuevo" "$MATERIAL/$(basename "$f")"
done
install -m 0600 "$SALIDA/personal_b2_v3.json" "$MATERIAL/.personal_b2_v3.json.nuevo"
mv -f "$MATERIAL/.personal_b2_v3.json.nuevo" "$MATERIAL/personal_b2_v3.json"
```

Al arrancar, `vec-interno` vuelve a sondear el material con
`comprobar_material_emision_interna_v2('personal_b2', …)`; si el gobierno cambió
entre la preparación y el arranque, B2 queda fuera (404) sin afectar a CT.

## Pruebas

- Unitarias (`go test ./cmd/vec-preparar-material-interno/`): gobierno
  inconsistente (clave ausente, puntero no vigente, revocación, checkpoint,
  caducidad, huella, audiencia, vigencia, acto ajeno, raíz distinta o no
  vigente), material privado divergente, permisos, enlaces simbólicos, salida
  con contenido o reaparecida en carrera, interrupción antes de activar,
  motivos inválidos, DSN y ausencia de secretos en la salida, con carga final
  mediante los cargadores reales.
- PostgreSQL 18.4 desechable: `cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh`
  instala la cadena canónica AD3 1/2, publica un gobierno sintético con la
  derivación de T3 y ejecuta el cotejo real: positivo, transacción de solo
  lectura (escritura rechazada con 25006), LOGIN sin rol propietario, clave
  base divergente, revocación programada y puntero rotado. Requiere Docker (o
  `VEC_CONTENEDOR_MOTOR=podman`) y la imagen `postgres:18.4-alpine`.
