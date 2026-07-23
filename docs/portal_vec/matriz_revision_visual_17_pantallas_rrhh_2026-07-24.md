# Matriz de revisión visual de las 17 pantallas de RRHH

Fecha: 24 de julio de 2026.

Estado: **NO-GO visual**. Contrato de captura implementado; ejecución Docker
válida pendiente sobre la imagen reconstruida que integre la ampliación visual.

Fuente funcional: documento «Pantalla de procedimiento de gestión de
contratación y gestión de bolsas». El documento original no se incorpora ni se
modifica. Las imágenes se consultaron temporalmente fuera de Git.

## Decisión

Las diecisiete pantallas se describen en
`scripts/revision_web/pantallas_rrhh.py`. La matriz no contiene HTML, CSS,
reglas ni datos de negocio. Solo declara:

- ruta canónica y perfil de presentación;
- expediente, tarea y pestaña que forman el contexto;
- secuencia estable de controles públicos para alcanzar el estado;
- selector cuyo asentamiento acredita que la navegación terminó;
- nombre determinista de la captura;
- criterios que debe revisar la puerta automática y una persona;
- estado de revisión y brecha explícita cuando la superficie aún no reproduce por
  completo la referencia.

El capturador abre cada escenario en un contexto limpio, sin cookies ni
almacenamiento del navegador. Las pantallas se capturan en el **viewport**, no
como página completa, a 1536×1024 —dimensión de la referencia—, 1440×1000 y
1280×900. Tras la navegación se restaura el origen superior sin enfocar el
enlace de salto. No se manipula el DOM para saltar fases. La única
preparación de un estado posterior es la operación sintética y autorizada de la
pantalla 16; se ejecuta una vez, mediante el adaptador de presentación, y exige
su recibo. Esto no acredita idempotencia, transmisión ni conciliación
productivas.

Ruta común:

```text
/portal-empleado/?presentacion=rrhh&perfil=administrador#contratacion-temporal
```

Expediente sintético común desde la pantalla 3:

```text
exp-demo-contratacion-005487
```

## Matriz 1..17

| N.º | Pantalla | Pestaña / tarea | Asentamiento | Captura | Revisión visual |
|---:|---|---|---|---|---|
| 1 | Inicio y cuadro de mando | `cuadro` | `.ct-exp-listado` | `01-inicio-cuadro-mando.png` | pendiente · NO-GO |
| 2 | Nueva petición de personal | `alta` | `[data-ct-form]` | `02-nueva-peticion-personal.png` | pendiente · NO-GO |
| 3 | Análisis de RRHH | `expediente` / `tarea-analisis` | tarea actual 3 | `03-analisis-rrhh.png` | pendiente · NO-GO |
| 4 | Gestión de bolsa y vía de cobertura | `expediente` / `tarea-cobertura` | tarea actual 4 | `04-gestion-bolsa-cobertura.png` | pendiente · NO-GO |
| 5 | Bandeja y asignación de la unidad | `expediente` / `tarea-asignacion` | tarea actual 5 | `05-bandeja-asignacion-unidad.png` | pendiente · NO-GO + brecha funcional |
| 6 | Informe jurídico de la unidad | `expediente` / `tarea-informe-juridico` | tarea actual 6 | `06-informe-juridico-unidad.png` | pendiente · NO-GO |
| 7 | Firma de jefatura y envío a Intervención | `expediente` / `tarea-envio-intervencion` | tarea actual 7 | `07-firma-jefatura-envio-intervencion.png` | pendiente · NO-GO |
| 8 | Fiscalización por Intervención | `expediente` / `tarea-fiscalizacion` | tarea actual 8 | `08-fiscalizacion-intervencion.png` | pendiente · NO-GO |
| 9 | Subsanación de reparos | `expediente` / `tarea-subsanacion` | tarea actual 9 | `09-subsanacion-reparos.png` | pendiente · NO-GO |
| 10 | Inicio del llamamiento | `expediente` / `tarea-iniciar-llamamiento` | tarea actual 10 | `10-inicio-llamamiento.png` | pendiente · NO-GO |
| 11 | Selección de candidatura | `expediente` / `tarea-seleccion-candidato` | tarea actual 11 | `11-seleccion-candidatura.png` | pendiente · NO-GO |
| 12 | Resultado del llamamiento | `expediente` / `tarea-resultado-llamamiento` | tarea actual 12 | `12-resultado-llamamiento.png` | pendiente · NO-GO |
| 13 | Preparación y traslado a Intervención | `expediente` / `tarea-traslado-intervencion` | tarea actual 13 | `13-documentacion-traslado-intervencion.png` | pendiente · NO-GO |
| 14 | Informe definitivo para formalización | `expediente` / `tarea-informe-definitivo` | tarea actual 14 | `14-informe-definitivo-formalizacion.png` | pendiente · NO-GO |
| 15 | Preparación de datos para GINPIX | `expediente` / `tarea-ginpix` | tarea GINPIX | `15-preparacion-datos-ginpix.png` | pendiente · NO-GO |
| 16 | Resumen y recibo de envío GINPIX | `expediente` / `tarea-ginpix` + operación DEMO | `[data-ct-exp-recibo]` | `16-resumen-recibo-ginpix.png` | pendiente · NO-GO + brecha funcional |
| 17 | Generación documental para formalización | `expediente` / `tarea-formalizacion` | tarea actual 17 | `17-documentos-formalizacion.png` | pendiente · NO-GO |

