# Denominación declarada de Persona

Esta biblioteca prepara el nombre usado para mostrar una Persona existente.
Conserva `persona_ref` y una versión propia; no modifica la identidad ni crea
otra Persona. Dos personas con el mismo nombre siguen siendo dos personas.
El nombre declarado no acredita identidad civil, empleo ni permisos.

`NuevoProtector` recibe claves de una fuente del KMS existente y una norma
versionada de configuración. Los ámbitos de cifrado y búsqueda son dedicados
y distintos de correo, contacto y autenticación. No hay claves ni valores
funcionales implícitos. La norma fija NFC, tratamiento de mayúsculas,
separadores, tamaño y cantidad de palabras; su referencia y huella forman
parte del sobre. La comparación no depende de un idioma.

`NuevoPreparador` comprueba Persona por su puerto antes y después de preparar
el sobre. `PrepararDenominacionPersona` cifra con AES-GCM y calcula tokens
HMAC-SHA256 de palabras completas, ligados al ámbito, la norma y la clave.
El AAD liga Persona, versión, esquema, clave e índice. El resultado contiene
procedencia opaca, versión esperada y SHA256 del sobre cifrado.

`NuevoLector` consume una fuente autorizada. La fuente debe confirmar consumo
V3 y auditoría antes de devolver el sobre y su acuse. El acuse conserva las
referencias de consumo, auditoría, decisión, correlación y recurso, Persona,
versión y SHA256 del sobre. El constructor del acuse comprueba su estructura;
la fuente debe verificarlo contra el material V3 original y el consumo común
confirmado. Esta biblioteca no inventa una firma ni acredita por sí sola COMMIT.
El lector verifica el destinatario
y la versión, descifra, comprueba nuevamente las claves y revalida Persona y
acceso antes del callback; vuelve a comprobar el sobre con la clave vigente
después de esa revalidación. Un fallo de COMMIT, KMS o revalidación impide
prestar el nombre. La búsqueda devuelve referencias mediante el índice;
cada nombre necesita su propia lectura autorizada.

El lector exige el registrador común de intentos, el contexto registrado V2,
su vínculo de autenticación original y el motivo gobernado de error.
Si la lectura falla, registra el error observado mediante esa autoridad,
conservando la misma orden si el COMMIT del intento resulta ambiguo.
Un error posterior al consumo conserva el acceso permitido anterior;
no afirma que se haya deshecho ni añade un registro propio de auditoría.

El puerto `RegistroDenominacionPersona` exige CAS, historia por adición,
puntero actual, auditoría y outbox en la transacción del consumo V3.
Su futuro adaptador debe volver a comprobar Persona y ligar la autorización
al sobre, procedencia, versión, ámbito, finalidad y campos exactos. Los permisos
de correo no sirven para esta faceta. No se registra nombre, búsqueda ni
SHA256 de un nombre claro en auditoría, errores o trazas.

Una rotación de búsqueda requiere reindexar y anunciar la clave del índice;
una referencia esperada distinta cierra altas y búsquedas. La lectura histórica
usa una clave de cifrado retenida y vigente; una revocación la cierra.
Si cambia la norma, la composición debe conservar su versión para las lecturas
históricas. El puntero actual nunca borra los sobres anteriores.

Pruebas sintéticas: homónimos sin fusión, versiones separadas, palabras
completas, normalización Unicode, ámbito, sustitución de Persona/versión/AAD,
rotación, revocación, separación de claves y orden de lectura autorizado.
Las fuentes de prueba son dobles; no acreditan SQL, CAS durable, V3 instalado
ni recuperación tras reinicio. No hay composición productiva ni CLI en este
corte. La siguiente dependencia es el adaptador autorizado y transaccional
de la faceta común de Persona, previsto para CA32.

Comprobaciones de este corte, 3 de octubre de 2026:

- `go test -race -p 8 ./internal/vec/adapters/denominacionpersona`: correcto.
- `go vet -p 8 ./internal/vec/adapters/denominacionpersona ./internal/vec/domain ./internal/vec/ports`: correcto.
- Semgrep, 42 reglas Go locales: ningún hallazgo en los nueve archivos nuevos
  de implementación; dos archivos de pruebas excluidos por la configuración.
- Gosec: ningún hallazgo en el código nuevo. Al revisar también los paquetes
  comunes, aparecen 37 avisos en 16 archivos heredados, cuyos blobs coinciden
  con la base de esta rama. Este corte no modifica esos archivos.

Las pruebas y Gosec se ejecutaron sin red, con fuentes, módulos y herramientas
de sólo lectura, entorno vacío y un directorio temporal aislado. Se limitaron
CPU, procesos, memoria, tamaño de archivo y tiempo. La revisión focal usó el
índice de código y `gopls` para localizar la autoridad común de intentos.
No se ejecutaron SQL, servicios, navegador ni pruebas globales.
