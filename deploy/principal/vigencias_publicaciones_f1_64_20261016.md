# Vigencia de las nuevas publicaciones de confianza

Tarea F1-64, acotada por dirección el 16/10: cambia la duración de las
publicaciones nuevas de CT y del portal externo. Conserva las raíces, las
claves y las publicaciones anteriores. La rotación de la raíz y la clave
legadas de 2026–2036 queda fuera de este corte y la coordina dirección.
No añade SQL ni permisos.

## Activación en una copia local

Partir de la copia propia con el prefijo completo que ya contiene AD263 y
AD282. Comprobar esas funciones y su historia; no reaplicar migraciones ni
instalar una antigua sobre historia. Conservar material, conexiones y recibos
fuera de Git. El proceso web no recibe la conexión del publicador externo.

Sin `VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS`, las publicaciones nuevas duran
24 horas. Para hasta 90 días, el proceso que publica necesita las tres llaves
existentes y la misma variable que ya usa Administración:

```sh
export VEC_EXECUTION_PROFILE=desarrollo
export VEC_AUTH_MODE=desarrollo
export VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
export VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS=2160
```

Con el material y la conexión privados de esa copia, ejecutar el preparador
existente descrito en `confianza_externa_hz38.md`:

```sh
vec-server preparar-portal-externo   --material-externo "$MATERIAL_EXTERNO"   --publicador-confianza-dsn "$DSN_PUBLICADOR_PRIVADO"   --consumidores mi_bolsa,portal_candidato,preferencias_bolsas,inscripcion_bolsa,meritos_cartera
```

La publicación vigente se recupera con sus fechas originales; cambiar el valor
no la sustituye. El valor se aplica al publicar o renovar. La renovación
administrativa existente debe recibir ese mismo entorno. No alterar el reloj,
la publicación ni su fecha para forzar la renovación. En CT se conserva el
origen de la publicación: una publicación de Administración sigue necesitando
su acto y no pasa a renovación automática por tener el mismo plazo.

Producción admite de 1 a 24 horas; desarrollo con las tres llaves, de 1 a 2160.
Se rechazan valores desconocidos, cero, negativos o superiores al máximo.
Las publicaciones no pueden rebasar la raíz existente. Un plazo inferior a
24 horas empieza en el instante de publicación; los demás conservan el corte
de medianoche UTC. Renovar el mismo día crea otra secuencia y referencia.

## Comprobación y retirada

Arrancar con la configuración privada existente sin la variable y después
con 2160 horas. Comprobar salud, ficha y descarga con certificado, y registrar
las fechas de la publicación vigente. Reiniciar sin la variable: debe conservar
la misma publicación, no acortarla. El caso de publicación nueva y los rechazos
de producción se prueban además con las pruebas focales de bootstrap. No se
afirma que producción esté habilitada para el recorrido de desarrollo.

Para retirar el ajuste, quitar la variable y reiniciar con vigilancia. Las
publicaciones vigentes conservan sus fechas; las siguientes vuelven a 24 horas.
La ventana breve para aplicar planes no cambia. La vigencia de los certificados
personales de firma y la rotación de sus raíces se gestionan por sus guiones.
