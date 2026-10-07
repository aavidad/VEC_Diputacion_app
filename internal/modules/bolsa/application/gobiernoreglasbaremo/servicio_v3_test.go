package gobiernoreglasbaremo

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Dobles estructurales exclusivamente de aplicación: no firman ni acreditan
// consumo PostgreSQL. El consumidor nominal debe verificar todo V3 real.
type brokerGobiernoPrueba struct {
	credenciales CredencialesGobiernoV3
	ahora        *time.Time
	solicitudes  []ports.SolicitudMaterialGobiernoReglasV3
	denegar      bool
	mutar        string
}

func (b *brokerGobiernoPrueba) ProveerMaterialGobiernoReglasV3(_ context.Context, _ vd.VinculoAutenticacionActorV2, s ports.SolicitudMaterialGobiernoReglasV3) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	b.solicitudes = append(b.solicitudes, s)
	if b.denegar {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrGobiernoV3Prohibido
	}
	decision := []byte(fmt.Sprintf("decision_fixture_%d", len(b.solicitudes)))
	contexto, _ := b.credenciales.actor.RepresentacionCanonicaVinculadaV2()
	motivo, _ := motivoCanonicoGobiernoV3(s.Motivo)
	accion, ref, audiencia := s.Accion, s.Recurso.Referencia, s.Audiencia
	if b.mutar == "accion" {
		accion = "bolsa.reglas_baremo.publicar"
	}
	if b.mutar == "recurso" {
		ref = "reglas-baremo:" + strings.Repeat("f", 64)
	}
	if b.mutar == "audiencia" {
		audiencia = "audiencia:otra"
	}
	efecto, _ := s.Recurso.HuellaContextoAutorizacionSHA256()
	r, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(string(decision), shaGobiernoV3(decision), shaGobiernoV3(motivo), "rca_fixture", shaGobiernoV3(contexto), accion, ref, efecto, audiencia, *b.ahora, b.ahora.Add(3*time.Second))
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	persona, perfil := b.credenciales.actor.Instantanea.PersonaVersion, b.credenciales.actor.Instantanea.PerfilVersion
	if b.mutar == "perfil" {
		perfil++
	}
	if b.mutar == "contexto" {
		contexto = []byte("contexto:ajeno")
	}
	if b.mutar == "motivo" {
		motivo = []byte("motivo:ajeno")
	}
	return vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), r, decision, motivo, contexto, persona, perfil, []byte("payload_fixture"), []byte("sobre_fixture"), []byte("evidencia_fixture"), raiz)
}

type repositorioGobiernoPrueba struct {
	ahora                                *time.Time
	recibos                              map[string]ports.ReciboAltaBorradorReglasV3
	guardados, consultas, recuperaciones int
	mutar                                string
	ultimas                              []ports.OrdenAltaBorradorReglasV3
}

