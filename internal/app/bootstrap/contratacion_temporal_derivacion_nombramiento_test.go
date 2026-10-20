package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestRutaDerivacionOpcionalSinFuentesResponde503(t *testing.T) {
	h, cerrar, err := nuevaRutaDerivacionNombramientoOpcionalDesarrollo(
		config.Config{}, nil, nil, nil, catalogoFronterasComunDesarrollo{}, nil, relojContratacionTemporalDesarrollo{})
	if err != nil || h == nil || cerrar == nil {
		t.Fatalf("ruta opcional no compuesta: %v", err)
	}
	defer cerrar()
	e := ports.EntradaDerivacionNombramiento{PadreRef: "expediente:padre:dos-plazas", VersionPadre: 6,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1, SecuenciaPlaza: 1,
		ParticipacionRef: "participacion:uno", ReciboAdjudicacionRef: "recibo:adjudicacion:uno",
		ClaveIdempotencia: "938f47a6-5d2b-4c10-aa11-1234567890ab"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, httpinterno.RutaDerivacionNombramiento, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("ruta sin fuentes respondió %d: %s", w.Code, w.Body.String())
	}
}

func TestPerfilDerivacionNombramientoExigeAccionYAuditoriaPropias(t *testing.T) {
	p := politicaPerfilOfertaCTDesarrollo{accion: "contratacion_temporal.nombramiento.derivar",
		modulo: "contratacion_temporal", tipo: ports.TipoRecursoDerivacionNombramiento,
		finalidad: "gestionar_contratacion_temporal", obligaciones: []string{"auditar"}}
	c := core.ConcesionRol{Accion: p.accion, ModuloID: p.modulo, TipoRecurso: p.tipo,
		Finalidades: []string{p.finalidad}, GarantiaMinima: core.AuthAssuranceHigh,
		Obligaciones: []string{"auditar"}}
	if !concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("la concesión propia publicada debe poder consumirse")
	}
	c.Obligaciones = nil
	if concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("se aceptó derivar sin obligación de auditoría")
	}
	c.Obligaciones = []string{"auditar"}
	c.Accion = "contratacion_temporal.propuesta_formalizacion"
	if concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("se heredó el permiso de la propuesta")
	}
	// Los perfiles de oferta anteriores conservan su contrato de cero obligaciones.
	c = core.ConcesionRol{Accion: politicaVinculoOfertaCTDesarrollo().accion,
		ModuloID:       politicaVinculoOfertaCTDesarrollo().modulo,
		TipoRecurso:    politicaVinculoOfertaCTDesarrollo().tipo,
		Finalidades:    []string{politicaVinculoOfertaCTDesarrollo().finalidad},
		GarantiaMinima: core.AuthAssuranceHigh}
	if !concesionExactaPerfilOfertaCTDesarrollo(c, politicaVinculoOfertaCTDesarrollo()) {
		t.Fatal("cambió el perfil de oferta existente")
	}
}

func TestPerfilDerivacionNombramientoUsaFronteraNominalPropia(t *testing.T) {
	p := politicaPerfilOfertaCTDesarrollo{accion: string(domain.AccionDerivarNombramiento),
		ruta: httpinterno.RutaDerivacionNombramiento, claveFrontera: "ct-nombramiento-derivar"}
	c := perfilActivacionOfertaCT{PerfilRef: "prf:ct:nombramiento-derivado"}
	d, err := descriptoresPerfilOfertaCTDesarrollo(c, p, false)
	if err != nil || len(d) != 1 || d[0].Ruta != p.ruta || d[0].Clave != p.claveFrontera ||
		d[0].ClaveCapacidad != p.accion {
		t.Fatalf("frontera propia incorrecta: %v %+v", err, d)
	}
	p.claveFrontera = ""
	if _, err := descriptoresPerfilOfertaCTDesarrollo(c, p, false); err == nil {
		t.Fatal("configuración parcial heredó la ruta de oferta")
	}
}

type detalleDerivacionPrueba struct {
	t        *testing.T
	porRef   map[string]ports.DetalleExpedienteRRHH
	llamadas []string
}

