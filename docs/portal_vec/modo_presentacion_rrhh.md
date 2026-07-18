# Modo de presentación RRHH

## Decisión

La presentación del martes se entrega como un artefacto separado y desechable,
no como una rama provisional del servidor productivo. Las pantallas, sus
componentes accesibles y los contratos que consumen son parte del producto. Lo
intercambiable son los adaptadores que suministran datos o ejecutan órdenes.

Esto permite enseñar los recorridos completos sin fingir que existe una firma,
un registro o una notificación reales y sin obligar a reescribir la web si el
proyecto continúa.

| Elemento | Presentación | Producción |
| --- | --- | --- |
| Binario | `vec-presentacion` | Proceso interno por definir; `vec-server` sigue siendo composición heredada de desarrollo |
| Perfil | `presentacion_rrhh` | `produccion` |
| Datos privados | Prohibidos | Solo mediante conectores autorizados |
| Datos de la muestra | Metadatos BOP públicos reales; personas, expedientes, actos y resultados privados sintéticos y marcados | El adaptador DEMO se excluye físicamente de la imagen |
| Escrituras durables o de servidor | Ninguna; solo cambia memoria volátil de la pestaña | Casos de uso, autorización y recibos reales |
| API | Consulta pública local de Bolsa | API pública/interna según frontera |
| Firma, registro, pagos y mensajes | Simulación visual | Adaptadores productivos aún por integrar |
| Red saliente de la aplicación | Local: no compone clientes externos; Docker: red `internal` sin salida | Según allowlist y decisión de Sistemas |

## Activación cerrada

La presentación parte deshabilitada. El servidor exige simultáneamente:

1. binario específico `cmd/vec-presentacion`;
2. perfil `VEC_EXECUTION_PROFILE=presentacion_rrhh`;
3. selector `VEC_RRHH_PRESENTATION_ENABLED=true`;
4. primera guarda literal
   `ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO`;
5. segunda guarda literal
   `CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA`;
6. autenticación deshabilitada, almacenamiento en memoria y catálogo personal
   ausente;
7. fuentes cuyos nombres terminen en `.demo.json`;
8. listener con IP literal loopback, privada o link-local y allowlist compuesta
   únicamente por redes locales enumeradas.

La comprobación se hace en configuración, bootstrap y servidor. No depende de
un `build tag`. Los binarios normales rechazan incluso un selector parcial de
presentación para evitar que una variable copiada por error exponga datos
sintéticos.

## Superficie servida

El handler usa lista positiva y métodos `GET` y `HEAD`:

- `/presentacion/`: selector de recorridos;
- `/bolsa/`: consulta pública;
- `/area-personal/`: punto de vista de la persona candidata;
- `/portal-empleado/`: punto de vista técnico de RRHH;
- `/api/publico/`: consulta pública local de convocatorias y categorías;
- `/healthz`, `/styles.css` y `/favicon.svg`.

No se sirven `/api/vec/`, `/api/demo`, `/candidates`, el árbol de datos, la
documentación del repositorio ni rutas estáticas generales. Las rutas no
canónicas y los escapes de directorio se rechazan. Todas las superficies
prohíben cookies, `Authorization`, credenciales de proxy y cabeceras de
identidad ambiental. Conservan CSP, `nosniff`, `DENY`, política de referente y
el resto de cabeceras del servidor común.

## Arranque para la revisión

La forma local más simple es:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

Después se abre `http://127.0.0.1:8081/presentacion/`. El script fija las dos
guardas, loopback, memoria y las fuentes `.demo.json`; estas incorporan
únicamente metadatos públicos reales del BOP y estado privado sintético. No
carga credenciales.

El arranque del binario y las fronteras HTTP se comprueban de extremo a extremo
con un entorno limpio mediante:

```bash
scripts/smoke_presentacion_rrhh.sh
```

La revisión funcional y visual automatizada se ejecuta con:

```bash
python3 scripts/capturar_presentacion_web.py \
  --url-base http://127.0.0.1:8081 \
  --ejecutable-navegador /snap/bin/chromium
```

