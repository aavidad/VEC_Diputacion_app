# Preparar los identificadores originales de ADMIN

`vec-admin` necesita un archivo privado con los identificadores originales de
cada persona administradora (`fuente_identificadores_archivo` y su SHA256 en
`VEC_ADMIN_RUNTIME_CONFIG_FILE`). Este comando lo genera a partir de lo que ya
dejó el arranque de fuentes iniciales. No conecta con PostgreSQL ni muestra
identificadores.

Entradas, todas archivos propios 0600 en un directorio 0700, sin enlaces y
fuera de cualquier repositorio:

- `--originales`: `originales.json` del arranque.
- `--certificados`: `certificados.json` (huellas del certificado y de la CA).
- `--fuente`: `fuente.json`, el plan de fuentes iniciales.
- `--acuse`: el acuse confirmado de `vec-aplicar-fuentes-admin`, de donde salen
  las cuentas `cta_`.
- `--material`: `material-hmac.raw`, el material HMAC confirmado.
- `--proveedor`: la configuración del proveedor de seudónimos.
- `--configuracion-admin` (opcional, recomendado): la configuración privada de
  `vec-admin`. Se lee solo su bloque `identidad` y se exige que coincida con
  `--proveedor` en directorio, configuración HMAC, espacio, dominio, espacio de
  clave y dominio HMAC. Así se comprueba lo que usará de verdad el arranque.

Antes de escribir, el comando coteja que todas las entradas son de la misma
preparación y recalcula las huellas de sujeto, cuenta y cuenta ordinaria con la
misma función y proveedor que `vec-admin`. Si alguna no coincide con el
material confirmado, se detiene con su código y no escribe nada.

La salida (`--salida`) se crea 0600, sin sobrescribir, en un directorio 0700
fuera de Git. Después se carga con el mismo cargador del arranque; si no la
acepta, se retira y el mensaje dice si se pudo retirar. El comando imprime solo
un código, su mensaje y `identificadores_sha256`, que es la huella a aprobar.

```sh
go build -o /ruta/privada/vec-preparar-identificadores-admin ./cmd/vec-preparar-identificadores-admin
/ruta/privada/vec-preparar-identificadores-admin \
  --originales /ruta/privada/fuentes/originales.json \
  --certificados /ruta/privada/fuentes/certificados.json \
  --fuente /ruta/privada/fuentes/fuente.json \
  --acuse /ruta/privada/fuentes/acuse-primero-resguardado.json \
  --material /ruta/privada/fuentes/material-hmac.raw \
  --proveedor /ruta/privada/fuentes/proveedor.json \
  --configuracion-admin /ruta/privada/vec-admin/configuracion.json \
  --salida /ruta/privada/identificadores/identificadores.json \
  --textos web/static/textos/es/admin-identificadores-preparar.json
```

Los textos están en `web/static/textos/{es,en}/admin-identificadores-preparar.json`.
