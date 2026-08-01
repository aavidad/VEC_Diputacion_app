# Decisión F0-D2d: auxiliar operativo privado del arnés

Fecha: 1 de agosto de 2026.
Estado: **decisión técnica cerrada; implementación y producción NO-GO**.

## Resultado

H0 incorpora un segundo auxiliar privado, dedicado exclusivamente al ciclo de
vida operativo del runner PostgreSQL 18.4:

```text
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  operaciones_runner_fuente_corporativa_contexto_actor_v1.sh
```

La [decisión F0 principal](decision_f0_capacidad_fuente_corporativa_2026-08-01.md)
conserva el protocolo funcional, el DAG y los write-sets. D2d solo separa una
responsabilidad probatoria que no cabe de forma legible en el runner H0 y que
no pertenece al auxiliar SQL fijado por D2c.

La dependencia exacta queda:

```text
D2c -> D2d -> H0
```

D2d no abre API, autoridad, despliegue ni tratamiento de datos. Las métricas no
cambian y producción permanece en `NO-GO`.

## Motivo de la separación

La revisión adversarial de H0 demostró que acreditar la propiedad por nombre
después de `docker run` deja una ventana entre la creación en el daemon y la
marca local. Cerrar esa ventana exige un protocolo revisable de intención,
token, etiqueta, identificador, `cidfile`, inspección, retirada y acreditación
de ausencia.

Comprimirlo en el runner agotaría la reserva de I0 y dificultaría revisar una
frontera de seguridad. Moverlo al auxiliar SQL contradiría D2c, que le prohíbe
Docker y decisiones de ejecución. Un auxiliar operativo separado mantiene una
sola responsabilidad y un límite de líneas verificable.

## Write-set y límites

H0 posee exactamente estos cuatro artefactos:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_fuente_corporativa_contexto_actor_v1.sh
  operaciones_runner_fuente_corporativa_contexto_actor_v1.sh
  capturar_snapshot_fuente_corporativa_contexto_actor_v1.go