Recorre 32 vistas y 21 estados de interacción en escritorio, tableta y móvil:
159 escenarios en total. Exige la cabecera técnica exclusiva de presentación,
comprueba menús y recibos DEMO, y falla ante almacenamiento del navegador,
errores, recursos fallidos, controles sin nombre o desbordamientos. El detalle
y los informes generados se describen en
[captura y revisión de la presentación](revision_web_presentacion.md).

El contenedor aislado se arranca con:

```bash
docker compose --profile presentacion up --build -d
```

El proxy publica únicamente `127.0.0.1:8081`. La aplicación queda en la red
Docker `internal`, sin ruta de salida, no publica puertos directamente, usa
usuario sin privilegios, sistema de ficheros de solo lectura, límites de
procesos/memoria/CPU y todas las capacidades Linux retiradas.

Parada y retirada de contenedores:

```bash
docker compose --profile presentacion down
```

## Separación física de artefactos

El `Dockerfile` contiene dos destinos:

- `runtime-presentacion`, que incluye el launcher, los adaptadores de
  presentación, los ficheros `.demo.json` y solo `vec-presentacion`;
- `runtime`, que elimina del árbol web cualquier ruta cuyo nombre contenga
  `presentacion` o `demo`, no copia `data/demo`, no incorpora
  `vec-presentacion` y solo contiene `vec-server`.

La inspección física reproducible es:

```bash
scripts/verificar_contenido_artefactos_presentacion.sh
```

La prueba construye ambos destinos, exporta sus sistemas de ficheros y falla si
producción contiene un adaptador, launcher, dato o binario de presentación, o
si el artefacto de muestra carece de sus piezas declaradas.

## Modelo de amenaza acotado

Este modo reduce, pero no convierte la muestra en producción:

- evita introducir datos personales: solo reutiliza metadatos públicos del BOP
  y fixtures sintéticos para identidad, expedientes y actuaciones;
- impide que un botón visual produzca un acto administrativo;
- evita que cookies o cabeceras del navegador se interpreten como identidad;
- impide publicar por accidente otros directorios del repositorio;
- impide seleccionar la muestra desde el binario normal;
- el arranque local no compone clientes para PostgreSQL, S3, OSRM, Autofirma,
  registro, pasarela de pago o comunicaciones; el perfil Docker añade una red
  `internal` sin salida que impide alcanzarlos por red;
- muestra siempre avisos de “demostración” y “sin validez administrativa”.

No acredita autenticación, autorización nominal, persistencia, firma, registro,
notificación fehaciente ni cumplimiento productivo. Tampoco debe recibir una
copia de datos personales o expedientes reales “para que la demo parezca más
completa”; los datos públicos verificables se incorporan con su procedencia.

## Sustitución incremental de adaptadores

Cuando se autorice continuar, no se cambia la web completa. Para cada
capacidad se sigue este orden:

1. cerrar el contrato de entrada, salida, errores y recibos de la capacidad;
2. implementar el adaptador real detrás del puerto de aplicación ya definido;
3. probar dominio, adaptador, autorización y vertical E2E con datos sintéticos;
4. habilitar la composición real solo en el perfil correspondiente;
5. hacer que la pantalla prefiera el contrato real y falle cerrada si no está
   disponible;
6. retirar de la presentación únicamente el adaptador sintético de esa
   capacidad cuando ya no sea necesario.

Así pueden cambiarse, de uno en uno, consulta pública, perfil, expedientes,
documentos, baremación, revisión, llamamientos, firma, registro, pagos y
comunicaciones. Nunca se conectará un adaptador de presentación a un puerto de
producción ni se migrarán sus estados: los actos simulados son descartables y
no autoritativos.

## Retirada completa

Si no se aprueba el proyecto, basta con retirar el perfil Compose
`presentacion`, el destino `runtime-presentacion`, `cmd/vec-presentacion`,
`web/static/presentacion`, los `datos-presentacion.js`, los ficheros
`.demo.json` exclusivos y los dos scripts de presentación. El runtime
productivo no depende de ellos.

Si se aprueba, el artefacto puede mantenerse solo para formación y pruebas
visuales, siempre en red local, con metadatos públicos verificables y datos
privados sintéticos, fuera del inventario de servicios productivos. Cada
publicación debe volver a ejecutar las pruebas de contenido y seguridad.

`skill_ref: admin-data-web`
