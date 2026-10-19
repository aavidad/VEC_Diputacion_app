package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type repositorioComprobacionPrueba struct {
	normalizaciones, escrituras int
	canon, huella               string
	intencion                   ports.IntencionComprobacion
	recibo                      ports.ReciboComprobacion
}

func (r *repositorioComprobacionPrueba) NormalizarComprobacion(_ context.Context, q ports.SolicitudComprobacion) (string, string, error) {
	r.normalizaciones++
	if r.canon == "" {
		return "", "", nil
	}
	var original, normalizado any
	b, _ := json.Marshal(q)
	if json.Unmarshal(b, &original) != nil || json.Unmarshal([]byte(r.canon), &normalizado) != nil || !reflect.DeepEqual(original, normalizado) {
		return "", "", ports.ErrComprobacionInvalida
	}
	return r.canon, r.huella, nil
}
func (r *repositorioComprobacionPrueba) RegistrarComprobacion(_ context.Context, i ports.IntencionComprobacion) (ports.ReciboComprobacion, error) {
	r.escrituras++
	r.intencion = i
	return r.recibo, nil
}

type autorizadorComprobacionPrueba struct {
	t         *testing.T
	audiencia string
	ultima    vd.SolicitudAutorizacionLigadaV3
	decision  vd.DecisionAutorizacionLigadaV3
	material  vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err       error
}

func (a *autorizadorComprobacionPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s vd.SolicitudAutorizacionLigadaV3, c vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if a.err != nil {
		return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, a.err
	}
	datos := datosSolicitudBorradorPrueba(a.t, s)
	v, _ := datos.VinculoAutenticacionActor.Datos()
	ahora := time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC)
	rol := vd.VersionRol{RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico", Estado: vd.EstadoVersionRolPublicada, Concesiones: []vd.ConcesionRol{{Accion: datos.Accion, ModuloID: "bolsa", TipoRecurso: datos.Recurso.Tipo, Finalidades: []string{datos.Finalidad}, GarantiaMinima: vd.AuthAssuranceHigh, Obligaciones: []string{"auditar"}}}, PublicadaPor: "seguridad", PublicadaEn: ahora.Add(-time.Hour)}
	ambitos := []vd.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{datos.Recurso.Ambitos["unidad_ref"]}}, {Clave: "ambito_ref", Valores: []string{datos.Recurso.Ambitos["ambito_ref"]}}}
	h, _ := vd.HuellaCatalogoPoliticasAutorizacion(nil)
	i := vd.InstantaneaAutorizacion{AsignacionPerfil: vd.AsignacionPerfil{AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: v.PerfilActivoRef, PrincipalID: v.PrincipalID, VersionRolRef: rol.Referencia(), Estado: vd.EstadoAsignacionPerfilActiva, Ambitos: ambitos, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "seguridad", EmitidaEn: ahora.Add(-time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: vd.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: vd.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "seguridad", ActualizadoEn: ahora.Add(-time.Hour)}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: h}
	e, err := vd.NuevaEvidenciaEvaluacionAutorizacionV3(s, i, "decision:borrador:prueba", ahora, ahora.Add(time.Minute))
	if err != nil {
		a.t.Fatal(err)
	}
	d, err := vd.NuevaDecisionAutorizacionLigadaV3(s, e)
	if err != nil {
		a.t.Fatal(err)
	}
	o, err := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, c)
	if err != nil {
		a.t.Fatal(err)
	}
	confirmacion, err := vp.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroConcesionBorradorPrueba{instante: ahora}, o)
	if err != nil {
		a.t.Fatal(err)
	}
	a.ultima = s
	a.decision = d
	a.material = materialCargaPrueba(a.t, d, c, datos, ahora, a.audiencia)
	return d, confirmacion, exportadorBorradorPrueba{material: a.material}, nil
}

func solicitudComprobacionPrueba(t *testing.T) ports.SolicitudComprobacion {
	t.Helper()
	base := solicitudSituacionPrueba(t, time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC))
	return ports.SolicitudComprobacion{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion, Contexto: ports.ContextoComprobacion{AmbitoRef: "ambito:bolsa", BolsaRef: "bolsa:b2", CategoriaRef: "categoria:uno", CausaOfrecida: "sustitucion", ExpedienteRef: "expediente:uno", HuellaPoliticaOfertasSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModalidadOfrecida: "sustitucion", NecesidadRef: "necesidad:uno", OrdenActa: 1, OrganizacionRef: "organizacion:uno", ParticipacionRef: "participacion:b2", UnidadRef: "unidad:seleccion", VersionBolsa: 1, VersionExpediente: 1, VersionInstantanea: 1, VersionPoliticaOfertas: 1}, Hechos: map[string]ports.HechoComprobacion{}, ClaveIdempotencia: "clave:12345678"}
}

