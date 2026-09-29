# Recorrido del centro a RRHH

Este guion comprueba con Chrome del sistema una petición sintética presentada
por el centro, su ratificación por otra identidad y la creación del expediente
en RRHH. Compara la referencia del expediente, el número visible, el recibo y
la fecha en un replay y después de reiniciar aplicación y PostgreSQL. La
petición queda en el clon: ejecute el recorrido una sola vez por caso preparado.

## Preparación privada

Use un **clon sintético aislado** con H3, H4 y H5 instalados y un binario apto
para ese esquema. Entregue fuera de Git un JSON de acreditación como este:

```json
{
  "tipo": "clon_aislado_sintetico",
  "origen": "https://127.0.0.1:8443",
  "clon_id": "identificador-local-del-clon",
  "h3_instalado": true,
  "h4_instalado": true,
  "h5_instalado": true,
  "binario_sha256": "SHA256_HEX_DEL_BINARIO"
}
```

La acreditación es una declaración del responsable del clon. El guion coteja
el SHA256 del binario y comprueba HTTP y los recibos, pero no inspecciona por
sí mismo el historial de migraciones. Prepare también tres certificados y
claves mTLS distintos, autorizados respectivamente para solicitante,
ratificador y RRHH. Chrome debe confiar en la CA del servidor mediante el
almacén de confianza del sistema. Nunca copie certificados, claves, DSN o
datos personales al repositorio.

El ejecutable indicado en `--reinicio` reinicia **solo** la aplicación y
PostgreSQL de ese clon. Debe terminar cuando ambos vuelvan a estar disponibles.
Su salida se descarta para evitar datos privados en el registro. El guion lo
invoca después de obtener el recibo inicial y un replay idéntico.

```sh
python3 scripts/recorridos/centro_rrhh/recorrer.py \
  --origen https://127.0.0.1:8443 \
  --acreditacion /ruta/privada/acreditacion.json \
  --binario /ruta/privada/vec-server \
  --reinicio /ruta/privada/reiniciar-clon \
  --cert-solicitante /ruta/privada/solicitante.crt \
  --clave-solicitante /ruta/privada/solicitante.key \
  --cert-ratificador /ruta/privada/ratificador.crt \
  --clave-ratificador /ruta/privada/ratificador.key \
  --cert-rrhh /ruta/privada/rrhh.crt \
  --clave-rrhh /ruta/privada/rrhh.key
```

Salida `NO EJECUTADO` y código 2 significan que falta una condición previa:
no se abre Chrome ni se escribe en la base. Salida `FALLÓ` y código 1 indican
que el recorrido empezó y que su estado requiere inspección del clon antes de
repetir una escritura incierta. Salida `EJECUTADO` y código 0 exige respuestas
reales, la ratificación v2 por otra identidad, una sola fila visible de entrega tras
el reinicio y el mismo recibo completo en ambos replays. El guion no acredita
firma legal, envío externo ni otro paso completo de Contratación temporal.

Prueba focal de la lógica sin navegador ni base:

```sh
python3 -m unittest discover -s scripts/recorridos/centro_rrhh -p 'test_*.py'
```
