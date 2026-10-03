package contrastecopias

// Todos los selectores de catálogo excluyen OID de su representación semántica.
// Los OID de joins/localizadores nunca aparecen en las huellas publicadas.
const aclNamespace = `(n.nspname !~ '^pg_' OR n.nspname='pg_catalog')`

const userNamespace = `n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'`

type consulta struct{ clase, sql string }

var agregados = []consulta{
	{"esquema", esquemaSQL},
	{"roles", rolesSQL},
	{"acl", aclSQL},
	{"extensiones", `SELECT jsonb_build_array(e.extname,e.extversion,pg_get_userbyid(e.extowner),n.nspname,e.extrelocatable)::text FROM pg_extension e JOIN pg_namespace n ON n.oid=e.extnamespace`},
	{"privilegios_defecto", `SELECT jsonb_build_array(pg_get_userbyid(d.defaclrole),CASE WHEN d.defaclnamespace=0 THEN '' ELSE n.nspname END,d.defaclobjtype,coalesce((SELECT jsonb_agg(jsonb_build_array(pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) ORDER BY pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) FROM aclexplode(d.defaclacl) a),'[]'::jsonb))::text FROM pg_default_acl d LEFT JOIN pg_namespace n ON n.oid=d.defaclnamespace`},
}

const esquemaSQL = `
 SELECT jsonb_build_array('database',d.datname,pg_get_userbyid(d.datdba),pg_encoding_to_char(d.encoding),d.datlocprovider,d.datcollate,d.datctype,d.datlocale,d.daticurules,d.datcollversion,d.datconnlimit,d.datallowconn,t.spcname,shobj_description(d.oid,'pg_database'),pg_database_collation_actual_version(d.oid))::text
 FROM pg_database d JOIN pg_tablespace t ON t.oid=d.dattablespace WHERE d.datname=current_database()
 UNION ALL SELECT jsonb_build_array('tablespace',s.spcname,pg_get_userbyid(s.spcowner),s.spcoptions,shobj_description(s.oid,'pg_tablespace'))::text FROM pg_tablespace s
 UNION ALL SELECT jsonb_build_array('namespace',n.nspname,pg_get_userbyid(n.nspowner),obj_description(n.oid,'pg_namespace'))::text FROM pg_namespace n WHERE ` + userNamespace + `
 UNION ALL SELECT jsonb_build_array('relation',n.nspname,r.relname,r.relkind,r.relpersistence,pg_get_userbyid(r.relowner),r.relrowsecurity,r.relforcerowsecurity,r.relreplident,r.reloptions,am.amname,coalesce(s.spcname,''))::text FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace LEFT JOIN pg_am am ON am.oid=r.relam LEFT JOIN pg_tablespace s ON s.oid=r.reltablespace WHERE ` + userNamespace + `
 UNION ALL SELECT jsonb_build_array('column',n.nspname,r.relname,row_number() OVER(PARTITION BY r.oid ORDER BY a.attnum),a.attname,format_type(a.atttypid,a.atttypmod),tn.nspname,t.typname,a.attnotnull,a.attidentity,a.attgenerated,pg_get_expr(d.adbin,d.adrelid,false),coalesce(cn.nspname,''),coalesce(coll.collname,''),a.attstorage,a.attcompression,a.attoptions)::text FROM pg_attribute a JOIN pg_class r ON r.oid=a.attrelid JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_type t ON t.oid=a.atttypid JOIN pg_namespace tn ON tn.oid=t.typnamespace LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum LEFT JOIN pg_collation coll ON coll.oid=a.attcollation LEFT JOIN pg_namespace cn ON cn.oid=coll.collnamespace WHERE ` + userNamespace + ` AND a.attnum>0 AND NOT a.attisdropped
 UNION ALL SELECT jsonb_build_array('constraint',n.nspname,r.relname,c.conname,c.contype,pg_get_constraintdef(c.oid,false),c.convalidated,c.condeferrable,c.condeferred,c.connoinherit)::text FROM pg_constraint c JOIN pg_class r ON r.oid=c.conrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + `
 UNION ALL SELECT jsonb_build_array('index',n.nspname,r.relname,i.indisunique,i.indisprimary,i.indisexclusion,i.indimmediate,i.indisvalid,i.indisready,i.indisreplident,i.indnullsnotdistinct,pg_get_indexdef(i.indexrelid,0,false))::text FROM pg_index i JOIN pg_class r ON r.oid=i.indexrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + `
 UNION ALL SELECT jsonb_build_array('sequence',n.nspname,r.relname,format_type(s.seqtypid,NULL),s.seqstart::text,s.seqincrement::text,s.seqmax::text,s.seqmin::text,s.seqcache::text,s.seqcycle,(SELECT jsonb_agg(jsonb_build_array(tn.nspname,tr.relname,a.attname,d.deptype) ORDER BY tn.nspname,tr.relname,a.attname,d.deptype) FROM pg_depend d JOIN pg_class tr ON tr.oid=d.refobjid JOIN pg_namespace tn ON tn.oid=tr.relnamespace JOIN pg_attribute a ON a.attrelid=tr.oid AND a.attnum=d.refobjsubid WHERE d.classid='pg_class'::regclass AND d.objid=r.oid AND d.refclassid='pg_class'::regclass AND d.deptype IN('a','i')))::text FROM pg_sequence s JOIN pg_class r ON r.oid=s.seqrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + `
 UNION ALL SELECT jsonb_build_array('view',n.nspname,r.relname,pg_get_viewdef(r.oid,false))::text FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + ` AND r.relkind IN('v','m')
 UNION ALL SELECT jsonb_build_array('trigger',n.nspname,r.relname,t.tgname,t.tgenabled,pg_get_triggerdef(t.oid,false))::text FROM pg_trigger t JOIN pg_class r ON r.oid=t.tgrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + ` AND NOT t.tgisinternal
