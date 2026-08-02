# Enmienda F0-H0b: auxiliar privado y snapshot aislado de R0

Fecha: 2 de agosto de 2026.

Estado: **doble `GO` documental, `P0=P1=P2=0`; implementación pendiente**.

## Motivo

El candidato H0b `99491d3` demostró que los dos subensayos R0 son viables,
pero recibió doble `NO-GO` antes de integrarse. Sustituyó la autoprueba H0a y
trasladó Docker, sesiones y decisiones de ejecución al auxiliar SQL, contra la
frontera D2c. Restaurar literalmente H0a sobre el mismo árbol tampoco es
correcto: el snapshot H0b ya contiene M010...M070 y la autoprueba usa y retira
deliberadamente M010/T010.

No se relajan límites ni se minifica código de seguridad. Esta enmienda
sustituye únicamente la composición probatoria de H0b; no cambia C2, R0
productivo, una métrica funcional ni el `NO-GO` de producción.

## Write-set corregido

H0b puede modificar exclusivamente:

```text
deploy/postgresql/autorizacion_atestada_v3/
  probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/
  arnes_fuente_corporativa_contexto_actor_v1.sh
  arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh
```

El tercer fichero es un auxiliar privado de prueba, no una migración, un
fixture productivo ni una fachada. Debe quedar preferiblemente por debajo de
400 líneas y siempre por debajo de 800. El runner conserva el máximo de 550;
el auxiliar SQL y el capturador permanecen por debajo de 800 y el auxiliar
operativo D2d, por debajo de 200.

## Fronteras exactas

El runner sigue siendo dueño de:

- el orden exterior H0a -> línea base -> H0b sin R0 -> línea base -> R0 ->
  H0b focal -> retirada -> línea base;
- temporales, contenedor, sesiones, credenciales efímeras y propagación de
  estados;
- captura, acreditación y carga de todos los auxiliares privados;
- la invocación exclusiva bajo `etapa=H0` de la autoprueba H0a nominal/error.

El auxiliar SQL D2c vuelve a contener solo análisis, clasificación, rutas,
clausuras, inventarios, snapshot y huellas. No contiene Docker, sesiones,
roles, fixture R0 ni decisiones de ejecución.

El auxiliar operativo D2d permanece byte a byte inmutable y conserva solo la
propiedad, descubrimiento y retirada del contenedor. No recibe SQL, R0 ni
fixtures.

El nuevo auxiliar H0b:

- falla con estado `64` si se ejecuta directamente o sin carga privada;
- contiene la plantilla virtual C2, las identidades sintéticas, el catálogo
  exacto de diez roles y siete aristas y los oráculos R0;
- usa exclusivamente primitivas acreditadas del runner para copiar, ejecutar
  SQL y consultar valores; no crea ni retira contenedores, no conoce secretos,
  no abre red ni implementa reintentos;
- no crea ficheros ficticios M080/T080 en el árbol Git ni los incorpora al
  inventario productivo;
- deja toda creación y retirada R0 dentro de transacciones explícitas y hace
  que el runner acredite la línea base antes de propagar un error.

Después de que R0 pueda haberse confirmado, todo retorno atraviesa un único
finalizador. Este conserva el estado causal original, intenta sin cortocircuito
retirar fixture, componentes y wrapper, acredita ausencia y ambas raíces y
solo entonces lo propaga. Cualquier fallo de limpieza sustituye el estado por
`65`. Las trampas de señal y salida continúan retirando el contenedor como
última defensa, pero no reemplazan este finalizador.

El runner fija como literales la ruta y la SHA-256 del nuevo auxiliar. El
capturador lo copia con el mismo protocolo privado de los demás auxiliares,
se contrasta el manifiesto antes de `source`, se prueba su rechazo autónomo y
ShellCheck analiza la copia acreditada. I0 deberá conservar byte a byte el
auxiliar y su SHA, igual que los otros tres artefactos privados.

## Una captura coherente y dos raíces sin colisión