```

Límites obligatorios:

- runner H0: como máximo 550 líneas;
- auxiliar operativo: menos de 200 líneas;
- auxiliar SQL: menos de 800 líneas;
- capturador Go: menos de 800 líneas.

La [corrección H0a](decision_f0_h0a_guardia_autoprueba_sintetica_2026-08-01.md)
es una corrección excepcional del primer escritor H0, descubierta al ejecutar
su primer consumidor A1. Después de H0a, I0 es el único escritor posterior del
runner. Ambos auxiliares y el capturador quedan inmutables; I0 conserva sus
tres SHA-256 literales y no los modifica.

## Fronteras de responsabilidad

El runner coordina etapas, crea el token efímero, instala las trampas antes de
reservar recursos y decide cuándo invocar `docker run`. Conserva los SHA-256
esperados como literales y carga ambos auxiliares solo desde una captura
privada acreditada.

El auxiliar operativo únicamente:

- descubre y acredita el contenedor de esta ejecución;
- cruza nombre, etiqueta, identificador y `cidfile` cuando exista;
- retira solo el recurso cuya propiedad completa ha demostrado;
- compara huellas host–contenedor;
- exige salidas exactas de las sondas operativas.

No contiene SQL, inventarios SQL, parser, clasificación SQLSTATE, política de
reintentos, credenciales persistentes ni reglas funcionales.

El auxiliar SQL conserva en exclusiva el analizador léxico, la clasificación
SQLSTATE, las clausuras, las rutas literales de etapa y el inventario del
snapshot SQL. No contiene Docker ni decisiones operativas.

El capturador Go mantiene la captura segura por descriptor y no administra
contenedores ni interpreta SQL.

Los dos auxiliares carecen de modo autónomo y fallan con estado `64` si se
ejecutan directamente en vez de cargarse desde el runner.

## Cadena de confianza

El runner fija como literales las rutas fuente y los tres SHA-256 esperados.
Con `umask 077`, crea un directorio privado `0700` y destinos nuevos `0600`,
sin seguir enlaces y con exclusión.

El arranque no es circular. El runner copia de forma acotada el fuente Go del
capturador, contrasta tipo, enlaces, dispositivo, inode, tamaño y tiempos antes
y después, verifica su SHA-256 literal y compila únicamente esa copia privada.
Solo el binario ya acreditado captura después, por descriptor, los dos
auxiliares y verifica sus SHA-256 literales antes de que el runner los cargue.

Cada una de las tres fuentes tiene un máximo exacto de 1.048.576 bytes. La
lectura se limita antes de recorrer, copiar, calcular huellas o reservar
memoria y consume como máximo 1.048.577 bytes, incluido el único byte de sonda
de exceso. Un artefacto ausente, enlazado, mutado, sobredimensionado o con SHA
distinto falla antes de compilar, cargar o abrir PostgreSQL.

El runner:

- ejecuta ShellCheck sobre los tres shells;
- analiza la copia acreditada del capturador con `go vet`;
- compila esa copia con `go build -race`, toolchain local, módulos de solo
  lectura y red de módulos desactivada;
- ejecuta el binario privado con `--autoprueba`;
- acredita el estado `64` de la invocación directa de ambos auxiliares.

Nunca usa `go run`, carga auxiliares desde el árbol vivo ni vuelve a abrir por
nombre un artefacto ya capturado.

## Protocolo de propiedad y limpieza

Antes de `docker run`, el runner genera un token aleatorio efímero, instala las
trampas y registra la intención de limpieza. El contenedor recibe:

- un nombre exacto reservado para la ejecución;
- la etiqueta `es.dipgra.vep.f0.propietario=<token>`;
- un `cidfile` privado y ausente al inicio.

La limpieza consulta el daemon por nombre y etiqueta. El resultado admisible
es cero o un contenedor; una cardinalidad mayor o una deriva falla sin eliminar
nada. Cuando existe uno, inspecciona su identificador, nombre y etiqueta.

El `cidfile`, si ya existe, debe ser un fichero regular con un único enlace. El
runner lo cambia inmediatamente a `0600`, antes de interpretar su contenido, y
vuelve a acreditar tipo, modo y enlaces. El directorio padre `0700` protege la
ventana entre la escritura del cliente Docker y ese cambio; una señal o fallo
en la ventana conserva un resultado no cero y limpia por la propiedad
independiente, sin confiar en el fichero.

El contenido admisible ocupa exactamente 64 bytes hexadecimales o 65 bytes si
añade un único salto de línea final; no admite otro espacio ni una segunda
línea. Debe coincidir con el identificador inspeccionado. Su lectura se limita
a 65 bytes más un byte de sonda antes de interpretar el contenido. Si está
corrupto, excede ese límite o no coincide, la ejecución conserva un resultado
no cero, pero esa entrada no confiable no impide retirar el contenedor
acreditado de forma independiente por nombre, etiqueta e identificador ni
autoriza a retirar otro. Si el cliente Docker falló antes de escribirlo, nombre
y etiqueta permiten recuperar y reacreditar el identificador sin convertir la
ausencia del fichero en éxito.

Solo la coincidencia conjunta de nombre, etiqueta y identificador autoriza la
retirada. Una coincidencia aislada por nombre, `cidfile`, etiqueta o
identificador nunca autoriza a borrar un recurso.

Tras solicitar la retirada, el auxiliar vuelve a consultar el daemon. El token
y la intención solo se liberan después de acreditar cardinalidad cero. Un fallo
de `docker rm` seguido de ausencia acreditada es cierre; un fallo con el
contenedor presente mantiene la intención y termina con estado no cero.

## Matriz adversarial de H0

H0 debe demostrar, como mínimo:

- un contenedor ajeno con el nombre exacto pero sin etiqueta sobrevive;
- un contenedor propio etiquetado sin `cidfile` se recupera y retira;
- un `cidfile` corrupto o ajeno no autoriza borrado y el recurso propio se
  recupera por nombre y etiqueta;
- `SIGINT` y `SIGTERM` después de crear el recurso devuelven respectivamente
  `130` y `143`, sin contenedor ni etiqueta residual;
- un fallo de `docker run` posterior a la creación deja cero residuos;
- un fallo de `docker rm` después de retirar se acepta solo tras reacreditar
  ausencia;
- un fallo de `docker rm` con el contenedor presente termina no cero y conserva
  la intención;
- ninguna coincidencia parcial elimina un contenedor;
- todos los temporales, tokens, etiquetas, sesiones y contenedores quedan
  ausentes al cerrar una ejecución válida.

Las pruebas de propiedad no usan temporizadores para decidir pertenencia. Las
señales y errores conservan el estado de salida causal; la limpieza nunca los
convierte en éxito.

## Límites del auxiliar SQL

Antes de contar líneas o invocar `awk`, el auxiliar SQL acredita que cada
componente es un fichero regular no vacío y mide su tamaño con una sonda
acotada. Rechaza todo componente mayor de un mebibyte. Solo después ejecuta el
analizador léxico y el resto de comprobaciones fijadas por D2c.

## Criterio de cierre

H0 solo puede confirmarse cuando:

1. los cuatro artefactos respetan su write-set y sus límites;
2. los tres SHA-256 literales coinciden con las capturas privadas;
3. todas las pruebas adversariales de propiedad, señales, tamaño y mutación
   pasan;
4. el runner completo pasa tres veces sobre PostgreSQL 18.4 por digest;
5. ShellCheck, `go vet`, `go build -race`, autoprueba, `git diff --check`,
   Gitleaks y ausencia de residuos quedan verdes;
6. dos revisores independientes emiten `GO`, con `P0=P1=P2=0`.

Hasta entonces no existe una migración `000007` instalable y las métricas
funcionales permanecen sin cambios.