`

const rolesSQL = `
 SELECT jsonb_build_array('role',r.rolname,r.rolsuper,r.rolinherit,r.rolcreaterole,r.rolcreatedb,r.rolcanlogin,r.rolreplication,r.rolbypassrls,r.rolconnlimit,r.rolvaliduntil::text,r.rolconfig)::text FROM pg_roles r
 UNION ALL SELECT jsonb_build_array('membership',parent.rolname,member.rolname,grantor.rolname,a.admin_option,a.inherit_option,a.set_option)::text FROM pg_auth_members a JOIN pg_roles parent ON parent.oid=a.roleid JOIN pg_roles member ON member.oid=a.member JOIN pg_roles grantor ON grantor.oid=a.grantor
 UNION ALL SELECT jsonb_build_array('setting',coalesce(d.datname,''),coalesce(r.rolname,''),s.setconfig)::text FROM pg_db_role_setting s LEFT JOIN pg_database d ON d.oid=s.setdatabase LEFT JOIN pg_roles r ON r.oid=s.setrole
`

const aclSQL = `
 SELECT jsonb_build_array('database',d.datname,pg_get_userbyid(d.datdba),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname=current_database()
 UNION ALL SELECT jsonb_build_array('namespace',n.nspname,pg_get_userbyid(n.nspowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE ` + aclNamespace + `
 UNION ALL SELECT jsonb_build_array('relation',n.nspname,r.relname,r.relkind,pg_get_userbyid(r.relowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace CROSS JOIN LATERAL aclexplode(coalesce(r.relacl,acldefault(CASE WHEN r.relkind='S' THEN 's'::"char" ELSE 'r'::"char" END,r.relowner))) a WHERE ` + aclNamespace + ` AND r.relkind IN('r','S','v','m','f','p')
 UNION ALL SELECT jsonb_build_array('column',n.nspname,r.relname,c.attname,pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_attribute c JOIN pg_class r ON r.oid=c.attrelid JOIN pg_namespace n ON n.oid=r.relnamespace CROSS JOIN LATERAL aclexplode(c.attacl) a WHERE ` + aclNamespace + ` AND c.attnum>0 AND NOT c.attisdropped
 UNION ALL SELECT jsonb_build_array('function',n.nspname,p.proname,pg_get_function_identity_arguments(p.oid),pg_get_userbyid(p.proowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE ` + aclNamespace + `
 UNION ALL SELECT jsonb_build_array('type',n.nspname,t.typname,pg_get_userbyid(t.typowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a WHERE ` + aclNamespace + `
 UNION ALL SELECT jsonb_build_array('tablespace',s.spcname,pg_get_userbyid(s.spcowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_tablespace s CROSS JOIN LATERAL aclexplode(coalesce(s.spcacl,acldefault('t',s.spcowner))) a
 UNION ALL SELECT jsonb_build_array('parameter',p.parname,pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_parameter_acl p CROSS JOIN LATERAL aclexplode(p.paracl) a
 UNION ALL SELECT jsonb_build_array('language_owner',l.lanname,pg_get_userbyid(l.lanowner))::text FROM pg_language l
 UNION ALL SELECT jsonb_build_array('language',l.lanname,pg_get_userbyid(l.lanowner),pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)::text FROM pg_language l CROSS JOIN LATERAL aclexplode(coalesce(l.lanacl,acldefault('l',l.lanowner))) a
`

type comprobacion struct{ motivo, sql string }

var comprobaciones = []comprobacion{
	{"catalogo_sistema_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname='pg_catalog' AND r.oid>=16384) OR EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='pg_catalog' AND p.oid>=16384) OR EXISTS(SELECT 1 FROM pg_cast WHERE oid>=16384)`},

	{"credenciales_roles_no_comprobables", `SELECT EXISTS(SELECT 1 FROM pg_authid WHERE rolpassword IS NOT NULL)`},
	{"etiquetas_seguridad_no_admitidas", `SELECT EXISTS(SELECT 1 FROM pg_seclabel) OR EXISTS(SELECT 1 FROM pg_shseclabel)`},

	{"relacion_no_admitida", `SELECT EXISTS(SELECT 1 FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + ` AND (r.relkind NOT IN('r','i','S','t') OR r.relpersistence<>'p' OR r.relrowsecurity OR r.relforcerowsecurity))`},
	{"tipo_columna_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_class r ON r.oid=a.attrelid JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_type t ON t.oid=a.atttypid JOIN pg_namespace tn ON tn.oid=t.typnamespace WHERE ` + userNamespace + ` AND r.relkind='r' AND a.attnum>0 AND NOT a.attisdropped AND (a.attgenerated='v' OR tn.nspname<>'pg_catalog' OR t.typname NOT IN('bool','int2','int4','int8','float4','float8','numeric','text','varchar','bpchar','bytea','uuid','date','time','timetz','timestamp','timestamptz','interval','json','jsonb')))`},
	{"funciones_no_admitidas", `SELECT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE ` + userNamespace + `)`},
	{"tipos_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE ` + userNamespace + ` AND t.typrelid=0 AND t.typelem=0)`},
	{"disparadores_internos_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_trigger t JOIN pg_class r ON r.oid=t.tgrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + ` AND t.tgisinternal AND t.tgenabled<>'O')`},
	{"disparadores_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_trigger t JOIN pg_class r ON r.oid=t.tgrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + ` AND NOT t.tgisinternal)`},
	{"herencia_no_admitida", `SELECT EXISTS(SELECT 1 FROM pg_inherits)`},
	{"reglas_no_admitidas", `SELECT EXISTS(SELECT 1 FROM pg_rewrite t JOIN pg_class r ON r.oid=t.ev_class JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + `)`},
	{"politicas_no_admitidas", `SELECT EXISTS(SELECT 1 FROM pg_policy)`},
	{"eventos_ddl_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_event_trigger)`},
	{"extension_no_admitida", `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname<>'plpgsql')`},
	{"objetos_extension_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_depend d JOIN pg_extension e ON e.oid=d.refobjid WHERE d.refclassid='pg_extension'::regclass AND d.deptype='e' AND e.extname<>'plpgsql')`},
	{"collation_no_admitida", `SELECT EXISTS(SELECT 1 FROM pg_collation coll JOIN pg_namespace n ON n.oid=coll.collnamespace WHERE ` + userNamespace + `)`},
	{"indice_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_am a ON a.oid=r.relam WHERE ` + userNamespace + ` AND ((r.relkind='r' AND a.amname<>'heap') OR (r.relkind='i' AND a.amname<>'btree')))`},
	{"operadores_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_operator o JOIN pg_namespace n ON n.oid=o.oprnamespace WHERE ` + userNamespace + `) OR EXISTS(SELECT 1 FROM pg_opclass o JOIN pg_namespace n ON n.oid=o.opcnamespace WHERE ` + userNamespace + `) OR EXISTS(SELECT 1 FROM pg_opfamily o JOIN pg_namespace n ON n.oid=o.opfnamespace WHERE ` + userNamespace + `)`},
	{"foreign_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_foreign_server) OR EXISTS(SELECT 1 FROM pg_foreign_data_wrapper) OR EXISTS(SELECT 1 FROM pg_user_mapping)`},
	{"replicacion_no_admitida", `SELECT EXISTS(SELECT 1 FROM pg_publication) OR EXISTS(SELECT 1 FROM pg_subscription)`},
	{"tablespace_externo_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_tablespace WHERE spcname NOT IN('pg_default','pg_global'))`},
	{"otras_bases_no_inventariadas", `SELECT EXISTS(SELECT 1 FROM pg_database WHERE NOT datistemplate AND datname<>current_database())`},
	{"objetos_grandes_sin_vinculos_comprobables", `SELECT EXISTS(SELECT 1 FROM pg_largeobject_metadata)`},
	{"estadisticas_extendidas_no_admitidas", `SELECT EXISTS(SELECT 1 FROM pg_statistic_ext)`},
	{"texto_busqueda_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_ts_config c JOIN pg_namespace n ON n.oid=c.cfgnamespace WHERE ` + userNamespace + `) OR EXISTS(SELECT 1 FROM pg_ts_dict c JOIN pg_namespace n ON n.oid=c.dictnamespace WHERE ` + userNamespace + `) OR EXISTS(SELECT 1 FROM pg_ts_parser c JOIN pg_namespace n ON n.oid=c.prsnamespace WHERE ` + userNamespace + `) OR EXISTS(SELECT 1 FROM pg_ts_template c JOIN pg_namespace n ON n.oid=c.tmplnamespace WHERE ` + userNamespace + `)`},
	{"comentarios_no_admitidos", `SELECT EXISTS(SELECT 1 FROM pg_description d WHERE (d.classoid='pg_class'::regclass AND d.objoid IN(SELECT r.oid FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace WHERE ` + userNamespace + `)) OR (d.classoid='pg_namespace'::regclass AND d.objoid IN(SELECT n.oid FROM pg_namespace n WHERE ` + userNamespace + ` AND n.nspname<>'public'))) OR EXISTS(SELECT 1 FROM pg_shdescription WHERE classoid='pg_authid'::regclass AND objoid IN(SELECT oid FROM pg_roles WHERE rolname !~ '^pg_'))`},
	{"lenguaje_no_admitido", `SELECT EXISTS(SELECT 1 FROM pg_language WHERE lanname NOT IN('internal','c','sql','plpgsql'))`},
	{"acceso_catalogo_incompleto", `SELECT NOT (SELECT rolsuper FROM pg_roles WHERE rolname=current_user)`},
}
