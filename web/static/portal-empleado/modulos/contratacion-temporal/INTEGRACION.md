# Integración productiva de contratación temporal

Estado: contrato visual y cliente HTTP productivo verificados, todavía no
conectados al coordinador ni integrados de extremo a extremo.

Este directorio contiene la interfaz, sus contratos neutrales y el cliente
HTTP cerrado para alta, propuesta, decisión y rectificación de cobertura. Los
adaptadores y datos sintéticos siguen separados y no entran en los manifiestos
productivos. La raíz corporativa debe inyectar autoridad y dependencias reales;
el E2E debe demostrar el recorrido completo.

## Puerto que debe inyectar O2-08

El presentador recibe:

```js
crearPresentadorAltaContratacionTemporal({
  catalogos,
  capacidad: "contratacion_temporal.solicitud.crear",
  ejecutor,
  generarClaveIdempotencia,
})
```

La capacidad es una indicación de presentación resuelta por composición. No
autoriza el efecto ni sustituye la decisión de servidor. Sin capacidad exacta,
con un catálogo válido pero no operable o sin una función `ejecutor`, el
formulario queda no disponible y no llama a ninguna frontera. Un catálogo
ausente, malformado o de esquema incompatible es un error de composición: el
constructor lo rechaza antes de crear una vista.

El ejecutor tiene esta forma neutral:

```js
async function ejecutor(comando, { signal }) {
  // La composición inyecta un puerto neutral; HTTP es un adaptador.
  return reciboPublico;
}
```

En éxito resuelve únicamente el recibo público cerrado. El cliente HTTP
descarta las causas privadas y publica códigos cerrados y localizables. No
existe un esquema remoto para devolver texto libre ni detalles por campo. Los
errores por campo actuales proceden exclusivamente de la validación local. El
servidor no debe confiar en que la vista interprete cuerpos, estados o mensajes
privados.

`signal` permite cancelar la espera. Una cancelación o pérdida de transporte
después de enviar no demuestra que el servidor haya descartado el efecto. El
cliente real marca el resultado como indeterminado; el presentador conserva la
intención en memoria, retira los controles de reenvío y espera una recuperación
protegida.

La frontera de O2-08 deberá usar una credencial breve ligada al cliente y
obtenida mediante composición confiable, fuera de este comando. No podrá usar
`Cookie`, `Set-Cookie`, almacenamiento web, credenciales en URL ni cabeceras
libres aportadas por el navegador para declarar identidad, perfil, organización
o autoridad. El transporte deberá fijar una lista positiva de entradas y
revalidar en servidor la credencial, el contexto y la capacidad.

## Comando cerrado

El ejecutor recibe exactamente:

```json
{
  "clave_idempotencia": "uuid-v4-canónico",
  "solicitud": {
    "centro_ref": "referencia-opaca",
    "contacto_ref": "referencia-opaca",
    "categoria_ref": "referencia-opaca",
    "grupo_subgrupo": "A2",
    "motivo_clave": "clave-gobernada",
    "detalle": "texto",
    "periodo": {
      "inicio": "2026-08-01T00:00:00Z",
      "fin": "2026-08-31T00:00:00Z"
    },
    "rc": {
      "existe": false
    },
    "documentos_adjuntos": [],
    "observaciones": ""
  }
}
```

Cuando existe RC, `rc` contiene además `numero`, `fecha`,
`importe: {centimos, moneda: "EUR"}` y `documento_ref`. No se aceptan campos
adicionales en ningún nivel.

La vista aplica los mismos límites públicos que la frontera O2-08B: el final
no puede superar la fecha civil de inicio desplazada cien años y el importe de
la RC no puede superar `922337203685477` céntimos. El importe se analiza como
dígitos decimales antes de convertirlo a un entero seguro de JavaScript; no se
multiplica una cantidad monetaria de coma flotante.

La clave se genera con CSPRNG al pasar a revisión, se conserva solo en memoria
durante el envío/reintento y nunca forma parte del estado renderizable, un
mensaje visible, un recibo, un log o una traza. Sí forma parte del comando
neutral que recibe el ejecutor. O2-08 debe conservar su semántica al traducir la
petición y rechazar su reutilización con contenido diferente.

O2-08 incorpora desde una frontera confiable, nunca desde este comando:

- referencia de autenticación y sesión;
- perfil activo;
- organización;
- actor, unidad y garantía;
- decisión, correlación, capacidad ejecutable y evidencias.

## Catálogos de composición

La vista no compila opciones funcionales. Espera el esquema cerrado
`vec.contratacion_temporal.catalogos_alta.v1`:

```json
{
  "esquema": "vec.contratacion_temporal.catalogos_alta.v1",
  "centros": [{
    "referencia": "referencia-opaca",
    "etiqueta": "texto",
    "contactos": [{
      "referencia": "referencia-opaca",
      "etiqueta": "texto"
    }]
  }],
  "categorias": [{
    "referencia": "referencia-opaca",
    "etiqueta": "texto",
    "grupos_subgrupos": [{
      "clave": "A2",
      "etiqueta": "texto"
    }]
  }],
  "motivos": [{
    "clave": "sustitucion",
    "etiqueta": "texto"
  }],
  "documentos": [{
    "referencia": "referencia-opaca",
    "etiqueta": "texto"
  }]
}
```

`documentos` solo lista referencias ya incorporadas al sistema documental. La
pantalla no sube bytes. La relación visible centro–contacto y
categoría–grupo reduce errores, pero no es autoridad.

Los máximos de 1.000 opciones por colección, 5.000 opciones totales y 200
caracteres por etiqueta son límites técnicos provisionales de protección del
navegador. O2-08 debe alinearlos, paginar antes de superarlos o publicar una
versión nueva del esquema; no son reglas funcionales.