func evidenciaGobiernoPrueba(v vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) ports.EvidenciaAccesoGobiernoReglasV3 {
	r := v.ResumenCapacidad()
	return ports.EvidenciaAccesoGobiernoReglasV3{DecisionRef: r.DecisionRef(), DecisionHuellaSHA256: r.DecisionHuellaSHA256(), EfectoRef: r.EfectoRef(), EfectoHuellaSHA256: r.EfectoHuellaSHA256(), ConsumoHuellaSHA256: shaGobiernoV3([]byte(r.DecisionRef() + "consumo")), AuditoriaRef: "audit:" + r.DecisionRef(), ConsumidaEn: ahora}
}
func (r *repositorioGobiernoPrueba) ConfirmarAltaBorrador(_ context.Context, o ports.OrdenAltaBorradorReglasV3) (ports.ResultadoAltaBorradorReglasV3, error) {
	r.guardados++
	r.ultimas = append(r.ultimas, o)
	acceso := evidenciaGobiernoPrueba(o.Autorizacion, *r.ahora)
	if previo, existe := r.recibos[o.ClaveOperacion]; existe {
		if previo.HuellaSolicitudSHA256 != o.HuellaSolicitudSHA256 {
			return ports.ResultadoAltaBorradorReglasV3{}, ports.ErrClaveIdempotenciaReglasReutilizada
		}
		return ports.ResultadoAltaBorradorReglasV3{Recibo: previo, Acceso: acceso, Replay: true}, nil
	}
	for _, anterior := range r.recibos {
		if anterior.Estado.Contenido().Referencia() == o.EstadoPropuesto.Contenido().Referencia() && anterior.Estado.Contenido().Version() == o.EstadoPropuesto.Contenido().Version() {
			return ports.ResultadoAltaBorradorReglasV3{}, ports.ErrConflictoOCCReglasBaremo
		}
	}
	recibo := ports.ReciboAltaBorradorReglasV3{ReciboRef: "recibo:" + o.ClaveOperacion, ClaveOperacion: o.ClaveOperacion, HuellaSolicitudSHA256: o.HuellaSolicitudSHA256, Estado: o.EstadoPropuesto, VersionCanonica: bytes.Clone(o.VersionCanonica), TransaccionRef: "tx:" + o.ClaveOperacion, AuditoriaRef: acceso.AuditoriaRef, OutboxRef: "evento:" + o.ClaveOperacion, ConsumoOriginal: acceso, ConfirmadaEn: *r.ahora}
	r.recibos[o.ClaveOperacion] = recibo
	resultado := ports.ResultadoAltaBorradorReglasV3{Recibo: recibo, Acceso: acceso}
	switch r.mutar {
	case "sin_outbox":
		resultado.Recibo.OutboxRef = ""
	case "sin_consumo":
		resultado.Acceso = ports.EvidenciaAccesoGobiernoReglasV3{}
	case "otro_canon":
		resultado.Recibo.VersionCanonica = []byte("{}")
	case "otra_intencion":
		resultado.Recibo.HuellaSolicitudSHA256 = strings.Repeat("f", 64)
	}
	return resultado, nil
}
func (r *repositorioGobiernoPrueba) ObtenerExacta(_ context.Context, o ports.OrdenConsultaGobiernoReglasV3) (ports.ResultadoConsultaGobiernoReglasV3, error) {
	r.consultas++
	for _, recibo := range r.recibos {
		if recibo.Estado.CoincideExactamenteCon(o.Selector.Estado) {
			return ports.ResultadoConsultaGobiernoReglasV3{VersionCanonica: bytes.Clone(recibo.VersionCanonica), Acceso: evidenciaGobiernoPrueba(o.Autorizacion, *r.ahora)}, nil
		}
	}
	return ports.ResultadoConsultaGobiernoReglasV3{}, ports.ErrReglasBaremoNoEncontradas
}
func (r *repositorioGobiernoPrueba) RecuperarRecibo(_ context.Context, o ports.OrdenConsultaGobiernoReglasV3) (ports.ResultadoRecuperacionGobiernoReglasV3, error) {
	r.recuperaciones++
	recibo, existe := r.recibos[o.ClaveOperacion]
	return ports.ResultadoRecuperacionGobiernoReglasV3{Recibo: recibo, Existe: existe, Acceso: evidenciaGobiernoPrueba(o.Autorizacion, *r.ahora)}, nil
}

type revalidadorGobiernoPrueba struct{ aut vd.AutenticacionRevalidadaV1 }

func (r revalidadorGobiernoPrueba) RevalidarAutenticacionActorV1(context.Context, vd.SolicitudRevalidacionAutenticacionActorV1) (vd.AutenticacionRevalidadaV1, error) {
	return r.aut, nil
}

type resolutorGobiernoPrueba struct {
	resultado vd.ResultadoContextoActorRegistradoV2
}

