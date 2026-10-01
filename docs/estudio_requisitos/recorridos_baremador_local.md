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

Para revisar solo las explicaciones de Concursos después de una corrección de
textos o formato:

```sh
node scripts/probar_baremador_navegador.mjs --explicaciones
```

Esta comprobación visita español e inglés a 1440 y 390 píxeles. Abre los
desgloses por teclado y contrasta los motivos visibles con los datos que
devuelve Go, conservando los decimales. Comprueba las tablas de grado y sus
topes. Cambia el corte a `2024-05-01`, fecha exacta del curso, y comprueba su
exclusión y explicación. Ejecuta 16 POST en total; no repite la matriz de 120.
El informe distingue este modo de la matriz completa.

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

La matriz completa pasó el 1 de octubre de 2026 sobre el árbol de comprobación
limpio `28a99e69dc7d69cf257786234cfb57f3e8929d44`, con el guion
`845a94da969fb1059571174f116cc32bbbd696b8`, web `27508ddb2` y motor `10b883`.
Comando: `node scripts/probar_baremador_navegador.mjs`. Usó Go 1.26.6 y Chrome
149.0.7827.200. Los seis recorridos funcionales y las dos visitas de reflujo
equivalente pasaron. Hubo 120 POST: 96 respuestas 200, 18 rechazos 400 y seis
rechazos 422 provocados por el guion. No hubo errores JavaScript, errores de
consola inesperados, cookies, almacenamiento web ni llamadas externas.

El informe de esa ejecución está en
`~/.local/state/vec-codexb-concursos-20261001/evidencia/matriz/resultado.json`, con SHA256
`5e2c1bbf05d226e79942ef9aafd599bab5443186b69297b24630e39f53a06e1b`.
El directorio conserva 16 capturas. Son artefactos temporales fuera de Git;
dirección debe conservar las capturas que use en la entrega. Aquellas capturas
se tomaron después del teclado, con posición de desplazamiento variable. El
guion ahora encuadra el inicio de los paneles y tablas en capturas de la ventana;
los detalles tienen capturas propias al desplazarse hasta ellos.

La revisión visual posterior encontró una explicación que redondeaba a cero
coeficientes y puntos decimales, pese a que el total y el desglose del motor eran
correctos. La matriz anterior acredita sus comprobaciones automáticas; no
aprueba aquellos textos explicativos.

La comprobación focal de la corrección pasó sobre la candidata limpia
`5536d3c1e13215ef16148562788f5e2fd8789a54`, que incluye el renderer y catálogos
de web `f5a0ba308b38f7975a06f4a23dee3ad0a4117cb5`. Comando:
`node scripts/probar_baremador_navegador.mjs --explicaciones`. Los cuatro casos
de español e inglés a 1440 y 390 píxeles pasaron, con 16 POST que devolvieron
200. Los motivos de las seis familias conservan las cantidades que devuelve
Go. El curso muestra un coeficiente de 0,012 y 0,48 puntos; el grado conserva
19 puntos y su tabla. El corte `2024-05-01` excluye el curso obtenido ese mismo
día, muestra la explicación traducida y devuelve un total de `23129315`
micropuntos. No hubo errores JavaScript, errores de consola, cookies,
almacenamiento web ni llamadas externas.

El informe focal está en `~/.local/state/vec-codexb-concursos-20261001/evidencia/explicaciones/resultado.json`,
con SHA256 `32126ba2a423974f27cda7110440955945bf7828c27f02cfe11ec0ab468db319`.
Conserva 20 capturas de ventana: inicio de Bolsa y Concursos, curso abierto,
tabla de grado y curso excluido. La revisión visual independiente recibe estas
capturas. No se repitió la matriz de 120 POST ni las visitas de reflujo.
La corrección web afecta únicamente a textos, representación y versiones de
recursos; no cambia el transporte HTTP ni el motor acreditados por esa matriz.
