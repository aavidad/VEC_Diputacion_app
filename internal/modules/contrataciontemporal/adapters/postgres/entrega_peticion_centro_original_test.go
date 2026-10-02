package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaEntregaOriginalPrueba struct{ entrega, original []byte }

func (f filaEntregaOriginalPrueba) Scan(destinos ...any) error {
	if len(destinos) != 2 {
		return errors.New("columnas de fachada distintas")
	}
	valores := [][]byte{f.entrega, f.original}
	for indice, destino := range destinos {
		p, ok := destino.(*[]byte)
		if !ok {
			return errors.New("tipo de columna distinto")
		}
		if valores[indice] == nil {
			*p = []byte(`{}`)
		} else {
			*p = append([]byte(nil), valores[indice]...)
		}
	}
	return nil
}

type proveedorEntregaOriginalPrueba struct {
	t                 *testing.T
	falloVerificacion error
	verificaciones    int
}

func (p *proveedorEntregaOriginalPrueba) VerificarOriginalAltaEntrega(_ context.Context, e ports.EntregaPeticionCentro, o ports.OriginalAltaEntrega) error {
	p.verificaciones++
	if e.ValidarReserva() != nil || o.Validar() != nil {
		p.t.Fatal("original o entrega de prueba inválidos")
	}
	return p.falloVerificacion
}

func (*proveedorEntregaOriginalPrueba) ActorEntregaPeticionCentro(context.Context) (string, string, error) {
	return "actor:rrhh:sintetico", "perfil:rrhh:sintetico", nil
}
func (*proveedorEntregaOriginalPrueba) ComprobarPerfilEntregaPeticionCentro(context.Context) error {
	return nil
}
func (*proveedorEntregaOriginalPrueba) RegistrarDenegacionEntregaPreV3(context.Context) error {
	return nil
}
func (*proveedorEntregaOriginalPrueba) NuevaClaveAltaDePeticion(context.Context) (string, string, error) {
	return "12345678-1234-4234-8234-123456789abc", "", nil
}
func (p *proveedorEntregaOriginalPrueba) AutorizarEntregaPeticionCentro(
	_ context.Context, material ports.MaterialEntregaPeticionCentro,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	recurso, err := RecursoEntregaPeticionCentro(material)
	if err != nil {
		p.t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		p.t.Fatal(err)
	}
	marco := strings.Repeat("a", 64)
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:entrega:original:001", marco, marco, "contexto:entrega:original", marco,
		AccionEntregaPeticionCentro(material), recurso.Referencia, huella,
		audienciaPeticionCentro, ahora, ahora.Add(5*time.Second),
	)
	if err != nil {
		p.t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		p.t.Fatal(err)
	}
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki,
	)
}

func TestFachadaOriginalRevierteAntesDelCommitSiFallaHMAC(t *testing.T) {
	material := ports.MaterialEntregaPeticionCentro{
		Modo: "preparar", ActorRef: "actor:rrhh:sintetico", PerfilRef: "perfil:rrhh:sintetico",
		PeticionRef: "peticion:centro:sintetica", CentroRef: "centro:sintetico",
		CategoriaRef: "categoria:sintetica", VersionEsperada: 2,
		ClaveAltaCandidata: "12345678-1234-4234-8234-123456789abc",
		AmbitoAltaHMAC:     "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64),
	}
	tx := &transaccionAltaCandidataPrueba{fila: filaEntregaOriginalPrueba{}}
	r := &RepositorioEntregasPeticionCentroPostgreSQL{
		pool:      &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}},
		proveedor: &proveedorEntregaOriginalPrueba{t: t},
	}
	err := r.ejecutarConOriginal(context.Background(), material, true, func(_, _ []byte) error {
		return ports.ErrClaveIdempotenciaUsada
	})
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) || tx.commits != 0 || tx.rollbacks != 1 ||
		!strings.Contains(tx.consulta, funcionPrepararEntregaConOriginal) {
		t.Fatalf("contraste original no revirtió autoenlace: error=%v commits=%d rollbacks=%d consulta=%q",
			err, tx.commits, tx.rollbacks, tx.consulta)
	}
}

