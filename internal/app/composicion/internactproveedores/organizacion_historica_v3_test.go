package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Todos los materiales, actores, claves y autoridades de estos tests son
// sintéticos. El registro en memoria no acredita PostgreSQL ni consumo P10.
type fuenteOHPrueba struct {
	valor             ct.ContextoAutorizacionAltaV3
	organismo, unidad string
	err               error
	llamadas          int
}

func (f *fuenteOHPrueba) ContextoVinculadoOrganizacionHistorica(context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	f.llamadas++
	return f.valor, f.organismo, f.unidad, f.err
}

func (f *fuenteOHPrueba) ContextoOriginalOrganizacionHistoricaParaAuditoria(context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	f.llamadas++
	return f.valor, f.organismo, f.unidad, f.err
}

type revalidadorOHPrueba struct {
	valor core.AutenticacionRevalidadaV1
}

func (r revalidadorOHPrueba) RevalidarAutenticacionActorV1(context.Context, core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	return r.valor, nil
}

type resolutorOHPrueba struct {
	valor core.ResultadoContextoActorRegistradoV2
}

func (r resolutorOHPrueba) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return r.valor, nil
}

func contextoOHPrueba(t *testing.T, ahora time.Time) ct.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}
	i := core.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3, CuentaRef: cuenta.CuentaRef, CuentaVersion: 4, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	a, err := core.NuevoContextoActor(cuenta, i, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := a.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := a.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	ac := core.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("4", 64), ProcedenciaAutoridad: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := core.ManifiestoProcedenciaContextoActorV1{Esquema: core.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:   core.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: i.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Persona:  core.ProcedenciaPersonaContextoActorV1{PersonaRef: i.PersonaRef, Version: i.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Perfil:   core.ProcedenciaPerfilContextoActorV1{PerfilRef: i.PerfilActivoRef, Version: i.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Contexto: core.ProcedenciaVinculoContextoActorV1{VinculoRef: i.VinculoRef, Version: i.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []core.ProcedenciaVinculoReferenciaContextoActorV1{}}
	cm, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	hm, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(cm)
	if err != nil {
		t.Fatal(err)
	}
	r := core.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: a, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: cm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: a.ResueltoEn}
	auth := core.AutenticacionRevalidadaV1{AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64), AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2, ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: core.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia, PoliticaGarantiaRef: "pga_0123456789abcdefghijkl", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64), AutenticacionVerificadaEn: ahora.Add(-10 * time.Minute), SesionEmitidaEn: ahora.Add(-9 * time.Minute), SesionRevalidadaEn: ahora.Add(-3 * time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute)}
	v, err := core.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorOHPrueba{auth}, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorOHPrueba{r}, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: i.PerfilActivoRef}, &relojB2{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: v, Resultado: r}
}

