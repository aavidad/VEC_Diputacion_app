# Preparar una denominación de Persona

Esta CLI prepara un nombre sintético para una Persona referenciada. Produce un
sobre cifrado y un índice de búsqueda; no comprueba que Persona exista, no
publica datos, no aplica SQL ni concede permisos. CA32 deberá cotejar Persona,
procedencia, versión y autorización antes de confirmar la publicación.

Usa el protector de denominación existente y el material maestro privado del
KMS de desarrollo. Deriva una clave para cifrado y otra para búsqueda mediante
`derivarClaveDesarrollo`, en dominios dedicados. Las referencias y versiones
provienen de la configuración. Nunca selecciona claves de contacto, correo,
SMTP o autenticación. No crea una clave maestra por defecto.

Ejemplo de ejecución (rutas privadas propias, fuera de Git):

```sh
go run -p 8 ./cmd/vec-preparar-denominacion-persona \
  --configuracion /ruta/privada/denominacion.json \
  --maestra-fichero /ruta/privada/maestra-existente.bin \
  --nombre-fichero /ruta/privada/nombre.txt \
  --salida /ruta/privada/preparacion.json \
  --persona-ref per_aaaaaaaaaaaaaaaaaaaaaaaa \
  --version-esperada 0 \
  --procedencia-ref prc_bbbbbbbbbbbbbbbbbbbbbbbb \
  --ambito-ref ambito:administracion_pruebas \
  --sintetico --idioma es \
  --textos /ruta/repositorio/web/static/textos/es/persona-denominacion-preparar.json
```

La referencia del ejemplo no demuestra una Persona existente. Para el recorrido
real de desarrollo se usa la referencia recibida de la autoridad común y la
procedencia aprobada. Persona y procedencia tienen un máximo total de 128
bytes. La procedencia usa `prc_[A-Za-z0-9_-]{22,124}`, igual que CA32; se
comprueba tanto al preparar como al reutilizar una salida. No se toma el
nombre del CN del certificado, cargo o rol.
El archivo de nombre contiene únicamente el UTF8 exacto, sin salto final;
la norma exige NFC y rechaza controles y blancos exteriores.

Los archivos privados requieren `0600` y su directorio inmediato `0700`, propios
del operador, sin enlaces y fuera de Git. El maestro es el archivo binario
existente de 32 bytes, no una cadena hexadecimal ni una clave nueva. La salida
se crea con O_EXCL y `0600`. Si el contenido del nombre coincide con los 32
bytes del maestro, la CLI lo rechaza, incluso mediante otra ruta, enlace duro
o copia; no compara inodos. El catálogo de mensajes es público y versionado.

Configuración cerrada, sin material secreto:

```json
{
  "version": 1,
  "entorno": "desarrollo",
  "norma": {
    "referencia": "norma:denominacion:ejemplo:v1",
    "case": "fold",
    "forma_unicode": "NFC",
    "separadores": " -'",
    "max_bytes": 512,
    "max_tokens": 12
  },
  "cifrado": {
    "referencia": "clave:denominacion:cifrado:ejemplo",
    "version": 1,
    "revocada": false,
    "retener_hasta": "0001-01-01T00:00:00Z"
  },
  "busqueda": {
    "referencia": "clave:denominacion:busqueda:ejemplo",
    "version": 1,
    "revocada": false,
    "retener_hasta": "0001-01-01T00:00:00Z"
  },
  "cifrado_retenidas": []
}
```

Son valores de ejemplo reemplazables por configuración aprobada. La referencia
que usa el sobre añade `:v<version>` a la referencia base de cada clave. Una
versión nueva de cifrado puede conservar hasta ocho referencias anteriores,
con fecha de retención explícita; todas derivan del mismo material maestro
existente. Cambiar el maestro requiere su procedimiento de rotación, no cambiar
el archivo bajo la misma referencia. La rotación de búsqueda requiere reindexar
antes de utilizar el índice nuevo.

El proveedor relee el archivo en cada carga y revalidación: una revocación,
caducidad o referencia distinta cierra la operación. Si cambia la norma, el
protector anterior queda cerrado; hace falta componer la versión correspondiente.
No conserva una configuración favorable en memoria para eludir el retiro.

La salida declara `preparada_pendiente_cotejo_y_autorizacion` y
`sintetico_declarado`. `preparacion` conserva la estructura del puerto existente:
`PersonaRef`, `ProcedenciaRef`, `SobreSHA256`, `VersionEsperada` y `Sobre`.
El sobre mantiene exactamente `Esquema`, `PersonaRef`, `ClaveRef`, `Version`,
`Nonce`, `Cifrado`, `Indice`; su índice conserva `AmbitoRef`, `NormaRef`,
`NormaSHA256`, `ClaveRef` y `Tokens`. Nonce, cifrado y tokens usan base64.

`sobre_canonico` conserva en base64 los bytes originales de `json.Marshal(Sobre)`.
`SobreSHA256` se calcula exclusivamente sobre esos bytes. No contiene nombre
claro, claves, una huella del nombre ni una atestación ficticia. La consola sólo
muestra el resultado y un mensaje del catálogo, sin Persona, nombre o rutas.

Al repetir con la misma salida, la CLI valida el artefacto, su SHA, coordenadas,
protección vigente y coincidencia del nombre mediante un callback temporal.
Devuelve la misma preparación y no genera otro nonce ni sobrescribe el archivo.
Si cambian nombre, versión, Persona, ámbito, procedencia o protección, rechaza
el reintento. Conserve la salida original para reconciliar una publicación
posterior; no vuelva a cifrar ante un COMMIT incierto.

El método de composición
`ComposicionSeguridadDesarrollo.ProtectorDenominacionPersonaDesarrollo` utiliza
el KMS ya cargado. La fábrica offline
`NuevoProtectorDenominacionPersonaDesarrollo` recibe el maestro existente y el
lector de metadatos. Ninguna modifica la composición común o la autoridad V3.

Comprobaciones del productor: pruebas normales y `-race` focales de bootstrap
y la CLI, `go vet -p 8` de ambos paquetes, catálogo Node (1/1), Semgrep local
con métricas desactivadas (cuatro reglas, tres archivos, cero hallazgos),
vecsilencio sin fallos nuevos y `git diff --check`. La ejecución Go se aisló
sin red, con fuente/módulos de solo lectura, temporales propios y caché en disco.

Gosec se ejecutó sólo sobre los dos paquetes afectados. Un G304 nuevo del
lector público de catálogo se corrigió usando os.Root y NOFOLLOW. La pasada
final deja cero hallazgos en archivos nuevos y 17 heredados en diez archivos
de bootstrap, cuyos bytes coinciden con la base `483a321ea`: material_desarrollo,
fake_credentials, portal_externo_material_v3, bolsa_importacion_convoca_custodia,
catalogo_rpt_desarrollo, contratacion_temporal_centros_anteriores,
contratacion_temporal_comunicacion_llamamiento_desarrollo,
contratacion_temporal_incorporacion_configuracion,
contratacion_temporal_propuesta_publicaciones_desarrollo y
contratacion_temporal_subsanacion_politica_desarrollo. Esta pieza no modifica
esos avisos previos ni añade exclusiones para ocultarlos.

La revisión independiente y la CI de la PR quedan pendientes. Estas pruebas
no acreditan una Persona existente, publicación SQL ni autorización V3.