func (r resolutorGobiernoPrueba) ResolverContextoActorRegistradoV2(context.Context, vd.SolicitudContextoActor) (vd.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojGobiernoPrueba struct{ ahora time.Time }

func (r relojGobiernoPrueba) Ahora() time.Time { return r.ahora }
func credencialesGobiernoPrueba(t *testing.T, ahora time.Time) CredencialesGobiernoV3 {
	t.Helper()
	z := strings.Repeat("a", 24)
	cuenta := vd.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}
	snap := vd.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: actorPlanPrueba, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vd.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vd.NuevoContextoActor(cuenta, snap, ahora)
	debeSinError(t, err)
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vd.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vd.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vd.ManifiestoProcedenciaContextoActorV1{Esquema: vd.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vd.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vd.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vd.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vd.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vd.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vd.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vd.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vd.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vd.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	aut := vd.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: vd.SuperficieAutenticacionInternaCorporativaV1, MetodoObservado: vd.AuthMethodCertificate, GarantiaObservada: vd.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Hour)}
	vinculo, err := vd.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorGobiernoPrueba{aut}, vd.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: aut.AutenticacionRef, SesionRef: aut.SesionRef}, resolutorGobiernoPrueba{res}, vd.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojGobiernoPrueba{ahora})
	debeSinError(t, err)
	c, err := NuevasCredencialesGobiernoV3(actor, vinculo)
	debeSinError(t, err)
	return c
}
func servicioGobiernoPrueba(t *testing.T) (*ServicioGobiernoV3, *brokerGobiernoPrueba, *repositorioGobiernoPrueba, CredencialesGobiernoV3, PeticionAltaBorradorV3, *time.Time) {
	t.Helper()
	ahora := instanteBasePrueba
	c := credencialesGobiernoPrueba(t, ahora)
	broker := &brokerGobiernoPrueba{credenciales: c, ahora: &ahora}
	repo := &repositorioGobiernoPrueba{ahora: &ahora, recibos: map[string]ports.ReciboAltaBorradorReglasV3{}}
	s, err := NuevoServicioGobiernoV3(repo, repo, broker, func() time.Time { return ahora })
	debeSinError(t, err)
	p := PeticionAltaBorradorV3{Conjunto: conjuntoPrueba(t), Motivo: borradorPrueba(t).MotivoCreacion(), ClaveOperacion: strings.Repeat("a", 32)}
	return s, broker, repo, c, p, &ahora
}

func TestServicioGobiernoV3AltaReplayYLecturasSinNuevaEscritura(t *testing.T) {
	s, b, r, c, p, ahora := servicioGobiernoPrueba(t)
	ctx := context.Background()
	primera, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	*ahora = ahora.Add(time.Second)
	replay, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	if !replay.Replay || replay.Recibo.ReciboRef != primera.Recibo.ReciboRef || !replay.Recibo.ConfirmadaEn.Equal(primera.Recibo.ConfirmadaEn) || !bytes.Equal(replay.Recibo.VersionCanonica, primera.Recibo.VersionCanonica) || len(r.recibos) != 1 {
		t.Fatal("replay perdió el canon/recibo original")
	}
	if r.ultimas[0].HuellaSolicitudSHA256 != r.ultimas[1].HuellaSolicitudSHA256 || r.ultimas[0].HuellaMaterialSHA256 == r.ultimas[1].HuellaMaterialSHA256 || bytes.Equal(r.ultimas[0].VersionCanonica, r.ultimas[1].VersionCanonica) {
		t.Fatal("se mezcló intención estable con propuesta/reloj fresco")
	}
	selector := ports.SelectorGobiernoReglasV3{Identidad: p.Conjunto.Identidad(), Estado: primera.Recibo.Estado}
	motivo, _ := motivoGobiernoV3(p.Motivo)
	consulta, err := s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo})
	debeSinError(t, err)
	recuperada, err := s.RecuperarRecibo(ctx, c, PeticionRecuperarReciboV3{Selector: selector, Motivo: motivo, ClaveOperacion: p.ClaveOperacion, HuellaSolicitudSHA256: primera.Recibo.HuellaSolicitudSHA256})
	debeSinError(t, err)
	if !recuperada.Existe || !bytes.Equal(consulta.VersionCanonica, primera.Recibo.VersionCanonica) || recuperada.Recibo.ReciboRef != primera.Recibo.ReciboRef || r.guardados != 2 || r.consultas != 1 || r.recuperaciones != 1 {
		t.Fatal("lectura generó otro guardado")
	}
	if b.solicitudes[2].Accion != "bolsa.reglas_baremo.version.consultar" || b.solicitudes[3].Accion != "bolsa.reglas_baremo.recibo.consultar" ||
		b.solicitudes[3].Operacion != operacionRecuperarGobiernoV3 || len(b.solicitudes[3].Campos) != 2 ||
		b.solicitudes[3].Campos[0] != "estado_reglas_baremo" || b.solicitudes[3].Campos[1] != "recibo" {
		t.Fatal("lectura usa operación ajena")
	}
	consulta.VersionCanonica[0] = 'x'
	if bytes.Equal(consulta.VersionCanonica, r.recibos[p.ClaveOperacion].VersionCanonica) {
		t.Fatal("respuesta comparte bytes persistidos")
	}
}