func inventarioOHPrueba(t *testing.T, desde time.Time) (MaterialOrganizacionHistorica, string, map[string]any) {
	t.Helper()
	dir, err := os.MkdirTemp("/var/tmp", "vec-organizacion-historica-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	secreto := bytes.Repeat([]byte{0x51}, 32)
	if err := os.WriteFile(filepath.Join(dir, "historica.key"), secreto, 0600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(secreto)
	doc := map[string]any{"version": 1, "catalogo_motivos": "motivos.b2", "motivo_consulta": map[string]any{"catalogo_id": "motivos.b2", "catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("a", 64), "entrada_clave": "motivo_0123456789abcdef0123456789abcdef"},
		"capacidad": map[string]any{"archivo": "historica.key", "sha256": hex.EncodeToString(h[:]), "clave_id": "clave:capacidad:personal:historica:1", "version": 1, "emisor_id": "emisor:personal:historica:prueba", "desde": desde.Add(-time.Hour), "hasta": desde.Add(96 * time.Hour), "revision_gobierno": 1, "huella_gobierno": strings.Repeat("b", 64)},
		"contextos": map[string]any{"cta_0123456789abcdefghijkl": map[string]any{"perfil_activo_ref": "prf_0123456789abcdefghijkl", "perfil_version": 5, "organismo_ref": "organizacion:prueba", "unidad_clave": "unidad:prueba"}}}
	escribirInventarioOH(t, dir, doc)
	m, err := CargarMaterialOrganizacionHistorica(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Cerrar() })
	return m, dir, doc
}
func escribirInventarioOH(t *testing.T, dir string, doc map[string]any) {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "organizacion_historica_v3.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMaterialOrganizacionHistoricaPrivadoYEstricto(t *testing.T) {
	_, dir, doc := inventarioOHPrueba(t, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC))
	for _, caso := range []struct {
		nombre    string
		modificar func(map[string]any)
	}{
		{"version", func(d map[string]any) { d["version"] = 2 }},
		{"raiz_propia", func(d map[string]any) { d["raiz"] = map[string]any{} }},
		{"motivo_ajeno", func(d map[string]any) { d["catalogo_motivos"] = "catalogo.ajeno" }},
		{"contextos_vacios", func(d map[string]any) { d["contextos"] = map[string]any{} }},
		{"perfil_sin_version", func(d map[string]any) {
			d["contextos"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["perfil_version"] = 0
		}},
		{"scope_invalido", func(d map[string]any) {
			d["contextos"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["organismo_ref"] = "*"
		}},
		{"archivo_escape", func(d map[string]any) { d["capacidad"].(map[string]any)["archivo"] = "../clave" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			b, _ := json.Marshal(doc)
			var clon map[string]any
			if err := json.Unmarshal(b, &clon); err != nil {
				t.Fatal(err)
			}
			caso.modificar(clon)
			escribirInventarioOH(t, dir, clon)
			if _, err := CargarMaterialOrganizacionHistorica(dir); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
				t.Fatalf("inventario admitido: %v", err)
			}
		})
	}
	ruta := filepath.Join(dir, "organizacion_historica_v3.json")
	for _, contenido := range []string{`{"version":1,"version":1}`, `{"version":1} {}`} {
		if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := CargarMaterialOrganizacionHistorica(dir); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
			t.Fatalf("JSON inseguro: %v", err)
		}
	}
	escribirInventarioOH(t, dir, doc)
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialOrganizacionHistorica(dir); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("permisos abiertos: %v", err)
	}
	if _, err := CargarMaterialOrganizacionHistorica("."); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("ruta relativa: %v", err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialOrganizacionHistorica(dir); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("directorio abierto: %v", err)
	}
}

func TestMaterialOrganizacionHistoricaNoExponeConfiguracion(t *testing.T) {
	m := MaterialOrganizacionHistorica{CatalogoMotivos: "catalogo_privado_prueba", Contextos: map[string]internagobierno.AmbitoOrganizacionHistorica{"cuenta_privada": {OrganismoRef: "organismo_privado"}}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{m.String(), m.GoString(), fmt.Sprintf("%+v %#v", m, m), m.LogValue().String(), string(b)} {
		if strings.Contains(s, "catalogo_privado") || strings.Contains(s, "cuenta_privada") || strings.Contains(s, "organismo_privado") {
			t.Fatal("configuracion privada expuesta")
		}
	}
}

type pdpOHPrueba struct {
	t           *testing.T
	ahora       time.Time
	solicitudes []core.SolicitudAutorizacionLigadaV3
	resultados  []core.ResultadoContextoActorRegistradoV2
	denegar     bool
}

func (p *pdpOHPrueba) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.t.Helper()
	p.solicitudes = append(p.solicitudes, s)
	p.resultados = append(p.resultados, r)
	datos, err := s.Datos()
	if err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	v := core.VersionRol{RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH", Estado: core.EstadoVersionRolPublicada, Concesiones: []core.ConcesionRol{{Accion: personal.AccionConsultaOrganizacionHistorica, ModuloID: "personal", TipoRecurso: "organizacion_historica", Finalidades: []string{"consultar_organizacion_historica"}, GarantiaMinima: core.AuthAssuranceSubstantial}}, PublicadaPor: "responsable-seguridad", PublicadaEn: p.ahora.Add(-24 * time.Hour)}
	if p.denegar {
		v.Concesiones[0].Accion = "personal.organizacion_historica.otra"
	}
	h, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		p.t.Fatal(err)
	}
	instantanea := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-rrhh", Version: 1, PerfilActivoRef: r.Contexto.PerfilActivoRef, PrincipalID: r.Contexto.PersonaRef, VersionRolRef: v.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "organismo_ref", Valores: []string{datos.Recurso.Ambitos["organismo_ref"]}}, {Clave: "unidad_clave", Valores: []string{datos.Recurso.Ambitos["unidad_clave"]}}}, VigenteDesde: p.ahora.Add(-time.Hour), VigenteHasta: p.ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: p.ahora.Add(-2 * time.Hour)}, VersionRol: v, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: v.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: v.PublicadaPor, ActualizadoEn: v.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: h}
	e, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, instantanea, "dec_0123456789abcdef0123456789abcdef", p.ahora, p.ahora.Add(90*time.Second))
	if err != nil {
		p.t.Fatal(err)
	}
	d, err := core.NuevaDecisionAutorizacionLigadaV3(s, e)
	if err != nil {
		p.t.Fatal(err)
	}
	concedida, _, err := d.Resultado()
	if err != nil {
		p.t.Fatal(err)
	}
	if !concedida {
		return d, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errors.Join(core.ErrAutorizacionDenegada, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, r)
	if err != nil {
		p.t.Fatal(err)
	}
	c, err := vecports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroOHPrueba{p.ahora}, orden)
	return d, c, err
}

