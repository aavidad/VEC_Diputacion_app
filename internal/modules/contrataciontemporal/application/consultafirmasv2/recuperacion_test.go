package consultafirmasv2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func lecturaRecuperacionPrueba(t *testing.T) ports.LecturaRecuperacionFirmasV2 {
	t.Helper()
	base := lecturaPrueba()
	f, r := base.Firmas[0], base.RevisionesPDF[0]
	ref := func(d string, v uint64, h string) map[string]any {
		return map[string]any{"referencia": d, "version": v, "huella_sha256": h}
	}
	identidad := camposCanonPrueba("certificado_der_sha256", "persona_ref", "persona", "cuenta",
		"vinculo_cuenta_persona", "cuenta_persona_cuenta_ref", "cuenta_persona_persona_ref",
		"vinculo_certificado", "vinculo_cuenta_ref", "vinculo_persona_ref", "vinculo_der_sha256")
	identidad["persona_ref"] = "per_firmante_prueba"
	competencia := camposCanonPrueba("asignacion", "rol", "rol_id", "control_rol", "persona_ref",
		"perfil_esperado_ref", "perfil_activo_ref", "modulo_id", "tipo_recurso", "recurso_ref",
		"ambito_organizacion_ref", "ambito_unidad_ref", "asignacion_rol_ref", "control_rol_ref",
		"vigente_desde", "vigente_hasta")
	competencia["persona_ref"] = identidad["persona_ref"]
	competencia["modulo_id"] = ports.ModuloContratacion
	competencia["tipo_recurso"] = "documento"
	competencia["recurso_ref"] = f.OriginalRef
	competencia["ambito_organizacion_ref"] = "organizacion:central"
	competencia["ambito_unidad_ref"] = "unidad:prueba"
	relacion := camposCanonPrueba("expediente_ref", "unidad_ref", "origen_ref", "origen_version",
		"prueba_snapshot_sha256", "evento_ref", "evento_huella_sha256", "confirmada_en")
	relacion["expediente_ref"] = "expediente:prueba"
	relacion["unidad_ref"] = "unidad:prueba"
	canon := map[string]any{
		"esquema": "vec.competencia-firmante.historica.v1", "identidad": identidad,
		"competencia": competencia,
		"personal": camposCanonPrueba("cargo", "enlace_ocupante", "ocupante_persona_ref", "cargo_ref_enlace",
			"cargo_vigente_desde", "cargo_vigente_hasta", "enlace_vigente_desde", "enlace_vigente_hasta", "delegacion"),
		"relacion_ct": relacion,
		"accion":      ports.AccionRegistrarFirmaVec, "finalidad": ports.FinalidadFirmaDocumento,
		"motivo":          camposCanonPrueba("catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"),
		"fecha_historica": f.RegistradaEn.Format("2006-01-02T15:04:05Z"),
		"paso_ref":        f.PasoRef, "paso_orden": f.PasoOrden,
		"circuito": ref(f.CatalogoRef, 1, f.CatalogoHuella),
		"recurso": map[string]any{
			"organizacion_ref": "organizacion:central", "unidad_ref": "unidad:prueba",
			"expediente_ref": "expediente:prueba", "documento_ref": f.OriginalRef,
			"recurso_autorizable_ref": f.OriginalRef, "modulo_id": ports.ModuloContratacion,
			"tipo_recurso": "documento", "recurso_contexto_sha256": strings.Repeat("1", 64),
			"original":           ref(f.OriginalRef, f.OriginalVersion, f.OriginalHuella),
			"pdf_raiz_sha256":    f.OriginalHuella,
			"firmado":            ref(f.DocumentoCustodiaRef, f.DocumentoCustodiaVersion, f.FirmadoHuella),
			"pdf_firmado_sha256": f.FirmadoHuella, "numero_firmas": 1, "entrada_revision": nil,
		},
	}
	b, err := json.Marshal(canon)
	if err != nil || len(b) < 512 {
		t.Fatalf("fixture nominal: %v, %d bytes", err, len(b))
	}
	h := sha256.Sum256(b)
	return ports.LecturaRecuperacionFirmasV2{LecturaFirmasR5V2: base,
		Recuperaciones: []ports.RecuperacionFirmaV2{{FirmaRef: r.FirmaRef, MaterialRootSHA256: strings.Repeat("2", 64),
			CanonNominal: string(b), CanonNominalSHA256: hex.EncodeToString(h[:]),
			CanonNominalRef: "evidencia:competencia-firmante-ct:" + strings.Repeat("3", 64)}}}
}

func camposCanonPrueba(nombres ...string) map[string]any {
	m := make(map[string]any, len(nombres))
	for _, nombre := range nombres {
		m[nombre] = "ejemplo"
	}
	return m
}

