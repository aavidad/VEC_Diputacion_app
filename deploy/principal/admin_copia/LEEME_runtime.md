# Preparación privada de runtime ADMIN

Estos scripts generan configuración y SQL para una copia local ya confirmada por
`arranque.py`. No ejecutan SQL, arrancan servicios ni publican actos. El conductor
aplica los archivos en el orden autorizado y conserva los recibos en el directorio
privado de la copia. El kit compilado debe incluir los seis scripts y
`plantilla-runtime.json`, con sus huellas en `manifiesto-kit.json`.

Primero se deriva la entrada desde el mismo kit, su `contrato-arranque.json` y los
recibos de fuentes, unidad y bootstrap. `--motivo-publicado` es un JSON privado
con `catalogo_id`, `catalogo_version`, `catalogo_huella_sha256` y `entrada_clave`.
Las fechas y redes son valores aprobados para esta copia, no valores del kit.

```sh
python3 KIT/preparar_runtime.py --run RUN --kit KIT --instancia INSTANCIA \
  --dia-gobierno DIA --semilla-raiz SEMILLA --ca-pg CA_PG \
  --pg-host HOST_PG --pg-base BASE_PG --puerto-pg PUERTO_PG \
  --sid-copia SID --vigente-hasta-config FECHA_UTC \
  --retirada-en FECHA_UTC --redes CIDR --motivo-publicado MOTIVO_JSON
```

Produce `RUN/vec-admin/identificadores/identificadores.json` y
`RUN/entrada-vec-admin.json`, ambos privados y de creación exclusiva. Requiere
dos personas y certificados ADMIN distintos ligados al plan y a los recibos.
Si la salida existe, se detiene sin reemplazarla.

El gobierno usa el conjunto 7, que contiene 12 audiencias. El primer paso
prepara LOGIN nominales y configuración; el segundo genera el SQL que liga el
plan exacto al SID. Entre ambos, el conductor instala los LOGIN y obtiene
`aprobacion-candidata.json` mediante el CLI de gobierno existente.

```sh
python3 KIT/preparar_gobierno.py --fase entrada --run RUN --dia DIA --sid-copia SID
python3 KIT/preparar_gobierno.py --fase configurar --run RUN --dia DIA --sid-copia SID
```

Los resultados son `RUN/dba/roles-gobierno-DIA.sql`,
`RUN/config-gobierno-DIA.json` y `RUN/dba/config-gobierno-DIA.sql`. El CLI de
gobierno debe dejar `RUN/gobierno/DIA/configuracion-material.json`,
`material.json` y `acuse-aplicar.json` antes de preparar claves y overlays.

Las fases de `preparar_vec_admin.py` consumen `RUN/entrada-vec-admin.json`.
Se ejecutan en orden `tls`, `activos`, `pools`, `claves`, `base`, `overlay`,
`runtime`, `entorno`. Las fases `claves`, `overlay` y `entorno` requieren `--dia DIA`.
`base` también requiere `--dia DIA`. `crl` solo renueva la lista cuando
corresponda. Cada fase escribe en
`RUN/vec-admin/`; `pools` genera `dba-logins.sql` para revisión y aplicación
posterior. `activos` coteja cada archivo con el manifiesto de fuentes del kit.

```sh
python3 KIT/preparar_vec_admin.py --entrada RUN/entrada-vec-admin.json --fase FASE --dia DIA
python3 KIT/preparar_overlay_externo.py --run RUN --dia DIA --motivos-publicados MOTIVOS_JSON
```

`preparar_overlay_externo.py` crea `RUN/config-gobierno-externo-DIA.json`;
el CLI Go autentica el plan, el acuse y el material al publicarlo.
Los LOGIN de los pools tienen una vigencia máxima de 90 días.

Los archivos bajo `RUN` contienen secretos e identificadores ficticios. Se
mantienen fuera de Git con permisos privados. El conductor coteja el SID real,
revisa el SQL, aplica los actos y verifica los recibos antes de arrancar
`vec-admin`. La preparación por sí sola no acredita ese arranque ni el recorrido.