func (d *detalleDerivacionPrueba) Consultar(ctx context.Context, q ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	d.t.Helper()
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	f, real := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || c.ruta != httpinterno.RutaConsultaDetalleRRHH || c.consultaRRHH == nil ||
		!real || f.descriptor.Clave != "ct-expediente-consultar" || q.VersionObservada() != 0 {
		d.t.Fatal("detalle sin suboperación de lectura nominal")
	}
	d.llamadas = append(d.llamadas, q.ExpedienteRef())
	v, existe := d.porRef[q.ExpedienteRef()]
	if !existe {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoObservable
	}
	return v, nil
}

func TestDerivacionPreparaConDetalleAutorizadoSinLectorRaw(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	catalogo := catalogoDetalleCTPrueba(t, alta.soporte)
	descriptor, existe := catalogo.resolver(http.MethodPost, httpinterno.RutaDerivacionNombramiento)
	if !existe {
		t.Fatal("frontera real de derivación ausente")
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaDerivacionNombramiento)
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.metodo, c.consultaRRHH = http.MethodPost, &contextoConsultaRRHHPeticionDesarrollo{}
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpinterno.RutaDerivacionNombramiento,
		superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: descriptor})
	e := ports.EntradaDerivacionNombramiento{PadreRef: "expediente:padre:dos-plazas", VersionPadre: 6,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1, SecuenciaPlaza: 1,
		ParticipacionRef: "participacion:uno", ReciboAdjudicacionRef: "recibo:adjudicacion:uno",
		ClaveIdempotencia: "938f47a6-5d2b-4c10-aa11-1234567890ab"}
	padre := detalleResolucionDesarrolloPrueba()
	padre.Resumen.ExpedienteRef, padre.Resumen.OrganizacionRef = e.PadreRef, organizacionAltaContratacionTemporalDesarrollo
	padre.Resumen.Version, padre.Resumen.FaseClave = 6, domain.FaseFiscalizacion
	padre.Resumen.EstadoClave = domain.EstadoEnCurso
	padre.Fiscalizacion = &ports.FiscalizacionOperativaRRHH{ResultadoClave: domain.FiscalizacionFavorable,
		RegistradaEn: time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)}
	detalle := &detalleDerivacionPrueba{t: t, porRef: map[string]ports.DetalleExpedienteRRHH{e.PadreRef: padre}}
	p := preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, fronteras: catalogo, detalle: detalle}
	s, err := p.PrepararDerivacionNombramiento(ctx, e)
	if err != nil || s.UnidadRef != padre.Asignacion.UnidadRef || s.PadreRef != e.PadreRef ||
		len(detalle.llamadas) != 1 || detalle.llamadas[0] != e.PadreRef {
		t.Fatalf("lectura nominal no preparó 12 campos: err=%v sol=%+v llamadas=%v", err, s, detalle.llamadas)
	}
	// Un circuito del padre no entra en esta proyección: el preparador carece
	// de lector raw y recibe solo el detalle nominal autorizado.
	padre.Resumen.Version = 7
	detalle.porRef[e.PadreRef] = padre
	hijoRef := ports.HijoRefDerivacionNombramiento(organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza)
	hijo := padre
	hijo.Resumen.ExpedienteRef, hijo.Resumen.Version = hijoRef, 7
	hijo.Asignacion = &ports.AsignacionOperativaRRHH{UnidadRef: "unidad:origen:derivacion", AsignadaEn: padre.Asignacion.AsignadaEn}
	hijo.Resumen.UnidadRef = hijo.Asignacion.UnidadRef
	detalle.porRef[hijoRef] = hijo
	detalle.llamadas = nil
	_, err = p.PrepararDerivacionNombramiento(ctx, e)
	if !errors.Is(err, ports.ErrDerivacionNombramientoNoDisponible) || len(detalle.llamadas) != 2 ||
		detalle.llamadas[1] != hijoRef {
		t.Fatalf("replay sin ámbito propio quedó abierto: err=%v llamadas=%v", err, detalle.llamadas)
	}
	delete(detalle.porRef, e.PadreRef)
	detalle.llamadas = nil
	_, err = p.PrepararDerivacionNombramiento(ctx, e)
	if !errors.Is(err, ports.ErrDerivacionNombramientoNoDisponible) || len(detalle.llamadas) != 2 ||
		detalle.llamadas[1] != hijoRef {
		t.Fatalf("hijo sin ámbito propio abrió replay tras denegación del padre: err=%v llamadas=%v", err, detalle.llamadas)
	}
	delete(detalle.porRef, hijoRef)
	detalle.llamadas = nil
	if _, err := p.PrepararDerivacionNombramiento(ctx, e); err == nil || len(detalle.llamadas) != 2 {
		t.Fatalf("hijo no observable se interpretó como ausencia: err=%v llamadas=%v", err, detalle.llamadas)
	}
	padre.Resumen.OrganizacionRef = "organizacion:ajena"
	detalle.porRef[e.PadreRef], detalle.porRef[hijoRef] = padre, hijo
	detalle.llamadas = nil
	if _, err := p.PrepararDerivacionNombramiento(ctx, e); !errors.Is(err, ports.ErrDerivacionNombramientoNoConfiable) || len(detalle.llamadas) != 1 {
		t.Fatalf("padre ajeno permitió usar la lectura del hijo: err=%v llamadas=%v", err, detalle.llamadas)
	}
	if _, err := (preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, fronteras: catalogo}).PrepararDerivacionNombramiento(ctx, e); !errors.Is(err, ports.ErrDerivacionNombramientoNoDisponible) {
		t.Fatal("lectura no compuesta no cerró la derivación", err)
	}
}

