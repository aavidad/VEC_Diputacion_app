# Preparar Alegaciones

Ejecutar desde el lado interno con el entorno privado vigente, `PGSERVICE`
apuntando a la misma base que el servidor y `PGPASSFILE` protegido. El destino
es el material externo separado de #1103. No usar el directorio interno.

```sh
deploy/principal/alegaciones_bolsa/preparar.sh \
  --servidor /ruta/al/vec-server \
  --material-externo "$VEC_MATERIAL_PORTAL_EXTERNO_DIR"
```

La primera ejecución comprueba el catálogo, el material y los archivos
existentes sin escribir. Con las comprobaciones correctas, repetir añadiendo
`--aplicar`. El guion reutiliza `vec-server preparar-portal-externo` para las
dos audiencias del consumidor `inscripcion_bolsa` y el publicador V2 de
Autorización para el catálogo de motivos. Mantiene la audiencia de inscripción
y los demás consumidores. Escribe `identidad/bolsa-alegaciones.json` al final,
con permisos `0600`, sin sobrescribir una referencia diferente.

CONFIG NUEVA: en el material externo, `identidad/bolsa-alegaciones.json`, el
catálogo `motivos_alegaciones_bolsa` v1 y la audiencia
`vec_bolsa_llamamientos.alegacion.presentar.v1` del consumidor
`inscripcion_bolsa`. Sus valores exactos salen de los dos JSON de esta carpeta;
las claves las produce el preparador existente, no este guion. `--servidor`
debe señalar al binario actualizado que incorpora esta corrección.

Una discrepancia de catálogo, archivo o separación impide comenzar los efectos.
Una interrupción durante la publicación o la escritura exige repetir el mismo
comando: los publicadores existentes son idempotentes y los archivos usan
renombrado. El guion no concede perfiles ni admite acciones. Dirección debe
publicar las tres acciones y sus concesiones antes de comprobar el recorrido.
Después de reiniciar, verificar Inscripción y Alegaciones con cada identidad.
Mientras falte material de Alegaciones, sus rutas quedan no disponibles y la
inscripción conserva sus propias dependencias obligatorias.