En `etapa=H0` el capturador realiza una sola lectura viva del inventario
superset base + M010...M070 antes de abrir PostgreSQL. Produce un snapshot y
un manifiesto únicos. Sin volver a leer el árbol vivo, el runner deriva por
lista positiva el manifiesto base, copia el mismo snapshot privado a dos
raíces y elimina de `/repo` únicamente los componentes M y directorios vacíos
exactos:

1. `/repo` queda acreditado contra el manifiesto base, sin M010...M070; allí se ejecuta la
   autoprueba H0a nominal/error exactamente sobre sus M010/T010 sintéticos y
   se acredita su limpieza;
2. `/repo_h0b` conserva el superset con M010...M070 reales; solo allí se
   materializan temporalmente las plantillas virtuales M080/T080.

Cada raíz tiene modo `0700`, manifiesto y huellas propios. Tras la única
captura, el analizador valida los bytes del snapshot superset privado, nunca
los del árbol vivo; ninguna derivación, copia o apertura de PostgreSQL ocurre
antes. El filtrado del manifiesto base exige las siete rutas literales y no usa
patrón, prefijo ni descubrimiento dinámico. Después de retirar los siete M, elimina el único directorio hoja
`migraciones/000007_componentes` solo si es real, no simbólico, modo `0700` y
está vacío; cualquier otra forma o contenido deniega. No elimina otro padre.
El runner rechaza enlaces,
ficheros adicionales, modos inseguros, divergencia byte a byte o residuos en
cualquiera de las dos raíces. El wrapper H0b resuelve sus `\ir` solo dentro de
`/repo_h0b`; nunca sobrescribe ni restaura M010/T010 de `/repo`.

Los M/T sintéticos de H0a y las plantillas M080/T080 no pertenecen a la
captura superset porque se materializan después. Antes de cada copia el runner
los somete a `validar_componentes_sql_f0`; después acredita igualdad SHA-256
host-contenedor y solo entonces permite que `psql` los lea.

Para A1--T2 solo existe el snapshot focal ordinario de su clausura. La doble
raíz es una autoprueba exclusiva de H0 y no altera las etapas reales.

## Matriz de cierre

Antes de integrar H0b se exige sobre una huella congelada:

1. `bash -n`, ShellCheck, `git diff --check`, límites y Gitleaks;
2. rechazo autónomo de los tres auxiliares shell y autoprueba del capturador;
3. cuatro SHA-256 exactas y manifiesto de tres filas para los auxiliares shell;
4. captura superset única, validación posterior de sus bytes privados y
   derivación exacta de los dos manifiestos antes de abrir PostgreSQL;
5. tres H0 limpios en PostgreSQL 18.4 mostrando por separado H0a y las dos
   integraciones virtuales C2;
6. H0a nominal y de error sin sobrescribir la raíz H0b, con M/T validados,
   huella de copia y directorio hoja retirado al volver a la base;
7. instalación M010...M080 sin R0, M080/T080 validados y ligados por huella,
   `42501` exacto, rollback y cero efectos;
8. R0 exacto: diez roles, siete aristas, atributos, `grantor=datdba`, ajustes,
   descripciones y etiquetas;
9. integración virtual C2 nominal y con error posterior, retirada previa a
   propagar y retorno a catálogo, sesiones, roles y filesystem de base;
10. inyección de fallo después de cada frontera posterior al `COMMIT` R0:
   acreditación, materialización, copia, huella, wrapper, sesión y comprobación
   final; todas atraviesan el finalizador y conservan estado salvo fallo de
   limpieza;
11. inyección independiente dentro del finalizador al retirar R0, wrapper y
   M080/T080, y al acreditar ausencia R0, `/repo_h0b`, `/repo`, catálogo,
   checkpoint, roles, sesiones, objetos y temporales; un centinela demuestra
   que las acciones posteriores aún se intentan sin cortocircuito y cada caso
   devuelve `65`, mientras la trampa final acredita cero recursos externos;
12. C1 real, A1 real nominal y el mutante A1 que falla después de crear un
   objeto;
13. dos revisiones independientes finales con `P0=P1=P2=0`.

La plantilla virtual versionada es prueba reproducible autorizada; lo que no
se versiona son componentes ficticios bajo las rutas catalogadas M080/T080.
H0b sigue sin acreditar la implementación real de C2.
