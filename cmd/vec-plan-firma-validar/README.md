# Preparar y comprobar un material local del plan de firma

F1-121 añade la preparación completa de los dos materiales con la interfaz
que usa la receta de la copia. Se compila desde Git y trabaja con archivos
locales, sin conexiones a la red o a la base:

```sh
go_local=$(scripts/seleccionar_toolchain_go_local.sh)
GOTOOLCHAIN=local GOPROXY=off "$go_local" build -p 6 \
  -o "$RUN/vec-plan-firma-preparar" ./cmd/vec-plan-firma-validar
"$RUN/vec-plan-firma-preparar" --config "$RUN/config.json" --salida "$RUN/material"
```

`RUN` es una carpeta privada propia. La configuración contiene exactamente
las siguientes claves, compatibles con el consumidor cedido por H el 20/10:

| Claves | Contenido |
| --- | --- |
| `version`, `copia_ref` | `1` e identificador de la copia |
| `circuito_archivo`, `circuito_archivo_sha256` | Paquete local del circuito publicado v2 y huella de sus bytes |
| `entradas_archivo`, `entradas_archivo_sha256` | Las cinco entradas ordenadas y la huella de ese archivo |
| `fuente_ref` | Referencia de la fuente del plan |
| `creado_por`, `creado_perfil_ref`, `publicado_por`, `publicado_perfil_ref` | Referencias de los dos administradores y sus perfiles |
| `aprobacion_ref`, `preparacion_ref` | Referencias de aprobación propuesta y preparación local |
| `clave_operacion_crear`, `clave_operacion_publicar` | Claves distintas que se conservarán con los materiales |
| `correlacion_crear`, `correlacion_publicar` | Correlaciones de cada operación |

Cada entrada aporta su cargo, rol, perfil esperado, organización, unidad,
correspondencia y vigencia, como indica la
[receta de firma](../../deploy/principal/candidata_hz39/firma/LEEME.md).
El preparador completa las referencias de circuito y paso desde el paquete
v2 y calcula la huella del catálogo con el modelo común. Los textos del
catálogo proceden de `plantilla_plan_copia.json`.

La salida debe ser nueva y su padre, privado. Guarda seis archivos `0600`:
`crear-catalogo.json`, `crear-material.json`, `crear-material.sha256` y sus
equivalentes `publicar-*`. Ambos materiales pasan el validador existente
antes de escribirse; la publicación queda ligada a la huella del borrador.
El mismo ejecutable acepta la validación histórica por ruta y huella.

La fecha propuesta se fija al preparar. Para recuperar una operación se usan
los archivos y claves conservados, sin regenerarlos. Un error de escritura
conserva los archivos completos anteriores en esa carpeta y termina con código
1; nunca reemplaza una salida existente. El resumen solo contiene estado y
huellas. La preparación produce propuestas: los dos administradores crean y
publican el plan mediante `/api/admin/perfiles/v1/plan-firma/gobierno`, con
sus autorizaciones y recibos reales.

`vec-plan-firma-validar` comprueba un fichero preparado para el kit privado de
gobierno del plan de firma CT. Recibe la ruta local y el SHA-256 esperado de
los bytes exactos:

```sh
go run ./cmd/vec-plan-firma-validar /ruta/privada/material.json <sha256-esperado>
```

El submodo `preparar` genera ese fichero a partir de una configuración pequeña y
los tres ficheros canónicos ya suministrados por su fuente. Recibe, en este orden,
configuración, catálogo, traza, evento y fichero de salida:

```sh
go run ./cmd/vec-plan-firma-validar preparar /ruta/privada/config.json \
  /ruta/privada/catalogo.json /ruta/privada/traza.json \
  /ruta/privada/evento.json /ruta/privada/material.json
```

La configuración contiene exactamente estas cuatro claves. `huella_esperada`
es `null` para `crear` y el SHA-256 anterior para las demás operaciones:

```json
{
  "operacion": "publicar",
  "revision_esperada": 1,
  "huella_esperada": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "clave_operacion": "clave-sintetica-0001"
}
```

La herramienta calcula los SHA-256 y envuelve los bytes originales en base64.
Comprueba el paquete con el mismo validador antes de guardarlo. El catálogo, la
traza y el evento deben venir completos y ser coherentes entre sí: conserva sus
actores, fechas y decisiones, sin crear otros ni volver a serializar los bloques.
Con las mismas entradas y configuración produce los mismos bytes y SHA-256.

La carpeta de salida debe existir y ser privada, sin permisos para grupo u otros.
El fichero se crea con permisos `0600`; si el destino ya existe, lo rechaza sin
modificarlo. Sincroniza el fichero y coteja su relectura antes de emitir el resumen
JSON con el SHA-256 y `preparado_sin_autorizacion`. Ante un error posterior a la
creación elimina la salida incompleta. Un rechazo no emite un resumen favorable.

Solo admite entradas regulares; rechaza enlaces simbólicos en el fichero de
entrada. Los límites son 16 KiB para la configuración, 2 MiB para el catálogo y
64 KiB para cada traza y evento. Las rutas y el contenido privado quedan fuera
del resumen y del registro de error.

Con material válido emite un JSON con SHA-256, operación, ID del catálogo,
versión, revisión y `material_validado_sin_autorizacion`. El estado solo
acredita una comprobación local del fichero. La herramienta no publica el
catálogo, no obtiene permisos, no consume una decisión V3 y no registra
auditoría ni outbox.

El fichero tiene un límite de 4 MiB. Sus trece claves y tipos deben coincidir
exactamente con el contrato; se rechazan variantes de mayúsculas, valores
`null` indebidos y claves repetidas. Las trazas y eventos deben conservar los
bytes canónicos del modelo común. La comprobación nunca sustituye los bytes
del fichero por una representación nueva.

Guarde el fichero original con acceso privado para repetir la operación con
los mismos bytes y la misma clave. Cada intento de gobierno necesitará una
autorización nueva por el circuito correspondiente. Esta herramienta no
modifica el fichero ni prepara una autorización. Use únicamente datos
sintéticos mientras el circuito no esté admitido.

Un rechazo sale con código 1. Un error de argumentos o escritura del resumen
en la invocación histórica sale con código 2; en `preparar`, un error al guardar
o informar sale con código 1 y limpia el fichero creado. El registro de error indica `material_rechazado` y una etapa
cerrada (`argumentos`, `entrada`, `lectura`, `validacion` o `salida`): no incluye
la ruta, el contenido ni los datos de actor del fichero.
