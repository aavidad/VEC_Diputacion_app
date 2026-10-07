# Recogida local de incidencias técnicas

`vec-registros-tecnicos` recibe por la entrada estándar las incidencias
`vec.incidencia_tecnica.v1` y los resultados `vec.resultado_tecnico.v1` que emite
VEC. Sistemas puede recoger en un directorio privado la salida técnica de varios
procesos, con límites de archivo,
rotación, retención y avisos por umbral. El catálogo común sigue siendo la
autoridad de códigos, componentes, etapas, severidades y mensajes.

La herramienta solo acepta esos dos esquemas. Rechaza campos extra, claves duplicadas,
nombres de campo con otra capitalización, valores fuera del catálogo y líneas
demasiado largas. Cuenta las entradas rechazadas y continúa con la siguiente,
sin guardar su contenido. Los archivos contienen una proyección reconstruida de
los campos admitidos, sin rutas, cabeceras, cuerpos, SQL ni errores libres.

## Resultados técnicos

El resultado técnico usa el mismo emisor, cola, trabajador y archivo que las
incidencias. Su catálogo admite `correcto`, `denegado`, `entrada_invalida`,
`cancelado` y `no_disponible`. El nivel se fija por resultado: `info` para
correcto o cancelado, `warn` para denegado o entrada inválida y `error` para
no disponible. Componente y etapa proceden del catálogo técnico existente;
el entorno y la versión proceden de la configuración del emisor. El registro
lleva la fecha UTC, la correlación técnica de 32 caracteres y su referencia
V3 `correlacion_` seguida de esos mismos caracteres. No incluye mensajes
libres, errores de bibliotecas, identidad ni recurso.

La frontera crea una correlación por petición. El puerto
`ReferenciaCorrelacionAutorizacionV2DePeticion` deriva la referencia V3 de esa
correlación privada, sin leer cabeceras ni generar otro identificador. Si falta
la correlación, el emisor descarta el resultado y aumenta el contador
`SinCorrelacion`. Un fallo del archivo aumenta `FallosEscritura`. Ambos
contadores son técnicos: no cambian un recibo SQL confirmado ni provocan otro
intento de negocio.

El recolector comprueba el esquema, el resultado y nivel catalogados, la pareja
correlación/referencia y la lista exacta de campos. Guarda la proyección
validada y suma `por_resultado` en las métricas. Las alertas por umbral se
configuran para códigos de incidencia y, opcionalmente, para denegaciones e
indisponibilidad. La auditoría funcional nominal conserva su propia cadena,
permisos y retención; no entra en estos archivos rotatorios.

## Avisos por resultados

El campo opcional `umbrales_resultado` admite únicamente `denegado` y
`no_disponible`, con umbrales enteros positivos. Si se omite o se deja vacío,
se conserva el comportamiento anterior. Sigue siendo obligatorio configurar
al menos un umbral de incidencia en `umbrales_alerta`.

Este fragmento es un ejemplo sintético para añadir a la configuración completa;
Sistemas debe fijar sus umbrales y ventana. No constituye una política aprobada:

```json
{
  "ventana_alertas_segundos": 60,
  "umbrales_resultado": {"denegado": 10, "no_disponible": 1}
}
```

Cada resultado validado y guardado suma una ocurrencia. Incidencias y resultados
comparten la ventana del recolector, medida desde su recepción, y cada resultado
produce como máximo un aviso por ventana. Un aviso de denegación indica que se
ha alcanzado el volumen configurado; no identifica a una persona ni declara un
ataque. Un aviso `no_disponible` informa de resultados técnicos observados sin
atribuirlos al validador, PostgreSQL u otro servicio.

Los avisos salen por stderr con el esquema `vec.alerta_resultado_tecnico.v1` y
solo seis campos: `esquema`, `instante`, `resultado`, `nivel`, `recuento` y
`ventana_segundos`. El nivel procede del catálogo del resultado: `warn` para
denegado y `error` para no disponible. No incluyen identidad, recurso,
componente, etapa ni correlación. Las alertas de incidencia mantienen su
esquema anterior. El contador `alertas` suma ambos tipos de aviso.

Si falla la escritura del aviso, incluida una escritura parcial sin error del
destino, la CLI termina con código 2 y no lo cuenta como entregado. El registro
técnico que originó el aviso ya está escrito. Los
contadores y ventanas se reinician con el proceso; la herramienta no deduplica
líneas repetidas ni conserva los umbrales alcanzados entre reinicios.

La prueba focal del emisor, la CLI, el archivo y los avisos se ejecuta con:

```sh
GOCACHE="$HOME/.cache/go-build" GOPROXY=off go test -p 8 \
  ./cmd/vec-registros-tecnicos -run '^TestCLIEmisorArchivoYAlertasResultados$' -count=1
```

Usa umbrales sintéticos de dos denegaciones y una indisponibilidad en 60
segundos. Guarda seis resultados, rechaza una entrada con campos ajenos y
emite dos avisos sin repetirlos. También comprueba que una entrada rechazada
no suma para alcanzar el umbral.

## Prueba con una incidencia sintética

Desde la raíz del repositorio:

```sh
registros_tmp=$(mktemp -d)
trap 'rm -rf "$registros_tmp"' EXIT
GOCACHE="$HOME/.cache/go-build" GOPROXY=off go build -p 8 -buildvcs=false \
  -o "$registros_tmp/vec-registros-tecnicos" ./cmd/vec-registros-tecnicos
cat > "$registros_tmp/config.json" <<EOF
{
  "directorio": "$registros_tmp/tecnicos",
  "max_linea_bytes": 1024,
  "max_archivo_bytes": 1048576,
  "max_archivos": 4,
  "retencion_segundos": 86400,
  "ventana_alertas_segundos": 60,
  "umbrales_alerta": {"ARRANQUE_FALLIDO": 1}
}
EOF
chmod 600 "$registros_tmp/config.json"
"$registros_tmp/vec-registros-tecnicos" \
  --config "$registros_tmp/config.json" \
  --textos web/static/textos/es/registros-tecnicos.json <<'EOF'
{"esquema":"vec.incidencia_tecnica.v1","instante":"2026-10-03T00:00:00.000Z","codigo":"ARRANQUE_FALLIDO","severidad":"critica","componente":"servidor","etapa":"escucha","entorno":"pruebas","version_binario":"936aac665","correlacion":"0123456789abcdef0123456789abcdef","recuento":1,"mensaje":"El servidor no ha podido arrancar."}
EOF
```

Devuelve un resumen con una entrada recibida y escrita, ninguna rechazada y un
aviso. El aviso sale por stderr, como `vec.alerta_tecnica.v1`, con fecha, código,
severidad, recuento y ventana. Para inglés, cambie el catálogo por
`web/static/textos/en/registros-tecnicos.json`. `--ayuda --textos …` muestra el
uso desde el catálogo elegido.

Estos límites ilustran el funcionamiento; Sistemas debe fijar los valores de
su instalación. Ningún plazo del ejemplo determina la conservación de expedientes
o de auditoría funcional.

## Configuración y archivos

La configuración exige todos los límites y al menos un umbral positivo para un
código del catálogo. `max_archivos` incluye el archivo activo; admite de 2 a 1024.
`max_linea_bytes` admite de 512 a 1048576 y `max_archivo_bytes` debe permitir al
menos esa cantidad más el salto de línea. La ventana de avisos debe ser positiva
y no superar la retención. Los topes de memoria son límites de la herramienta;
el plazo de conservación procede de la configuración.

El directorio debe ser absoluto, exclusivo para registros técnicos, propiedad
del usuario del proceso y con modo 0700. Si no existe, la herramienta lo crea.
Los archivos tienen modo 0600; rechaza enlaces simbólicos, enlaces duros en el
archivo activo y archivos administrados de otra identidad. Un bloqueo impide
que dos recolectores roten o retiren archivos en el mismo directorio a la vez.
La herramienta no cambia permisos de directorios existentes.

El activo se llama `incidencias-activo.jsonl`. Al superar el tamaño configurado,
o la antigüedad de su tramo al recibir una nueva incidencia, rota a
`incidencias-` seguido de una secuencia de veinte cifras y `.jsonl`. Solo retira
archivos con esos nombres exactos cuando exceden la cantidad o la antigüedad
configuradas. Aplica la retención al abrir y al rotar; un proceso sin nueva
entrada puede conservar un archivo vencido hasta el siguiente arranque o rotación.
Otros nombres quedan intactos. Si un nombre tiene el prefijo `incidencias-` y la
extensión `.jsonl`, su parte numérica debe poder interpretarse como un entero
sin signo de 64 bits; si no, se detiene la recogida para que Sistemas revise el
directorio. Las secuencias válidas pero sin el relleno canónico de veinte cifras,
o iguales a cero, se ignoran y quedan fuera de la rotación y la retención.

Al reiniciar conserva el activo y continúa la secuencia de los archivos rotados.
Un activo sin salto de línea final se rechaza para evitar añadir registros sobre
una escritura interrumpida. Se sincronizan los archivos al rotar y al cerrar;
una caída anterior puede perder las últimas escrituras. La herramienta no promete
recepción exactamente una vez ni sustituye la custodia judicial.

