# Recorrer el baremador local en Chrome

El script `scripts/probar_baremador_navegador.mjs` abre el editor local y comprueba
sus cálculos contra el servidor Go. Usa los ejemplos sintéticos incluidos en el
repositorio. No lee expedientes, bases de datos ni archivos del operador.

Desde la raíz del repositorio:

```sh
node scripts/probar_baremador_navegador.mjs
```

El recorrido completo exige integrar el motor y la interfaz propios de Concursos.
Para comprobar únicamente Bolsa durante esa integración:

```sh
node scripts/probar_baremador_navegador.mjs --solo-bolsa
```

Esta opción queda registrada en el informe; no acredita Concursos.

El script reutiliza Playwright instalado y `/usr/bin/google-chrome`. Puede fijar
otros ejecutables locales mediante `PLAYWRIGHT_MODULE`, `CHROME_BIN` y `GO_BIN`.
No descarga navegador, SDK, toolchain ni dependencias Go. Si no indica `GO_BIN`,
el selector del repositorio exige la versión exacta declarada en `go.mod`.

Compila solamente `cmd/vec-baremador-web`, arranca su propio proceso en
`127.0.0.1` con puerto asignado por el sistema y abre contextos nuevos de Chrome.
No reutiliza un servidor, perfil o sesión existentes. Al terminar, elimina el
binario y los archivos temporales de importación/exportación, cierra Chrome y
detiene exclusivamente el grupo de procesos que creó. También contempla la
interrupción y un límite total de ocho minutos.

## Qué comprueba

El recorrido visita español e inglés a 1440, 1024 y 390 píxeles. En cada visita
comprueba el idioma del documento, navegación por teclado hasta la comparación,
foco visible sin obstrucción y ausencia de desbordamiento horizontal. En
escritorio comprueba además que la página no se desplaza verticalmente y que las
acciones frecuentes quedan a la vista. Las tablas anchas deben desplazarse
dentro de su contenedor. En los segmentos nativos de fecha, compara imágenes del
control antes y después del foco; no exige un contorno en el elemento exterior.

Añade una visita por idioma con un ancho CSS de 720 píxeles y escala de
dispositivo 2. Comprueba el reflujo equivalente al 200 % sobre 1440 píxeles;
no acredita haber accionado el zoom nativo de Chrome. Las capturas y las
medidas quedan diferenciadas en el informe.

| Caso de Bolsa | Resultado esperado, en micropuntos |
| --- | --- |
| Experiencia original | 101667 |
| Coeficiente de experiencia de 0,1 a 0,2 | 203333 |
| Otros méritos originales | 4000000 |
| Tope total de otros méritos de 4 a 3 | 3000000 |

Concursos usa su configuración y entrada propias, con seis familias. El ejemplo
da `28386027` micropuntos. Cambiar el coeficiente de antigüedad de 0,1 a 1 debe
dar `39186027`. El recorrido contrasta grado, valoración del trabajo, antigüedad,
permanencia, cursos y titulaciones con los importes previstos del ejemplo Go.
Comprueba también la huella propia de Provisión, que se calcula con el campo de
huella del resultado vacío.

Comprueba las dos respuestas HTTP de cada comparación, su alcance de simulación
y la huella SHA256 del resultado. Repetir la misma configuración debe producir
el mismo objeto y huella. No sustituye respuestas de red ni introduce otro motor
de cálculo en JavaScript.

El borrador de cada módulo debe conservarse al cambiar de panel y después de importar JSON
roto, un archivo excesivo o reglas de otra familia. Un campo negativo debe
mostrar su error, impedir la exportación y recibir el foco al comparar. Una
configuración con un campo desconocido debe ser rechazada por Go sin perder
el borrador. Una petición JSON rota debe devolver un error JSON real.

También comprueba ayuda desplegable, exportación y reimportación. Vigila
peticiones fuera del origen local, credenciales HTTP, `Set-Cookie`, cookies,
escrituras de almacenamiento web, IndexedDB, cachés y registros de service
workers. Los errores de ejecución JavaScript y los errores inesperados de
consola hacen fallar el recorrido. Chrome puede escribir un diagnóstico de
red para los rechazos 400/422 provocados: se contabiliza por separado.

## Evidencia y límites

Cada ejecución crea un directorio nuevo `vec-baremador-evidencia-*` en el
directorio temporal del sistema. La consola muestra su ruta. Contiene capturas
y `resultado.json`, con commit, cambios locales, versiones de Go y Chrome,
casos ejecutados, estados HTTP, puntuaciones, huellas y resultado final.
Una interrupción o un fallo dejan evidencia con ese estado y código de salida 1.
Solo `estado: pasado` acredita los casos enumerados en ese informe.

La herramienta sigue siendo una simulación local. Este recorrido no acredita
aprobación de bases, puntuación oficial, publicación, permisos institucionales,
persistencia, integración con Personal ni cumplimiento global de accesibilidad.
Las capturas permiten una revisión visual independiente antes de integrar.

Validación del guion: `node --check scripts/probar_baremador_navegador.mjs`,
`node scripts/probar_baremador_navegador.mjs --help` y `git diff --check`.
La prueba provisional de Bolsa llegó a los cálculos y a la conservación del
borrador; no completó la matriz por el 404 del icono solicitado por Chrome.
La corrección corresponde al transporte web y debe entrar antes de ejecutar
el recorrido completo. No hay un informe aprobado de la matriz en este commit.