type registroOHPrueba struct{ ahora time.Time }

func (r registroOHPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.ahora, nil
}

func escenarioOHPrueba(t *testing.T) (*ProveedorAutorizacionOrganizacionHistorica, *fuenteOHPrueba, *pdpOHPrueba, *escenarioB2, MaterialOrganizacionHistorica) {
	t.Helper()
	dia := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	raiz, clave := raizB2(t, 8, dia)
	coord, err := coordenadasRaizV3("clave:atestacion:prueba-b2:v3", 1, audienciaAtestacionCTInterna, clave.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	e := &escenarioB2{dia: dia, reloj: &relojB2{ahora: dia.Add(time.Hour)}, raiz: raiz, coord: coord, pubA: publicacionB2(t, raiz, dia, 20260925)}
	e.firmante = &firmanteV3{claveID: coord.ClaveID, audiencia: coord.AudienciaDespliegue, privada: clave, reloj: e.reloj}
	e.gobierno = &gobiernoB2Falso{raiz: coord, actual: e.pubA, revocadas: map[string]bool{}}
	e.base = e.baseCT(t, e.pubA)
	m, _, _ := inventarioOHPrueba(t, e.dia)
	f := &fuenteOHPrueba{valor: contextoOHPrueba(t, e.reloj.Ahora()), organismo: "organizacion:prueba", unidad: "unidad:prueba"}
	pdp := &pdpOHPrueba{t: t, ahora: e.reloj.Ahora()}
	p, err := construirOrganizacionHistorica(context.Background(), m, pdp, m.CatalogoMotivos, e.base, e.firmante, f, e.reloj)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Cerrar)
	return p, f, pdp, e, m
}
func consultaOHPrueba(t *testing.T, f *fuenteOHPrueba, organismo, unidad string) personal.MaterialConsultaOrganizacionHistorica {
	t.Helper()
	m, err := personal.NuevoMaterialConsultaOrganizacionHistorica(personal.SolicitudConsultaOrganizacionHistorica{Actor: f.valor.Resultado.Contexto, Selector: personal.SelectorOrganizacionHistorica{OrganismoRef: organismo, UnidadClave: unidad, VigenteEn: personal.FechaCivil("2026-09-25"), ConocidoEn: f.valor.Resultado.Contexto.ResueltoEn, Limite: 10}})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestOrganizacionHistoricaEmiteV3ConRaizComunYAmbitoExacto(t *testing.T) {
	p, f, pdp, e, m := escenarioOHPrueba(t)
	if p.firmante.base != e.firmante {
		t.Fatal("raiz duplicada")
	}
	antes := e.gobierno.numeroLecturas()
	material := consultaOHPrueba(t, f, f.organismo, f.unidad)
	exp, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	if exp.ValidarEstructura() != nil {
		t.Fatal("exportacion invalida")
	}
	if f.llamadas != 1 || len(pdp.solicitudes) != 1 {
		t.Fatal("fuente o PDP no invocados exactamente una vez")
	}
	datos, err := pdp.solicitudes[0].Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.Accion != personal.AccionConsultaOrganizacionHistorica || datos.Finalidad != "consultar_organizacion_historica" || datos.ReferenciaMotivo != m.MotivoConsulta || !reflect.DeepEqual(datos.Recurso, material.Recurso()) || !reflect.DeepEqual(pdp.resultados[0], f.valor.Resultado) {
		t.Fatal("solicitud nominal o ligadura alterada")
	}
	capacidad := exp.CapacidadCanonica()
	var c map[string]any
	if err := json.Unmarshal(capacidad, &c); err != nil {
		t.Fatal(err)
	}
	if c["audiencia_consumo"] != personal.AudienciaConsultaOrganizacionHistorica || c["clave_id"] != m.Capacidad.ClaveID {
		t.Fatal("capacidad o audiencia ajena")
	}
	if e.gobierno.numeroLecturas() != antes {
		t.Fatal("OH ejecutó preflight B2")
	}
	// El material original no puede alterar el perfil que conservó el emisor.
	m.Contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef] = internagobierno.AmbitoOrganizacionHistorica{}
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material); err != nil {
		t.Fatalf("mapa compartido: %v", err)
	}
	p.Cerrar()
	if len(e.firmante.privada) == 0 || e.base.lector == nil {
		t.Fatal("cierre destruyó autoridad común")
	}
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("proveedor cerrado: %v", err)
	}
}

