# Base privada del portal externo

`scripts/operacion/inscripcion_bolsa/preparar_material_externo.py` prepara una base externa nueva a partir del material interno autorizado. Use dos rutas absolutas, privadas y sin relación padre–hijo. Prepare la base **en el host**, dentro de un directorio padre privado, antes de montarla en `/vec-material-externo`; la raíz `/` no cumple esa condición. El destino debe estar ausente: un directorio vacío ya existente se rechaza para evitar sobrescrituras. El material interno conserva el certificado y la identidad aprobados del candidato y no se modifica.

Dirección debe fijar `PERFIL_EXTERNO_REF` a la referencia `prf_…` que figura en el plan aprobado de ContextoActor externo. Debe ser distinta del perfil interno. El script no calcula un perfil por nombre, certificado o cargo. En `identidad/bolsa-candidato.json` del destino conserva exactamente cuenta, persona, candidatura, sujeto y certificado; sustituye solo `perfil_ref` por la referencia externa indicada. `--comprobar` coteja la misma referencia y el resto de datos con la fuente interna.

Primero compruebe si la base externa existe:

```sh
python3 scripts/operacion/inscripcion_bolsa/preparar_material_externo.py \
  --material-interno "$MATERIAL_INTERNO" --material-externo "$MATERIAL_EXTERNO" \
  --perfil-externo-ref "$PERFIL_EXTERNO_REF" --comprobar
```

Si aún no existe, use los mismos argumentos con `--aplicar`. El script no reemplaza un árbol existente: lo comprueba y termina, o rechaza una diferencia. Ante una escritura interrumpida deja de aceptar ese destino; Dirección debe revisar la causa antes de recuperar la preparación. No se ejecutó aquí contra material privado.

La base contiene la CA **pública**, el certificado público y los ficheros de identidad existentes del candidato, un servidor TLS nuevo firmado por la CA interna, una marca de portal externo y claves de idempotencia propias. El formato de `idempotencia/configuracion.json` procede del material interno y se comprueba según el lector Go; ninguna clave HMAC se copia. La clave privada del candidato permanece fuera del servicio. La clave privada de la CA, KMS y TSA tampoco se copian. El manifiesto conserva las huellas de la CA, del servidor nuevo y del certificado candidato.

Después, el proceso interno puede usar el [preparador existente](../../../cmd/vec-server/separacion_portales.go) `vec-server preparar-portal-externo --material-externo "$MATERIAL_EXTERNO" --consumidores inscripcion_bolsa` para publicar el gobierno de claves V3 y escribir `externo/v3/` y, cuando proceda, los alias autorizados. Ese preparador exige una base externa ya separada y no admite como destino `MATERIAL_INTERNO`. Nuestro script no conecta con PostgreSQL; el preparador sí publica material de confianza, pero no concede perfiles ni publica una convocatoria. El servidor externo o combinado debe cargar su configuración nominal y verificar el material con el cargador Go antes de atender peticiones.

Falta fuente aprobada si el árbol interno no contiene la identidad candidata exacta o si no se dispone del perfil externo del plan ContextoActor. En tal caso deténgase: no derive otra persona, candidatura ni perfil desde el nombre o el certificado. Para la provisión y los permisos consulte [el guion de autoridad externa](autoridad_externa_combinado.md).
