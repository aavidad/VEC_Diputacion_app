# Revisión cruzada Orquesta del cliente web de borradores

Fecha: 2026-07-18
Estado: **NO-GO; corregir antes de integrar**
Modo: revisión estática y de solo lectura.

## Revisión fuente

| Fichero | SHA-256 revisado |
|---|---|
| `web/static/portal-empleado/portal-borradores-api.js` | `8925c9189a02fec277dfb45d0ff6bc9434890abd4bb6e5966b094e8a2959e12b` |
| `web/static/portal-empleado/portal-borradores-contrato.js` | `47759b037199481d34bdeb8f3eac6a99c2d8d8ec598a1757798e1c2d9c469f60` |
| `web/static/portal-empleado/portal-borradores.test.mjs` | `84dc2dd58010fef7e2d34e446397ae738aeeb2a4b46e71de2bbb89aa154ab35c` |

Los hashes coincidieron antes y después de ambas tandas.

## Revisión primaria

- Goal `goal:0e36c737c2740dcb2ca04c298b65d73d`, revisión 8,
  estado `succeeded`.
- AppSpec `app-spec:b6bbae092b4546d0697a7374e8c28d84`.
- Contrato:
  - ejecución `execution:dde8ac22219ba14ee9b6fd889ec0fa4c`;
  - artefacto
    `artifact:sha256:1c3c93b1e5420c22532bf84ad4d5fa70cf17921b21849e395406e113d5803e84`;
  - veredicto `NO-GO`.
- Seguridad de navegador:
  - ejecución `execution:422ee1a19e5ca9f4949e45aea45e8433`;
  - artefacto
    `artifact:sha256:12478c96c13afad824f2e7d17de94edbf790df0e160f323d4495915343b663d6`;
  - veredicto `NO-GO`.

## Metarrevisión independiente

- Goal `goal:ee4675e07098c5ccddfb7520d45579f5`, revisión 8,
  estado `succeeded`.
- AppSpec `app-spec:861553c289388209c0f8700199c73e0b`.
- Validez factual:
  - ejecución `execution:b198564ff1fa0c8a19cad2a8994b3c64`;
  - artefacto
    `artifact:sha256:76066e9dcd986db08e0ad0154c13ac29ecfa6c46fba50f07d9212b2f7448e09d`;
  - veredicto `NO-GO`.
- Riesgo de portal interno administrativo:
  - ejecución `execution:380f873f3ccd3bda50aa1fed3fbe3d15`;
  - artefacto
    `artifact:sha256:e882b5c2229dba3ca0128a8c95d89f76419f5c9bfae22eb905f8567be02d4511`;
  - veredicto `NO-GO`.

Los cuatro artefactos superaron las comprobaciones de Base64, UTF-8, tamaño,
tipo, Goal, ámbito y SHA-256 recalculado. Las attestations acreditan presencia
durable del análisis, no ejecución de tests ni integración.

## Hallazgos confirmados por Orquesta

1. **Alta:** una misma referencia y versión de catálogo puede aparecer con
   claves distintas y huellas contradictorias. Categorías y tipos deben exigir
   una única huella por identidad versionada del catálogo.
2. **Media:** no todas las salidas tempranas cancelan el cuerpo de respuesta:
   estado inesperado, cabeceras rechazadas, fragmento inválido o fallo de
   lectura deben cerrar el flujo conservando el error original.
3. **Media:** `AbortSignal` no gobierna la espera asíncrona del proveedor de
   credencial anterior a `fetch`.
4. **Baja y condicional:** el fallback sin `ReadableStream` materializa
   `response.text()` antes de medirlo. Si la matriz soportada admite ese camino,
   debe fallar antes de leer o disponer de lectura incremental realmente
   acotada.

## Alarmas retiradas o corregidas

- `Failed to create stream fd` era contaminación del canal de captura, no
  contenido de los ficheros.
- No se exige que `Content-Length` coincida con los bytes decodificados por
  Fetch cuando existe `Content-Encoding`; el límite debe aplicarse en la capa
  de transporte declarada.
- El fallback sin stream no se clasifica como vulnerabilidad alta de un
  navegador moderno sin antes acreditar que forma parte de la matriz soportada.

## Comprobaciones adicionales del director

La revisión independiente no sustituye la validación final del director. Se
confirmó además:

1. La expresión regular de `Idempotency-Key` acepta valores de 43 caracteres
   que decodifican a 32 bytes pero no son Base64URL canónico. Por ejemplo, una
   terminación con bits de relleno no nulos se normaliza a otro carácter. El
   cliente debe decodificar estrictamente, exigir 32 bytes y recodificar para
   comprobar igualdad, igual que el constructor Go.
2. El ETag fuerte debe quedar derivado y comprobado contra revisión y huella de
   `referencia_estado`, no ser únicamente una cadena opaca coincidente entre
   cuerpo y cabecera.
3. `409` y `412` son conflictos recuperables distintos; el error tipado y el
   futuro controlador deben conservar el código máquina y los cambios locales.
4. Los errores no exitosos necesitan un envelope cerrado y acotado con código
   máquina y correlación. No se mostrará texto arbitrario del servidor.

## Dependencias externas, no defectos de estos tres ficheros

- integración de la vista `elaboracion` en `portal.js`;
- handler Go y composición interna autenticada;
- Opciones V2 con paquete de catálogos `ID + versión + huella`, tipos de plazo,
  categorías de ayuda y zona horaria IANA;
- adaptación histórica por el paquete exacto fijado en cada borrador.

## Gate para la revisión sucesora

1. Huella única por catálogo y versión, con pruebas contradictorias y casos
   positivos de varias claves coherentes.
2. Cancelación de mejor esfuerzo uniforme, incluidas respuestas no exitosas,
   preservando siempre el error causal.
3. Aborto explícito durante credencial, fetch y streaming.
4. Matriz Fetch/navegadores declarada; sin fallback materializador no acotado.
5. Clave idempotente Base64URL estricta, canónica y de 32 bytes.
6. ETag ligado a revisión y huella, y envelope cerrado de errores.
7. Pruebas de 4 MiB, 65.536 fragmentos, UTF-8 inválido, cancelación, abortos,
   catálogo contradictorio, ETag y rutas.
8. Tests JavaScript verdes y nueva revisión estática Orquesta sin hallazgos
   altos abiertos.