func TestFachadaRevierteAutoenlaceHistoricoHastaContrastarHuella(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	material := ports.MaterialEntregaPeticionCentro{
		Modo: "preparar", ActorRef: "actor:rrhh:sintetico", PerfilRef: "perfil:rrhh:sintetico",
		PeticionRef: "peticion:centro:sintetica", CentroRef: "centro:sintetico",
		CategoriaRef: "categoria:sintetica", VersionEsperada: 2,
		ClaveAltaCandidata: "12345678-1234-4234-8234-123456789abc",
		AmbitoAltaHMAC:     "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64),
	}
	recibo := ports.ReciboAlta{ExpedienteRef: "expediente:ct:sintetico", NumeroVisible: "2026/CT-0001", Version: 1,
		ReciboRef: "recibo:ct:sintetico", AuditoriaRef: "auditoria:ct:sintetico", EventoRef: "evento:ct:sintetico", ConfirmadaEn: ahora}
	e := ports.EntregaPeticionCentro{
		EstadoEntrega: "confirmada", ClaveAlta: material.ClaveAltaCandidata,
		AmbitoAltaHMAC: material.AmbitoAltaHMAC, ActorRef: material.ActorRef,
		PerfilRef: "perfil:rrhh:historico", ReciboAlta: &recibo,
		Peticion: domain.DatosPeticionCentro{
			Referencia: material.PeticionRef, Version: 2, Estado: "ratificada",
			CreadaEn: ahora, RatificadaEn: ahora.Add(time.Second), MotivoRatificacion: "Revisión sintética",
			Configuracion: domain.ConfiguracionPeticionCentro{Referencia: "configuracion:centro:sintetica", Version: 1,
				Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:centro:solicita", PerfilRef: "perfil:centro:solicita", CentroRef: material.CentroRef, PuestoRef: "puesto:centro:solicita"},
				Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:centro:ratifica", PerfilRef: "perfil:centro:ratifica", CentroRef: material.CentroRef, PuestoRef: "puesto:centro:ratifica"}},
			Solicitud: domain.SolicitudCentro{CentroRef: material.CentroRef, ContactoRef: "contacto:centro:sintetico",
				CategoriaRef: material.CategoriaRef, GrupoSubgrupo: "C2", MotivoClave: "sustitucion",
				Detalle: "Necesidad temporal sintética", Periodo: domain.PeriodoPrevisto{
					Inicio: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
					Fin:    time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)}}},
	}
	if err := e.ValidarReserva(); err != nil {
		t.Fatalf("entrega de prueba inválida: %v", err)
	}
	original := ports.OriginalAltaEntrega{Esquema: "vec.contratacion-temporal.original-alta-entrega.v1",
		OrganizacionRef: "organizacion:desarrollo:dipgra", ActorRef: e.ActorRef, PerfilRef: e.PerfilRef,
		Flujo:              domain.ReferenciaFlujo{DefinicionRef: "flujo:ct:sintetico", Version: 1, HuellaSHA256: strings.Repeat("c", 64)},
		AmbitoHMAC:         e.AmbitoAltaHMAC,
		HuellaPeticionHMAC: "hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:" + strings.Repeat("b", 64),
		ReciboAlta:         recibo,
	}
	falso, verdadero := false, true
	salida, err := json.Marshal(resultadoEntregaPeticionCentroSQL{EntregaPeticionCentro: e,
		ReservaCreadaAhora: &falso, ConfirmacionCreadaAhora: &verdadero})
	if err != nil {
		t.Fatal(err)
	}
	contenidoOriginal, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	fallo := &transaccionAltaCandidataPrueba{fila: filaEntregaOriginalPrueba{entrega: salida, original: contenidoOriginal}}
	exito := &transaccionAltaCandidataPrueba{fila: filaEntregaOriginalPrueba{entrega: salida, original: contenidoOriginal}}
	proveedor := &proveedorEntregaOriginalPrueba{t: t, falloVerificacion: ports.ErrClaveIdempotenciaUsada}
	r := &RepositorioEntregasPeticionCentroPostgreSQL{pool: &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{fallo, exito}}, proveedor: proveedor}
	if _, err := r.entrega(context.Background(), material, true); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) ||
		fallo.commits != 0 || fallo.rollbacks != 1 || proveedor.verificaciones != 1 {
		t.Fatalf("autoenlace inválido comprometido: error=%v commits=%d rollbacks=%d verificaciones=%d", err, fallo.commits, fallo.rollbacks, proveedor.verificaciones)
	}
	proveedor.falloVerificacion = nil
	conservada, err := r.entrega(context.Background(), material, true)
	if err != nil || exito.commits != 1 || conservada.AltaAnterior == nil ||
		conservada.AltaAnterior.Recibo != recibo || proveedor.verificaciones != 2 {
		t.Fatalf("replay anterior no conservó recibo: error=%v commits=%d entrega=%+v", err, exito.commits, conservada)
	}
}
