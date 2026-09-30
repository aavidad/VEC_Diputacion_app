# Recorrido de análisis a informe jurídico

Este guion recorre en Chrome el expediente sintético existente de RRHH. Abre el
cuadro y el detalle autorizados, registra las operaciones que aún procedan y
comprueba sus recibos al registrarlas. Tras reiniciar, compara la versión y el
número de hitos que muestra el detalle; no reconsulta esos recibos. La decisión de vía
de cobertura es un paso necesario entre análisis y asignación. Si falta el
formulario, la concesión o una fuente gobernada, se detiene en esa puerta.
Todas las peticiones quedan en el origen HTTPS local declarado; el filtro de
Playwright inspecciona cada respuesta sin seguir redirecciones y corta también
un salto a otro puerto local. Los WebSocket se cierran antes de conectar.

## Preparación externa

Necesita un clon local H3–H5 con aplicación y PostgreSQL, dos certificados mTLS
sintéticos (uno con concesión RRHH para el expediente y otro sin ella), Chrome
del sistema y Playwright Python. El guion no crea datos ni arranca servicios.
El responsable del clon entrega fuera de Git un manifiesto JSON con esta forma:

```json
{
  "tipo": "clon_local_h3_h5",
  "sintetico": true,
  "origen": "https://127.0.0.1:8443",
  "expediente_ref": "expediente:ejemplo-sintetico",
  "hitos": ["H3", "H4", "H5"],
  "commit_binario": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "sha256_binario": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}
```

`commit_binario` identifica el árbol empleado para compilar el servidor. El
guion compara el SHA256 del ejecutable que se le entrega con el manifiesto;
el responsable debe acreditar que ese ejecutable es el que sirve la URL. Use la
referencia exacta de un expediente visible en la primera página autorizada del
cuadro. El archivo de datos, también externo a Git, contiene `analisis` con los
diez campos que pide el formulario y `via_cobertura` y `motivo_cobertura` tomados
de la propuesta gobernada del propio clon. No copie datos personales reales.

```json
{
  "analisis": {
    "modalidad_clave": "CLAVE_PUBLICADA",
    "categoria_ref": "categoria:ejemplo-sintetico",
    "grupo_subgrupo": "CLAVE_PUBLICADA",
    "causa_clave": "CLAVE_PUBLICADA",
    "inicio": "2026-10-01",
    "fin": "2026-10-31",
    "jornada_horas": "35",
    "jornada_minutos": "0",
    "entrada_rc_referencia": "rc:ejemplo-sintetico",
    "observaciones": "Ejercicio sintético de RRHH."
  },
  "via_cobertura": "CLAVE_PUBLICADA",
  "motivo_cobertura": ""
}
```

Las claves en mayúsculas son marcadores: sustitúyalas por valores publicados
en el clon. El motivo vacío solo sirve si la vía elegida no es alternativa.

```bash
python3 scripts/recorridos/analisis_informe/recorrer.py \
  --origen https://127.0.0.1:8443 \
  --expediente-ref expediente:ejemplo-sintetico \
  --certificado /ruta/privada/rrhh.pem --clave /ruta/privada/rrhh.key \
  --certificado-denegado /ruta/privada/sin-concesion.pem \
  --clave-denegada /ruta/privada/sin-concesion.key \
  --clon /ruta/privada/clon-h3-h5.json --binario /ruta/privada/vec-server \
  --datos /ruta/privada/datos-analisis.json --modo registrar --efectos \
  > /ruta/privada/resultado-antes.json
```

Después de reiniciar **aplicación y PostgreSQL del mismo clon**, compruebe la
recuperación sin escrituras:

```bash
python3 scripts/recorridos/analisis_informe/recorrer.py \
  --origen https://127.0.0.1:8443 \
  --expediente-ref expediente:ejemplo-sintetico \
  --certificado /ruta/privada/rrhh.pem --clave /ruta/privada/rrhh.key \
  --certificado-denegado /ruta/privada/sin-concesion.pem \
  --clave-denegada /ruta/privada/sin-concesion.key \
  --clon /ruta/privada/clon-h3-h5.json --binario /ruta/privada/vec-server \
  --modo recuperar \
  --esperado /ruta/privada/resultado-antes.json
```

El resultado inicial conserva referencias opacas de recibo, versión y fecha; la
recuperación compara versión y número de hitos en escritorio y móvil. El detalle
RRHH no expone de nuevo todos los recibos originales: esa igualdad exige una
consulta autorizada específica o un replay con las claves originales. Por ello,
el guion **no acredita identidad del recibo tras reinicio**. Tampoco acredita
firma, entrega, fiscalización ni publicación. La PR 161 se evalúa por su hash
exacto y no forma parte de `main` por el hecho de existir.

La salida `NO EJECUTADO` indica que falta una entrada o el clon acreditado; la
salida `CORTADO` indica la primera puerta que falló. No reenvíe un POST tras un
fallo indeterminado: conserve su clave y use el circuito de recuperación propio.
