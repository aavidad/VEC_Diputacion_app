# Capturas anotadas para manuales

`capturar_y_anotar.py` abre un servidor local con Chrome del sistema y guarda una captura a 1440 × 900 y otra a 390 × 844 por pantalla. Cada número dibujado corresponde al texto de una marca en `manifiesto.json`. El fichero incluye dimensiones y SHA-256 de cada PNG. `estado_http` corresponde al documento inicial de cada captura. No guarda URL, cookies, consola ni contenido de las respuestas.

## Comprobar la herramienta

Desde la raíz del repositorio:

```bash
python3 scripts/manuales/capturar_y_anotar.py --ensayo --salida /tmp/vec-capturas-ensayo
python3 -m unittest scripts/tests/test_manual_capturas.py
```

El ensayo levanta una página local mínima y genera dos PNG. Lleva la etiqueta `ensayo-sintetico`: sus imágenes no son capturas de VEC y no deben incluirse en un manual. Necesita Python Playwright, Pillow y `/usr/bin/google-chrome`.

## Capturar un recorrido local

Prepare una instancia local con **datos exclusivamente sintéticos** y confirme ese origen con `--datos-sinteticos-confirmados`. Revise cada pantalla y señale en `ocultar` cualquier dato que deba taparse. El guion tapa también campos de entrada, contenido editable y elementos marcados con `data-private` o `data-sensitive`. Detecta algunos correos, DNI, teléfonos y secretos en el texto visible. No detecta todos los nombres, direcciones ni datos dentro de imágenes, SVG o CSS. Revise **cada PNG** antes de copiarlo a `docs/manuales` o compartirlo. El manifiesto conserva `pendiente_revision_visual: true` y el guion no publica capturas por sí mismo.

Ejemplo de `escenario.json`:

```json
{
  "pantallas": [
    {
      "clave": "peticiones",
      "ruta": "/portal-empleado/peticiones-centro/",
      "exito": "#detalle:not([hidden])",
      "pasos": [
        {"accion": "esperar", "selector": "main h1"},
        {"accion": "rellenar", "selector": "#busqueda", "valor": "ejemplo sintético"},
        {"accion": "clic", "selector": "[data-vista='detalle']"},
        {"accion": "esperar", "selector": "#detalle:visible"}
      ],
      "marcas": [
        {"numero": 1, "selector": "main h1", "texto": "Título de la página", "tipo": "recuadro"},
        {"numero": 2, "selector": "#detalle", "texto": "Detalle de la petición", "tipo": "flecha"}
      ],
      "ocultar": [".datos-personales"]
    }
  ]
}
```

```bash
python3 scripts/manuales/capturar_y_anotar.py \
  --base-url http://127.0.0.1:8081 \
  --escenario escenario.json \
  --salida /tmp/vec-capturas-manual \
  --datos-sinteticos-confirmados
```

`ruta` debe empezar por `/` y no admite parámetros. Si el recorrido cambia de ruta, declare `ruta_final` con la ruta exacta esperada. `exito` señala un elemento propio del estado final; úselo cuando una consulta AJAX actualice la vista. El guion comprueba que la URL final coincida con la ruta prevista y que `exito`, si se declara, esté visible. Los pasos admiten `clic`, `esperar`, `rellenar` y `seleccionar`; los dos últimos requieren `valor` sintético. `seleccionar` usa el valor de una opción HTML. Los selectores son de Playwright. Cada marca admite `recuadro` o `flecha`. Los mismos pasos y selectores se ejecutan en **ambos tamaños**; los elementos marcados deben quedar visibles y destapados en los dos.

Si el servidor local exige certificado de cliente, añada `--mtls-certificado /ruta/externa/cliente.crt --mtls-clave /ruta/externa/cliente.key`. Los archivos deben estar fuera de todos los worktrees Git. Si la clave tiene frase, expóngala en una variable del proceso y pase solo su nombre mediante `--mtls-frase-env NOMBRE_VARIABLE`. El guion no copia esos archivos ni guarda sus rutas o la frase en el manifiesto. Los recursos internos con `?v=` se cargan sin registrar sus parámetros.

La navegación se limita al mismo origen de loopback. El guion corta las redirecciones HTTP y los WebSocket. Chrome usa un contexto nuevo por captura, sin perfil ni almacenamiento persistente. La salida debe estar fuera de todos los worktrees Git, en un directorio vacío y privado (0700); los PNG y el manifiesto se crean con permiso 0600. Un clic o un formulario pueden activar una escritura: use una instancia sintética y configure solo las acciones necesarias para el recorrido. Compruebe en la aplicación el resultado real de cada paso antes de describirlo en el manual; una captura por sí sola no acredita una operación ni su persistencia.
