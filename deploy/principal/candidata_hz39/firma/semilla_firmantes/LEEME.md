# Firmantes de una copia local

Este generador prepara los datos ficticios de cuatro firmantes y sus
dieciséis perfiles. No conecta con PostgreSQL ni envía peticiones. La
publicación de cargos, certificados y plan de firma sigue los actos del
[recorrido de firma](../LEEME.md).

Se publican las cinco fuentes que usó la preparación privada de la candidata
de F1-27: el generador y sus cuatro plantillas. No se incluyen certificados,
claves, conexiones, aprobaciones ni recibos.

## Antes de generar

Usa una copia propia y material privado fuera de Git. Necesitas:

- El recibo de alta de la unidad administrativa de esa copia.
- Las referencias de organización, procedencia, unidad y nodo que devuelve.
- La clave de identidad de esa copia, de 32 bytes.
- Los cuatro certificados de desarrollo, uno por persona.

La configuración y las entradas privadas deben pertenecer a la persona que
ejecuta el generador y tener permisos `0600`. Las plantillas versionadas se
leen desde su carpeta; no es necesario cambiar los permisos del checkout.
La salida debe ser una carpeta absoluta nueva, fuera de Git.

## Dos fases

Primero prepara identidades y la fuente de los dieciséis roles:

```sh
python3 deploy/principal/candidata_hz39/firma/semilla_firmantes/generar_semilla_p.py \
  --config "$VEC_FIRMANTES_CONFIG" --salida "$VEC_FIRMANTES_INICIAL"
```

El archivo de configuración conserva el contrato del generador privado:
`version`, `copia_ref`, `organizacion_canonica_ref`,
`organizacion_procedencia_ref`, `organizacion_procedencia_sha256`,
`organizacion_operativa_ref`, `unidad_ref`, `nodo_ref`,
`hmac_g2_clave_archivo`, `certificados` y `acuse_unidad_h10`.
`certificados` contiene las rutas de `tecnico`, `jefatura`, `diputacion` y
`secretaria`. Todos los valores corresponden a la copia propia.

Después publica los roles por la herramienta y aprobación existentes, como
describe la guía de firma. Con el plan y su recibo auténticos, genera una
segunda salida:

```sh
python3 deploy/principal/candidata_hz39/firma/semilla_firmantes/generar_semilla_p.py \
  --config "$VEC_FIRMANTES_CONFIG" --salida "$VEC_FIRMANTES_FINAL" \
  --roles-plan "$VEC_FIRMANTES_PLAN" \
  --roles-recibo "$VEC_FIRMANTES_RECIBO" \
  --roles-fuente-publicada "$VEC_FIRMANTES_FUENTE"
```

Sin esos tres archivos no se genera el SQL de asignaciones. El generador
coteja el plan, su fuente y el recibo; no los fabrica. Tampoco ejecuta los
archivos producidos. Revisa su destino antes de aplicar datos en la copia.
No los apliques en producción ni sobre identidades ya instaladas.

La recuperación exige fechas UTC canónicas y un intervalo de entre 60 y
90 días. Ambas fases rechazan una retirada de firmantes ya vencida, incluso
si todavía no se han aportado los certificados.

## Alcance de la entrega

Esto permite repetir la preparación de quienes firman desde la ficha de
Contratación temporal. No registra firmas, no publica permisos por sí solo y
no acredita los pasos 12 a 15 de F1-27. El preparador del plan de firma se
entrega por separado dentro de F1-121.

Las cuatro plantillas conservan sus referencias de personas ficticias y las
fechas del kit: identidad y política hasta el 18/11/2027 a las 16:06:24 UTC;
retirada de los firmantes el 14/01/2027 a las 21:00 UTC. Son datos de esta
candidata de octubre de 2026. Revisa esas fechas antes de otra preparación;
no prolongues autoridad histórica editando recibos.

La propuesta nueva de roles abarca 90 días. Al recuperar una publicación se
conservan sus fechas y huellas originales. Las asignaciones deben quedar
cubiertas por la admisión y los certificados, como exige la guía de firma.
La decisión de dirección del 20/10 mantiene los certificados de desarrollo
del kit y limita la autoridad, admisiones y asignaciones a unos 90 días.
