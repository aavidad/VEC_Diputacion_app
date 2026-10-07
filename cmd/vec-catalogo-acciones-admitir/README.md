# Admisión del catálogo de acciones ADMIN

Esta herramienta consulta una instantánea publicada o presenta un plan ya
aprobado a la fachada `registrar_catalogo_acciones_admin_v1`. No crea el LOGIN,
la aprobación ni los descriptores de los módulos. La instalación de AD219 y
AUT58 deja el registro vacío.

La base necesita AD215 instalada antes de AD219; después se instala AUT58. La
lista del corte está en `deploy/principal/lista_sql_codexv_catalogo_acciones_admin_20261008.txt`.
El DBA configura fuera de Git un LOGIN exclusivo del grupo
`vec_admin_catalogo_acciones_ejecutor` y una fila vigente en
`config_catalogo_acciones_admin_v1`. Esa fila liga el LOGIN, la huella exacta
del plan, la referencia y huella de aprobación, la persona aprobadora, el
paquete, el destino, el entorno y la ventana. La fuente de cada descriptor
procede del módulo propietario y forma parte del paquete aprobado. Sin esa
configuración, la admisión se deniega y queda auditada.

El archivo privado del CLI tiene modo `0600`, pertenece al usuario que ejecuta
la herramienta y contiene `dsn` y `tiempo_segundos` (1–60). El plan también
tiene modo `0600`. Ambos se mantienen fuera de Git. Para aplicar el plan:

```sh
vec-catalogo-acciones-admitir --fase aplicar --config /ruta/privada/config.json \
  --plan /ruta/privada/plan.json --sha <sha256-del-plan> --idioma es
```

El plan es un objeto JSON de 17 cadenas: `esquema`, `operacion_ref`,
`aprobacion_ref`, `aprobacion_sha256`, `paquete_canon`, `paquete_ref`,
`paquete_version`, `paquete_sha256`, `catalogo_canon`, `catalogo_ref`,
`catalogo_version`, `catalogo_sha256`, `esperado_version`,
`esperado_sha256`, `preparado_en`, `caduca_en` y `entorno`. Los campos
`paquete_canon` y `catalogo_canon` contienen JSON canónico como cadena; sus
huellas son SHA-256 de los bytes UTF-8 exactos. El paquete declara fuentes
versionadas por módulo y el censo completo de perfiles; el catálogo conserva
las entradas y perfiles exactos del paquete. Un solicitante de perfil selecciona
después referencias publicadas: este CLI no acepta concesiones de una petición
como autoridad.

El JSON de `paquete_canon` tiene `esquema`, `referencia`, `version`, `fuentes`
y `perfiles`. Cada fuente tiene `modulo_id`, `referencia`, `version`,
`huella_sha256` y sus `entradas` completas. Cada perfil lleva el documento
vigente del rol, su control de vigencia y el tipo explícito `fijo_sistema` o
`administrable`. AUT58 coteja todos los RolID actuales y sus huellas con la
autoridad central, y rechaza la omisión o reclasificación de un fijo histórico.

Para recuperar la instantánea exacta por la función de lectura:

```sh
vec-catalogo-acciones-admitir --fase consultar --config /ruta/privada/config.json \
  --ref <referencia> --version <numero> --catalogo-sha <sha256> --idioma es
```

La salida JSON contiene un código y un mensaje del catálogo `es` o `en`. Una
admisión confirmada entrega el recibo; un error de confirmación obliga a
consultar el estado y a conservar la misma operación, sin generar otra. El
catálogo no asigna perfiles. Los descriptores nominales de B1 y FirmaDEV y su
aprobación externa siguen siendo dependencias: una prueba sintética del clon
no los publica ni reduce la garantía HIGH de producción.
