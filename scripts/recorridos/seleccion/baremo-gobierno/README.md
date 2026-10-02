# Ensayo directo del gobierno de baremos V3

El driver llama a `ServicioGobiernoV3` con el PDP central, el broker nominal,
la sesión PostgreSQL, el proveedor criptográfico y el repositorio PostgreSQL
existentes. Reutiliza la composición de seguridad y los constructores del
soporte base de CT, con sus autoridades nominales de gobierno y registro.
No levanta un servidor HTTP. El contexto de cada llamada se sella en el harness
con la identidad y el certificado verificados por la composición de desarrollo.
Este ensayo no acredita un recorrido HTTP, una conexión mTLS de navegador ni
firma legal.

Necesita un clon sintético desechable preparado por Dirección, las dependencias
de identidad y autorización que consume el soporte existente y AD144/BR4
instaladas y revisadas. AD144
se apoya en el núcleo real posterior a AD142; Copias/AD143 lleva otro circuito.
El script no arranca PostgreSQL ni instala o revierte migraciones.

El ensayo no arranca el caso de alta de CT ni su lector de cobertura O4-05:
el gobierno de baremos no los consume. Las guardas y la composición de CT en
producción permanecen en sus constructores originales.

La cuenta y el alias se preparan desde el soporte base. La sesión del broker
usa el contexto registrado del perfil propio de baremo y los mismos puertos
nominales de registro y revalidación. Se conserva el patrón de las rutas de
Plantillas CT; el perfil de baremo no sustituye el perfil base al crear la cuenta.

## Configuración privada

Crear el JSON fuera de cualquier árbol Git, con permisos `0600`, en un directorio
sin enlaces simbólicos. Sus ficheros referenciados también deben ser privados.
No guardar credenciales, material de idempotencia, conjuntos ni continuidad en
esta carpeta del repositorio.

| Campo | Contenido |
| --- | --- |
| `version` | `1` |
| `entorno` | Variables `VEC_` de la composición existente, incluidos material de desarrollo, doble llave y DSN nominales de CT, identidad, registro, fuente y motivos. |
| `runtime_dsn` | LOGIN exclusivo del ejecutor de gobierno del baremo sobre el clon, con TLS verificado y una sola membresía nominal. |
| `evidencia_dsn` | LOGIN separado para la consulta de evidencia, sobre el mismo clon. El driver abre transacciones de solo lectura. |
| `resumen_sql` | SELECT parametrizado por `$1`, la clave de operación. Devuelve cinco columnas: números de versiones, recibos, historias y outbox ligados a esa intención, y SHA256 estable de todas esas filas. Excluye la auditoría de accesos nuevos. |
| `conjunto` | Ruta al conjunto sintético canónico válido, aceptado por `RestaurarConjuntoReglasBaremo`. |
| `conjunto_ajeno` | Ruta a otro conjunto sintético válido con convocatoria o expediente diferente. |
| `motivo` | `ReferenciaEntradaCatalogo` existente: `catalogo_id`, `catalogo_version`, `catalogo_huella_sha256`, `entrada_clave`. |
| `clave_operacion` | Clave estable de 32 caracteres hexadecimales minúsculos. Se conserva en todas las fases. |
| `continuidad` | Fichero privado nuevo que conservará el recibo original, el canon, el resumen durable y el instante de arranque de PostgreSQL. |
| `rutas` | Objeto con `Alta`, `Consulta` y `Recuperar`, tres coordenadas exactas distintas de la composición nominal del ensayo. No son endpoints publicados. |

Las conexiones PostgreSQL se restringen a direcciones IP de loopback. No se
acepta el puerto ordinario `5432`. Se limpian las variables de otras instancias
VEC antes de cargar `entorno`; no se imprimen sus valores. La composición
existente verifica sus propios materiales, permisos y cuentas técnicas.

Dirección y el responsable SQL deben revisar el SELECT de evidencia y las
entradas antes de ejecutar. Un resumen vacío, incompleto o que incluya los
accesos nuevos no sirve para demostrar conservación del negocio.

## Fases

Con un compilador local compatible ya instalado y las dependencias descargadas:

```bash
export VEC_BAREMO_PG_CONFIG=/ruta/privada/configuracion.json
export VEC_BAREMO_PG_DESECHABLE=si
export VEC_ENSAYO_GO=/ruta/al/go/instalado
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar.sh preparar
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar.sh alta
```

`preparar` publica el contexto, el perfil inicial, el motivo y el descriptor de
material mediante las autoridades existentes. Una asignación revocada o distinta
no se restaura: esta preparación no aporta aprobación de una preimagen. Las
factories del soporte CT conservan su preparación habitual de arranque. El
constructor común del material nominal se repite al recomponer para tomar las
versiones de raíz, clave y configuración realmente gobernadas en PostgreSQL.
Conserva el material privado persistente y no publica por operación del servicio.

`alta` exige que la intención esté vacía y que aún no exista la continuidad.
Comprueba un alta nueva y conserva el recibo antes del replay. Repite la misma
intención, consulta la versión exacta y recupera el recibo. Exige canon y recibo
originales, un acceso nuevo separado y una sola versión, historia, recibo y outbox.
También comprueba ámbito ajeno, operación en otra frontera, campos cruzados
entre consulta y recuperación, y una caída real del pool local del consumidor.
Los errores se comprueban como errores nominales del servicio; no son estados
HTTP observados.

Si se perdió la primera respuesta, `ensayar.sh recuperar` conserva la misma
clave y el mismo conjunto. Cuando aún no hay fichero de continuidad, exige
los cuatro efectos ya presentes, un replay confirmado y una huella de negocio
idéntica antes y después de recuperar. Si la continuidad ya existe, comprueba
su recibo sin exigir un reinicio. No realiza un alta nueva para resolver el fallo.

Después de que Dirección reinicie PostgreSQL en el clon autorizado:

```bash
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar.sh reinicio
```

La segunda ejecución recompone las dependencias desde el mismo material privado.
Exige un `pg_postmaster_start_time()` posterior al conservado y repite el replay,
la consulta, la recuperación y los negativos. Compara la huella de negocio
completa con la anterior. No borra el fichero de continuidad ni crea otra clave.

Un fallo después de guardar la continuidad conserva el recibo para investigar.
Si el fallo ocurrió antes de guardarla y el efecto ya existe, usar `recuperar`.
No volver a lanzar `alta` ni borrar historia para hacer pasar el ensayo.

## Evidencia pendiente

En este corte el driver está preparado para compilación y revisión. Sin
configuración privada la prueba se omite expresamente. Esa omisión no demuestra
instalación, ejecución PostgreSQL, persistencia ni recuperación tras reinicio.
Los negativos de campos del broker no sustituyen los negativos del consumidor
SQL ni sus revisiones independientes.
