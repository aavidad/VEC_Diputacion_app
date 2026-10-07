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

Para un ensayo sintético aislado se puede añadir `-permitir-ejemplo`. El manifiesto saldrá con `ejemplo=true`; no sirve como catálogo real de la principal.

Este comando solo prepara y comprueba los artefactos. La publicación y activación requieren un canal autorizado que registre la aprobación y su auditoría. Conserve juntos los tres ficheros hasta cotejar ese registro. Si cambia la fuente, prepare de nuevo el catálogo y el manifiesto.
