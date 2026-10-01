# Recorrido local de Convoca

El guion consulta dos procesos locales al mismo tiempo con Chrome del sistema y
Playwright Python. Comprueba el listado público gobernado, abre su primera ficha
y consulta Mi bolsa y Mis preferencias en el proceso externo. Captura también la
ficha propia. Reutiliza las utilidades de auditoría y captura de `revision_web`.

Ambos procesos deben contener exclusivamente datos sintéticos. El guion no los
arranca ni publica permisos. Rechaza destinos fuera de loopback, TLS no válido,
peticiones que puedan escribir y recursos servidos por otro origen. Los perfiles
de Chrome son efímeros. No recibe cabeceras libres de identidad.

Para un proceso externo HTTPS que exige mTLS, indique juntas
`--certificado-externo` y `--clave-externa`. Deben ser rutas absolutas fuera de
Git, sin enlaces, de archivos regulares propios con permisos 0600 y directorio
privado. El material se aplica solo al contexto externo y al origen indicado;
el certificado del servidor se sigue verificando. El guion no genera material
ni lo importa a NSS. El informe guarda únicamente si se configuró, nunca rutas
ni contenidos. Configurar mTLS no acredita autenticación, permiso o inscripción.
Sin material, el guion conserva el diagnóstico de las dependencias: los HTTP 200
del fixture no acreditan la frontera mTLS ni un acceso H6 real.

```sh
python3 scripts/recorridos/convoca_integrado.py \
  --url-publico http://127.0.0.1:PUERTO_PUBLICO \
  --url-externo http://127.0.0.1:PUERTO_EXTERNO \
  --datos-sinteticos \
  --expectativas scripts/recorridos/convoca_expectativas.json \
  --salida /ruta/propia/nueva/recorrido
```

Los puertos los fija quien mantiene los procesos. La salida debe ser nueva y
quedar fuera de Git: contiene capturas sintéticas e informe JSON. El informe
conserva estados HTTP, cantidades y códigos de error; omite cuerpos, personas,
contacto, credenciales y mensajes de consola. `--identificador-publico` permite
elegir una ficha que esté en la primera página del listado validado.

Las expectativas de ejemplo exigen consultas públicas disponibles y HTTP 403
para las dependencias externas cerradas. Cambie esas expectativas conforme al contrato
del ejercicio. Una respuesta cerrada distingue 401, 403, 404 y 503. Solo se
atribuye a H6 cuando su código coincide con `codigos_h6`, informado por el dueño
de la dependencia. El estado HTTP por sí solo no identifica H6.
Las lecturas auxiliares de imagen y correos que hace Mis preferencias tienen
expectativas cerradas separadas. Otros errores de red siempre hacen fallar el
guion; no se admite una excepción general para las rutas de la API.
También falla ante una petición sin respuesta HTTP o un error de consola
inesperado. Un diagnóstico de recurso de Chrome solo se admite si coincide en
ruta y estado con una respuesta cerrada prevista; cada respuesta permite un solo
diagnóstico. Los mensajes de consola y los detalles libres del fallo no se guardan.

El código de salida es 0 cuando las cuatro consultas son válidas y se cumplen
las comprobaciones de página; 2 cuando se confirma una dependencia cerrada; 1
si hay una discrepancia, error de página o comprobación omitida. Los errores de
configuración también devuelven 2 mediante argparse. Una dependencia cerrada
siempre deja `flujo_funcional_completado: false`. Una prueba con servidores
sintéticos de la herramienta debe usar `--tipo-ejecucion prueba_guion`.

El listado y el detalle usan el validador V2 de la web. Mi bolsa y preferencias
usan sus consumidores existentes. La ficha propia se captura sin atribuirle una
consulta independiente que no tenga contrato. Las dos anchuras son 1440 y
390 px; la auditoría cubre controles etiquetados, desbordamiento y ausencia de
cookies y almacenamiento web. Las capturas requieren revisión visual humana.