func TestRecuperacionCruzaCanonHistoricoConFirmaYRevision(t *testing.T) {
	l := lecturaRecuperacionPrueba(t)
	m := materialRecuperacionPrueba()
	r, err := proyectarRecuperacion(m, l)
	if err != nil || len(r.Recuperaciones) != 1 || r.Recuperaciones[0].CanonNominal != l.Recuperaciones[0].CanonNominal {
		t.Fatalf("canon original perdido: %v", err)
	}
	for nombre, alterar := range map[string]func(*ports.LecturaRecuperacionFirmasV2){
		"root": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones[0].MaterialRootSHA256 = "invalida" },
		"huella": func(x *ports.LecturaRecuperacionFirmasV2) {
			x.Recuperaciones[0].CanonNominalSHA256 = strings.Repeat("0", 64)
		},
		"ausente": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones = nil },
		"duplicada": func(x *ports.LecturaRecuperacionFirmasV2) {
			x.Recuperaciones = append(x.Recuperaciones, x.Recuperaciones[0])
		},
		"firma ajena": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones[0].FirmaRef = "firma:ajena" },
		"documento": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, "recurso", "documento_ref", "ref:ajeno")
		},
		"pdf": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, "recurso", "pdf_raiz_sha256", strings.Repeat("0", 64))
		},
		"paso":    func(x *ports.LecturaRecuperacionFirmasV2) { mutarCanonPrueba(t, x, "", "paso_orden", 2) },
		"esquema": func(x *ports.LecturaRecuperacionFirmasV2) { mutarCanonPrueba(t, x, "", "esquema", "otro") },
		"clave duplicada": func(x *ports.LecturaRecuperacionFirmasV2) {
			v := &x.Recuperaciones[0]
			v.CanonNominal = strings.Replace(v.CanonNominal, `"paso_ref":`, `"paso_ref":"paso:ajeno","paso_ref":`, 1)
			h := sha256.Sum256([]byte(v.CanonNominal))
			v.CanonNominalSHA256 = hex.EncodeToString(h[:])
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			x := lecturaRecuperacionPrueba(t)
			alterar(&x)
			r, err := proyectarRecuperacion(m, x)
			if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(r.Firmas) != 0 || len(r.Recuperaciones) != 0 {
				t.Fatalf("material incoherente expuesto: %v", err)
			}
		})
	}
}

func mutarCanonPrueba(t *testing.T, l *ports.LecturaRecuperacionFirmasV2, objeto, clave string, valor any) {
	t.Helper()
	v := &l.Recuperaciones[0]
	var m map[string]any
	if err := json.Unmarshal([]byte(v.CanonNominal), &m); err != nil {
		t.Fatal(err)
	}
	if objeto == "" {
		m[clave] = valor
	} else {
		m[objeto].(map[string]any)[clave] = valor
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	v.CanonNominal = string(b)
	h := sha256.Sum256(b)
	v.CanonNominalSHA256 = hex.EncodeToString(h[:])
}

func materialRecuperacionPrueba() ports.MaterialConsultaFirmasR5V2 {
	q := solicitudPrueba()
	return ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: "organizacion:central", ExpedienteRef: q.ExpedienteRef, VersionExpediente: q.VersionExpediente,
		Documento: q.Documento, FirmantePrincipalCandidatoRef: "per_candidato_nominal", PasoOrden: q.PasoOrden,
		ClaveIdempotencia: q.ClaveIdempotencia, CatalogoHuella: q.CatalogoHuella}, Via: q.Via}
}

type autorizadorRecuperacionPrueba struct {
	t        *testing.T
	ajena    bool
	consulta bool
	campos44 bool
	llamado  int
}

func (a *autorizadorRecuperacionPrueba) AutorizarRecuperacionFirmasV2(_ context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadRecuperacionFirmasV2, error) {
	a.llamado++
	if a.ajena {
		m.ExpedienteRef = "expediente:ajeno"
	}
	base := &autorizadorPrueba{t: a.t}
	c, err := base.AutorizarConsultaFirmasR5V2(context.Background(), m)
	if err != nil {
		return ports.CapacidadRecuperacionFirmasV2{}, err
	}
	x := c.ExportarMaterialParaConsumidor()
	if a.consulta {
		return ports.TransportarMaterialRecuperacionFirmasV2(x, ports.CamposRecuperacionFirmasV2(), nil), nil
	}
	s := x.ResumenCapacidad()
	res, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(s.DecisionRef(), s.DecisionHuellaSHA256(),
		s.MotivoHuellaSHA256(), s.ContextoRef(), s.ContextoHuellaSHA256(), ports.AccionRecuperarFirmasR5V2,
		s.EfectoRef(), s.EfectoHuellaSHA256(), ports.AudienciaRecuperacionFirmasR5V2, s.EmitidaEn(), s.ExpiraEn())
	if err != nil {
		a.t.Fatal(err)
	}
	y, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), res,
		x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PersonaVersion(), x.PerfilVersion(),
		x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	if err != nil {
		a.t.Fatal(err)
	}
	campos := ports.CamposRecuperacionFirmasV2()
	if a.campos44 {
		campos = ports.CamposConsultaFirmasR5V2()
	}
	return ports.TransportarMaterialRecuperacionFirmasV2(y, campos, nil), nil
}

type lectorRecuperacionPrueba struct {
	l       ports.LecturaRecuperacionFirmasV2
	llamado int
	despues func()
}

func (l *lectorRecuperacionPrueba) RecuperarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadRecuperacionFirmasV2) (ports.LecturaRecuperacionFirmasV2, error) {
	l.llamado++
	if l.despues != nil {
		l.despues()
	}
	return l.l, nil
}

func TestServicioRecuperacionExigeCapacidadPropiaYNoExponeTrasCancelar(t *testing.T) {
	a := &autorizadorRecuperacionPrueba{t: t, ajena: true}
	l := &lectorRecuperacionPrueba{l: lecturaRecuperacionPrueba(t)}
	s, err := NuevaRecuperacion(&fuentePrueba{c: Contexto{"organizacion:central", "per_candidato_nominal"}}, a, l)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("capacidad ajena consumida: %v", err)
	}
	a.ajena = false
	a.consulta = true
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("acción de consulta antigua consumida: %v", err)
	}
	a.consulta = false
	a.campos44 = true
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("concesión de 44 campos consumida: %v", err)
	}
	a.campos44 = false
	ctx, cancelar := context.WithCancel(context.Background())
	l.despues = cancelar
	r, err := s.Recuperar(ctx, solicitudPrueba())
	if !errors.Is(err, context.Canceled) || !CubiertoPorLector(err) || len(r.Recuperaciones) != 0 {
		t.Fatalf("cancelación expone datos: %v", err)
	}
}
