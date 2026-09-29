# Guion supervisado del llamamiento

Este guion observa en Chrome del sistema un ejercicio con datos sintéticos. Abre
dos contextos mTLS separados: RRHH en el portal interno y la persona candidata
en Mi bolsa. El operador realiza las acciones en las pantallas; el programa
comprueba un solo POST por etapa, su estado y los campos estables del recibo.
No crea peticiones de efecto por API.

## Punto de corte

En `origin/main` `3b910a170`, el formulario de Contratación ofrece selección,
aviso local, declaración de RRHH, resolución manual y continuación tras renuncia.
Mi bolsa ofrece una respuesta propia con certificado de candidato. Ese recibo
de Bolsa no se convierte por sí solo en declaración ni resolución de
Contratación. El aviso local y la resolución manual sintética no acreditan
entrega, plazo legal, firma ni incorporación. Si la respuesta es aceptación,
el guion termina en la resolución; no abre otro llamamiento. Si es renuncia,
exige una continuación expresa y un único recibo del siguiente llamamiento.

No hay aún un servidor local con clon H3–H5 y binario acreditados para este
recorrido. El clon HITO1 de `127.0.0.1:55441` y los servicios de otros trabajos
quedan fuera. El estado actual del recorrido es **NO EJECUTADO**.

## Entradas que prepara dirección fuera de Git

- Clon PostgreSQL aislado con H3, H4 y H5 instalados, datos sintéticos y
  manifiesto local `clon.json`. Dirección debe verificar la instalación real:
  el manifiesto es una puerta de entrada, no una prueba SQL.
- Binario ejecutable exacto del servidor y su SHA256, idéntico al del manifiesto.
  El servidor debe estar arrancado en HTTPS loopback con la composición real.
- Dos certificados y claves mTLS sintéticos diferentes, uno para RRHH y otro
  para la persona candidata. Sus rutas absolutas permanecen fuera del repositorio.
- Un `escenario.json` externo con origen, rutas de ambos portales, bolsa,
  respuesta y recibos esperados. Prepare una sola rama: `aceptacion` o
  `renuncia`. Incluya en `campos_recibo` la referencia y fecha exactas; añada
  versión cuando exista. Para `candidato_respuesta`, use `recibo` y
  `registrada_en`, que se compararán luego con el GET propio de Mi bolsa.

La secuencia de `etapas` para renuncia es `seleccion`, `comunicacion`,
`candidato_respuesta`, `declaracion_rrhh`, `resolucion_rrhh`, `siguiente`.
Para aceptación se omite `siguiente`. En cada etapa, por ejemplo:

```json
{"nombre":"candidato_respuesta","campos_recibo":{"recibo":"recibo:sintetico","registrada_en":"2026-09-29T00:00:00.000000Z"}}
```

El ejemplo muestra la forma del JSON; no es un recibo para usar. Las claves de
idempotencia, expediente, correo original y justificación se preparan y revisan
fuera de Git. No se copian al guion ni a sus logs. La declaración de RRHH
requiere su evidencia original y sus dos revisiones expresas; el programa no
la deriva del recibo de Mi bolsa.

El manifiesto externo tiene esta estructura. La huella procede del binario
arrancado, tras verificar el clon aislado y sus migraciones:

```json
{"origen":"https://127.0.0.1:PUERTO","clon":"H3-H5","hitos_instalados":["H3","H4","H5"],"datos":"sinteticos","listo":true,"binario_sha256":"SHA256_REAL"}
```

El escenario usa los mismos `origen` y `binario_sha256`, más `clon`,
`manifiesto_clon`, `binario`, `bolsa_ref`, `respuesta`, `identidades` y `etapas`.
Cada identidad lleva `certificado`, `clave` y `ruta`: la de RRHH empieza por
`/portal-empleado/`, y la candidata por `/area-personal/`. Todas las rutas de
archivos son absolutas y externas al repositorio.

## Ejecución

```sh
python3 scripts/recorridos/llamamiento/recorrido.py --escenario /ruta/privada/escenario.json
python3 scripts/recorridos/llamamiento/recorrido.py --escenario /ruta/privada/escenario.json --ejecutar
# Tras reiniciar aplicación y PostgreSQL del clon aislado:
python3 scripts/recorridos/llamamiento/recorrido.py --escenario /ruta/privada/escenario.json --recuperar
```

La primera orden solo comprueba entradas. Sin H3–H5, binario, certificados o
recibos esperados, termina con código `3` y **NO EJECUTADO**, antes de abrir
Chrome. No apunta por defecto a ningún servicio. La segunda abre las dos
pestañas: complete cada paso una sola vez y pulse Intro después de ver el
recibo. Un resultado incierto detiene el guion; investigue y recupere con la
misma clave, sin crear otra operación. El programa nunca pulsa Enviar por usted.

Tras reiniciar el servidor y PostgreSQL aislados, `--recuperar` abre contextos
nuevos. Comprueba por GET propio el recibo y la fecha de la persona candidata;
RRHH recupera las operaciones originales con las mismas claves y debe obtener
POST `200` y los mismos campos estables. No afirma ausencia de duplicados en
tablas ni auditoría SQL: dirección debe contrastar historia, outbox y recuentos
en el clon antes de llamar cerrado al recorrido. Las dos pasadas comprueban
cookies, almacenamiento web, errores JavaScript y desbordamiento a 1440 y 390 px.

Prueba focal sin navegador ni servicios:

```sh
python3 -m unittest discover -s scripts/recorridos/llamamiento -p 'test_*.py'
```