func TestReplayHijoV5SinReciboNominalQuedaCerrado(t *testing.T) {
	e := ports.EntradaDerivacionNombramiento{PadreRef: "expediente:padre:dos-plazas", VersionPadre: 6,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1, SecuenciaPlaza: 1,
		ParticipacionRef: "participacion:uno", ReciboAdjudicacionRef: "recibo:adjudicacion:uno",
		ClaveIdempotencia: "938f47a6-5d2b-4c10-aa11-1234567890ab"}
	hijoRef := ports.HijoRefDerivacionNombramiento(organizacionAltaContratacionTemporalDesarrollo, e.PadreRef, e.NumeroPlaza)
	hijo := detalleResolucionDesarrolloPrueba()
	hijo.Resumen.ExpedienteRef, hijo.Resumen.OrganizacionRef = hijoRef, organizacionAltaContratacionTemporalDesarrollo
	hijo.Resumen.Version = 6
	hijo.InstruccionCompartida = &ports.InstruccionCompartidaRRHH{
		DerivacionRef: "derivacion:prueba", PadreRef: e.PadreRef, OfertaRef: e.OfertaRef,
		NumeroPlaza: e.NumeroPlaza, ReciboDerivacionRef: "recibo:derivacion:prueba",
	}
	confiable, err := replayHijoDerivacionNombramientoConfiable(hijo, hijoRef, e)
	if err != nil {
		t.Fatal(err)
	}
	if confiable {
		t.Fatal("la proyección V5 sin recibo nominal acreditó un replay")
	}
}

func TestReplayHijoPropagaErrorDeConsulta(t *testing.T) {
	_, esperado := ports.NuevaSolicitudDetalleRRHH("", 0)
	confiable, err := replayHijoDerivacionNombramientoConfiable(
		ports.DetalleExpedienteRRHH{}, "", ports.EntradaDerivacionNombramiento{})
	if esperado == nil || confiable || !errors.Is(err, esperado) {
		t.Fatalf("error de consulta perdido: confiable=%v err=%v esperado=%v", confiable, err, esperado)
	}
}

func TestActivacionNombramientoPrivadaOpcionalYCerrada(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := configuracionOfertaCTPrueba(dir)
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); presente || err != nil {
		t.Fatalf("ausencia no opcional: presente=%v err=%v", presente, err)
	}
	c := archivoActivacionNombramientoCT{Esquema: esquemaActivacionNombramientoCTDesarrollo,
		Derivacion: perfilActivacionOfertaCT{PerfilRef: "prf:ct:nombramiento-derivado",
			Rol: referenciaRolOfertaCTDesarrollo{RolID: "ct_nombramiento_derivado", Version: 1,
				HuellaSHA256: strings.Repeat("a", 64)},
			Motivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_ct_nombramiento", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_nombramiento_derivar"}},
		FuenteAutorizacionDSNFile: "identidad/fuente.dsn", MotivosAutorizacionDSNFile: "identidad/motivos.dsn"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, archivoActivacionNombramientoCTDesarrollo)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); !presente || err != nil {
		t.Fatalf("activación válida rechazada: presente=%v err=%v", presente, err)
	}
	c.FuenteAutorizacionDSNFile = "../fuente.dsn"
	b, err = json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); !presente || err == nil {
		t.Fatalf("ruta fuera del árbol admitida: presente=%v err=%v", presente, err)
	}
}
