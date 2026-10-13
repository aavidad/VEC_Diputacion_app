# Ensayo de la candidata HZ39

`recorrer.py` recorre las pantallas del caso único de `FLUJO_APP.md`, apartado
4.0. El destino y los certificados son parámetros. Puede apuntar a una copia
local o a una segunda instancia autorizada por dirección a través de un túnel
local. No accede directamente a cidonia.

La preparación del guion no demuestra que la candidata arranque ni que termine
el caso. Esos resultados se obtienen al ejecutarlo sobre el destino congelado
con su configuración, datos y actos ADMIN. El material privado lo entrega
dirección; no se obtiene del servidor desde este guion.

Reutiliza la captura y auditoría de `scripts/revision_web`, el registro HTTP de
`recorridos/convoca_integrado.py`, los ficheros privados y filtro sin redirección
de `recorridos/intervencion/recorrer.py`, y las acciones/selectores de los tramos
de centro, análisis, llamamiento, Intervención, firma y B2 existentes.

## Entradas privadas

Python y Playwright ya instalados, `/usr/bin/google-chrome` y OpenSSL. Chrome
confía en la CA del destino por el almacén preparado por dirección. La validación
TLS permanece activa. No hay respuesta simulada, cookie ni estado web guardado.

- `--base-url`: origen HTTPS de loopback con puerto, sin credenciales ni ruta;
  si se usa un túnel, dirección fija antes qué instancia hay detrás.
- `--certificados`: carpeta de P12, sólo lectura. Nombres del juego actual:
  `candidato.p12`, `centro-solicitante.p12`, `centro-ratificador.p12`,
  `cliente.p12`, `intervencion.p12`.
- `--claves`: JSON privado `0600` fuera de Git. Las claves son `aspirante`,
  `centro`, `ratificador`, `rrhh` e `intervencion`; sus valores son las
  contraseñas P12 entregadas por dirección. No ponerlas en argumentos ni Git.
  El guion comprueba los certificados públicos y rechaza personas repetidas.
- `--datos`: JSON privado `0600` con `candidata` (hash exacto), `valores` y
  `extraer`. Los valores contienen el MOAD, categoría/puesto/centro publicados,
  fechas, RC, motivo, bolsa, datos de mérito, archivos aportados y opciones B2
  de ese caso. K proporciona la selección y receta de los datos existentes.
- `extraer`: lista de `{metodo,ruta,campo,variable}`. Coteja la ruta observada y
  copia una referencia real del JSON de respuesta, por ejemplo
  `campo: data.expediente_ref` a `variable: expediente_ref`. Las referencias de
  petición, solicitud e incorporación se obtienen de sus respuestas reales;
  no se inventan ni se supone que una ruta haya creado un expediente.
- `--salida`: carpeta nueva fuera de Git, con padre privado `0700`.
  Cada ejecución tiene su salida; las anteriores no se sobrescriben.

`pasos_hz39.json` mantiene el caso aprobado: cinco pasos externos y los pasos
1 a 21, con 7b (firma de remisión) y 16a → 17 → 16b (descarga, incorporación,
confirmación GINPIX). Los datos que falten se ajustan por los guiones/semillas
existentes en la copia; este guion no fabrica filas ni otra implementación.

## Uso

```sh
umask 077
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_NUEVA" --modo comprobar

# Sólo después de que dirección fije destino, candidata y material:
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_NUEVA" --modo ejecutar --visible --video
```

`comprobar` no abre Chrome ni conecta: revisa entradas y enumera lo disponible.
`--pasos e1,e2,e3` limita una observación; su resultado nunca cierra el caso
completo. Cada paso ejecutado guarda HTTP, tiempo, dueño, recibos y capturas de
1440/390. Los vídeos son privados y se graban por contexto/certificado. Las
contraseñas, cuerpos y cabeceras de identidad no van a la evidencia.

Si falla un paso, queda con su causa y dueño; los dependientes quedan pendientes,
y los independientes continúan. Una firma por H se observa en el panel y en el
estado real de firmas. 7b exige informe verificado; 15 exige informe/remisión y
resolución de tres cargos. Los otros cuatro documentos se descargan y comparan,
con firma pendiente, según la decisión de dirección del 14/10 a las 20:55. No se
considera una firma completada porque se abra el panel o el operador lo afirme.

`continuar` conserva los pasos que ya terminaron. Un acto previo iniciado pero
sin paso cerrado se bloquea: primero se recupera el resultado original, nunca
se pulsa otra vez con una clave nueva para superar una respuesta perdida.

## Reinicio y recuperación

Dirección o quien posee la copia reinicia únicamente sus contenedores. Este
guion no reinicia servicios ni escribe SQL. Se mantiene la misma candidata y
los mismos datos; el origen debe seguir apuntando a la copia acordada.

```sh
python3 deploy/principal/ensayo_candidata/recorrer.py \
  --base-url "$URL_COPIA" --certificados "$CARPETA_P12" \
  --claves "$CLAVES_JSON_PRIVADAS" --datos "$DATOS_JSON_PRIVADOS" \
  --salida "$SALIDA_RECUPERACION_NUEVA" --modo recuperar \
  --anterior "$RESULTADO_ORIGINAL_JSON" --reinicio-acreditado
```

La recuperación usa pantallas y lecturas de la proyección. Bloquea las peticiones
con efecto antes de enviarlas; permite los POST de consulta declarados. Compara
referencias de recibo y huellas de descargas con la ejecución anterior. Si una
pantalla no permite recuperar un recibo, se informa y no se acredita ese paso.
La opción `--reinicio-acreditado` declara la actuación del dueño de la copia;
por sí sola no demuestra el reinicio.

Salida: `resultado.json`, `paso-*.json`, intenciones minimizadas antes de los
POST, capturas, documentos y vídeo. Sólo `completo: true` con todos los pasos
OK acredita el recorrido del guion; parcial o pendiente devuelve código 1.
Una precondición inválida devuelve código 2 sin atribuir una ejecución válida.

La focal de guardas del guion se ejecuta con:

```sh
python3 -m unittest discover -s deploy/principal/ensayo_candidata -p test_recorrer.py -v
```

No sustituye el ensayo único de la aplicación real congelada.