func TestOrganizacionHistoricaDeniegaActorPerfilAmbitoOFuenteAlterados(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*ProveedorAutorizacionOrganizacionHistorica, *fuenteOHPrueba)
	}{
		{"fuente_caida", func(_ *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) {
			f.err = errors.New("fuente de prueba")
		}},
		{"actor_ajeno", func(_ *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) {
			f.valor.Resultado.Contexto.ResueltoEn = f.valor.Resultado.Contexto.ResueltoEn.Add(time.Microsecond)
		}},
		{"perfil_ajeno", func(p *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) {
			a := p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef]
			a.PerfilActivoRef = "prf_abcdefghijkl0123456789"
			p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef] = a
		}},
		{"perfil_version", func(p *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) {
			a := p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef]
			a.PerfilVersion++
			p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef] = a
		}},
		{"organismo_sello", func(_ *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) {
			f.organismo = "organizacion:otra"
		}},
		{"unidad_sello", func(_ *ProveedorAutorizacionOrganizacionHistorica, f *fuenteOHPrueba) { f.unidad = "unidad:otra" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p, f, pdp, _, _ := escenarioOHPrueba(t)
			material := consultaOHPrueba(t, f, f.organismo, f.unidad)
			caso.cambiar(p, f)
			if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
				t.Fatalf("alteracion admitida: %v", err)
			}
			if len(pdp.solicitudes) != 0 {
				t.Fatal("PDP invocado antes de validar sello")
			}
		})
	}
	for _, scope := range [][2]string{{"organizacion:otra", "unidad:prueba"}, {"organizacion:prueba", "unidad:otra"}, {"organizacion:prueba", ""}} {
		t.Run(scope[0]+scope[1], func(t *testing.T) {
			p, f, pdp, _, _ := escenarioOHPrueba(t)
			if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), consultaOHPrueba(t, f, scope[0], scope[1])); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
				t.Fatalf("ambito ajeno admitido: %v", err)
			}
			if len(pdp.solicitudes) != 0 {
				t.Fatal("PDP invocado para ambito ajeno")
			}
		})
	}
}

func TestOrganizacionHistoricaScopeOrganismoPermiteFiltroUnidadConPDP(t *testing.T) {
	p, f, pdp, _, _ := escenarioOHPrueba(t)
	f.unidad = ""
	a := p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef]
	a.UnidadClave = ""
	p.contextos[f.valor.Resultado.Contexto.Instantanea.CuentaRef] = a
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), consultaOHPrueba(t, f, f.organismo, "unidad:filtro")); err != nil {
		t.Fatal(err)
	}
	datos, err := pdp.solicitudes[0].Datos()
	if err != nil || datos.Recurso.Ambitos["unidad_clave"] != "unidad:filtro" {
		t.Fatal("filtro no ligado a decision")
	}
	pdp.denegar = true
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), consultaOHPrueba(t, f, f.organismo, "unidad:filtro")); !errors.Is(err, personal.ErrConsultaOrganizacionHistoricaDenegada) {
		t.Fatalf("scope privado concedió permiso: %v", err)
	}
}

func TestOrganizacionHistoricaConfianzaFrescaYFallaCerrado(t *testing.T) {
	p, f, pdp, e, m := escenarioOHPrueba(t)
	material := consultaOHPrueba(t, f, f.organismo, f.unidad)
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material); err != nil {
		t.Fatal(err)
	}
	e.gobierno.revocar(e.pubA.Revision)
	if _, err := p.AutorizarConsultaOrganizacionHistorica(context.Background(), material); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("revocacion común ignorada: %v", err)
	}
	if len(pdp.solicitudes) != 1 {
		t.Fatal("PDP invocado tras fallo de confianza")
	}
	if _, err := ConstruirOrganizacionHistorica(context.Background(), m, nil, nil, e.reloj); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("sin base: %v", err)
	}
	var nulo *ProveedorAutorizacionOrganizacionHistorica
	if _, err := nulo.AutorizarConsultaOrganizacionHistorica(context.Background(), material); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("nulo: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := p.AutorizarConsultaOrganizacionHistorica(cancelado, material); !errors.Is(err, ErrOrganizacionHistoricaV3NoDisponible) {
		t.Fatalf("cancelado: %v", err)
	}
}
