package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func contextoEconomicoPrueba(t *testing.T) vecdomain.ResultadoContextoActorRegistradoV2 {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	in := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, e := vecdomain.NuevoContextoActor(cuenta, in, ahora)
	if e != nil {
		t.Fatal(e)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: in.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, AcreditacionProcedenciaComponenteContextoActorV1: ac}}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if res.Validar() != nil {
		t.Fatal("resultado")
	}

	return res
}

func solicitudEconomicaPrueba(t *testing.T) ports.SolicitudLiquidacionEconomica {
	t.Helper()
	n := 0
	d := domain.DocumentoComision{GrupoDieta: 2, VersionTarifaAceptada: "tarifa:publicada:v1", TramosAceptados: []int{0},
		Lineas:              []domain.LineaDocumentoComision{{Tipo: "dieta", Grupo: 2, IndiceTramo: &n, Fecha: "2026-10-01", Concepto: "manutencion", ImporteCentimos: 500, VersionTarifaRef: "tarifa:publicada:v1", Rotulo: "dietas_tarifa_publicada"}},
		ManutencionCentimos: 500, TotalOrientativoCentimos: 500}
	dh, _ := domain.HuellaDatosLiquidacion(d)
	c := domain.CatalogoLiquidacionPublicado{Referencia: "catalogo:liquidacion:publicado", Version: "version:uno", PublicacionRef: "publicacion:acto:uno",
		FechaPublicacion: "2026-09-20", VigenteDesde: "2026-10-01", VigenteHasta: "2026-12-31", Ambito: "granada:ordinario",
		PaisISO2: "ES", Grupo: 2, Vehiculo: "sin_vehiculo_propio", VersionTarifaRef: "tarifa:publicada:v1", RotuloClave: "dietas_tarifa_publicada",
		Normas: []domain.NormaLiquidacionPublicada{{Referencia: "norma:bases", Articulo: "30", URL: "https://www.dipgra.es/bases"},
			{Referencia: "norma:boe", Articulo: "10", URL: "https://www.boe.es/boe"}},
		Reglas: []domain.ReglaLiquidacionPublicada{{ReglaLiquidacionPropuesta: domain.ReglaLiquidacionPropuesta{Referencia: "regla:manutencion:dos", Tipo: "dieta", Concepto: "manutencion", Grupo: 2, TopeCentimos: 1000}, NormaRef: "norma:bases"}}}
	c.ReglaDevengo = domain.ReglaDevengoPublicada{Referencia: "regla:devengo:uno", NormaRef: "norma:bases", Configuracion: domain.ConfiguracionDevengoPublicada{
		Esquema: "nacional_ordinaria_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5,
		HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14,
		HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, PorcentajeMismoDia: 50,
		PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50,
		PorcentajeIntermedio: 100, Alojamiento: "cerrado_pendiente_rrhh"}}
	c.ReglaDevengo.HuellaSHA256, _ = c.ReglaDevengo.HuellaCalculada()
	c.HuellaSHA256, _ = c.HuellaCalculada()
	p, err := domain.PrepararLiquidacionPublicada("dco_aaaaaaaaaaaaaaaaaaaaaa", 2, "2026-10-01", "08:00", "2026-10-01", "18:00", dh, d, c,
		[]domain.RevisionLineaLiquidacion{{Indice: 0, ReglaRef: "regla:manutencion:dos", ReconocidoPropuestoCentimos: 400, MotivoCodigo: "revision_importe"}})
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudLiquidacionEconomica{Decision: ports.SolicitudDecisionCircuito{Referencia: "dco_aaaaaaaaaaaaaaaaaaaaaa", UnidadRef: "unidad:rrhh", Etapa: domain.EtapaLiquidacion,
		Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_economia_20261014", VersionEsperada: 2}, Preparacion: p}
}

type repoEconomicoPrueba struct{ llamadas int }

func (r *repoEconomicoPrueba) Registrar(context.Context, ports.IdentidadEfectivaCircuito, ports.SolicitudLiquidacionEconomica) (domain.InstantaneaLiquidacionEconomica, error) {
	r.llamadas++
	return domain.InstantaneaLiquidacionEconomica{}, nil
}

type relojEconomicoPrueba struct{}

func (relojEconomicoPrueba) AhoraUTC() time.Time { return time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC) }

func TestLiquidacionEconomicaSinV3NoEscribe(t *testing.T) {
	r := &repoEconomicoPrueba{}
	s, err := NuevoServicioLiquidacionEconomica(r, relojEconomicoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.RegistrarPreparada(context.Background(), ports.IdentidadEfectivaCircuito{}, solicitudEconomicaPrueba(t))
	if !errors.Is(err, ports.ErrAccesoCircuitoDenegado) || r.llamadas != 0 {
		t.Fatalf("sin V3: %v %d", err, r.llamadas)
	}
	var nilRepo *repoEconomicoPrueba
	if _, err := NuevoServicioLiquidacionEconomica(nilRepo, relojEconomicoPrueba{}); err == nil {
		t.Fatal("nil admitido")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.RegistrarPreparada(ctx, ports.IdentidadEfectivaCircuito{}, solicitudEconomicaPrueba(t)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestMaterialEconomicoLigaRevisionYVersion(t *testing.T) {
	e := solicitudEconomicaPrueba(t)
	contexto := contextoEconomicoPrueba(t)
	uno, err := ConstruirEfectoAutorizacionLiquidacionEconomica(contexto, "unidad:rrhh", e)
	if err != nil {
		t.Fatal(err)
	}
	dos, err := ConstruirEfectoAutorizacionLiquidacionEconomica(contexto, "unidad:rrhh", e)
	if err != nil || string(uno.Material) != string(dos.Material) {
		t.Fatal("material inestable", err)
	}
	e.Decision.ClaveIdempotencia = "otra_clave_economia_20261014"
	otro, err := ConstruirEfectoAutorizacionLiquidacionEconomica(contexto, "unidad:rrhh", e)
	if err != nil || uno.Recurso.Atributos["material_sha256"] == otro.Recurso.Atributos["material_sha256"] {
		t.Fatal("clave sin ligadura", err)
	}
	e.Decision.VersionEsperada++
	if _, err := ConstruirEfectoAutorizacionLiquidacionEconomica(contexto, "unidad:rrhh", e); err == nil {
		t.Fatal("version no cotejada")
	}
	if uno.Recurso.Atributos["operacion"] != AccionRegistrarLiquidacionEconomica {
		t.Fatal("accion")
	}
}

func TestMaterialEconomicoTransportaPreimagenesVerificablesPorSQL(t *testing.T) {
	solicitud := solicitudEconomicaPrueba(t)
	efecto, err := ConstruirEfectoAutorizacionLiquidacionEconomica(contextoEconomicoPrueba(t), "unidad:rrhh", solicitud)
	if err != nil {
		t.Fatal(err)
	}
	// Deserializar reproduce el contrato bytea/base64 de PostgreSQL, sin volver
	// a serializar JSONB: su orden de claves no es el de los bytes atestados.
	var material struct {
		Economia          domain.InstantaneaLiquidacionPublicada `json:"economia"`
		PreimagenEconomia []byte                                 `json:"preimagen_economia"`
		PreimagenCatalogo []byte                                 `json:"preimagen_catalogo"`
		DocumentoCanonico []byte                                 `json:"documento_canonico"`
		SemanticaCanonica []byte                                 `json:"semantica_canonica"`
		HuellaSemantica   string                                 `json:"huella_semantica"`
	}
	if err := json.Unmarshal(efecto.Material, &material); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		bytes  []byte
		huella string
	}{
		{"economia", material.PreimagenEconomia, material.Economia.SnapshotSHA256},
		{"catalogo", material.PreimagenCatalogo, material.Economia.CatalogoSHA256},
		{"documento", material.DocumentoCanonico, material.Economia.DocumentoSHA256},
		{"semantica", material.SemanticaCanonica, material.HuellaSemantica},
	} {
		h := sha256.Sum256(caso.bytes)
		if !json.Valid(caso.bytes) || hex.EncodeToString(h[:]) != caso.huella {
			t.Fatalf("preimagen %s no acredita su huella", caso.nombre)
		}
	}
	var economia domain.InstantaneaLiquidacionPublicada
	if err := json.Unmarshal(material.PreimagenEconomia, &economia); err != nil || economia.SnapshotSHA256 != "" || economia.DocumentoSHA256 != material.Economia.DocumentoSHA256 {
		t.Fatal("preimagen económica incompleta", err)
	}
	var catalogo domain.CatalogoLiquidacionPublicado
	if err := json.Unmarshal(material.PreimagenCatalogo, &catalogo); err != nil || catalogo.HuellaSHA256 != "" || catalogo.Referencia != material.Economia.CatalogoRef {
		t.Fatal("preimagen del catálogo incompleta", err)
	}
	var semantica struct {
		PersonaRef string                                 `json:"persona_ref"`
		Comando    ports.SolicitudDecisionCircuito        `json:"comando"`
		Economia   domain.InstantaneaLiquidacionPublicada `json:"economia"`
	}
	if err := json.Unmarshal(material.SemanticaCanonica, &semantica); err != nil || semantica.Comando.ClaveIdempotencia != solicitud.Decision.ClaveIdempotencia || semantica.PersonaRef != contextoEconomicoPrueba(t).Contexto.PersonaRef || semantica.Economia.SnapshotSHA256 != material.Economia.SnapshotSHA256 {
		t.Fatal("semántica sin actor, clave o economía exacta", err)
	}
}
