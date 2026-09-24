package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func cambioCatalogoPrueba(t *testing.T) domain.SolicitudCambioCatalogoEmpleadoB2 {
	t.Helper()
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	s := domain.SolicitudCambioCatalogoEmpleadoB2{
		Operacion: "publicar", OrganismoRef: "organismo:dipgra", Tipo: "regimen", Ref: "regimen:uno",
		Version: 1, Revision: 1, Denominacion: "Régimen sintético", VigenteDesde: fecha,
		ActoRef: "acto:publicacion", IdempotenciaRef: "11111111-1111-4111-8111-111111111111", Actor: solicitudP(t).Actor,
	}
	s.HuellaSHA256 = domain.HuellaPublicacionCatalogoEmpleadoB2(s)
	return s
}

func TestCatalogoEmpleadoMaterialLigaOrganismoVersionYContenido(t *testing.T) {
	s := cambioCatalogoPrueba(t)
	m, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	if m.Recurso().Referencia != "organismo:dipgra:regimen:regimen:uno:1" ||
		m.Recurso().Ambitos["organismo_ref"] != s.OrganismoRef ||
		!bytes.Contains(m.Canonico(), []byte(`"huella_sha256":"`+s.HuellaSHA256+`"`)) {
		t.Fatal("material sin objetivo nominal")
	}
	primeraHuella, _ := m.HuellaSHA256()
	otro := s
	otro.OrganismoRef = "organismo:otro"
	otro.HuellaSHA256 = domain.HuellaPublicacionCatalogoEmpleadoB2(otro)
	mOtro, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(otro)
	if err != nil {
		t.Fatal(err)
	}
	otraHuella, _ := mOtro.HuellaSHA256()
	if primeraHuella == otraHuella {
		t.Fatal("organismo no ligado a V3")
	}
	alterado := s
	alterado.Denominacion = "Otro régimen"
	if _, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(alterado); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("publicación sin huella del contenido")
	}
	sinOrganismo := s
	sinOrganismo.OrganismoRef = ""
	if _, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(sinOrganismo); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("organismo implícito")
	}
}

func TestCatalogoEmpleadoConsultaExigeFiltroNominalYPaginacionCoherente(t *testing.T) {
	s := domain.SolicitudConsultaCatalogoEmpleadoB2{OrganismoRef: "organismo:dipgra", Tipo: "situacion", Limite: 50, Actor: solicitudP(t).Actor}
	m, err := domain.NuevoMaterialConsultaCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	if m.Recurso().Referencia != "organismo:dipgra:situacion" || m.Recurso().Ambitos["organismo_ref"] != s.OrganismoRef ||
		!bytes.Contains(m.Canonico(), []byte(`"estado":null`)) || !bytes.Contains(m.Canonico(), []byte(`"cursor_ref":null`)) {
		t.Fatal("filtro V3 sin organismo, tipo o nulos explícitos")
	}
	s.CursorRef = "situacion:uno"
	if _, err := domain.NuevoMaterialConsultaCatalogoEmpleadoB2(s); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("cursor incompleto")
	}
	s.CursorVersion = 1
	if _, err := domain.NuevoMaterialConsultaCatalogoEmpleadoB2(s); err != nil {
		t.Fatal(err)
	}
	s.Limite = 101
	if _, err := domain.NuevoMaterialConsultaCatalogoEmpleadoB2(s); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) {
		t.Fatal("límite superior")
	}
}

type autorizadorCatalogoPrueba struct {
	a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	n int
}

func (p *autorizadorCatalogoPrueba) AutorizarCatalogoRegistroEmpleadoB2(_ context.Context, _ domain.MaterialCatalogoEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.n++
	return p.a, nil
}

type repoCatalogoPrueba struct {
	resultado ports.ResultadoCambioCatalogoEmpleadoB2
	n         int
}

func (p *repoCatalogoPrueba) ConsultarRRHH(context.Context, ports.OrdenCatalogoEmpleadoB2) (ports.ResultadoConsultaCatalogoEmpleadoB2, error) {
	return ports.ResultadoConsultaCatalogoEmpleadoB2{}, nil
}
func (p *repoCatalogoPrueba) CambiarRRHH(_ context.Context, _ ports.OrdenCatalogoEmpleadoB2) (ports.ResultadoCambioCatalogoEmpleadoB2, error) {
	p.n++
	return p.resultado, nil
}

