package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type continuidadCategoriaRPTPrueba struct {
	vinculo              vinculoContinuidadCategoriaRPT
	expediente           ports.ExpedienteParaSeleccion
	uso                  puertosvec.ResultadoUsoCategoriaRPT
	publicacion          puertosvec.ResultadoPublicacionCategoriaRPT
	consultasUso         []puertosvec.ConsultaUsoCategoriaRPT
	consultasPublicacion []puertosvec.ConsultaPublicacionCategoriaRPT
	ordenes              int
	errorFuente          error
	errorUso             error
	errorPublicacion     error
	cambiarConsulta      bool
	reutilizarDecision   bool
	alterarEvidencia     bool
}

func (f *continuidadCategoriaRPTPrueba) ResolverVinculoContinuidadCategoriaRPT(_ context.Context, expediente ports.ExpedienteParaSeleccion) (vinculoContinuidadCategoriaRPT, error) {
	if expediente.Fiscalizado.Referencia != f.expediente.Fiscalizado.Referencia {
		return vinculoContinuidadCategoriaRPT{}, puertosvec.ErrLecturaRPTNoConfiable
	}
	return f.vinculo, f.errorFuente
}

// Material nominal de prueba del contrato. No acredita criptografía ni
// consumos PostgreSQL; esas comprobaciones pertenecen al adaptador común.
func materialContinuidadCategoriaRPTPrueba(accion string, contador int) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rpt:"+strconv.Itoa(contador), h, h, "contexto:rpt:prueba", h,
		accion, "rpt.categorias", h, "vec_catalogos_configurables.lectura_categorias.v1", ahora, ahora.Add(3*time.Second))
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), []byte("c"), 1, 1,
		[]byte("p"), []byte("s"), []byte("e"), raiz)
}

func (f *continuidadCategoriaRPTPrueba) AutorizarConsultaUsoCategoriaRPT(_ context.Context, q puertosvec.ConsultaUsoCategoriaRPT) (puertosvec.OrdenUsoCategoriaRPT, error) {
	f.ordenes++
	m, err := materialContinuidadCategoriaRPTPrueba("vec.catalogos.categorias.consultar_uso", f.ordenes)
	if f.cambiarConsulta {
		q.Consumidor = "personal"
	}
	return puertosvec.OrdenUsoCategoriaRPT{Consulta: q, Autorizacion: m}, err
}

func (f *continuidadCategoriaRPTPrueba) AutorizarPublicacionCategoriaRPT(_ context.Context, q puertosvec.ConsultaPublicacionCategoriaRPT) (puertosvec.OrdenPublicacionCategoriaRPT, error) {
	f.ordenes++
	contador := f.ordenes
	if f.reutilizarDecision {
		contador--
	}
	m, err := materialContinuidadCategoriaRPTPrueba("vec.catalogos.categorias.consultar_historica", contador)
	if f.cambiarConsulta {
		q.Referencia.Version++
	}
	return puertosvec.OrdenPublicacionCategoriaRPT{Consulta: q, Autorizacion: m}, err
}

func (f *continuidadCategoriaRPTPrueba) ListarCategoriasHabilitadasRPT(context.Context, puertosvec.OrdenCategoriasHabilitadasRPT) (puertosvec.ResultadoCategoriasHabilitadasRPT, error) {
	return puertosvec.ResultadoCategoriasHabilitadasRPT{}, errors.New("la continuidad no consulta habilitadas")
}

func evidenciaContinuidadCategoriaRPTPrueba(m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) puertosvec.EvidenciaLecturaRPT {
	r := m.ResumenCapacidad()
	return puertosvec.EvidenciaLecturaRPT{DecisionRef: r.DecisionRef(), EfectoRef: r.EfectoRef(),
		HuellaEfectoSHA256: r.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("b", 64),
		AuditoriaRef: "auditoria:" + r.DecisionRef(), ConsumidaEn: r.EmitidaEn(), ConsumoNuevo: true}
}

func (f *continuidadCategoriaRPTPrueba) ConsultarUsoCategoriaRPT(_ context.Context, o puertosvec.OrdenUsoCategoriaRPT) (puertosvec.ResultadoUsoCategoriaRPT, error) {
	f.consultasUso = append(f.consultasUso, o.Consulta)
	r := f.uso
	r.Evidencia = evidenciaContinuidadCategoriaRPTPrueba(o.Autorizacion)
	if f.alterarEvidencia {
		r.Evidencia.ConsumoNuevo = false
	}
	return r, f.errorUso
}