Los contadores son acumulados desde el arranque: recibidas, escritas, rechazadas,
avisos, archivos retirados y recuentos por código y resultado. El recuento de
incidencia suma las ocurrencias declaradas en cada incidencia. Un umbral
produce como máximo un aviso por código
y ventana; con umbrales de resultado activos, ambos tipos comparten el reinicio
de ventana al recibir un registro validado. La siguiente ventana se inicia tras
vencer la anterior. Un fallo de entrada, archivo o salida termina con código 2.
Los errores no imprimen la ruta ni el mensaje de la biblioteca.

## Alcance de este corte

La entrada debe proceder de los emisores técnicos de confianza, con transporte
y permisos locales a cargo de Sistemas. No es un receptor HTTP ni se instala o
conecta automáticamente a los servicios. La vista administrativa para Sistemas,
la recogida de todas las raíces y una señal técnica específica del validador
requieren cortes posteriores. Los avisos por resultados dependen de que los
consumidores emitan esos resultados mediante el contrato común.

Cuando el emisor recibe el contexto de una petición, escribe su correlación
técnica de 32 caracteres hexadecimales en la incidencia o el resultado. Este
recolector la conserva sin sustituirla. Una incidencia emitida sin contexto
recibe otra correlación aleatoria y no queda ligada a la petición; un resultado
sin contexto se descarta y se cuenta. La auditoría nominal, su cadena y su
exportación siguen en la auditoría común.
Nunca se debe apuntar esta herramienta a su almacenamiento.

El diseño sigue prácticas de [journald](https://www.freedesktop.org/software/systemd/man/252/journald.conf.html)
para límites y retención, de [Fluent Bit](https://docs.fluentbit.io/manual/4.0/data-pipeline/inputs/tail)
para entradas acotadas y rotación, de [Vector](https://vector.dev/docs/reference/configuration/sinks/file/)
para separar el destino de archivos, y de [OpenTelemetry](https://opentelemetry.io/docs/concepts/context-propagation/)
para conservar la correlación del contexto. Aquí se reutiliza el formato técnico
cerrado de VEC, sin añadir esas dependencias.

El registro de acceso por petición, las métricas y los perfiles se explican en
[Seguir una petición lenta](observabilidad_tecnica.md). No pasan por esta
herramienta.

## Consulta local para Sistemas

`vec-registro-tecnico-consultar` lee archivos JSONL locales sin modificar el
recolector ni sus registros. Admite incidencias y resultados con el mismo
contrato que valida el recolector, y los eventos `http.server.request` y
`vec.process.startup` de los procesos. La consulta usa el instante de cada
evento, nunca la fecha del archivo.

```sh
GOCACHE="$HOME/.cache/go-build" GOPROXY=off go run ./cmd/vec-registro-tecnico-consultar \
  --desde 2026-10-07T10:00:00Z --hasta 2026-10-07T11:00:00Z \
  --archivo /ruta/privada/tecnico.jsonl \
  --ruta '/api/vec/personal/rpt/positions/{valor}'
```

`--desde` se incluye y `--hasta` se excluye. Se pueden indicar hasta ocho
`--archivo` absolutos. `--codigo` acepta un código de incidencia del catálogo;
`--ruta` exige el camino depurado exacto que escribe el servidor. Los dos filtros
se usan por separado. Sin ellos, el resumen incluye las cuatro familias. Las
fechas RFC3339 se comparan en UTC. `--idioma en` cambia los mensajes de ayuda y
estado al inglés.

La salida suma incidencias por código, resultados por tipo y errores por clase
cerrada. Para peticiones muestra cantidad, lentas, errores 5xx, consultas a la
base de datos y duración total, media y máxima de petición, consulta y espera de
conexión. Para el arranque muestra preparaciones, fallos y tiempos. Las unidades
son segundos. No devuelve líneas originales, rutas consultadas, correlaciones,
texto SQL, direcciones ni mensajes de error libres.

La lectura admite como máximo 16 MiB por archivo y 16 KiB por línea, y el
intervalo no puede superar 31 días. Una línea ajena al contrato o incompleta
incrementa `rechazadas`: el resumen se entrega como `parcial` y el proceso
termina con código 3. Un archivo ilegible, un enlace simbólico, el mismo archivo
indicado dos veces, un objeto no regular o un archivo que supera el límite
termina con código 2 y no entrega un resumen incompleto. Una consulta completa
termina con código 0.

Esta CLI sirve a quien ya tiene acceso local a esos archivos. La acción
administrativa `administracion.registros_tecnicos.consultar` conserva su propio
circuito de sesión, autorización y auditoría. Esta herramienta no lo invoca ni
admite el formato de la auditoría funcional.