La expresión «tarea actual N» de la tabla se materializa en un selector exacto
`[data-ct-exp-tarea="<referencia>"][aria-current="step"]`; no depende del texto
visible ni de la posición física de un botón.

## Brechas no ocultas

### Pantalla 5

La asignación a unidad es navegable y trazable. Todavía no existe una bandeja
independiente de la unidad con su propio listado de expedientes, como muestra
la referencia. La navegación no autoriza a declarar paridad visual.

Condición de cierre: la ampliación debe exponer una vista de bandeja gobernada
por unidad y ámbito, con totales, estados, selección y apertura del expediente,
sin ampliar permisos desde el cliente.

### Pantalla 16

La pantalla previa GINPIX existe. El estado posterior se prepara mediante
`Enviar a GINPIX` en el adaptador DEMO y se asienta únicamente cuando aparece
un recibo correlacionado. No existe todavía una pantalla final independiente
equivalente a la referencia.

La operación:

- exige cabecera y banner de presentación;
- exige el perfil y la capacidad sintética concedidos;
- usa un contexto de navegador nuevo;
- no contacta GINPIX ni otro servicio exterior;
- no salta fases mediante escritura del DOM;
- no se presenta como prueba de idempotencia, entrega o conciliación real.

Condición de cierre: la ampliación debe ofrecer resumen final, índice
documental, estado del envío, acuse y conciliación dentro del contrato visual.
Producción añadirá autorización transaccional, idempotencia durable y adaptador
GINPIX real.

No hay actualmente una pantalla marcada como bloqueada por navegación. Si una
evolución elimina un selector o impide alcanzar el estado, la matriz admite el
estado `bloqueada` con causa obligatoria; la ejecución no debe sustituirlo por
una captura falsa.

## Evidencia rechazada y condición NO-GO

Las 18 capturas obtenidas antes de este contrato no son evidencia aceptable:
usaron página completa y produjeron imágenes de 1440×1270 a 1440×2317, por lo
que no son comparables con la referencia 1536×1024. Además mostraban, según la
pantalla, un gran hueco superior, cabecera o lateral fijo duplicado/desplazado
y el enlace «Saltar al contenido» visible.

No se reutilizan ni se contabilizan. El NO-GO solo podrá levantarse después de:

1. reconstruir la imagen con la superficie final;
2. ejecutar los 51 escenarios de viewport;
3. comprobar automáticamente ausencia de errores y desbordamientos;
4. inspeccionar manualmente 1536, 1440 y 1280, comparando 1536 con la
   referencia;
5. confirmar que no aparecen huecos, duplicados sticky ni enlace de salto;
6. reevaluar las brechas funcionales de las pantallas 5 y 16.

## Criterios visuales

Todas las pantallas deben:

1. conservar el shell, navegación lateral y tokens del tema común;
2. comunicar fase, estado y acción con texto e icono/forma además del color;
3. evitar recortes, solapes y desbordamiento de títulos, formularios, tablas,
   estados y acciones;
4. mantener visible la marca de datos sintéticos sin efectos administrativos;
5. mostrar en la primera región útil el trabajo, estado, evidencia y siguiente
   acción, no una portada decorativa;
6. diferenciar guardar, validar, enviar, firmar, registrar y confirmar;
7. conservar referencias, actor/unidad, instante, versión y recibo cuando
   corresponda;
8. mantener foco visible, nombres accesibles, orden de teclado y contraste
   WCAG 2.1 AA.

Cada entrada añade criterios propios. Estos se serializan en
`resultados.json`, junto con ruta, perfil, expediente, tarea, pestaña,
asentamiento, estado de revisión y brecha, para que una revisión posterior no dependa de
memoria oral.

## Ejecución reproducible

Con la composición de presentación reconstruida y saludable:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion \
  python3 scripts/capturar_presentacion_web.py \
  --url-base http://127.0.0.1:8080 \
  --salida var/revision-web-rrhh-17 \
  --solo-pantallas-rrhh
```

La salida esperada, si las 17 son navegables, es:

- 51 escenarios: 17 pantallas × 3 viewports;
- capturas bajo
  `var/revision-web-rrhh-17/capturas/rrhh-1536/pantalla-rrhh/`,
  `var/revision-web-rrhh-17/capturas/rrhh-1440/pantalla-rrhh/` y
  `var/revision-web-rrhh-17/capturas/rrhh-1280/pantalla-rrhh/`;
- `resultados.json` con la matriz y los hallazgos;
- `informe.md` con enlaces a cada captura.

La puerta estricta falla ante navegación incompleta, selector no asentado,
recurso roto, error JavaScript, control sin nombre, almacenamiento del
navegador, desbordamiento o cualquier hallazgo ya soportado por el auditor.

## Pruebas sin navegador

```bash
python3 -m unittest scripts.tests.test_capturar_presentacion_web
python3 -m compileall -q \
  scripts/capturar_presentacion_web.py \
  scripts/revision_web \
  scripts/tests/test_capturar_presentacion_web.py
```

Las pruebas comprueban numeración 1..17, nombres únicos, ruta/perfil, contexto,
tarea, asentamiento superior, criterios, tamaños, estado de revisión, brechas y la lista positiva
exacta de la operación GINPIX. No sustituyen la ejecución Docker ni la
inspección humana de las 51 capturas.