func exportacionCatalogoPrueba(t *testing.T, m domain.MaterialCatalogoEmpleadoB2, operacion, efecto string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	huella, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	accion, audiencia := accionAudienciaCatalogoEmpleadoB2(operacion)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, efecto, huella, audiencia,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 20, 10, 0, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actor := m.Actor()
	ctx, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	x, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen,
		[]byte("d"), []byte("m"), ctx, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion,
		[]byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestCatalogoEmpleadoNoLlamaRepositorioConEfectoAjeno(t *testing.T) {
	s := cambioCatalogoPrueba(t)
	m, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	a := &autorizadorCatalogoPrueba{a: exportacionCatalogoPrueba(t, m, "publicar", "organismo:otro:regimen:regimen:uno:1")}
	repo := &repoCatalogoPrueba{}
	servicio, err := NuevoServicioCatalogosRegistroEmpleadoB2(a, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado) || repo.n != 0 {
		t.Fatalf("efecto ajeno llegó a SQL: %v, llamadas=%d", err, repo.n)
	}
}

func TestCatalogoEmpleadoRechazaRespuestaDeOtroOrganismo(t *testing.T) {
	s := cambioCatalogoPrueba(t)
	m, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	a := &autorizadorCatalogoPrueba{a: exportacionCatalogoPrueba(t, m, "publicar", m.Recurso().Referencia)}
	instante := time.Date(2026, 9, 20, 10, 0, 1, 0, time.UTC)
	repo := &repoCatalogoPrueba{resultado: ports.ResultadoCambioCatalogoEmpleadoB2{
		Entrada: domain.EntradaCatalogoRegistroEmpleadoB2{OrganismoRef: "organismo:otro", Tipo: s.Tipo, Ref: s.Ref,
			Version: s.Version, Revision: s.Revision, Denominacion: s.Denominacion, HuellaSHA256: s.HuellaSHA256,
			VigenteDesde: s.VigenteDesde, Estado: "publicada"},
		Recibo:       ports.ReciboCatalogoEmpleadoB2{DecisionRef: "dec_primera", AuditoriaRef: "aud_primera", ConsumoHuellaSHA256: strings.Repeat("d", 64), RegistradoEn: instante},
		AccesoActual: ports.AccesoActualCatalogoEmpleadoB2{DecisionRef: "dec_prueba", AuditoriaRef: "aud_actual", ConsumoHuellaSHA256: strings.Repeat("e", 64), RegistradoEn: instante, EstadoReplay: "registrado"},
	}}
	servicio, _ := NuevoServicioCatalogosRegistroEmpleadoB2(a, repo)
	if _, err := servicio.Cambiar(context.Background(), s); !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatalf("filtración de entrada ajena: %v", err)
	}
}

func TestCatalogoEmpleadoAdmiteReplayConReciboOriginal(t *testing.T) {
	s := cambioCatalogoPrueba(t)
	m, err := domain.NuevoMaterialCambioCatalogoEmpleadoB2(s)
	if err != nil {
		t.Fatal(err)
	}
	a := &autorizadorCatalogoPrueba{a: exportacionCatalogoPrueba(t, m, "publicar", m.Recurso().Referencia)}
	primerInstante := time.Date(2026, 9, 20, 9, 59, 0, 0, time.UTC)
	instanteActual := time.Date(2026, 9, 20, 10, 0, 1, 0, time.UTC)
	repo := &repoCatalogoPrueba{resultado: ports.ResultadoCambioCatalogoEmpleadoB2{
		Entrada: domain.EntradaCatalogoRegistroEmpleadoB2{OrganismoRef: s.OrganismoRef, Tipo: s.Tipo, Ref: s.Ref,
			Version: s.Version, Revision: s.Revision, Denominacion: s.Denominacion, HuellaSHA256: s.HuellaSHA256,
			VigenteDesde: s.VigenteDesde, Estado: "publicada"},
		Recibo:       ports.ReciboCatalogoEmpleadoB2{DecisionRef: "dec_original", AuditoriaRef: "aud_original", ConsumoHuellaSHA256: strings.Repeat("d", 64), RegistradoEn: primerInstante},
		AccesoActual: ports.AccesoActualCatalogoEmpleadoB2{DecisionRef: "dec_prueba", AuditoriaRef: "aud_actual", ConsumoHuellaSHA256: strings.Repeat("e", 64), RegistradoEn: instanteActual, EstadoReplay: "replay"},
	}}
	servicio, _ := NuevoServicioCatalogosRegistroEmpleadoB2(a, repo)
	resultado, err := servicio.Cambiar(context.Background(), s)
	if err != nil || repo.n != 1 || resultado.Recibo.DecisionRef != "dec_original" || resultado.AccesoActual.DecisionRef != "dec_prueba" {
		t.Fatalf("replay con recibo original: resultado=%+v, err=%v", resultado, err)
	}
}
