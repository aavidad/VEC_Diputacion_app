# Validar un paquete local de colores

`vec-temas-validar` comprueba un paquete JSON contra una política de plataforma
revisada. Devuelve sus colores normalizados, las huellas y los contrastes calculados.
El estado `validado_sin_instalar` corresponde a esta preparación técnica local.
La herramienta no publica catálogos ni modifica las preferencias de nadie.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-temas-validar \
  -politica data/temas/politica-v1.json \
  -politica-sha256 548c28a7452217e88c3b0590682f627b2ab5ea5f6b290a8c4a6234495e15af44 \
  < data/temas/institucional-v1.json
```

La huella esperada procede de la configuración revisada. No debe obtenerse de
un paquete recibido ni calcularse automáticamente sobre una política desconocida
para aceptarla. La CLI lee la política indicada y el paquete de la entrada
estándar. No escribe archivos ni consulta la red. Un rechazo devuelve un código
distinto de cero y un JSON `error_clave` por la salida de diagnóstico. Las frases
para esas claves viven en los catálogos `temas-paquetes.json` de cada idioma.

## Contrato de datos

El paquete declara esquema, identificador libre de tema, versión, referencias de
sistema visual y política, clave de nombre, nombres por idioma y dos variantes
completas: `clara` y `oscura`. La clave del nombre es exactamente
`ui.temas.<tema_id>.nombre`. Los nombres son texto sencillo, sin HTML, controles,
interpolaciones ni direcciones web. Cada variante contiene exactamente los tokens
de la política, con colores opacos de seis dígitos hexadecimales.

La política fija los tokens, idiomas requeridos, valores protegidos, pares de
contraste y límites. La política v1 toma los idiomas del índice común de datos
`web/static/textos/idiomas.json`, protege el fondo blanco del logotipo y exige 29
pares por variante. El mínimo técnico es 4,5 para texto y 3 para componentes.
Las versiones son enteros positivos dentro del rango de 32 bits. Una política
puede exigir un contraste mayor y reducir los límites técnicos:
64 KiB por documento, 64 tokens, 32 idiomas y 100 caracteres por nombre.
El paquete no define su propia política.

Se rechazan claves duplicadas, claves con mayúsculas, campos desconocidos,
documentos UTF-8 inválidos, contenido posterior al JSON y profundidad superior
a ocho niveles. No se admite CSS, JavaScript, recursos externos ni referencias
`var()` en los colores. Cada variante resuelve todos sus valores sin herencia.

## Ejemplos y huellas

`data/temas/` contiene la política y ocho ejemplos exportados de los tokens
del tema común en la base `296f78373f4874b9a81def9df744df12c1096af1`:
institucional, granate, Diputación de Granada, arena, salvia, lavanda, azul sereno y noche
suave. Los alias están resueltos. La variante `clara` conserva la paleta existente;
noche suave ya usa colores oscuros en esa paleta. Granate conserva también su
variante oscura existente. Los otros siete ejemplos reproducen el modo oscuro
común. No se ha diseñado otra paleta oscura por tema.

`original_sha256` identifica los bytes recibidos. Cambiar espacios u orden de
propiedades cambia esa huella. `canonico_sha256` identifica el material normalizado:
colores en minúsculas, propiedades del contrato en orden fijo y claves de mapas
ordenadas. `politica_sha256` identifica los bytes revisados de la política.
El informe enumera la razón y el mínimo para cada par en cada variante.

## Alcance pendiente

Esta comprobación de colores no certifica la accesibilidad de una pantalla.
Quedan pendientes instalación durable, gobierno común de catálogos, autorización,
auditoría, selección por Usuarios y recorrido en navegador. Los estilos, módulos,
preferencias y migraciones existentes permanecen sin cambios.

Un consumidor posterior puede generar una hoja del mismo origen sólo desde el
material validado. Debe mantener el alto contraste común y funcionar con la CSP
de ADMIN `style-src 'self'`. La combinación con `html[data-tema="granate"]`
requiere retirar y restaurar ese estado legado durante la aplicación del paquete;
sus selectores tienen mayor especificidad. No se ha añadido ese consumidor aquí.

El código separa las responsabilidades necesarias: el dominio valida contrato y
contraste; el adaptador comprueba huella, tamaño y estructura JSON; la CLI
interpreta argumentos locales y escribe el resultado. No hace falta un caso de
uso con efectos, un puerto de permisos ni una capa de aplicación para este corte.

## Comprobación del corte

Se ejecutaron pruebas focales de dominio, adaptador y CLI, con detección de
carreras, y `go vet -p 8` sobre esos tres paquetes. Los ocho ejemplos superaron
los 29 pares por variante. Las pruebas incluyen claves repetidas, campos
desconocidos, mayúsculas, límites, rechazo de colores libres, nombres inválidos,
contraste insuficiente, estabilidad canónica y fallos de escritura.

Gosec y Semgrep local no encontraron problemas en los cuatro archivos de
producción. Semgrep usó 42 reglas Go descargadas como datos de `p/golang`, con
`--metrics=off` y `--disable-version-check`; los archivos de pruebas quedaron
excluidos por la configuración existente. No se envió código a un servicio de
análisis. `vecsilencio` no encontró fallos silenciosos nuevos respecto a la base.
Los procesos Go se ejecutaron sin red, con fuente y herramientas de sólo
lectura, entorno explícito y archivos temporales en un directorio aislado.
