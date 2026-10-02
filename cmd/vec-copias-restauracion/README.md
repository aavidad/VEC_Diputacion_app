# Propuestas de restauración offline (CS10)

Esta CLI guarda una propuesta y la revisión de otra persona en un registro local
segregado. Recupera ambas tras volver a abrir el registro y compara la versión
esperada antes de añadir una revisión. No conecta con servidores ni sustituye datos.

El ejemplo contiene personas, tiempo, preimagen y permisos declarados. Siempre
informa `autorizacion: no_comprobada` y `habilita_restauracion: false`. Sus huellas
protegen la coherencia de la declaración; no autentican a quien la presenta.
La preimagen declarada no acredita los bytes actuales del destino.

## Recorrido local

Prepare un directorio propio vacío, con permisos `0700`, fuera de los volúmenes que
se restaurarían. El programa no lo crea y solo escribe `propuestas.v1.jsonl` dentro.

```sh
mkdir -m 700 ./registro-sintetico
GOTOOLCHAIN=local go run ./cmd/vec-copias-restauracion \
  -registro ./registro-sintetico \
  < cmd/vec-copias-restauracion/testdata/proponer.json
```

Para la segunda revisión, copie la entrada de ejemplo en un archivo de trabajo:

- Cambie `accion` a `revisar` y `persona_declarada_ref` a `persona:sintetica:2`.
- Copie en `sha256` el sello que devuelve la propuesta y ponga `version: 1`.
- Mantenga la propuesta, el manifiesto, la política y el destino originales.

Envíe ese archivo con el mismo comando. La salida conserva la propuesta y añade la
revisión como versión 2. Con `accion: consultar` puede recuperarla después de cerrar
la CLI. Repetir el alta idéntica devuelve el registro existente; repetir una revisión
con versión 1 devuelve conflicto y no añade otra revisión.

El ejemplo usa un instante sintético fijo. `ahora` es una declaración del ejercicio,
no un reloj seguro. Todos los campos son obligatorios, salvo que `doble_control`
puede ser `null`: ese valor exige dos personas. Se rechazan claves duplicadas,
campos desconocidos, objetos incompletos y entradas mayores de 4 MiB. La salida
contiene claves de los catálogos `copias_restauracion.json`, sin textos de idioma
embebidos en Go. Códigos: 0 declaración guardada/consultada; 2 bloqueo; 3 formato o
argumentos; 4 fallo al escribir la salida.

El doble control está activo por defecto. Solo el ejercicio `sintetico_offline`
admite desactivarlo expresamente con `doble_control: false`. El servicio operativo
rechaza esa configuración. La CLI no permite acciones de sustitución.

## Servicio y fronteras de integración

`application/restauracioncopias.Servicio` prepara propuestas, registra revisiones y
comprueba las precondiciones antes de cercar una operación. Usa referencias de sesión
de una frontera confiable; `Autoridad` resuelve la Persona canónica y exige concesión
exacta actual. Revalida a proponente y revisor, incluida la revocación, también después
de observar el destino. El formulario no aporta roles ni la Persona autorizada.

La propuesta sella conjunto, destino, preimagen completa, motivo, ventana, política,
Persona proponente, caducidad y configuración. Cambiar cualquiera de ellos exige una
propuesta nueva. CS01 bloquea estados `incompatible` y `no_comprobable`. El observador
operativo debe verificar autenticidad y bytes; un booleano enviado por el cliente no
es ese proveedor. El estado de verificación debe ser `valida`.

Antes del cercado se exige la misma preimagen bajo exclusión de escritores, una copia
previa completa autenticada y verificada del destino actual, otra Persona vigente,
ventana y permiso de sustitución actuales, versión y sello CAS. La copia previa
pertenece a la misma exclusión, empieza dentro de la ventana y termina/verifica antes
del momento de decisión. La restauración recupera el conjunto completo con su binario;
la pérdida y conciliación de cambios posteriores deben explicarse antes de aprobar.

El registro local usa bloqueo de archivo, CAS, historia enlazada por SHA256 y `fsync`
de archivo/directorio. Acepta solo declaraciones sintéticas, limita el journal a 8 MiB
y deniega si detecta manipulación o una escritura cortada. SHA256 no impide que el
propietario reescriba toda la historia; este archivo no sustituye auditoría externa.
Tras una escritura cortada, conservar el archivo para diagnóstico: no hay reparación
automática ni garantía de disponibilidad. Está preparado para sistemas Unix locales
con `flock` y `fsync`; no se afirma compatibilidad con sistemas de archivos de red.

La composición operativa necesita autoridad V3 real y un registro externo que consuma
concesión y CAS con auditoría en la transacción aplicable. El puerto `RegistroPropuestas`
expresa ese contrato; este adaptador offline no lo acredita ni permite cercar. CS07
conserva su registro de captura/ensayo: las propuestas no son otro estado de esa copia.
CS11 debe consumir de nuevo permiso, exclusión y CAS al sustituir; un resultado previo
de CS10 no es una autorización transferible. No hay ejecutor ni sustitución en CS10.

## Comprobación focal

```sh
GOTOOLCHAIN=local go test -p 8 -race \
  ./internal/modules/administracion/domain/restauracioncopias \
  ./internal/modules/administracion/application/restauracioncopias \
  ./internal/modules/administracion/adapters/restauracioncopias \
  ./cmd/vec-copias-restauracion
```

Las pruebas cubren misma Persona, propuesta alterada/expirada, permiso revocado,
preimagen distinta, copia previa ausente, incompatibilidad, CAS, recuperación del
registro y ocho revisiones concurrentes con un único resultado. No acreditan una
restauración, permisos V3, PostgreSQL, montaje ADMIN ni recorrido de navegador.
