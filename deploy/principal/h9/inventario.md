# Inventario H9 sobre main fijado

El inventario fija el árbol `39ec9858d2983e24786db9d909e86cdce19db720` y compara las altas SQL con `ed9c7de86`. Contiene **48 UP nuevas y ocho pendientes anteriores**. Las PR posteriores y abiertas quedan fuera; en particular, AD172 y AD173 no pertenecen a este árbol. CT172 sí pertenece: el número coincide entre módulos, pero son migraciones diferentes.

La selección completa ya está ensayada: 39 migraciones y un soporte de roles. AUT30 entró en main durante el trabajo (#453) y se añadió antes de CT170. Arranque, reinicio y puerta completa acreditados. El paquete se entrega para revisión de Claude; no se ha instalado en cidonia. El guion exige preimagen y mantenimiento privados antes de intervenir servicios.


## Preimagen

La copia fría `h7-pg.tgz` tiene SHA256 `a1f56a65d53c6aaa20ab9f9b753f08ce80d390178ae03d6926072722ae192ce4`. Para reconstruir el clon se añaden las nueve SQL de H7, las tres de H8 y las versiones de main de AD155, Bolsa77 e importación Convoca5. Sus nombres y SHA completos están en [orden.json](orden.json), `preimagen.aplicar_solo_en_clon`.

La principal ya conserva esa historia: no reaplicar estas SQL ni ejecutar DOWN. El estado de instalación procede del relevo del 03/10 y de la reconstrucción comunicada por Dirección; este inventario no consulta la principal. Convocatorias, Méritos y Reglas de baremo no tienen esquema en la copia fría. Si se habilitan, necesitarán sus roles y prefijos estructurales, además de las migraciones de este inventario.

## Orden de la selección H9

Cada paso presupone que las guardas completas del archivo coinciden; el número de migración no sustituye ese cotejo. La lista reproducible está en `seleccion_h9` de [orden.json](orden.json), con rutas y huellas en `migraciones`. `orden_aplicacion` incorpora además el soporte de roles y es la lista completa de 40 archivos para el ensayo.

1. Base ADMIN: CA19 → AUT22 → AUT23 → CA20 → IS9 → AUT24.
2. Auditoría común: roles_intentos_up.sql → CA26 → IS13 → AD169 → CA27 → AD171.
3. Fuentes nominales: AD160 → AD164 → AUT33 → CA25.
4. Circuito y originales: AD151 → CT163 → CT164 → AD156 → AD157 → AD158 → Documentos13.
5. Firma V2: AD162 → AD170 → AD159 → AD165 → AD166.
6. Cargo y competencia: Personal28 → Personal29 → AD167 → AUT32 → AUT35.
7. Capacidades y consumidores CT: AUT34 → AD168 → AUT30 → CT170 → CT174 → CT172 → Personal31.

**AD162 va antes de AD159.** AD159 exige las fachadas y la huella POST-AD162, aunque su número sea menor. CT172 exige AD162/AD170, CT174 y AUT32/AUT35. AUT32 exige CA25, Personal28 y AD167; AUT35 añade la fuente de Personal29. La futura AD172 debe quedar detrás de AD154 cuando se incorpore a main, con su revisión y ensayo propios.

## Cadenas excluidas y conflicto de preimágenes

Las siguientes exclusiones dependen de que el montaje conserve sus consumidores apagados o ausentes. La evidencia de configuración debe quedar registrada antes de usar el kit; no basta que un módulo no figure en el menú. Si alguno está activo o su comprobación es obligatoria al arrancar, se detiene la selección y se incorpora su cadena compatible.

| Cadena | Orden propio | Condición para excluir |
| --- | --- | --- |
| CRN11 | AD149 → Personal26 | Consumidor histórico CRN11 sin montar. |
| RPT | AD149 → Personal26 → AD154 → Personal27 → Personal30 | Consulta de relación RPT sin montar. |
| Selectivos/Méritos/Baremo | AD149 → Personal26 → roles BC/Méritos/Baremo → AD161 → Méritos1 → rol lector Méritos → AD145 → Méritos2 → authBC1/BC1 → AD153 → BC8; CT168 para la frontera | Preparación de bases, Méritos y gobierno de baremo sin montar. |
| Históricas AD141/142/144 | No ejecutar en este kit | AD161 crea sus contratos sobre POST149+155 sin registrar las históricas como reaplicadas. |

AD149 y AD151 parten de POST155. AD149 espera definición `5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330` y fuente `cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e`. Tras AD149, el núcleo pasa a definición `d912064d905e4349ffb2e1e8f1aab1aebef71e1fd842d5603373c4e6236fcd15` y fuente `1bb33d97bf8bbaa7f97dec4e1af1aa41577be38cea17f4b18375346cf7d62dcd`. AD151 exige la primera preimagen y no reconstruye CRN11. Por ello, **AD149 → AD151 no forma una cadena válida**. AD154 y AD161 compiten a su vez sobre POST149; tampoco se concatenan sin reanclaje.

Los archivos históricos AD141/142/144 fijan huellas anteriores a AD155. AD161 sustituye la instalación de esos contratos; la mera presencia de los archivos en main no permite ejecutarlos. Las restantes SQL opcionales permanecen inventariadas, con su huella y guardas, para preparar su incorporación cuando se monte el consumidor.

## Soporte fuera de las 48 altas

Se han consultado todas las listas causales `deploy/principal/lista_sql_*.txt` del árbol fijado. El soporte se inventaría por separado: no altera el recuento de UP nuevas.

| Archivo | SHA256 | Uso |
| --- | --- | --- |
| `deploy/postgresql/autorizacion_atestada_v3/roles_intentos_up.sql` | `70397114c01b77323633ed5c3bb87c2b9f0a23188042739d97b0ba850929b092` | H9: antes de CA26/IS13/AD169. |
| `deploy/postgresql/bolsa_convocatorias/roles_up.sql` | `3f49a405fd798bea91a66c00bb3135b4f250c02c19f441771d1abc1c66b5533f` | Opcional: AD161. |
| `deploy/postgresql/meritos/roles_up.sql` | `c2a3c87ca48e53429962f8d089c7aa2488d5f46c5146ae2e87f9cb84f186b3d8` | Opcional: AD161. |
| `deploy/postgresql/bolsa_reglas_baremo/roles_up.sql` | `5c2d4507dbb5f183e4a2ad53ec897b01467b38313df98bc3e70700eb9a0805d8` | Opcional: AD161. |
| `deploy/postgresql/meritos/roles_consulta_propia_up.sql` | `fdb087494afedde1244c7f40f866c3b5ae0b5433fef9d17b07a266ce0b0987cf` | Opcional: AD145. |
| `deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql` | `fc63053318b5f2d4b5b53d5265ec98f5b6bcaec5f086b1c04fbf4732685a1209` | Opcional: AD153, BC8. |
| `deploy/postgresql/bolsa_convocatorias/migraciones/000001_almacen_convocatorias.up.sql` | `818916065270d42bf263fbe679196fdac3ecb6cb4a43dc9b4856b1993451ce1e` | Opcional: AD153, BC8. |

`lista_sql_codexl_intentos.txt` exige `roles_intentos_up.sql`: el rol registrador común no se crea en AD169. Un fallo por ese rol ausente es una dependencia de instalación, no permiso para omitir su guarda. Méritos requiere además `roles_consulta_propia_up.sql` antes de AD145 y después de Méritos1. Para Preparación S2, `lista_sql_codexa_selectivos_s2.txt` exige authBC1 y BC1, y excluye expresamente BC2..7 como dependencias. BC7 conserva su propio inventario para la consulta S1; no se instala por conveniencia al montar S2.

## Tabla del corte

`H9` significa seleccionada para el ensayo; `cond.` exige evidencia del montaje para excluirla; `AD161` identifica contratos históricos sustituidos. La columna SHA corresponde a los bytes del archivo UP. Las preimágenes incluyen guardas estructurales y de ACL, no sólo literales SHA: [orden.json](orden.json) enumera **todos los bloques DO** de cada archivo, sus líneas, SHA del bloque, categorías y literales de huellas. También enumera las fachadas SQL creadas con su línea de origen. Su fuente completa sigue siendo el archivo UP fijado.

| Módulo/número | PR final del archivo | Preimagen / cadena | SHA256 UP | Consumidor SQL | Arranque / selección |
| --- | --- | --- | --- | --- | --- |
| `contexto_actor_v1/000019` | #479 | Objetos/owner/ACL; bloques DO en JSON | `7cbfe25c0e78fe34f6f8fee0d986c5e0bf0e8abc7238bdf1d8e7045b250ebb97` | `impedir_revivir_perfil_v1, listar_perfiles_cuenta_v1` | H9; ensayada |
| `autorizacion/000022` | #479 | Objetos/owner/ACL; bloques DO en JSON | `535cf354235083fb98f3bdff7c19c210d0b7e876328441e4faf233d026809495` | `impedir_revivir_asignacion_v1` | H9; ensayada |
| `autorizacion/000023` | #479 | Objetos/owner/ACL; bloques DO en JSON | `30246fab3cd3ba1d290ce0c15f6333d5824cecfff56bcc90eb7edeb59b893095` | `validar_avance_continuidad_admin_v1, avanzar_continuidad_admin_interna_v1; más en JSON` | H9; ensayada |
| `contexto_actor_v1/000020` | #479 | Objetos/owner/ACL; bloques DO en JSON | `a8f719d49a9014585b16c30a4f23f8ea4734f0716d23ee3c375fcc5eed1d867c` | `bloquear_contexto_admin_v1, crear_perfil_vinculo_admin_v1; más en JSON` | H9; ensayada |
| `identidad_sesiones_v1/000009` | #479 | Objetos/owner/ACL; bloques DO en JSON | `a51d8e64f07bb901727689dae432146bdf98c5a3f609d116381e77e92f875e43` | `inmutabilidad_politica_admin_v1, avance_vinculo_certificado_admin_v1; más en JSON` | H9; ensayada |
| `autorizacion/000024` | #479 | Objetos/owner/ACL; bloques DO en JSON | `a07ded3cbc503f409eeff04b937c3bc0fe5d6d78622dd0e1475c7193bbda5f2a` | `json_cadena_canonica_go_admin_v1, fecha_canonica_go_admin_v1; más en JSON` | H9; ensayada |
| `contexto_actor_v1/000026` | #502 | Objetos/owner/ACL; bloques DO en JSON | `a15215662dfca166a2a9cdc0033e7a24261a5ed1935917eb0bc76bff8cca9aa8` | `cotejar_contexto_historico_auditoria_v1` | H9; ensayada |
| `identidad_sesiones_v1/000013` | #502 | Objetos/owner/ACL; bloques DO en JSON | `0002112de0813f8075eca104f6a7d91dd672233baa967d4d9a213ba15a45232b` | `cotejar_autenticacion_historica_auditoria_v1` | H9; ensayada |
| `autorizacion_atestada_v3/000169` | #502 | Objetos/owner/ACL; bloques DO en JSON | `9c9c51245234ea25e0e033535871959530f96c3c68b607a104bca782aa732cc7` | `acreditar_registrador_intentos_v1, registrar_intento_nominal_v1; más en JSON` | H9; ensayada |
| `contexto_actor_v1/000027` | #521 | Objetos/owner/ACL; bloques DO en JSON | `72243349e6af4914c41b39e712484348c5a1be67c5c7b68a1cb7eed8280c8c97` | `delta sobre funciones/tablas existentes` | H9; ensayada |
| `autorizacion_atestada_v3/000171` | #530 | Objetos/owner/ACL; bloques DO en JSON | `7a5b2ac33e53a04c85e854dc558755ec95fc8ba360d34b9559c699c02027a7b8` | `registrar_evento_admin_preperfil_v1` | H9; ensayada |
| `autorizacion_atestada_v3/000160` | #483 | Objetos/owner/ACL; bloques DO en JSON | `b6bc945a36af85ef136b35300655488fe533977c738e021330ef3321b24a7572` | `acreditar_destino_cargo_ct_v1, acreditar_operador_cargos_ct_v1; más en JSON` | H9; ensayada |
| `autorizacion_atestada_v3/000164` | #483 | Objetos/owner/ACL; bloques DO en JSON | `07e2269fe8d40fdfd221d0e397e7681141f7268ba3896c3bb769bd6c029714be` | `comprobar_independencia_cargo_ct_v1, ` | H9; ensayada |
| `autorizacion/000033` | #497 | Objetos/owner/ACL; bloques DO en JSON | `35dbdfa9b9aa43582f7565121e5d6405f77700a53fc713a8ade805ef594333e8` | `acreditar_perfil_aplicacion_nominal_v1, acreditar_ambito_certificado_nominal_v1; más en JSON` | H9; ensayada |
| `contexto_actor_v1/000025` | #497 | Objetos/owner/ACL; bloques DO en JSON | `38364c8118883cf6c56a164a74994b7134f80c115a684d391ff6c66535e686f3` | `fuentes_certificado_firmante_ct_v2, acreditar_organizacion_destino_certificado_v1; más en JSON` | H9; ensayada |
| `autorizacion_atestada_v3/000151` | #440 | POST155 | `78a642daa482e9d7c6dbc33b9ec51d96dfabc66cd194d1290a4670f82a6593cf` | `registrar_y_consumir_consulta_circuito_ct_v3_atestada` | H9; ensayada |
| `contratacion_temporal/000163` | #439 | Objetos/owner/ACL; bloques DO en JSON | `4d8e54209c8c6bab2b6f061edb53d31b8648f434f15a1ec286748a44068931c1` | `circuito_vinculado_ct163, circuito_siguiente_ct163; más en JSON` | H9; ensayada |
| `contratacion_temporal/000164` | #441 | CT163 + CT165 conservada | `1734bc466b2c3ae13582518b8e4cfbc2eaf16f5bafeb5a2abf2e33145b90a942` | `circuito_flujo_nuevo_ct164, circuito_agregado_valido_ct164; más en JSON` | H9; ensayada |
| `autorizacion_atestada_v3/000156` | #448 | AD151 | `07a774db534503a845d34eb710239d0a7b9508c684f4cec7c408c6b66cd2c87f` | `registrar_y_consumir_firma_externa_ct_v3_atestada` | H9; ensayada |
| `autorizacion_atestada_v3/000157` | #451 | AD156; fachada cerrada | `0faa42b50dd092f78d71d6693ad285c6a3489c83df6bc5293a38500219d12ccb` | `registrar_y_consumir_firma_vec_ct_v3_atestada` | H9; ensayada |
| `autorizacion_atestada_v3/000158` | #457 | AD151/156/157 | `4a5c0004e01ca7eb8f0c8cb8e41f0fe18b137b910cbb254f28f0cfc8c73a0b7a` | `delta sobre funciones/tablas existentes` | H9; ensayada |
| `documentos/000013` | #460 | AD158 + Documentos previos | `358603cc746aa3db4ffacbecf965da81f345dbc3ec07af2080705245d820d2b1` | `reservar_identificador_v1, reservar_original_firmable_v1; más en JSON` | H9; ensayada |
| `autorizacion_atestada_v3/000162` | #509 | POST158 | `f9387e72612ec129ee09281802875a8ee7f9a4cd7b8150e4912208dffc377bd9` | `registrar_y_consumir_firma_verificada_ct_v2_atestada, consumir_consulta_firmas_r5_ct_v2_atestada` | H9; ensayada |
| `autorizacion_atestada_v3/000170` | #514 | Objetos/owner/ACL; bloques DO en JSON | `51a157cc3a6e4ff6d13cc2d8c515d4cff42e216294c7ef5d191d461dbdb94381` | `registrar_y_consumir_firma_descriptor_ct_v2_atestada` | H9; ensayada |
| `autorizacion_atestada_v3/000159` | #456 | POST162 | `ccd88c9b57acc1ddeb02d629ee22a20d2571b58ea4a3d774896dc3f14b21aad2` | `consumir_consulta_firmas_r5_ct_v3_atestada` | H9; ensayada |
| `autorizacion_atestada_v3/000165` | #497 | POST159 + AUT33/CA25 | `007325925e706e07872476dc69a22c4dd4144873510747dea19e37b06d5c79bf` | `consumir_publicacion_certificado_nominal_v3_atestada, operar_certificado_nominal_v3` | H9; ensayada |
| `autorizacion_atestada_v3/000166` | #498 | POST165 | `37e81c59df5412bec19ab6ea977c8446ad1ca5f774dcbddd96db85f535bd0302` | `consumir_publicacion_cargo_competencial_v3_atestada` | H9; ensayada |
| `personal/000028` | #498 | Objetos/owner/ACL; bloques DO en JSON | `304aa1761cc2ab7222d2f429ed8087a2fcce55467dd227f50f71db0244046e3a` | `leer_revalidar_cargo_ocupante_ct_v1, publicar_cargo_competencial_v1` | H9; ensayada |
| `personal/000029` | #508 | Objetos/owner/ACL; bloques DO en JSON | `a2e5a385bd12d60a137315ee2c481d85a19a3dc7945d8efa107499d942a1ff74` | `resolver_fuente_cargo_ocupante_ct_v1` | H9; ensayada |
| `autorizacion_atestada_v3/000167` | #496 | Objetos/owner/ACL; bloques DO en JSON | `d46b1dfbca3d4b3636091916fb7716bc8c8b1ca76e568a4203ac0f47d9bfa883` | `comprobar_consumo_firma_ct_v1` | H9; ensayada |
| `autorizacion/000032` | #496 | Objetos/owner/ACL; bloques DO en JSON | `881c208971775f08e0641961780964a34ef7706410c2f8279544d275993ce041` | `bloquear_evidencia_competencia_ct_v1, validar_competencia_nominal_firmante_ct_v1; más en JSON` | H9; ensayada |
| `autorizacion/000035` | #510 | Objetos/owner/ACL; bloques DO en JSON | `9882cba105401f52a1fad5e4404c36444855b07d5d10c16514cfa251e41c1615` | `canon_texto_json_go_ct_v1, canon_json_competencia_firmante_ct_v1; más en JSON` | H9; ensayada |
| `autorizacion/000034` | #500 | Objetos/owner/ACL; bloques DO en JSON | `df0aa95bf3917adb616d00ffdfc073770297dbd855398935a6e7983d02a03127` | `delta sobre funciones/tablas existentes` | H9; ensayada |
| `autorizacion_atestada_v3/000168` | #500 | POST166 + AUT34 | `059936cbe6a5728fe16d34488f9ea11538f91279f1dbcf364fb9f1f030d7753a` | `consumir_capacidades_admin_v3_atestada, consultar_capacidades_admin_v1` | H9; ensayada |
| `contratacion_temporal/000170` | #464 | Objetos/owner/ACL; bloques DO en JSON | `688998cca70e0f72665ea1e76d023c109702c5aa18848fc48a634e2d52c6cb48` | `nueva_huella_firma_historia_v1, avanzar_cabeza_firma_v1; más en JSON` | H9; ensayada |
| `contratacion_temporal/000174` | #493 | Objetos/owner/ACL; bloques DO en JSON | `b27e191c6103298f81ee61921013d5f66accb8ba6fa61af3a703b6ed15200952` | `leer_revalidar_relacion_unidad_expediente_ct_v1` | H9; ensayada |
| `contratacion_temporal/000172` | #515 | Objetos/owner/ACL; bloques DO en JSON | `0d16408c03b66d31c08b2f03e54dda1066a266aa35539d283e7629f09d66f709` | `impedir_degradacion_firma_pdf_v2, comprobar_hija_firma_pdf_v2; más en JSON` | H9; ensayada |
| `personal/000031` | #544 | Objetos/owner/ACL; bloques DO en JSON | `4f193433096b2002e14e7fb03e332ecea3bf159e4717d82f7a2dd830f6f363e4` | `avanzar_barrera_unidad_bootstrap_admin_v1, cotejar_unidad_bootstrap_admin_v1` | H9; ensayada |
| `autorizacion_atestada_v3/000149` | #470 | POST155 | `73a6c7081ae9b11b6c22240d83fa2baabe205aeceb4337e9611b6a40a57b11aa` | `consumir_vinculo_propio_crn11_v3_atestada` | cond.; montaje pendiente |
| `personal/000026` | #470 | Objetos/owner/ACL; bloques DO en JSON | `14e4221926ce8530bd8ae939f0361adf48d5937f01a550aef4d7dcf085890d3d` | `consultar_vinculo_propio_historico_crn11_v1` | cond.; montaje pendiente |
| `autorizacion_atestada_v3/000154` | #438 | POST149 | `87845cf6c3da71912eca2b1bbe798c818e536e626599028a87913890088e5fb9` | `consumir_relacion_para_rpt_v3_atestada` | cond.; montaje pendiente |
| `personal/000027` | #438 | Objetos/owner/ACL; bloques DO en JSON | `66bdeee92c6941cbbf66fd47d12ea0309ca7ff76083c2441ce7804ba76e1f646` | `avanzar_generacion_relacion_rpt_v1, consultar_relacion_para_rpt_v1` | cond.; montaje pendiente |
| `personal/000030` | #532 | Personal27 fuente 9e4733bc… | `4f24fcc7ab1035cb8c905662239f265c55aec075a8f24513517d7615a4945e41` | `delta sobre funciones/tablas existentes` | cond.; montaje pendiente |
| `autorizacion_atestada_v3/000161` | #507 | POST149 | `9c9e607de0ae3d22b9ef2b474dde87c281ff2558d50cca6094b5483557a74501` | `consumir_consulta_version_convocatoria_v3_atestada, consumir_operacion_meritos_v3_atestada; más en JSON` | cond.; montaje pendiente |
| `bolsa_convocatorias/000007` | #390 | Objetos/owner/ACL; bloques DO en JSON | `90cfe7d9dfbb41888c5d84cae1ef25cf522525f56fd7917dccff2216abd72ba5` | `obtener_version_exacta_v3` | cond.; montaje pendiente |
| `meritos/000001` | #400 | Objetos/owner/ACL; bloques DO en JSON | `f1faf970e2394179caec71d763da9ea8c4d814463b0a6047dc583eb441adcc0b` | `referencia_valida_v1, claves_exactas_v1; más en JSON` | cond.; montaje pendiente |
| `bolsa_reglas_baremo/000004` | #412 | Objetos/owner/ACL; bloques DO en JSON | `ddc1e06b024055063c1dfc42de9543db81622d46d75358c8d149ce534297e3a4` | `validar_material_borrador_v3, operar_borrador_v3` | cond.; montaje pendiente |
| `autorizacion_atestada_v3/000145` | #435 | POST161 | `a1db9084203be7731af490e54eb8061cd7c5ec5c95bd93c6af87a6bd993644f5` | `consumir_consulta_hecho_propio_v3_atestada` | cond.; montaje pendiente |
| `meritos/000002` | #435 | Objetos/owner/ACL; bloques DO en JSON | `f35f0f950541ec0bab484939026af7456298340063b4c05b1de05a8d2e200f21` | `consultar_hecho_propio_v1` | cond.; montaje pendiente |
| `autorizacion_atestada_v3/000153` | #445 | POST145 | `023dbd1607599ce496ffc00113b91736f4fc029ddfb5f72e15b9b1c663516794` | `consumir_guardar_preparacion_bases_bolsa_v3_atestada, consumir_consultar_preparacion_bases_bolsa_v3_atestada` | cond.; montaje pendiente |
| `bolsa_convocatorias/000008` | #445 | Objetos/owner/ACL; bloques DO en JSON | `7156fd573a05f152399fd49e362238311b01c3ff78d23da0fa26185fd73f6b8e` | `avanzar_preparacion_bases_actual_v3, validar_material_preparacion_bases_v3; más en JSON` | cond.; montaje pendiente |
| `contratacion_temporal/000168` | #442 | POST-CT162 | `68fd471d8b810205de7edf5566560550de6888874199dabb461db0d76dfc9db2` | `delta sobre funciones/tablas existentes` | cond.; montaje pendiente |
| `autorizacion_atestada_v3/000141` | #390 | Objetos/owner/ACL; bloques DO en JSON | `77fa9038841699c2a903a7feac6e34913e1423c798071b85b189f2146c6236d2` | `consumir_consulta_version_convocatoria_v3_atestada` | AD161; no ejecutar |
| `autorizacion_atestada_v3/000142` | #400 | Objetos/owner/ACL; bloques DO en JSON | `f1ed0a48b50b8665f36718623bde930619b833d67b54686b1f1b94cd694cb310` | `consumir_operacion_meritos_v3_atestada` | AD161; no ejecutar |
| `autorizacion_atestada_v3/000144` | #411 | Objetos/owner/ACL; bloques DO en JSON | `bb4d966c663491c9912d98b4c5af67dbfaef1e9a4ba24ecdd7333c893019f05e` | `registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada` | AD161; no ejecutar |

## Verificación y límites

La comprobación estática coteja 56 rutas de migraciones contra el árbol fijado y siete soportes. El orden contiene 39 migraciones seleccionadas y un soporte de roles, 40 archivos únicos. Las 14 condicionadas y tres históricas quedan fuera de la lista. Rutas, SHA, consumidores y guardas constan en `orden.json`; la selección fue aplicada completa en el clon.

La configuración actual conserva apagados o ausentes los consumidores nuevos de CRN11/RPT/Selectivos. Eso permite excluir sus cadenas incompatibles sin reconstruirlas ni saltar guardas. Arrancar tras la selección no demuestra que cada migración sea por sí sola obligatoria al arrancar; el JSON expresa ese límite. Se incluyen por los consumidores de main y sus dependencias, evitando que una operación alcance una función ausente.

`ejecutable=true` acredita el ensayo del paquete, no una instalación. Claude debe verificar preimagen y cierre de escritores antes de aplicarlo. Instalar fuentes nominales no publica perfiles ni cargos ni firma documentos. No se habilitan aquí consumidores excluidos ni se prueba el circuito de Firma completo.

## Primer ensayo, conservado como antecedente

PostgreSQL 18.4 se ejecutó con UID/GID del operador, límite de 2 GB, `--network none`, `--rm` y datos en disco bajo el estado privado de P. Las 15 SQL que reconstruyen la principal terminaron correctamente. El núcleo inicial coincidió con definición `5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330` y fuente `cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e`.

Se confirmaron 34 migraciones nuevas y el soporte de roles. AD169 falló primero porque el inventario omitía `roles_intentos_up.sql`: se comprobó el rollback, se añadió el soporte existente y se confirmó AD169 una vez. No se modificó SQL de producto. CT170 falló después por AUT30 ausente; su transacción revirtió completa. CT170, CT174, CT172 y Personal31 no quedaron instaladas en el clon.

El binario de la base reprodujo SHA256 `6ee586a88d984e8293a32d6e3a654a52e2f23ec3fd0d802d4ea5c92740c8fdda`. El binario de main fijado tiene SHA256 `5d84b1b0637181328f0228862c8bfc25457f5b613ebf810b2becf74e68765d5b`. Ambos emitieron `vec server listening` con la configuración actual copiada por lectura, TCP/TLS y almacenes sintéticos existentes. La adaptación inicial a socket se descartó porque Usuarios exige una topología TCP. No se cambió esa guarda.

En main, siete lecturas HTTP dieron 200 antes y después de reiniciar aplicación y PostgreSQL: portal, pantalla de Calendarios, centros, calendario, bolsas, cuadro CT y detalle CT. Los datos coinciden, salvo las fechas de generación de cada consulta. Se conservaron 71 expedientes y un detalle de versión 9 con nueve hitos. La comparación HTTP acredita ese recorrido existente con SQL parcial; no acredita las operaciones nuevas de Firma ni un recorrido en Chrome.

La puerta `scripts/verificar_calidad.sh`, con `-p 8`, caché en disco y TMPDIR corto, falló en ocho paquetes intactos: `cmd/vec-admin`, `cmd/vec-auditoria-checkpoint`, `cmd/vec-comparar-organizacion`, `cmd/vec-preparar-admin-bootstrap`, `cmd/vec-preparar-material-interno`, `cmd/vec-provisionar-candidato-externo`, `internal/app/bootstrap` e `internal/app/composicion/internactproveedores`. No se repitió la puerta completa. ShellCheck, sintaxis Bash, mocks de recuperación y Semgrep local dieron verde. No hay cambios Go en esta pieza; gosec no tiene paquetes cambiados que analizar.

Las exclusiones CRN11/RPT/Selectivos siguen fuera. La principal mantiene B2 e incorporación acreditada apagados y carece del montaje de esas consultas nuevas; el arranque observado no las exige. No se omiten de forma silenciosa: sus conflictos de preimagen siguen anotados. La necesidad individual de cada SQL del tramo confirmado no se ha aislado; arrancar tras el conjunto no demuestra que todas sean obligatorias.

Antes de instalar faltan AUT30 en main, el ensayo causal completo, la puerta verde y el recorrido de las capacidades nuevas. También falta que Dirección acredite la exclusión de los diez procesos `psql` ajenos observados en el espacio de red de la principal. P no los paró. El guion exige mantenimiento privado revisado y coteja los montajes efectivos de binario, web, catálogos y PGDATA.

## Reanudación tras AUT30 integrada

La PR #453 entró en main por `63eb05717229f904e1d3a5d9dedcff8158abe9c1`. El árbol se actualizó a `39ec9858d2983e24786db9d909e86cdce19db720`, con una UP adicional desde el inventario inicial: AUT30. La copia fría se reconstruyó en un clon nuevo porque el anterior ya estaba retirado. Las 15 SQL de base y las 40 de la selección completa terminaron correctamente, una confirmación por archivo. No se modificó SQL de producto. El nuevo binario SHA256 `16b8d71f19dc34db4d4bac35e69179526cdcea128ebef7da06f5e8b6f4248922` emitió `vec server listening` con TCP/TLS, configuración y almacenes copiados por lectura.

Los primeros fallos globales procedían de un directorio ajeno `/tmp/.git`: las pruebas rechazaban sus claves sintéticas por estar bajo un ancestro Git. Se dejó intacto. Con un temporal privado en disco los dos paquetes focales fallidos dieron verde; la puerta completa está en curso en ese entorno corregido. No se debilitó ninguna guarda.

| Migración añadida | PR | Preimagen | SHA256 | Consumidor |
| --- | --- | --- | --- | --- |
| AUT30 | #453, integrada | Fachada ausente, roles/tablas centrales presentes, migrador superusuario | 704933566ff5170feb01bfc5f9a3a650be21189306d20348293733c4ae676c32 | CT170; su operación legada permanece cerrada. |

## Cierre del ensayo

`39ec9858d` queda fijado para H9. Son 48 UP nuevas y ocho pendientes anteriores; se seleccionan 39 migraciones más un soporte. Las otras 17 permanecen fuera por las condiciones de montaje y preimágenes descritas. Nunca se reaplican las 15 SQL de la principal: se usaron solo para reconstruir el clon.

La puerta completa `scripts/verificar_calidad.sh` terminó con código 0, Go con `-p 8`, caché en disco y `TMPDIR=/var/tmp/h9p`. Go, `-race`, vet, Node, PDF, i18n, manifiestos y dependencias dieron verde. `/tmp/.git` rechazaba las claves de prueba y AppArmor de qpdf impedía escribir en el temporal oculto bajo HOME: se corrigió el lugar del temporal, sin modificar guardas ni archivos ajenos y sin excepciones de seguridad.

Tras reiniciar aplicación y PostgreSQL, las siete lecturas HTTP retornaron 200 con datos idénticos, 71 expedientes y el mismo detalle v9/nueve hitos. Chrome del sistema 149 mostró CT, Bolsa y Calendarios a 1440/390 sin errores JS, cookies, almacenamiento web ni desbordamiento global. La consulta adicional de CT127 confirmó nueve hitos, cancelación y firmas 200; borradores 404 en ese expediente. Quedan anotados avisos TLS del worker de Playwright y consultas adicionales bloqueadas por la restricción del recorrido: no se presentan como un recorrido sin incidencias ni como validación de Firma completa.

El guion final tiene dos GO independientes sobre `25f5cd8cc38cbb90f690b5eea0e2907c21ea4a3a`; el snapshot SQL conserva sus dos revisiones. Recupera automáticamente la copia fría, los árboles realmente servidos y el servicio anterior. Si el servicio anterior o su mantenimiento fallan, contiene la aplicación. No abre tráfico con un arranque fallido ni restaura después de haber aceptado escrituras.