func TestServicioGobiernoV3DenegacionAntesDelRepositorio(t *testing.T) {
	for _, caso := range []string{"sin_concesion", "accion", "recurso", "audiencia", "perfil", "contexto", "motivo", "sesion_caducada", "clave", "actor_cero"} {
		t.Run(caso, func(t *testing.T) {
			s, b, r, c, p, ahora := servicioGobiernoPrueba(t)
			switch caso {
			case "sin_concesion":
				b.denegar = true
			case "sesion_caducada":
				*ahora = ahora.Add(2 * time.Hour)
			case "clave":
				p.ClaveOperacion = "predecible"
			case "actor_cero":
				c = CredencialesGobiernoV3{}
			default:
				b.mutar = caso
			}
			if _, err := s.GuardarAltaBorrador(context.Background(), c, p); err == nil || r.guardados != 0 {
				t.Fatal("rechazo llegó al efecto")
			}
		})
	}
}

func TestServicioGobiernoV3ConflictosYReciboIncompleto(t *testing.T) {
	s, _, r, c, p, _ := servicioGobiernoPrueba(t)
	ctx := context.Background()
	_, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	otro := p
	otro.Motivo = motivoPrueba(t, "publicacion")
	if _, err := s.GuardarAltaBorrador(ctx, c, otro); !errors.Is(err, ports.ErrClaveIdempotenciaReglasReutilizada) {
		t.Fatalf("misma clave cambió payload: %v", err)
	}
	otro = p
	otro.ClaveOperacion = strings.Repeat("b", 32)
	if _, err := s.GuardarAltaBorrador(ctx, c, otro); !errors.Is(err, ports.ErrConflictoOCCReglasBaremo) {
		t.Fatalf("se sobrescribió contenido existente: %v", err)
	}
	if len(r.recibos) != 1 {
		t.Fatal("conflictos añadieron historia")
	}
	for _, mutacion := range []string{"sin_outbox", "sin_consumo", "otro_canon", "otra_intencion"} {
		t.Run(mutacion, func(t *testing.T) {
			s, _, r, c, p, _ := servicioGobiernoPrueba(t)
			r.mutar = mutacion
			if _, err := s.GuardarAltaBorrador(ctx, c, p); !errors.Is(err, ports.ErrConfirmacionReglasBaremoInvalida) {
				t.Fatalf("recibo corrupto aceptado: %v", err)
			}
		})
	}
}

func TestMaterialGobiernoV3CanonicoYOpacidad(t *testing.T) {
	s, b, _, c, p, _ := servicioGobiernoPrueba(t)
	_, err := s.GuardarAltaBorrador(context.Background(), c, p)
	debeSinError(t, err)
	var m MaterialGobiernoV3
	debeSinError(t, json.Unmarshal(b.solicitudes[0].MaterialCanonico, &m))
	if m.EstadoEsperado != nil || m.Estado.Revision != 1 || m.PersonaRef != c.actor.PersonaRef || m.PerfilRef != c.actor.PerfilActivoRef || m.Operacion != operacionAltaGobiernoV3 || m.Esquema != esquemaMaterialGobiernoV3 {
		t.Fatal("material no derivado de dominio y sesión")
	}
	if _, err := json.Marshal(c); err == nil {
		t.Fatal("credenciales serializables")
	}
	if _, err := json.Marshal(p); err == nil {
		t.Fatal("orden de negocio serializable")
	}
	var repo *repositorioGobiernoPrueba
	if _, err := NuevoServicioGobiernoV3(repo, repo, b, time.Now); err == nil {
		t.Fatal("dependencia tipada nula aceptada")
	}
}