func (f *continuidadCategoriaRPTPrueba) LeerPublicacionCategoriaRPT(_ context.Context, o puertosvec.OrdenPublicacionCategoriaRPT) (puertosvec.ResultadoPublicacionCategoriaRPT, error) {
	f.consultasPublicacion = append(f.consultasPublicacion, o.Consulta)
	r := f.publicacion
	r.Evidencia = evidenciaContinuidadCategoriaRPTPrueba(o.Autorizacion)
	return r, f.errorPublicacion
}

func escenarioContinuidadCategoriaRPTPrueba(t *testing.T) (*consumidorContinuidadCategoriaRPT, *continuidadCategoriaRPTPrueba) {
	t.Helper()
	ctx, _, _ := escenarioRevisionManualPrueba(t)
	p := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	p.expediente.Fiscalizado = p.expediente.Fiscalizado.Clonar()
	p.expediente.Fiscalizado.Analisis.CategoriaRef = "categoria:rpt:administrativo"
	p.expediente.Fiscalizado.Analisis.GrupoSubgrupo = "C1"
	if p.expediente.Fiscalizado.Validar() != nil {
		t.Fatal("expediente de prueba inválido")
	}
	// Documento nominal: la validación canónica se prueba en el lector D.
	documento := `{"categoria":"categoria:rpt:administrativo","version":1}`
	suma := sha256.Sum256([]byte(documento))
	referencia := puertosvec.ReferenciaPublicacionRPT{CatalogoID: "rpt.categorias", Version: 1, HuellaSHA256: hex.EncodeToString(suma[:])}
	fecha := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	f := &continuidadCategoriaRPTPrueba{expediente: p.expediente,
		vinculo: vinculoContinuidadCategoriaRPT{organizacionRef: p.expediente.Fiscalizado.OrganizacionRef,
			expedienteRef: p.expediente.Fiscalizado.Referencia, versionExpediente: p.expediente.Fiscalizado.Version, categoriaRef: p.expediente.Fiscalizado.Analisis.CategoriaRef,
			publicacion: referencia, usoRef: "uso:ct:prueba", reservaReciboRef: "recibo:reserva:prueba"},
		uso: puertosvec.ResultadoUsoCategoriaRPT{Encontrado: true, Uso: &puertosvec.UsoCategoriaRPT{
			Consumidor: "contratacion_temporal", UsoRef: "uso:ct:prueba", CategoriaID: p.expediente.Fiscalizado.Analisis.CategoriaRef,
			Publicacion: referencia, Estado: "reservado", Revision: 1, ReservaReciboRef: "recibo:reserva:prueba", ReservadoEn: fecha}},
		publicacion: puertosvec.ResultadoPublicacionCategoriaRPT{Encontrado: true,
			Publicacion:   &puertosvec.PublicacionRPT{Referencia: referencia, DocumentoCanonico: documento, PublicadaEn: fecha},
			Entrada:       &dominiovec.EntradaCatalogoConfigurable{Clave: p.expediente.Fiscalizado.Analisis.CategoriaRef},
			ControlActual: &puertosvec.ControlCategoriaRPT{Publicacion: puertosvec.ReferenciaPublicacionRPT{CatalogoID: referencia.CatalogoID, Version: 2, HuellaSHA256: strings.Repeat("c", 64)}, Revision: 3, Estado: "deshabilitada"}},
	}
	c, err := nuevoConsumidorContinuidadCategoriaRPT(puertosvec.DescriptorCatalogoRPT{CatalogoID: referencia.CatalogoID, ModuloID: "organizacion"}, f, f, f)
	if err != nil {
		t.Fatal(err)
	}
	return c, f
}

func TestContinuidadCategoriaRPTConservaPublicacionAunqueEsteDeshabilitada(t *testing.T) {
	c, f := escenarioContinuidadCategoriaRPTPrueba(t)
	for i := 0; i < 2; i++ {
		categoria, err := c.validar(context.Background(), f.expediente)
		if err != nil || categoria != f.vinculo.categoriaRef {
			t.Fatalf("reserva histórica rechazada: %q %v", categoria, err)
		}
	}
	if len(f.consultasUso) != 2 || len(f.consultasPublicacion) != 2 || f.ordenes != 4 ||
		f.consultasPublicacion[0].Referencia.Version != 1 || f.consultasUso[0].Consumidor != "contratacion_temporal" {
		t.Fatal("no conservó historia y lecturas frescas por petición")
	}
	terminal := "recibo:confirmacion:prueba"
	fecha := f.uso.Uso.ReservadoEn.Add(time.Minute)
	f.uso.Uso.Estado, f.uso.Uso.Revision = "confirmado", 2
	f.uso.Uso.TerminalReciboRef, f.uso.Uso.TerminalEn = &terminal, &fecha
	if _, err := c.validar(context.Background(), f.expediente); err != nil {
		t.Fatal("uso confirmado no recuperable", err)
	}
}

