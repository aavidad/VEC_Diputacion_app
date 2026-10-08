# Emisor V3 de las consultas de usuarios

`NuevaConfianzaUsuariosV3` compone la cadena real de autorización: fuente y
registro PostgreSQL, catálogo de motivos, PDP durable, firmante COSE,
verificador de confianza y emisor HMAC. Recibe material y gobierno privados,
ya aprovisionados; no genera claves, abre conexiones ni emite permisos al
construir. Exige pools distintos para fuente, registro y motivos.

La factoría exige exactamente `vec.admin.usuarios.listar.v1` y
`vec.admin.usuarios.consultar.v1`. La factoría legada conserva sus once
audiencias exactas; ninguno de los dos conjuntos acepta el otro ni una unión.
Ambas reutilizan las comprobaciones de raíz, gobierno, fechas, claves,
referencias y materiales distintos.

`NuevoEmisorUsuarios` recibe esos dos emisores reales, dos motivos del catálogo
privado y reloj. Implementa `ports.EmisorLecturaUsuariosAdministrables`.
Valida el formato canónico mediante el helper puro del lector, sin copiar
reglas SQL ni convertir un DTO en permiso. La solicitud usa el vínculo V2
original, una copia del resultado registrado, la correlación interna, el
recurso ligado al material y el motivo privado de su audiencia.

La instantánea recibida debe describir Aplicación v5, asignación y control
vigentes, ámbito cubierto y concesión exacta para `gestion_usuarios`, garantía
alta y obligación `auditar`. Listar exige cinco campos; consultar exige los
mismos salvo `siguiente_cursor`. Esa instantánea sólo es una precondición: el
PDP común consulta otra vez su fuente durable antes de conceder o denegar.
El efecto se liga a la huella del contexto del recurso, no al SHA aislado del
material.

La emisión delega todos los pasos criptográficos al proveedor común. Sólo su
marcador de denegación explícita registrada se traduce a denegación; errores de
configuración, validación, proveedor o cancelación devuelven indisponibilidad
sin mensaje bruto ni exportación. La respuesta coteja audiencia, acción,
recurso, contexto, registro y versiones CA con la evidencia original.

La vigencia de la concesión se mide con la confirmación durable que devuelve
el PDP, ligada a la misma decisión, motivo y contexto mediante la orden de
registro. `DecisionAutorizacionLigadaV3.VigenteEn` siempre responde que no,
porque la decisión en memoria no prueba el registro; usarla dejaba sin lectura
toda concesión. El consumo SQL vuelve a exigir la decisión registrada.

Las pruebas construyen ambas factorías con la cadena común y pools sin abrir,
y comprueban solicitudes sintéticas y rechazos anteriores al PDP. Una prueba
fabrica la confirmación con un registro que no persiste nada, sólo para medir
la ventana y la ligadura; no acredita una emisión con PostgreSQL.
El montaje operativo requiere los pools y el firmante reales, el gobierno y
las dos claves privadas aprobadas, el catálogo de motivos y el árbol SQL
AUT42/AD184/CA32/CA34/AUT43/AD185. Esta pieza no modifica SQL ni monta rutas.

## Comprobación focal de seguridad

Gosec se ejecutó sólo sobre `internal/app/administracion` y `internal/vec/ports`,
el paquete del puerto importado. No señaló los archivos del emisor. Emitió
12 avisos en archivos heredados de `ports`, sin cambios en esta rama:

- G101, cinco: `pagos_capacidades.go` (líneas 18 y 19), `documentos.go` (35), `cotejo.go` (42), `cargas_documentales.go` (39). Son etiquetas públicas de separación de dominios de huella; no contienen credenciales ni material secreto.
- G115, dos en `atestacion_autorizacion.go` (96 y 99): se comprueba el espacio restante antes de convertir la longitud; la suma queda dentro del slice, cuyo tamaño ya es representable como `int`.
- G115, cuatro en `preimagen_recurso_autorizacion_ejecucion_v4.go` (241, 245, 294 y 316): el lector comprueba espacio y límites antes de convertir; el formato limita la preimagen a 2 MiB y los mapas a 512 entradas. Los escritores reciben el recurso validado y codifican esos tamaños acotados.
- G115, uno en `reconciliacion_documental_v4.go` (839): el encuadre recibe referencias, hashes, estados y un reto validado de hasta 64 bytes. Son campos acotados del contrato, no un blob libre de la petición.

Los avisos se justifican en este corte sin modificar código ajeno ni añadir
supresiones. Las dos revisiones independientes y el ensayo nominal real siguen
pendientes; las pruebas locales no sustituyen ese recorrido.
