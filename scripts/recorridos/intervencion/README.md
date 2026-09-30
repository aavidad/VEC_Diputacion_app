# Recorrido sintético de fiscalización y reparo

Este primer corte prepara dos expedientes distintos para el paso de RRHH de fiscalización. Intervención registra un resultado favorable en el primero y un reparo en el segundo. RRHH registra la subsanación del reparo. El guion observa los tres POST del navegador, sus estados `201`, las versiones y los recibos. Comprueba ambos perfiles a 1440 y 390 px, sin desbordamiento, cookies, almacenamiento web ni errores JavaScript. Después de reiniciar aplicación y PostgreSQL, repite cada petición con su clave original; exige `201` tanto para fiscalización como para subsanación, además de los mismos recibos, fechas y referencias de auditoría y evento. Consulta también la versión y la secuencia de actuaciones de ambos expedientes.

El guion no crea expedientes, instala migraciones, inicia servicios ni reinicia contenedores. Necesita dos expedientes sintéticos preparados en la fase de informe jurídico, sus versiones exactas, el clon local con H3–H5 instalados, un binario VEC identificado por SHA256 y dos identidades mTLS sintéticas distintas. La configuración y la evidencia se guardan fuera del árbol Git. El origen solo admite HTTPS loopback; el contexto del navegador bloquea peticiones de páginas y ventanas emergentes a otros orígenes, redirecciones y canales WebSocket.

El modo `registrar` conserva capturas de cada resultado y del primer fallo a 1440 y 390 px. Las imágenes quedan junto a la evidencia, en una carpeta privada con permiso `0700`; cada PNG tiene permiso `0600` y huella SHA256. Antes de abrir Chrome, comprueba que las salidas están fuera de cualquier repositorio Git, worktree o repositorio bare, sin enlaces y con padre propio `0700`. Los archivos se crean mediante el descriptor del directorio, con `O_EXCL`, `O_NOFOLLOW` y permiso `0600`; ninguno existente se sobrescribe. Registra método, ruta y estado HTTP de las respuestas API, sin parámetros, cuerpos ni cabeceras. También recoge geometría, cookies, almacenamiento web e IndexedDB. Las capturas se generan al cambiar el ancho de la misma página, sin repetir escrituras.

La evidencia nueva es un registro de solo adición: cada línea contiene una instantánea JSON completa. Antes de permitir cada POST, añade y sincroniza la intención pendiente con su petición y clave originales. Cada reapertura comprueba archivo regular, propietario, permisos, número de enlaces e identidad del archivo. Una última línea interrumpida no borra las intenciones ni los recibos completos anteriores. `recuperar` admite tanto este formato como el JSON anterior, lee la última instantánea completa y no escribe evidencia ni crea capturas. Un fallo parcial conserva lo registrado y no acredita el resto del recorrido.

Ejemplo de configuración **sin valores operativos**:

```json
{
  "origen": "https://127.0.0.1:PUERTO",
  "chrome": "/RUTA/SISTEMA/chrome",
  "binario": "/RUTA/PRIVADA/vec-server",
  "binario_sha256": "SHA256_HEX_DEL_BINARIO",
  "hitos_clon": ["H3", "H4", "H5"],
  "uso_sintetico": true,
  "intervencion_cert": "/RUTA/PRIVADA/intervencion.crt",
  "intervencion_key": "/RUTA/PRIVADA/intervencion.key",
  "rrhh_cert": "/RUTA/PRIVADA/rrhh.crt",
  "rrhh_key": "/RUTA/PRIVADA/rrhh.key",
  "favorable": {"expediente_ref": "REFERENCIA_SINTETICA_A", "version_esperada": 5},
  "reparo": {"expediente_ref": "REFERENCIA_SINTETICA_B", "version_esperada": 5}
}
```

La marca H3–H5 es una declaración del operador sobre el clon preparado; el guion comprueba la huella del binario, pero no inspecciona migraciones. Antes de usar `registrar`, dirección debe acreditar la composición real, las concesiones vigentes y que ambos expedientes admiten fiscalización. Los certificados y claves nunca se pasan como texto al comando ni se copian a la evidencia.

```bash
python3 scripts/recorridos/intervencion/recorrer.py preparar --config /RUTA/PRIVADA/config.json --evidencia /RUTA/PRIVADA/intervencion-evidencia.json
python3 scripts/recorridos/intervencion/recorrer.py registrar --config /RUTA/PRIVADA/config.json --evidencia /RUTA/PRIVADA/intervencion-evidencia.json
# Dirección reinicia los mismos servicios y comprueba que el clon no cambió.
python3 scripts/recorridos/intervencion/recorrer.py recuperar --reinicio-acreditado --config /RUTA/PRIVADA/config.json --evidencia /RUTA/PRIVADA/intervencion-evidencia.json
```

`preparar` valida las entradas y prepara la carpeta privada de capturas; devuelve `PREPARADO` con `ejecutado: false`, sin abrir Chrome. El padre de la evidencia debe existir, pertenecer al usuario y tener permiso `0700`. Ante una precondición ausente antes de abrir el navegador, el guion devuelve `NO EJECUTADO` y código 2. Un fallo tras abrirlo devuelve `FALLO`; si ya se permitió un POST, devuelve `FALLO_CON_EFECTO_POSIBLE`, ambos con código 1. `--reinicio-acreditado` declara una comprobación externa de dirección; el guion no puede probar por sí solo que ambos servicios se reiniciaron. La evidencia puede contener referencias y observaciones sintéticas; se crea con permisos `0600` y no debe subirse a Git.

La recuperación exige que las tres peticiones iniciales hayan respondido `201`. Antes de cada POST se conserva la petición exacta en `intencion_pendiente`; si el navegador pierde la respuesta, la evidencia parcial queda cerrada para el guion. Dirección debe resolver esa intención con la misma clave antes de continuar. El guion nunca improvisa una nueva operación para superar un fallo.

El corte se basa en `origin/main` `3b910a170` y sus rutas visibles. La PR #166 seguía abierta al crear esta rama: ninguna capacidad de esa PR se atribuye aquí a `main`. La ejecución H3–H5 y la recuperación tras reinicio están pendientes; este README y el test local no las acreditan.

Para el ensayo del 30/09 se usará exclusivamente el clon de `main` identificado por dirección. Los manejadores de fiscalización y subsanación de `main@5a193a7a015c4077d3d7e206b99af4a8a43379eb` responden `201` también al reintento correcto; el modo `recuperar` exige ese estado en ambas rutas y conserva la petición y el recibo originales. Una guarda de conflicto que responda `409` impide acreditar la recuperación. La recuperación no debe ejecutarse sin coordinar el reinicio. La preparación del guion y sus pruebas locales no demuestran registro, firma ni recuperación en el clon.
