# Autenticacion fake local segura

## Alcance

`fake` es una herramienta de desarrollo local y **no es un mecanismo de
produccion**. Parte cerrado: no contiene usuarios, roles, mecanismos, garantias
ni secretos predeterminados. Si falta cualquier dato, la peticion obtiene
`401`; si el fichero es inseguro, el proceso no arranca.

## Frontera obligatoria

Para habilitarlo deben cumplirse simultaneamente estas condiciones:

- `VEC_AUTH_MODE=fake`;
- `VEC_FAKE_CREDENTIALS_FILE` apunta a un fichero regular existente;
- el fichero no es un enlace, mide como maximo 1 MiB y no concede ningun
  permiso a grupo ni a otros (`0600` es la opcion recomendada);
- `VEC_HTTP_ADDR` declara una IP loopback literal, no `:8080`, comodines,
  nombres DNS ni interfaces corporativas;
- `VEC_HTTP_ALLOWED_CIDRS` contiene exclusivamente loopback;
- cada peticion autenticada presenta exactamente un
  `Authorization: Bearer <token-opaco>`.

Aunque el par remoto sea local, el servidor rechaza `Forwarded`, `Via` y toda
cabecera `X-Forwarded-*`. Las familias `X-Auth-*` y `X-VEC-*` no aportan
identidad ni autoridad en este modo. `X-Auth-Token` tampoco sustituye a
`Authorization`.

## Token y fichero

El operador genera fuera del repositorio un token criptograficamente aleatorio
de al menos 256 bits, codificado en base64url sin relleno o hexadecimal. El
valor presentado debe medir entre 43 y 128 caracteres y usar exclusivamente
letras ASCII, numeros, guion o guion bajo. Nunca se incluye en JavaScript,
documentacion, variables de imagen, historial compartido ni control de
versiones.

El JSON guarda solamente su SHA-256 hexadecimal minusculo:

```json
{
  "version": 1,
  "credentials": [
    {
      "token_sha256": "<SHA-256 HEXADECIMAL DE 64 CARACTERES>",
      "subject": "<IDENTIFICADOR INTERNO EXPLICITO>",
      "display_name": "<NOMBRE EXPLICITO>",
      "email": "<CORREO OPCIONAL>",
      "roles": ["<ROL VEC EXACTO>"],
      "mechanism": "<clave|dnie|kerberos_ad>",
      "assurance": "<bajo|sustancial|alto>",
      "legacy_role": "<candidate|validator_l1|validator_l2|system_admin>"
    }
  ]
}
```

No se admiten campos desconocidos, valores no canonicos, huellas duplicadas,
mecanismos o garantias inferidos ni principales que fallen su validacion de
dominio. Cada credencial declara **un unico rol VEC**. La carga rechaza el
fichero completo ante un rol adicional, repetido, desconocido o expresado
mediante un alias.

`legacy_role` es obligatorio porque la API Bolsa heredada aun usa un perfil
grueso. Ambos valores se aportan de forma expresa, pero deben coincidir con
esta lista positiva exacta:

| Rol VEC canonico | `legacy_role` unico compatible |
| --- | --- |
| `ciudadano` | `candidate` |
| `administrativo` | `validator_l1` |
| `tecnico_rrhh` | `validator_l2` |
| `jefatura_rrhh` | `validator_l2` |
| `administrador` | `system_admin` |

No hay equivalencias inferidas. En particular, `personal_interno`,
`jefe_servicio` y `jefe_seccion` no se traducen a un validador de Bolsa: ser
empleado o ejercer una jefatura ordinaria no concede tramitacion de procesos
selectivos. Los alias ingleses usados historicamente por la carcasa tampoco
son roles VEC validos en este fichero. Incorporar un perfil nuevo exige revisar
y publicar de forma deliberada esta frontera; hasta entonces el arranque falla.

## Capacidades de la carcasa de demostracion

El rol declarado no se transforma en un paquete funcional por semejanza. El
traductor temporal aplica esta lista positiva exacta:

| Perfil fake | Capacidades de carcasa | Capacidades funcionales no concedidas |
| --- | --- | --- |
| `ciudadano` | Sesion, menu y entradas de Bolsa para solicitud, documentos, alegaciones y notificaciones | Gestion, accion demo y auditoria de Bolsa; todo Personal, Cronos, Dietas y Administracion |
| `administrador` | Sesion, catalogo de modulos, menu y administracion tecnica de roles, catalogos, integraciones y monitorizacion | Todo Personal, Cronos, Dietas y Bolsa; workspace y auditoria generica |
| `jefatura_rrhh` | Sesion, catalogo de modulos y menu | Roles, catalogos, integraciones, seguridad, auditoria y toda operacion funcional sin ambito resuelto |
| `tecnico_rrhh` | Sesion, catalogo de modulos y menu | Cronos, nominas, Dietas, Administracion y cualquier lista o expediente sin asignacion resuelta |
| `administrativo` | Sesion, catalogo de modulos y menu | Cronos, nominas, Dietas, Administracion y cualquier lista o expediente sin asignacion resuelta |

`personal_interno`, `jefe_servicio` y `jefe_seccion` solo existen en fixtures
unitarios heredados de la carcasa y reciben tambien exclusivamente sesion,
catalogo de modulos y menu. No aparecen en el fichero fake valido y no obtienen
listas globales por su nombre. Los alias `candidate`, `validator_l1`,
`validator_l2` y `system_admin` conservados por pruebas se comportan como su
perfil unico correspondiente, pero no se pueden mezclar con el nombre canonico.

Las entradas ciudadanas de Bolsa permiten representar el menu de la demo; no
son por si solas una autorizacion de negocio. Cada futura ruta debe resolver en
servidor la persona titular o representada y el recurso exacto. Mientras ese
resolvedor no exista, no se monta la operacion. De igual modo, el permiso de
auditoria generica no se entrega al administrador tecnico porque sus resultados
pueden contener referencias funcionales: se necesitara una vista tecnica
minimizada y una capacidad separada.

Procedimiento operativo recomendado:

1. aplicar `umask 077` y generar el token con un CSPRNG del sistema;
2. calcular `SHA-256` sin saltos de linea y escribir solo la huella en el JSON;
3. aplicar `chmod 600` al JSON y guardar el token en un gestor local de
   secretos o fichero temporal `0600` separado;
4. arrancar con una direccion como `127.0.0.1:8080`;
5. presentar el token en tiempo de ejecucion, por ejemplo mediante una variable
   de shell privada: `-H "Authorization: Bearer $VEC_FAKE_TOKEN"`;
6. destruir la copia temporal al terminar.

El almacen se carga una vez al arrancar. Una alta, baja o rotacion exige crear
un fichero completo valido y reiniciar. Una huella repetida invalida todo el
arranque; no se elige silenciosamente una entrada.

## Respuesta ante fallos

La ausencia de Bearer, un esquema mal formado, multiples cabeceras
`Authorization`, un token desconocido o una credencial fuera del formato
devuelven `401` sin explicar cual fue la diferencia. El propio resolvedor exige
una `RemoteAddr` con IP loopback literal y puerto numerico, y rechaza
`Forwarded`, `Via` y cualquier `X-Forwarded-*`; por ello un manejador API usado
por error sin el envoltorio seguro tampoco autentica una peticion externa. El
envoltorio HTTP mantiene ademas su rechazo temprano (`400`) de cabeceras de
proxy. Un fichero ausente, permisivo, enlazado, sobredimensionado o
semanticamente incorrecto detiene el arranque.

## Limites y salida a produccion

No se ha creado un binario demo separado: la composicion comun permanece
cerrada y solo activa fake con toda la configuracion anterior. Separar los
artefactos demo/productivos sigue siendo una mejora de defensa en profundidad,
pero no se presenta como terminada.

Produccion debe sustituir este adaptador por los conectores de identidad
aprobados, sesiones de vida corta, autenticacion reforzada de la superficie
interna y autorizacion RBAC+ABAC por operacion. Ningun fichero fake se monta en
una imagen o despliegue productivo.