func TestComprobacionRegistraConDecisionV3YMaterialLigado(t *testing.T) {
	q := solicitudComprobacionPrueba(t)
	b, _ := json.Marshal(q)
	canon := string(b)
	suma := sha256.Sum256([]byte(canon))
	huella := hex.EncodeToString(suma[:])
	recibo := ports.ReciboComprobacion{ComprobacionRef: "comprobacion:uno", Version: 1}
	repo := &repositorioComprobacionPrueba{canon: canon, huella: huella, recibo: recibo}
	a := &autorizadorComprobacionPrueba{t: t, audiencia: ports.AudienciaComprobacionOfrecimiento}
	s, _ := NuevoServicioComprobacionOfrecimiento(contextoSituacionPrueba{}, a, repo)
	r, err := s.Registrar(context.Background(), q)
	if err != nil || r != recibo || repo.escrituras != 1 {
		t.Fatalf("recibo=%+v error=%v escrituras=%d", r, err, repo.escrituras)
	}
	datos, _ := a.ultima.Datos()
	restricciones, err := a.decision.RestriccionesProyeccionPara(a.ultima)
	if err != nil || !reflect.DeepEqual(restricciones.Obligaciones, []string{"auditar"}) {
		t.Fatalf("obligación de auditoría: %+v, %v", restricciones, err)
	}
	if datos.Accion != ports.AccionComprobacionOfrecimiento || datos.Finalidad != ports.FinalidadComprobacionOfrecimiento || datos.Recurso.Referencia != q.Contexto.ParticipacionRef || datos.Recurso.Ambitos["unidad_ref"] != q.Contexto.UnidadRef || datos.Recurso.Ambitos["ambito_ref"] != q.Contexto.AmbitoRef || datos.Recurso.Atributos["material_sha256"] != huella {
		t.Fatal("autorización desacoplada de operación o canon")
	}
	i := repo.intencion
	if !reflect.DeepEqual(i.Solicitud, q) || i.CanonPostgreSQL != canon || i.HuellaSHA256 != huella || i.Material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaComprobacionOfrecimiento || i.Material.ValidarEstructura() != nil {
		t.Fatal("intención o material desacoplados")
	}
	entregada, _ := i.Material.HuellaConjuntoSHA256()
	emitida, _ := a.material.HuellaConjuntoSHA256()
	if entregada != emitida {
		t.Fatal("material V3 cambiado antes del repositorio")
	}
}

func TestComprobacionCierraSinEscribirAnteDenegacionYMaterialAjeno(t *testing.T) {
	for _, caso := range []struct {
		nombre, audiencia string
		err               error
		esperado          error
	}{
		{"PDP", ports.AudienciaComprobacionOfrecimiento, vp.ErrDenegacionExplicitaAutorizacionLigadaV3, vd.ErrAutorizacionDenegada},
		{"audiencia", ports.AudienciaCrearBorradorLlamamientoInterno, nil, ports.ErrComprobacionDependencia},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := solicitudComprobacionPrueba(t)
			b, _ := json.Marshal(q)
			suma := sha256.Sum256(b)
			repo := &repositorioComprobacionPrueba{canon: string(b), huella: hex.EncodeToString(suma[:])}
			a := &autorizadorComprobacionPrueba{t: t, audiencia: caso.audiencia, err: caso.err}
			s, _ := NuevoServicioComprobacionOfrecimiento(contextoSituacionPrueba{}, a, repo)
			_, err := s.Registrar(context.Background(), q)
			if !errors.Is(err, caso.esperado) || repo.escrituras != 0 {
				t.Fatalf("error=%v escrituras=%d", err, repo.escrituras)
			}
		})
	}
}

