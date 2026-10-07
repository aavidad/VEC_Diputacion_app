# Simular el progreso y la recuperación de una copia

CS07-A ofrece una CLI para comprobar el modelo de operaciones con JSON sintético.
Reconstruye la historia y muestra qué falta comprobar después de una interrupción.
No conecta servidores, escribe ficheros, copia datos, autentica evidencias ni ejecuta
restauraciones. No es el registro durable CS07 ni concede permisos.

Desde la raíz del repositorio:

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-copias-operacion \
  -textos web/static/textos/es/copias-operacion.json \
  < cmd/vec-copias-operacion/testdata/doble-ensayo.sintetico.json

GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-copias-operacion \
  -textos web/static/textos/es/copias-operacion.json \
  < cmd/vec-copias-operacion/testdata/interrupcion.sintetica.json
```

El primer ejemplo devuelve `estado: verificada_declarada`, `version: 5` y recomienda
autenticar las evidencias antes de usar la copia. El segundo conserva
`estado: capturando`, `version: 1` y recomienda conciliar el resultado antes de
repetir una captura. El catálogo inglés está en la misma ruta bajo `en`.

Todas las respuestas correctas incluyen `alcance: simulacion_offline`,
`autenticidad: no_comprobada`, `habilita_copia: false`,
`habilita_restauracion: false` y `registro_durable: false`.
Los resultados declarados por quien prepara el JSON no prueban una restauración real.

## Contrato y recuperación

La entrada declara `sintetica: true`, una `solicitud`, una lista opcional `historia`
y `comandos`. El límite es 1 MiB y 128 elementos entre historia y comandos. La CLI
termina tras 30 segundos si la entrada queda bloqueada. Rechaza campos desconocidos,
claves repetidas en el mismo objeto y nombres de campo que no usen su forma exacta
en minúsculas. Rechaza también JSON adicional, referencias vacías y huellas que no
sean SHA256 hexadecimal minúscula.
Los errores no muestran el contenido original ni rutas privadas. Un conflicto de
versión publica la clave comparada y sus valores numéricos.

La solicitud fija operación, clave idempotente, sello de solicitud y referencias
opacas de conjunto, destino y política. No contiene nombres, rutas, secretos ni
permisos. El sello lo aporta el llamante: no se autentica en este ejercicio.

La secuencia admitida es:

1. `iniciar_captura`, desde `solicitada`.
2. `confirmar_captura`, declarando la huella del manifiesto.
3. `iniciar_verificacion`, declarando la misma huella y una ejecución.
4. `declarar_ensayo` físico y lógico, en cualquier orden, con evidencias ligadas al
   mismo conjunto, manifiesto y ejecución.

Un solo ensayo satisfactorio deja `verificando`. Cualquier ensayo fallido deja
`no_valida_declarada`. Dos ensayos satisfactorios dejan `verificada_declarada`.
Las evidencias representan declaraciones; sus referencias y huellas no sustituyen
el verificador CS06 ni la autenticación del conjunto CS03.

Cada comando nuevo exige `version_esperada` y `solicitud_sha256` exactos. Añade un
evento con secuencia, versiones previa y resultante, vínculos, sello semántico y
estado. La misma clave y el mismo contenido recuperan el evento original sin añadir
otro, incluso después de pasos posteriores. Cambiar el contenido con la misma clave
produce conflicto. Cambiar la versión esperada del reintento también cambia su sello.

El paquete de dominio permite recuperar una operación con `Reconstruir(solicitud,
historia)`. La CLI admite esos eventos en `historia`: reproduce cada transición y
compara el evento completo. Rechaza saltos, duplicados, orden alterado, vínculos
sustituidos y estados o sellos adulterados. No reordena ni repara la historia.
La salida resume los comandos nuevos sin volcar las referencias de entrada.

La recomendación de reconciliación solo pide observación y revalidación. No calcula
un permiso, certifica estado de plataforma ni relanza un efecto incierto. Las fases
incompletas siguen pendientes. Un ensayo fallido requiere revisar su causa; este
modelo no reescribe evidencia ni convierte el fallo en éxito con otro comando.

## Integración pendiente

El dominio mantiene inmutabilidad y append-only dentro del proceso. `Reservar`
compara una reserva existente; no impone unicidad entre procesos. El adaptador CS07
posterior debe hacer reserva y CAS atómicos, persistir historia y recibos fuera del
conjunto restaurado y consumir autorización y auditoría reales por los puertos
comunes. No deben usarse las entradas de esta CLI como fuente confiable de control.

La CLI no evalúa compatibilidad: corresponde a CS01 y debe revalidarse antes de
efectos reales. Tampoco implementa aprobación, sustitución ni recuperación CS10/11.

## Pruebas focales

```sh
GOCACHE=/dev/shm/go-build go test -p 8 -race \
  ./internal/modules/administracion/domain/operacionescopias \
  ./cmd/vec-copias-operacion
GOCACHE=/dev/shm/go-build go vet -p 8 \
  ./internal/modules/administracion/domain/operacionescopias \
  ./cmd/vec-copias-operacion
```

Las pruebas comprueban replay y conflicto, CAS, vínculo de ambas evidencias, fallo
de ensayo, recuperación estricta en cada fase, referencias no reflejadas, límites
de JSON y uso de ambos catálogos mediante el traductor común.
