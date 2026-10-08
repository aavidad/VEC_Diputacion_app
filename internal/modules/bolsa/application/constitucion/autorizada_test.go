package constitucion

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type autorizadoPrueba struct {
	guardada ports.Constitucion
	lote     importacion.LoteValidado
	vinculos []ports.VinculoCandidato
	err      error
}

func (a *autorizadoPrueba) ConfirmarCargaConvocaAutorizada(_ context.Context, lote importacion.LoteValidado, c ports.Constitucion, vinculos []ports.VinculoCandidato, _ ports.OriginalProtegidoCargaConvoca, _ puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	a.guardada, a.lote, a.vinculos = c, lote, vinculos
	if a.err != nil {
		return ports.ReciboCargaConvoca{}, a.err
	}
	var r ports.ReciboCargaConvoca
	r.ActaRef, r.BolsaRef, r.AuditoriaRef = c.ActaRef, c.Bolsa.BolsaRef, "aud_v3_0123456789abcdef0123456789abcdef"
	return r, nil
}

func TestServicioAutorizadoExigeDependenciasYMaterial(t *testing.T) {
	if _, err := NuevoServicioAutorizado(nil, time.Now, &autorizadoPrueba{}); !errors.Is(err, ErrDependenciasRequeridas) {
		t.Fatalf("sin base: %v", err)
	}
	if _, err := NuevoServicioAutorizado(derivadorPrueba(t), time.Now, nil); !errors.Is(err, ErrDependenciasRequeridas) {
		t.Fatalf("sin repositorio autorizado: %v", err)
	}
	s, err := NuevoServicioAutorizado(derivadorPrueba(t), time.Now, &autorizadoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Constituir(context.Background(), importacion.LoteValidado{}, Solicitud{HuellaFicheroSHA256: "h", CategoriaRef: "c", ActorRef: "a"}, ports.OriginalProtegidoCargaConvoca{}, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaInvalida) {
		t.Fatalf("material vacío admitido: %v", err)
	}
}

func TestServicioAutorizadoConstituyeConElMismoOrdenYRegistraVinculos(t *testing.T) {
	contenido, err := os.ReadFile("../testdata/carga_convoca/carga_convoca_ejemplo.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	preparador, err := importacionapp.NuevoPreparador(xlsconvoca.NuevoLector(), func() time.Time { return time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	lote, err := preparador.PrepararLote(context.Background(), importacionapp.SolicitudImportacion{
		CategoriaRef: "categoria:rpt:administrativo", NombreFichero: "carga_convoca_ejemplo.xlsx",
		FicheroCustodiadoRef: "original:convoca:" + strings.Repeat("a", 64), ActorRef: "actor:rrhh:prueba", Contenido: contenido,
	})
	if err != nil {
		t.Fatal(err)
	}
	original := ports.OriginalProtegidoCargaConvoca{Referencia: lote.Acta.FicheroCustodiadoRef}
	autorizado := &autorizadoPrueba{}
	s, err := NuevoServicioAutorizado(derivadorPrueba(t), func() time.Time { return time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC) }, autorizado)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := s.Constituir(context.Background(), lote, Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "per_0123456789abcdefghijkl"}, original, materialPruebaConstitucion(t))
	if err != nil {
		t.Fatal(err)
	}
	if autorizado.guardada.ActorRef != "per_0123456789abcdefghijkl" || len(autorizado.guardada.Entradas) != 11 || autorizado.guardada.Entradas[0].FilaNumero != 2 {
		t.Fatalf("constitución inesperada: actor=%q entradas=%d primera=%d", autorizado.guardada.ActorRef, len(autorizado.guardada.Entradas), autorizado.guardada.Entradas[0].FilaNumero)
	}
	if autorizado.lote.Acta.BolsaRef == "" || autorizado.lote.Acta.BolsaRef != autorizado.guardada.Bolsa.BolsaRef ||
		autorizado.lote.Acta.ActaRef != autorizado.guardada.ActaRef {
		t.Fatalf("acta y constitución divergen: acta=%q bolsa_acta=%q bolsa_constitucion=%q", autorizado.lote.Acta.ActaRef, autorizado.lote.Acta.BolsaRef, autorizado.guardada.Bolsa.BolsaRef)
	}
	if recibo.AuditoriaRef == "" || len(autorizado.vinculos) != 9 {
		t.Fatalf("recibo o vínculos inesperados: %+v %+v", recibo, autorizado.vinculos)
	}
	autorizado.err = dominiovec.ErrAutorizacionDenegada
	if _, err := s.Constituir(context.Background(), lote, Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "per_0123456789abcdefghijkl"}, original, materialPruebaConstitucion(t)); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación: err=%v", err)
	}
	autorizado.err = nil
	autorizado.guardada = ports.Constitucion{}
	lote.Acta.BolsaRef = "bolsa:otra:referencia"
	if _, err := s.Constituir(context.Background(), lote, Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "per_0123456789abcdefghijkl"}, original, materialPruebaConstitucion(t)); !errors.Is(err, ports.ErrConstitucionBolsaInvalida) || autorizado.guardada.ActaRef != "" {
		t.Fatalf("referencia ajena alcanzó el repositorio: %v", err)
	}
}

// materialPruebaConstitucion es un material estructuralmente válido: el
// servicio no lo interpreta, solo lo entrega al repositorio que lo consume.
func materialPruebaConstitucion(t *testing.T) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	c, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	r, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64), c.RegistroContextoRef, c.HuellaSHA256,
		ports.AccionConfirmarCargaConvoca, "acta:importacion-convoca:"+strings.Repeat("ab", 32), strings.Repeat("e", 64), ports.AudienciaConfirmarCargaConvoca, ahora.Add(-time.Microsecond), ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), r, []byte("d"), []byte("m"), c.RepresentacionCanonica,
		c.Contexto.Instantanea.PersonaVersion, c.Contexto.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
