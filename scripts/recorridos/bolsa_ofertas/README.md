# Recorrido local de política, oferta y respuesta en Bolsa

El guion recorre con Chrome del sistema dos identidades sintéticas por mTLS:
RRHH publica la política de una bolsa reservada, publica una oferta y consulta
sus avisos; la persona ve la oferta en «Mi Bolsa» y se ofrece. Guarda recibos y
fechas fuera de Git. Tras reiniciar VEC y PostgreSQL del **mismo clon**, la fase
`recuperar` comprueba que la oferta y la disposición conservan sus referencias
y que cada una aparece una sola vez. No repite ningún POST en esa fase.

La emisión masiva B7 es una operación aparte. Con `emitir_b7: true`, el guion
elige personas elegibles en el orden visible, exige al menos dos y llega a la
revisión. Si el botón está bloqueado por segunda revisión, registra ese punto
de corte. Si puede emitir, exige el recibo de la API y registra únicamente que
VEC aceptó la emisión. El total seleccionado procede del estado global de B7,
incluidas las páginas no visibles. Una oferta publicada no envía correos.
Mailpit, el relay y la entrega corporativa quedan sin acreditar por este guion.

## Preparación por Dirección

Se necesita un clon aislado con H3, H4 y H5 instalados, binario exacto de ese
clon, dos certificados sintéticos con concesiones separadas y una bolsa
sintética reservada. La URL debe ser HTTPS de loopback. El guion no arranca
servicios, no instala SQL y no usa la base principal. Los valores y rutas del
JSON privado se conservan fuera de Git. El JSON y ambas claves privadas deben
tener permisos `0600`; el directorio de evidencia, `0700`. Chrome filtra cada
petición al origen exacto: rechaza
orígenes externos, WebSockets y cualquier redirección HTTP sin seguirla.
Los certificados y claves de RRHH y candidato han de tener rutas y huellas
distintas. Ni estos materiales, ni el binario, la configuración o la evidencia
pueden residir en ninguna raíz Git o worktree, aunque se alcance por un enlace.
Ejemplo de estructura:

```json
{
  "origen": "https://127.0.0.1:8443",
  "clon": "aislado_h3_h4_h5",
  "hitos": ["H3", "H4", "H5"],
  "binario": "/RUTA_PRIVADA/vec-server",
  "binario_sha256": "SHA256_HEX_DEL_BINARIO",
  "chrome": "/usr/bin/google-chrome",
  "rrhh": {"certificado": "/RUTA_PRIVADA/rrhh.crt", "clave": "/RUTA_PRIVADA/rrhh.key"},
  "candidato": {"certificado": "/RUTA_PRIVADA/candidato.crt", "clave": "/RUTA_PRIVADA/candidato.key"},
  "bolsa_ref": "bolsa:sintetica-reservada",
  "bolsa_sintetica_reservada": true,
  "municipio_sintetico": "18001",
  "oferta": {
    "categoria": "Auxiliar sintético", "centro": "Centro sintético",
    "fecha_inicio": "2026-10-15", "numero_plazas": 2,
    "descripcion": "Sustitución sintética"
  },
  "emitir_b7": false,
  "reinicio_confirmado": false
}
```

`binario_sha256` se calcula con `sha256sum` sobre el binario que se ha
arrancado. Antes de `recuperar`, Dirección reinicia aplicación y PostgreSQL,
verifica salud del clon y pone `reinicio_confirmado: true` en el JSON privado.

```bash
python3 scripts/recorridos/bolsa_ofertas/recorrido.py \
  --config /RUTA_PRIVADA/bolsa-ofertas.json --fase alta \
  --evidencia /RUTA_PRIVADA/bolsa-ofertas-evidencia.json
python3 scripts/recorridos/bolsa_ofertas/recorrido.py \
  --config /RUTA_PRIVADA/bolsa-ofertas.json --fase recuperar \
  --evidencia /RUTA_PRIVADA/bolsa-ofertas-evidencia.json
```

Si falta clon, binario o material mTLS, devuelve `NO EJECUTADO` (código 2)
sin abrir Chrome. En `alta` crea primero una marca privada `INICIADO` y conserva cada recibo
a medida que avanza, limitados a referencias, fechas, versión y estado.
Registra método, ruta y estado HTTP sin copiar consultas
ni cuerpos de peticiones. Guarda capturas a 1440 y 390 px en un directorio
privado junto a la evidencia, con PNG de permisos `0600`, creados sin seguir
enlaces ni sobrescribir archivos. Si falla, captura la última pantalla alcanzada.
Si falla después de alguna escritura, no se debe repetir el guion sobre el mismo
clon: Dirección reconcilia sus recibos o prepara otro clon limpio. `FALLÓ`
(código 1) indica un fallo real del recorrido. La configuración, la evidencia,
las capturas y los certificados se guardan fuera del repositorio.

La comprobación sintética sin servicios es:

```bash
python3 -m unittest discover -s scripts/recorridos/bolsa_ofertas -p 'test_*.py'
```

La prueba de red usa dos servidores efímeros en loopback: el primero responde
`302` hacia otro puerto y el segundo debe recibir **cero** peticiones. No
contacta VEC, Mailpit ni servicios externos.

En este corte no se dispone del clon H3–H5 ni de su binario; el recorrido real
queda **NO EJECUTADO**. La prueba sintética solo acredita las guardas de
entrada, no navegación, SQL, recibos ni recuperación.
