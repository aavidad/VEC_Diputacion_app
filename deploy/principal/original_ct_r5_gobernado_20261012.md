# Preparación del original de resolución CT

Este montaje conecta la preparación del PDF al servicio de Documentos existente. El comprobante contiene su número VEC y la fecha de custodia confirmada. Preparar el documento no lo envía a Portafirmas ni registra una firma. La pantalla consumidora está en la PR #1033.

Reutiliza la fuente de identidad F1, el PDP y el emisor de CT, la consulta del expediente y la propuesta, el renderizador de formalización, el repositorio D13 y el almacén configurado en `identidad/documentos.json`. No altera los seis pools de `ct_v3.json`. La composición de registro de firmas sigue usando `NuevoRegistroConPlanV2`; este incremento no habilita sus dependencias ausentes.

## Instalación y configuración

La lista ordenada es `lista_sql_codexh_r5_original_20261012.txt`. AD69 se instala únicamente si sus dos funciones v2 no existen. AD244 se instala una vez sobre esa preimagen exacta. Si una guarda encuentra otro cuerpo, propietario o permisos, aborta con la clave y los valores: hay que cotejar la definición instalada antes de adaptar la migración. No ejecutar DOWN ni reaplicar migraciones instaladas.

CONFIG NUEVA: fichero privado `ct_firma_r5.json`, versión `1`, permisos `0600`, dentro del directorio ya configurado por `VEC_INTERNO_MATERIAL_DIR`. Su ausencia mantiene deshabilitada la preparación del original; su presencia exige material completo y verificable. Dirección debe aprovisionarlo antes de habilitar la pantalla en cidonia. No debe copiarse a Git.

| Campo | Valor requerido |
| --- | --- |
| `version` | `1` |
| `catalogo_motivos` | El mismo catálogo que `ct_v3.json`. |
| `motivo_original` | Entrada versionada publicada de ese catálogo, con su huella. |
| `capacidad_documentos` | Metadatos y archivo privado de la capacidad publicada vigente para `vec_documentos.operacion.v1`; formato existente `capacidadMaterial`. |
| `documentos_archivo` | `identidad/documentos.json`, relativo a `VEC_DEVELOPMENT_MATERIAL_DIR`. |
| `documentos_sha256` | SHA256 del fichero anterior completo, incluidos DSN y almacén. |
| `plantillas_archivo` | Exactamente `VEC_CT_PLANTILLAS_SOURCE_PATH`. |
| `plantillas_sha256` | SHA256 del fichero de plantillas. |
| `plantillas_id`, `plantillas_version`, `plantillas_huella_sha256` | Terna de la última versión publicada del catálogo de plantillas CT. |
| `organizacion_archivo`, `organizacion_sha256` | Exactamente `VEC_PERSONAL_ORGANIZACION_SOURCE_PATH` y SHA256 del fichero. |
| `rpt_archivo`, `rpt_sha256` | Exactamente `VEC_RPT_CATALOGO_PATH` y SHA256 del fichero. |
| `seudonimos_archivo`, `seudonimos_sha256` | Archivo relativo al material interno, con 32 bytes ya aprovisionados, y su SHA256. No se genera al arrancar. |

Se conservan las guardas de configuración y conservación del montaje existente. El catálogo provisional exige la doble habilitación de desarrollo y su almacén admitido; no sirve para declarar producción. El LOGIN documental debe pertenecer exclusivamente al rol ejecutor publicado y conservar sus nueve funciones exactas. El grupo de Documentos no acepta material de Personal ni un consumidor arbitrario.

El perfil fijo de RRHH debe tener las concesiones documentales publicadas para la organización correspondiente. El montaje conserva la garantía real de la autenticación F1 y no la eleva. Las guardas de garantía del original y del registro de firma se siguen aplicando. Si falta una concesión o la garantía exigida, la operación se deniega antes del efecto.

Centro, categoría y unidad se resuelven con los lectores de los mismos catálogos publicados que utiliza el alta. Una referencia sin nombre no produce un PDF con referencias opacas. En la copia ensayada, `unidad:desarrollo:rrhh` no tiene una etiqueta publicada; requiere resolver ese dato antes de preparar esos expedientes.

## Contrato de preparación

`POST /api/vec/contratacion-temporal/firmas-documento/original`, cuerpo JSON con `expediente_ref`, `documento: "resolucion"` y `version_observada`. Para obtener el comprobante, enviar una única cabecera `Accept: application/vnd.vec.ct.original-firmable.v2+json`. El contrato JSON v1 se conserva para sus consumidores anteriores.

La respuesta v2 añade `comprobante.numero_vec` y `comprobante.custodiado_en` a los datos del original y al PDF. Proceden del documento confirmado en Documentos. La respuesta solo sale tras confirmar la custodia; una respuesta de error no incluye PDF ni comprobante. Cada petición conserva su identidad nominal y la auditoría existente. Descargar de nuevo recupera el original confirmado, con la misma referencia, huella, número y fecha; no se presenta como envío ni firma.

## Evidencia y límites del ensayo

AD69 y AD244 se instalaron como migraciones nuevas en una copia propia de HZ21, PostgreSQL 18.6, contenedor sin red, con `--rm`, `--restart=no`, memoria de 2 GB y datos en disco. No se modificó cidonia ni el archivo de origen. Un primer intento de AD244 falló por un separador SQL y revirtió íntegro; la instalación corregida confirmó su transacción.

La prueba `pruebas_sql/ad244_original_firmable_ct.sql` usa los metadatos de la publicación documental existente. Comprueba la lectura y el contraste positivo, y rechaza consumidor desconocido, audiencia de Personal, huella alterada y material documental enviado como CT. Su LOGIN de contraste y sus permisos desaparecen con ROLLBACK. Propietarios y ACL de las funciones v2 permanecen exactos; los cuerpos y permisos v1 anteriores no cambian. La postimagen reforzada se comprobó aparte en ROLLBACK, sin reinstalar AD244.

Las pruebas focales Go y JavaScript acreditan composición, contratos, denegaciones y comprobante. Todavía faltan el recorrido con aplicación y Chrome, la medida HTTP con volumen realista y la recuperación del comprobante tras reiniciar aplicación y base. No atribuir a este ensayo SQL una firma, un envío, un tiempo de pantalla ni el cierre del recorrido completo de CT.
