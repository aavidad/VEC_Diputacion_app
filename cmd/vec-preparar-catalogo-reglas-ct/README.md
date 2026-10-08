# Preparar el catálogo base de reglas CT190

Este comando lee el paquete JSON indicado por el operador y genera dos ficheros: el catálogo canónico y un manifiesto que liga su huella con la del fichero fuente. No consulta ni modifica PostgreSQL. No se ejecuta al arrancar VEC.

```bash
go run ./cmd/vec-preparar-catalogo-reglas-ct \
  -fuente "$FUENTE_CATALOGO_CT" \
  -salida "$CATALOGO_CANONICO_CT" \
  -manifiesto "$MANIFIESTO_CATALOGO_CT" \
  -aprobacion-ref "$APROBACION_REF_CT"
```

Las tres rutas deben ser absolutas. Cree antes un directorio privado fuera de Git y asigne nombres nuevos a las salidas; el comando no sobrescribe ficheros ni escribe en `/tmp`. Las salidas tienen permisos `0600`. El catálogo canónico contiene exactamente los bytes de `json.Marshal(catalogo.ClonarCanonico())`, sin salto de línea final. La huella SHA256 del manifiesto corresponde a esos bytes; la huella de fuente corresponde a los bytes originales, incluidos espacios y saltos de línea. El manifiesto también conserva identificador, versión, referencia y revisión de la fuente.

El paquete debe tener el sobre `version_esquema`, `fuente` y `catalogo` del catálogo configurable vigente. La fuente de instalación debe ser un catálogo publicado, con `fuente.demostracion=false` y una referencia distinta de la marca de ejemplo. El comando valida cada entrada con el lector tipado de reglas, incluidas las entradas futuras. La referencia de aprobación suministrada debe coincidir exactamente con `catalogo.aprobacion_ref`. Los valores y plazos proceden del fichero proporcionado. El cotejo de la referencia no acredita por sí solo la aprobación: esta corresponde al circuito administrativo.

Para un ensayo sintético aislado se puede añadir `-permitir-ejemplo`. El manifiesto saldrá con `ejemplo=true`. El publicador CT190 solo acepta ese caso en desarrollo con autorización explícita; no sirve como catálogo real de la principal.

Tras instalar CT190 y CT191, detenga las altas de fases y compruebe la última secuencia de activación. El publicador recibe los tres ficheros y esa secuencia. La conexión se configura con las variables habituales de libpq y una cuenta técnica que pueda asumir `vec_contratacion_temporal_propietario`.

```bash
VEC_CT190_ENTORNO=produccion \
VEC_CT190_APROBACION_REF="$APROBACION_REF_CT" \
deploy/principal/publicar_catalogo_ct190.sh \
  "$FUENTE_CATALOGO_CT" "$CATALOGO_CANONICO_CT" \
  "$MANIFIESTO_CATALOGO_CT" "$SECUENCIA_ESPERADA"
```

Para ensayar el paquete de ejemplo en una base sintética, use `VEC_CT190_ENTORNO=desarrollo VEC_CT190_PERMITIR_EJEMPLO=1` y prepare antes los artefactos con `-permitir-ejemplo`. Esa vía no representa una aprobación de RRHH. La referencia de aprobación de una publicación real debe coincidir con la del catálogo y proceder del circuito administrativo.

El script comprueba las huellas y publica la versión y su activación en una transacción serializable. Si la secuencia cambió, falla sin conservar una publicación parcial. Conserve los tres ficheros hasta cotejar la fila publicada, la activación y sus huellas. Si cambia la fuente, prepare nuevos artefactos; una versión publicada es inmutable. La cuenta técnica que publica queda en `publicada_por` y `activada_por`; esos campos no atribuyen una decisión a RRHH.
