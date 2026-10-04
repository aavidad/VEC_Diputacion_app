package bootstrap

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/auditoria"
)

type relojGobiernoUsuariosEnsayo struct{}

func (relojGobiernoUsuariosEnsayo) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// Opt-in exclusivo del clon sintético privado. El test normal no abre PG ni
// genera material; preparación y aplicación son fases separadas.
func TestGobiernoUsuariosPostgreSQLPrivado(t *testing.T) {
	ruta := os.Getenv("VEC_GOBIERNO_USUARIOS_ENSAYO_CONFIG")
	if ruta == "" {
		t.Skip("ensayo_privado_no_configurado")
	}
	b, err := leerFicheroMaterialSeguro(ruta, 16384)
	if err != nil {
		t.Fatal("ensayo_config_invalida")
	}
	defer borrarBytes(b)
	var f configuracionEnsayoGobiernoUsuarios
	if decodificarGobiernoUsuarios(b, &f) != nil || validarConfiguracionEnsayoGobiernoUsuarios(f) != nil {
		t.Fatal("ensayo_config_invalida")
	}
	raizSalida, err := AbrirRaizPrivadaDenominacionPersona(filepath.Join(f.Salida, "configuracion-material.json"))
	if err != nil {
		t.Fatal("ensayo_salida_invalida")
	}
	defer raizSalida.Close()
	escribir := func(nombre string, data []byte) {
		t.Helper()
		if escribirEnsayoGobiernoUsuarios(raizSalida, nombre, data) != nil {
			t.Fatal("ensayo_salida_invalida")
		}
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, f.DSNPropietario)
	if err != nil {
		t.Fatal("ensayo_pg_no_disponible")
	}
	defer pool.Close()
	if f.Fase == "verificar" {
		var data []byte
		query := `WITH rows AS(SELECT * FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='intento_gobierno_usuarios_admin' ORDER BY secuencia), rango AS(SELECT min(secuencia) primero,max(secuencia) ultimo,count(*) cuenta FROM rows)
   SELECT jsonb_build_object('esquema',$1::text,'manifiesto',jsonb_build_object('cadena_id','cadena:comun:interna','primera_secuencia',ra.primero,'ultima_secuencia',ra.ultimo,'registros',ra.cuenta,'anterior_sha256',(SELECT anterior_sha256 FROM rows ORDER BY secuencia LIMIT 1),'cabeza_sha256',(SELECT huella_sha256 FROM rows ORDER BY secuencia DESC LIMIT 1)),
   'registros',(SELECT jsonb_agg(jsonb_build_object('tipo_registro',a.tipo_registro,'intento_gobierno_usuarios',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'anterior_sha256',a.anterior_sha256,'huella_sha256',a.huella_sha256,'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'evento_ref',a.evento_ref,'evento_material_sha256',a.evento_material_sha256,'operador_login',a.operador_login,'accion',a.accion,'modulo_id',a.modulo_id,'recurso_ref',a.recurso_ref,'resultado',a.resultado,'motivo_ref',a.motivo_ref,'proceso',a.proceso,'canal',a.canal,'finalidad_ref',a.finalidad_ref,'correlacion_ref',a.correlacion_ref,'solicitud_sha256',a.gobierno_usuarios_solicitud_sha256)) ORDER BY a.secuencia) FROM rows a)) FROM rango ra`
		if pool.QueryRow(ctx, query, auditoria.EsquemaVerificacionGobiernoUsuarios).Scan(&data) != nil {
			t.Fatal("ensayo_cadena_no_disponible")
		}
		var doc auditoria.DocumentoVerificacionMixta
		if decodificarGobiernoUsuarios(data, &doc) != nil {
			t.Fatal("ensayo_cadena_invalida")
		}
		informe := auditoria.VerificarCadenaGobiernoUsuariosV1(doc, doc.Manifiesto, 100)
		b, _ := json.Marshal(informe)
		escribir("verificacion-cadena.json", b)
		if informe.Estado != "verificada" {
			t.Fatalf("ensayo_cadena_rechazada_%s", informe.Fallo.Codigo)
		}
		return
	}
	type instantanea struct {
		Revision    string    `json:"revision"`
		Secuencia   uint64    `json:"secuencia"`
		SPKI        string    `json:"spki"`
		ClaveID     string    `json:"clave_id"`
		Version     uint64    `json:"version"`
		Audiencia   string    `json:"audiencia"`
		Desde       time.Time `json:"desde"`
		Hasta       time.Time `json:"hasta"`
		Orden       uint64    `json:"orden"`
		MaxVersion  uint64    `json:"max_version"`
		MaxRevision uint64    `json:"max_revision"`
		PreSHA      string    `json:"pre_sha"`
	}
	var actual instantanea
	var raw []byte
	if err = pool.QueryRow(ctx, `SELECT jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,'spki',encode(r.clave_publica_spki,'base64'),'clave_id',r.clave_id,'version',r.version,'audiencia',r.audiencia_despliegue,'desde',r.valida_desde,'hasta',r.valida_hasta,'orden',(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),'max_version',(SELECT max(version) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'max_revision',(SELECT max(revision_gobierno) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),'pre_sha',encode(sha256(convert_to(vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()::text,'UTF8')),'hex')) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version ORDER BY p.orden DESC LIMIT 1`).Scan(&raw); err != nil {
		t.Fatal("ensayo_preimagen_no_disponible")
	}
	if decodificarGobiernoUsuarios(raw, &actual) != nil {
		t.Fatal("ensayo_preimagen_invalida")
	}
	reloj := relojGobiernoUsuariosEnsayo{}
	now := reloj.Ahora()
	dia := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	der, err := base64.StdEncoding.DecodeString(actual.SPKI)
	if err != nil {
		t.Fatal("ensayo_spki_invalido")
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		t.Fatal("ensayo_spki_invalido")
	}
	pub, ok := key.(ed25519.PublicKey)
	if !ok {
		t.Fatal("ensayo_spki_invalido")
	}
	cfg := ConfiguracionMaterialUsuariosAdmin{DirectorioMaterial: f.DirectorioMaterial, RutaConfiguracionHMAC: f.RutaConfiguracionHMAC, PrefijoEvidencia: "evidencia:firma:admin:usuarios:", Raiz: administracion.MaterialRaizPerfilesV3{ClaveID: actual.ClaveID, Version: actual.Version, Audiencia: actual.Audiencia, Publica: pub, Estado: confianza.EstadoClaveAtestacionAutorizacionV3Activa, ValidaDesde: actual.Desde.UTC(), ValidaHasta: actual.Hasta.UTC()}}
	configFile := filepath.Join(f.Salida, "configuracion-material.json")
	if f.Fase == "preparar" {
		cfg.Gobierno = administracion.GobiernoConfianzaPerfilesV3{Revision: "confianza:atestacion:ct:desarrollo:" + dia.Format("2006-01-02"), Secuencia: actual.Secuencia + 1, PublicadaEn: dia, ExpiraEn: dia.Add(24 * time.Hour)}
		raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cfg.Raiz.ClaveID, cfg.Raiz.Version, pub, cfg.Raiz.Audiencia, cfg.Raiz.Estado, cfg.Raiz.ValidaDesde, cfg.Raiz.ValidaHasta, time.Time{})
		if err != nil {
			t.Fatal("ensayo_raiz_invalida")
		}
		g, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(cfg.Gobierno.Revision, cfg.Gobierno.Secuencia, dia, dia.Add(24*time.Hour), raiz)
		if err != nil {
			t.Fatal("ensayo_gobierno_invalido")
		}
		cfg.Gobierno.HuellaSHA256, err = g.HuellaSHA256ParaGobierno()
		if err != nil {
			t.Fatal("ensayo_gobierno_invalido")
		}
		for i, e := range []struct{ a, n string }{{administracion.AudienciaUsuariosListarV3, "listar"}, {administracion.AudienciaUsuariosConsultarV3, "consultar"}} {
			cfg.Entradas = append(cfg.Entradas, DescriptorClaveUsuariosAdmin{Audiencia: e.a, Dominio: "vec.admin.desarrollo.usuarios." + e.n + ".capacidad-v3", PrefijoClave: "clave:capacidad:admin:usuarios:" + e.n + ":", EmisorID: "emisor:admin:usuarios:desarrollo:v1", Version: actual.MaxVersion + uint64(i) + 1, RevisionGobierno: actual.MaxRevision + uint64(i) + 1, ValidaDesde: now.Add(-time.Minute), ValidaHasta: now.Add(2 * time.Hour)})
		}
	} else {
		data, err := leerFicheroMaterialSeguro(configFile, 16384)
		if err != nil || decodificarGobiernoUsuarios(data, &cfg) != nil {
			t.Fatal("ensayo_material_original_ausente")
		}
		// JSON UTC de configuración procede de los bytes originales, no del reloj.
		cfg.Raiz.ValidaDesde = cfg.Raiz.ValidaDesde.UTC()
		cfg.Raiz.ValidaHasta = cfg.Raiz.ValidaHasta.UTC()
		cfg.Gobierno.PublicadaEn = cfg.Gobierno.PublicadaEn.UTC()
		cfg.Gobierno.ExpiraEn = cfg.Gobierno.ExpiraEn.UTC()
		for i := range cfg.Entradas {
			cfg.Entradas[i].ValidaDesde = cfg.Entradas[i].ValidaDesde.UTC()
			cfg.Entradas[i].ValidaHasta = cfg.Entradas[i].ValidaHasta.UTC()
		}
	}
	m, err := PrepararMaterialUsuariosAdmin(ctx, cfg, reloj)
	if err != nil {
		mat, e := cargarMaterialIdempotenciaDesarrollo(cfg.DirectorioMaterial, cfg.RutaConfiguracionHMAC)
		if e == nil {
			d, e := nuevoDerivadorIdentidadOperacionDesarrollo(&mat)
			if e == nil {
				defer d.borrar()
				base, e := nuevoMaterialAtestacionContratacionTemporalDesarrollo(d, reloj.Ahora())
				if e == nil {
					defer borrarBytes(base.privada)
					defer borrarBytes(base.claveHMAC)
					pubDER, _ := x509.MarshalPKIXPublicKey(base.privada.Public())
					actualSHA := sha256.Sum256(pubDER)
					esperadoSHA := sha256.Sum256(der)
					t.Fatalf("PARO clave=SPKI_sha actual=%s esperado=%s", hex.EncodeToString(actualSHA[:]), hex.EncodeToString(esperadoSHA[:]))
				}
			}
		}
		t.Fatal("ensayo_proveedor_no_coincide_con_raiz")
	}
	defer m.Cerrar()
	var secreto bytes.Buffer
	defer func() { borrarBytes(secreto.Bytes()); secreto.Reset() }()
	if m.EscribirMaterialPrivado(&secreto) != nil {
		t.Fatal("ensayo_material_no_disponible")
	}

	if f.Fase == "preparar" {
		escribir("material.json", secreto.Bytes())
		datos, _ := json.Marshal(cfg)
		escribir("configuracion-material.json", datos)
		p := planGobiernoUsuariosAdmin{Version: 1, OperacionRef: "gcu_" + actual.PreSHA[:32], PreparadoEn: now, CaducaEn: now.Add(2 * time.Hour), PreimagenSHA256: actual.PreSHA, Ordenes: []uint64{actual.Orden + 1, actual.Orden + 2}}
		p.Configuracion.Revision = cfg.Gobierno.Revision
		p.Configuracion.Secuencia = cfg.Gobierno.Secuencia
		p.Configuracion.Huella = cfg.Gobierno.HuellaSHA256
		p.Configuracion.PublicadaEn = cfg.Gobierno.PublicadaEn
		p.Configuracion.ExpiraEn = cfg.Gobierno.ExpiraEn
		data, _ := json.Marshal(p)
		escribir("plan.json", data)
		h := sha256.Sum256(data)
		mh := sha256.Sum256(secreto.Bytes())
		escribir("aprobacion-candidata.json", []byte(`{"plan_sha256":"`+hex.EncodeToString(h[:])+`","material_sha256":"`+hex.EncodeToString(mh[:])+`","preimagen_sha256":"`+actual.PreSHA+`"}`))
		return
	}
	plan, err := leerFicheroMaterialSeguro(filepath.Join(f.Salida, "plan.json"), 16384)
	if err != nil {
		t.Fatal("ensayo_plan_ausente")
	}
	h := sha256.Sum256(plan)
	original, err := leerFicheroMaterialSeguro(filepath.Join(f.Salida, "material.json"), 16384)
	defer borrarBytes(original)
	if err != nil || !bytes.Equal(original, secreto.Bytes()) {
		t.Fatal("ensayo_material_reintento_distinto")
	}
	operador, err := pgxpool.New(ctx, f.DSNOperador)
	if err != nil {
		t.Fatal("ensayo_login_ausente")
	}
	defer operador.Close()
	acuse, err := AplicarGobiernoUsuariosAdmin(ctx, operador, string(plan), hex.EncodeToString(h[:]), m)
	if err != nil {
		t.Fatal("ensayo_aplicacion_sin_acuse_confirmado")
	}
	data, _ := json.Marshal(acuse)
	escribir("acuse-"+f.Fase+".json", data)
	if acuse.Estado != "permitido" {
		t.Fatalf("ensayo_resultado_%s", acuse.Codigo)
	}
}