func TestContinuidadCategoriaRPTRechazaCrucesAusenciaYCategoriaAislada(t *testing.T) {
	casos := map[string]func(*continuidadCategoriaRPTPrueba){
		"vinculo_ausente":       func(f *continuidadCategoriaRPTPrueba) { f.vinculo = vinculoContinuidadCategoriaRPT{} },
		"expediente_ajeno":      func(f *continuidadCategoriaRPTPrueba) { f.vinculo.expedienteRef = "expediente:ajeno" },
		"organizacion_ajena":    func(f *continuidadCategoriaRPTPrueba) { f.vinculo.organizacionRef = "organizacion:ajena" },
		"categoria_otra":        func(f *continuidadCategoriaRPTPrueba) { f.vinculo.categoriaRef = "categoria:desarrollo:c2" },
		"publicacion_inventada": func(f *continuidadCategoriaRPTPrueba) { f.vinculo.publicacion.Version = 2 },
		"catalogo_ajeno":        func(f *continuidadCategoriaRPTPrueba) { f.vinculo.publicacion.CatalogoID = "otro.catalogo" },
		"uso_ausente":           func(f *continuidadCategoriaRPTPrueba) { f.uso.Encontrado = false; f.uso.Uso = nil },
		"uso_personal":          func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.Consumidor = "personal" },
		"otro_uso":              func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.UsoRef = "uso:ajeno" },
		"otro_recibo_reserva":   func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.ReservaReciboRef = "recibo:ajeno" },
		"uso_otra_categoria":    func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.CategoriaID = "categoria:rpt:auxiliar" },
		"uso_cancelado":         func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.Estado = "cancelado" },
		"uso_sin_revision":      func(f *continuidadCategoriaRPTPrueba) { f.uso.Uso.Revision = 0 },
		"publicacion_ausente": func(f *continuidadCategoriaRPTPrueba) {
			f.publicacion.Encontrado = false
			f.publicacion.Publicacion = nil
		},
		"entrada_ajena":          func(f *continuidadCategoriaRPTPrueba) { f.publicacion.Entrada.Clave = "categoria:rpt:auxiliar" },
		"documento_alterado":     func(f *continuidadCategoriaRPTPrueba) { f.publicacion.Publicacion.DocumentoCanonico += " " },
		"decision_reutilizada":   func(f *continuidadCategoriaRPTPrueba) { f.reutilizarDecision = true },
		"consulta_sustituida":    func(f *continuidadCategoriaRPTPrueba) { f.cambiarConsulta = true },
		"auditoria_no_consumida": func(f *continuidadCategoriaRPTPrueba) { f.alterarEvidencia = true },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			c, f := escenarioContinuidadCategoriaRPTPrueba(t)
			alterar(f)
			categoria, err := c.validar(context.Background(), f.expediente)
			if err == nil || categoria != "" {
				t.Fatalf("aceptó material desligado: %q %v", categoria, err)
			}
		})
	}
}

func TestContinuidadCategoriaRPTNoSuplantaErroresNiContinuaSinDependencias(t *testing.T) {
	c, f := escenarioContinuidadCategoriaRPTPrueba(t)
	f.errorFuente = puertosvec.ErrLecturaRPTNoDisponible
	if _, err := c.validar(context.Background(), f.expediente); !errors.Is(err, f.errorFuente) || f.ordenes != 0 {
		t.Fatalf("fuente ausente reconstruida: %v", err)
	}
	f.errorFuente = nil
	f.errorUso = puertosvec.ErrLecturaRPTDenegada
	if _, err := c.validar(context.Background(), f.expediente); !errors.Is(err, f.errorUso) || len(f.consultasPublicacion) != 0 {
		t.Fatalf("lectura denegada habilitó histórica: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.validar(ctx, f.expediente); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida", err)
	}
	var lector *continuidadCategoriaRPTPrueba
	if _, err := nuevoConsumidorContinuidadCategoriaRPT(c.descriptor, lector, f, f); !errors.Is(err, puertosvec.ErrLecturaRPTNoDisponible) {
		t.Fatal("dependencia nula admitida", err)
	}
}
