# Ensayo HTTP del gobierno de baremos

El runner usa el binder final de PR443, el middleware común de identidad,
el PDP, la sesión PostgreSQL, el material criptográfico y el repositorio real.
Levanta un servidor TLS en loopback y presenta los certificados sintéticos
RRHH e Intervención del material privado existente. CT168 conserva la auditoría
de rechazos en su destino segregado.

Reutiliza los helpers del harness `29905e989478a0480a43127bbe4058f7a0e33a17`
mediante un overlay temporal. No copia esos helpers al producto, instala SQL,
arranca PostgreSQL ni publica endpoints. Las fases deben coordinarse con el
responsable del único clon desechable; no usar otra base.

La configuración privada contiene `harness`, con el JSON completo del ensayo
directo, y dos consultas revisadas de solo lectura. `auditoria_sql` cuenta los
intentos nominales comunes de la acción de alta, resultado `denegado`,
y actor exacto `$1`. Ese actor procede de la composición real del binder. `auditoria_pre_contexto_sql` cuenta los rechazos CT168 de la
misma ruta, motivo y superficie previa al contexto, con actor vacío. El responsable
SQL debe limitar ambas consultas al perímetro sintético del ensayo.
`harness.rutas` debe contener las tres rutas nominales de PR443.
La composición requiere `auditoria-intentos.json` privado con el esquema
`vec.auditoria.intentos.servidor.v1`, `dsn_file`, `proceso`,
`canal: interna_corporativa` y `limite_segundos`. El LOGIN del archivo DSN sólo
pertenece al registrador común y tiene configuración DBA de proceso/canal.
La configuración de Baremo incluye las referencias gobernadas
`motivo_intento_denegado` y `motivo_intento_error`; son causas del resultado
observado y no motivos elegidos por el formulario. Sin el preflight de la
auditoría común no se montan las rutas operativas. CT173 no forma parte de esta
candidata: se conserva su reserva y el borrador histórico sin instalarlo.

La continuidad HTTP usa un fichero distinto de la continuidad del ensayo
directo, siempre fuera de Git y con permisos `0600`.

El material privado debe incluir `bolsa/gobierno-reglas-baremo-v3.json`, sus
tres DSN nominales y el perfil ya provisionado. El binder del recorrido no
provisiona perfiles. La preparación inicial reutiliza las fábricas del harness;
no sirve para restaurar una revocación ni sustituye una aprobación con huella CAS.
Si existe historia anterior, recuperar la clave y el conjunto originales.

```bash
export VEC_ENSAYO_GO=/ruta/al/go/instalado
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar-http.sh compilar
export VEC_BAREMO_HTTP_PG_CONFIG=/ruta/privada/configuracion-http.json
export VEC_BAREMO_PG_DESECHABLE=si
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar-http.sh preparar
bash scripts/recorridos/seleccion/baremo-gobierno/ensayar-http.sh alta
```

Dirección autorizó para el clon nuevo posterior a AD155 una intención sintética
nueva, con otra clave de operación y el catálogo fijo. Las dos órdenes anteriores
se refieren a esa intención vacía. No restauran ni repiten el alta del clon
antiguo. Su acta debe identificar la convergencia AD161 y BR4 instalada una vez,
y mantener separada la evidencia histórica del ensayo directo del 2 de octubre.

`recuperar` sin continuidad exige que exista exactamente una versión, recibo,
historia y outbox de la intención. Recupera el recibo por replay HTTP `200`,
conserva su fecha y compara la huella durable antes y después. No debe lanzarse
`alta` sobre una historia conservada. `alta` queda limitada a una intención
vacía expresamente autorizada en el clon; comprueba `201` y guarda la continuidad
antes de los siguientes pasos.

El ensayo comprueba replay, consulta exacta y recuperación `200`, el mismo canon,
recibo y fecha, un `403` con una nueva auditoría durable y un `503` al cerrar
exclusivamente el pool local del auditor. Ese cierre no detiene PostgreSQL ni
otra aplicación. La huella de negocio debe permanecer idéntica.

Tras el reinicio del clon por su responsable, ejecutar `reinicio` con la misma
configuración. Otro proceso recompone las autoridades y exige un arranque
PostgreSQL posterior al conservado. Conserva las filas originales y repite
los HTTP. El recibo describe un borrador `disponible_para_preparacion`; no
acredita aprobación formal, activación del baremo ni firma legal.

La preparación del runner y su compilación no acreditan este recorrido. Sólo
una ejecución con el clon convergente permite declarar los HTTP y el reinicio
observados; no incluye navegador.
