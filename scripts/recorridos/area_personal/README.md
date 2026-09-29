# Recorrido del área personal

Este guion prepara un recorrido con Chrome del sistema y datos sintéticos: **Mis preferencias → Mis correos verificados → Mi imagen → Mi bolsa → Mi ficha Aspirantes**. Al primer fallo devuelve el paso exacto y deja los pasos posteriores sin acreditar. Una vista genérica de «Perfil y contacto» no cuenta como «Mi ficha Aspirantes».

## Antes de ejecutarlo

Dirección debe montar un clon aislado con H3, H4 y H5, arrancar el binario de ese mismo corte y disponer de una identidad sintética con concesiones propias. El acta de preparación, binario, certificado y clave mTLS se guardan **fuera del repositorio**. No se conecta a servicios de otros agentes ni a cidonia.

El acta JSON contiene solo metadatos de preparación, sin DSN, claves ni datos personales:

```json
{
  "origen": "https://127.0.0.1:PUERTO",
  "clon_sintetico": true,
  "clon_ref": "referencia-opaca-del-clon",
  "hitos_verificados": ["H3", "H4", "H5"],
  "binario": "/ruta/externa/vec-server",
  "binario_sha256": "SHA256_HEX_DEL_BINARIO"
}
```

La puerta comprueba formato, existencia y huella del binario. El acta documenta la comprobación previa del clon; **el guion no verifica por sí solo migraciones, permisos o restauración de PostgreSQL**. Si falta cualquiera de estas precondiciones, devuelve `NO EJECUTADO` y código 2 antes de abrir Chrome. El material mTLS debe pertenecer a la identidad sintética autorizada.

```bash
python3 scripts/recorridos/area_personal/recorrer.py \
  --origen https://127.0.0.1:PUERTO \
  --acta /ruta/externa/acta.json \
  --certificado /ruta/externa/certificado.pem \
  --clave /ruta/externa/clave.pem
```

## Evidencia y límites

El guion fija el castellano, cambia el número de filas de preferencias y elige otra paleta de imagen del catálogo. Comprueba sus recibos y la recuperación mediante un nuevo GET tras recargar. «Mis correos» exige un correo **ya verificado** por su circuito propio: no intercepta correo, no inventa código y no interpreta un aviso de envío como entrega. «Mi bolsa» exige sus dos GET autorizados; la ficha requiere un acceso propio visible. El resultado solo imprime la huella SHA256 de cada referencia de recibo, versión y estados de paso. No imprime correos, identidad, cuerpos HTTP, certificados, claves ni URL con datos.

La recuperación aquí es **recarga de navegador**. No acredita recuperación tras reiniciar aplicación o PostgreSQL, auditoría completa, entrega de correo, ni funciones administrativas de Aspirantes. Esas afirmaciones requieren un recorrido adicional en el clon, con inspección de historia y recibos. El guion comprueba ausencia de cookies, almacenamiento web, errores JavaScript, peticiones externas y desbordamiento horizontal en la vista final alcanzada.

En `origin/main` de partida (`3b910a1`), «Mi ficha Aspirantes» de #160 seguía abierta. Si todas las etapas anteriores llegan a ella, el primer corte esperado es `mi_ficha_aspirantes`; **no se atribuye #160 a main**. Hoy falta además el runtime H3–H5 con binario acreditado para este ensayo, por lo que el estado real de esta rama es `NO EJECUTADO`.

Prueba focal sin servicios:

```bash
python3 -m unittest discover -s scripts/recorridos/area_personal -p 'test_*.py'
```