Este esquema v1 es una proyección para selección y no transporta referencia,
versión, vigencia, huella ni procedencia de la publicación. O2-08 deberá ligar
la proyección a un catálogo gobernado en composición confiable, conservar allí
su identidad y revalidar publicación, vigencia y pertenencias al ejecutar. No
debe convertir etiquetas o metadatos del navegador en autoridad. La forma
canónica de esa ligadura sigue siendo una brecha de integración; no se inventa
como campo del comando de dominio.

## Recibo público minimizado

O2-08 devuelve a la vista exactamente:

```json
{
  "expediente_ref": "referencia-opaca",
  "numero_visible": "2026/CT-0001",
  "version": 1,
  "recibo_ref": "referencia-opaca",
  "confirmada_en": "2026-07-23T09:15:00Z"
}
```

Antes de proyectarlo debe verificar el `ReciboAlta` interno contra el
expediente confirmado. `auditoria_ref` y `evento_ref`, aunque son obligatorios
en el puerto Go, no se serializan hacia la vista. Tampoco se exponen identidad,
capacidad, clave de idempotencia, decisión, HMAC, atestación, correlación,
token, sello ni traza privada.

La UI solo valida estructura y límites de la proyección. No puede verificar
autenticidad criptográfica ni detectar la sustitución por otra referencia
opaca sintácticamente válida; esa garantía es bloqueante en O2-08 y en la
confirmación durable.

## Brechas bloqueantes

1. **Inyección productiva ausente.** El transporte real existe en
   `cliente-http.js`, pero el coordinador todavía no lo inyecta porque faltan
   las proyecciones y dependencias corporativas. No existe caída a DEMO.
2. **Contexto confiable ausente.** Autenticación, sesión, perfil,
   organización y actor deben proceder de composición/servidor. La vista no
   ofrece campos ni cabeceras para declararlos.
3. **Catálogo no autoritativo de contacto y grupo.** El dominio solo valida su
   sintaxis. `ResolverFlujoAlta` coteja centro, categoría y motivo, pero no
   recibe contacto ni grupo/subgrupo. El servidor debe demostrar existencia,
   vigencia, pertenencia centro–contacto y relación categoría–grupo antes del
   efecto.
4. **Proyección y autenticidad del recibo.** O2-08 debe validar el recibo
   interno completo y publicar solo los cinco campos minimizados.
5. **Enteros interoperables.** JavaScript exige céntimos y versión como enteros
   seguros. Go admite `int64` para el importe y `uint64` para la versión; O2-08
   debe fijar máximos interoperables o una codificación canónica antes de
   publicar valores superiores a `Number.MAX_SAFE_INTEGER`.
6. **Sistema documental.** Falta la fuente real de referencias incorporadas y
   la comprobación autoritativa de que cada referencia pertenece al ámbito
   permitido. Esta tarea no implementa carga documental.
7. **Composición del módulo.** Los activos del contrato y del cliente ya están
   en `web/interno.manifest` y `web/produccion.manifest`. El bootstrap
   productivo todavía no compone las dependencias reales ni entrega las
   capacidades y catálogos protegidos al coordinador.
8. **Router del shell.** `portal.js` mantiene títulos, hash y contador en una
   lista cerrada y queda fuera del write-set de O2-09B. Cambiar solo el
   coordinador dejaría una navegación falsa. También queda pendiente avanzar
   la cascada de cacheado cuando el integrador autorice ese archivo.
9. **Borrador administrativo.** El documento normalizado distingue guardar
   borrador de registrar, pero el caso de uso revisado solo ofrece alta. Esta
   pantalla no simula persistencia de borrador.
10. **E2E y aceptación.** Las 378 pruebas web y las pruebas Go acreditan el
    corte aislado. Falta probar navegador → API → aplicación → PostgreSQL →
    recibo, incluidos reinicio, concurrencia, recuperación del resultado
    indeterminado, accesibilidad asistida y acta de RRHH.

## Lectura del documento de RRHH

Los datos del alta mostrados en el Word están representados por el dominio:
centro, petición/detalle, categoría, periodo, RC y documentación
complementaria. Modalidad, validación de RC, unidad responsable,
fiscalización, llamamiento, formalización y GINPIX pertenecen a fases
posteriores y no se han añadido a esta pantalla.

La persona de contacto del centro no debe confundirse con el responsable de
una fase posterior. Número y referencia de expediente son resultados del
servidor, nunca entradas del formulario.

## Declaración de alcance

El recorrido productivo no contiene adaptador DEMO, datos sintéticos de
ejecución, subida documental, autoridad de navegador, cookies, almacenamiento
web, telemetría ni llamadas externas. Sí contiene un adaptador HTTP de mismo
origen, cerrado y sin credenciales ambientales.

**Cliente productivo verificado; composición y E2E pendientes.**

## Actualización de composición web de 23/07/2026

La declaración anterior conserva el corte histórico de O2-09B. Desde el
checkpoint web de 23/07/2026, el alta se compone con la superficie neutral de
expedientes, el shell registra el módulo, las traducciones y manifiestos están
actualizados y existe un adaptador volátil exclusivo de presentación.

La matriz de diecisiete tareas, las pruebas y el mapa de sustitución se recogen
en `docs/portal_vec/estado_web_contratacion_temporal_2026-07-23.md`.
`datos-presentacion.js`, `datos-presentacion-ampliacion.js` y
`adaptador-presentacion.js` siguen excluidos de los manifiestos interno y
productivo. El cliente productivo O4-05 se incorporó después en `023b890`, sin
sustituir esta separación. Continúan pendientes identidad corporativa real,
proyecciones protegidas, recuperación de recibo, composición y E2E durable con
PostgreSQL.