Añada `--preparacion-local` cuando el proceso sirva `/bolsa/preparacion/`.
Comprueba revisión de bases, selección de un archivo sintético, resumen y
descarga UTF-8 del borrador. Verifica que el borrador conserve el nombre del
archivo, sin su contenido, y que Presentar quede deshabilitado con explicación.
La salida guarda la huella y tamaño del borrador; el navegador elimina la
descarga efímera al cerrar el contexto. También comprueba Volver al paso anterior.

Este guion no acredita presentación, firma, registro, pago, envío ni
persistencia. Tampoco reinicia aplicación o base de datos. La consulta válida o
el borrador local no cierran el recorrido funcional de inscripción.

Para comprobar los límites del guion:

```sh
python3 -m unittest discover -s scripts/recorridos -p test_convoca_integrado.py -v
```

El archivo `convoca_fixture.json` conserva los DTO sintéticos del ensayo. Con
el consumidor de preparación integrado, puede arrancar y cerrar dos servidores
efímeros de loopback para repetir la comprobación de la herramienta:

```sh
python3 scripts/recorridos/test_convoca_integrado.py \
  --ensayar-chrome --salida /ruta/propia/nueva/ensayo
```

Este ayudante devuelve 0 si el guion detecta el cierre externo previsto con
código 2 y conserva las acreditaciones en falso. Ejecútelo en el entorno aislado
local de pruebas; el fixture solo se usa en este ayudante.

Para probar únicamente las dos consultas externas HTTP 200 y sus importaciones:

```sh
python3 scripts/recorridos/test_convoca_integrado.py \
  --ensayar-chrome --escenario externo-200 --salida /ruta/propia/nueva/imports
```

Este escenario usa el DTO sintético de Mi bolsa ya probado por su adaptador y
el catálogo v1 de preferencias probado por el cliente. Comprueba las funciones
exportadas desde las URL que cargó la página, rechaza contratos malformados sin
perder su HTTP 200 e impide consultar si el módulo no aparece montado. También
detecta una emisión `Set-Cookie` expirada sin conservar su valor, aunque Chrome
no retenga la cookie. La detección usa todas las cabeceras de respuesta.

La CLI usa nombres de argumentos y códigos nominales, sin mensajes narrativos.
`configuracion_invalida` indica que debe revisar URL de loopback, expectativas,
Chrome, tiempo máximo y directorio nuevo. El informe mantiene los estados HTTP
recibidos, incluso si su contrato es inválido. Los módulos externos se importan
desde la URL que ya cargó la propia página, con su versión de caché.

El ensayo de la herramienta del 1 de octubre usó dos servidores HTTP sintéticos
propios, sin conexión exterior, y Chrome del sistema. El listado y detalle V2
respondieron 200; la preparación produjo dos borradores locales de 692 bytes a
1440 y 390 px. Las consultas externas devolvieron 403 previsto. El guion terminó
con código 2 y conservó registro, persistencia y flujo completo en falso.
Las doce capturas pasaron las comprobaciones de DOM, almacenamiento y
desbordamiento. Este ensayo comprueba la herramienta y su contrato de UI;
no acredita instalación de servidores VEC, permisos o PostgreSQL.

La comprobación focal posterior de HTTP 200 pasó siete casos en Chrome:
dos contratos válidos, dos contratos malformados rechazados manteniendo HTTP 200,
dos módulos ausentes sin consulta y una cookie expirada detectada sin retenerla.
Mi bolsa usó `contrato.js` sin query; preferencias usó la URL observada de
`cliente-http.js?v=20260930-temas-v2-historico-v1`. El informe identifica ambas
funciones exportadas y mantiene registro, persistencia y flujo completo en falso.

El último parche pasó cuatro pruebas focales nuevas: fallo de red sin HTTP con
ocho capacidades conformes, error de consola inesperado con ese mismo agregado,
cierre HTTP previsto con código 2 y configuración mTLS por contexto mediante
un espía del SDK. La validación de material incluye pareja de archivos, HTTPS
loopback, permisos, propietario, directorio privado y rechazo de enlaces o Git.
Estas pruebas no usan certificados reales ni acreditan una conexión mTLS.