func TestComprobacionRechazaCanonAjenoAntesDeAutorizar(t *testing.T) {
	q := solicitudComprobacionPrueba(t)
	repo := &repositorioComprobacionPrueba{canon: `{}`, huella: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	a := &autorizadorComprobacionPrueba{t: t, audiencia: ports.AudienciaComprobacionOfrecimiento}
	s, _ := NuevoServicioComprobacionOfrecimiento(contextoSituacionPrueba{}, a, repo)
	_, err := s.Registrar(context.Background(), q)
	_, errSolicitud := a.ultima.Datos()
	if !errors.Is(err, ports.ErrComprobacionInvalida) || repo.escrituras != 0 || errSolicitud == nil {
		t.Fatalf("canon ajeno: %v escrituras=%d", err, repo.escrituras)
	}
}

func TestComprobacionDeniegaContextoAjenoAntesDeNormalizar(t *testing.T) {
	ahora := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	base := solicitudSituacionPrueba(t, ahora)
	q := ports.SolicitudComprobacion{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion, Contexto: ports.ContextoComprobacion{AmbitoRef: "ambito:otro", BolsaRef: "bolsa:b2", CategoriaRef: "categoria:uno", CausaOfrecida: "sustitucion", ExpedienteRef: "expediente:uno", HuellaPoliticaOfertasSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ModalidadOfrecida: "sustitucion", NecesidadRef: "necesidad:uno", OrdenActa: 1, OrganizacionRef: "organizacion:uno", ParticipacionRef: "participacion:b2", UnidadRef: "unidad:seleccion", VersionBolsa: 1, VersionExpediente: 1, VersionInstantanea: 1, VersionPoliticaOfertas: 1}, Hechos: map[string]ports.HechoComprobacion{}, ClaveIdempotencia: "clave:12345678"}
	repo := &repositorioComprobacionPrueba{}
	s, _ := NuevoServicioComprobacionOfrecimiento(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo)
	_, err := s.Registrar(context.Background(), q)
	if !errors.Is(err, vd.ErrAutorizacionDenegada) || repo.normalizaciones != 0 {
		t.Fatalf("contexto ajeno: %v, normalizaciones=%d", err, repo.normalizaciones)
	}
}

func TestCopiaHechosComprobacion(t *testing.T) {
	original := map[string]ports.HechoComprobacion{"es_vacante": {Valor: []byte("true"), DocumentoRef: "ref:uno"}}
	copia := copiarHechosComprobacion(original)
	v := original["es_vacante"]
	v.Valor[0] = 'f'
	original["es_vacante"] = ports.HechoComprobacion{Valor: []byte("false")}
	if string(copia["es_vacante"].Valor) != "true" {
		t.Fatal("mapa compartido")
	}
}

func TestVectorContextoComprobacionV3(t *testing.T) {
	const esperado = "dc43219aa3ce118e97cc05226ee9cad24d222d18e58925f405c5004ceda3e007"
	r := vd.RecursoAutorizable{Referencia: "participacion:001", ModuloID: "bolsa", Tipo: "participacion_bolsa", Ambitos: map[string]string{"ambito_ref": "ambito:bolsa", "unidad_ref": "unidad:seleccion"}, Atributos: map[string]string{"material_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || h != esperado {
		t.Fatalf("huella V3: %s, %v", h, err)
	}
	r.Atributos["material_sha256"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if otra, _ := r.HuellaContextoAutorizacionSHA256(); otra == esperado {
		t.Fatal("cambio de evidencia conserva la huella")
	}
	r.Atributos["material_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	r.Ambitos["unidad_ref"] = "unidad:otra"
	if otra, _ := r.HuellaContextoAutorizacionSHA256(); otra == esperado {
		t.Fatal("cambio de contexto conserva la huella")
	}
}

// B141: la evidencia admite la referencia legible del documento en custodia
// o el evento de contrato de Bolsa; rechaza datos de identidad.
func TestComprobacionDocumentoRefLegible(t *testing.T) {
	huella := strings.Repeat("a", 64)
	evento := "evento:ct:contrato-bolsa:12345678a" + strings.Repeat("b", 55)
	for ref, valida := range map[string]bool{
		"registro/2026/0042": true, "ref:" + strings.Repeat("c", 64): true, evento: true,
		"dni:prueba": false, "documento:12345678Z": false, "ab": false, "con espacio": false,
		"otro:contrato-bolsa:12345678a" + strings.Repeat("b", 55): false,
	} {
		q := solicitudComprobacionPrueba(t)
		q.Hechos = map[string]ports.HechoComprobacion{"es_vacante": {Valor: []byte("true"), DocumentoRef: ref, Huella: huella}}
		if err := validarSolicitudComprobacion(q); (err == nil) != valida {
			t.Errorf("%q: valida=%v, error=%v", ref, valida, err)
		}
	}
}
