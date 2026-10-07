# Preparar el cotejo de promoción interna

La CLI recibe bases, hechos y un dictamen aportados para un ensayo. Comprueba que
la respuesta corresponde a la consulta completa y conserva el resultado del productor.
Carrera no interpreta condiciones de las bases ni calcula requisitos o puntos.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-carrera-cotejar-promocion < cmd/vec-carrera-cotejar-promocion/testdata/entrada.json
go run ./cmd/vec-carrera-cotejar-promocion --idioma en < cmd/vec-carrera-cotejar-promocion/testdata/entrada.json
```

El archivo contiene una `consulta` y un `dictamen_sintetico`. Son configurables:
referencias y versiones de persona, proceso, bases, hechos y requisitos; hitos,
procedencia, vigencia, estado probatorio y evidencia; resultados del productor.
El ejemplo aporta `cumple`, `no_cumple` y `pendiente`. Sus datos, fuentes y
comprobaciones son sintéticos y retirables, sin aprobación competente.

La `huella_bases_aportada` es un valor de ensayo proporcionado en el archivo.
La CLI no calcula ni verifica esa huella, la firma o la aprobación de las bases.
La salida mantiene `bases_verificadas: false` y `estado_global: pendiente`.

Para preparar la respuesta del productor sobre una consulta modificada:

```sh
go run ./cmd/vec-carrera-cotejar-promocion --huella-consulta < cmd/vec-carrera-cotejar-promocion/testdata/entrada.json
```

Este modo devuelve solo la huella SHA256 del material local de consulta. Copie
ese valor en `dictamen_sintetico.huella_consulta_sha256`. No es firma, recibo,
verificación de bases ni evaluación. El lector nominal futuro deberá aportar
su propio dictamen y autoridad; el adaptador actual devuelve la respuesta del archivo.

La salida conserva `dictamen_aportado` y los `resultados` mostrados. Una
comprobación concluyente sobre texto libre, regla sin referencia o hechos sin
acreditación/evidencia identificada queda pendiente. Conserva el estado recibido
y explica la guarda aplicada; una declaración tampoco acredita un incumplimiento.
Las referencias de reglas y los hechos etiquetados como acreditados son datos
aportados del ensayo, sin validación de su autoridad real.

Si falta el productor o la respuesta cruza versiones, huellas, hechos, requisitos
o hitos, la CLI devuelve `carrera.cotejo.error.no_disponible` por stderr y código 1,
sin lista parcial. Rechaza claves duplicadas, campos desconocidos, documentos
concatenados y entradas mayores de 1 MiB. Admite hasta 64 requisitos y 128 hechos.

Los textos se cargan desde `web/static/textos` con el traductor común; puede indicar
otra carpeta mediante `--catalogos`. El resultado incluye los límites y motivos
traducidos en castellano o inglés. No se publica una pantalla ni se abre una red.

Siguen pendientes el contrato confirmado de Selección, la autorización de Carrera
y los lectores nominales de Personal/Méritos. La presentación, la admisión,
las pruebas y la decisión selectiva corresponden a Selección. Esta preparación
no registra solicitudes, excluye personas ni reconoce derechos.
