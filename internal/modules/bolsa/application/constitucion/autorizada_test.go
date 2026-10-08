package constitucion

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type autorizadoPrueba struct {
	guardada ports.Constitucion
	err      error
}

func (a *autorizadoPrueba) ConstituirCargaConvocaAutorizada(_ context.Context, c ports.Constitucion, _ puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	a.guardada = c
	if a.err != nil {
		return ports.ReciboCargaConvoca{}, a.err
	}
	var r ports.ReciboCargaConvoca
	r.ActaRef, r.BolsaRef, r.AuditoriaRef = c.ActaRef, c.Bolsa.BolsaRef, "aud_v3_0123456789abcdef0123456789abcdef"
	return r, nil
}

func TestServicioAutorizadoExigeDependenciasYMaterial(t *testing.T) {
	if _, err := NuevoServicioAutorizado(nil, &autorizadoPrueba{}); !errors.Is(err, ErrDependenciasRequeridas) {
		t.Fatalf("sin base: %v", err)
	}
	base, err := NuevoServicio(recuperadorPrueba{}, &repositorioPrueba{}, derivadorPrueba(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoServicioAutorizado(base, nil); !errors.Is(err, ErrDependenciasRequeridas) {
		t.Fatalf("sin repositorio autorizado: %v", err)
	}
	s, err := NuevoServicioAutorizado(base, &autorizadoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Constituir(context.Background(), Solicitud{HuellaFicheroSHA256: "h", CategoriaRef: "c", ActorRef: "a"}, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaInvalida) {
		t.Fatalf("material vacío admitido: %v", err)
	}
}

func TestServicioAutorizadoConstituyeConElMismoOrdenYRegistraVinculos(t *testing.T) {
	lote := importacion.LoteValidado{
		Acta: importacion.ActaImportacion{
			CategoriaRef: "categoria:rpt:administrativo", BolsaRef: "bolsa:administrativo:2026-10-05",
			ActaRef: "acta:importacion-convoca:" + strings.Repeat("ab", 32), ImportacionRef: "importacion:convoca:" + strings.Repeat("cd", 32),
			HuellaFicheroSHA256: strings.Repeat("ef", 32), Esquema: importacion.EsquemaResumenPersona,
		},
		Aceptadas: []importacion.FilaAceptada{
			fila(1, "***0001**", "Reyes", "Antonio", "10.5"),
			fila(2, "***0002**", "Moreno", "Lucía", "22.25"),
		},
	}
	repo, autorizado := &repositorioPrueba{}, &autorizadoPrueba{}
	base, err := NuevoServicio(recuperadorPrueba{lote}, repo, derivadorPrueba(t), func() time.Time { return time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	s, err := NuevoServicioAutorizado(base, autorizado)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := s.Constituir(context.Background(), Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "per_0123456789abcdefghijkl"}, materialPruebaConstitucion(t))
	if err != nil {
		t.Fatal(err)
	}
	if autorizado.guardada.ActorRef != "per_0123456789abcdefghijkl" || len(autorizado.guardada.Entradas) != 2 || autorizado.guardada.Entradas[0].FilaNumero != 2 {
		t.Fatalf("constitución inesperada: %+v", autorizado.guardada)
	}
	if repo.guardada.ActaRef != "" {
		t.Fatal("usó la constitución sin consumo")
	}
	if recibo.AuditoriaRef == "" || repo.actaRef != lote.Acta.ActaRef || len(repo.vinculos) != 2 {
		t.Fatalf("recibo o vínculos inesperados: %+v %+v", recibo, repo.vinculos)
	}
	autorizado.err = dominiovec.ErrAutorizacionDenegada
	repo.vinculos = nil
	if _, err := s.Constituir(context.Background(), Solicitud{HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256, CategoriaRef: lote.Acta.CategoriaRef, ActorRef: "per_0123456789abcdefghijkl"}, materialPruebaConstitucion(t)); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.vinculos != nil {
		t.Fatalf("denegación: err=%v vinculos=%v", err, repo.vinculos)
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
